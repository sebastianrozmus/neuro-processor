package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
)

// worldMsg tells the interface that the shared world changed.
type worldMsg struct{}

// Save stores the game and returns where it was written.
type Save func() (string, error)

// Voice reads a line aloud in a mood.
type Voice interface {
	Say(text string, mood speech.Emotion)
}

// namedVoice is a voice that can speak as a given character.
type namedVoice interface {
	SayAs(name, kind, text string, mood speech.Emotion)
}

// Ear records the player's voice and turns it into text.
type Ear interface {
	Start() error
	Stop() (string, error)
	Recording() bool
}

// heardMsg carries what the player said into the microphone.
type heardMsg struct {
	text string
	err  error
}

// Record keeps a finished exchange with a keeper: what the model was asked
// and what it answered. It is how the game collects its own training data.
type Record func(request game.NarrateRequest, reply string)

// Game is one player's interface onto a realm.
type Game struct {
	realm   *game.Realm
	name    string
	brain   game.Brain
	library *avatars.Library
	save    Save
	voice   Voice
	// debug shows on the screen why the rules refused a model's line.
	debug  bool
	ear    Ear
	record Record
	// hearing is what the help line says about the microphone.
	hearing string

	view   game.View
	input  textinput.Model
	last   string
	bag    bool
	cursor int
	// book opens the bestiary; beast is the page it is on.
	book  bool
	beast int
	// effects are the blows being shown, frame the frame they are at and
	// shown the serial of the last blow taken for showing.
	effects []game.BlowView
	frame   int
	shown   int
	// card is whose card the left panel shows: 0 the hero, then companions.
	card int
	// wanted is the monster the player pointed at; target is the one
	// actually picked, which follows the monsters still standing.
	wanted int
	target int
	// step, action and choice are the place in the fight menu.
	step   int
	action int
	choice int
	shop   shop
	// inline shows painted pictures as real images; only the local
	// player's terminal can, guests over SSH keep block art.
	inline bool
	width  int
	height int
}

// NewGame builds the interface of the named player, who must have joined
// the realm already. The brain serves their talks with shopkeepers. Save
// may be nil for players who are guests in somebody else's world, and voice
// may be nil for a silent game, as may ear for a game without a
// microphone. Record, when set, is given every exchange the model answered
// well.
func NewGame(realm *game.Realm, name string, brain game.Brain, library *avatars.Library, save Save, voice Voice, ear Ear, record Record) Game {
	input := textinput.New()
	input.Placeholder = "wpisz polecenie, np. idź na północ (pomoc: ?)"
	input.Prompt = "> "
	input.CharLimit = 120
	input.Focus()

	scene := Game{realm: realm, name: name, brain: brain, library: library, save: save, voice: voice, ear: ear, record: record, input: input, shop: newShop(), shown: -1}
	scene.refresh()

	// Blows from before the screen opened are never replayed; everything
	// from now on is.
	if scene.shown < 0 {
		scene.shown = 0
	}

	return scene
}

// Inline turns showing painted pictures as real images on or off for
// this screen.
func (g Game) Inline(on bool) Game {
	g.inline = on

	return g
}

// Debug makes the screen say why a model's line was refused and a canned
// one spoken instead; the inspector keeps the same.
func (g Game) Debug(on bool) Game {
	g.debug = on

	return g
}

// Init starts listening for changes of the world.
func (g Game) Init() tea.Cmd {
	return tea.Batch(g.listen(), textinput.Blink)
}

// listen waits for the next change of the world.
func (g Game) listen() tea.Cmd {
	updates := g.realm.Updates(g.name)

	return func() tea.Msg {
		<-updates

		return worldMsg{}
	}
}

// Update handles input, world changes and resizes, and keeps the blows
// animated while there are any to show.
func (g Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if frame, ok := msg.(frameMsg); ok {
		if g.advance(frame) {
			return g, g.nextFrame()
		}

		return g, nil
	}

	running := g.animating()
	model, cmd := g.update(msg)

	if next, ok := model.(Game); ok && next.animating() && (!running || next.frame == 0 && next.shown != g.shown) {
		cmd = tea.Batch(cmd, next.nextFrame())
	}

	return model, cmd
}

