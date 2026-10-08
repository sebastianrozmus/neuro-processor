package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	cellWidth     = 3
	leftWidth     = world.Width*cellWidth + 4
	mapHeight     = world.Height + 3
	sceneHeightP  = sceneRows + 3
	inputHeight   = 3
	barWidth      = 8
	gaugeWidth    = 12
	avatarColumns = 20
)

// View renders the current screen.
func (g Game) View() string {
	if g.width < MinWidth || g.height < MinHeight {
		return fmt.Sprintf("Powiększ okno terminala do co najmniej %dx%d (teraz %dx%d).", MinWidth, MinHeight, g.width, g.height)
	}

	if g.view.Shop != nil {
		return g.shopView()
	}

	bodyHeight := g.height - inputHeight - 1
	rightWidth := g.width - leftWidth

	left := lipgloss.JoinVertical(lipgloss.Left,
		panel(leftWidth, mapHeight, titleStyle.Render(g.view.World.Name)+"\n"+g.mapView(), false),
		panel(leftWidth, bodyHeight-mapHeight, g.partyView(bodyHeight-mapHeight-2), false),
	)

	right := panel(rightWidth, bodyHeight, g.bagView(), false)

	if g.book {
		right = panel(rightWidth, bodyHeight, g.bookView(rightWidth-4), false)
	}

	if !g.bag && !g.book {
		parts := []string{panel(rightWidth, sceneHeightP, g.sceneView(rightWidth-4), false)}
		rest := bodyHeight - sceneHeightP

		if g.view.Fight != nil {
			height := min(g.fightHeight(), rest-3)
			parts = append(parts, panel(rightWidth, height, g.fightView(rightWidth-4), false))
			rest -= height
		}

		right = lipgloss.JoinVertical(lipgloss.Left, append(parts, panel(rightWidth, rest, g.logView(), true))...)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		box.Width(g.width-2).Render(g.input.View()),
		g.helpView(),
	)
}

func (g Game) mapView() string {
	rows := make([]string, 0, world.Height)

	for y := range world.Height {
		var row strings.Builder

		for x := range world.Width {
			row.WriteString(g.cell(world.Point{X: x, Y: y}))
		}

		rows = append(rows, row.String())
	}

	return strings.Join(rows, "\n")
}

// cell draws one map tile: its terrain and, in the middle, whatever stands
// on it.
func (g Game) cell(point world.Point) string {
	index := g.view.World.Index(point)
	if !g.view.Revealed[index] {
		return emptyStyle.Render(" · ")
	}

	place := g.view.PlaceAt(point)
	if place.Name == "" {
		return emptyStyle.Render("   ")
	}

	terrain := terrainLooks[place.Terrain]
	texture := []rune(terrain.textures[(point.X*5+point.Y*3)%len(terrain.textures)])
	tile := g.view.World.At(point)
	background := terrain.style.GetBackground()

	var marker string

	switch mark := g.view.Marks[index]; {
	case point == g.view.Me.Pos:
		marker = meMarker.Render("@")
	case g.someoneAt(point):
		marker = friendMarker.Render("&")
	case mark.Boss:
		marker = foeMarker.Render("B")
	case mark.Monsters > 0:
		marker = foeMarker.Render("m")
	case mark.Caravan:
		marker = caravanMarker.Background(background).Render("$")
	case tile.Town >= 0:
		marker = townMarker.Background(background).Render("⌂")
	case tile.Site >= 0 && g.view.World.Sites[tile.Site].Kind == world.SiteShrine:
		marker = siteMarker.Background(background).Render("+")
	case tile.Site >= 0:
		marker = siteMarker.Background(background).Render("?")
	default:
		return terrain.style.Render(string(texture))
	}

	return terrain.style.Render(string(texture[0])) + marker + terrain.style.Render(string(texture[2]))
}

func (g Game) someoneAt(point world.Point) bool {
	for _, other := range g.view.Others {
		if other.Pos == point {
			return true
		}
	}

	return false
}

