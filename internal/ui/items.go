package ui

import (
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Items are drawn as small icons: one drawing for each shape of thing the
// worlds sell (a sword, an axe, a flask, a scroll…), picked by the kind
// and the name, coloured by the effect. The letters are the figures'
// palette; L is the liquid, glow or stone that takes the effect's colour.

// An icon is 10 pixels square: 10 columns and, in half blocks, 5 rows.
const (
	iconColumns = 10
	iconRows    = 5
)

var iconPalette = func() map[rune]uint8 {
	palette := map[rune]uint8{'x': 252, 'd': 240}
	for letter, colour := range figurePalette {
		palette[letter] = colour
	}

	return palette
}()

var iconDrawings = map[string][]string{
	"sword": {
		"........xx",
		".......xxx",
		"......xxx.",
		".....xxx..",
		"....xxx...",
		".y.xxx....",
		"..yyy.....",
		".yyyy.....",
		"yy..y.....",
		"..........",
	},
	"axe": {
		"....tt....",
		"...xxtxx..",
		"..xxxtxxx.",
		"..xxxtxxx.",
		"...xxtxx..",
		"....tt....",
		"....tt....",
		"....tt....",
		"....tt....",
		"....tt....",
	},
	"bow": {
		"....tt....",
		"...t..w...",
		"..t...w...",
		"..t...w...",
		".t....w...",
		"..t...w...",
		"..t...w...",
		"...t..w...",
		"....tt....",
		"..........",
	},
	"dagger": {
		"..........",
		"..........",
		"......xx..",
		".....xxx..",
		"....xxx...",
		"...xxx....",
		"..yyy.....",
		".kky......",
		"kk........",
		"..........",
	},
	"hammer": {
		"..xxxxxx..",
		"..xxxxxx..",
		"..xxxxxx..",
		"..xxxxxx..",
		"....tt....",
		"....tt....",
		"....tt....",
		"....tt....",
		"....tt....",
		"..........",
	},
	"spear": {
		"........x.",
		".......xx.",
		"......xx..",
		".....tt...",
		"....tt....",
		"...tt.....",
		"..tt......",
		".tt.......",
		"tt........",
		"..........",
	},
	"gun": {
		"..........",
		"..........",
		".AAAAAAAA.",
		".AAAAAAAAA",
		"...AAkk...",
		"...AAk....",
		"...AA.....",
		"...AA.....",
		"..........",
		"..........",
	},
	"staff": {
		".....LL...",
		"....LLLL..",
		"....LLLL..",
		".....tt...",
		".....tt...",
		"....tt....",
		"....tt....",
		"...tt.....",
		"...tt.....",
		"..tt......",
	},
	"armor": {
		".AA....AA.",
		"AAaA..AaAA",
		"AAaAAAAaAA",
		"AAaAAAAaAA",
		".AAAAAAAA.",
		".AAaAAaAA.",
		".AAAAAAAA.",
		"..AAAAAA..",
		"..AAAAAA..",
		"..........",
	},
	"shield": {
		".aaaaaaaa.",
		"aAAAAAAAAa",
		"aAAAyyAAAa",
		"aAAAyyAAAa",
		"aAAAAAAAAa",
		".aAAAAAAa.",
		"..aAAAAa..",
		"...aAAa...",
		"....aa....",
		"..........",
	},
	"helmet": {
		"...AAAA...",
		"..AaAAAA..",
		".AaAAAAAA.",
		".AAAAAAAA.",
		".AA.AA.AA.",
		".AAAAAAAA.",
		"..A....A..",
		"..........",
		"..........",
		"..........",
	},
	"flask": {
		"....ww....",
		"....ww....",
		"...wwww...",
		"...w..w...",
		"..w....w..",
		"..wLLLLw..",
		".wLLLLLLw.",
		".wLLLLLLw.",
		"..wLLLLw..",
		"...wwww...",
	},
	"scroll": {
		"..........",
		".tssssssst",
		".t.ssssst.",
		"...ssssss.",
		"...sLLsss.",
		"...sLLsss.",
		"...ssssss.",
		"...ssssss.",
		".t.ssssst.",
		".tssssssst",
	},
	"book": {
		"..........",
		".LLLLLLLL.",
		".LwwwwwwL.",
		".LwwwwwwL.",
		".LwLLLLwL.",
		".LwwwwwwL.",
		".LwwwwwwL.",
		".LwwwwwwL.",
		".LLLLLLLL.",
		"..........",
	},
	"chip": {
		"..........",
		".k.k.k.k..",
		"dddddddddd",
		"dLLLLLLLLd",
		"dLddddddLd",
		"dLddddddLd",
		"dLLLLLLLLd",
		"dddddddddd",
		".k.k.k.k..",
		"..........",
	},
	"amulet": {
		"....yy....",
		"...y..y...",
		"..y....y..",
		"..y....y..",
		"...y..y...",
		"....yy....",
		"...yLLy...",
		"..yLLLLy..",
		"...yLLy...",
		"....yy....",
	},
	"ring": {
		"..........",
		"....LL....",
		"...LLLL...",
		"..yy..yy..",
		".yy....yy.",
		".yy....yy.",
		"..yy..yy..",
		"...yyyy...",
		"..........",
		"..........",
	},
}

// shapeWords pick a drawing by a word in the item's name; the first match
// in the list wins, the kind decides when none matches.
var shapeWords = []struct {
	shape string
	stems []string
}{
	{"axe", []string{"topor", "siekier", "halabard", "berdysz"}},
	{"bow", []string{"luk", "kusz", "proc"}},
	{"dagger", []string{"sztylet", "noz", "kozik", "ostrz"}},
	{"hammer", []string{"mlot", "maczug", "bulaw", "palk", "obuch", "klucz"}},
	{"spear", []string{"wloczn", "pik", "oszczep", "harpun", "dzid"}},
	{"gun", []string{"pistolet", "karabin", "strzelb", "rewolwer", "blaster", "miotacz", "laser", "obrzyn", "paralizator"}},
	{"staff", []string{"kij", "lask", "kostur", "rozdz", "berl"}},
	{"shield", []string{"tarcz", "puklerz", "pawez"}},
	{"helmet", []string{"helm", "kask", "przylbic", "czapk", "kaptur", "maska"}},
	{"book", []string{"ksieg", "tom", "kodeks", "podrecznik", "manuskrypt", "grimuar"}},
	{"chip", []string{"program", "chip", "dysk", "modul", "schemat", "karta", "implant", "oprogramowanie"}},
	{"ring", []string{"pierscien", "sygnet", "obracz", "bransolet"}},
}

var kindShapes = map[world.Kind]string{
	world.KindWeapon: "sword", world.KindArmor: "armor", world.KindPotion: "flask", world.KindSpell: "scroll", world.KindCharm: "amulet",
}

// effectColours are the letters L stands for, by effect.
var effectColours = map[world.Effect]rune{
	world.EffectHeal: 'r', world.EffectMana: 'b', world.EffectBolt: 'o', world.EffectBlast: 'r',
	world.EffectPower: 'r', world.EffectWard: 'b', world.EffectVigor: 'g', world.EffectFocus: 'p',
}

// kindGlyphs mark an item's kind in a list, before its name.
var kindGlyphs = map[world.Kind]string{
	world.KindWeapon: "†", world.KindArmor: "◈", world.KindPotion: "δ", world.KindSpell: "✧", world.KindCharm: "◉",
}

var icons sync.Map

// shapeOf picks the drawing for an item.
func shapeOf(kind world.Kind, name string) string {
	folded := strings.Fields(strings.ToLower(fold(name)))

	for _, candidate := range shapeWords {
		for _, word := range folded {
			for _, stem := range candidate.stems {
				if strings.HasPrefix(word, stem) {
					return candidate.shape
				}
			}
		}
	}

	if shape, ok := kindShapes[kind]; ok {
		return shape
	}

	return "amulet"
}

// fold strips Polish diacritics, so that stems match however the name is
// spelled.
func fold(text string) string {
	return strings.NewReplacer("ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n", "ó", "o", "ś", "s", "ź", "z", "ż", "z",
		"Ą", "a", "Ć", "c", "Ę", "e", "Ł", "l", "Ń", "n", "Ó", "o", "Ś", "s", "Ź", "z", "Ż", "z").Replace(text)
}

// itemIcon draws an item: its shape in the effect's colour, framed in
// gold when it is of the highest tier.
func itemIcon(kind world.Kind, effect world.Effect, tier int, name string) pix.Image {
	shape := shapeOf(kind, name)
	colour, ok := effectColours[effect]
	if !ok {
		colour = 'p'
		if kind == world.KindPotion {
			colour = 'g'
		}
	}

	// Leather armour is brown, not steel.
	leather := false
	for _, stem := range []string{"skor", "kaftan", "kamizel", "kurtk", "plaszcz"} {
		leather = leather || kind == world.KindArmor && strings.Contains(fold(strings.ToLower(name)), stem)
	}

	key := shape + string(colour) + strconv.Itoa(tier)
	if leather {
		key += "t"
	}

	if cached, ok := icons.Load(key); ok {
		return cached.(pix.Image)
	}

	lines := make([]string, len(iconDrawings[shape]))
	for index, line := range iconDrawings[shape] {
		line = strings.ReplaceAll(line, "L", string(colour))
		if leather {
			line = strings.NewReplacer("A", "t", "a", "h").Replace(line)
		}

		lines[index] = line
	}

	drawn := parseIcon(lines)
	switch tier {
	case 3:
		drawn.Frame(iconPalette['y'], 1)
	case 2:
		drawn.Frame(iconPalette['A'], 1)
	}

	icons.Store(key, drawn)

	return drawn
}

// parseIcon turns the lines of a drawing into a picture, as parseFigure
// does, with the icons' palette.
func parseIcon(lines []string) pix.Image {
	img := pix.New(iconColumns, len(lines))

	for y, line := range lines {
		for x, char := range []rune(line) {
			if colour, ok := iconPalette[char]; ok {
				img.Set(x, y, colour)
			}
		}
	}

	return img
}

// icon renders an item's picture for a panel.
func icon(kind world.Kind, effect world.Effect, tier int, name string) string {
	return itemIcon(kind, effect, tier, name).Render()
}

// glyph marks an item's kind in a list, in the colour of its effect.
func glyph(kind world.Kind, effect world.Effect) string {
	mark, ok := kindGlyphs[kind]
	if !ok {
		mark = "•"
	}

	colour, ok := effectColours[effect]
	if !ok {
		colour = 'a'
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color(strconv.Itoa(int(iconPalette[colour])))).Render(mark)
}

// withIcon puts an item's picture beside a description, which wraps to
// the width left.
func withIcon(picture, text string, width int) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, picture, "  ", dimStyle.Width(max(1, width-iconColumns-2)).Render(text))
}
