package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// stubSharing is a server that takes its time: Send waits for release or
// for the deadline, like a server that does not answer.
type stubSharing struct {
	consent bool
	pending bool
	release chan struct{}
	sent    []share.Payload
	forgot  int
}

func (s *stubSharing) Consented() bool { return s.consent }

func (s *stubSharing) ForgetPending() bool { return s.pending }

func (s *stubSharing) SetConsent(yes bool) error {
	s.consent = yes
	s.pending = s.pending || !yes

	return nil
}

func (s *stubSharing) Game() string { return "neuro test" }

func (s *stubSharing) Send(ctx context.Context, payload share.Payload) error {
	select {
	case <-s.release:
		s.sent = append(s.sent, payload)

		return nil
	case <-ctx.Done():
		return share.ErrUnavailable
	}
}

func (s *stubSharing) Forget(context.Context) error {
	s.forgot++
	s.pending = false

	return nil
}

func sharingGame(t *testing.T, sharing *stubSharing) tea.Model {
	t.Helper()

	model := pausedGame(t, Pause{Share: sharing}, nil)
	g := model.(Game)
	g.realm.ExchangeBy(g.name, "run23", game.NarrateRequest{Text: "Ile za miecz?", Merchant: game.Merchant{Name: "Borin"}}, "[spokój] Dwadzieścia.", nil)

	// Esc opens the pause menu; the share entry is the one before the last.
	model = press(model, tea.KeyEsc)
	for range len(model.(Game).pauseItems()) - 2 {
		model = press(model, tea.KeyDown)
	}

	return press(model, tea.KeyEnter)
}

func TestShareAsksForConsentFirst(t *testing.T) {
	sharing := &stubSharing{release: make(chan struct{})}
	model := sharingGame(t, sharing)

	assertShows(t, model, "consent", "Udostępnij rozmowy", "Zgadzam się", "Nie teraz")
	assertFits(t, model, sizes[1], "consent")

	model = press(model, tea.KeyEnter)
	if !sharing.consent {
		t.Fatal("consent not kept")
	}

	assertShows(t, model, "consented", "Wyślij nowe rozmowy z tej gry (2 kwestii)")

	model = press(model, tea.KeyDown, tea.KeyEnter)
	assertShows(t, model, "preview", "Ile za miecz?", "Borin")

	if len(sharing.sent) != 0 {
		t.Fatal("something was sent before the player asked")
	}
}

// Sending never waits for the server: Enter returns at once, the game
// goes on, and the outcome comes later as a line.
func TestShareSendsInTheBackground(t *testing.T) {
	sharing := &stubSharing{consent: true, release: make(chan struct{})}
	model := sharingGame(t, sharing)

	began := time.Now()
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if time.Since(began) > 100*time.Millisecond || cmd == nil {
		t.Fatal("Enter waited for the server")
	}

	assertShows(t, model, "sending", "Wysyłam w tle")

	// A second Enter while the first send runs does not start another.
	model, again := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if again != nil {
		t.Fatal("second send started")
	}

	// Back to the game while the send is on its way.
	model = press(model, tea.KeyEsc, tea.KeyEsc)
	if model.(Game).pause.open {
		t.Fatal("menu stuck open")
	}

	close(sharing.release)

	model, _ = model.Update(cmd())
	if len(sharing.sent) != 1 || len(sharing.sent[0].Turns) != 2 || sharing.sent[0].Turns[1].Model != "run23" {
		t.Fatalf("sent %+v", sharing.sent)
	}

	assertShows(t, model, "thanks", "wysłano 2 kwestii")

	// What was sent is not offered again.
	model = press(model, tea.KeyEsc)
	for range len(model.(Game).pauseItems()) - 2 {
		model = press(model, tea.KeyDown)
	}

	model = press(model, tea.KeyEnter)
	assertShows(t, model, "nothing new", "(0 kwestii)")
}

func TestShareReportsAServerThatDoesNotAnswer(t *testing.T) {
	sharing := &stubSharing{consent: true, release: make(chan struct{})}
	model := sharingGame(t, sharing)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(sharedMsg{err: share.ErrUnavailable, upTo: 1, payload: 2})

	assertShows(t, model, "no answer", "nie odpowiada")

	if model.(Game).pause.sharing || model.(Game).pause.sharedUpTo != 0 {
		t.Fatal("a failed send counted as sent")
	}
}

func TestShareWithdrawsConsent(t *testing.T) {
	sharing := &stubSharing{consent: true, release: make(chan struct{})}
	model := sharingGame(t, sharing)

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if sharing.consent || cmd == nil {
		t.Fatal("consent kept")
	}

	model, _ = model.Update(cmd())
	if sharing.forgot != 1 || !strings.Contains(model.View(), "Serwer usuwa twoje rozmowy") {
		t.Fatalf("forget: %d", sharing.forgot)
	}
}

func TestClipKeepsPayloadValid(t *testing.T) {
	text := clip("a\x1b[2J b\xff"+strings.Repeat("ż", share.MaxText), share.MaxText)

	payload := share.Payload{Version: share.Version, Game: "t", Kind: share.KindTalk, Consent: true,
		Turns: []share.Turn{{Who: share.WhoPlayer, Text: text}}}
	if err := payload.Validate(); err != nil {
		t.Fatal(err)
	}
}
