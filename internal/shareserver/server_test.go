package shareserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

func talk(text string) share.Payload {
	return share.Payload{
		Version: share.Version, Game: "test", Kind: share.KindTalk, Consent: true,
		Turns: []share.Turn{
			{Who: share.WhoPlayer, Text: "Ile za miecz?"},
			{Who: share.WhoNPC, Name: "Borin", Text: text, Model: "run23"},
		},
	}
}

// newServer is New that releases the directory when the test ends.
func newServer(t *testing.T, cfg Config) (*Server, error) {
	t.Helper()

	server, err := New(cfg)
	if err == nil {
		t.Cleanup(func() { _ = server.Close() })
	}

	return server, err
}

func start(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()

	dir := t.TempDir()

	server, err := newServer(t, Config{Data: dir, Difficulty: 8, Limits: Limits{PerKeyDay: 3}})
	if err != nil {
		t.Fatal(err)
	}

	web := httptest.NewServer(server.Handler())
	t.Cleanup(web.Close)

	return server, web, dir
}

// unlimited lets one address send as much as it likes, so a test can reach
// the store's own limits.
func unlimited(server *Server) *httptest.Server {
	server.challengeLimit = NewLimiter(1000, 1000, 10, time.Now)
	server.writeLimit = NewLimiter(1000, 1000, 10, time.Now)
	server.networkLimit = NewLimiter(1000, 1000, 10, time.Now)

	return httptest.NewServer(server.Handler())
}

func challenge(t *testing.T, url string) share.Challenge {
	t.Helper()

	response, err := http.Get(url + "/v1/challenge")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	var c share.Challenge
	if err := json.NewDecoder(response.Body).Decode(&c); err != nil {
		t.Fatal(err)
	}

	return c
}

func postRaw(t *testing.T, url string, body []byte) (int, string) {
	t.Helper()

	response, err := http.Post(url+"/v1/submit", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	var text bytes.Buffer
	_, _ = text.ReadFrom(response.Body)

	return response.StatusCode, strings.TrimSpace(text.String())
}

func post(t *testing.T, url string, envelope share.Envelope) (int, string) {
	t.Helper()

	body, _ := json.Marshal(envelope)

	return postRaw(t, url, body)
}

func records(t *testing.T, dir string) int {
	t.Helper()

	report, err := Verify(dir)
	if err != nil || len(report.Problems) > 0 {
		t.Fatalf("verify: %+v %v", report, err)
	}

	return report.Records
}

func TestSendStoresAndVerifies(t *testing.T) {
	_, web, dir := start(t)
	client := share.NewClient(web.URL)
	id, _ := share.NewIdentity()

	for _, text := range []string{"Dwadzieścia sztuk złota.", "Trzydzieści, i ani grosza mniej."} {
		if err := client.Send(context.Background(), id, talk(text)); err != nil {
			t.Fatal(err)
		}
	}

	// Sending again after a lost answer is safe: it counts as sent and is
	// stored once.
	if err := client.Send(context.Background(), id, talk("Dwadzieścia sztuk złota.")); err != nil {
		t.Fatalf("repeat: %v", err)
	}

	if n := records(t, dir); n != 2 {
		t.Fatalf("records %d", n)
	}

	// Another player may say the very same thing.
	other, _ := share.NewIdentity()
	if err := client.Send(context.Background(), other, talk("Dwadzieścia sztuk złota.")); err != nil {
		t.Fatal(err)
	}

	if n := records(t, dir); n != 3 {
		t.Fatalf("records %d", n)
	}
}

func TestReplayTamperAndWorkAreRefused(t *testing.T) {
	_, web, _ := start(t)
	id, _ := share.NewIdentity()
	payload, _ := json.Marshal(talk("Dwadzieścia sztuk złota."))

	envelope, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), payload, nil)
	if code, text := post(t, web.URL, envelope); code != http.StatusCreated {
		t.Fatalf("first: %d %s", code, text)
	}

	if code, text := post(t, web.URL, envelope); code != http.StatusBadRequest || text != "challenge już użyty" {
		t.Fatalf("replay: %d %s", code, text)
	}

	tampered, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), payload, nil)
	tampered.Payload = bytes.Replace(tampered.Payload, []byte("Dwadzieścia"), []byte("Dwieście"), 1)

	if code, text := post(t, web.URL, tampered); code != http.StatusBadRequest || text != "zły podpis" {
		t.Fatalf("tampered: %d %s", code, text)
	}

	lazy, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), payload, nil)
	for n := 0; ; n++ {
		lazy.Nonce = strconv.Itoa(n)
		if share.Work(lazy.Challenge, lazy.Key, lazy.Nonce) < 8 {
			break
		}
	}

	if code, text := post(t, web.URL, lazy); code != http.StatusBadRequest || text != "brak dowodu pracy" {
		t.Fatalf("no work: %d %s", code, text)
	}
}

