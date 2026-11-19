package ui

import (
	"strings"
	"sync"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The heroes' side of a battlefield is drawn with small figures: one for
// each built-in look of a hero, one for each role of companion, and a
// plain one for looks the game has not seen. They face right, towards
// the monsters.

// figurePalette maps the letters of the drawings to palette colours.
var figurePalette = map[rune]uint8{
	's': 223, 'h': 94, 'a': 250, 'A': 243, 'b': 27, 'B': 18, 'r': 160, 'g': 34, 'G': 22,
	'y': 220, 'w': 231, 'k': 232, 'p': 93, 'o': 208, 't': 130, 'c': 51,
}

var figureDrawings = map[string][]string{
	"wojownik": {
		"....aaaa..",
		"....asss..",
		"....ass...",
		".....ss...",
		"..wAAAA.a.",
		".wwAAAA.a.",
		".wwAAAAaa.",
		"..AAAA..a.",
		"..tttt....",
		"..tt.tt...",
		"..kk.kk...",
		"..kk..kk..",
	},
	"czarodziej": {
		".....p....",
		"....ppp...",
		"...ppppp..",
		"....sss...",
		"....ss..y.",
		"...pppp.y.",
		"..pppppyy.",
		"..ppppp.y.",
		"..ppppp.y.",
		"..ppppp.y.",
		".ppppppp..",
		".ppppppp..",
	},
	"lotrzyk": {
		"....kkk...",
		"...kkkkk..",
		"...ksss...",
		"....ss....",
		"...AAAA...",
		"..AAAAA.a.",
		"..AAAAAaa.",
		"..AAAA....",
		"...tt.....",
		"..tt.tt...",
		"..kk.kk...",
		"..kk..kk..",
	},
	"lowczyni": {
		"....hhh...",
		"...hhsss..",
		"...hhss...",
		"....ss..y.",
		"...GGGG.y.",
		"..GGGGG.y.",
		"..GGGGGyy.",
		"..GGGG..y.",
		"..tttt..y.",
		"..tt.tt.y.",
		"..kk.kk...",
		"..kk..kk..",
	},
	"szabrownik": {
		"....AAA...",
		"...AkkkA..",
		"...Akok...",
		"....ss....",
		"...tttt.a.",
		"..ttttt.a.",
		"..tttttaa.",
		"..tttt..a.",
		"..AAAA..a.",
		"..AA.AA...",
		"..kk.kk...",
		"..kk..kk..",
	},
	"pilot": {
		"....www...",
		"...wcccw..",
		"...wccc...",
		"....ss....",
		"...bbbb...",
		"..bbbbb.a.",
		"..bbbbbaa.",
		"..bbbb..a.",
		"..BBBB....",
		"..BB.BB...",
		"..kk.kk...",
		"..kk..kk..",
	},
	"fighter": {
		"....hhh...",
		"...hsss...",
		"...hss....",
		"....ss..a.",
		"...aaaa.aa",
		"..aaaaa.aa",
		"..aaaaaa.a",
		"..aaaa..t.",
		"..tttt..t.",
		"..tt.tt...",
		"..kk.kk...",
		"..kk..kk..",
	},
	"healer": {
		"....www...",
		"...wsss...",
		"...wss....",
		"....ss....",
		"...wwww...",
		"..wwrwww..",
		"..wrrrww..",
		"..wwrwww..",
		"..wwwww...",
		"..wwwww...",
		"..wwwww...",
		".wwwwwww..",
	},
	"mage": {
		".....b....",
		"....bbb...",
		"...bbbbb..",
		"....sss...",
		"....ss..o.",
		"...bbbb.y.",
		"..bbbbbyy.",
		"..bbbbb.y.",
		"..bbbbb.y.",
		"..bbbbb.y.",
		".bbbbbbb..",
		".bbbbbbb..",
	},
	"generic": {
		"....AAA...",
		"...Asss...",
		"...Ass....",
		"....ss....",
		"...aaaa...",
		"..aaaaa...",
		"..aaaaa...",
		"..aaaa....",
		"..tttt....",
		"..tt.tt...",
		"..kk.kk...",
		"..kk..kk..",
	},
}

var figures sync.Map

// figureFor returns the figure of a piece on the heroes' side.
func figureFor(piece game.PieceView) pix.Image {
	name := piece.Avatar
	if piece.Role != "" {
		name = string(piece.Role)
	}

	if _, ok := figureDrawings[name]; !ok {
		name = "generic"
	}

	if cached, ok := figures.Load(name); ok {
		return cached.(pix.Image)
	}

	drawn := parseFigure(figureDrawings[name])
	figures.Store(name, drawn)

	return drawn
}

// parseFigure turns the lines of a drawing into a picture.
func parseFigure(lines []string) pix.Image {
	width := 0
	for _, line := range lines {
		width = max(width, len([]rune(line)))
	}

	img := pix.New(width, len(lines))

	for y, line := range lines {
		for x, char := range []rune(strings.TrimRight(line, " ")) {
			if colour, ok := figurePalette[char]; ok {
				img.Set(x, y, colour)
			}
		}
	}

	return img
}

// roleOf is kept for the compiler: companions' roles name their figures.
var _ = world.RoleFighter
