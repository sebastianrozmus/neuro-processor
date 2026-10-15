package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/inline"
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
		panel(leftWidth, bodyHeight-mapHeight, g.partyView(), false),
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

// partyView draws the party as tabs, one for each member with the key
// that picks them, and the card of the chosen one under the tabs.
func (g Game) partyView() string {
	all := g.cards()
	selected := min(g.card, len(all)-1)
	lines := []string{tabBar(all, selected, leftWidth-4), g.fullCard(all[selected])}

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

// tabBar draws the members as tabs with the keys that pick them: the
// chosen one lit and underlined, the others dim. First names only, so
// that three fit beside each other.
func tabBar(all []card, selected, width int) string {
	var tabs, rule []string

	for index, member := range all {
		label := fmt.Sprintf(" F%d %s ", index+1, strings.Fields(member.name)[0])
		if index == selected {
			tabs = append(tabs, tabOnStyle.Render(label))
			rule = append(rule, goldBar.Render(strings.Repeat("━", lipgloss.Width(label))))
		} else {
			tabs = append(tabs, labelStyle.Render(label))
			rule = append(rule, emptyStyle.Render(strings.Repeat("─", lipgloss.Width(label))))
		}
	}

	under := strings.Join(rule, "")
	if rest := width - lipgloss.Width(under); rest > 0 {
		under += emptyStyle.Render(strings.Repeat("─", rest))
	}

	clip := lipgloss.NewStyle().MaxWidth(width)

	return clip.Render(strings.Join(tabs, "")) + "\n" + clip.Render(under)
}

// fullCard draws a member's portrait with their figures and gear beside it.
func (g Game) fullCard(member card) string {
	name := selfStyle.Render(member.name)
	experience := gauge("EXP ", member.xp, member.nextXP, goldBar)
	if member.level >= game.MaxLevel {
		experience = labelStyle.Render("EXP ") + goldStyle.Render("maks.")
	}

	if g.view.Rooms() {
		clock := labelStyle.Render("Tab: plecak")
		if g.view.Investigation() || g.view.HoursLeft >= 0 {
			clock = labelStyle.Render(fmt.Sprintf("Dzień %d, %s", g.view.Me.Day, g.view.Hour))
			if g.view.HoursLeft >= 0 {
				clock = warnStyle.Render(fmt.Sprintf("Zostało %d h", g.view.HoursLeft))
			}
		}

		stats := []string{name, titleStyle.Render(g.view.World.Name), "", labelStyle.Render(fmt.Sprintf("Rzeczy: %d", member.bag)), clock}
		if avatar := g.portrait(member.avatar); avatar != "" {
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

	if avatar := g.portrait(member.avatar); avatar != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, avatar, " ", strings.Join(stats, "\n"))
	}

	return strings.Join(stats, "\n")
}

// portrait is the small picture on a card: the real one when the
// terminal shows images and the player drew the avatar, else block art.
func (g Game) portrait(avatar string) string {
	if g.inline {
		if picture := g.picture(avatar, avatarColumns, avatarColumns/2); picture != "" {
			return picture
		}
	}

	return g.library.Art(avatar)
}

// picture places the PNG of a player-made avatar, if the terminal shows
// images and there is one.
func (g Game) picture(avatar string, cols, rows int) string {
	if !g.inline || g.library == nil || avatar == "" {
		return ""
	}

	png, path := g.library.Picture(avatar)

	return inline.Show(path, png, cols, rows)
}

// combatLine is the figures of a card: in the D&D mode the armour class
// stands where the defence does otherwise.
func (g Game) combatLine(member card) string {
	if g.view.DnD() {
		return fmt.Sprintf("%s %d %s %d %s %d", labelStyle.Render("Atk"), member.attack, labelStyle.Render("KP"), member.ac, labelStyle.Render("Szyb"), member.speed)
	}

	return fmt.Sprintf("%s %d %s %d %s %d", labelStyle.Render("Atk"), member.attack, labelStyle.Render("Obr"), member.defense, labelStyle.Render("Szyb"), member.speed)
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

	picture := lipgloss.PlaceHorizontal(width, lipgloss.Center, scene(g.view, g.target, g.effects, g.frame, g.step == stepMove, g.inline))

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
			line = titleStyle.Render(game.MasterMark) + g.names.paint(strings.TrimPrefix(line, game.MasterMark))
		default:
			line = g.names.paint(line)
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
	if g.view.Rooms() {
		return g.thingsView()
	}

	index := min(g.card, len(g.view.Me.Party))
	who := g.sheetOf(index)
	width := g.width - leftWidth - 4

	head := []string{
		tabBar(g.cards(), index, width),
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

		lines = append(lines, pointer+glyph(item.Kind, item.Effect)+" "+item.Name+" "+labelStyle.Render(game.Effect(item)))
	}

	if g.cursor < len(who.bag) {
		item := who.bag[g.cursor]
		lines = append(lines, "", withIcon(icon(item.Kind, item.Effect, item.Tier, item.Name), item.Description, width))
	}

	lines = append(lines, "", titleStyle.Render(fmt.Sprintf("Moce (%d)", len(who.spells))))

	if len(who.spells) == 0 {
		lines = append(lines, dimStyle.Render("Żadnych. Uczy się ich z przedmiotów i każda zostaje na stałe."))
	}

	for _, spell := range who.spells {
		lines = append(lines, "  "+glyph(spell.Kind, spell.Effect)+" "+spell.Name+" "+labelStyle.Render(game.SpellEffect(spell)))
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

		lines = append(lines, pointer+glyph(item.Kind, item.Effect)+" "+item.Name)
	}

	if g.cursor < len(g.view.Me.Bag) {
		item := g.view.Me.Bag[g.cursor]
		lines = append(lines, "", withIcon(icon(item.Kind, item.Effect, item.Tier, item.Name), item.Description, g.width-leftWidth-4))
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
	case g.view.Investigation():
		help = "rozmawiaj z Y · zbadaj X · czekaj · śledź Y · dziennik · oskarżam Y · pomoc   Strzałki ruch · Tab plecak · Esc kończy rozmowę"
	case g.view.Rooms():
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

	switch {
	case g.view.Lost:
		help = badStyle.Render("× Przegrana × ") + help
	case g.view.Won && g.view.Investigation():
		help = goldStyle.Render("★ Sprawa rozwiązana ★ ") + help
	case g.view.Won && g.view.Rooms():
		help = goldStyle.Render("★ Przygoda ukończona ★ ") + help
	case g.view.Won:
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
