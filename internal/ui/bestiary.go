package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
)

const bookListWidth = 30

// bookKey moves through the bestiary. Esc and F4 close it.
func (g *Game) bookKey(msg tea.KeyMsg) {
	pages := g.view.Me.Bestiary

	switch msg.Type {
	case tea.KeyUp:
		g.beast = max(0, g.beast-1)
	case tea.KeyDown:
		g.beast = min(g.beast+1, max(0, len(pages)-1))
	case tea.KeyEsc, tea.KeyF4, tea.KeyTab:
		g.book = false
	}
}

// bookView draws the bestiary: the kinds of monster of the world down the
// left, and the page of the chosen one, with its picture, on the right.
// Kinds not met yet show only a shadow.
func (g Game) bookView(width int) string {
	pages := g.view.Me.Bestiary
	known := 0

	for _, page := range pages {
		if page.Known {
			known++
		}
	}

	list := []string{titleStyle.Render("Bestiariusz") + labelStyle.Render(fmt.Sprintf("  poznane %d/%d", known, len(pages))), ""}

	for index, page := range pages {
		pointer := "  "
		if index == g.beast {
			pointer = selfStyle.Render("> ")
		}

		name := strings.Repeat("?", 3)
		if page.Known {
			name = page.Name
		}

		if page.Boss {
			name = foeStyle.Render(name)
		}

		tally := ""
		if page.Slain > 0 {
			tally = labelStyle.Render(fmt.Sprintf(" ×%d", page.Slain))
		}

		line := pointer + tier(page.Tier) + " " + name + tally
		list = append(list, lipgloss.NewStyle().MaxWidth(bookListWidth).Render(line))
	}

	left := lipgloss.NewStyle().Width(bookListWidth).Render(strings.Join(list, "\n"))

	if len(pages) == 0 {
		return left
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, g.pageView(pages[min(g.beast, len(pages)-1)], width-bookListWidth))
}

// tier shows a monster's tier as filled dots.
func tier(level int) string {
	return goldBar.Render(strings.Repeat("●", level)) + emptyStyle.Render(strings.Repeat("●", 3-level))
}

// pageView draws one page of the bestiary.
func (g Game) pageView(page game.BeastView, width int) string {
	sprite, ok := loadSprite(game.MonsterView{Name: page.Name, Shape: page.Shape, Boss: page.Boss})

	var lines []string

	switch {
	case !page.Known:
		lines = []string{foeStyle.Render("Nieznany"), dimStyle.Render("Spotkaj go w walce, a karta się wypełni.")}
	default:
		lines = []string{
			foeStyle.Render(page.Name) + "  " + tier(page.Tier),
			dimStyle.Render(page.Description),
			"",
			g.beastLine(page),
			warnStyle.Render(page.Trait.Name+":") + " " + page.Trait.Text,
		}

		if page.Slain > 0 {
			lines = append(lines, labelStyle.Render(fmt.Sprintf("Pokonane w twoich walkach: %d.", page.Slain)))
		} else {
			lines = append(lines, labelStyle.Render("Jeszcze żadnego nie pokonano."))
		}
	}

	text := lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))

	if !ok {
		return text
	}

	if !page.Known {
		sprite = sprite.Tint(colourShadow)
	}

	canvas := pix.New(sprite.W+4, sprite.H+2)
	canvas.Draw(sprite, 2, 1)

	return text + "\n\n" + canvas.Render()
}

// beastLine is the figures of a page; the D&D mode shows the armour class.
func (g Game) beastLine(page game.BeastView) string {
	line := fmt.Sprintf("%s %d   %s %s   %s %d", labelStyle.Render("Zdrowie"), page.HP, labelStyle.Render("Cios"), page.Damage(), labelStyle.Render("Szybkość"), page.Speed)
	if g.view.DnD() {
		line += fmt.Sprintf("   %s %d", labelStyle.Render("KP"), page.AC)
	}

	return line
}
