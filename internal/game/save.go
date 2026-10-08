package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const saveVersion = 1

// State is a saved game: the world it was played in and everything that
// changed in it. Fights are not saved; heroes who were fighting meet their
// opponents again when they return.
type State struct {
	Version int        `json:"version"`
	SavedAt time.Time  `json:"saved_at"`
	Host    string     `json:"host"`
	Spec    world.Spec `json:"spec"`

	Players  []PlayerState    `json:"players"`
	Extras   []ExtraState     `json:"extras,omitempty"`
	Monsters [][]MonsterState `json:"monsters"`
	Ground   [][]string       `json:"ground"`
	Revealed []bool           `json:"revealed"`
	Won      bool             `json:"won"`
	// Flags and Done are the state of an adventure's story; Places are
	// the places as the story has left them.
	Flags  []string          `json:"flags,omitempty"`
	Done   []int             `json:"done,omitempty"`
	Places []world.PlaceSpec `json:"places,omitempty"`
	// Day is the world's day; Markup the drift of prices, Slain the kills
	// by kind of monster and Facts what the world knows.
	Day    int            `json:"day,omitempty"`
	Markup map[string]int `json:"markup,omitempty"`
	Slain  map[string]int `json:"slain,omitempty"`
	Facts  []FactState    `json:"facts,omitempty"`
	// Weather is the sky of the day; Caravan where the merchant on the
	// road is; Heir the threat growing; Quests the keepers' jobs.
	Weather Weather       `json:"weather,omitempty"`
	Caravan *CaravanState `json:"caravan,omitempty"`
	Heir    *HeirState    `json:"heir,omitempty"`
	Quests  []QuestState  `json:"quests,omitempty"`
}

// CaravanState is the saved caravan.
type CaravanState struct {
	Pos  world.Point `json:"pos"`
	Goal int         `json:"goal"`
}

// HeirState is the saved threat to come.
type HeirState struct {
	Kind int `json:"kind"`
	Due  int `json:"due"`
}

// QuestState is a saved job.
type QuestState struct {
	Keeper   int    `json:"keeper"`
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Monster  int    `json:"monster"`
	Count    int    `json:"count"`
	Done     int    `json:"done"`
	Reward   int    `json:"reward"`
	Item     string `json:"item,omitempty"`
	Taker    string `json:"taker,omitempty"`
	Finished bool   `json:"finished,omitempty"`
	Paid     bool   `json:"paid,omitempty"`
	Day      int    `json:"day"`
}

// FactState is a saved fact with who knows it.
type FactState struct {
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	Source  string   `json:"source,omitempty"`
	Day     int      `json:"day"`
	Known   []int    `json:"known"`
	Heard   []string `json:"heard,omitempty"`
}

// PlayerState is a saved hero. Items are stored by ID.
type PlayerState struct {
	Name   string      `json:"name"`
	Avatar string      `json:"avatar"`
	Pos    world.Point `json:"pos"`
	Prev   world.Point `json:"prev"`
	Home   int         `json:"home"`
	Scores Abilities   `json:"scores"`
	Day    int         `json:"day"`
	Steps  int         `json:"steps"`
	Level  int         `json:"level"`
	XP     int         `json:"xp"`
	HP     int         `json:"hp"`
	Mana   int         `json:"mana"`
	Gold   int         `json:"gold"`
	Bag    []string    `json:"bag"`
	Spells []string    `json:"spells"`
	// Party holds the companions following the hero.
	Party     []CompanionState `json:"party,omitempty"`
	AutoFight bool             `json:"auto_fight,omitempty"`
	// Bestiary lists the kinds of monster met, with how many fell.
	Bestiary []BeastState `json:"bestiary,omitempty"`
	Weapon   string       `json:"weapon"`
	Armor    string       `json:"armor"`
	Charm    string       `json:"charm"`
	// Standing is what each settlement thinks of the hero.
	Standing map[int]int `json:"standing,omitempty"`
}

// CompanionState is a saved companion with everything they carry.
type CompanionState struct {
	Kind   int       `json:"kind"`
	Scores Abilities `json:"scores"`
	Level  int       `json:"level"`
	XP     int       `json:"xp"`
	HP     int       `json:"hp"`
	Mana   int       `json:"mana"`
	Weapon string    `json:"weapon,omitempty"`
	Armor  string    `json:"armor,omitempty"`
	Charm  string    `json:"charm,omitempty"`
	Bag    []string  `json:"bag,omitempty"`
	Spells []string  `json:"spells,omitempty"`
	Auto   bool      `json:"auto,omitempty"`
}