// card is what the left panel shows about one member of the party.
type card struct {
	name, title, avatar      string
	level, xp, nextXP        int
	hp, maxHP, mana, maxMana int
	gold                     string
	attack, defense, speed   int
	ac                       int
	weapon, armor, charm     string
	bag, spells              int
	note                     string
}

// cards lists the hero and their companions as the panel shows them.
func (g Game) cards() []card {
	me := g.view.Me
	all := []card{{
		name: me.Name, title: fmt.Sprintf("Poziom %d", me.Level), avatar: me.Avatar,
		level: me.Level, xp: me.XP, nextXP: me.NextXP, hp: me.HP, maxHP: me.MaxHP, mana: me.Mana, maxMana: me.MaxMana,
		gold:   fmt.Sprintf("%d %s", me.Gold, g.view.World.Currency),
		attack: me.Attack, defense: me.Defense, speed: me.Speed, ac: me.AC,
		weapon: me.Weapon, armor: me.Armor, charm: me.Charm, bag: len(me.Bag), spells: len(me.Spells),
	}}

	for _, ally := range me.Party {
		note := ""

		switch {
		case ally.HP <= 0:
			note = "nieprzytomny"
		case ally.Auto || me.AutoFight:
			note = "walczy sam"
		case ally.Legend:
			note = fmt.Sprintf("żołd %d dziennie", ally.Wage)
		}

		all = append(all, card{
			name: ally.Name, title: fmt.Sprintf("%s, poziom %d", ally.Role, ally.Level), avatar: ally.Avatar,
			level: ally.Level, xp: ally.XP, nextXP: ally.NextXP, hp: ally.HP, maxHP: ally.MaxHP, mana: ally.Mana, maxMana: ally.MaxMana,
			gold: fmt.Sprintf("plecak %d, moce %d", len(ally.Bag), len(ally.Spells)), attack: ally.Attack, defense: ally.Defense, speed: ally.Speed, ac: ally.AC,
			weapon: ally.Weapon, armor: ally.Armor, charm: ally.Charm, bag: len(ally.Bag), spells: len(ally.Spells), note: note,
		})
	}

	return all
}

// cardRows is the height of a full card: the portrait with the figures
// beside it.
const cardRows = 10

func (g Game) heroView() string {
	return g.partyView(0)
}

// partyView draws the hero and the companions one under another. Cards
// that do not fit the given height are folded to a line each, except the
// selected one; height 0 means unlimited.
func (g Game) partyView(height int) string {
	all := g.cards()
	selected := min(g.card, len(all)-1)
	others := len(g.view.Others)
	if others > 0 {
		others++
	}

	full := map[int]bool{}
	if height == 0 || cardRows*len(all)+len(all)-1+others <= height {
		for index := range all {
			full[index] = true
		}
	} else {
		full[selected] = true
	}

	var lines []string

	for index, member := range all {
		if len(lines) > 0 {
			lines = append(lines, "")
		}

		if full[index] {
			lines = append(lines, g.fullCard(index, member, index == selected))
		} else {
			lines = append(lines, g.foldedCard(index, member))
		}
	}

	if len(g.view.Others) > 0 {
		lines = append(lines, "", labelStyle.Render("Inni gracze"))
	}

	for _, other := range g.view.Others {
		where := g.view.PlaceAt(other.Pos).Name
		if other.Pos == g.view.Me.Pos {
			where = "obok ciebie"
		}

		if other.Fighting {
			where += ", walczy"
		}

		lines = append(lines, fmt.Sprintf("%s %s %s", playerStyle.Render(other.Name), healthBar(other.HP, other.MaxHP, barWidth/2), labelStyle.Render(where)))
	}

	return strings.Join(lines, "\n")
}