func (g Game) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width, g.height = msg.Width, msg.Height
		g.input.Width = max(1, g.width-8)
		g.shop.resize(g.width, g.height)

		return g, nil
	case worldMsg:
		g.refresh()

		return g, g.listen()
	case intentMsg:
		return g.haggle(msg)
	case pieceMsg:
		g.shop.partial += msg.piece
		g.shop.render()

		return g, next(g.shop.stream)
	case heardMsg:
		g.hearing = ""

		if msg.err != nil {
			g.realm.Tell(g.name, "Nie udało się rozpoznać mowy: "+msg.err.Error())
		} else if msg.text != "" {
			g.dictate(msg.text)
		}

		g.refresh()

		return g, nil
	case replyMsg:
		mood, line := speech.SplitMood(msg.text)
		g.realm.HaggleRemember(g.name, msg.said, line)
		g.shop.waiting, g.shop.partial = false, ""
		g.shop.say(speakerMerchant, withMood(line, mood))
		g.speak(msg.text)

		if g.record != nil && msg.problem == nil {
			g.record(msg.request, msg.text)
		}

		// What the rules refused goes to the inspector, and to the screen
		// when it is debugging.
		var rejection *game.Rejection
		errors.As(msg.problem, &rejection)
		g.realm.Exchange(g.name, msg.request, msg.text, rejection)

		if rejection != nil {
			if g.debug {
				for index, attempt := range rejection.Attempts {
					verdict := "Odrzucone"
					if rejection.Kept && index == 0 {
						verdict = "Przyjęte mimo wszystko"
					}

					g.shop.say(speakerSystem, fmt.Sprintf("%s (%s): „%s”", verdict, attempt.Reason, attempt.Text))
				}
			}
		}

		// A model that keeps contradicting the rules is not worth a note,
		// but one that cannot be reached is: the player may fix that.
		if msg.problem != nil && !errors.Is(msg.problem, game.ErrInconsistent) {
			g.shop.say(speakerSystem, "Model nie odpowiedział, więc padła gotowa kwestia: "+msg.problem.Error())
		}

		g.refresh()

		// A line that sent the player elsewhere is a lead, not news: the
		// world learns nothing from it, and nothing in it is for sale.
		if msg.request.Lead != "" && msg.problem == nil {
			g.shop.say(speakerSystem, msg.request.Lead+" (dziennik)")

			return g, nil
		}

		if msg.look && g.view.Shop != nil {
			commands := []tea.Cmd{invent(g.brain, msg.text, g.view.Shop.Wares)}
			if game.Noteworthy(msg.text) {
				commands = append(commands, note(g.brain, msg.text, g.realm.Subjects()))
			}

			return g, tea.Batch(commands...)
		}

		return g, nil
	case inventedMsg:
		if item, ok := g.realm.Invent(g.name, msg.item, msg.line); msg.offered && ok {
			g.shop.say(speakerSystem, fmt.Sprintf("Do świata wpada: %s (%d %s). %s", item.Name, item.Price(), g.view.World.Currency, game.Effect(item)))
			g.refresh()
		}

		return g, nil
	case notedMsg:
		if msg.found && g.realm.Learn(g.name, msg.fact, msg.line) {
			g.shop.say(speakerSystem, "Zapamiętujesz: "+msg.fact.Subject+". (dziennik)")
			g.refresh()
		}

		return g, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return g, tea.Quit
		}

		if msg.Type == tea.KeyF5 {
			return g.pushToTalk()
		}

		switch {
		case g.view.Shop != nil:
			return g.shopKey(msg)
		case g.book:
			g.bookKey(msg)
			g.refresh()

			return g, nil
		case msg.Type == tea.KeyF4 && len(g.view.Me.Bestiary) > 0:
			g.book, g.bag = true, false

			return g, nil
		case g.bag:
			g.bagKey(msg)
			g.refresh()

			return g, nil
		}

		return g.mapKey(msg)
	}

	var cmd tea.Cmd
	g.input, cmd = g.input.Update(msg)

	return g, cmd
}

// refresh takes a new snapshot of the world and keeps the screen valid.
func (g *Game) refresh() {
	fighting := g.view.Fight != nil
	heard := len(g.view.Log)
	g.view = g.realm.View(g.name)
	g.collect()

	// New lines of the game master are read aloud as they come in.
	for _, line := range g.view.Log[min(heard, len(g.view.Log)):] {
		if strings.HasPrefix(line, game.MasterMark) {
			g.narrate(strings.TrimPrefix(line, game.MasterMark))
		}
	}
	bag := g.view.Me.Bag
	if g.card > 0 && g.card <= len(g.view.Me.Party) {
		bag = g.view.Me.Party[g.card-1].Bag
	}

	g.cursor = min(g.cursor, max(0, len(bag)-1))
	g.target = 0

	g.card = min(g.card, len(g.view.Me.Party))

	if g.view.Fight != nil {
		if !fighting {
			g.enterFight()
		}

		g.target = min(g.wanted, len(g.view.Fight.Monsters)-1)
		g.action = min(g.action, len(g.actions())-1)
		g.choice = min(g.choice, max(0, len(g.choices())-1))

		if g.step == stepChoice && len(g.choices()) == 0 {
			g.step = stepAction
		}

		// The card follows whoever is deciding.
		if actor := g.view.Fight.Actor; actor != nil {
			g.card = actor.Companion + 1
		}
	}

	if g.view.Shop != nil && g.view.Shop.Serial != g.shop.serial {
		portrait := keeperPortrait(g.view.World.Spec, g.view.Shop.Kind)
		if g.view.Shop.Avatar != "" && g.library != nil {
			portrait = g.library.Art(g.view.Shop.Avatar)
		}

		if g.inline {
			if picture := g.picture(g.view.Shop.Avatar, portraitWidth, portraitHeight); picture != "" {
				portrait = picture
			} else if picture := keeperPicture(g.view.World.Spec, g.view.Shop.Kind); picture != "" && g.view.Shop.Avatar == "" {
				portrait = picture
			}
		}

		g.shop.open(*g.view.Shop, portrait)
		g.speak(g.view.Shop.Greeting)
	}
}

