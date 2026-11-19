// Package shareserver is the collecting server for conversations players
// agreed to share (see package share for the protocol). It is meant to be
// small: one process, no database, append-only JSON lines on disk.
//
// Protection against floods is layered: token buckets per address and per
// network, separate caps on requests in flight for challenges and for
// writes, small body and header limits, short timeouts, a proof of work on
// every write, and daily caps per key, on all records and on bytes. A
// volumetric attack on the network itself must be stopped before it
// reaches the host (a provider's firewall or a CDN); no process can.
package shareserver

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// Config sets the server up.
type Config struct {
	// Data is the directory the records are kept in.
	Data string
	// Secret is mixed into the key that signs challenges. Each start adds
	// random bytes of its own, so challenges never outlive the process and
	// nothing accepted before a restart can be replayed after it.
	Secret []byte
	// Difficulty is the proof of work, in leading zero bits (1..32).
	Difficulty int
	// Limits bound what the store takes; zero values get defaults.
	Limits Limits
	// Proxies are the addresses or networks of the proxy in front (Caddy):
	// only from them is the client's address taken from the last
	// X-Forwarded-For entry.
	Proxies []*net.IPNet
	// Models, when not empty, are the model names the game ships; others
	// are flagged, not refused.
	Models map[string]bool
	// InFlight caps writes handled at once; challenges get as many again.
	InFlight int
	// Now is the clock; time.Now when nil.
	Now func() time.Time
}

const (
	challengeTTL  = 10 * time.Minute
	challengeSkew = time.Minute
	challengeLen  = 8 + 16 + 1 + 16
	maxUsed       = 200_000
	maxAddresses  = 100_000
)

// Server handles the HTTP API.
type Server struct {
	cfg    Config
	macKey []byte
	store  *Store
	reads  chan struct{}
	writes chan struct{}

	usedMu sync.Mutex
	used   map[string]time.Time

	challengeLimit, writeLimit *Limiter
	// networkLimit counts whole networks (/24, /48), so that one owner of
	// many addresses cannot rotate through them.
	networkLimit *Limiter
}

// New opens the store and returns a server.
func New(cfg Config) (*Server, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	if cfg.Difficulty == 0 {
		cfg.Difficulty = 20
	}

	if cfg.Difficulty < 1 || cfg.Difficulty > 32 {
		return nil, fmt.Errorf("trudność %d poza 1..32", cfg.Difficulty)
	}

	if cfg.InFlight <= 0 {
		cfg.InFlight = 32
	}

	limits := &cfg.Limits
	if limits.PerKeyDay <= 0 {
		limits.PerKeyDay = 200
	}

	if limits.PerDay <= 0 {
		limits.PerDay = 20_000
	}

	if limits.PerDayBytes <= 0 {
		limits.PerDayBytes = 200 << 20
	}

	if limits.Total <= 0 {
		limits.Total = 5 << 30
	}

	if limits.ForgetsPerDay <= 0 {
		limits.ForgetsPerDay = 5_000
	}

	boot := make([]byte, 32)
	if _, err := rand.Read(boot); err != nil {
		return nil, err
	}

	mixed := sha256.Sum256(append(append([]byte(share.Domain+"\x00"), cfg.Secret...), boot...))

	store, err := OpenStore(cfg.Data, cfg.Limits, cfg.Now)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg: cfg, macKey: mixed[:], store: store, used: map[string]time.Time{},
		reads: make(chan struct{}, cfg.InFlight), writes: make(chan struct{}, cfg.InFlight),
		// Per address: 10 challenges a minute with a burst of 10, writes 5 a
		// minute with a burst of 5; per network eight times that.
		challengeLimit: NewLimiter(10.0/60, 10, maxAddresses, cfg.Now),
		writeLimit:     NewLimiter(5.0/60, 5, maxAddresses, cfg.Now),
		networkLimit:   NewLimiter(8*15.0/60, 8*15, maxAddresses, cfg.Now),
	}, nil
}

