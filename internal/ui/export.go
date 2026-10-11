package ui

import (
	"fmt"
	"hash/fnv"

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

// SpriteName names the embedded drawing a kind of monster is shown with.
func SpriteName(name string, shape world.Shape, boss bool) string {
	hash := fnv.New32a()
	hash.Write([]byte(name))

	drawn := fmt.Sprintf("%s%d", shape, hash.Sum32()%spriteVariants+1)
	if boss {
		drawn += "-boss"
	}

	return drawn
}

// IconName names an item's icon: its shape and the colour of its effect.
func IconName(item world.ItemSpec) string {
	return shapeOf(item.Kind, item.Name) + " · " + string(item.Effect)
}

// FigureName names the figure a role or avatar is drawn with.
func FigureName(avatar string, role world.Role) string {
	name := avatar
	if role != "" {
		name = string(role)
	}

	if _, ok := figureDrawings[name]; !ok {
		return "generic"
	}

	return name
}
