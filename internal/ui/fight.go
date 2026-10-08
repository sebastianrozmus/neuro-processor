package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The fight menu is walked left to right: whom to act on, what to do and,
// for powers and items, which one. It always decides for the unit whose
// turn it is: the hero or one of the companions.
const (
	stepTarget = iota
	stepAction
	stepChoice
	// stepMove is walking across the board of a D&D fight with the arrows.
	stepMove
)

const (
	actAttack = iota
	actPower
	actItem
	actFlee
	actAuto
	actMove
)

const (
	actionWidth  = 13
	targetShare  = 42
	fightBar     = 6
	choiceWindow = 5
)

var actionNames = []string{"Atak", "Moc", "Przedmiot", "Ucieczka", "Auto", "Ruch"}

// choice is one entry of the last column of the fight menu.
type choice struct {
	label string
	note  string
	// index is the position of the power among known ones, or of the item
	// in the bag of the unit whose turn it is.
	index  int
	usable bool
}

// menu lists the actions the unit whose turn it is may take, in the order
// shown: only companions can be left to act on their own, and only a D&D
// fight has movement.
func (g Game) menu() []int {
	var ids []int
	if g.view.Fight.Board != nil {
		ids = append(ids, actMove)
	}

	ids = append(ids, actAttack, actPower, actItem, actFlee)

	if actor := g.view.Fight.Actor; actor != nil && actor.Companion >= 0 {
		ids = append(ids, actAuto)
	}

	return ids
}

// actions names the entries of the menu.
func (g Game) actions() []string {
	ids := g.menu()
	names := make([]string, 0, len(ids))

	for _, id := range ids {
		names = append(names, actionNames[id])
	}

	return names
}

// act is the action the cursor rests on.
func (g Game) act() int {
	ids := g.menu()

	return ids[min(max(g.action, 0), len(ids)-1)]
}

// choices lists what the last column offers for the selected action.
func (g Game) choices() []choice {
	actor := g.view.Fight.Actor
	if actor == nil {
		return nil
	}

	var list []choice

	switch g.act() {
	case actPower:
		for index, power := range actor.Powers {
			cost := power.Cost()
			if power.ID == "innate-heal" || power.ID == "innate-blast" {
				cost = 0
			}

			list = append(list, choice{
				label:  fmt.Sprintf("%s (%d)", power.Name, cost),
				note:   game.SpellEffect(power),
				index:  index,
				usable: actor.Mana >= cost,
			})
		}
	case actItem:
		for index, item := range actor.Bag {
			if item.Kind == world.KindPotion {
				list = append(list, choice{label: item.Name, note: game.Effect(item), index: index, usable: true})
			}
		}
	}

	return list
}

// enterFight puts the menu where a new fight starts: on the targets when
// there is a choice of them, on the actions otherwise.
func (g *Game) enterFight() {
	g.wanted, g.action, g.choice, g.step = 0, 0, 0, stepAction

	// The attack is the first action unless the board puts movement first.
	for index, id := range g.menu() {
		if id == actAttack {
			g.action = index
		}
	}

	if len(g.view.Fight.Monsters) > 1 {
		g.step = stepTarget
	}
}

// fightKey moves through the fight menu. It reports false for keys the
// menu has no use for.
func (g *Game) fightKey(msg tea.KeyMsg) bool {
	if g.view.Fight.Actor == nil {
		return msg.Type != tea.KeyRunes
	}

	options := g.choices()

	// Walking: the arrows move the piece; Enter or Esc end the walk.
	if g.step == stepMove {
		switch msg.Type {
		case tea.KeyUp:
			g.realm.Step(g.name, 0, -1)
		case tea.KeyDown:
			g.realm.Step(g.name, 0, 1)
		case tea.KeyLeft:
			g.realm.Step(g.name, -1, 0)
		case tea.KeyRight:
			g.realm.Step(g.name, 1, 0)
		case tea.KeyEnter, tea.KeyEsc:
			g.step = stepAction
		default:
			return false
		}

		return true
	}

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		step := 1
		if msg.Type == tea.KeyUp {
			step = -1
		}

		switch g.step {
		case stepTarget:
			g.wanted = min(max(g.target+step, 0), len(g.view.Fight.Monsters)-1)
		case stepAction:
			g.action = min(max(g.action+step, 0), len(g.actions())-1)
		default:
			g.choice = min(max(g.choice+step, 0), len(options)-1)
		}
	case tea.KeyLeft, tea.KeyEsc:
		g.step = max(g.step-1, stepTarget)
	case tea.KeyRight:
		if g.step == stepTarget || (g.step == stepAction && len(options) > 0) {
			g.step++
			g.choice = 0
		}
	case tea.KeyEnter:
		g.confirm(options)
	default:
		return false
	}

	return true
}

