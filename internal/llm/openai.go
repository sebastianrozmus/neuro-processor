package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/trace"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const errorBodyLimit = 300

// probeTimeout is how long finding out what the server is may take.
const probeTimeout = 3 * time.Second

// wordy is how many words a message needs before a model is asked to judge
// how persuasive it is.
const wordy = 6

// Client talks to any server exposing the OpenAI chat completions API,
// which covers hosted providers as well as local runtimes.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	// MaxTokens caps the length of spoken replies; 0 leaves it to the server.
	MaxTokens int
	HTTP      *http.Client
	// Context is the window, in tokens, the model is run in when the server
	// turns out to be an Ollama, whose default of 4096 is too small to
	// design a world in: the answer gets cut in the middle of the JSON.
	// Ollama takes the window only through its own API, so requests go
	// there instead of the OpenAI one. 0 keeps the server's setting; other
	// servers are not asked.
	Context int
	// Role is what the model is for in the game (rozmowy, walka, umysł,
	// światy, obrazki), for the trace of calls.
	Role string
	// CPU runs the model on the processor rather than the graphics card
	// (an Ollama option; other servers ignore it).
	CPU bool

	// server is what the server turned out to be, found out once; see
	// the serverKind values.
	server int32
	// mu guards Model and CPU, which the settings may change while the
	// game runs.
	mu sync.Mutex
}

// model is the model requests go to.
func (c *Client) model() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Model
}

// onCPU reports whether the model is to run on the processor.
func (c *Client) onCPU() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.CPU
}

// Use switches the client to another model, on the processor or the
// graphics card, while the game runs.
func (c *Client) Use(model string, cpu bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Model, c.CPU = model, cpu
}

// Name is the model requests go to and whether it runs on the processor.
func (c *Client) Name() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Model, c.CPU
}

// What a server turned out to be.
const (
	serverUnknown int32 = iota
	serverOllama
	serverOther
)

// ErrCut says the server stopped the answer at its length limit before the
// model finished. For a request without a cap of its own, that is the
// context window running out.
var ErrCut = errors.New("the model's answer was cut short")

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	ResponseFormat any       `json:"response_format,omitempty"`
	MaxTokens      int       `json:"max_tokens,omitempty"`
	Temperature    float64   `json:"temperature,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	// StreamOptions asks OpenAI servers to count the tokens of a stream.
	StreamOptions map[string]any `json:"stream_options,omitempty"`

	// op, who and said describe the request for the trace: what was
	// asked, for which character, after what the player said.
	op, who, said string
}

// ollamaRequest is the same request in the shape of Ollama's own API,
// which is the one that takes the context window.
type ollamaRequest struct {
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Format   any            `json:"format,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Stream   bool           `json:"stream"`
	// KeepAlive is how long Ollama holds the model in memory after the
	// answer; its own default of 5 minutes drops it between two talks, and
	// loading an 11B model again costs some 20 seconds.
	KeepAlive string `json:"keep_alive,omitempty"`
}

// keepAlive is how long a local model stays loaded between requests.
const keepAlive = "30m"

