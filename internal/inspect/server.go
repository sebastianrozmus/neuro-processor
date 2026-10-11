// Package inspect serves the running world to a browser: the map with
// everyone on it, the knowledge graph, the bestiary, the items and the
// people, every line of talk with the prompt behind it, with the means to
// put right what a model wrote badly and to mark what a person finds
// wrong, as material for the next training.
package inspect

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/ui"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

//go:embed index.html
var page []byte

const (
	pictureScale   = 6
	rewriteTimeout = 90 * time.Second
)

// Server shows one realm at a time: the one being played.
type Server struct {
	mu       sync.Mutex
	realm    *game.Realm
	rewriter game.Rewriter
	addr     net.Addr
	// Feedback is the JSON Lines file the marks a person leaves are
	// appended to, with everything behind what they marked; empty keeps
	// them in the realm alone.
	Feedback string
	// TTS, when set, is the Chatterbox server whose voices the inspector
	// lets a person assign and hear.
	TTS *TTS
}

// Start listens on the address and serves the inspector. The realm is
// attached later with Watch, when a game starts.
func Start(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	server := &Server{addr: listener.Addr()}

	go func() {
		_ = http.Serve(listener, server.Handler())
	}()

	return server, nil
}

// New is a server without a listener, for tests and for serving through
// a handler of one's own.
func New() *Server {
	return &Server{}
}

// Watch makes the realm the one shown. The brain, when it can reword
// descriptions, serves the refresh button.
func (s *Server) Watch(realm *game.Realm, brain any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.realm = realm
	s.rewriter, _ = brain.(game.Rewriter)
}

// URL is where the inspector is in a browser.
func (s *Server) URL() string {
	if s.addr == nil {
		return ""
	}

	return "http://" + s.addr.String() + "/"
}

// Handler routes the inspector's page and API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("GET /api/world", s.world)
	mux.HandleFunc("GET /api/exchange", s.exchange)
	mux.HandleFunc("POST /api/fact", s.setFact)
	mux.HandleFunc("DELETE /api/fact", s.dropFact)
	mux.HandleFunc("POST /api/describe", s.describe)
	mux.HandleFunc("POST /api/rewrite", s.rewrite)
	mux.HandleFunc("POST /api/player", s.player)
	mux.HandleFunc("POST /api/mark", s.mark)
	mux.HandleFunc("GET /api/picture", s.picture)
	s.voicesHandlers(mux)

	return mux
}

// current returns the watched realm, or answers that there is none.
func (s *Server) current(w http.ResponseWriter) (*game.Realm, game.Rewriter, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.realm == nil {
		http.Error(w, "żaden świat nie jest teraz grany", http.StatusNotFound)

		return nil, nil, false
	}

	return s.realm, s.rewriter, true
}

func (s *Server) world(w http.ResponseWriter, _ *http.Request) {
	realm, rewriter, ok := s.current(w)
	if !ok {
		return
	}

	report := realm.Inspect()
	report.Rewrites = rewriter != nil
	built := realm.World()

	// The names of the drawings are the screens' business.
	for index := range report.Beasts {
		beast := &report.Beasts[index]
		beast.Picture = ui.SpriteName(beast.Name, world.Shape(beast.Shape), beast.Boss)
	}

	for index := range report.Items {
		item := &report.Items[index]
		item.Picture = ui.IconName(world.ItemSpec{Kind: world.Kind(item.Kind), Effect: world.Effect(item.Effect), Name: item.Name})
	}

	for index := range report.Companions {
		if index < len(built.Companions) {
			report.Companions[index].Picture = ui.FigureName("", built.Companions[index].Role)
		}
	}

	reply(w, report)
}

// exchange shows one line of talk as the model saw it: the full prompt.
func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	request, entry, ok := realm.ExchangeRequest(id)
	if !ok {
		http.Error(w, "nie ma takiej wymiany", http.StatusNotFound)

		return
	}

	reply(w, map[string]any{"exchange": entry, "messages": llm.Transcript(request, entry.Reply)})
}

