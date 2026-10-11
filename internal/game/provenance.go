package game

import (
	"fmt"
	"strings"
	"time"
)

// Everything a model wrote into the world leaves a trace: which line of
// talk a fact was noted from, which conversation offered an item, what a
// description said before it was reworded and why. The realm also keeps
// the latest exchanges with their full requests, so that the inspector can
// show the prompt behind any line, and the notes a person marks them with.

// Trace is one step in the history of a thing in the world.
type Trace struct {
	Day int `json:"day"`
	// How says what happened: made up in talk, noted from a line, reworded
	// by the model, corrected by hand.
	How string `json:"how"`
	// Prompt is what the model was given, or the line something came from.
	Prompt string `json:"prompt,omitempty"`
	// Was is the text before a change; Why the reason for it.
	Was string `json:"was,omitempty"`
	Why string `json:"why,omitempty"`
}

// Exchange is one line of talk with everything behind it.
type Exchange struct {
	ID     int    `json:"id"`
	Day    int    `json:"day"`
	Player string `json:"player"`
	Keeper string `json:"keeper"`
	Role   string `json:"role,omitempty"`
	Said   string `json:"said"`
	Reply  string `json:"reply"`
	// Brief is the game's decision as the model read it; Hint the piece of
	// lore it was pointed at.
	Brief string `json:"brief"`
	Hint  string `json:"hint,omitempty"`
	// Attempts are the lines the rules refused before Reply; Kept says the
	// first of them was spoken anyway; Canned that Reply is the game's own
	// line, not the model's.
	Attempts []Attempt `json:"attempts,omitempty"`
	Kept     bool      `json:"kept,omitempty"`
	Canned   bool      `json:"canned,omitempty"`
	// Marks are the comments a person left on this exchange.
	Marks []string `json:"marks,omitempty"`
}

// Note is a comment a person left on something in the world.
type Note struct {
	Time    string   `json:"time"`
	Kind    string   `json:"kind"`
	Keys    []string `json:"keys"`
	Comment string   `json:"comment"`
}

const (
	exchangesKept = 200

	HowNoted     = "zapamiętane z kwestii"
	HowInvented  = "wymyślone w rozmowie"
	HowRewritten = "przepisane modelem w podglądzie"
	HowEdited    = "poprawione ręcznie w podglądzie"
	HowAdded     = "dodane w podglądzie"
)

// traceKey names a thing for its traces: "fact:3", "item:extra-1".
func traceKey(kind string, key any) string {
	return fmt.Sprintf("%s:%v", kind, key)
}

// trace adds a step to a thing's history.
func (r *Realm) trace(kind string, key any, step Trace) {
	if r.traces == nil {
		r.traces = map[string][]Trace{}
	}

	step.Day = r.day
	name := traceKey(kind, key)
	r.traces[name] = append(r.traces[name], step)
}

// Exchange records a line of talk as it was spoken: the request behind
// it, the reply, and what the rules refused on the way.
func (r *Realm) Exchange(name string, request NarrateRequest, reply string, rejection *Rejection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.exchangeSerial++
	entry := Exchange{
		ID: r.exchangeSerial, Day: r.day, Player: name, Keeper: request.Merchant.Name, Role: request.Role,
		Said: request.Text, Reply: reply, Brief: request.Decision.BriefFor(request.Role), Hint: request.Hint,
	}

	if rejection != nil {
		entry.Attempts, entry.Kept, entry.Canned = rejection.Attempts, rejection.Kept, !rejection.Kept
	}

	r.exchanges = append(r.exchanges, entry)
	if r.requests == nil {
		r.requests = map[int]NarrateRequest{}
	}

	r.requests[entry.ID] = request

	if len(r.exchanges) > exchangesKept {
		delete(r.requests, r.exchanges[0].ID)
		r.exchanges = r.exchanges[1:]
	}
}

// ExchangeRequest returns the request and reply of a recorded exchange,
// for showing the prompt as the model saw it.
func (r *Realm) ExchangeRequest(id int) (NarrateRequest, Exchange, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	request, ok := r.requests[id]
	if !ok {
		return NarrateRequest{}, Exchange{}, false
	}

	for _, entry := range r.exchanges {
		if entry.ID == id {
			return request, entry, true
		}
	}

	return NarrateRequest{}, Exchange{}, false
}

// Mark leaves a comment on things in the world: exchanges by ID, or
// anything else by its trace key. It returns the note as kept.
func (r *Realm) Mark(kind string, keys []string, comment string) (Note, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	comment = strings.TrimSpace(comment)
	if comment == "" || len(keys) == 0 {
		return Note{}, false
	}

	note := Note{Time: time.Now().Format(time.RFC3339), Kind: kind, Keys: keys, Comment: comment}
	r.notes = append(r.notes, note)

	if kind == "exchange" {
		for index := range r.exchanges {
			if containsKey(keys, r.exchanges[index].ID) {
				r.exchanges[index].Marks = append(r.exchanges[index].Marks, comment)
			}
		}
	} else {
		for _, key := range keys {
			r.trace(kind, key, Trace{How: "uwaga", Why: comment})
		}
	}

	return note, true
}

func containsKey(keys []string, id int) bool {
	for _, key := range keys {
		if key == fmt.Sprint(id) {
			return true
		}
	}

	return false
}
