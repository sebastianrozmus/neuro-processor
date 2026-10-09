package ui

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/assets"
	"github.com/sebastianrozmus/neuro-processor/internal/gallery"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// artCell matches one coloured character of an embedded picture.
var artCell = regexp.MustCompile(`\x1b\[[0-9;]*m[^\x1b]\x1b\[0m`)

var backdrops sync.Map

// painted is the gallery of pictures painted for generated worlds; nil
// when the game has none, in which case every world uses the embedded art.
var painted *gallery.Gallery

// UseGallery points the screens at the pictures painted for generated
// worlds: menu backdrops, scenes and keepers' portraits.
func UseGallery(g *gallery.Gallery) {
	painted = g
}

// backdrop is a picture cut into character cells, so that something can be
// drawn over a part of it.
type backdrop [][]string

func loadBackdrop(name string) backdrop {
	if cached, ok := backdrops.Load(name); ok {
		return cached.(backdrop)
	}

	art, _ := assets.Art(name)
	cells := cutBackdrop(art)
	backdrops.Store(name, cells)

	return cells
}

// cutBackdrop splits rendered art into its character cells.
func cutBackdrop(art string) backdrop {
	var cells backdrop
	for _, line := range strings.Split(art, "\n") {
		cells = append(cells, artCell.FindAllString(line, -1))
	}

	return cells
}

// backdropFor picks the picture behind the menu for a world: a painted
// one when the world has them, else embedded art that fits its look.
func backdropFor(spec world.Spec, slide int) backdrop {
	key := fmt.Sprintf("painted/%s/%d", world.Slug(spec.Name), slide%gallery.Backdrops)
	if cached, ok := backdrops.Load(key); ok {
		return cached.(backdrop)
	}

	if art := painted.Backdrop(spec, slide%gallery.Backdrops); art != "" {
		cells := cutBackdrop(art)
		backdrops.Store(key, cells)

		return cells
	}

	pictures := backdropsFor(spec)

	return loadBackdrop(pictures[slide%len(pictures)])
}

// The embedded art comes in three looks; a world gets the one that fits
// the terrain of its first settlement.
const (
	lookFantasy = "fantasy"
	lookPostapo = "postapo"
	lookScifi   = "scifi"
)

var lookBackdrops = map[string][]string{
	lookFantasy: {"forest1", "dragon1", "forest2", "dragon2", "forest3", "dragon3"},
	lookPostapo: {"wasteland1", "wasteland2"},
	lookScifi:   {"station1", "station2"},
}

// lookOf picks the look of the art used for a world.
func lookOf(spec world.Spec) string {
	terrain := world.Terrains[0]
	if towns := spec.TownPlaces(); len(towns) > 0 {
		terrain = spec.Places[towns[0]].Terrain
	}

	switch terrain {
	case "station", "street":
		return lookScifi
	case "wasteland", "desert", "ruins":
		return lookPostapo
	}

	return lookFantasy
}

// backdropsFor lists the pictures to show behind a world in the menu: its
// own, or ones that fit its look.
func backdropsFor(spec world.Spec) []string {
	if len(spec.Backdrops) > 0 {
		return spec.Backdrops
	}

	return lookBackdrops[lookOf(spec)]
}

// keeperPortrait returns the portrait of the keeper of a kind of shop in a
// world: the painted one when there is one, else the one of the world's
// look, which every look has for each kind.
func keeperPortrait(spec world.Spec, kind world.ShopKind) string {
	if portrait := painted.Keeper(spec, kind); portrait != "" {
		return portrait
	}

	portrait, _ := assets.Art("keepers/" + lookOf(spec) + "-" + string(kind))

	return portrait
}

// under fills a screen of the given size with the picture, centred, and
// draws the block of text over its middle.
func (b backdrop) under(block string, width, height int) string {
	lines := strings.Split(block, "\n")
	blockWidth := lipgloss.Width(block)
	left, top := (width-blockWidth)/2, (height-len(lines))/2

	offsetY := (len(b) - height) / 2
	rows := make([]string, 0, height)

	for y := range height {
		var row strings.Builder

		var cells []string
		if source := y + offsetY; source >= 0 && source < len(b) {
			cells = b[source]
		}

		offsetX := (len(cells) - width) / 2

		for x := 0; x < width; x++ {
			if line := y - top; line >= 0 && line < len(lines) && x == left {
				row.WriteString(lines[line])
				row.WriteString(strings.Repeat(" ", blockWidth-lipgloss.Width(lines[line])))
				x += blockWidth - 1

				continue
			}

			if source := x + offsetX; source >= 0 && source < len(cells) {
				row.WriteString(cells[source])
			} else {
				row.WriteByte(' ')
			}
		}

		rows = append(rows, row.String())
	}

	return strings.Join(rows, "\n")
}
