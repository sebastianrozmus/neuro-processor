package world

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Mode is the kind of game a world is played as.
type Mode string

const (
	// ModeImagination is the game as it began: a map, fights resolved by
	// the rules' own numbers, settlements and shopkeepers to haggle with.
	ModeImagination Mode = "imagination"
	// ModeDnD plays the same world by the rules of the tabletop game: a
	// d20 against an armour class, saving throws and fights on a grid.
	ModeDnD Mode = "dnd"
	// ModeAdventure is a text adventure in the manner of Mózgoprocesor:
	// rooms, things to find and use, people to talk to and a goal, with
	// no fights at all.
	ModeAdventure Mode = "adventure"
)

// Modes lists every mode.
var Modes = []Mode{ModeImagination, ModeDnD, ModeAdventure}

// Title names the mode for players.
func (m Mode) Title() string {
	switch m {
	case ModeDnD:
		return "D&D"
	case ModeAdventure:
		return "Przygodowa"
	}

	return "Gra wyobraźni"
}

// Blurb explains the mode in a line.
func (m Mode) Blurb() string {
	switch m {
	case ModeDnD:
		return "Zasady stołowego RPG: rzut k20 przeciw klasie pancerza, rzuty obronne, walka na siatce z ruchem i atakami okazyjnymi."
	case ModeAdventure:
		return "Tekstowa przygodówka jak Mózgoprocesor: pomieszczenia, przedmioty, zagadki i ludzie, z którymi trzeba pogadać. Bez walki."
	}

	return "Mapa, osady, kupcy i walka rozstrzygana prostymi liczbami, z narratorem i rysowanymi scenami."
}

// KindThing is an object of an adventure: it has no numbers, it is only
// found, carried and used on other things.
const KindThing Kind = "thing"

// Adventure limits.
const (
	maxRooms  = 24
	maxThings = 40
	maxPeople = 12
	maxSteps  = 60
)

