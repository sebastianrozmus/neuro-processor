// Package world describes a game world as data. A Spec holds only names,
// descriptions and tiers, so it can come from a built-in preset or from a
// language model; every number the game uses is derived from it here.
package world

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Kind is what an item is used for.
type Kind string

const (
	KindWeapon Kind = "weapon"
	KindArmor  Kind = "armor"
	KindPotion Kind = "potion"
	// KindSpell is a scroll or manual that teaches a power when used.
	KindSpell Kind = "spell"
	// KindCharm is a magic item worn for a passive bonus.
	KindCharm Kind = "charm"
)

// Kinds lists every item kind.
var Kinds = []Kind{KindWeapon, KindArmor, KindPotion, KindSpell, KindCharm}

// Effect says what a potion, spell or charm does.
type Effect string

const (
	EffectNone  Effect = "none"
	EffectHeal  Effect = "heal"
	EffectMana  Effect = "mana"
	EffectBolt  Effect = "bolt"
	EffectBlast Effect = "blast"
	EffectPower Effect = "power"
	EffectWard  Effect = "ward"
	EffectVigor Effect = "vigor"
	EffectFocus Effect = "focus"
)

// Effects lists every effect.
var Effects = []Effect{
	EffectNone, EffectHeal, EffectMana, EffectBolt, EffectBlast, EffectPower, EffectWard, EffectVigor, EffectFocus,
}

// effectsOf lists the effects an item kind may have, the default first.
var effectsOf = map[Kind][]Effect{
	KindWeapon: {EffectNone},
	KindArmor:  {EffectNone},
	KindPotion: {EffectHeal, EffectMana},
	KindSpell:  {EffectBolt, EffectBlast, EffectHeal},
	KindCharm:  {EffectPower, EffectWard, EffectVigor, EffectFocus},
}

// Terrain selects how a place is drawn.
type Terrain string

// Terrains lists every terrain a place can have.
var Terrains = []Terrain{
	"plains", "forest", "mountain", "swamp", "desert", "ruins",
	"cave", "snow", "wasteland", "water", "street", "station",
}

// Shape selects how a monster is drawn.
type Shape string

// Shapes lists every monster shape.
var Shapes = []Shape{"beast", "undead", "humanoid", "spirit", "serpent", "insect", "machine", "blob"}

// ShopKind decides what a shop sells.
type ShopKind string

const (
	ShopArmory    ShopKind = "armory"
	ShopAlchemist ShopKind = "alchemist"
	ShopMagic     ShopKind = "magic"
	// ShopCaravan is the merchant on the road, who deals in everything;
	// no world declares one, the game sends it out.
	ShopCaravan ShopKind = "caravan"
)

// ShopKinds lists every kind of shop.
var ShopKinds = []ShopKind{ShopArmory, ShopAlchemist, ShopMagic}

// Role is what a companion does in a fight.
type Role string

const (
	RoleFighter Role = "fighter"
	RoleHealer  Role = "healer"
	RoleMage    Role = "mage"
)

// Roles lists every companion role.
var Roles = []Role{RoleFighter, RoleHealer, RoleMage}

// SiteKind decides what a special place holds.
type SiteKind string

const (
	// SiteTreasure is a guarded cache of valuable items.
	SiteTreasure SiteKind = "treasure"
	// SiteShrine restores health and power to whoever visits it.
	SiteShrine SiteKind = "shrine"
)

// SiteKinds lists every kind of special place.
var SiteKinds = []SiteKind{SiteTreasure, SiteShrine}

const (
	// MaxTier is the highest tier of places, monsters and items.
	MaxTier = 3

	maxTowns        = 3
	maxPlaces       = 12
	maxSites        = 5
	maxCompanions   = 6
	maxMonsters     = 12
	maxItems        = 24
	maxName         = 40
	maxText         = 400
	defaultCurrency = "szt. złota"
	defaultGreeting = "Czego szukasz?"
)

// PlaceSpec is a kind of location. Danger 0 marks a settlement; 1 to 3 are
// increasingly hostile regions.
type PlaceSpec struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Terrain     Terrain `json:"terrain"`
	Danger      int     `json:"danger"`
}

// SiteSpec is a special place found on a single tile of the map.
type SiteSpec struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Kind        SiteKind `json:"kind"`
}

// MonsterSpec is a kind of monster of tier 1 to 3.
type MonsterSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Shape       Shape  `json:"shape"`
	Tier        int    `json:"tier"`
}

// BossSpec is the monster whose defeat wins the game.
type BossSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Shape       Shape  `json:"shape"`
}

// CompanionSpec is a hero for hire, waiting in a settlement's inn. A legend
// is far stronger than the rest and is not bought but paid by the day.
type CompanionSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        Role   `json:"role"`
	Tier        int    `json:"tier"`
	Legend      bool   `json:"legend,omitempty"`
}