// narrate reads a game master's line aloud, as the narrator.
func (g Game) narrate(text string) {
	if g.voice == nil {
		return
	}

	mood, line := speech.SplitMood(text)
	if named, ok := g.voice.(namedVoice); ok {
		named.SayAs("", "narrator", line, mood)

		return
	}

	g.voice.Say(line, mood)
}

// speak reads a character's line aloud in its mood, if the game has a
// voice: as the keeper being talked to, or as the narrator.
func (g Game) speak(text string) {
	if g.voice == nil {
		return
	}

	mood, line := speech.SplitMood(text)
	name, kind := "", "narrator"
	if g.view.Shop != nil {
		name, kind = g.view.Shop.Keeper, string(g.view.Shop.Kind)
		if kind == "" {
			kind = "towarzysz"
		}
	}

	if named, ok := g.voice.(namedVoice); ok {
		named.SayAs(name, kind, line, mood)

		return
	}

	g.voice.Say(line, mood)
}

// withMood shows the mood a line was said in, when it was not a calm one.
func withMood(line string, mood speech.Emotion) string {
	if mood == speech.Calm {
		return line
	}

	return dimStyle.Render("("+string(mood)+") ") + line
}

// pushToTalk turns the microphone on, or off and then into text that goes
// to the command line.
func (g Game) pushToTalk() (tea.Model, tea.Cmd) {
	switch {
	case g.ear == nil:
		g.realm.Tell(g.name, "Rozpoznawanie mowy nie jest skonfigurowane: uruchom tools/avatar/setup.sh (opis w README).")
		g.refresh()

		return g, nil
	case !g.ear.Recording():
		if err := g.ear.Start(); err != nil {
			g.realm.Tell(g.name, "Mikrofon nie działa: "+err.Error())
			g.refresh()

			return g, nil
		}

		g.hearing = "● Nagrywam, mów. F5 kończy."

		return g, nil
	}

	g.hearing = "Rozpoznaję…"
	ear := g.ear

	return g, func() tea.Msg {
		text, err := ear.Stop()

		return heardMsg{text: text, err: err}
	}
}

// dictate puts recognised speech into whichever line the player is typing.
func (g *Game) dictate(text string) {
	field := &g.input
	if g.view.Shop != nil {
		field = &g.shop.input
	}

	current := strings.TrimSpace(field.Value())
	if current != "" {
		text = current + " " + text
	}

	field.SetValue(text)
	field.CursorEnd()
}

func (g Game) mapKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	typing := g.input.Value() != ""

	// While nothing is being typed, a fight is steered through its menu.
	if g.view.Fight != nil && !typing && msg.Type != tea.KeyTab && msg.Type != tea.KeyF1 && msg.Type != tea.KeyF2 && msg.Type != tea.KeyF3 && g.fightKey(msg) {
		g.refresh()

		return g, nil
	}

	switch msg.Type {
	case tea.KeyF1, tea.KeyF2, tea.KeyF3:
		// Function keys count downwards in this library, so the distance is
		// taken the other way round.
		g.card = min(int(tea.KeyF1-msg.Type), len(g.view.Me.Party))
	case tea.KeyEsc:
		g.input.SetValue("")
	case tea.KeyTab:
		g.bag = true
	case tea.KeyUp:
		g.realm.Move(g.name, 0, -1)
	case tea.KeyDown:
		g.realm.Move(g.name, 0, 1)
	case tea.KeyLeft, tea.KeyRight:
		if typing {
			var cmd tea.Cmd
			g.input, cmd = g.input.Update(msg)

			return g, cmd
		}

		step := 1
		if msg.Type == tea.KeyLeft {
			step = -1
		}

		g.realm.Move(g.name, step, 0)
	case tea.KeyEnter:
		return g.enter()
	default:
		var cmd tea.Cmd
		g.input, cmd = g.input.Update(msg)

		return g, cmd
	}

	g.refresh()

	return g, nil
}