// BeastState is a saved page of the bestiary.
type BeastState struct {
	Kind  int `json:"kind"`
	Slain int `json:"slain"`
}

// ExtraState is a saved item that a keeper talked into the world.
type ExtraState struct {
	Shop int            `json:"shop"`
	ID   string         `json:"id"`
	Item world.ItemSpec `json:"item"`
}

// MonsterState is a saved monster: its kind (-1 for the boss) and health.
// A boss of a kind is an heir, saved with its name.
type MonsterState struct {
	Kind int    `json:"kind"`
	HP   int    `json:"hp"`
	Boss bool   `json:"boss,omitempty"`
	Name string `json:"name,omitempty"`
}

// State captures the game so it can be saved. Host names the player whose
// character the save belongs to.
func (r *Realm) State(host string) State {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := State{
		Version:  saveVersion,
		SavedAt:  time.Now(),
		Host:     host,
		Spec:     r.world.Spec,
		Monsters: make([][]MonsterState, len(r.monsters)),
		Ground:   make([][]string, len(r.ground)),
		Revealed: append([]bool(nil), r.revealed...),
		Won:      r.won,
		Places:   append([]world.PlaceSpec(nil), r.places...),
	}

	for flag, set := range r.story.flags {
		if set {
			state.Flags = append(state.Flags, flag)
		}
	}

	sort.Strings(state.Flags)
	state.Done = sortedKeys(r.story.done)
	state.Day, state.Markup, state.Slain, state.Weather = r.day, r.markup, r.slain, r.weather

	if r.caravan != nil {
		state.Caravan = &CaravanState{Pos: r.caravan.pos, Goal: r.caravan.goal}
	}

	if r.heir != nil {
		state.Heir = &HeirState{Kind: r.heir.kind, Due: r.heir.due}
	}

	for _, job := range r.quests {
		state.Quests = append(state.Quests, QuestState{
			Keeper: job.keeper, Kind: string(job.kind), Subject: job.subject, Monster: job.monster, Count: job.count, Done: job.done,
			Reward: job.reward, Item: job.item, Taker: job.taker, Finished: job.finished, Paid: job.paid, Day: job.day,
		})
	}

	for _, fact := range r.wisdom.facts {
		saved := FactState{Subject: fact.Subject, Text: fact.Text, Source: fact.Source, Day: fact.Day, Known: sortedKeys(fact.known)}
		for name := range fact.heard {
			saved.Heard = append(saved.Heard, name)
		}

		sort.Strings(saved.Heard)
		state.Facts = append(state.Facts, saved)
	}

	for index, group := range r.monsters {
		for _, foe := range group {
			// Fights are not saved, so the survivors recover as they do
			// when a fight is abandoned.
			if foe.hp > 0 {
				saved := MonsterState{Kind: foe.kind, HP: foe.maxHP}
				if foe.boss && foe.kind >= 0 {
					saved.Boss, saved.Name = true, foe.name
				}

				state.Monsters[index] = append(state.Monsters[index], saved)
			}
		}

		state.Ground[index] = ids(r.ground[index])
	}

	for _, added := range r.extras {
		state.Extras = append(state.Extras, ExtraState{Shop: added.shop, ID: added.item.ID, Item: added.item})
	}

	for _, name := range r.order {
		saved := r.players[name]

		var party []CompanionState
		for _, member := range saved.party {
			party = append(party, CompanionState{
				Kind: member.kind, Scores: member.abilities, Level: member.level, XP: member.xp, HP: member.hp, Mana: member.mana,
				Weapon: id(member.weapon), Armor: id(member.armor), Charm: id(member.charm),
				Bag: ids(member.bag), Spells: ids(member.spells), Auto: member.auto,
			})
		}

		var pages []BeastState
		for _, kind := range sortedKeys(saved.bestiary) {
			pages = append(pages, BeastState{Kind: kind, Slain: saved.bestiary[kind]})
		}

		state.Players = append(state.Players, PlayerState{
			Party: party, AutoFight: saved.autoFight, Bestiary: pages,
			Name: saved.name, Avatar: saved.avatar, Pos: saved.pos, Prev: saved.prev, Home: saved.home,
			Scores: saved.abilities, Day: saved.day, Steps: saved.steps,
			Level: saved.level, XP: saved.xp, HP: saved.hp, Mana: saved.mana, Gold: saved.gold,
			Bag: ids(saved.bag), Spells: ids(saved.spells),
			Weapon: id(saved.weapon), Armor: id(saved.armor), Charm: id(saved.charm),
			Standing: saved.standing,
		})
	}

	return state
}