// confirm goes one step deeper into the menu, or acts when the selection
// is complete. After acting the menu rests on the actions, so that Enter
// repeats the last one.
func (g *Game) confirm(options []choice) {
	switch {
	case g.step == stepTarget:
		g.step = stepAction
	case g.step == stepAction && g.act() == actMove:
		g.step = stepMove
	case g.step == stepAction && g.act() == actAttack:
		g.realm.Attack(g.name, g.target)
	case g.step == stepAction && g.act() == actFlee:
		g.realm.Flee(g.name)
	case g.step == stepAction && g.act() == actAuto:
		g.realm.Auto(g.name)
	case g.step == stepAction && len(options) == 0:
		g.realm.Tell(g.name, map[int]string{
			actPower: g.view.Fight.Actor.Name + " nie zna żadnych mocy.",
			actItem:  g.view.Fight.Actor.Name + " nie ma przy sobie niczego, co pomogłoby w walce.",
		}[g.act()])
	case g.step == stepAction:
		g.step, g.choice = stepChoice, 0
	case g.act() == actPower:
		g.realm.Cast(g.name, options[g.choice].index, g.target)
		g.step = stepAction
	default:
		g.realm.Use(g.name, options[g.choice].index)
		g.step = stepAction
	}
}

// fightHeight is how many rows the fight panel needs.
func (g Game) fightHeight() int {
	rows := max(len(g.view.Fight.Monsters), len(g.actions()), min(len(g.choices()), choiceWindow))

	return rows + 4
}

// fightView draws the fight menu: the monsters, the actions and, for
// powers and items, the list to pick from, under the order of turns.
func (g Game) fightView(width int) string {
	fight := g.view.Fight
	options := g.choices()
	targetWidth := width * targetShare / 100
	choiceWidth := width - targetWidth - actionWidth

	title := foeStyle.Render(fmt.Sprintf("Walka, runda %d", fight.Round))

	switch {
	case fight.Actor != nil:
		title += "   " + selfStyle.Render("Tura: "+fight.Actor.Name)
	case fight.Waiting != "":
		title += "   " + labelStyle.Render("Tura: "+fight.Waiting)
	}

	if fight.Board != nil && fight.Actor != nil {
		title += "   " + labelStyle.Render(fmt.Sprintf("ruch: %d pól", fight.Board.Moves))
	}

	if g.view.Me.Poison > 0 {
		title += "   " + badStyle.Render(fmt.Sprintf("☠ jad: %d rund", g.view.Me.Poison))
	}

	order := make([]string, 0, len(fight.Order))
	for _, turn := range fight.Order {
		name := turn.Name
		if turn.Foe {
			name = foeStyle.Render(name)
		}

		if turn.Current {
			name = "[" + name + "]"
		}

		order = append(order, name)
	}

	// The last column scrolls so that the selection stays in view.
	first := min(max(g.choice-choiceWindow+1, 0), max(len(options)-choiceWindow, 0))
	shown := options[first:min(first+choiceWindow, len(options))]

	lines := []string{title, labelStyle.Render("Kolejność: ") + strings.Join(order, " › ")}
	actions := g.actions()

	for row := range g.fightHeight() - 4 {
		var target, action, option string

		if row < len(fight.Monsters) {
			foe := fight.Monsters[row]
			target = g.pointer(stepTarget, row == g.target) + healthBar(foe.HP, foe.MaxHP, fightBar) + " " + foe.Name +
				labelStyle.Render(fmt.Sprintf(" %d/%d", foe.HP, foe.MaxHP))

			if fight.Board != nil {
				target += labelStyle.Render(fmt.Sprintf(" KP %d", foe.AC))
			}

			target += dimStyle.Render(" " + foe.Trait)
		}

		if row < len(actions) {
			action = g.pointer(stepAction, row == g.action) + actions[row]
		}

		if row < len(shown) && g.step == stepChoice {
			label := shown[row].label
			if !shown[row].usable {
				label = emptyStyle.Render(label)
			}

			option = g.pointer(stepChoice, first+row == g.choice) + label
		}

		lines = append(lines, column(target, targetWidth)+column(action, actionWidth)+column(option, choiceWidth))
	}

	return strings.Join(lines, "\n")
}

// pointer marks the selected row of a column, brightly in the column the
// player is choosing in.
func (g Game) pointer(step int, selected bool) string {
	switch {
	case !selected:
		return "  "
	case step == g.step && g.view.Fight.Actor != nil:
		return selfStyle.Render("> ")
	}

	return labelStyle.Render("› ")
}

// column fits one line of text into a column of the given width.
func column(text string, width int) string {
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true).Render(text)
}

// fightHelp words what the keys do at the current step of the menu.
func (g Game) fightHelp() string {
	fight := g.view.Fight

	if fight.Actor == nil {
		if fight.Waiting != "" {
			return "Tura: " + fight.Waiting + ". Czekasz."
		}

		return "Czekasz…"
	}

	if options := g.choices(); g.step == stepChoice && g.choice < len(options) {
		return options[g.choice].note + "   Enter użyj · ← wróć"
	}

	if g.step == stepMove && fight.Board != nil {
		return fmt.Sprintf("Ruch: strzałki przesuwają %s po polu, zostało %d pól · Enter lub Esc kończy ruch", fight.Actor.Name, fight.Board.Moves)
	}

	return "↑↓ wybór · Enter dalej · ← wróć   Decydujesz za: " + fight.Actor.Name + " · F1-F3 karty · autowalka"
}
