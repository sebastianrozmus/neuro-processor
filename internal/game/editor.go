package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The editor lets whoever runs the inspector set a hero or a companion up
// by hand, the way a long game would have: their numbers, what they wear,
// what they carry and the powers they know, from the world's items or
// from ones made on the spot.

// Change is one edit of a hero or of a companion in their party.
type Change struct {
	// Member names the companion edited; empty edits the hero.
	Member string `json:"member"`
	// Op is "stat", "equip", "give", "teach" or "take".
	Op string `json:"op"`
	// Field is the number for "stat" (gold, hp, mana, level, xp, or an
	// ability: sil, zre, kon, int, mad, cha), the slot for "equip" (weapon,
	// armor, charm) and the list for "take" (bag, spells).
	Field string `json:"field"`
	// Item is the ID of the item given, taught or worn; empty takes the
	// slot off.
	Item  string `json:"item"`
	Value int    `json:"value"`
}

// sheet is what the editor may touch of a hero or a companion.
type sheet struct {
	label                string
	weapon, armor, charm **world.ItemSpec
	bag, spells          *[]world.ItemSpec
	hp, mana, level, xp  *int
	// gold is the hero's purse; companions have none.
	gold           *int
	abilities      *Abilities
	maxHP, maxMana func() int
}

// abilityFields are the editor's names of the abilities, in order;
// maxAbility the highest score it sets.
var abilityFields = [abilityCount]string{"sil", "zre", "kon", "int", "mad", "cha"}

const maxAbility = 30

var (
	errNoHero   = errors.New("nie ma takiego gracza")
	errNoMember = errors.New("nie ma takiego towarzysza w drużynie")
	errNoItem   = errors.New("nie ma takiego przedmiotu")
	errField    = errors.New("nie ma takiego pola")
)

// Edit applies a change made in the inspector and tells the hero.
func (r *Realm) Edit(name string, change Change) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hero := r.players[name]
	if hero == nil {
		return errNoHero
	}

	sheet, err := r.sheetOf(hero, change.Member)
	if err != nil {
		return err
	}

	said, err := r.apply(sheet, change)
	if err != nil {
		return err
	}

	*sheet.hp, *sheet.mana = min(*sheet.hp, sheet.maxHP()), min(*sheet.mana, sheet.maxMana())
	r.tell(hero, "Podgląd zmienia: "+sheet.label+" — "+said+".")
	r.notify()

	return nil
}

func (r *Realm) sheetOf(hero *player, member string) (sheet, error) {
	if member == "" {
		return sheet{
			label: hero.name, weapon: &hero.weapon, armor: &hero.armor, charm: &hero.charm, bag: &hero.bag, spells: &hero.spells,
			gold: &hero.gold, hp: &hero.hp, mana: &hero.mana, level: &hero.level, xp: &hero.xp, abilities: &hero.abilities, maxHP: hero.maxHP, maxMana: hero.maxMana,
		}, nil
	}

	for _, ally := range hero.party {
		if ally.spec().Name == member {
			return sheet{
				label: member, weapon: &ally.weapon, armor: &ally.armor, charm: &ally.charm, bag: &ally.bag, spells: &ally.spells,
				hp: &ally.hp, mana: &ally.mana, level: &ally.level, xp: &ally.xp, abilities: &ally.abilities, maxHP: ally.maxHealth, maxMana: ally.maxEnergy,
			}, nil
		}
	}

	return sheet{}, errNoMember
}

// apply makes the change and words what it did.
func (r *Realm) apply(sheet sheet, change Change) (string, error) {
	switch change.Op {
	case "stat":
		return r.setStat(sheet, change.Field, change.Value)
	case "equip":
		return r.wear(sheet, change.Field, change.Item)
	case "give", "teach":
		item, ok := r.item(change.Item)
		if !ok {
			return "", errNoItem
		}

		if change.Op == "teach" {
			if item.Kind != world.KindSpell {
				return "", errors.New("uczyć można tylko zaklęć")
			}

			*sheet.spells = append(*sheet.spells, item)

			return "zna teraz: " + item.Name, nil
		}

		*sheet.bag = append(*sheet.bag, item)

		return "dostaje: " + item.Name, nil
	case "take":
		list := sheet.bag
		if change.Field == "spells" {
			list = sheet.spells
		}

		if change.Value < 0 || change.Value >= len(*list) {
			return "", errNoItem
		}

		item := (*list)[change.Value]
		*list = slices.Delete(*list, change.Value, change.Value+1)

		return "traci: " + item.Name, nil
	}

	return "", errField
}

func (r *Realm) setStat(sheet sheet, field string, value int) (string, error) {
	if value < 0 {
		return "", errors.New("wartość nie może być ujemna")
	}

	switch field {
	case "hp":
		*sheet.hp = min(value, sheet.maxHP())
	case "mana":
		*sheet.mana = min(value, sheet.maxMana())
	case "level":
		*sheet.level = max(value, 1)
	case "xp":
		*sheet.xp = value
	case "gold":
		if sheet.gold == nil {
			return "", errors.New("towarzysz nie ma sakiewki")
		}

		*sheet.gold = value
	default:
		index := slices.Index(abilityFields[:], field)
		if index < 0 {
			return "", errField
		}

		sheet.abilities[index] = min(max(value, 1), maxAbility)
		value = sheet.abilities[index]
	}

	return field + " = " + strconv.Itoa(value), nil
}

// wear puts an item in a slot, whatever it was before; an empty ID takes
// the slot off. The item does not have to be in the bag.
func (r *Realm) wear(sheet sheet, field, id string) (string, error) {
	slots := map[string]struct {
		slot **world.ItemSpec
		kind world.Kind
	}{
		"weapon": {sheet.weapon, world.KindWeapon},
		"armor":  {sheet.armor, world.KindArmor},
		"charm":  {sheet.charm, world.KindCharm},
	}

	slot, ok := slots[field]
	if !ok {
		return "", errField
	}

	if id == "" {
		*slot.slot = nil

		return "zdejmuje " + field, nil
	}

	item, found := r.item(id)
	if !found {
		return "", errNoItem
	}

	if item.Kind != slot.kind {
		return "", fmt.Errorf("%s nie pasuje do tego miejsca", item.Name)
	}

	*slot.slot = &item

	return "nosi: " + item.Name, nil
}

// Craft makes a new item by hand: it joins the world, so that it can be
// given, taught and saved, but no shop sells it.
func (r *Realm) Craft(item world.ItemSpec) (world.ItemSpec, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := world.CleanItem(item)
	if !ok || item.Kind == world.KindThing {
		return world.ItemSpec{}, errors.New("przedmiot potrzebuje nazwy i znanego rodzaju")
	}

	if r.known(item.Name) {
		return world.ItemSpec{}, errors.New("przedmiot o takiej nazwie już jest")
	}

	item.ID = fmt.Sprintf("made-%d", len(r.extras)+1)
	r.extras = append(r.extras, extra{shop: -1, item: item})
	r.trace(DescribeItem, item.ID, Trace{How: HowAdded})
	r.notify()

	return item, nil
}
