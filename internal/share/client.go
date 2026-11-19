package share

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Timeout bounds one whole send: the challenge, the proof of work and the
// submission. The game never waits on it: Send runs in a tea.Cmd, and when
// the time is up the player only sees Message(err).
const Timeout = 20 * time.Second

// requestTimeout bounds each HTTP request, so a server that accepts the
// connection and then says nothing is given up quickly.
const requestTimeout = 8 * time.Second

// keyFile is where an installation keeps its signing key.
const keyFile = "share.key"

// Errors a send can end with; Message words them for the player.
var (
	ErrUnavailable = errors.New("serwer nie odpowiada")
	ErrRejected    = errors.New("serwer odrzucił dane")
	ErrBusy        = errors.New("serwer jest zajęty")
)

// Identity is an installation's signing key.
type Identity struct {
	private ed25519.PrivateKey
}

// Key returns the public key as the server sees it.
func (id Identity) Key() string {
	public, _ := id.private.Public().(ed25519.PublicKey)

	return base64.RawURLEncoding.EncodeToString(public)
}

// Seal does the proof of work for a challenge and signs payload for
// action. It gives up with ok false when stop says so.
func (id Identity) Seal(action string, challenge Challenge, payload []byte, stop func() bool) (Envelope, bool) {
	key := id.Key()

	nonce, ok := Solve(challenge.Challenge, key, challenge.Difficulty, stop)
	if !ok {
		return Envelope{}, false
	}

	return Envelope{
		Key: key, Challenge: challenge.Challenge, Nonce: nonce, Payload: payload,
		Sig: base64.RawURLEncoding.EncodeToString(ed25519.Sign(id.private, Signed(action, key, challenge.Challenge, nonce, payload))),
	}, true
}

// NewIdentity makes a fresh key, not saved anywhere.
func NewIdentity() (Identity, error) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, err
	}

	return Identity{private: private}, nil
}

// LoadIdentity reads the key kept in dir, making and saving one when there
// is none. dir is normally the game's data directory.
func LoadIdentity(dir string) (Identity, error) {
	path := filepath.Join(dir, keyFile)

	data, err := os.ReadFile(path)
	if err == nil {
		seed, decodeErr := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(data)))
		if decodeErr != nil || len(seed) != ed25519.SeedSize {
			return Identity{}, fmt.Errorf("uszkodzony klucz %s", path)
		}

		return Identity{private: ed25519.NewKeyFromSeed(seed)}, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return Identity{}, err
	}

	id, err := NewIdentity()
	if err != nil {
		return Identity{}, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Identity{}, err
	}

	// The key is written whole to a file of its own and then linked in
	// place only if none is there, so two games starting at once agree on
	// one key and a crash never leaves half of one.
	tmp, err := os.CreateTemp(dir, keyFile+".*")
	if err != nil {
		return Identity{}, err
	}
	defer os.Remove(tmp.Name())

	seed := base64.RawURLEncoding.EncodeToString(id.private.Seed())
	_, err = tmp.WriteString(seed + "\n")

	if err == nil {
		err = tmp.Sync()
	}

	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return Identity{}, err
	}

	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return LoadIdentity(dir)
		}

		return Identity{}, err
	}

	return id, nil
}

// ReadIdentity reads the key kept in dir without making one; ok is false
// when there is none, that is when nothing was ever sent from here. Any
// other trouble reading it is an error, not "nothing sent".
func ReadIdentity(dir string) (Identity, bool, error) {
	if _, err := os.Stat(filepath.Join(dir, keyFile)); errors.Is(err, os.ErrNotExist) {
		return Identity{}, false, nil
	} else if err != nil {
		return Identity{}, false, err
	}

	id, err := LoadIdentity(dir)

	return id, err == nil, err
}