// fullCard draws a member's portrait with their figures and gear beside it.
func (g Game) fullCard(index int, member card, selected bool) string {
	name := fmt.Sprintf("F%d %s", index+1, member.name)
	if selected {
		name = selfStyle.Render(name)
	} else {
		name = playerStyle.Render(name)
	}

	experience := gauge("EXP ", member.xp, member.nextXP, goldBar)
	if member.level >= game.MaxLevel {
		experience = labelStyle.Render("EXP ") + goldStyle.Render("maks.")
	}

	if g.view.Adventure() {
		stats := []string{name, titleStyle.Render(g.view.World.Name), "", labelStyle.Render(fmt.Sprintf("Rzeczy: %d", member.bag)), labelStyle.Render("Tab: plecak")}
		if avatar := g.library.Art(member.avatar); avatar != "" {
			return lipgloss.JoinHorizontal(lipgloss.Top, avatar, " ", strings.Join(stats, "\n"))
		}

		return strings.Join(stats, "\n")
	}

	stats := []string{
		name,
		titleStyle.Render(member.title),
		gauge("HP  ", member.hp, member.maxHP, healthStyle(member.hp, member.maxHP)),
		gauge("MANA", member.mana, member.maxMana, manaStyle),
		experience,
		g.combatLine(member),
		labelStyle.Render("Broń ") + orNone(member.weapon),
		labelStyle.Render("Panc ") + orNone(member.armor),
		labelStyle.Render("Amul ") + orNone(member.charm),
		goldStyle.Render(member.gold),
	}

	if member.note != "" {
		stats[len(stats)-1] = warnStyle.Render(member.note)
	}

	// The column beside the portrait must not wrap, or the portrait breaks.
	column := lipgloss.NewStyle().MaxWidth(leftWidth - 4 - avatarColumns - 1)
	for index, line := range stats {
		stats[index] = column.Render(line)
	}

	if avatar := g.library.Art(member.avatar); avatar != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, avatar, " ", strings.Join(stats, "\n"))
	}

	return strings.Join(stats, "\n")
}

// combatLine is the figures of a card: in the D&D mode the armour class
// stands where the defence does otherwise.
func (g Game) combatLine(member card) string {
	if g.view.DnD() {
		return fmt.Sprintf("%s %d %s %d %s %d", labelStyle.Render("Atk"), member.attack, labelStyle.Render("KP"), member.ac, labelStyle.Render("Szyb"), member.speed)
	}

	return fmt.Sprintf("%s %d %s %d %s %d", labelStyle.Render("Atk"), member.attack, labelStyle.Render("Obr"), member.defense, labelStyle.Render("Szyb"), member.speed)
}

// foldedCard sums a member up in one line.
func (g Game) foldedCard(index int, member card) string {
	return fmt.Sprintf("%s %s %s %d/%d %s %d/%d",
		playerStyle.Render(fmt.Sprintf("F%d %s", index+1, member.name)),
		labelStyle.Render(member.title),
		healthBar(member.hp, member.maxHP, barWidth/2), member.hp, member.maxHP,
		bar(member.mana, member.maxMana, barWidth/2, manaStyle), member.mana, member.maxMana)
}

// gauge draws a labelled bar beside the portrait with its numbers.
func gauge(label string, value, limit int, style lipgloss.Style) string {
	return fmt.Sprintf("%s %s %d/%d", labelStyle.Render(label), bar(value, limit, gaugeWidth, style), value, limit)
}

// sceneView draws the picture of the place under its name.
func (g Game) sceneView(width int) string {
	title := g.view.Place.Name
	if g.view.Tile.Site >= 0 {
		title = g.view.World.Sites[g.view.Tile.Site].Name
	}

	picture := lipgloss.PlaceHorizontal(width, lipgloss.Center, scene(g.view, g.target, g.effects, g.frame, g.step == stepMove))

	return titleStyle.Render(title) + "\n" + picture
}

