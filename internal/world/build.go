package world

import (
	"hash/fnv"
	"math/rand"
)

const (
	// Width and Height are the size of every world map, in tiles.
	Width  = 15
	Height = 9

	// Tiles closer to the first settlement than these shares of the map
	// radius are of danger 1 and 2; everything beyond is danger 3.
	innerRing = 0.5
	outerRing = 0.84

	// Places of one danger level form blocks of this size.
	regionWidth  = 3
	regionHeight = 2

	townSpacing  = 5
	siteSpacing  = 3
	treasurePack = 2
)

// spawnPercent is the chance that a tile of a given danger holds monsters.
var spawnPercent = [MaxTier + 1]int{0, 50, 60, 70}

// Point is a position on the map.
type Point struct {
	X int
	Y int
}

// Tile is one square of the map.
type Tile struct {
	// Place indexes Spec.Places.
	Place int
	// Monsters index Spec.Monsters, one entry per monster on the tile.
	Monsters []int
	Boss     bool
	// Town indexes World.Towns, or is -1 outside settlements.
	Town int
	// Site indexes Spec.Sites, or is -1 on ordinary tiles.
	Site int
	// Loot indexes Spec.Items lying on the tile from the start.
	Loot []int
}

// World is a playable map built from a spec.
type World struct {
	Spec
	Tiles []Tile
	// Towns are the settlements; the first is where heroes start.
	Towns []Point
	Lair  Point
}

// Build lays out the map for a spec. The layout depends only on the world's
// name, so the same spec always gives the same map.
func Build(spec Spec) (*World, error) {
	if err := spec.Normalize(); err != nil {
		return nil, err
	}

	if spec.Mode == ModeAdventure {
		return buildAdventure(spec)
	}

	hash := fnv.New64a()
	hash.Write([]byte(spec.Name))
	rng := rand.New(rand.NewSource(int64(hash.Sum64())))

	world := &World{
		Spec:  spec,
		Tiles: make([]Tile, Width*Height),
		Towns: []Point{{Width / 2, Height / 2}},
	}

	corners := []Point{{0, 0}, {Width - 1, 0}, {0, Height - 1}, {Width - 1, Height - 1}}
	world.Lair = corners[rng.Intn(len(corners))]

	townPlaces := spec.TownPlaces()
	for len(world.Towns) < len(townPlaces) {
		spot, ok := world.pick(rng, 2, townSpacing, world.Towns)
		if !ok {
			break
		}

		world.Towns = append(world.Towns, spot)
	}

	taken := append([]Point{world.Lair}, world.Towns...)
	sites := map[Point]int{}

	for index := range spec.Sites {
		if spot, ok := world.pick(rng, 2, siteSpacing, taken); ok {
			sites[spot] = index
			taken = append(taken, spot)
		}
	}

	regionSeed := rng.Intn(1 << 16)

	for y := range Height {
		for x := range Width {
			point := Point{x, y}
			danger := world.danger(point)
			tile := &world.Tiles[world.Index(point)]
			tile.Place = world.place(danger, regionSeed+x/regionWidth*31+y/regionHeight*17)
			tile.Town, tile.Site = -1, -1

			site, isSite := sites[point]

			switch {
			case world.townAt(point) >= 0:
				tile.Town = world.townAt(point)
				tile.Place = townPlaces[tile.Town]
			case point == world.Lair:
				tile.Boss = true
			case isSite:
				tile.Site = site

				if spec.Sites[site].Kind == SiteTreasure {
					tile.Monsters = world.pack(rng, danger, treasurePack)
					tile.Loot = world.treasure(rng, danger)
				}
			case rng.Intn(100) < spawnPercent[danger]:
				size := 1
				if !world.near(point) && rng.Intn(100) < 25*danger {
					size = 2
				}

				tile.Monsters = world.pack(rng, danger, size)
			}
		}
	}

	return world, nil
}

// Start is where heroes enter the world.
func (w *World) Start() Point {
	return w.Towns[0]
}

// In reports whether the point lies on the map.
func (w *World) In(p Point) bool {
	return p.X >= 0 && p.X < Width && p.Y >= 0 && p.Y < Height
}

