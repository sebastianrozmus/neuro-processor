package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const merchantPatience = 6

// MerchantFor builds the haggling keeper of one of the world's shops. A
// shop sells the kinds of items it specialises in, up to the given tier;
// prices come from item tiers and potions never run out.
func MerchantFor(spec world.Spec, shop, maxTier int) Merchant {
	keeper := spec.Shops[shop]
	merchant := Merchant{
		Name:     keeper.Keeper,
		Persona:  keeper.Persona,
		Greeting: keeper.Greeting,
		Patience: merchantPatience,
		Currency: spec.Currency,
	}

	for _, kind := range world.Kinds {
		if spec.ShopFor(kind) == shop {
			merchant.Kinds = append(merchant.Kinds, kind)
		}
	}

	for _, item := range spec.Items {
		if spec.ShopFor(item.Kind) == shop && item.Tier <= maxTier {
			merchant.Items = append(merchant.Items, ware(item))
		}
	}

	return merchant
}

// ware turns a world item into something a merchant haggles over.
func ware(item world.ItemSpec) Item {
	return Item{
		ID:          item.ID,
		Name:        item.Name,
		Kind:        item.Kind,
		Description: strings.TrimSpace(item.Description + " " + Effect(item)),
		Aliases:     Aliases(item.Name),
		ListPrice:   item.Price(),
		MinPrice:    item.MinPrice(),
		Unlimited:   item.Kind == world.KindPotion,
	}
}

// Aliases derives the folded word stems by which an item can be recognised
// in folded player text.
func Aliases(name string) []string {
	var aliases []string

	for _, word := range strings.Fields(Fold(name)) {
		if len(word) >= minQueryWord {
			aliases = append(aliases, Stem(word))
		}
	}

	return aliases
}

// Effect words what an item does.
func Effect(item world.ItemSpec) string {
	power := item.Power()

	switch item.Kind {
	case world.KindWeapon:
		return fmt.Sprintf("Broń, atak +%d.", power)
	case world.KindArmor:
		return fmt.Sprintf("Pancerz, obrona +%d.", power)
	case world.KindPotion:
		if item.Effect == world.EffectMana {
			return fmt.Sprintf("Przywraca %d mocy.", power)
		}

		return fmt.Sprintf("Leczy %d zdrowia.", power)
	case world.KindSpell:
		return "Do nauczenia. " + SpellEffect(item)
	}

	switch item.Effect {
	case world.EffectWard:
		return fmt.Sprintf("Noszony: obrona +%d.", power)
	case world.EffectVigor:
		return fmt.Sprintf("Noszony: zdrowie +%d.", power)
	case world.EffectFocus:
		return fmt.Sprintf("Noszony: moc +%d.", power)
	}

	return fmt.Sprintf("Noszony: atak +%d.", power)
}

// SpellEffect words what casting a spell does and what it costs.
func SpellEffect(spell world.ItemSpec) string {
	switch spell.Effect {
	case world.EffectBlast:
		return fmt.Sprintf("Rani wszystkich wrogów za %d, koszt %d mocy.", spell.Power(), spell.Cost())
	case world.EffectHeal:
		return fmt.Sprintf("Leczy drużynę o %d, koszt %d mocy.", spell.Power(), spell.Cost())
	}

	return fmt.Sprintf("Rani jednego wroga za %d, koszt %d mocy.", spell.Power(), spell.Cost())
}