// reply is an answer, or one streamed piece of it, in either the OpenAI
// shape (choices) or Ollama's own (message, done_reason).
type reply struct {
	Choices []struct {
		Message      message `json:"message"`
		Delta        message `json:"delta"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Message    message `json:"message"`
	DoneReason string  `json:"done_reason"`

	// The tokens and times, as OpenAI servers (usage) and Ollama count
	// them; Ollama's durations are in nanoseconds.
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	PromptEvalCount    int   `json:"prompt_eval_count"`
	EvalCount          int   `json:"eval_count"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalDuration       int64 `json:"eval_duration"`
}

// counted reports whether the reply carries token counts.
func (r reply) counted() bool {
	return r.Usage.PromptTokens+r.Usage.CompletionTokens+r.PromptEvalCount+r.EvalCount > 0
}

// trace keeps a finished call in the game's trace: its prompt, answer,
// times and tokens.
func (c *Client) trace(payload chatRequest, started time.Time, first time.Duration, text string, counts reply, err error) {
	model, cpu := c.Name()
	call := trace.Call{
		At: started, Role: c.Role, Op: payload.op, Model: model, CPU: cpu, Who: payload.who, Said: payload.said,
		Reply: text, TotalMs: trace.Ms(time.Since(started)), FirstMs: trace.Ms(first),
		PromptTokens: counts.Usage.PromptTokens + counts.PromptEvalCount, ReplyTokens: counts.Usage.CompletionTokens + counts.EvalCount,
		LoadMs: float64(counts.LoadDuration) / 1e6, PromptMs: float64(counts.PromptEvalDuration) / 1e6, GenMs: float64(counts.EvalDuration) / 1e6,
	}

	for _, sent := range payload.Messages {
		call.Messages = append(call.Messages, trace.Message{Role: sent.Role, Content: sent.Content})
	}

	if err != nil {
		call.Error = err.Error()
	}

	trace.Default.AddCall(call)
}

// text is the content of the reply, whichever shape it came in.
func (r reply) text() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content + r.Choices[0].Delta.Content
	}

	return r.Message.Content
}

// cut reports whether the server stopped the answer at a length limit.
func (r reply) cut() bool {
	return r.DoneReason == "length" || (len(r.Choices) > 0 && r.Choices[0].FinishReason == "length")
}

// Local reports whether the model runs on this machine, where it has to be
// loaded into memory before its first answer.
func (c *Client) Local() bool {
	address, err := url.Parse(c.BaseURL)

	return err == nil && (address.Hostname() == "localhost" || net.ParseIP(address.Hostname()).IsLoopback())
}

// Warm makes the server load the model, so that the first real answer does
// not have to wait for it. Failures are ignored: the first real request
// will report them.
func (c *Client) Warm(ctx context.Context) {
	_, _ = c.chat(ctx, chatRequest{Model: c.model(), op: "warm", Messages: []message{{Role: "user", Content: "ok"}}, MaxTokens: 1})
}

// VoiceModel names the model that words the lines: with one client, this one.
func (c *Client) VoiceModel() string {
	return c.model()
}

// RewriteModel names the model the inspector rewrites with by default.
func (c *Client) RewriteModel() string {
	return c.model()
}

// RewriterWith rewrites with the named model on the same server.
func (c *Client) RewriterWith(model string) game.Rewriter {
	return c.WithModel(model)
}

// WithModel is a client of the same server running another model. What
// the server turned out to be is found out again.
func (c *Client) WithModel(model string) *Client {
	return &Client{BaseURL: c.BaseURL, APIKey: c.APIKey, Model: model, MaxTokens: c.MaxTokens, HTTP: c.HTTP, Context: c.Context}
}

// Models lists the models the server offers (GET /models, which Ollama
// serves too).
func (c *Client) Models(ctx context.Context) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}

	if c.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models: %s", response.Status)
	}

	var listing struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(listing.Data))
	for _, model := range listing.Data {
		names = append(names, model.ID)
	}

	return names, nil
}

// Interpret reads what the player told a shopkeeper. The game's rules read
// the message first. The model is asked only when its opinion could matter:
// to tell whether a bare number is an offer, or to judge the argument in a
// long message. What it says is merged under the rules' control. Plain talk
// needs no reading at all, which keeps conversations quick. If the model
// fails, the rules' own reading is used whenever they were sure of it.
func (c *Client) Interpret(ctx context.Context, request game.InterpretRequest) (game.Intent, error) {
	reading := game.Read(request)
	if !reading.Open(len(strings.Fields(request.Text)) >= wordy) {
		return reading.Intent, nil
	}

	model, err := c.interpret(ctx, request)
	if err != nil {
		if reading.Sure {
			return reading.Intent, nil
		}

		return game.Intent{}, err
	}

	return reading.Merge(request.Sanitize(model)), nil
}

