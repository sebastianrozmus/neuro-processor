package ui

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/sebastianrozmus/neuro-processor/internal/gallery"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/inline"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	sceneWidth  = 64
	sceneHeight = 32
	// sceneRows is how many terminal rows a scene takes.
	sceneRows = sceneHeight / 2

	sceneVariants  = gallery.Variants
	spriteVariants = 3
	spriteGap      = 2
	groundLine     = sceneHeight - 2
)

// scenes caches drawn scenes: the same tile in the same state always looks
// the same.
var scenes sync.Map

// scene draws what the player sees where they stand: the place, what was
// built or left there and, in a fight, the monsters they face. While blows
// are being shown, the given frame of them is painted over it.
func scene(view game.View, target int, blows []game.BlowView, frame int, moving bool, inlined bool) string {
	variant := (view.Me.Pos.X*7+view.Me.Pos.Y*13)%sceneVariants + 1

	// A terminal that shows images gets the painting itself, as long as
	// nothing the player must see is drawn over the place: fights, blows,
	// a shrine or chest and things lying about stay block art.
	if inlined && view.Fight == nil && len(blows) == 0 && view.Tile.Site < 0 && !portable(view.Ground) {
		if png, path := painted.ScenePicture(view.World.Spec, view.Tile.Place, variant); len(png) > 0 {
			if picture := inline.Show(path, png, sceneWidth, sceneRows); picture != "" {
				return picture
			}
		}
	}

	key := sceneKey(view, target, moving)
	if painted.Painted(view.World.Spec) {
		key += "|painted"
	}

	if cached, ok := scenes.Load(key); ok && len(blows) == 0 {
		return cached.(string)
	}

	// A world with painted scenes shows its own; until they are painted
	// (and for the built-in worlds) the terrain's embedded picture serves.
	base, ok := painted.Scene(view.World.Spec, view.Tile.Place, variant)
	if !ok {
		base, ok = loadScene(view.Place.Terrain, variant)
	}

	if !ok {
		return ""
	}

	canvas := base.Clone()
	if view.Night {
		canvas = canvas.Dim()
	}

	canvas = compose(canvas, view, target, blows, frame, moving)

	if shaken(blows, frame) {
		canvas = canvas.Shifted(1)
	}

	drawn := canvas.Render()

	if len(blows) == 0 {
		scenes.Store(key, drawn)
	}

	return drawn
}

// compose puts everything that stands in the place onto its picture.
func compose(canvas pix.Image, view game.View, target int, blows []game.BlowView, frame int, moving bool) pix.Image {

	if view.Tile.Town >= 0 {
		stand(&canvas, buildingsFor(view.Place.Terrain), sceneWidth/2)
	}

	if view.Tile.Site >= 0 {
		switch site := view.World.Sites[view.Tile.Site]; {
		case site.Kind == world.SiteShrine:
			stand(&canvas, "shrine", sceneWidth/2)
		case view.Fight == nil:
			stand(&canvas, "chest", sceneWidth/2)
		}
	}

	foeAt := map[int][2]int{}

	switch {
	case view.Fight != nil && view.Fight.Board != nil:
		var handled []game.BlowView
		foeAt, handled = drawField(&canvas, view, target, blows, frame, moving)
		blows = unhandled(blows, handled)
	case view.Fight != nil:
		foeAt = drawMonsters(&canvas, view.Fight.Monsters, target, blows, frame)
	case portable(view.Ground):
		stand(&canvas, "sack", sceneWidth*3/4)
	}

	paintScene(&canvas, blows, frame, foeAt)

	return canvas
}

// portable reports whether anything lying here could be picked up.
func portable(ground []world.ItemSpec) bool {
	for _, item := range ground {
		if !item.Fixed {
			return true
		}
	}

	return false
}

func sceneKey(view game.View, target int, moving bool) string {
	var key strings.Builder

	fmt.Fprintf(&key, "%s|%v|%s|%d|%d|%d|%v|%v", view.World.Name, view.Me.Pos, view.Place.Terrain, view.Tile.Town, view.Tile.Site, target, portable(view.Ground), view.Night)

	if view.Fight != nil {
		for _, foe := range view.Fight.Monsters {
			fmt.Fprintf(&key, "|%s", foe.Name)
		}
	}

	if view.Fight != nil && view.Fight.Board != nil {
		fmt.Fprintf(&key, "|%v|%d", moving, view.Fight.Board.Moves)

		for _, piece := range append(append([]game.PieceView(nil), view.Fight.Board.Pieces...), view.Fight.Board.Fallen...) {
			fmt.Fprintf(&key, "|%s@%d,%d:%v%v%d", piece.Name, piece.X, piece.Y, piece.Current, piece.Mine, piece.HP)
		}
	}

	return key.String()
}

// buildingsFor picks the look of a settlement that fits its surroundings.
func buildingsFor(terrain world.Terrain) string {
	switch terrain {
	case "station", "street", "wasteland", "desert":
		return "outpost"
	}

	return "village"
}

// stand draws a sprite standing on the ground, centred on x.
func stand(canvas *pix.Image, name string, x int) {
	if sprite, ok := pix.Load(name); ok {
		canvas.Draw(sprite, x-sprite.W/2, groundLine-sprite.H)
	}
}

