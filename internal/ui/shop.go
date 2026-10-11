package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	portraitWidth  = 40
	portraitHeight = 20
	sideWidth      = portraitWidth + 2

	talkTimeout = 90 * time.Second
)

type speaker int

const (
	speakerSystem speaker = iota
	speakerPlayer
	speakerMerchant
)

type chatLine struct {
	speaker speaker
	text    string
}

// intentMsg carries the model's reading of what the player told the merchant.
type intentMsg struct {
	said   string
	intent game.Intent
	err    error
}

// replyMsg carries the merchant's reply to the last decision. A problem
// means the model could not word it and a canned line stands in. Idle talk
// may have offered something new, worth a look.
type replyMsg struct {
	request game.NarrateRequest
	said    string
	text    string
	problem error
	look    bool
}

// pieceMsg carries a piece of the reply still being written.
type pieceMsg struct {
	piece string
}

// notedMsg carries a new fact the model spotted in a keeper's line.
type notedMsg struct {
	fact  game.FactSpec
	found bool
	// line is the keeper's line the fact was noted from.
	line string
}

// inventedMsg carries what a keeper's line turned out to offer.
type inventedMsg struct {
	item    world.ItemSpec
	offered bool
	// line is the keeper's line that offered the item.
	line string
}

// shop is the conversation screen with a shopkeeper.
type shop struct {
	serial   int
	merchant string
	greeting string
	portrait string
	chat     viewport.Model
	input    textinput.Model
	lines    []chatLine
	waiting  bool
	// partial is the reply so far, while it is being written.
	partial string
	stream  chan tea.Msg
	width   int
	height  int
}

func newShop() shop {
	input := textinput.New()
	input.Placeholder = "Napisz coś do kupca… (ile za X · dam 30 · sprzedam X za 20 · żegnaj)"
	input.Prompt = "> "
	input.CharLimit = 300
	input.Focus()

	return shop{chat: viewport.New(0, 0), input: input}
}

// open resets the screen for a new conversation with the given keeper.
func (s *shop) open(view game.ShopView, portrait string) {
	s.serial = view.Serial
	s.merchant = view.Keeper
	s.waiting = false
	s.input.SetValue("")
	s.portrait = portrait

	s.greeting = view.Greeting
	advice := "Pisz zwykłym tekstem: pytaj o towar, podawaj swoją cenę, targuj się. Kilka sztuk naraz wychodzi taniej, np. «kupię 5 naparów za 60». Enter wysyła, PgUp/PgDn przewija, Esc wychodzi z budynku."

	if view.Avatar != "" {
		advice = "Pisz zwykłym tekstem: pytaj o warunki, podawaj swoją cenę, targuj się; «zgoda» przypieczętowuje umowę. Enter wysyła, PgUp/PgDn przewija, Esc kończy rozmowę."
	}

	s.lines = []chatLine{
		{speakerSystem, advice},
		{speakerMerchant, view.Greeting},
	}
	s.resize(s.width, s.height)
}

// resize fits the chat and the input to the terminal.
func (s *shop) resize(width, height int) {
	s.width, s.height = width, height
	s.chat.Width = max(1, width-sideWidth-4)
	s.chat.Height = max(1, height-inputHeight-2)
	s.input.Width = max(1, width-sideWidth-8)
	s.render()
}

func (s *shop) say(who speaker, text string) {
	s.lines = append(s.lines, chatLine{who, text})
	s.render()
}

func (s *shop) render() {
	wrap := lipgloss.NewStyle().Width(s.chat.Width)
	blocks := make([]string, 0, len(s.lines)+1)

	for _, line := range s.lines {
		switch line.speaker {
		case speakerPlayer:
			blocks = append(blocks, wrap.Render(playerStyle.Render("Ty: ")+line.text))
		case speakerMerchant:
			blocks = append(blocks, wrap.Render(titleStyle.Render(s.merchant+": ")+line.text))
		default:
			blocks = append(blocks, wrap.Render(dimStyle.Render(line.text)))
		}
	}

	if s.partial != "" {
		blocks = append(blocks, wrap.Render(titleStyle.Render(s.merchant+": ")+strings.Join(strings.Fields(s.partial), " ")+"▌"))
	}

	s.chat.SetContent(strings.Join(blocks, "\n\n"))
	s.chat.GotoBottom()
}