func TestOnlyCanonicalFormsPass(t *testing.T) {
	server, _, _ := start(t)
	web := unlimited(server)
	defer web.Close()

	id, _ := share.NewIdentity()

	// Each of these is valid JSON that another reader could take for
	// something else than what was checked.
	for name, raw := range map[string]string{
		"no consent":   `{"version":1,"game":"t","kind":"rozmowa","consent":false,"turns":[{"who":"gracz","text":"x"}]}`,
		"unknown":      `{"version":1,"game":"t","kind":"rozmowa","consent":true,"turns":[{"who":"gracz","text":"x"}],"extra":1}`,
		"duplicate":    `{"version":1,"game":"t","kind":"rozmowa","consent":false,"consent":true,"turns":[{"who":"gracz","text":"x"}]}`,
		"letter case":  `{"version":1,"game":"t","kind":"rozmowa","Consent":true,"turns":[{"who":"gracz","text":"x"}]}`,
		"order":        `{"game":"t","version":1,"kind":"rozmowa","consent":true,"turns":[{"who":"gracz","text":"x"}]}`,
		"spaced":       `{ "version":1,"game":"t","kind":"rozmowa","consent":true,"turns":[{"who":"gracz","text":"x"}]}`,
		"missing turn": `{"version":1,"game":"t","kind":"ocena","consent":true,"turns":[{"who":"gracz","text":"x"}],"rating":{"value":1}}`,
		"bad utf8":     "{\"version\":1,\"game\":\"t\",\"kind\":\"rozmowa\",\"consent\":true,\"turns\":[{\"who\":\"gracz\",\"text\":\"\xff\"}]}",
		"not json":     `"x"`,
	} {
		envelope, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), []byte(raw), nil)
		if code, text := post(t, web.URL, envelope); code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, code, text)
		}
	}

	good, _ := json.Marshal(talk("ok"))
	envelope, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), good, nil)
	body, _ := json.Marshal(envelope)

	if code, _ := postRaw(t, web.URL, append(body, []byte(`{"more":1}`)...)); code != http.StatusBadRequest {
		t.Errorf("trailing data: %d", code)
	}

	// The same key bytes spelled with padding are another spelling.
	envelope, _ = id.Seal(share.ActionSubmit, challenge(t, web.URL), good, nil)
	envelope.Key += "="

	if code, _ := post(t, web.URL, envelope); code != http.StatusBadRequest {
		t.Errorf("padded key: %d", code)
	}
}

func TestQuotaPerKey(t *testing.T) {
	server, _, _ := start(t)
	web := unlimited(server)
	defer web.Close()

	client := share.NewClient(web.URL)
	id, _ := share.NewIdentity()

	for i := range 3 {
		if err := client.Send(context.Background(), id, talk(strings.Repeat("a", i+1))); err != nil {
			t.Fatal(err)
		}
	}

	if err := client.Send(context.Background(), id, talk("dość")); !errors.Is(err, share.ErrBusy) {
		t.Fatalf("quota: %v", err)
	}
}

func TestDailyCapOnEveryone(t *testing.T) {
	dir := t.TempDir()

	server, err := newServer(t, Config{Data: dir, Difficulty: 8, Limits: Limits{PerDay: 2}})
	if err != nil {
		t.Fatal(err)
	}

	web := unlimited(server)
	defer web.Close()

	client := share.NewClient(web.URL)

	for i := range 3 {
		id, _ := share.NewIdentity()
		err := client.Send(context.Background(), id, talk(strings.Repeat("c", i+1)))

		if i < 2 && err != nil || i == 2 && !errors.Is(err, share.ErrBusy) {
			t.Fatalf("send %d: %v", i, err)
		}
	}
}

