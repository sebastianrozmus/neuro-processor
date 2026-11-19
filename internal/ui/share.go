package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// Sharing is what the host gives the pause menu to send conversations to
// the collecting server, when the player agrees to it.
type Sharing interface {
	// Consented reports whether the player agreed; SetConsent keeps the
	// answer for the next game (withdrawing also owes a deletion).
	Consented() bool
	SetConsent(yes bool) error
	// ForgetPending reports a withdrawal the server has not confirmed.
	ForgetPending() bool
	// Send submits a payload; Forget asks the server to delete everything
	// this installation sent and drops its key. Both return within
	// share.Timeout and are only ever called off the UI thread.
	Send(ctx context.Context, payload share.Payload) error
	Forget(ctx context.Context) error
	// Game names the game version sent with the payload.
	Game() string
}

// sharedMsg is the outcome of a send or a forget, done in the background.
type sharedMsg struct {
	err     error
	forgot  bool
	upTo    int
	payload int
}

// consentText is what the player agrees to: every field that is sent.
const consentText = "Możesz wysyłać rozmowy z gry autorowi, żeby ulepszać model. " +
	"Wysyłane są: teksty kwestii (twoje, postaci i Mistrza Gry), imiona postaci, nazwa świata, " +
	"nazwa modelu, który napisał kwestię, wersja gry i losowy identyfikator tej instalacji; serwer dopisuje czas odbioru. " +
	"Nie są wysyłane: głos, twoje imię w grze, adres IP (serwer go nie zapisuje), dane komputera. " +
	"Rozmowy mogą trafić do danych treningowych modelu. Nie wpisuj w grze prawdziwych danych osobowych. " +
	"Wysyłasz zawsze ręcznie, a przed wysłaniem możesz podejrzeć całość. Zgodę możesz w każdej chwili wycofać: " +
	"serwer usunie wtedy twoje rozmowy, ale modelu już wytrenowanego na nich nie da się z nich wyczyścić. " +
	"Potwierdzasz, że masz co najmniej 16 lat."

// The rows of the share screen.
const (
	shareAgree = iota
	shareDecline
	shareSend
	sharePreview
	shareWithdraw
	shareRetryForget
	shareBack
)

// previewPage is how many screen lines a page of the preview shows.
const previewPage = 14

// batchBytes is what a payload may take, leaving room for the envelope.
const batchBytes = share.MaxBody - 8<<10

func (g Game) shareRows() []int {
	switch sharing := g.pauser.Share; {
	case sharing.Consented():
		return []int{shareSend, sharePreview, shareWithdraw, shareBack}
	case sharing.ForgetPending():
		// Consent comes back only once the old key is forgotten.
		return []int{shareRetryForget, shareDecline}
	default:
		return []int{shareAgree, shareDecline}
	}
}

// sharePayload gathers the player's oldest exchanges not sent yet, as many
// as fit one payload; upTo is the last exchange it holds, so that what does
// not fit goes in the next one.
func (g Game) sharePayload() (share.Payload, int) {
	payload := share.Payload{
		Version: share.Version, Game: clipLine(g.pauser.Share.Game(), share.MaxGame), Kind: share.KindTalk, Consent: true,
		World: clipLine(g.view.World.Name, share.MaxWorld),
	}

	upTo := g.pause.sharedUpTo
	size := 512

	for _, exchange := range g.realm.Inspect().Exchanges {
		if exchange.Player != g.name || exchange.ID <= g.pause.sharedUpTo {
			continue
		}

		var turns []share.Turn

		if said := strings.TrimSpace(clip(exchange.Said, share.MaxText)); said != "" {
			turns = append(turns, share.Turn{Who: share.WhoPlayer, Text: said})
		}

		if reply := strings.TrimSpace(clip(exchange.Reply, share.MaxText)); reply != "" {
			who := share.WhoNPC
			if exchange.Keeper == game.MasterName {
				who = share.WhoGM
			}

			turns = append(turns, share.Turn{
				Who: who, Name: clipLine(exchange.Keeper, share.MaxName), Text: reply, Model: clipLine(exchange.Model, share.MaxModel),
			})
		}

		grown := size
		for _, turn := range turns {
			raw, _ := json.Marshal(turn)
			grown += len(raw) + 1
		}

		if len(payload.Turns)+len(turns) > share.MaxTurns || grown > batchBytes {
			break
		}

		payload.Turns, size, upTo = append(payload.Turns, turns...), grown, exchange.ID
	}

	return payload, upTo
}