func (g Game) shopKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		g.realm.CloseShop(g.name)
		g.refresh()

		return g, nil
	case tea.KeyPgUp, tea.KeyPgDown, tea.KeyUp, tea.KeyDown:
		var cmd tea.Cmd
		g.shop.chat, cmd = g.shop.chat.Update(msg)

		return g, cmd
	case tea.KeyEnter:
		text := strings.TrimSpace(g.shop.input.Value())
		if text == "" || g.shop.waiting {
			return g, nil
		}

		request, ok := g.realm.HaggleRequest(g.name, text)
		if !ok {
			return g, nil
		}

		g.shop.input.SetValue("")
		g.shop.waiting = true
		g.shop.say(speakerPlayer, text)

		return g, interpret(g.brain, request)
	}

	var cmd tea.Cmd
	g.shop.input, cmd = g.shop.input.Update(msg)

	return g, cmd
}

// haggle applies the model's reading of the player's words to the rules and
// asks for the merchant's reply to the outcome.
func (g Game) haggle(msg intentMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		g.shop.waiting = false
		g.shop.say(speakerSystem, "Kupiec cię nie dosłyszał: "+msg.err.Error())

		return g, nil
	}

	decision, request, ok := g.realm.Haggle(g.name, msg.said, msg.intent)
	if !ok {
		g.shop.waiting = false

		return g, nil
	}

	if note := ledger(decision); note != "" {
		g.shop.say(speakerSystem, note)
	}

	g.refresh()

	// Only idle talk invents things; a deal is about what is on the shelf.
	idle := decision.Outcome == game.OutcomeChat || decision.Outcome == game.OutcomeWares || decision.Outcome == game.OutcomeUnclear
	g.shop.partial, g.shop.stream = "", make(chan tea.Msg, 64)

	return g, narrate(g.brain, request, idle, g.shop.stream)
}

// interpret asks the brain to read the player's message. The request is a
// snapshot, so the realm is never touched from the command's goroutine.
func interpret(brain game.Brain, request game.InterpretRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), talkTimeout)
		defer cancel()

		intent, err := brain.Interpret(ctx, request)

		return intentMsg{said: request.Text, intent: request.Sanitize(intent), err: err}
	}
}

// narrate starts writing the reply in the background and returns the
// command that waits for its first piece; the pieces and finally the reply
// arrive through the stream channel.
func narrate(brain game.Brain, request game.NarrateRequest, look bool, stream chan tea.Msg) tea.Cmd {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), talkTimeout)
		defer cancel()

		reply, problem := game.NarrateLive(ctx, brain, request, func(piece string) {
			stream <- pieceMsg{piece: piece}
		})

		stream <- replyMsg{request: request, said: request.Text, text: reply, problem: problem, look: look && problem == nil}
	}()

	return next(stream)
}

// next waits for the next message of a reply being written.
func next(stream chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-stream
	}
}

// invent asks the brain, when it can, whether a keeper's line offered
// something that does not exist yet.
// note asks the model whether a line brought a new fact into the world.
func note(brain game.Brain, reply string, known []string) tea.Cmd {
	noter, ok := brain.(game.Noter)
	if !ok {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), talkTimeout)
		defer cancel()

		fact, found, err := noter.Note(ctx, game.NoteRequest{Reply: reply, Known: known})

		return notedMsg{fact: fact, found: found && err == nil, line: reply}
	}
}

func invent(brain game.Brain, reply string, wares []game.Ware) tea.Cmd {
	inventor, ok := brain.(game.Inventor)
	if !ok {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), talkTimeout)
		defer cancel()

		item, offered, err := inventor.Invent(ctx, game.InventRequest{Reply: reply, Wares: wares})

		return inventedMsg{item: item, offered: offered && err == nil, line: reply}
	}
}

