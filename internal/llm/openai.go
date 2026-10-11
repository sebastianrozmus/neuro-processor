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
	"sync/atomic"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
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

	// server is what the server turned out to be, found out once; see
	// the serverKind values.
	server int32
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
}

// ollamaRequest is the same request in the shape of Ollama's own API,
// which is the one that takes the context window.
type ollamaRequest struct {
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Format   any            `json:"format,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Stream   bool           `json:"stream"`
}

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
	_, _ = c.chat(ctx, chatRequest{Model: c.Model, Messages: []message{{Role: "user", Content: "ok"}}, MaxTokens: 1})
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
		Model: c.Model,
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
		Model: c.Model,
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
		Model: c.Model,
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

	return found.FactSpec, found.Found && found.Subject != "" && found.Text != "", nil
}

// Rewrite words a description of the world again with the Mind: the same
// meaning and names in cleaner Polish, for the inspector's refresh.
func (c *Client) Rewrite(ctx context.Context, request game.RewriteRequest) (string, error) {
	content, err := c.chat(ctx, chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: rewriteInstructions},
			{Role: "user", Content: rewritePrompt(request)},
		},
		MaxTokens: 200,
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
		Model: c.Model,
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

	return found.ItemSpec, found.Offered, nil
}

// Narrate asks the model to word the merchant's reply.
func (c *Client) Narrate(ctx context.Context, request game.NarrateRequest) (string, error) {
	return c.chat(ctx, chatRequest{Model: c.Model, Messages: narrateMessages(request), MaxTokens: c.MaxTokens})
}

// Stream asks for the merchant's reply as a stream of pieces, delivering
// each as it arrives, and returns the whole.
func (c *Client) Stream(ctx context.Context, request game.NarrateRequest, deliver func(piece string)) (string, error) {
	response, err := c.send(ctx, chatRequest{Model: c.Model, Messages: narrateMessages(request), MaxTokens: c.MaxTokens, Stream: true})
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// OpenAI servers send events ("data: {…}"), Ollama's own API one JSON
	// object per line; both end with a line that carries no text.
	var text strings.Builder
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

		if piece := event.text(); piece != "" {
			text.WriteString(piece)
			deliver(piece)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return text.String(), nil
}

func (c *Client) chat(ctx context.Context, payload chatRequest) (string, error) {
	response, err := c.send(ctx, payload)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var parsed reply
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("model API returned invalid JSON: %w", err)
	}

	if len(parsed.Choices) == 0 && parsed.Message.Content == "" {
		return "", errors.New("model API returned no choices")
	}

	// An answer the server cut short is whole only when it is prose the
	// request capped itself; a cut JSON object is never whole, and an
	// uncapped answer was cut by the window running out.
	// The cut text comes back with the error, for looking into.
	if parsed.cut() && (payload.MaxTokens == 0 || payload.ResponseFormat != nil) {
		return parsed.text(), fmt.Errorf("%w: it did not fit the context window", ErrCut)
	}

	return parsed.text(), nil
}

// send posts a request and returns the response once its status is known
// to be good; the caller reads and closes the body.
func (c *Client) send(ctx context.Context, payload chatRequest) (*http.Response, error) {
	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"

	var sent any = payload
	if c.ollama(ctx) {
		url, sent = c.root()+"/api/chat", native(payload, c.Context)
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
	if c.Context <= 0 {
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
func native(payload chatRequest, window int) ollamaRequest {
	request := ollamaRequest{
		Model:    payload.Model,
		Messages: payload.Messages,
		Options:  map[string]any{"num_ctx": window},
		Stream:   payload.Stream,
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