// interpret asks the model for an intent constrained by a JSON schema.
func (c *Client) interpret(ctx context.Context, request game.InterpretRequest) (game.Intent, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "interpret", said: request.Text,
		Messages: []message{
			{Role: "system", Content: interpretInstructions},
			{Role: "user", Content: interpretPrompt(request)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "intent",
				"strict": true,
				"schema": intentSchema(request.Wares),
			},
		},
	})
	if err != nil {
		return game.Intent{}, err
	}

	var intent game.Intent
	if err := json.Unmarshal([]byte(extractJSON(content)), &intent); err != nil {
		return game.Intent{}, fmt.Errorf("model returned invalid intent: %w", err)
	}

	return intent, nil
}

// Translate renders a short description in English, for tools that only
// understand English. It returns the text alone, without commentary.
func (c *Client) Translate(ctx context.Context, text string) (string, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "translate",
		Messages: []message{
			{Role: "system", Content: "Translate the user's text into English. Reply with the translation only: no quotes, no notes."},
			{Role: "user", Content: text},
		},
		MaxTokens: 120,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(strings.TrimSpace(content), "\"„”«»"), nil
}

// Invent asks the model whether a keeper's line offered an item that is not
// among the wares, and what that item is like.
// Note asks the model whether a keeper's line stated a new fact about the
// world, so that the world can remember it.
func (c *Client) Note(ctx context.Context, request game.NoteRequest) (game.FactSpec, bool, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "note",
		Messages: []message{
			{Role: "system", Content: noteInstructions},
			{Role: "user", Content: notePrompt(request)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "new_fact",
				"strict": true,
				"schema": noteSchema(),
			},
		},
	})
	if err != nil {
		return game.FactSpec{}, false, err
	}

	var found struct {
		Found bool `json:"found"`
		game.FactSpec
	}

	if err := json.Unmarshal([]byte(extractJSON(content)), &found); err != nil {
		return game.FactSpec{}, false, fmt.Errorf("model returned an invalid fact: %w", err)
	}

	return found.FactSpec, found.Found && saysSomething(found.Subject) && saysSomething(found.Text), nil
}

// saysSomething reports whether a field the Mind filled says anything: a lone comma
// or quote mark is what a model writes when it has nothing to say.
func saysSomething(text string) bool {
	letters := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			letters++
		}
	}

	return letters >= 2
}

// SelfNote asks the model what a companion's line said about themselves,
// so that they can keep to it later.
func (c *Client) SelfNote(ctx context.Context, request game.SelfNoteRequest) ([]string, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "selfnote", who: request.Speaker, said: request.Said,
		Messages: []message{
			{Role: "system", Content: selfNoteInstructions},
			{Role: "user", Content: selfNotePrompt(request)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "self_facts",
				"strict": true,
				"schema": selfNoteSchema(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var found struct {
		Facts []string `json:"facts"`
	}

	if err := json.Unmarshal([]byte(extractJSON(content)), &found); err != nil {
		return nil, fmt.Errorf("model returned invalid self facts: %w", err)
	}

	var facts []string
	for _, fact := range found.Facts {
		if fact = strings.TrimSpace(fact); saysSomething(fact) {
			facts = append(facts, fact)
		}
	}

	return facts, nil
}

// QuestHook asks the model whether a keeper's line told of someone dear
// gone missing near a monster or a place, so that the game can make a
// rescue of it.
func (c *Client) QuestHook(ctx context.Context, request game.QuestHookRequest) (game.QuestHook, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "questhook", who: request.Speaker, said: request.Said,
		Messages: []message{
			{Role: "system", Content: questHookInstructions},
			{Role: "user", Content: questHookPrompt(request)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "quest_hook",
				"strict": true,
				"schema": questHookSchema(request),
			},
		},
	})
	if err != nil {
		return game.QuestHook{}, err
	}

	var hook game.QuestHook
	if err := json.Unmarshal([]byte(extractJSON(content)), &hook); err != nil {
		return game.QuestHook{}, fmt.Errorf("model returned an invalid quest hook: %w", err)
	}

	hook.Found = hook.Found && strings.TrimSpace(hook.Person) != ""

	return hook, nil
}

// Rewrite words a description of the world again with the Mind: the same
// meaning and names in cleaner Polish, for the inspector's refresh.
func (c *Client) Rewrite(ctx context.Context, request game.RewriteRequest) (string, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "rewrite",
		Messages: []message{
			{Role: "system", Content: rewriteInstructions},
			{Role: "user", Content: RewritePrompt(request)},
		},
		MaxTokens:   200,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}

	text := strings.Trim(strings.TrimSpace(content), "\"„”«»")
	if text == "" {
		return "", errors.New("model returned an empty description")
	}

	return text, nil
}

