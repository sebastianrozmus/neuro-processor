// Package ui is the terminal interface of the game.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	// MinWidth and MinHeight are the smallest terminal the game draws in.
	MinWidth  = 118
	MinHeight = 36
)

var (
	border      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	box         = border.Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	goldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	playerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	selfStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true)
	foeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	goodStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	manaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	goldBar     = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	badStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	emptyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	// Map markers sit in the middle of a tile on their own background.
	meMarker      = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("226")).Bold(true)
	friendMarker  = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("51")).Bold(true)
	foeMarker     = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160")).Bold(true)
	townMarker    = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	caravanMarker = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	siteMarker    = lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Bold(true)
)

// terrainLook is how a kind of terrain is painted on the map: a few
// three-character textures in one colour on a background.
type terrainLook struct {
	textures []string
	style    lipgloss.Style
}

func look(fg, bg string, textures ...string) terrainLook {
	return terrainLook{textures, lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg))}
}

var terrainLooks = map[world.Terrain]terrainLook{
	"plains":    look("107", "64", " ' ", ", .", " . ", "'  "),
	"forest":    look("34", "22", "♣♠♣", "♠♣♠", "♣♣♠"),
	"mountain":  look("252", "240", "▲^▲", "^▲^", "▲▲^"),
	"swamp":     look("72", "23", "~\"~", "\"~~", "~~\""),
	"desert":    look("137", "179", ". .", " . ", ".. "),
	"ruins":     look("250", "241", "#.#", ".#.", "#:."),
	"cave":      look("97", "236", "o.o", ".o.", "oo."),
	"snow":      look("245", "254", " * ", "*  ", "  *"),
	"wasteland": look("137", "94", ", .", ". ,", " ,."),
	"water":     look("39", "24", "≈~≈", "~≈~", "≈≈~"),
	"street":    look("248", "238", "═╬═", "╬═╬", "═══"),
	"station":   look("146", "60", "[=]", "=[]", "]=["),
}

// panel draws a bordered box of exactly width by height cells. Content that
// does not fit is cut: from the bottom, or from the top when tail is set.
func panel(width, height int, content string, tail bool) string {
	innerWidth, innerHeight := max(1, width-4), max(1, height-2)
	lines := strings.Split(lipgloss.NewStyle().Width(innerWidth).Render(content), "\n")

	if len(lines) > innerHeight {
		if tail {
			lines = lines[len(lines)-innerHeight:]
		} else {
			lines = lines[:innerHeight]
		}
	}

	return box.Width(width - 2).Height(innerHeight).Render(strings.Join(lines, "\n"))
}

// bar draws a gauge of the given width.
func bar(value, limit, width int, full lipgloss.Style) string {
	value = min(max(value, 0), limit)
	filled := (value*width + limit - 1) / max(limit, 1)

	return full.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", width-filled))
}

// healthBar draws a health gauge that reddens as it empties.
func healthBar(value, limit, width int) string {
	return bar(value, limit, width, healthStyle(value, limit))
}

// healthStyle is the colour of a health gauge: it reddens as it empties.
func healthStyle(value, limit int) lipgloss.Style {
	switch {
	case value*4 <= limit:
		return badStyle
	case value*2 <= limit:
		return warnStyle
	}

	return goodStyle
}