func (s *Server) setFact(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	var body struct {
		ID      int      `json:"id"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		Known   []string `json:"known"`
	}

	if !decode(w, r, &body) {
		return
	}

	fact, ok := realm.SetFact(body.ID, body.Subject, body.Text, body.Known)
	if !ok {
		http.Error(w, "nie ma takiego faktu albo brak treści", http.StatusBadRequest)

		return
	}

	reply(w, fact)
}

func (s *Server) dropFact(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if !realm.DropFact(id) {
		http.Error(w, "nie ma takiego faktu", http.StatusNotFound)

		return
	}

	reply(w, map[string]bool{"ok": true})
}

func (s *Server) describe(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	var body struct {
		Kind  string `json:"kind"`
		Index int    `json:"index"`
		Text  string `json:"text"`
	}

	if !decode(w, r, &body) {
		return
	}

	if !realm.Describe(body.Kind, body.Index, body.Text, game.HowEdited, "") {
		http.Error(w, "nie ma czego opisać", http.StatusBadRequest)

		return
	}

	reply(w, map[string]bool{"ok": true})
}

// rewrite has the model word a description again and keeps the result,
// with the prompt it was given, as the thing's history.
func (s *Server) rewrite(w http.ResponseWriter, r *http.Request) {
	realm, rewriter, ok := s.current(w)
	if !ok {
		return
	}

	if rewriter == nil {
		http.Error(w, "bez modelu nie ma czym przepisać", http.StatusServiceUnavailable)

		return
	}

	var body struct {
		Kind  string `json:"kind"`
		Index int    `json:"index"`
	}

	if !decode(w, r, &body) {
		return
	}

	name, text, ok := realm.Description(body.Kind, body.Index)
	if !ok {
		http.Error(w, "nie ma czego przepisać", http.StatusBadRequest)

		return
	}

	intro, style := realm.Tone()
	request := game.RewriteRequest{Kind: body.Kind, Name: name, Text: text, World: intro, Style: style}
	ctx, cancel := context.WithTimeout(r.Context(), rewriteTimeout)
	defer cancel()

	fresh, err := rewriter.Rewrite(ctx, request)
	if err != nil {
		http.Error(w, "model nie przepisał: "+err.Error(), http.StatusBadGateway)

		return
	}

	if !realm.Describe(body.Kind, body.Index, fresh, game.HowRewritten, llm.RewritePrompt(request)) {
		http.Error(w, "nie udało się zapisać", http.StatusInternalServerError)

		return
	}

	reply(w, map[string]string{"text": fresh, "was": text})
}

func (s *Server) player(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	var body struct {
		Name  string `json:"name"`
		Field string `json:"field"`
		Value int    `json:"value"`
	}

	if !decode(w, r, &body) {
		return
	}

	if !realm.Adjust(body.Name, body.Field, body.Value) {
		http.Error(w, "nie ma takiego gracza albo pola", http.StatusBadRequest)

		return
	}

	reply(w, map[string]bool{"ok": true})
}

// mark leaves a person's comment on exchanges or on things in the world,
// and appends it with everything behind what was marked to the feedback
// file: the prompts, the replies, the refused lines, the histories.
func (s *Server) mark(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	var body struct {
		Kind    string   `json:"kind"`
		Keys    []string `json:"keys"`
		Comment string   `json:"comment"`
	}

	if !decode(w, r, &body) {
		return
	}

	note, ok := realm.Mark(body.Kind, body.Keys, body.Comment)
	if !ok {
		http.Error(w, "zaznacz coś i napisz uwagę", http.StatusBadRequest)

		return
	}

	record := map[string]any{"time": note.Time, "world": realm.World().Name, "kind": note.Kind, "keys": note.Keys, "comment": note.Comment}

	if body.Kind == "exchange" {
		var marked []map[string]any
		for _, key := range body.Keys {
			id, _ := strconv.Atoi(key)
			if request, entry, ok := realm.ExchangeRequest(id); ok {
				marked = append(marked, map[string]any{"exchange": entry, "messages": llm.Transcript(request, entry.Reply)})
			}
		}

		record["items"] = marked
	} else {
		report := realm.Inspect()
		var marked []map[string]any
		for _, key := range body.Keys {
			index, _ := strconv.Atoi(key)
			name, text, _ := realm.Description(body.Kind, index)
			marked = append(marked, map[string]any{"kind": body.Kind, "key": key, "name": name, "text": text, "traces": report.Traces[body.Kind+":"+key]})
		}

		record["items"] = marked
	}

	if s.Feedback != "" {
		if err := appendLine(s.Feedback, record); err != nil {
			http.Error(w, "uwaga zapisana w świecie, ale nie w pliku: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	reply(w, note)
}

// appendLine adds a JSON record to a JSON Lines file.
func appendLine(path string, record any) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(data, '\n'))

	return err
}

// picture serves the game's own pixel art as PNG: an item's icon, a
// monster's sprite, a companion's or hero's figure.
func (s *Server) picture(w http.ResponseWriter, r *http.Request) {
	realm, _, ok := s.current(w)
	if !ok {
		return
	}

	built := realm.World()
	query := r.URL.Query()
	index, _ := strconv.Atoi(query.Get("index"))

	var (
		image pix.Image
		found bool
	)

	switch query.Get("kind") {
	case "item":
		if name, _, ok := realm.Description(game.DescribeItem, index); ok {
			item := world.ItemSpec{Name: name}
			if index < len(built.Items) {
				item = built.Items[index]
			} else {
				for _, candidate := range realm.Inspect().Items {
					if candidate.Index == index {
						item = world.ItemSpec{Name: candidate.Name, Kind: world.Kind(candidate.Kind), Effect: world.Effect(candidate.Effect), Tier: candidate.Tier}
					}
				}
			}

			image, found = ui.ItemIcon(item), true
		}
	case "monster":
		if index >= 0 && index < len(built.Monsters) {
			image, found = ui.MonsterSprite(built.Monsters[index].Name, built.Monsters[index].Shape, false)
		}
	case "boss":
		image, found = ui.MonsterSprite(built.Boss.Name, built.Boss.Shape, true)
	case "companion":
		if index >= 0 && index < len(built.Companions) {
			image, found = ui.Figure("", built.Companions[index].Role), true
		}
	case "hero":
		image, found = ui.Figure(query.Get("avatar"), world.Role("")), true
	}

	if !found {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age=3600")
	_, _ = w.Write(image.PNG(pictureScale))
}

// decode reads a JSON body, answering the client when it is bad.
func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(into); err != nil && !errors.Is(err, context.Canceled) {
		http.Error(w, "złe żądanie: "+err.Error(), http.StatusBadRequest)

		return false
	}

	return true
}

func reply(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}
