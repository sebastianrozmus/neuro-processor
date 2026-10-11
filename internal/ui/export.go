package ui

import (
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The pictures the screens draw are also wanted outside the terminal: the
// browser inspector shows the same icons, sprites and figures.

// ItemIcon is the icon of an item, as the bag and the shop draw it.
func ItemIcon(item world.ItemSpec) pix.Image {
	return itemIcon(item.Kind, item.Effect, item.Tier, item.Name)
}

// MonsterSprite is the sprite of a kind of monster, at full size.
func MonsterSprite(name string, shape world.Shape, boss bool) (pix.Image, bool) {
	return loadSprite(game.MonsterView{Name: name, Shape: shape, Boss: boss})
}

// Figure is the figure of a hero (by avatar) or a companion (by role).
func Figure(avatar string, role world.Role) pix.Image {
	return figureFor(game.PieceView{Avatar: avatar, Role: role})
}