// ledger words what the rules decided. Unlike the merchant's line it never
// comes from a model, so it is the authoritative record of a transaction.
func ledger(decision game.Decision) string {
	goods := decision.Item.Name
	if decision.Quantity > 1 {
		goods = fmt.Sprintf("%d × %s", decision.Quantity, decision.Item.Name)
	}

	switch decision.Outcome {
	case game.OutcomeSold:
		return fmt.Sprintf("Kupiono: %s za %d %s.", goods, decision.Price, decision.Currency)
	case game.OutcomeCounter:
		return fmt.Sprintf("Twoja oferta: %d. Cena kupca za %s: %d.", decision.Offer, goods, decision.Price)
	case game.OutcomeInsulted:
		return fmt.Sprintf("Twoja oferta: %d. Kupiec jest urażony, cena bez zmian: %d.", decision.Offer, decision.Price)
	case game.OutcomeNoGold:
		return fmt.Sprintf("Nie stać cię. Masz %d %s.", decision.Gold, decision.Currency)
	case game.OutcomeShort:
		return fmt.Sprintf("%s: w sklepie jest tylko jedna sztuka.", decision.Item.Name)
	case game.OutcomeInfo:
		if decision.Quantity > 1 {
			return fmt.Sprintf("Cena kupca za %s: %d (pojedynczo %d za sztukę).", goods, decision.Price, decision.Item.Price)
		}
	case game.OutcomeBought:
		return fmt.Sprintf("Sprzedano: %s za %d %s.", goods, decision.Price, decision.Currency)
	case game.OutcomeBid:
		if decision.Offer > 0 {
			return fmt.Sprintf("Żądasz %d. Kupiec daje za %s: %d.", decision.Offer, goods, decision.Price)
		}

		return fmt.Sprintf("Kupiec daje za %s: %d %s. «Zgoda» sprzedaje, albo podaj swoją cenę.", goods, decision.Price, decision.Currency)
	case game.OutcomeUnwanted:
		return "Kupiec nie skupuje takich rzeczy."
	case game.OutcomeWalkedAway:
		return "Rozmowa skończona: cierpliwość się wyczerpała. Wróć, gdy emocje opadną: odejdź z osady i przyjdź ponownie."
	}

	return ""
}

func (g Game) shopView() string {
	s, view := g.shop, *g.view.Shop

	chat := box.Width(s.chat.Width + 2).Height(s.chat.Height).Render(s.chat.View())
	input := box.Width(s.chat.Width + 2).Render(g.shopInput())
	portrait := border.Width(portraitWidth).Height(portraitHeight).MaxHeight(portraitHeight + 2).Render(s.portrait)
	status := panel(sideWidth, g.height-portraitHeight-2, g.shopStatus(view), false)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, portrait, status),
		lipgloss.JoinVertical(lipgloss.Left, chat, input),
	)
}

func (g Game) shopInput() string {
	switch {
	case g.view.Shop.Over:
		return dimStyle.Render("Rozmowa zakończona. Esc wychodzi z budynku.")
	case g.hearing != "":
		return badStyle.Render(g.hearing)
	case g.shop.waiting:
		return dimStyle.Render(g.shop.merchant + " się namyśla…")
	}

	return g.shop.input.View()
}

func (g Game) shopStatus(view game.ShopView) string {
	var status strings.Builder

	fmt.Fprintf(&status, "%s\n%s\n", titleStyle.Render(view.Building), view.Keeper)
	fmt.Fprintf(&status, "%s %s\n", labelStyle.Render("Cierpliwość:"), strings.Repeat("●", view.Patience)+strings.Repeat("○", max(0, view.MaxPatience-view.Patience)))
	fmt.Fprintf(&status, "%s %s\n\n", labelStyle.Render("Twoja sakwa:"), goldStyle.Render(fmt.Sprintf("%d %s", g.view.Me.Gold, g.view.World.Currency)))

	var focused *game.Ware

	for index, ware := range view.Wares {
		name := fmt.Sprintf("%-*s", portraitWidth-10, ware.Name)
		if ware.ID == view.Focus {
			name = titleStyle.Render(name)
			focused = &view.Wares[index]
		}

		fmt.Fprintf(&status, "%s %s %4d\n", glyph(ware.Kind, ware.Effect), name, ware.Price)
	}

	// The ware under discussion is shown, when the panel has the room.
	if room := g.height - portraitHeight - 4 - 5 - len(view.Wares); focused != nil && room >= iconRows+1 {
		status.WriteString("\n" + withIcon(icon(focused.Kind, focused.Effect, focused.Tier, focused.Name), focused.Description, portraitWidth-2))
	}

	return status.String()
}