// Handler returns the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("GET /v1/challenge", s.limited(s.challengeLimit, s.reads, s.challenge))
	mux.HandleFunc("POST /v1/submit", s.limited(s.writeLimit, s.writes, s.submit))
	mux.HandleFunc("POST /v1/forget", s.limited(s.writeLimit, s.writes, s.forget))

	return mux
}

// Sweep drops expired state; call it every minute or so.
func (s *Server) Sweep() {
	now := s.cfg.Now()

	s.usedMu.Lock()
	for challenge, expires := range s.used {
		if now.After(expires) {
			delete(s.used, challenge)
		}
	}
	s.usedMu.Unlock()

	s.challengeLimit.Sweep(10 * time.Minute)
	s.writeLimit.Sweep(10 * time.Minute)
	s.networkLimit.Sweep(10 * time.Minute)
}

// Close releases the data directory.
func (s *Server) Close() error { return s.store.Close() }

func (s *Server) limited(limiter *Limiter, slots chan struct{}, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address, network := s.address(r)

		if !s.networkLimit.Allow(network) || !limiter.Allow(address) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "za dużo zapytań", http.StatusTooManyRequests)

			return
		}

		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.Header().Set("Retry-After", "30")
			http.Error(w, "serwer zajęty", http.StatusServiceUnavailable)

			return
		}

		next(w, r)
	}
}

// address names the client for the limits: its address (IPv6 by /64) and
// its network (IPv4 /24, IPv6 /48).
func (s *Server) address(r *http.Request) (string, string) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peer := net.ParseIP(host)

	if peer != nil && s.fromProxy(peer) {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1])); ip != nil {
				peer = ip
			}
		}
	}

	if peer == nil {
		return host, host
	}

	if v4 := peer.To4(); v4 != nil {
		return v4.String(), v4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}

	return peer.Mask(net.CIDRMask(64, 128)).String() + "/64", peer.Mask(net.CIDRMask(48, 128)).String() + "/48"
}

func (s *Server) fromProxy(peer net.IP) bool {
	for _, network := range s.cfg.Proxies {
		if network.Contains(peer) {
			return true
		}
	}

	return false
}

func (s *Server) challenge(w http.ResponseWriter, _ *http.Request) {
	raw := make([]byte, challengeLen)
	binary.BigEndian.PutUint64(raw[:8], uint64(s.cfg.Now().Unix()))

	if _, err := rand.Read(raw[8:24]); err != nil {
		http.Error(w, "błąd serwera", http.StatusInternalServerError)

		return
	}

	raw[24] = byte(s.cfg.Difficulty)
	copy(raw[25:], s.mac(raw[:25]))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(share.Challenge{
		Challenge: base64.RawURLEncoding.EncodeToString(raw), Difficulty: s.cfg.Difficulty,
	})
}

func (s *Server) mac(data []byte) []byte {
	h := hmac.New(sha256.New, s.macKey)
	h.Write(data)

	return h.Sum(nil)[:16]
}

// errBad is a refusal whose text the client may show.
type errBad struct{ text string }

func (e errBad) Error() string { return e.text }

// decode reads canonical unpadded base64url: one spelling per value, so
// that the same key or challenge cannot pass as two.
func decode(text string) ([]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(text)

	return raw, err == nil && base64.RawURLEncoding.EncodeToString(raw) == text
}