// logView draws the transcript; the player's own commands stand out dimmed.
func (g Game) logView() string {
	lines := make([]string, 0, len(g.view.Log))

	for _, line := range g.view.Log {
		switch {
		case strings.HasPrefix(line, "> "):
			line = dimStyle.Render(line)
		case strings.HasPrefix(line, game.MasterMark):
			line = titleStyle.Render(game.MasterMark) + strings.TrimPrefix(line, game.MasterMark)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// sheet gathers what the character sheet shows about one member.
type sheet struct {
	card
	description string
	abilities   game.Abilities
	bag, spells []world.ItemSpec
	wage, day   int
	legend      bool
	sky         string
}

func (g Game) sheetOf(index int) sheet {
	me := g.view.Me
	all := g.cards()

	if index == 0 || index > len(me.Party) {
		return sheet{card: all[0], description: "Bohater tej opowieści.", abilities: me.Abilities, bag: me.Bag, spells: me.Spells, day: me.Day, sky: fmt.Sprintf("%s, %s", g.view.Season, g.view.Weather)}
	}

	ally := me.Party[index-1]

	return sheet{card: all[index], description: ally.Description, abilities: ally.Abilities, bag: ally.Bag, spells: ally.Spells, wage: ally.Wage, legend: ally.Legend}
}

// bagView draws the character sheet of the selected member: scores,
// combat figures, gear, bag and powers, as on a sheet of paper.
func (g Game) bagView() string {
	if g.view.Adventure() {
		return g.thingsView()
	}

	index := min(g.card, len(g.view.Me.Party))
	who := g.sheetOf(index)

	head := []string{
		selfStyle.Render(who.name) + "  " + titleStyle.Render(who.title),
		dimStyle.Render(who.description),
	}

	if who.legend {
		head = append(head, warnStyle.Render(fmt.Sprintf("Legenda na żołdzie: %d %s dziennie, płatne o świcie.", who.wage, g.view.World.Currency)))
	}

	if who.day > 0 {
		head = append(head, labelStyle.Render(fmt.Sprintf("Dzień %d w tym świecie, %s.", who.day, who.sky)))
	}

	if index == 0 && len(g.view.Me.Standing) > 0 {
		var fame []string
		for _, standing := range g.view.Me.Standing {
			fame = append(fame, fmt.Sprintf("%s: %s (%+d)", standing.Town, standing.Label, standing.Score))
		}

		head = append(head, labelStyle.Render("Sława: "+strings.Join(fame, ", ")+"."))
	}

	scores := make([]string, 0, len(who.abilities))
	for index, score := range who.abilities {
		scores = append(scores, fmt.Sprintf("%s %2d (%s)", labelStyle.Render(game.AbilityShort[index]), score, signed(game.Modifier(score))))
	}

	combat := []string{
		fmt.Sprintf("%s %d/%d   %s %d/%d   %s %d/%d", labelStyle.Render("Zdrowie"), who.hp, who.maxHP, labelStyle.Render("Moc"), who.mana, who.maxMana, labelStyle.Render("Dośw."), who.xp, who.nextXP),
		fmt.Sprintf("%s %d   %s %d   %s %d   %s %d", labelStyle.Render("Atak"), who.attack, labelStyle.Render("Obrona"), who.defense, labelStyle.Render("KP"), who.ac, labelStyle.Render("Szybkość"), who.speed),
		fmt.Sprintf("%s %s   %s %s   %s %s", labelStyle.Render("Broń"), orNone(who.weapon), labelStyle.Render("Pancerz"), orNone(who.armor), labelStyle.Render("Amulet"), orNone(who.charm)),
	}

	lines := append(head, "", titleStyle.Render("Cechy"), strings.Join(scores[:3], "   "), strings.Join(scores[3:], "   "), "", titleStyle.Render("Walka"))
	lines = append(lines, combat...)
	lines = append(lines, "", titleStyle.Render(fmt.Sprintf("Plecak (%d)", len(who.bag))))

	if len(who.bag) == 0 {
		lines = append(lines, dimStyle.Render("Pusty."))
	}

	for position, item := range who.bag {
		pointer := "  "
		if position == g.cursor {
			pointer = selfStyle.Render("> ")
		}

		lines = append(lines, pointer+item.Name+" "+labelStyle.Render(game.Effect(item)))
	}

	if g.cursor < len(who.bag) {
		lines = append(lines, dimStyle.Render(who.bag[g.cursor].Description))
	}

	lines = append(lines, "", titleStyle.Render(fmt.Sprintf("Moce (%d)", len(who.spells))))

	if len(who.spells) == 0 {
		lines = append(lines, dimStyle.Render("Żadnych. Uczy się ich z przedmiotów i każda zostaje na stałe."))
	}

	for _, spell := range who.spells {
		lines = append(lines, "  "+spell.Name+" "+labelStyle.Render(game.SpellEffect(spell)))
	}

	if index > 0 {
		lines = append(lines, "", dimStyle.Render("Rzeczy dajesz poleceniem «daj X "+who.name+"», odbierasz «odbierz X od "+who.name+"»."))
	}

	return strings.Join(lines, "\n")
}

// thingsView is the bag of an adventure: the things carried, with the
// description of the chosen one.
func (g Game) thingsView() string {
	lines := []string{selfStyle.Render(g.view.Me.Name), "", titleStyle.Render(fmt.Sprintf("Rzeczy (%d)", len(g.view.Me.Bag)))}

	if len(g.view.Me.Bag) == 0 {
		lines = append(lines, dimStyle.Render("Nic. Rzeczy podnosisz poleceniem «podnieś X»."))
	}

	for position, item := range g.view.Me.Bag {
		pointer := "  "
		if position == g.cursor {
			pointer = selfStyle.Render("> ")
		}

		lines = append(lines, pointer+item.Name)
	}

	if g.cursor < len(g.view.Me.Bag) {
		lines = append(lines, "", dimStyle.Render(g.view.Me.Bag[g.cursor].Description))
	}

	return strings.Join(lines, "\n")
}

// signed writes a modifier the way a character sheet does: +2, -1, +0.
func signed(value int) string {
	if value >= 0 {
		return fmt.Sprintf("+%d", value)
	}

	return fmt.Sprint(value)
}

func (g Game) helpView() string {
	var help string

	switch {
	case g.book:
		help = "Bestiariusz   ↑↓ strona · Esc wróć   Nieznane gatunki poznasz w walce."
	case g.bag:
		help = "Karta postaci   ↑↓ wybór w plecaku · Enter użyj, załóż albo naucz się · F1-F3 czyja karta · Esc wróć"
	case g.view.Fight != nil:
		help = g.fightHelp()
	case g.view.Adventure():
		help = "idź na północ / n · rozejrzyj się · zbadaj X · podnieś X · użyj X na Y · otwórz X · daj X Y · rozmawiaj z Y · pomoc   Strzałki ruch · Tab plecak"
	case g.view.Tile.Town >= 0:
		help = "idź do X (budynek) · najmij X · zlecenia · przyjmij · odpocznij · zapisz · pomoc   Strzałki ruch · Tab plecak · F4 bestiariusz · F1-F3 karty"
	default:
		help = "idź na północ / n · rozejrzyj się · podnieś X · pomoc   Strzałki ruch · Tab plecak · F4 bestiariusz   @ ty  & drużyna  m wrogowie  B boss  ⌂ osada  ? + miejsca  $ karawana"
	}

	if g.hearing != "" {
		help = badStyle.Render(g.hearing) + "   " + help
	} else if g.ear != nil {
		help += " · F5 mów"
	}

	if g.view.Won && g.view.Adventure() {
		help = goldStyle.Render("★ Przygoda ukończona ★ ") + help
	} else if g.view.Won {
		help = goldStyle.Render("★ Świat ocalony ★ ") + help
	}

	return lipgloss.NewStyle().MaxWidth(g.width).Render(" " + labelStyle.Render(help))
}

func orNone(name string) string {
	if name == "" {
		return labelStyle.Render("brak")
	}

	return name
}