// enter runs the typed command. An empty line repeats the last command.
func (g Game) enter() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(g.input.Value())
	g.input.SetValue("")

	if text == "" {
		text = g.last
	}

	g.last = text

	if text == "" {
		return g, nil
	}

	g.realm.Tell(g.name, "> "+text)

	// A line that starts with a companion's name is an order for them.
	if g.ordered(text) {
		g.refresh()

		return g, nil
	}

	command := game.Parse(text)
	command.Index = g.target

	switch command.Verb {
	case game.VerbQuit:
		return g, tea.Quit
	case game.VerbBag:
		g.bag = true
	case game.VerbBestiary:
		g.book = true
	case game.VerbJournal:
		for _, line := range game.JournalLines(g.realm.Journal(g.name)) {
			g.realm.Tell(g.name, line)
		}

		if jobs := g.realm.Quests(g.name); len(jobs) > 0 {
			for _, line := range game.QuestLines(jobs) {
				g.realm.Tell(g.name, line)
			}
		}
	case game.VerbHelp:
		switch {
		case g.view.Investigation():
			g.realm.Tell(g.name, game.InvestigationHelp)
		case g.view.Rooms():
			g.realm.Tell(g.name, game.AdventureHelp)
		default:
			g.realm.Tell(g.name, game.Help)
		}
	case game.VerbSave:
		g.realm.Tell(g.name, g.saved())
	case game.VerbShow:
		g.show(command.Arg)
	default:
		g.realm.Execute(g.name, command)
	}

	g.refresh()

	return g, nil
}

// ordered passes a line to the companion it is addressed to, reporting
// whether it was one.
func (g Game) ordered(text string) bool {
	first, _, _ := strings.Cut(strings.TrimSpace(text), " ")

	for _, ally := range g.view.Me.Party {
		if game.Matches(ally.Name, first) {
			return g.realm.Order(g.name, text)
		}
	}

	return false
}

// show turns the left panel to the named companion, or back to the hero.
func (g *Game) show(who string) {
	for index, ally := range g.view.Me.Party {
		if game.Matches(ally.Name, who) {
			g.card = index + 1
			g.realm.Tell(g.name, "Karta: "+ally.Name+".")

			return
		}
	}

	g.card = 0
	g.realm.Tell(g.name, "Karta: "+g.view.Me.Name+". Towarzysze: "+strings.Join(partyNames(g.view.Me.Party), ", ")+".")
}

func partyNames(party []game.CompanionView) []string {
	names := make([]string, 0, len(party))
	for _, ally := range party {
		names = append(names, ally.Name)
	}

	return names
}

// saved saves the game and words the result.
func (g Game) saved() string {
	if g.save == nil {
		return "Grę może zapisać tylko gospodarz świata."
	}

	path, err := g.save()
	if err != nil {
		return "Nie udało się zapisać gry: " + err.Error()
	}

	return "Gra zapisana: " + path
}

func (g *Game) bagKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyF1, tea.KeyF2, tea.KeyF3:
		g.card, g.cursor = min(int(tea.KeyF1-msg.Type), len(g.view.Me.Party)), 0
	case tea.KeyUp:
		g.cursor = max(0, g.cursor-1)
	case tea.KeyDown:
		g.cursor++
	case tea.KeyEnter:
		if g.card > 0 && g.view.Fight == nil {
			g.realm.Tell(g.name, "Rzeczy towarzysza przekładasz poleceniami «daj X Y» i «odbierz X od Y».")

			return
		}

		if g.view.Rooms() {
			if g.cursor < len(g.view.Me.Bag) {
				g.realm.Execute(g.name, game.Parse("zbadaj "+g.view.Me.Bag[g.cursor].Name))
			}

			g.bag = false

			return
		}

		g.realm.Use(g.name, g.cursor)

		if g.view.Fight != nil {
			g.bag = false
		}
	case tea.KeyEsc, tea.KeyTab:
		g.bag = false
	}
}

// Waiting reports that a shop talk is waiting for the model's reply.
func (g Game) Waiting() bool {
	return g.shop.waiting
}

// Frames renders the blows being shown frame by frame, for recordings;
// without a running animation it is the screen as it is.
func (g Game) Frames() []string {
	if !g.animating() {
		return []string{g.View()}
	}

	frames := make([]string, 0, effectFrames)
	for frame := range effectFrames {
		g.frame = frame
		frames = append(frames, g.View())
	}

	return frames
}

// Name is the name the screen's player plays under.
func (g Game) Name() string {
	return g.name
}
