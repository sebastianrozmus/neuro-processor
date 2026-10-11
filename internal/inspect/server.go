// Package inspect serves the running world to a browser: the map with
// everyone on it, the knowledge graph, the bestiary, the items and the
// people, with the means to put right what a model wrote badly.
package inspect

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
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
	mux.HandleFunc("POST /api/fact", s.setFact)
	mux.HandleFunc("DELETE /api/fact", s.dropFact)
	mux.HandleFunc("POST /api/describe", s.describe)
	mux.HandleFunc("POST /api/rewrite", s.rewrite)
	mux.HandleFunc("POST /api/player", s.player)
	mux.HandleFunc("GET /api/picture", s.picture)

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
	reply(w, report)
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

	if !realm.Describe(body.Kind, body.Index, body.Text) {
		http.Error(w, "nie ma czego opisać", http.StatusBadRequest)

		return
	}

	reply(w, map[string]bool{"ok": true})
}

// rewrite has the model word a description again and keeps the result.
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
	ctx, cancel := context.WithTimeout(r.Context(), rewriteTimeout)
	defer cancel()

	fresh, err := rewriter.Rewrite(ctx, game.RewriteRequest{Kind: body.Kind, Name: name, Text: text, World: intro, Style: style})
	if err != nil {
		http.Error(w, "model nie przepisał: "+err.Error(), http.StatusBadGateway)

		return
	}

	if !realm.Describe(body.Kind, body.Index, fresh) {
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
		if index >= 0 && index < len(built.Items) {
			image, found = ui.ItemIcon(built.Items[index]), true
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