// drawMonsters lines the living monsters up across the scene and points at
// the one the player has picked. It returns where the top of each stands,
// by monster ID, for the effects aimed at them.
func drawMonsters(canvas *pix.Image, foes []game.MonsterView, target int, blows []game.BlowView, frame int) map[int][2]int {
	type placed struct {
		foe    game.MonsterView
		sprite pix.Image
	}

	sprites := make([]placed, 0, len(foes))
	total := -spriteGap

	for _, foe := range foes {
		sprite, ok := loadSprite(foe)
		if !ok {
			continue
		}

		sprites = append(sprites, placed{foe, sprite})
		total += sprite.W + spriteGap
	}

	x := (sceneWidth - total) / 2
	tops := map[int][2]int{}

	for index, drawn := range sprites {
		y := groundLine - drawn.sprite.H
		tops[drawn.foe.ID] = [2]int{x + drawn.sprite.W/2, y}

		var taken []game.BlowView
		for _, struck := range blows {
			if struck.Foe == drawn.foe.ID {
				taken = append(taken, struck)
			}
		}

		paintFoe(canvas, drawn.sprite, x, y, taken, frame)

		if marker, ok := pix.Load("marker"); ok && index == target && len(sprites) > 1 && len(taken) == 0 {
			canvas.Draw(marker, x+(drawn.sprite.W-marker.W)/2, max(0, y-marker.H-1))
		}

		x += drawn.sprite.W + spriteGap
	}

	return tops
}

// loadScene returns one of the pictures of a kind of terrain.
func loadScene(terrain world.Terrain, variant int) (pix.Image, bool) {
	return pix.Load(fmt.Sprintf("%s%d", terrain, variant))
}

// loadSprite returns the picture of a monster: its shape in one of a few
// colours chosen by its name, larger for the boss.
func loadSprite(foe game.MonsterView) (pix.Image, bool) {
	hash := fnv.New32a()
	hash.Write([]byte(foe.Name))

	name := fmt.Sprintf("%s%d", foe.Shape, hash.Sum32()%spriteVariants+1)
	if foe.Boss {
		name += "-boss"
	}

	return pix.Load(name)
}

// The battlefield of a D&D fight is drawn in the scene like a tabletop:
// the monsters' sprites and the heroes' figures stand on the cells of
// the field, the back row higher up, with their health under their feet.
const (
	fieldCell  = sceneWidth / game.BoardWidth
	rowDepth   = 5
	barWidthPx = 6

	colourGrid   = 240
	colourReach  = 226
	colourAim    = 220
	colourFallen = 238
	colourHurt   = 160
	colourBolt   = 226
	colourBlast  = 208
)

// spot is where a piece stands in the picture: the bottom centre of its
// cell and the sprite drawn there.
type spot struct {
	x, y   int
	sprite pix.Image
	piece  game.PieceView
}

// drawField draws the field and everybody on it, with the blows of the
// current frame acted out: the striker lunges, the struck flashes, bolts
// fly and blasts burst. It returns where each monster's top is, for the
// effects drawn over the whole scene, and the blows it acted out.
func drawField(canvas *pix.Image, view game.View, target int, blows []game.BlowView, frame int, moving bool) (map[int][2]int, []game.BlowView) {
	board := view.Fight.Board

	drawGround(canvas, board, moving)

	for _, fallen := range board.Fallen {
		sprite := spriteFor(fallen).Tint(colourFallen)
		x, y := footOf(fallen)
		canvas.Draw(sprite, x-sprite.W/2, y-sprite.H+2)
	}

	spots := make([]spot, 0, len(board.Pieces))
	byName := map[string]spot{}
	byID := map[int]spot{}

	for _, piece := range board.Pieces {
		x, y := footOf(piece)
		placed := spot{x: x, y: y, sprite: spriteFor(piece), piece: piece}
		spots = append(spots, placed)
		byName[piece.Name] = placed

		if piece.Foe {
			byID[piece.ID] = placed
		}
	}

	aimed := -1
	if target >= 0 && target < len(view.Fight.Monsters) {
		aimed = view.Fight.Monsters[target].ID
	}

	var handled []game.BlowView
	tops := map[int][2]int{}

	for _, placed := range spots {
		taken, dealt := blowsOf(placed.piece, blows)
		handled = append(handled, taken...)

		x, y := placed.x-placed.sprite.W/2, placed.y-placed.sprite.H
		x += lunge(placed, dealt, byName, byID, frame)

		paintFoe(canvas, placed.sprite, x, y, taken, frame)
		drawBar(canvas, placed.x, placed.y+1, placed.piece.HP, placed.piece.MaxHP)

		if placed.piece.Foe {
			tops[placed.piece.ID] = [2]int{placed.x, y}
		}

		if marker, ok := pix.Load("marker"); ok {
			switch {
			case placed.piece.Current:
				canvas.Draw(marker, placed.x-marker.W/2, max(0, y-marker.H-1))
			case placed.piece.Foe && placed.piece.ID == aimed && len(view.Fight.Monsters) > 1:
				canvas.Draw(marker.Tint(colourAim), placed.x-marker.W/2, max(0, y-marker.H-1))
			}
		}
	}

	// Bolts fly from the caster to the struck; blasts burst around them.
	for _, struck := range blows {
		victim, ok := byID[struck.Foe]
		if !ok {
			victim, ok = byName[struck.To]
		}

		if !ok {
			continue
		}

		switch struck.Kind {
		case game.BlowBolt:
			if caster, known := byName[struck.From]; known {
				drawBolt(canvas, caster, victim, frame)
			}
		case game.BlowBlast:
			canvas.Ring(victim.x, victim.y-victim.sprite.H/2, 3+frame*3, colourBlast)
		}
	}

	return tops, handled
}