// clip cuts text to limit runes and drops control characters other than
// newlines, so a payload the game builds always passes Validate.
func clip(text string, limit int) string {
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r >= ' ' && r != 0x7f && (r < 0x80 || r > 0x9f) && r != ' ' && r != ' ' {
			return r
		}

		return -1
	}, strings.ToValidUTF8(text, ""))

	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	return string([]rune(text)[:limit])
}

// clipLine is clip for one-line fields (names, models, the world), where
// the server takes no newlines or tabs either.
func clipLine(text string, limit int) string {
	return clip(strings.Join(strings.Fields(text), " "), limit)
}

// openShare opens the share screen. A payload whose send ended without an
// answer is kept as it was, so that trying again sends the same bytes and
// the server, which already may have them, does not store them twice.
func (g Game) openShare() Game {
	g.pause.screen, g.pause.cursor, g.pause.note, g.pause.preview, g.pause.page = pauseShare, 0, "", false, 0

	if !g.pause.retry {
		g.pause.outbox, g.pause.outboxUpTo = g.sharePayload()
	}

	return g
}

func (g Game) shareKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := g.shareRows()
	g.pause.cursor = min(g.pause.cursor, len(rows)-1)
	pages := 1
	if g.pause.preview {
		pages = max(1, (len(g.previewLines())+previewPage-1)/previewPage)
	}

	switch msg.Type {
	case tea.KeyEsc:
		g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
	case tea.KeyUp:
		g.pause.cursor = (g.pause.cursor + len(rows) - 1) % len(rows)
	case tea.KeyDown:
		g.pause.cursor = (g.pause.cursor + 1) % len(rows)
	case tea.KeyLeft:
		g.pause.page = max(0, g.pause.page-1)
	case tea.KeyRight:
		g.pause.page = min(pages-1, g.pause.page+1)
	case tea.KeyEnter:
		return g.shareChoose(rows[g.pause.cursor])
	}

	return g, nil
}

func (g Game) shareChoose(row int) (tea.Model, tea.Cmd) {
	sharing := g.pauser.Share

	switch row {
	case shareAgree:
		if err := sharing.SetConsent(true); err != nil {
			g.pause.note = "Nie udało się zapisać zgody: " + err.Error()

			return g, nil
		}

		g.pause.cursor, g.pause.note = 0, "Dziękujemy. Nic nie zostało jeszcze wysłane."
	case shareDecline, shareBack:
		g.pause.screen, g.pause.cursor, g.pause.note = pauseMain, 0, ""
	case sharePreview:
		g.pause.preview, g.pause.page = !g.pause.preview, 0
	case shareSend:
		if g.pause.sharing {
			g.pause.note = "Wysyłka już trwa w tle."

			return g, nil
		}

		payload, upTo := g.pause.outbox, g.pause.outboxUpTo
		if len(payload.Turns) == 0 {
			g.pause.note = "Nie ma nowych rozmów do wysłania."

			return g, nil
		}

		g.pause.sharing, g.pause.note = true, "Wysyłam w tle, możesz grać dalej."

		return g, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), share.Timeout)
			defer cancel()

			return sharedMsg{err: sharing.Send(ctx, payload), upTo: upTo, payload: len(payload.Turns)}
		}
	case shareWithdraw, shareRetryForget:
		if g.pause.sharing {
			g.pause.note = "Poczekaj, aż skończy się wysyłka."

			return g, nil
		}

		// Consent ends at once; the deletion on the server follows.
		if row == shareWithdraw {
			if err := sharing.SetConsent(false); err != nil {
				g.pause.note = "Nie udało się wycofać zgody: " + err.Error()

				return g, nil
			}
		}

		g.pause.sharing, g.pause.cursor, g.pause.note = true, 0, "Zgoda wycofana. Proszę serwer o usunięcie twoich rozmów…"

		return g, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), share.Timeout)
			defer cancel()

			return sharedMsg{err: sharing.Forget(ctx), forgot: true}
		}
	}

	return g, nil
}