// ForgetIdentity removes the key kept in dir, if it is still id, so the
// next send starts a new, unlinked installation. Call it after Forget of
// id succeeded; a key another game made since is left alone.
func ForgetIdentity(dir string, id Identity) error {
	current, ok, err := ReadIdentity(dir)
	if err != nil || !ok || current.Key() != id.Key() {
		return err
	}

	err = os.Remove(filepath.Join(dir, keyFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

// Client talks to a collecting server.
type Client struct {
	// URL is the server's base address, e.g. https://share.example.org.
	URL  string
	HTTP *http.Client
}

// NewClient returns a client with short timeouts that does not follow
// redirects: a signed conversation goes to the address given or nowhere.
func NewClient(url string) *Client {
	return &Client{URL: strings.TrimRight(url, "/"), HTTP: &http.Client{
		Timeout:       requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// check refuses a server address that is not HTTPS, other than on this
// machine (for trying the server out locally).
func (c *Client) check() error {
	parsed, err := url.Parse(c.URL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%w: zły adres serwera", ErrRejected)
	}

	host := parsed.Hostname()
	local := host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()

	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && local) {
		return fmt.Errorf("%w: adres serwera musi zaczynać się od https://", ErrRejected)
	}

	return nil
}

// Send submits a payload. It returns within Timeout whatever the server
// does; run it off the UI thread (a tea.Cmd). A payload the server already
// has counts as sent, so trying again after a lost answer is safe.
func (c *Client) Send(ctx context.Context, id Identity, payload Payload) error {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	if err := payload.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrRejected, err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if len(body) > MaxBody-4<<10 {
		return fmt.Errorf("%w: rozmowa jest za długa", ErrRejected)
	}

	var answer struct {
		OK     bool `json:"ok"`
		Stored bool `json:"stored"`
	}

	err = c.post(ctx, id, ActionSubmit, "/v1/submit", body, &answer)

	switch {
	case errors.Is(err, errConflict):
		return nil
	case err != nil:
		return err
	case !answer.OK || !answer.Stored:
		return fmt.Errorf("%w: serwer nie potwierdził zapisu", ErrUnavailable)
	}

	return nil
}

// Forget asks the server to delete everything sent with this identity. It
// succeeds only when the server confirms the withdrawal is on its disk.
func (c *Client) Forget(ctx context.Context, id Identity) error {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var answer struct {
		OK        bool `json:"ok"`
		Forgotten bool `json:"forgotten"`
	}

	err := c.post(ctx, id, ActionForget, "/v1/forget", []byte("{}"), &answer)
	if errors.Is(err, errConflict) {
		err = fmt.Errorf("%w: zła odpowiedź serwera", ErrUnavailable)
	}

	if err == nil && (!answer.OK || !answer.Forgotten) {
		err = fmt.Errorf("%w: serwer nie potwierdził usunięcia", ErrUnavailable)
	}

	return err
}

// errConflict is the server saying, to a submission, that it already has
// the payload.
var errConflict = errors.New("already stored")

// maxChallenge is the longest challenge the client takes: the server's
// are 55 characters. maxAnswer is the longest answer it reads.
const (
	maxChallenge = 64
	maxAnswer    = 16 << 10
)

func (c *Client) post(ctx context.Context, id Identity, action, path string, payload []byte, answer any) error {
	if err := c.check(); err != nil {
		return err
	}

	var challenge Challenge

	err := c.do(ctx, http.MethodGet, "/v1/challenge", nil, &challenge)
	if errors.Is(err, errConflict) {
		err = fmt.Errorf("%w: zła odpowiedź serwera", ErrUnavailable)
	}

	if err != nil {
		return err
	}

	if challenge.Challenge == "" || len(challenge.Challenge) > maxChallenge || challenge.Difficulty < 1 || challenge.Difficulty > 32 {
		return fmt.Errorf("%w: zła odpowiedź serwera", ErrUnavailable)
	}

	envelope, ok := id.Seal(action, challenge, payload, func() bool { return ctx.Err() != nil })
	if !ok {
		return fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return c.do(ctx, http.MethodPost, path, body, answer)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, into any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.URL+path, reader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := c.HTTP
	if client == nil {
		client = NewClient(c.URL).HTTP
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()

	// One byte over the limit tells a cut answer from a whole one.
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAnswer+1))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	if len(data) > maxAnswer {
		return fmt.Errorf("%w: za długa odpowiedź serwera", ErrUnavailable)
	}

	var duplicate struct {
		Duplicate bool `json:"duplicate"`
	}

	switch code := response.StatusCode; {
	case code == http.StatusConflict && method == http.MethodPost && json.Unmarshal(data, &duplicate) == nil && duplicate.Duplicate:
		return errConflict
	case code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable:
		return ErrBusy
	case code >= 400 && code < 500:
		return fmt.Errorf("%w: %s", ErrRejected, plain(string(data)))
	case code < 200 || code > 299:
		return fmt.Errorf("%w: %s", ErrUnavailable, response.Status)
	}

	if into != nil {
		if err := json.Unmarshal(data, into); err != nil {
			return fmt.Errorf("%w: zła odpowiedź serwera", ErrUnavailable)
		}
	}

	return nil
}

// Message words the outcome of a send for the player, in one line.
func Message(err error) string {
	switch {
	case err == nil:
		return "Dziękujemy, rozmowa wysłana."
	case errors.Is(err, ErrBusy):
		return "Serwer jest teraz zajęty. Rozmowa nie została wysłana, spróbuj później."
	case errors.Is(err, ErrRejected):
		return "Serwer nie przyjął rozmowy: " + plain(strings.TrimPrefix(err.Error(), ErrRejected.Error()+": "))
	default:
		return "Serwer nie odpowiada. Nie wiadomo, czy rozmowa dotarła; możesz spróbować ponownie. Gra działa dalej."
	}
}

// plain makes text from the server safe for one line of the game's
// screen: no control characters or escape sequences, at most 120 runes.
func plain(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return ' '
		}

		return r
	}, strings.ToValidUTF8(text, ""))

	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > 120 {
		text = string(runes[:120]) + "…"
	}

	return text
}