// footOf is where a piece stands: the bottom centre of its cell, the
// back rows higher up the picture.
func footOf(piece game.PieceView) (int, int) {
	return piece.X*fieldCell + fieldCell/2, groundLine - (game.BoardHeight-1-piece.Y)*rowDepth
}

// spriteFor picks a monster's sprite, shrunk to the field, or a hero's
// figure.
func spriteFor(piece game.PieceView) pix.Image {
	if !piece.Foe {
		return figureFor(piece)
	}

	sprite, ok := loadSprite(game.MonsterView{Name: piece.Name, Shape: piece.Shape, Boss: piece.Boss})
	if !ok {
		return figureFor(game.PieceView{})
	}

	factor := 2
	if piece.Boss {
		factor = 1 + sprite.W/20
	}

	return sprite.Shrink(factor)
}

// drawGround marks the rows and cells of the field faintly and, while the
// player walks, the cells still within reach.
func drawGround(canvas *pix.Image, board *game.BoardView, moving bool) {
	for row := range game.BoardHeight {
		y := groundLine - (game.BoardHeight-1-row)*rowDepth

		for x := 0; x < game.BoardWidth*fieldCell; x += 2 {
			canvas.Set(x, y, colourGrid)
		}
	}

	if !moving {
		return
	}

	for _, piece := range board.Pieces {
		if !piece.Current || !piece.Mine {
			continue
		}

		for y := range game.BoardHeight {
			for x := range game.BoardWidth {
				if max(x-piece.X, piece.X-x)+max(y-piece.Y, piece.Y-y) <= board.Moves {
					cx, cy := footOf(game.PieceView{X: x, Y: y})
					canvas.Line(cx-1, cy, cx+1, cy, colourReach)
				}
			}
		}
	}
}

// drawBar draws the health under a piece.
func drawBar(canvas *pix.Image, cx, y, hp, maxHP int) {
	if maxHP <= 0 {
		return
	}

	full := max(0, min(barWidthPx, barWidthPx*hp/maxHP))
	left := cx - barWidthPx/2

	for x := range barWidthPx {
		colour := uint8(colourHurt)
		if x < full {
			colour = colourLife
		}

		canvas.Set(left+x, y, colour)
	}
}

// blowsOf sorts the blows into those a piece took and those it dealt.
func blowsOf(piece game.PieceView, blows []game.BlowView) (taken, dealt []game.BlowView) {
	for _, struck := range blows {
		switch {
		case piece.Foe && struck.Foe == piece.ID:
			taken = append(taken, struck)
		case !piece.Foe && struck.Foe < 0 && struck.To == piece.Name:
			taken = append(taken, struck)
		}

		if struck.From != "" && struck.From == piece.Name && (struck.Kind == game.BlowHit || struck.Kind == game.BlowWound) {
			dealt = append(dealt, struck)
		}
	}

	return taken, dealt
}

// lunge is how far a striker leans towards whoever it strikes this frame.
func lunge(striker spot, dealt []game.BlowView, byName map[string]spot, byID map[int]spot, frame int) int {
	if len(dealt) == 0 || frame > 1 {
		return 0
	}

	victim, ok := byID[dealt[0].Foe]
	if !ok {
		victim, ok = byName[dealt[0].To]
	}

	if !ok || victim.x == striker.x {
		return 0
	}

	reach := 3 - 2*frame
	if victim.x < striker.x {
		return -reach
	}

	return reach
}

// drawBolt draws a bolt on its way from the caster to the struck.
func drawBolt(canvas *pix.Image, caster, victim spot, frame int) {
	x0, y0 := caster.x, caster.y-caster.sprite.H/2
	x1, y1 := victim.x, victim.y-victim.sprite.H/2
	progress := frame + 1
	px, py := x0+(x1-x0)*progress/effectFrames, y0+(y1-y0)*progress/effectFrames

	canvas.Line(px-2, py, px+2, py, colourBolt)
	canvas.Set(px, py-1, colourFlash)
	canvas.Set(px, py+1, colourFlash)
}

// unhandled returns the blows nobody on the field acted out.
func unhandled(blows, handled []game.BlowView) []game.BlowView {
	done := map[int]bool{}
	for _, struck := range handled {
		done[struck.Serial] = true
	}

	var rest []game.BlowView
	for _, struck := range blows {
		if !done[struck.Serial] {
			rest = append(rest, struck)
		}
	}

	return rest
}