// shared takes the outcome of a background send: a line in the pause menu
// if it is open, and in the log either way.
func (g Game) shared(msg sharedMsg) (tea.Model, tea.Cmd) {
	g.pause.sharing = false

	var text string

	switch {
	case msg.forgot && msg.err == nil:
		text = "Zgoda wycofana. Serwer usuwa twoje rozmowy i nie przyjmie już niczego z tego klucza."
		g.pause.retry, g.pause.outbox = false, share.Payload{}
	case msg.forgot:
		text = "Nie udało się poprosić serwera o usunięcie rozmów (" + share.Message(msg.err) +
			") Zgoda jest wycofana; usuwanie możesz ponowić w tym menu."
	case msg.err == nil:
		g.pause.sharedUpTo = max(g.pause.sharedUpTo, msg.upTo)
		g.pause.retry = false
		text = fmt.Sprintf("Dziękujemy, wysłano %d kwestii.", msg.payload)
	case errors.Is(msg.err, share.ErrRejected):
		// A payload the server turned down is not sent again as it is.
		g.pause.retry = false
		text = share.Message(msg.err)
	default:
		// No answer: the same payload goes again next time, as the server
		// may already have it.
		g.pause.retry = true
		text = share.Message(msg.err)
	}

	if !g.pause.retry {
		g.pause.outbox, g.pause.outboxUpTo = g.sharePayload()
	}

	if g.pause.open && g.pause.screen == pauseShare {
		g.pause.note = text
	}

	g.realm.Tell(g.name, text)
	g.refresh()

	return g, nil
}

// previewLines are the screen lines of the whole outbox, every text in
// full, for the preview to page through.
func (g Game) previewLines() []string {
	var lines []string

	for _, turn := range g.pause.outbox.Turns {
		who := "Ty"
		if turn.Who != share.WhoPlayer {
			who = turn.Name
			if turn.Model != "" {
				who += " (" + turn.Model + ")"
			}
		}

		lines = append(lines, strings.Split(wrap(labelStyle.Render(who+": ")+turn.Text, 56), "\n")...)
	}

	return lines
}

// shareView draws the share screen.
func (g Game) shareView() []string {
	lines := []string{titleStyle.Render("Udostępnij rozmowy"), ""}
	sharing := g.pauser.Share

	if !sharing.Consented() {
		lines = append(lines, wrap(consentText, 56), "")
	}

	payload := g.pause.outbox

	for index, row := range g.shareRows() {
		var label string

		switch row {
		case shareAgree:
			label = goodStyle.Render("Zgadzam się")
		case shareDecline:
			label = "Nie teraz"
		case shareSend:
			label = goodStyle.Render(fmt.Sprintf("Wyślij nowe rozmowy z tej gry (%d kwestii)", len(payload.Turns)))
		case sharePreview:
			label = "Podejrzyj, co zostanie wysłane"
			if g.pause.preview {
				label = "Schowaj podgląd"
			}
		case shareWithdraw:
			label = "Wycofaj zgodę i usuń moje rozmowy z serwera"
		case shareRetryForget:
			label = goodStyle.Render("Ponów usuwanie moich rozmów z serwera")
		case shareBack:
			label = "Wróć"
		}

		lines = append(lines, pointer(index == g.pause.cursor)+label)
	}

	if g.pause.preview && sharing.Consented() {
		lines = append(lines, "", labelStyle.Render(fmt.Sprintf("Świat: %s · gra: %s", payload.World, payload.Game)))

		preview := g.previewLines()
		pages := max(1, (len(preview)+previewPage-1)/previewPage)
		page := min(g.pause.page, pages-1)
		lines = append(lines, preview[min(len(preview), page*previewPage):min(len(preview), (page+1)*previewPage)]...)

		if len(payload.Turns) == 0 {
			lines = append(lines, dimStyle.Render("Nie ma nowych rozmów."))
		} else {
			lines = append(lines, dimStyle.Render(fmt.Sprintf("strona %d z %d · ←/→: przewiń", page+1, pages)))
		}
	}

	return append(lines, "", dimStyle.Render("Enter: wybierz · Esc: wróć"))
}
