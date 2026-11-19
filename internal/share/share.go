// Package share is the protocol by which a player who agreed to it sends
// conversations from the game to the collecting server, to improve the
// model. It is shared by the game (the client) and cmd/neuro-share (the
// server), so both read the same rules.
//
// Every submission is:
//   - signed with an Ed25519 key made on the player's machine, so the server
//     knows it was not changed on the way and which installation sent it
//     (and can forget everything of that installation on request);
//   - bound to a one-time challenge from the server, so it cannot be replayed;
//   - paid for with a small proof of work over the challenge and the key, so
//     sending thousands of fake submissions costs real computing time;
//   - checked against the schema below before it is stored.
//
// None of this proves that a human played the game: the client is open
// source and its key is the player's own. It makes the data intact,
// attributable, deletable and expensive to flood.
package share

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Domain separates the bytes signed and hashed here from any other use of
// the same key.
const Domain = "neuro-share/v1"

// Version is the payload version this package writes and accepts.
const Version = 1

// Actions name what an envelope asks for; each is signed into it.
const (
	ActionSubmit = "submit"
	ActionForget = "forget"
)

// Kinds of payload.
const (
	KindTalk   = "rozmowa" // a stretch of conversation
	KindRating = "ocena"   // a stretch with one line rated by the player
)

// Speakers of a turn.
const (
	WhoPlayer = "gracz"
	WhoNPC    = "npc"
	WhoGM     = "mg"
)

// Limits of a payload. MaxBody bounds the whole request on the server.
const (
	MaxBody     = 128 << 10
	MaxTurns    = 300
	MaxText     = 2000
	MaxName     = 80
	MaxModel    = 120
	MaxNote     = 500
	MaxGame     = 40
	MaxWorld    = 120
	MaxCreated  = 40
	nonceDigits = 20
)

// Turn is one line of a conversation.
type Turn struct {
	Who   string `json:"who"`
	Name  string `json:"name,omitempty"`
	Text  string `json:"text"`
	Model string `json:"model,omitempty"`
}

// Rating is the player's judgement of one turn: +1 good, -1 bad.
type Rating struct {
	Turn  int    `json:"turn"`
	Value int    `json:"value"`
	Note  string `json:"note,omitempty"`
}

// Payload is what the player agreed to send.
type Payload struct {
	Version int     `json:"version"`
	Game    string  `json:"game"`
	Kind    string  `json:"kind"`
	Consent bool    `json:"consent"`
	Created string  `json:"created,omitempty"`
	World   string  `json:"world,omitempty"`
	Turns   []Turn  `json:"turns"`
	Rating  *Rating `json:"rating,omitempty"`
}

// Envelope carries a payload with its key, challenge, proof of work and
// signature. Payload is kept as the exact bytes sent, which are what is
// signed.
type Envelope struct {
	Key       string          `json:"key"`
	Challenge string          `json:"challenge"`
	Nonce     string          `json:"nonce"`
	Payload   json.RawMessage `json:"payload"`
	Sig       string          `json:"sig"`
}

// Challenge is the server's answer to a request for one.
type Challenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
}

// Signed returns the bytes an envelope's signature covers.
func Signed(action, key, challenge, nonce string, payload []byte) []byte {
	sum := sha256.Sum256(payload)

	return []byte(strings.Join([]string{Domain, action, key, challenge, nonce, hex.EncodeToString(sum[:])}, "\n"))
}

// Work returns the number of leading zero bits of the proof-of-work hash
// for a challenge, key and nonce.
func Work(challenge, key, nonce string) int {
	sum := sha256.Sum256([]byte(Domain + "\n" + challenge + "\n" + key + "\n" + nonce))

	zeros := 0
	for i := 0; i < len(sum); i += 8 {
		word := binary.BigEndian.Uint64(sum[i : i+8])
		zeros += bits.LeadingZeros64(word)

		if word != 0 {
			break
		}
	}

	return zeros
}