func (c *Client) Invent(ctx context.Context, request game.InventRequest) (world.ItemSpec, bool, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.model(), op: "invent",
		Messages: []message{
			{Role: "system", Content: inventInstructions},
			{Role: "user", Content: inventPrompt(request)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "offered_item",
				"strict": true,
				"schema": inventSchema(),
			},
		},
	})
	if err != nil {
		return world.ItemSpec{}, false, err
	}

	var found struct {
		Offered bool `json:"offered"`
		world.ItemSpec
	}

	if err := json.Unmarshal([]byte(extractJSON(content)), &found); err != nil {
		return world.ItemSpec{}, false, fmt.Errorf("model returned an invalid item: %w", err)
	}

	return found.ItemSpec, found.Offered && saysSomething(found.Name), nil
}

// Narrate asks the model to word the merchant's reply.
func (c *Client) Narrate(ctx context.Context, request game.NarrateRequest) (string, error) {
	return c.chat(ctx, chatRequest{Model: c.model(), Messages: narrateMessages(request), MaxTokens: c.MaxTokens, op: "narrate", who: request.Merchant.Name, said: request.Text})
}

// Stream asks for the merchant's reply as a stream of pieces, delivering
// each as it arrives, and returns the whole.
func (c *Client) Stream(ctx context.Context, request game.NarrateRequest, deliver func(piece string)) (string, error) {
	payload := chatRequest{
		Model: c.model(), Messages: narrateMessages(request), MaxTokens: c.MaxTokens, Stream: true,
		StreamOptions: map[string]any{"include_usage": true}, op: "narrate", who: request.Merchant.Name, said: request.Text,
	}

	var (
		started = time.Now()
		first   time.Duration
		counts  reply
		text    strings.Builder
	)

	response, err := c.send(ctx, payload)
	if err != nil {
		c.trace(payload, started, 0, "", counts, err)

		return "", err
	}
	defer response.Body.Close()

	// OpenAI servers send events ("data: {…}"), Ollama's own API one JSON
	// object per line; both end with a line that carries no text, and
	// the counts of tokens come with the last one.
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		data := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "data:"))
		if data == "[DONE]" {
			break
		}

		var event reply
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.counted() {
			counts = event
		}

		if piece := event.text(); piece != "" {
			if first == 0 {
				first = time.Since(started)
			}

			text.WriteString(piece)
			deliver(piece)
		}
	}

	err = scanner.Err()
	c.trace(payload, started, first, text.String(), counts, err)

	if err != nil {
		return "", err
	}

	return text.String(), nil
}

func (c *Client) chat(ctx context.Context, payload chatRequest) (string, error) {
	started := time.Now()
	text, counts, err := c.ask(ctx, payload)
	c.trace(payload, started, 0, text, counts, err)

	return text, err
}