// Restore rebuilds a realm from a saved game. Nobody is connected to it
// until they join; joining under a saved name resumes that hero.
func Restore(state State, seed int64) (*Realm, error) {
	built, err := world.Build(state.Spec)
	if err != nil {
		return nil, err
	}

	tiles := len(built.Tiles)
	if state.Version != saveVersion || len(state.Monsters) != tiles || len(state.Ground) != tiles || len(state.Revealed) != tiles {
		return nil, errors.New("save does not fit this version of the game")
	}

	realm := NewRealm(built, seed)
	realm.revealed = append([]bool(nil), state.Revealed...)
	realm.won = state.Won

	for _, flag := range state.Flags {
		realm.story.flags[flag] = true
	}

	for _, index := range state.Done {
		if index >= 0 && index < len(built.Steps) {
			realm.story.done[index] = true
		}
	}

	realm.day = max(state.Day, 1)
	for id, markup := range state.Markup {
		realm.markup[id] = clamp(markup, minMarkup, maxMarkup)
	}

	for name, count := range state.Slain {
		realm.slain[name] = max(count, 0)
	}

	if state.Weather != "" {
		realm.weather = state.Weather
	}

	if state.Caravan != nil && realm.caravan != nil && built.In(state.Caravan.Pos) && state.Caravan.Goal >= 0 && state.Caravan.Goal < len(built.Towns) {
		realm.caravan.pos, realm.caravan.goal = state.Caravan.Pos, state.Caravan.Goal
	}

	if state.Heir != nil && state.Heir.Kind >= 0 && state.Heir.Kind < len(built.Monsters) {
		realm.heir = &heir{kind: state.Heir.Kind, due: state.Heir.Due}
	}

	for _, saved := range state.Quests {
		if saved.Keeper < 0 || saved.Keeper > len(built.Shops) || saved.Monster < 0 || saved.Monster >= max(len(built.Monsters), 1) {
			continue
		}

		realm.quests = append(realm.quests, &quest{
			id: len(realm.quests) + 1, keeper: saved.Keeper, kind: questKind(saved.Kind), subject: saved.Subject, monster: saved.Monster,
			count: saved.Count, done: saved.Done, reward: saved.Reward, item: saved.Item, taker: saved.Taker, finished: saved.Finished, paid: saved.Paid, day: saved.Day,
		})
	}

	if len(state.Facts) > 0 {
		realm.wisdom.facts = nil

		for _, saved := range state.Facts {
			fact := realm.add(saved.Subject, saved.Text, saved.Source, saved.Known...)
			if fact == nil {
				break
			}

			fact.Day = saved.Day
			for _, name := range saved.Heard {
				fact.heard[name] = true
			}
		}
	}

	if len(state.Places) == len(realm.places) {
		for index, place := range state.Places {
			if place.Name == realm.places[index].Name {
				place.Terrain = world.KnownTerrain(place.Terrain)
				realm.places[index] = place
			}
		}
	}

	for _, saved := range state.Extras {
		item, ok := world.CleanItem(saved.Item)
		if !ok || saved.Shop < 0 || saved.Shop >= len(built.Shops) || saved.ID == "" {
			continue
		}

		item.ID = saved.ID
		realm.extras = append(realm.extras, extra{shop: saved.Shop, item: item})
	}

	for index := range built.Tiles {
		realm.monsters[index], realm.ground[index] = nil, realm.items(state.Ground[index])

		for _, saved := range state.Monsters[index] {
			if saved.Kind >= len(built.Monsters) || saved.HP <= 0 {
				continue
			}

			foe := realm.newMonster(saved.Kind)
			if saved.Boss && saved.Kind >= 0 {
				foe = realm.heirMonster(saved.Kind)
				if saved.Name != "" {
					foe.name = saved.Name
				}
			}

			foe.hp = min(saved.HP, foe.maxHP)
			realm.monsters[index] = append(realm.monsters[index], foe)
		}
	}

	for _, saved := range state.Players {
		if !built.In(saved.Pos) || !built.In(saved.Prev) || saved.Home < 0 || saved.Home >= len(built.Towns) {
			return nil, errors.New("save places a hero outside the world")
		}

		hero := &player{
			world: built, autoFight: saved.AutoFight, abilities: saved.Scores, day: max(saved.Day, 1), steps: max(saved.Steps, 0),
			name: saved.Name, avatar: saved.Avatar, pos: saved.Pos, prev: saved.Prev, home: saved.Home,
			level: min(max(saved.Level, 1), MaxLevel), xp: max(saved.XP, 0),
			gold: max(saved.Gold, 0), bag: realm.items(saved.Bag), spells: realm.items(saved.Spells),
			weapon: realm.slot(saved.Weapon), armor: realm.slot(saved.Armor), charm: realm.slot(saved.Charm),
			shop: -1, updates: make(chan struct{}, 1),
		}

		for town, score := range saved.Standing {
			if town >= 0 && town < len(built.Towns) {
				if hero.standing == nil {
					hero.standing = map[int]int{}
				}

				hero.standing[town] = clamp(score, standingMin, standingMax)
			}
		}
		// Saves from before character sheets have no scores: they get rolled.
		if !hero.abilities.valid() {
			hero.abilities = heroAbilities(realm.rng, hero.avatar)
		}

		hero.hp = min(max(saved.HP, 1), hero.maxHP())
		hero.mana = min(max(saved.Mana, 0), hero.maxMana())

		for _, page := range saved.Bestiary {
			if page.Kind >= -1 && page.Kind < len(built.Monsters) {
				if hero.bestiary == nil {
					hero.bestiary = map[int]int{}
				}

				hero.bestiary[page.Kind] = max(page.Slain, 0)
			}
		}

		for _, member := range saved.Party {
			if member.Kind < 0 || member.Kind >= len(built.Companions) || realm.hired[member.Kind] != "" || len(hero.party) >= MaxParty {
				continue
			}

			ally := realm.newCompanion(member.Kind, hero)
			if member.Scores.valid() {
				ally.abilities = member.Scores
			}

			ally.level, ally.xp, ally.auto = min(max(member.Level, 1), MaxLevel), max(member.XP, 0), member.Auto
			ally.weapon, ally.armor, ally.charm = realm.slot(member.Weapon), realm.slot(member.Armor), realm.slot(member.Charm)
			ally.bag, ally.spells = realm.items(member.Bag), realm.items(member.Spells)
			ally.hp = min(max(member.HP, 0), ally.maxHealth())
			ally.mana = min(max(member.Mana, 0), ally.maxEnergy())

			hero.party = append(hero.party, ally)
			realm.hired[member.Kind] = hero.name
		}

		realm.players[hero.name] = hero
		realm.order = append(realm.order, hero.name)
	}

	return realm, nil
}

