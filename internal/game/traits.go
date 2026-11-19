package game

import (
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Trait is what a shape of monster does beyond a plain blow. Every shape
// has one, so no two kinds of enemy fight quite the same way.
type Trait struct {
	Name string
	Text string
}

var traits = map[world.Shape]Trait{
	"beast":    {"Szał", "Poniżej połowy zdrowia atakuje dwa razy w rundzie."},
	"undead":   {"Nieumarły", "Na początku każdej rundy zrasta się i odzyskuje zdrowie."},
	"humanoid": {"Tarcza", "Co trzeci cios bronią odbija tarczą i traci od niego o połowę mniej."},
	"spirit":   {"Widmo", "Broń rani go o połowę słabiej; moce ranią w pełni."},
	"serpent":  {"Jad", "Jego ugryzienie zatruwa na kilka rund; napar leczniczy zdejmuje truciznę."},
	"insect":   {"Rój", "Kąsa dwóch naraz, każdego słabiej."},
	"machine":  {"Pancerz", "Każdy cios rani go mniej o grubość jego pancerza."},
	"blob":     {"Kwas", "Jego cios przeżera zbroję: obrona przed nim nie chroni."},
}

// bossTrait is what the boss does on top of the trait of its shape.
var bossTrait = Trait{"Furia", "Poniżej jednej trzeciej zdrowia bije o połowę mocniej."}

const (
	poisonRounds = 3
	poisonDamage = 3
	// shieldChance is one in this many blows that a humanoid blocks.
	shieldChance = 3
	// blowsKept is how many recent blows the realm remembers for the screens.
	blowsKept = 16
)

// TraitOf returns the trait of a shape of monster.
func TraitOf(shape world.Shape) Trait {
	return traits[shape]
}

// traitNames words the traits of a monster, for the player to learn from.
func (m *monster) traitNames() string {
	named := traits[m.shape].Name
	if m.boss {
		named += ", " + bossTrait.Name
	}

	return named
}

// BlowKind says what a blow was, so a screen can show it fittingly.
type BlowKind string

const (
	// BlowHit is a weapon striking a monster, BlowBolt and BlowBlast a
	// power; BlowWound is a monster hurting somebody on the heroes' side,
	// BlowPoison the venom in them; BlowHeal is health restored to the
	// heroes' side and BlowRegen to a monster.
	BlowHit    BlowKind = "hit"
	BlowBolt   BlowKind = "bolt"
	BlowBlast  BlowKind = "blast"
	BlowWound  BlowKind = "wound"
	BlowPoison BlowKind = "poison"
	BlowHeal   BlowKind = "heal"
	BlowRegen  BlowKind = "regen"
)

// BlowView is one thing that happened to somebody's health in a fight.
// Serial grows with every blow, so a screen can tell new ones from those
// it has shown.
type BlowView struct {
	Serial int
	Pos    world.Point
	// Foe is the ID of the monster concerned, or -1 for the heroes' side.
	Foe    int
	Amount int
	Kind   BlowKind
	// From names who struck and To who took it, as the fight labels them;
	// either may be empty when nobody in particular did.
	From string
	To   string
}

// blow records what just happened to somebody's health.
func (r *Realm) blow(pos world.Point, foe, amount int, kind BlowKind) {
	r.blowBy(pos, foe, amount, kind, "", "")
}

// blowBy records a blow together with who dealt it and who took it.
func (r *Realm) blowBy(pos world.Point, foe, amount int, kind BlowKind, from, to string) {
	r.serial++
	r.blows = append(r.blows, BlowView{Serial: r.serial, Pos: pos, Foe: foe, Amount: amount, Kind: kind, From: from, To: to})

	if len(r.blows) > blowsKept {
		r.blows = r.blows[len(r.blows)-blowsKept:]
	}
}

// BeastView is one page of the bestiary: a kind of monster, with its
// numbers once it has been met.
type BeastView struct {
	// Kind indexes the world's monsters; the boss has -1.
	Kind        int
	Name        string
	Description string
	Shape       world.Shape
	Tier        int
	Boss        bool
	HP          int
	Attack      int
	Speed       int
	AC          int
	Trait       Trait
	// Known reports that the viewer has met this kind; Slain counts the
	// ones that fell in fights the viewer took part in.
	Known bool
	Slain int
}

// Damage words the range of a blow of a monster.
func (b BeastView) Damage() string {
	return fmt.Sprintf("%d–%d", b.Attack, b.Attack+b.Attack/2)
}

// bestiaryView lists every kind of monster of the world, the boss last.
func (r *Realm) bestiaryView(viewer *player) []BeastView {
	pages := make([]BeastView, 0, len(r.world.Monsters)+1)

	for kind := range r.world.Monsters {
		pages = append(pages, r.page(viewer, kind))
	}

	return append(pages, r.page(viewer, -1))
}

func (r *Realm) page(viewer *player, kind int) BeastView {
	sample := r.newMonster(kind)
	slain, known := viewer.bestiary[kind]
	page := BeastView{
		Kind: kind, Name: sample.name, Shape: sample.shape, Tier: sample.tier, Boss: sample.boss,
		HP: sample.maxHP, Attack: sample.attack, Speed: sample.pace(), AC: monsterClass(sample), Trait: traits[sample.shape],
		Known: known, Slain: slain,
	}

	if kind < 0 {
		page.Description, page.Tier = r.world.Boss.Description, world.MaxTier
		page.Trait = Trait{bossTrait.Name + " i " + page.Trait.Name, bossTrait.Text + " " + page.Trait.Text}
	} else {
		page.Description = r.world.Monsters[kind].Description
	}

	return page
}

// meet notes in a player's bestiary that they have seen the monsters on a
// tile.
func (r *Realm) meet(viewer *player, pos world.Point) {
	if viewer.bestiary == nil {
		viewer.bestiary = map[int]int{}
	}

	for _, foe := range r.foes(pos) {
		if _, ok := viewer.bestiary[foe.kind]; !ok {
			viewer.bestiary[foe.kind] = 0
		}
	}
}

// fell credits everyone in a fight with a monster that fell, counts it
// for the world and fires what its death triggers.
func (r *Realm) fell(current *fight, foe *monster) {
	for _, name := range current.members {
		hero := r.players[name]
		if hero.bestiary == nil {
			hero.bestiary = map[int]int{}
		}

		hero.bestiary[foe.kind]++
	}

	r.slain[foe.name]++
	r.guarded(current)
	r.hunted(current, foe)

	for _, name := range current.members {
		r.fire(r.players[name], onKill, foe.name)
	}
}