// RoomSpec is one place of an adventure, at a position on the map.
type RoomSpec struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Terrain     Terrain `json:"terrain"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	// Things names what lies or stands here at the start.
	Things []string `json:"things,omitempty"`
	// Doors are exits that stay shut until a flag is set.
	Doors []DoorSpec `json:"doors,omitempty"`
}

// DoorSpec is a shut exit: Text is what the player sees when they try it
// and Flag is what opens it.
type DoorSpec struct {
	Dir  string `json:"dir"`
	Text string `json:"text"`
	Flag string `json:"flag"`
}

// ThingSpec is an object of an adventure. Fixed things stay where they are.
type ThingSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Fixed       bool   `json:"fixed,omitempty"`
}

// PersonSpec is somebody to talk to, standing in a room. Talk is what
// they say when no step of the story applies.
type PersonSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Room        string `json:"room"`
	Talk        string `json:"talk"`
}

// StepSpec is one scripted response of an adventure: what happens when
// the player does a thing in a place, and what it changes.
type StepSpec struct {
	// Verb is one of use, give, talk, open, examine, enter.
	Verb string `json:"verb"`
	// Thing is what the player uses or gives; On is what they use it on,
	// whom they give it to or talk to, what they open or examine, or the
	// room they enter.
	Thing string `json:"thing,omitempty"`
	On    string `json:"on"`
	// Room is where the step works; empty means anywhere.
	Room string `json:"room,omitempty"`
	// Needs is a flag the step waits for; Unless a flag that spoils it.
	Needs  string `json:"needs,omitempty"`
	Unless string `json:"unless,omitempty"`
	Once   bool   `json:"once,omitempty"`
	Text   string `json:"text"`
	// Flag is set when the step happens; Gives puts a thing in the bag,
	// Takes removes the thing used, Reveals makes a thing appear in the
	// room and Changes turns a room into something else.
	Flag    string      `json:"flag,omitempty"`
	Gives   string      `json:"gives,omitempty"`
	Takes   bool        `json:"takes,omitempty"`
	Reveals string      `json:"reveals,omitempty"`
	Changes *RoomChange `json:"changes,omitempty"`
	Win     bool        `json:"win,omitempty"`
}

// RoomChange is a room's new look after a step.
type RoomChange struct {
	Room        string  `json:"room"`
	Description string  `json:"description"`
	Terrain     Terrain `json:"terrain,omitempty"`
}

// Verbs of steps.
const (
	StepUse     = "use"
	StepGive    = "give"
	StepTalk    = "talk"
	StepOpen    = "open"
	StepExamine = "examine"
	StepEnter   = "enter"
)

// StepVerbs lists every verb a step may have.
var StepVerbs = []string{StepUse, StepGive, StepTalk, StepOpen, StepExamine, StepEnter}

// Directions of doors.
var Directions = []string{"n", "e", "s", "w"}

// normalizeAdventure cleans the adventure part of a spec and derives the
// places and items the rest of the game expects from the rooms and things.
func (s *Spec) normalizeAdventure() {
	s.Goal = clip(s.Goal, maxText)

	rooms := make([]RoomSpec, 0, len(s.Rooms))
	for _, room := range s.Rooms {
		room.Name = clip(room.Name, maxName)
		room.Description = clip(room.Description, maxText)
		room.Terrain = knownOr(Terrains, room.Terrain)
		room.X, room.Y = min(max(room.X, 0), Width-1), min(max(room.Y, 0), Height-1)

		taken := slices.ContainsFunc(rooms, func(other RoomSpec) bool {
			return other.Name == room.Name || (other.X == room.X && other.Y == room.Y)
		})
		if room.Name == "" || taken || len(rooms) == maxRooms {
			continue
		}

		doors := make([]DoorSpec, 0, len(room.Doors))
		for _, door := range room.Doors {
			door.Dir = strings.ToLower(strings.TrimSpace(door.Dir))
			if slices.Contains(Directions, door.Dir) && door.Flag != "" {
				door.Text = clip(door.Text, maxText)
				doors = append(doors, door)
			}
		}

		room.Doors = doors
		rooms = append(rooms, room)
	}

	things := make([]ThingSpec, 0, len(s.Things))
	for _, thing := range s.Things {
		thing.Name = clip(thing.Name, maxName)
		thing.Description = clip(thing.Description, maxText)

		taken := slices.ContainsFunc(things, func(other ThingSpec) bool { return other.Name == thing.Name })
		if thing.Name != "" && !taken && len(things) < maxThings {
			things = append(things, thing)
		}
	}

	people := make([]PersonSpec, 0, len(s.People))
	for _, person := range s.People {
		person.Name = clip(person.Name, maxName)
		person.Description = clip(person.Description, maxText)
		person.Talk = clip(person.Talk, maxText)

		taken := slices.ContainsFunc(people, func(other PersonSpec) bool { return other.Name == person.Name })
		known := slices.ContainsFunc(rooms, func(room RoomSpec) bool { return room.Name == person.Room })
		if person.Name != "" && !taken && known && len(people) < maxPeople {
			people = append(people, person)
		}
	}

	steps := make([]StepSpec, 0, len(s.Steps))
	for _, step := range s.Steps {
		step.Verb = strings.ToLower(strings.TrimSpace(step.Verb))
		step.Text = clip(step.Text, maxText)

		if slices.Contains(StepVerbs, step.Verb) && step.On != "" && len(steps) < maxSteps {
			if step.Changes != nil {
				step.Changes.Description = clip(step.Changes.Description, maxText)
				if step.Changes.Terrain != "" {
					step.Changes.Terrain = knownOr(Terrains, step.Changes.Terrain)
				}
			}

			steps = append(steps, step)
		}
	}

	s.Rooms, s.Things, s.People, s.Steps = rooms, things, people, steps

	// Rooms become places and things become items, so that the map, the
	// bag and the ground work as in any world.
	s.Places = s.Places[:0]
	for _, room := range rooms {
		s.Places = append(s.Places, PlaceSpec{Name: room.Name, Description: room.Description, Terrain: room.Terrain, Danger: 1})
	}

	s.Items = s.Items[:0]
	for index, thing := range things {
		s.Items = append(s.Items, ItemSpec{
			ID: fmt.Sprintf("item-%d", index+1), Name: thing.Name, Description: thing.Description,
			Kind: KindThing, Effect: EffectNone, Tier: 1, Fixed: thing.Fixed,
		})
	}

	s.Sites, s.Monsters, s.Shops, s.Companions, s.Boss = nil, nil, nil, nil, BossSpec{}
}

func (s *Spec) checkAdventure() error {
	switch {
	case s.Name == "":
		return errors.New("world has no name")
	case len(s.Rooms) < 2:
		return errors.New("adventure needs at least two rooms")
	case !slices.ContainsFunc(s.Steps, func(step StepSpec) bool { return step.Win }):
		return errors.New("adventure has no step that wins it")
	}

	return nil
}

// Room returns the room of the given name.
func (s *Spec) Room(name string) (RoomSpec, bool) {
	for _, room := range s.Rooms {
		if room.Name == name {
			return room, true
		}
	}

	return RoomSpec{}, false
}

// buildAdventure lays the rooms out where the spec puts them; every other
// tile is void.
func buildAdventure(spec Spec) (*World, error) {
	world := &World{Spec: spec, Tiles: make([]Tile, Width*Height)}

	for index := range world.Tiles {
		world.Tiles[index] = Tile{Place: -1, Town: -1, Site: -1}
	}

	for index, room := range spec.Rooms {
		tile := Tile{Place: index, Town: -1, Site: -1}

		for _, name := range room.Things {
			for item, thing := range spec.Items {
				if thing.Name == name {
					tile.Loot = append(tile.Loot, item)
				}
			}
		}

		at := Point{X: room.X, Y: room.Y}
		world.Tiles[world.Index(at)] = tile

		if index == 0 {
			world.Towns = []Point{at}
			world.Lair = at
		}
	}

	return world, nil
}

// RoomAt returns the room on a tile of an adventure.
func (w *World) RoomAt(p Point) (RoomSpec, bool) {
	if place := w.At(p).Place; w.Mode == ModeAdventure && place >= 0 && place < len(w.Rooms) {
		return w.Rooms[place], true
	}

	return RoomSpec{}, false
}

// Door returns the shut exit of a room in a direction, if there is one.
func (r RoomSpec) Door(dir string) (DoorSpec, bool) {
	for _, door := range r.Doors {
		if door.Dir == dir {
			return door, true
		}
	}

	return DoorSpec{}, false
}

// EventSpec is a scripted turn of a world's story: when it fires and what
// it does. It is the adventure's step for the worlds with maps.
type EventSpec struct {
	ID   string   `json:"id,omitempty"`
	When WhenSpec `json:"when"`
	// Once events fire a single time; Chance, in percent, makes an event
	// uncertain (0 means always).
	Once   bool `json:"once,omitempty"`
	Chance int  `json:"chance,omitempty"`
	// Text is told to the player who triggered the event, or to everyone
	// when Global.
	Text   string `json:"text,omitempty"`
	Global bool   `json:"global,omitempty"`
	// Effects.
	Flag      string      `json:"flag,omitempty"`
	Transform *RoomChange `json:"transform,omitempty"`
	Spawn     *SpawnSpec  `json:"spawn,omitempty"`
	Gives     string      `json:"gives,omitempty"`
	Takes     string      `json:"takes,omitempty"`
	Gold      int         `json:"gold,omitempty"`
	Fact      *FactSpec   `json:"fact,omitempty"`
	// Prices shifts every price in the world by this many percent;
	// Standing moves the hero's standing with the nearest settlement.
	Prices   int  `json:"prices,omitempty"`
	Standing int  `json:"standing,omitempty"`
	Win      bool `json:"win,omitempty"`
}

// WhenSpec is the moment and conditions of an event. Enter names a place,
// site or settlement entered; Killed a monster, with Count how many must
// have fallen; Day the day reached; Talked a keeper chatted with. Has
// requires a thing carried, Flag a flag set and Unless one unset.
type WhenSpec struct {
	Enter  string `json:"enter,omitempty"`
	Killed string `json:"killed,omitempty"`
	Count  int    `json:"count,omitempty"`
	Day    int    `json:"day,omitempty"`
	Talked string `json:"talked,omitempty"`
	Has    string `json:"has,omitempty"`
	Flag   string `json:"flag,omitempty"`
	Unless string `json:"unless,omitempty"`
}

// SpawnSpec brings monsters: Where is "here", "place:<name>" or
// "near:<settlement>".
type SpawnSpec struct {
	Monster string `json:"monster"`
	Count   int    `json:"count,omitempty"`
	Where   string `json:"where,omitempty"`
}

// FactSpec is a fact an event adds to what everybody knows.
type FactSpec struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

const maxEvents = 40

// normalizeEvents cleans the events of a world.
func (s *Spec) normalizeEvents() {
	events := make([]EventSpec, 0, len(s.Events))

	for _, event := range s.Events {
		event.Text = clip(event.Text, maxText)
		event.Chance = min(max(event.Chance, 0), 100)

		if event.Transform != nil {
			event.Transform.Description = clip(event.Transform.Description, maxText)
			if event.Transform.Terrain != "" {
				event.Transform.Terrain = knownOr(Terrains, event.Transform.Terrain)
			}
		}

		if event.Fact != nil {
			event.Fact.Subject, event.Fact.Text = clip(event.Fact.Subject, maxName), clip(event.Fact.Text, maxText)
		}

		if len(events) < maxEvents {
			events = append(events, event)
		}
	}

	s.Events = events
}