// Index returns the position of a point in Tiles.
func (w *World) Index(p Point) int {
	return p.Y*Width + p.X
}

// At returns the tile at a point.
func (w *World) At(p Point) Tile {
	return w.Tiles[w.Index(p)]
}

// PlaceAt returns the kind of place found at a point; the void between
// the rooms of an adventure is an empty place.
func (w *World) PlaceAt(p Point) PlaceSpec {
	if place := w.At(p).Place; place >= 0 && place < len(w.Places) {
		return w.Places[place]
	}

	return PlaceSpec{}
}

func (w *World) danger(p Point) int {
	start := w.Start()
	dx := float64(abs(p.X-start.X)) / float64(Width/2)
	dy := float64(abs(p.Y-start.Y)) / float64(Height/2)

	switch distance := max(dx, dy); {
	case distance == 0:
		return 0
	case distance <= innerRing:
		return 1
	case distance <= outerRing:
		return 2
	}

	return MaxTier
}

func (w *World) townAt(p Point) int {
	for index, town := range w.Towns {
		if town == p {
			return index
		}
	}

	return -1
}

// near reports whether a point touches a settlement.
func (w *World) near(p Point) bool {
	for _, town := range w.Towns {
		if abs(p.X-town.X) <= 1 && abs(p.Y-town.Y) <= 1 {
			return true
		}
	}

	return false
}

// pick chooses a free tile of at least the given danger that keeps its
// distance from the points already taken.
func (w *World) pick(rng *rand.Rand, minDanger, spacing int, taken []Point) (Point, bool) {
	var free []Point

	for y := range Height {
		for x := range Width {
			point := Point{x, y}
			if w.danger(point) < minDanger || point == w.Lair {
				continue
			}

			clear := true
			for _, other := range taken {
				clear = clear && abs(point.X-other.X)+abs(point.Y-other.Y) >= spacing
			}

			if clear {
				free = append(free, point)
			}
		}
	}

	if len(free) == 0 {
		return Point{}, false
	}

	return free[rng.Intn(len(free))], true
}

// place picks a wild place of the wanted danger, or of the closest one.
func (w *World) place(danger, variant int) int {
	for spread := 0; spread <= MaxTier; spread++ {
		var matches []int

		for index, place := range w.Places {
			if place.Danger > 0 && abs(place.Danger-danger) == spread {
				matches = append(matches, index)
			}
		}

		if len(matches) > 0 {
			return matches[variant%len(matches)]
		}
	}

	return 0
}

// pack rolls a group of monsters fit for a tile of the given danger.
func (w *World) pack(rng *rand.Rand, danger, size int) []int {
	var kinds []int

	for spread := 0; spread < MaxTier && len(kinds) == 0; spread++ {
		for index, monster := range w.Monsters {
			if abs(monster.Tier-danger) == spread {
				kinds = append(kinds, index)
			}
		}
	}

	pack := make([]int, size)
	for i := range pack {
		pack[i] = kinds[rng.Intn(len(kinds))]
	}

	return pack
}

// treasure picks what a guarded cache holds: something magical when the
// world has it, and a potion.
func (w *World) treasure(rng *rand.Rand, danger int) []int {
	var magic, potions, rest []int

	for index, item := range w.Items {
		switch {
		case item.Tier > danger:
		case item.Kind == KindSpell || item.Kind == KindCharm:
			magic = append(magic, index)
		case item.Kind == KindPotion:
			potions = append(potions, index)
		default:
			rest = append(rest, index)
		}
	}

	prize := magic
	if len(prize) == 0 {
		prize = rest
	}

	var loot []int

	for _, group := range [][]int{prize, potions} {
		if len(group) > 0 {
			loot = append(loot, group[rng.Intn(len(group))])
		}
	}

	return loot
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

// Pack rolls a group of monsters fit for a tile of the given danger.
func (w *World) Pack(rng *rand.Rand, danger, size int) []int {
	if len(w.Monsters) == 0 {
		return nil
	}

	return w.pack(rng, danger, size)
}

// Near reports whether a point touches a settlement.
func (w *World) Near(p Point) bool {
	return w.near(p)
}