// Solve finds a nonce whose work reaches difficulty. It checks stop every
// few thousand tries and gives up with ok false when it says so.
func Solve(challenge, key string, difficulty int, stop func() bool) (string, bool) {
	for n := uint64(0); ; n++ {
		if n%4096 == 0 && stop != nil && stop() {
			return "", false
		}

		nonce := strconv.FormatUint(n, 10)
		if Work(challenge, key, nonce) >= difficulty {
			return nonce, true
		}
	}
}

// smallOrder are the encodings of the Ed25519 points of small order, with
// the sign bit cleared. A "public key" among them proves nothing: a
// signature can verify without anyone holding a secret.
var smallOrder = [][32]byte{
	{0x01}, // the neutral point
	{0xec, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, // order 2
	{}, // order 4
	{0x26, 0xe8, 0x95, 0x8f, 0xc2, 0xb2, 0x27, 0xb0, 0x45, 0xc3, 0xf4, 0x89, 0xf2, 0xef, 0x98, 0xf0, 0xd5, 0xdf, 0xac, 0x05, 0xd3, 0xc6, 0x33, 0x39, 0xb1, 0x38, 0x02, 0x88, 0x6d, 0x53, 0xfc, 0x05}, // order 8
	{0xc7, 0x17, 0x6a, 0x70, 0x3d, 0x4d, 0xd8, 0x4f, 0xba, 0x3c, 0x0b, 0x76, 0x0d, 0x10, 0x67, 0x0f, 0x2a, 0x20, 0x53, 0xfa, 0x2c, 0x39, 0xcc, 0xc6, 0x4e, 0xc7, 0xfd, 0x77, 0x92, 0xac, 0x03, 0x7a}, // order 8
}

// WeakKey reports whether a 32-byte Ed25519 public key is a point of small
// order or a non-canonical encoding (y not below 2^255-19). The keys the
// game makes never are.
func WeakKey(key []byte) bool {
	if len(key) != 32 {
		return true
	}

	var masked [32]byte
	copy(masked[:], key)
	masked[31] &= 0x7f

	for _, point := range smallOrder {
		if masked == point {
			return true
		}
	}

	// y >= p: bytes 1..30 all 0xff, the top byte 0x7f, the lowest >= 0xed.
	if masked[31] != 0x7f || masked[0] < 0xed {
		return false
	}

	for _, b := range masked[1:31] {
		if b != 0xff {
			return false
		}
	}

	return true
}

// ValidNonce reports whether a nonce has the form Solve writes.
func ValidNonce(nonce string) bool {
	if nonce == "" || len(nonce) > nonceDigits {
		return false
	}

	_, err := strconv.ParseUint(nonce, 10, 64)

	return err == nil
}

// Validate checks a payload against the schema and limits.
func (p Payload) Validate() error {
	if p.Version != Version {
		return fmt.Errorf("wersja %d, oczekiwana %d", p.Version, Version)
	}

	if !p.Consent {
		return errors.New("brak zgody")
	}

	if p.Kind != KindTalk && p.Kind != KindRating {
		return fmt.Errorf("nieznany rodzaj %q", p.Kind)
	}

	for _, field := range []struct {
		name, value string
		limit       int
	}{{"game", p.Game, MaxGame}, {"world", p.World, MaxWorld}, {"created", p.Created, MaxCreated}} {
		if err := checkText(field.name, field.value, field.limit, false); err != nil {
			return err
		}
	}

	if len(p.Turns) == 0 || len(p.Turns) > MaxTurns {
		return fmt.Errorf("liczba kwestii %d poza 1..%d", len(p.Turns), MaxTurns)
	}

	for index, turn := range p.Turns {
		if turn.Who != WhoPlayer && turn.Who != WhoNPC && turn.Who != WhoGM {
			return fmt.Errorf("kwestia %d: nieznany mówca %q", index, turn.Who)
		}

		if strings.TrimSpace(turn.Text) == "" {
			return fmt.Errorf("kwestia %d: pusta", index)
		}

		if err := checkText(fmt.Sprintf("kwestia %d", index), turn.Text, MaxText, true); err != nil {
			return err
		}

		if err := checkText(fmt.Sprintf("kwestia %d: imię", index), turn.Name, MaxName, false); err != nil {
			return err
		}

		if err := checkText(fmt.Sprintf("kwestia %d: model", index), turn.Model, MaxModel, false); err != nil {
			return err
		}
	}

	if p.Kind == KindRating && p.Rating == nil {
		return errors.New("ocena bez oceny")
	}

	if p.Kind == KindTalk && p.Rating != nil {
		return errors.New("ocena w rozmowie")
	}

	if p.Rating != nil {
		if p.Rating.Turn < 0 || p.Rating.Turn >= len(p.Turns) {
			return fmt.Errorf("ocena kwestii %d poza zakresem", p.Rating.Turn)
		}

		if p.Rating.Value != 1 && p.Rating.Value != -1 {
			return fmt.Errorf("ocena %d, dozwolone 1 i -1", p.Rating.Value)
		}

		if err := checkText("uwaga do oceny", p.Rating.Note, MaxNote, true); err != nil {
			return err
		}
	}

	return nil
}

// ParseCanonical reads a payload as the server takes it: exactly one JSON
// object, no unknown fields, valid, and byte for byte what json.Marshal
// writes for it. The last rule turns away duplicate keys, other letter
// case, missing fields, replaced invalid UTF-8 and any other form that
// another JSON reader could read differently from this one.
func ParseCanonical(raw []byte) (Payload, error) {
	var payload Payload

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		return payload, errors.New("zły format danych")
	}

	if err := payload.Validate(); err != nil {
		return payload, err
	}

	again, err := json.Marshal(payload)
	if err != nil || !bytes.Equal(again, raw) {
		return payload, errors.New("dane nie są w postaci kanonicznej")
	}

	return payload, nil
}