// ask sends a request and reads the whole answer.
func (c *Client) ask(ctx context.Context, payload chatRequest) (string, reply, error) {
	response, err := c.send(ctx, payload)
	if err != nil {
		return "", reply{}, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", reply{}, err
	}

	var parsed reply
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", reply{}, fmt.Errorf("model API returned invalid JSON: %w", err)
	}

	if len(parsed.Choices) == 0 && parsed.Message.Content == "" {
		return "", parsed, errors.New("model API returned no choices")
	}

	// An answer the server cut short is whole only when it is prose the
	// request capped itself; a cut JSON object is never whole, and an
	// uncapped answer was cut by the window running out.
	// The cut text comes back with the error, for looking into.
	if parsed.cut() && (payload.MaxTokens == 0 || payload.ResponseFormat != nil) {
		return parsed.text(), parsed, fmt.Errorf("%w: it did not fit the context window", ErrCut)
	}

	return parsed.text(), parsed, nil
}

// send posts a request and returns the response once its status is known
// to be good; the caller reads and closes the body.
func (c *Client) send(ctx context.Context, payload chatRequest) (*http.Response, error) {
	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"

	var sent any = payload
	if c.ollama(ctx) {
		url, sent = c.root()+"/api/chat", native(payload, c.Context, c.onCPU())
	}

	body, err := json.Marshal(sent)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	if c.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		data, _ := io.ReadAll(response.Body)

		return nil, fmt.Errorf("model API returned %s: %s", response.Status, truncate(string(data), errorBodyLimit))
	}

	return response, nil
}

// root is the server's address without the OpenAI path.
func (c *Client) root() string {
	return strings.TrimSuffix(strings.TrimRight(c.BaseURL, "/"), "/v1")
}

// ollama reports whether the server is an Ollama that should be asked for
// the context window. It is found out once, by asking for the version
// Ollama's own API gives.
func (c *Client) ollama(ctx context.Context) bool {
	// The processor is an option of Ollama's own API too.
	if c.Context <= 0 && !c.onCPU() {
		return false
	}

	if kind := atomic.LoadInt32(&c.server); kind != serverUnknown {
		return kind == serverOllama
	}

	kind := serverOther
	if c.versioned(ctx) {
		kind = serverOllama
	}

	atomic.StoreInt32(&c.server, kind)

	return kind == serverOllama
}

// versioned asks the server for its version the way Ollama answers it.
func (c *Client) versioned(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.root()+"/api/version", nil)
	if err != nil {
		return false
	}

	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()

	var found struct {
		Version string `json:"version"`
	}

	return response.StatusCode == http.StatusOK && json.NewDecoder(response.Body).Decode(&found) == nil && found.Version != ""
}

// native puts a request into the shape of Ollama's own API, with the
// window to run the model in. The schema goes as the format; a cap on the
// answer as the number of tokens to predict.
func native(payload chatRequest, window int, cpu bool) ollamaRequest {
	request := ollamaRequest{
		Model:     payload.Model,
		Messages:  payload.Messages,
		Options:   map[string]any{},
		Stream:    payload.Stream,
		KeepAlive: keepAlive,
	}

	if window > 0 {
		request.Options["num_ctx"] = window
	}

	// No layers on the graphics card: the model runs on the processor and
	// leaves the card to the others.
	if cpu {
		request.Options["num_gpu"] = 0
	}

	if payload.MaxTokens > 0 {
		request.Options["num_predict"] = payload.MaxTokens
	}

	if payload.Temperature > 0 {
		request.Options["temperature"] = payload.Temperature
	}

	if format, ok := payload.ResponseFormat.(map[string]any); ok {
		if wrapped, ok := format["json_schema"].(map[string]any); ok {
			request.Format = wrapped["schema"]
		}
	}

	return request
}

// extractJSON cuts the JSON object out of a reply that wraps it in prose or
// a code fence, which servers without schema enforcement tend to do.
func extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start < 0 || end < start {
		return content
	}

	return content[start : end+1]
}

func truncate(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}

	return text[:limit] + "…"
}
