package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
)

// A blow is shown over a few frames: a flash, a colour and a recoil, with
// the number of points floating above whoever took it.
const (
	effectFrames = 3
	frameTime    = 110 * time.Millisecond
	numberScale  = 2
)

// Colours of the effects, as 256-colour palette indexes.
const (
	colourFlash  uint8 = 231
	colourBlood  uint8 = 196
	colourSpark  uint8 = 226
	colourBlaze  uint8 = 208
	colourLife   uint8 = 46
	colourVenom  uint8 = 135
	colourShadow uint8 = 238
)

// frameMsg moves the running animation on; serial names the animation,
// so that a late tick of an earlier one changes nothing.
type frameMsg struct {
	serial int
}

// collect picks the blows the screen has not shown yet and starts their
// animation. Blows from before the screen opened are not replayed.
func (g *Game) collect() {
	var fresh []game.BlowView
	last := g.shown

	for _, struck := range g.view.Blows {
		if struck.Serial > g.shown {
			fresh = append(fresh, struck)
			last = max(last, struck.Serial)
		}
	}

	if g.shown < 0 {
		g.shown, fresh = last, nil
	}

	if len(fresh) == 0 {
		return
	}

	g.shown, g.effects, g.frame = last, fresh, 0
}

// animating reports that a blow is being shown.
func (g Game) animating() bool {
	return len(g.effects) > 0
}

// nextFrame schedules the next frame of the running animation.
func (g Game) nextFrame() tea.Cmd {
	serial := g.shown

	return tea.Tick(frameTime, func(time.Time) tea.Msg { return frameMsg{serial: serial} })
}

// advance moves to the next frame and reports whether there is another.
func (g *Game) advance(msg frameMsg) bool {
	if msg.serial != g.shown || !g.animating() {
		return false
	}

	g.frame++

	if g.frame >= effectFrames {
		g.effects = nil

		return false
	}

	return true
}

// colourOf is the colour a kind of blow flashes in.
func colourOf(kind game.BlowKind) uint8 {
	switch kind {
	case game.BlowBolt:
		return colourSpark
	case game.BlowBlast:
		return colourBlaze
	case game.BlowHeal, game.BlowRegen:
		return colourLife
	case game.BlowPoison:
		return colourVenom
	}

	return colourBlood
}

// points words the amount of a blow for the floating number.
func points(amount int) string {
	if amount > 0 {
		return fmt.Sprintf("+%d", amount)
	}

	return fmt.Sprint(amount)
}

// paintFoe draws a monster's sprite as the current frame of the blows it
// took: white, then its colour, then back in place with a recoil.
func paintFoe(canvas *pix.Image, sprite pix.Image, x, y int, blows []game.BlowView, frame int) {
	if len(blows) == 0 {
		canvas.Draw(sprite, x, y)

		return
	}

	kind := blows[len(blows)-1].Kind

	switch frame {
	case 0:
		canvas.Draw(sprite.Tint(colourFlash), x, y)
	case 1:
		canvas.Draw(sprite.Tint(colourOf(kind)), x, y)
	default:
		if kind == game.BlowHit {
			x++
		}

		canvas.Draw(sprite, x, y)
	}

	total := 0
	for _, struck := range blows {
		total += struck.Amount
	}

	number := pix.Text(points(total), colourOf(kind), numberScale)
	canvas.Draw(number, x+(sprite.W-number.W)/2, max(0, y-number.H-1))
}

// paintScene adds what a blow does to the whole picture: the bolt that
// hit, the wave of a blast, the red edge of a wound and the numbers of
// the heroes' side, who are not drawn.
func paintScene(canvas *pix.Image, blows []game.BlowView, frame int, foeAt map[int][2]int) {
	// What the heroes' side took is added up by kind and shown in a row.
	totals := map[game.BlowKind]int{}
	var kinds []game.BlowKind

	for _, struck := range blows {
		colour := colourOf(struck.Kind)

		switch struck.Kind {
		case game.BlowBolt:
			if at, ok := foeAt[struck.Foe]; ok && frame < 2 {
				canvas.Bolt(2, 2, at[0], at[1], colour)
			}
		case game.BlowBlast:
			canvas.Ring(canvas.W/2, groundLine-6, 6+frame*9, colour)
		case game.BlowWound, game.BlowPoison, game.BlowHeal:
			if frame < 2 {
				canvas.Frame(colour, 1)
			}

			if _, seen := totals[struck.Kind]; !seen {
				kinds = append(kinds, struck.Kind)
			}

			totals[struck.Kind] += struck.Amount
		}
	}

	x := 3
	for _, kind := range kinds {
		number := pix.Text(points(totals[kind]), colourOf(kind), numberScale)
		canvas.Draw(number, x, 3)
		x += number.W + 4
	}
}

// shaken reports whether this frame of the blows rocks the whole scene.
func shaken(blows []game.BlowView, frame int) bool {
	for _, struck := range blows {
		if struck.Kind == game.BlowWound && frame == 1 {
			return true
		}
	}

	return false
}