// ItemSpec is a kind of item of tier 1 to 3.
type ItemSpec struct {
	ID          string `json:"-"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        Kind   `json:"kind"`
	Effect      Effect `json:"effect"`
	Tier        int    `json:"tier"`
	// Fixed things of an adventure cannot be picked up.
	Fixed bool `json:"fixed,omitempty"`
}

// ShopSpec is a building found in every settlement, with its keeper.
type ShopSpec struct {
	Name     string   `json:"name"`
	Kind     ShopKind `json:"kind"`
	Keeper   string   `json:"keeper"`
	Persona  string   `json:"persona"`
	Greeting string   `json:"greeting"`
}

// Spec is the full description of a world.
type Spec struct {
	Name string `json:"name"`
	// Mode is the kind of game the world is played as.
	Mode  Mode   `json:"mode,omitempty"`
	Intro string `json:"intro"`
	// Currency is the phrase put after an amount, e.g. "kredytów".
	Currency string `json:"currency"`
	// Backdrops name embedded art shown behind the world in the menu.
	Backdrops  []string        `json:"backdrops,omitempty"`
	Places     []PlaceSpec     `json:"places"`
	Sites      []SiteSpec      `json:"sites"`
	Monsters   []MonsterSpec   `json:"monsters"`
	Boss       BossSpec        `json:"boss"`
	Items      []ItemSpec      `json:"items"`
	Shops      []ShopSpec      `json:"shops"`
	Companions []CompanionSpec `json:"companions"`

	// Events are the scripted turns of the story of a world with a map.
	Events []EventSpec `json:"events,omitempty"`

	// An adventure is made of these instead; Goal is what the player is
	// told to achieve.
	Goal   string       `json:"goal,omitempty"`
	Rooms  []RoomSpec   `json:"rooms,omitempty"`
	Things []ThingSpec  `json:"things,omitempty"`
	People []PersonSpec `json:"people,omitempty"`
	Steps  []StepSpec   `json:"steps,omitempty"`
}

// Sells reports whether a shop of this kind trades in items of the given kind.
func (k ShopKind) Sells(kind Kind) bool {
	switch k {
	case ShopArmory:
		return kind == KindWeapon || kind == KindArmor
	case ShopAlchemist:
		return kind == KindPotion
	case ShopCaravan:
		return true
	}

	return kind == KindSpell || kind == KindCharm
}

// Normalize cleans a spec of any origin so the game can rely on it: texts
// are trimmed, tiers clamped, unknown values replaced, repeated names
// dropped and item IDs assigned. It fails only when the spec lacks something the game cannot
// do without.
func (s *Spec) Normalize() error {
	s.Name = clip(s.Name, maxName)
	s.Intro = clip(s.Intro, maxText)
	s.Currency = clip(s.Currency, maxName)
	s.Boss.Name = clip(s.Boss.Name, maxName)
	s.Boss.Description = clip(s.Boss.Description, maxText)
	s.Boss.Shape = knownOr(Shapes, s.Boss.Shape)

	if s.Currency == "" {
		s.Currency = defaultCurrency
	}

	s.Mode = knownOr(Modes, s.Mode)

	if s.Mode == ModeAdventure {
		s.normalizeAdventure()

		return s.checkAdventure()
	}

	s.normalizePlaces()
	s.normalizeCreatures()
	s.normalizeItems()
	s.normalizeShops()
	s.normalizeEvents()

	return s.check()
}

func (s *Spec) normalizePlaces() {
	places, towns := make([]PlaceSpec, 0, len(s.Places)), 0

	for _, place := range s.Places {
		place.Name = clip(place.Name, maxName)
		place.Description = clip(place.Description, maxText)
		place.Danger = min(max(place.Danger, 0), MaxTier)
		place.Terrain = knownOr(Terrains, place.Terrain)

		if place.Danger == 0 {
			towns++
		}

		taken := slices.ContainsFunc(places, func(other PlaceSpec) bool { return other.Name == place.Name })
		if place.Name == "" || taken || len(places) == maxPlaces || (place.Danger == 0 && towns > maxTowns) {
			continue
		}

		places = append(places, place)
	}

	sites := make([]SiteSpec, 0, len(s.Sites))
	for _, site := range s.Sites {
		site.Name = clip(site.Name, maxName)
		site.Description = clip(site.Description, maxText)
		site.Kind = knownOr(SiteKinds, site.Kind)

		taken := slices.ContainsFunc(sites, func(other SiteSpec) bool { return other.Name == site.Name })
		if site.Name != "" && !taken && len(sites) < maxSites {
			sites = append(sites, site)
		}
	}

	s.Places, s.Sites = places, sites
}

func (s *Spec) normalizeCreatures() {
	monsters := make([]MonsterSpec, 0, len(s.Monsters))

	for _, monster := range s.Monsters {
		monster.Name = clip(monster.Name, maxName)
		monster.Description = clip(monster.Description, maxText)
		monster.Tier = min(max(monster.Tier, 1), MaxTier)
		monster.Shape = knownOr(Shapes, monster.Shape)

		taken := slices.ContainsFunc(monsters, func(other MonsterSpec) bool { return other.Name == monster.Name })
		if monster.Name != "" && !taken && len(monsters) < maxMonsters {
			monsters = append(monsters, monster)
		}
	}

	s.Monsters = monsters

	companions := make([]CompanionSpec, 0, len(s.Companions))

	for _, companion := range s.Companions {
		companion.Name = clip(companion.Name, maxName)
		companion.Description = clip(companion.Description, maxText)
		companion.Tier = min(max(companion.Tier, 1), MaxTier)
		companion.Role = knownOr(Roles, companion.Role)

		// There is one legend to a world; further ones are ordinary heroes.
		legends := slices.ContainsFunc(companions, func(other CompanionSpec) bool { return other.Legend })
		companion.Legend = companion.Legend && !legends

		taken := slices.ContainsFunc(companions, func(other CompanionSpec) bool { return other.Name == companion.Name })
		if companion.Name != "" && !taken && len(companions) < maxCompanions {
			companions = append(companions, companion)
		}
	}

	s.Companions = companions
}

func (s *Spec) normalizeItems() {
	items := make([]ItemSpec, 0, len(s.Items))

	for _, item := range s.Items {
		item, ok := CleanItem(item)
		taken := slices.ContainsFunc(items, func(other ItemSpec) bool { return other.Name == item.Name })

		if !ok || taken || len(items) == maxItems {
			continue
		}

		item.ID = fmt.Sprintf("item-%d", len(items)+1)
		items = append(items, item)
	}

	s.Items = items
}

// CleanItem tidies an item of any origin: texts trimmed, tier clamped, the
// effect made to fit the kind. It reports false for an item the game cannot
// use: one without a name or of an unknown kind.
func CleanItem(item ItemSpec) (ItemSpec, bool) {
	item.Name = clip(item.Name, maxName)
	item.Description = clip(item.Description, maxText)
	item.Tier = min(max(item.Tier, 1), MaxTier)

	allowed, known := effectsOf[item.Kind]
	if item.Kind == KindThing {
		allowed, known = []Effect{EffectNone}, true
	}

	if item.Name == "" || !known {
		return item, false
	}

	item.Effect = knownOr(allowed, item.Effect)

	return item, true
}

func (s *Spec) normalizeShops() {
	shops := make([]ShopSpec, 0, len(s.Shops))

	for _, shop := range s.Shops {
		shop.Name = clip(shop.Name, maxName)
		shop.Keeper = clip(shop.Keeper, maxName)
		shop.Persona = clip(shop.Persona, maxText)
		shop.Greeting = clip(shop.Greeting, maxText)
		shop.Kind = knownOr(ShopKinds, shop.Kind)

		if shop.Greeting == "" {
			shop.Greeting = defaultGreeting
		}

		taken := slices.ContainsFunc(shops, func(other ShopSpec) bool { return other.Kind == shop.Kind })
		if shop.Name != "" && shop.Keeper != "" && !taken {
			shops = append(shops, shop)
		}
	}

	s.Shops = shops
}

func (s *Spec) check() error {
	switch {
	case s.Name == "":
		return errors.New("world has no name")
	case s.Boss.Name == "":
		return errors.New("world has no boss")
	case len(s.Shops) == 0:
		return errors.New("world has no shops")
	case len(s.Monsters) == 0:
		return errors.New("world has no monsters")
	case len(s.TownPlaces()) == 0 || len(s.TownPlaces()) == len(s.Places):
		return errors.New("world needs a settlement and at least one dangerous place")
	case !slices.ContainsFunc(s.Items, func(item ItemSpec) bool { return item.Kind == KindWeapon }):
		return errors.New("world has no weapons")
	}

	return nil
}

// TownPlaces returns the indexes of the places that are settlements.
func (s *Spec) TownPlaces() []int {
	var towns []int

	for index, place := range s.Places {
		if place.Danger == 0 {
			towns = append(towns, index)
		}
	}

	return towns
}

// Item returns the item with the given ID.
func (s *Spec) Item(id string) (ItemSpec, bool) {
	for _, item := range s.Items {
		if item.ID == id {
			return item, true
		}
	}

	return ItemSpec{}, false
}

// ShopFor returns the index of the shop that sells items of the given kind.
// Kinds no shop specialises in are sold by the first shop.
func (s *Spec) ShopFor(kind Kind) int {
	for index, shop := range s.Shops {
		if shop.Kind.Sells(kind) {
			return index
		}
	}

	return 0
}

func clip(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > limit {
		return strings.TrimSpace(string(runes[:limit]))
	}

	return text
}

// knownOr returns value if it is one of known, and the first known otherwise.
func knownOr[T comparable](known []T, value T) T {
	if slices.Contains(known, value) {
		return value
	}

	return known[0]
}

// KnownTerrain returns the terrain if the game can draw it, the first one
// otherwise.
func KnownTerrain(terrain Terrain) Terrain {
	return knownOr(Terrains, terrain)
}