// checkText rejects invalid UTF-8, text over limit runes and control
// characters (newlines and tabs are allowed where lines is true).
func checkText(name, text string, limit int, lines bool) error {
	if !utf8.ValidString(text) {
		return fmt.Errorf("%s: niepoprawne UTF-8", name)
	}

	if utf8.RuneCountInString(text) > limit {
		return fmt.Errorf("%s: dłuższe niż %d znaków", name, limit)
	}

	for _, r := range text {
		if lines && (r == '\n' || r == '\t') {
			continue
		}

		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return fmt.Errorf("%s: znak sterujący", name)
		}
	}

	return nil
}

// Flags are remarks the server stores with a payload that passed
// validation but looks thin; they do not reject it.
func (p Payload) Flags(knownModels map[string]bool) []string {
	var flags []string

	players, models := 0, 0
	seen := map[string]int{}
	repeated := 0

	for _, turn := range p.Turns {
		if turn.Who == WhoPlayer {
			players++
		}

		if turn.Model != "" {
			models++

			if len(knownModels) > 0 && !knownModels[turn.Model] {
				flags = appendOnce(flags, "nieznany-model")
			}
		}

		text := strings.ToLower(strings.TrimSpace(turn.Text))
		seen[text]++

		if seen[text] > 1 {
			repeated++
		}
	}

	if len(p.Turns) < 2 {
		flags = append(flags, "krotka")
	}

	if players == 0 {
		flags = append(flags, "bez-gracza")
	}

	if models == 0 {
		flags = append(flags, "bez-modelu")
	}

	if len(p.Turns) >= 4 && repeated*2 >= len(p.Turns) {
		flags = append(flags, "powtorzenia")
	}

	return flags
}

func appendOnce(list []string, value string) []string {
	for _, have := range list {
		if have == value {
			return list
		}
	}

	return append(list, value)
}