// WriteState saves a game in dir, one file per world.
func WriteState(dir string, state State) (string, error) {
	data, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, world.Slug(state.Spec.Name)+".json")

	return path, os.WriteFile(path, data, 0o644)
}

// ReadStates returns the games saved in dir, newest first. Files that are
// not valid saves are skipped.
func ReadStates(dir string) []State {
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	states := make([]State, 0, len(paths))

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var state State
		if json.Unmarshal(data, &state) == nil && state.Spec.Normalize() == nil && state.Version == saveVersion {
			states = append(states, state)
		}
	}

	sort.Slice(states, func(a, b int) bool { return states[a].SavedAt.After(states[b].SavedAt) })

	return states
}

func id(item *world.ItemSpec) string {
	if item == nil {
		return ""
	}

	return item.ID
}

func ids(list []world.ItemSpec) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.ID)
	}

	return out
}

// items looks saved item IDs up in the world, dropping unknown ones.
func (r *Realm) items(saved []string) []world.ItemSpec {
	var out []world.ItemSpec

	for _, itemID := range saved {
		if item, ok := r.item(itemID); ok {
			out = append(out, item)
		}
	}

	return out
}

func (r *Realm) slot(itemID string) *world.ItemSpec {
	if item, ok := r.item(itemID); ok {
		return &item
	}

	return nil
}

func sortedKeys[V any](counts map[int]V) []int {
	keys := make([]int, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}

	sort.Ints(keys)

	return keys
}
