package game

import (
	"fmt"
	"sort"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The D&D mode: the same world fought by the rules of the tabletop game.
// A d20 against an armour class decides whether a blow lands, dice decide
// what it does, saving throws soften poison and blasts, and every fight is
// played on a grid where reach, movement and attacks of opportunity
// matter.

const (
	// BoardWidth and BoardHeight are the cells of a battlefield: the
	// heroes' side starts on the left, the monsters on the right, and the
	// whole field is drawn in the scene with everybody on it.
	BoardWidth  = 8
	BoardHeight = 3

	// boltRange is how far a bolt reaches, in cells; blastRadius how far
	// around its target a blast burns.
	boltRange   = 5
	blastRadius = 1
	// heroCells is how far a hero walks in a turn.
	heroCells = 4
)

// roleCells is how far companions of each role walk in a turn.
var roleCells = map[world.Role]int{world.RoleFighter: 3, world.RoleHealer: 2, world.RoleMage: 4}

// shapeCells is how far monsters of each shape walk in a turn.
var shapeCells = map[world.Shape]int{
	"beast": 4, "spirit": 4, "serpent": 3, "insect": 3, "humanoid": 3, "machine": 2, "undead": 1, "blob": 1,
}

// board is where a fight of the D&D mode is played: who stands in which
// cell, and the initiative everybody rolled.
type board struct {
	cells      map[any]world.Point
	initiative map[any]int
	// moves is how many cells the actor whose turn it is may still walk.
	moves int
}

func (r *Realm) dnd() bool {
	return r.world.Mode == world.ModeDnD
}

// d rolls one die.
func (r *Realm) d(sides int) int {
	return 1 + r.rng.Intn(sides)
}

// proficiency is the bonus of a hero of the given level.
func proficiency(level int) int {
	return 2 + (level-1)/4
}

// armourClass is what an attack roll has to reach: 10, nimbleness and
// whatever is worn.
func armourClass(who unit) int {
	return 10 + Modifier(who.scores()[Dexterity]) + who.guard()
}

// monsterClass is the armour class of a monster.
func monsterClass(foe *monster) int {
	class := 10 + 2*foe.tier

	switch foe.shape {
	case "machine":
		class += 2
	case "humanoid":
		class++
	case "blob":
		class -= 2
	}

	if foe.boss {
		class += 2
	}

	return class
}

// levelOf is the level of a unit for its proficiency.
func levelOf(who unit) int {
	switch who := who.(type) {
	case *player:
		return who.level
	case *companion:
		return who.level
	}

	return 1
}

// weaponDie is the die a unit's weapon rolls.
func weaponDie(who unit) int {
	if weapon := who.arms(); weapon != nil {
		return [world.MaxTier]int{6, 8, 10}[min(max(weapon.Tier, 1), world.MaxTier)-1]
	}

	if member, ok := who.(*companion); ok {
		return map[world.Role]int{world.RoleFighter: 8, world.RoleHealer: 4, world.RoleMage: 6}[member.spec().Role]
	}

	return 4
}

// monsterDice is what a monster's blow rolls: the dice and the bonus.
func monsterDice(foe *monster) (count, sides, bonus int) {
	if foe.boss {
		return 2, 8, 6
	}

	dice := [world.MaxTier][3]int{{1, 6, 2}, {1, 8, 3}, {2, 6, 4}}[foe.tier-1]

	return dice[0], dice[1], dice[2]
}

// attackRoll rolls a d20 against a class and reports whether it hits and
// whether it is a critical hit.
func (r *Realm) attackRoll(bonus, class int) (hit, crit bool, roll int) {
	roll = r.d(20)

	switch roll {
	case 1:
		return false, false, roll
	case 20:
		return true, true, roll
	}

	return roll+bonus >= class, false, roll
}

// save rolls a saving throw against a difficulty.
func (r *Realm) save(bonus, difficulty int) bool {
	return r.d(20)+bonus >= difficulty
}

// cellsOf is how far somebody walks in a turn.
func (r *Realm) cellsOf(who any) int {
	switch who := who.(type) {
	case *player:
		return heroCells
	case *companion:
		return roleCells[who.spec().Role]
	case *monster:
		return shapeCells[who.shape]
	}

	return 1
}

// seat puts everybody of a new fight on the board: the heroes' side on
// the left, the monsters on the right.
func (r *Realm) seat(current *fight) {
	if current.board == nil {
		current.board = &board{cells: map[any]world.Point{}, initiative: map[any]int{}}
	}

	for _, name := range current.members {
		for index, who := range r.units(r.players[name]) {
			if _, seated := current.board.cells[who]; !seated {
				column := 1
				if index > 0 {
					column = 0
				}

				current.board.cells[who] = r.freeCell(current, column)
				current.board.initiative[who] = r.d(20) + Modifier(who.scores()[Dexterity])
			}
		}
	}

	for _, foe := range r.foes(current.pos) {
		if _, seated := current.board.cells[foe]; !seated {
			current.board.cells[foe] = r.freeCell(current, BoardWidth-2)
			current.board.initiative[foe] = r.d(20) + foe.pace() - 5
		}
	}
}

// freeCell finds an empty cell in or next to a column, the middle rows first.
func (r *Realm) freeCell(current *fight, column int) world.Point {
	for spread := 0; spread < BoardWidth; spread++ {
		for _, x := range []int{column - spread, column + spread} {
			for _, y := range []int{1, 0, 2} {
				at := world.Point{X: x, Y: y}
				if x >= 0 && x < BoardWidth && !r.occupied(current, at) {
					return at
				}
			}
		}
	}

	return world.Point{}
}

func (r *Realm) occupied(current *fight, at world.Point) bool {
	for who, cell := range current.board.cells {
		if cell == at && r.standingOnBoard(current, who) {
			return true
		}
	}

	return false
}

// standingOnBoard reports whether somebody still takes part in the fight.
func (r *Realm) standingOnBoard(current *fight, who any) bool {
	switch who := who.(type) {
	case *monster:
		return who.hp > 0
	case unit:
		return who.health() > 0 && current.has(who.leader().name)
	}

	return false
}

// unseat takes somebody off the board.
func (r *Realm) unseat(current *fight, who any) {
	if current.board != nil {
		delete(current.board.cells, who)
	}
}

// distance is the number of cells between two points, counting diagonals
// as one, as the tabletop game does.
func distance(a, b world.Point) int {
	return max(abs(a.X-b.X), abs(a.Y-b.Y))
}

// adjacent reports whether two combatants stand next to each other.
func (r *Realm) adjacent(current *fight, a, b any) bool {
	at, ok := current.board.cells[a]
	to, ok2 := current.board.cells[b]

	return ok && ok2 && distance(at, to) <= 1
}

// cellOf is where somebody stands.
func (r *Realm) cellOf(current *fight, who any) world.Point {
	return current.board.cells[who]
}

// Step walks the unit whose turn it is one cell across the board, if it
// has movement left. Leaving a monster's reach costs an attack of
// opportunity from it.
func (r *Realm) Step(name string, dx, dy int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	leader := r.active(name)
	if leader == nil {
		return
	}

	current, who := r.turn(leader)
	if current == nil {
		return
	}

	r.step(current, who, world.Point{X: dx, Y: dy})
}

// step moves a combatant one cell and reports whether it moved.
func (r *Realm) step(current *fight, who any, by world.Point) bool {
	if current.board == nil || abs(by.X)+abs(by.Y) != 1 {
		return false
	}

	from := current.board.cells[who]
	to := world.Point{X: from.X + by.X, Y: from.Y + by.Y}
	mover, isUnit := who.(unit)

	switch {
	case current.board.moves <= 0:
		if isUnit {
			r.tell(mover.leader(), mover.label()+" nie ma już ruchu w tej turze.")
		}

		return false
	case to.X < 0 || to.X >= BoardWidth || to.Y < 0 || to.Y >= BoardHeight:
		if isUnit {
			r.tell(mover.leader(), "Koniec pola bitwy.")
		}

		return false
	case r.occupied(current, to):
		if isUnit {
			r.tell(mover.leader(), "To pole jest zajęte.")
		}

		return false
	}

	// Attacks of opportunity: every enemy whose reach is left gets a blow.
	if isUnit {
		for _, foe := range r.foes(current.pos) {
			if distance(from, r.cellOf(current, foe)) <= 1 && distance(to, r.cellOf(current, foe)) > 1 {
				r.tellFight(current, fmt.Sprintf("%s wyprowadza atak okazyjny na odchodzącego: %s.", foe.name, mover.label()))
				r.maul(current, foe, mover, 1)

				if mover.health() <= 0 {
					return false
				}
			}
		}
	}

	current.board.cells[who] = to
	current.board.moves--

	return true
}

// approach walks a combatant towards a target as far as its movement
// allows, stopping when within reach.
func (r *Realm) approach(current *fight, who any, target any) {
	for current.board.moves > 0 && !r.adjacent(current, who, target) {
		from, to := r.cellOf(current, who), r.cellOf(current, target)
		best, moved := distance(from, to), false

		for _, by := range []world.Point{north, south, east, west} {
			next := world.Point{X: from.X + by.X, Y: from.Y + by.Y}
			if next.X < 0 || next.X >= BoardWidth || next.Y < 0 || next.Y >= BoardHeight || r.occupied(current, next) {
				continue
			}

			if d := distance(next, to); d < best {
				best, moved = d, true
				current.board.cells[who] = next
				current.board.moves--

				break
			}
		}

		if !moved {
			return
		}
	}
}

// nearestFoe is the living monster closest to a unit.
func (r *Realm) nearestFoe(current *fight, who any) *monster {
	var nearest *monster
	best := BoardWidth + BoardHeight

	for _, foe := range r.foes(current.pos) {
		if d := distance(r.cellOf(current, who), r.cellOf(current, foe)); nearest == nil || d < best {
			nearest, best = foe, d
		}
	}

	return nearest
}

// nearestUnit is the hero or companion closest to a monster.
func (r *Realm) nearestUnit(current *fight, foe *monster) unit {
	var nearest unit
	best := BoardWidth + BoardHeight

	for _, who := range r.standing(current) {
		if d := distance(r.cellOf(current, foe), r.cellOf(current, who)); nearest == nil || d < best {
			nearest, best = who, d
		}
	}

	return nearest
}

// inReach reports whether a unit can strike a monster from where it
// stands, telling its leader to move otherwise.
func (r *Realm) inReach(current *fight, who unit, foe *monster) bool {
	if !r.dnd() || r.adjacent(current, who, foe) {
		return true
	}

	r.tell(who.leader(), fmt.Sprintf("%s stoi za daleko od: %s. Podejdź (Ruch), zostało %d pól.", who.label(), foe.name, current.board.moves))

	return false
}

// BoardView is the battlefield of a D&D fight as a player sees it.
type BoardView struct {
	Width  int
	Height int
	// Pieces are the living, back row first; Fallen the monsters that
	// died where they stood.
	Pieces []PieceView
	Fallen []PieceView
	// Moves is how many cells the viewer's actor may still walk.
	Moves int
}

// PieceView is one combatant on the board.
type PieceView struct {
	X, Y int
	Name string
	// Foe pieces carry the monster's ID and shape; Mine marks the viewer's
	// own units and Current whose turn it is.
	Foe     bool
	ID      int
	Shape   world.Shape
	Boss    bool
	Mine    bool
	Current bool
	// Avatar is a hero's look and Role a companion's; HP and MaxHP draw
	// the bar under the piece.
	Avatar string
	Role   world.Role
	HP     int
	MaxHP  int
}

func (r *Realm) boardView(current *fight, viewer string) *BoardView {
	if current.board == nil {
		return nil
	}

	view := &BoardView{Width: BoardWidth, Height: BoardHeight, Moves: current.board.moves}
	now := r.current(current)

	for who, cell := range current.board.cells {
		piece := PieceView{X: cell.X, Y: cell.Y}

		switch who := who.(type) {
		case *monster:
			piece.Name, piece.Foe, piece.ID, piece.Shape, piece.Boss = who.name, true, who.id, who.shape, who.boss
			piece.HP, piece.MaxHP = who.hp, who.maxHP
			piece.Current = now != nil && now.foe == who

			// The fallen stay where they fell, as a mark on the field.
			if who.hp <= 0 {
				view.Fallen = append(view.Fallen, piece)

				continue
			}
		case unit:
			if !r.standingOnBoard(current, who) {
				continue
			}

			piece.Name, piece.Mine = who.label(), who.leader().name == viewer
			piece.HP, piece.MaxHP = who.health(), who.maxHealth()
			piece.Current = now != nil && now.unit == who

			switch who := who.(type) {
			case *player:
				piece.Avatar = who.avatar
			case *companion:
				piece.Role = who.spec().Role
			}
		}

		view.Pieces = append(view.Pieces, piece)
	}

	sort.Slice(view.Pieces, func(a, b int) bool {
		if view.Pieces[a].Y != view.Pieces[b].Y {
			return view.Pieces[a].Y < view.Pieces[b].Y
		}

		return view.Pieces[a].X < view.Pieces[b].X
	})

	return view
}