func TestRateLimitPerAddress(t *testing.T) {
	_, web, _ := start(t)
	client := share.NewClient(web.URL)

	var err error

	for i := range 12 {
		id, _ := share.NewIdentity()

		err = client.Send(context.Background(), id, talk(strings.Repeat("b", i+1)))
		if err != nil {
			break
		}
	}

	if !errors.Is(err, share.ErrBusy) || !strings.Contains(share.Message(err), "zajęty") {
		t.Fatalf("rate limit: %v", err)
	}
}

func TestFullTableSharesOneBucket(t *testing.T) {
	limiter := NewLimiter(1.0/60, 1, 2, time.Now)

	if !limiter.Allow("a") || !limiter.Allow("b") {
		t.Fatal("first addresses")
	}

	// The table is full: newcomers share the overflow bucket instead of
	// all being shut out.
	if !limiter.Allow("c") || !limiter.Allow("d") {
		t.Fatal("newcomers shut out")
	}

	if limiter.Allow("a") {
		t.Fatal("known address got a second token")
	}
}

func TestForgetRemovesAndRefusesTheKey(t *testing.T) {
	server, web, dir := start(t)
	client := share.NewClient(web.URL)
	id, _ := share.NewIdentity()
	other, _ := share.NewIdentity()

	_ = client.Send(context.Background(), other, talk("Cudzy wpis, zostaje."))
	_ = client.Send(context.Background(), id, talk("Mój wpis."))

	if err := client.Forget(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	// The key is refused at once; the janitor removes its records.
	if err := server.store.sweep(false); err != nil {
		t.Fatal(err)
	}

	if n := records(t, dir); n != 1 {
		t.Fatalf("after forget: %d", n)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	data, _ := os.ReadFile(files[0])

	if strings.Contains(string(data), id.Key()) || !strings.Contains(string(data), other.Key()) {
		t.Fatal("wrong records removed")
	}

	// A send of the forgotten key, even one signed before the forget, is
	// refused for good; the game sends with a new key after a forget.
	err := client.Send(context.Background(), id, talk("Jeszcze raz."))
	if !errors.Is(err, share.ErrRejected) || !strings.Contains(share.Message(err), "wycofała zgodę") {
		t.Fatalf("forgotten key: %v", err)
	}

	fresh, _ := share.NewIdentity()
	if err := client.Send(context.Background(), fresh, talk("Mój wpis.")); err != nil {
		t.Fatal(err)
	}
}

func TestForgottenStaysForgottenAfterRestart(t *testing.T) {
	dir := t.TempDir()

	server, _ := newServer(t, Config{Data: dir, Difficulty: 8})
	web := httptest.NewServer(server.Handler())
	id, _ := share.NewIdentity()

	_ = share.NewClient(web.URL).Forget(context.Background(), id)
	web.Close()
	server.Close()

	again, _ := newServer(t, Config{Data: dir, Difficulty: 8})
	web = httptest.NewServer(again.Handler())

	defer web.Close()

	if err := share.NewClient(web.URL).Send(context.Background(), id, talk("x")); !errors.Is(err, share.ErrRejected) {
		t.Fatalf("after restart: %v", err)
	}
}

func TestVerifyFindsTampering(t *testing.T) {
	_, web, dir := start(t)
	client := share.NewClient(web.URL)
	id, _ := share.NewIdentity()

	for _, text := range []string{"Pierwszy.", "Drugi.", "Trzeci."} {
		_ = client.Send(context.Background(), id, talk(text))
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	data, _ := os.ReadFile(files[0])
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Changing a word breaks the signature; dropping a line breaks the chain.
	changed := strings.Replace(lines[0], "Pierwszy.", "Pierwsza.", 1)
	_ = os.WriteFile(files[0], []byte(changed+"\n"+lines[2]+"\n"), 0o600)

	report, _ := Verify(dir)

	joined := strings.Join(report.Problems, "\n")
	if !strings.Contains(joined, "podpis") || !strings.Contains(joined, "łańcuch") {
		t.Fatalf("problems: %v", report.Problems)
	}
}

func TestRestartRepairsTailAndRefusesOldChallenges(t *testing.T) {
	dir := t.TempDir()

	server, err := newServer(t, Config{Data: dir, Difficulty: 8})
	if err != nil {
		t.Fatal(err)
	}

	web := httptest.NewServer(server.Handler())
	id, _ := share.NewIdentity()
	_ = share.NewClient(web.URL).Send(context.Background(), id, talk("Przed restartem."))

	payload, _ := json.Marshal(talk("Podpisane przed restartem."))
	old, _ := id.Seal(share.ActionSubmit, challenge(t, web.URL), payload, nil)
	web.Close()
	server.Close()

	// A crash in the middle of a write leaves half a line.
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	f, _ := os.OpenFile(files[0], os.O_APPEND|os.O_WRONLY, 0o600)
	_, _ = f.WriteString(`{"received":"2026`)
	f.Close()

	again, err := newServer(t, Config{Data: dir, Difficulty: 8})
	if err != nil {
		t.Fatalf("restart after a torn write: %v", err)
	}

	web = httptest.NewServer(again.Handler())
	defer web.Close()

	if code, text := post(t, web.URL, old); code != http.StatusBadRequest || text != "zły challenge" {
		t.Fatalf("challenge from before the restart: %d %s", code, text)
	}

	client := share.NewClient(web.URL)
	if err := client.Send(context.Background(), id, talk("Przed restartem.")); err != nil {
		t.Fatalf("repeat after restart: %v", err)
	}

	if err := client.Send(context.Background(), id, talk("Po restarcie.")); err != nil {
		t.Fatal(err)
	}

	if n := records(t, dir); n != 2 {
		t.Fatalf("records %d", n)
	}
}

func TestWeakKeysAreRefused(t *testing.T) {
	_, web, _ := start(t)
	c := challenge(t, web.URL)
	payload, _ := json.Marshal(talk("x"))

	// The neutral point verifies a signature (R = neutral, S = 0) of any
	// message, without any secret behind it.
	neutral := make([]byte, 32)
	neutral[0] = 1
	key := base64.RawURLEncoding.EncodeToString(neutral)
	sig := make([]byte, 64)
	sig[0] = 1

	envelope := share.Envelope{Key: key, Challenge: c.Challenge, Payload: payload, Sig: base64.RawURLEncoding.EncodeToString(sig)}
	envelope.Nonce, _ = share.Solve(c.Challenge, key, c.Difficulty, nil)

	if code, text := post(t, web.URL, envelope); code != http.StatusBadRequest || text != "zły klucz" {
		t.Fatalf("neutral key: %d %s", code, text)
	}

	for name, weak := range map[string]string{
		"order 2":       "ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"order 8":       "26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85",
		"non-canonical": "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
	} {
		raw, _ := hex.DecodeString(weak)
		if !share.WeakKey(raw) {
			t.Errorf("%s passed", name)
		}
	}

	id, _ := share.NewIdentity()
	real, _ := base64.RawURLEncoding.DecodeString(id.Key())

	if share.WeakKey(real) {
		t.Error("a real key counted as weak")
	}
}

func TestRemovalCutShortIsFinishedAtStart(t *testing.T) {
	dir := t.TempDir()

	server, _ := newServer(t, Config{Data: dir, Difficulty: 8})
	web := httptest.NewServer(server.Handler())
	id, _ := share.NewIdentity()
	other, _ := share.NewIdentity()
	client := share.NewClient(web.URL)

	_ = client.Send(context.Background(), id, talk("Do usunięcia."))
	_ = client.Send(context.Background(), other, talk("Zostaje."))

	// The withdrawal reached the disk, then the process died before the
	// janitor ran.
	if err := client.Forget(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	web.Close()
	server.Close()

	again, err := newServer(t, Config{Data: dir, Difficulty: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()

	if err := again.store.sweep(true); err != nil {
		t.Fatal(err)
	}

	report, _ := Verify(dir)
	if report.Records != 1 || len(report.Problems) > 0 {
		t.Fatalf("after the start: %+v", report)
	}
}

func TestVerifyReportsRecordsOfWithdrawnKeys(t *testing.T) {
	_, web, dir := start(t)
	client := share.NewClient(web.URL)
	id, _ := share.NewIdentity()

	_ = client.Send(context.Background(), id, talk("Jeszcze tu jestem."))
	_ = client.Forget(context.Background(), id)

	report, _ := Verify(dir)
	if !strings.Contains(strings.Join(report.Problems, "\n"), "wycofał") {
		t.Fatalf("problems: %v", report.Problems)
	}
}

func TestDifficultyOutOfRange(t *testing.T) {
	for _, difficulty := range []int{-1, 33, 256} {
		if _, err := newServer(t, Config{Data: t.TempDir(), Difficulty: difficulty}); err == nil {
			t.Errorf("difficulty %d accepted", difficulty)
		}
	}
}

func TestProxyHeaderOnlyFromPrivatePeer(t *testing.T) {
	_, proxy, _ := net.ParseCIDR("172.18.0.3/32")
	server := &Server{cfg: Config{Proxies: []*net.IPNet{proxy}}}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.7")

	request.RemoteAddr = "172.18.0.3:5555"
	if address, network := server.address(request); address != "203.0.113.7" || network != "203.0.113.0/24" {
		t.Errorf("from the proxy: %s %s", address, network)
	}

	// Another private peer is not the proxy either.
	for _, peer := range []string{"198.51.100.9", "172.18.0.4"} {
		request.RemoteAddr = peer + ":5555"
		if address, _ := server.address(request); address != peer {
			t.Errorf("from %s: %s", peer, address)
		}
	}
}

func TestClientDoesNotFollowRedirectsOrUsePlainHTTP(t *testing.T) {
	received := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { received = true }))
	defer elsewhere.Close()

	moved := httptest.NewServer(http.RedirectHandler(elsewhere.URL+"/v1/challenge", http.StatusTemporaryRedirect))
	defer moved.Close()

	id, _ := share.NewIdentity()

	if err := share.NewClient(moved.URL).Send(context.Background(), id, talk("x")); err == nil || received {
		t.Fatalf("redirect followed: %v", err)
	}

	err := share.NewClient("http://share.example.org").Send(context.Background(), id, talk("x"))
	if !errors.Is(err, share.ErrRejected) || !strings.Contains(err.Error(), "https") {
		t.Fatalf("plain http: %v", err)
	}
}

func TestServerTextIsCleanedForTheScreen(t *testing.T) {
	rude := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "\x1b[2J\nzłe\tdane"+strings.Repeat("!", 500), http.StatusBadRequest)
	}))
	defer rude.Close()

	id, _ := share.NewIdentity()
	message := share.Message(share.NewClient(rude.URL).Send(context.Background(), id, talk("x")))

	if strings.ContainsAny(message, "\x1b\n\t") || len([]rune(message)) > 200 {
		t.Fatalf("message %q", message)
	}
}

// The game must never hang on the server: a server that says nothing, at
// any step, or one that is not there, ends the send with a message.
func TestSilentOrMissingServerDoesNotHang(t *testing.T) {
	silent := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer silent.Close()

	// This one hands out a challenge and then hangs on the submission.
	real, _, _ := start(t)
	halfway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			real.Handler().ServeHTTP(w, r)

			return
		}

		// Read the body first: only then does the server notice the client
		// leaving, and Close can finish.
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer halfway.Close()

	missing := httptest.NewServer(http.NotFoundHandler())
	missingURL := missing.URL
	missing.Close()

	id, _ := share.NewIdentity()

	for name, url := range map[string]string{"silent": silent.URL, "halfway": halfway.URL, "missing": missingURL} {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		began := time.Now()
		err := share.NewClient(url).Send(ctx, id, talk("Halo?"))

		cancel()

		if !errors.Is(err, share.ErrUnavailable) {
			t.Errorf("%s: %v", name, err)
		}

		if took := time.Since(began); took > 2*time.Second {
			t.Errorf("%s: took %v", name, took)
		}

		if !strings.Contains(share.Message(err), "nie odpowiada") {
			t.Errorf("%s: message %q", name, share.Message(err))
		}
	}
}