// open checks an envelope: its challenge, proof of work and signature. It
// spends the challenge, so one envelope is accepted once.
func (s *Server) open(w http.ResponseWriter, r *http.Request, action string) (share.Envelope, error) {
	r.Body = http.MaxBytesReader(w, r.Body, share.MaxBody)

	var envelope share.Envelope

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&envelope); err != nil {
		return envelope, errBad{"zły format"}
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return envelope, errBad{"zły format"}
	}

	raw, ok := decode(envelope.Challenge)
	if !ok || len(raw) != challengeLen || !hmac.Equal(raw[25:], s.mac(raw[:25])) {
		return envelope, errBad{"zły challenge"}
	}

	issued := time.Unix(int64(binary.BigEndian.Uint64(raw[:8])), 0)
	fresh := func() bool {
		now := s.cfg.Now()

		return now.Sub(issued) <= challengeTTL && issued.Sub(now) <= challengeSkew
	}

	if !fresh() {
		return envelope, errBad{"challenge wygasł"}
	}

	key, ok := decode(envelope.Key)
	if !ok || len(key) != ed25519.PublicKeySize || share.WeakKey(key) {
		return envelope, errBad{"zły klucz"}
	}

	if !share.ValidNonce(envelope.Nonce) || share.Work(envelope.Challenge, envelope.Key, envelope.Nonce) < int(raw[24]) {
		return envelope, errBad{"brak dowodu pracy"}
	}

	sig, ok := decode(envelope.Sig)
	if !ok || !ed25519.Verify(key, share.Signed(action, envelope.Key, envelope.Challenge, envelope.Nonce, envelope.Payload), sig) {
		return envelope, errBad{"zły podpis"}
	}

	// The challenge is spent only once everything else held, so a forged
	// envelope cannot burn a real client's challenge.
	// Freshness is checked again under the lock: Sweep drops a spent
	// challenge only once it expired, so an expired one cannot slip past.
	s.usedMu.Lock()
	defer s.usedMu.Unlock()

	if !fresh() {
		return envelope, errBad{"challenge wygasł"}
	}

	if _, spent := s.used[envelope.Challenge]; spent {
		return envelope, errBad{"challenge już użyty"}
	}

	if len(s.used) >= maxUsed {
		return envelope, errBusy
	}

	s.used[envelope.Challenge] = issued.Add(challengeTTL + challengeSkew)

	return envelope, nil
}

var errBusy = errors.New("busy")

func (s *Server) refuse(w http.ResponseWriter, err error) {
	var bad errBad

	switch {
	case errors.As(err, &bad):
		http.Error(w, bad.text, http.StatusBadRequest)
	case errors.Is(err, errBusy), errors.Is(err, ErrFull):
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "serwer zajęty", http.StatusServiceUnavailable)
	case errors.Is(err, ErrDuplicate):
		// A plain conflict from some proxy must not pass for this.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprintln(w, `{"ok":true,"duplicate":true}`)
	case errors.Is(err, ErrForgotten):
		http.Error(w, "ta instalacja wycofała zgodę", http.StatusForbidden)
	case errors.Is(err, ErrQuota):
		w.Header().Set("Retry-After", "3600")
		http.Error(w, "dzienny limit wysyłek", http.StatusTooManyRequests)
	default:
		log.Printf("błąd: %v", err)
		http.Error(w, "błąd serwera", http.StatusInternalServerError)
	}
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	envelope, err := s.open(w, r, share.ActionSubmit)
	if err != nil {
		s.refuse(w, err)

		return
	}

	payload, err := share.ParseCanonical(envelope.Payload)
	if err != nil {
		s.refuse(w, errBad{err.Error()})

		return
	}

	if err := s.store.Add(envelope, payload.Flags(s.cfg.Models)); err != nil {
		s.refuse(w, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintln(w, `{"ok":true,"stored":true}`)
}

func (s *Server) forget(w http.ResponseWriter, r *http.Request) {
	envelope, err := s.open(w, r, share.ActionForget)
	if err != nil {
		s.refuse(w, err)

		return
	}

	if string(envelope.Payload) != "{}" {
		s.refuse(w, errBad{"zły format"})

		return
	}

	if err := s.store.Forget(envelope.Key); err != nil {
		s.refuse(w, err)

		return
	}

	// The withdrawal is on disk and the key refused; the janitor removes
	// the records within seconds, and finishes after a crash.
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintln(w, `{"ok":true,"forgotten":true}`)
}

// Janitor runs the store's janitor until stop is closed.
func (s *Server) Janitor(stop <-chan struct{}) {
	s.store.Janitor(stop, nil)
}
