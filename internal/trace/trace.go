// Package trace keeps what the game asked its models and voices: every
// prompt with its answer, how long it took and how many tokens it cost, and
// every sentence spoken. The inspector shows it; with NEURO_LOG_TALKS it is
// also written to calls.jsonl for later study.
package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// keep is how many calls and spoken sentences are held for the inspector.
const keep = 400

// Message is one message of a prompt.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Call is one request to a language model.
type Call struct {
	ID int64     `json:"id"`
	At time.Time `json:"at"`
	// Role is what the model is for in the game (rozmowy, walka, umysł,
	// światy, obrazki); Op what was asked (narrate, interpret, note,
	// invent, translate, world, rewrite, warm).
	Role  string `json:"role"`
	Op    string `json:"op"`
	Model string `json:"model"`
	CPU   bool   `json:"cpu,omitempty"`
	// Who is the character the line was for, Said what the player said.
	Who      string    `json:"who,omitempty"`
	Said     string    `json:"said,omitempty"`
	Messages []Message `json:"messages"`
	Reply    string    `json:"reply"`
	Error    string    `json:"error,omitempty"`
	// FirstMs is the wait for the first piece of a streamed answer, TotalMs
	// the whole call; LoadMs, PromptMs and GenMs are the server's own
	// account (Ollama): loading the model, reading the prompt, writing.
	FirstMs  float64 `json:"first_ms,omitempty"`
	TotalMs  float64 `json:"total_ms"`
	LoadMs   float64 `json:"load_ms,omitempty"`
	PromptMs float64 `json:"prompt_ms,omitempty"`
	GenMs    float64 `json:"gen_ms,omitempty"`
	// PromptTokens and ReplyTokens are counted by the server; 0 when it
	// does not say.
	PromptTokens int `json:"prompt_tokens,omitempty"`
	ReplyTokens  int `json:"reply_tokens,omitempty"`
}

// Speech is one sentence given to the voice.
type Speech struct {
	ID     int64     `json:"id"`
	At     time.Time `json:"at"`
	Engine string    `json:"engine"`
	Voice  string    `json:"voice,omitempty"`
	Who    string    `json:"who,omitempty"`
	Mood   string    `json:"mood,omitempty"`
	Text   string    `json:"text"`
	// SynthMs is how long the sentence took to synthesize, AudioMs how
	// long it sounds; 0 for voices that speak straight away.
	SynthMs float64 `json:"synth_ms,omitempty"`
	AudioMs float64 `json:"audio_ms,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// Log holds the latest calls and sentences.
type Log struct {
	mu     sync.Mutex
	next   int64
	calls  []Call
	speech []Speech
	file   *os.File
}

// Default is the game's log.
var Default = &Log{}

// WriteTo makes the log also append everything to a JSONL file.
func (l *Log) WriteTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.file = file

	return nil
}

// AddCall keeps a call.
func (l *Log) AddCall(call Call) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	call.ID = l.next
	l.calls = append(l.calls, call)

	if len(l.calls) > keep {
		l.calls = l.calls[len(l.calls)-keep:]
	}

	l.write(map[string]any{"call": call})
}

// AddSpeech keeps a spoken sentence.
func (l *Log) AddSpeech(speech Speech) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.next++
	speech.ID = l.next
	l.speech = append(l.speech, speech)

	if len(l.speech) > keep {
		l.speech = l.speech[len(l.speech)-keep:]
	}

	l.write(map[string]any{"speech": speech})
}

func (l *Log) write(entry any) {
	if l.file == nil {
		return
	}

	if data, err := json.Marshal(entry); err == nil {
		_, _ = l.file.Write(append(data, '\n'))
	}
}

// Since returns the calls and sentences kept after the given ID.
func (l *Log) Since(id int64) ([]Call, []Speech) {
	l.mu.Lock()
	defer l.mu.Unlock()

	calls := []Call{}
	for _, call := range l.calls {
		if call.ID > id {
			calls = append(calls, call)
		}
	}

	speech := []Speech{}
	for _, said := range l.speech {
		if said.ID > id {
			speech = append(speech, said)
		}
	}

	return calls, speech
}

// Ms is a duration in milliseconds.
func Ms(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
