package game

import (
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Two monsters of one name on a tile are numbered, the numbers stay while
// the fight goes on and go when the tile is calm again.
func TestTwinMonstersAreNumbered(t *testing.T) {
	realm := testRealm(t)
	pos := world.Point{X: 1, Y: 1}
	index := realm.world.Index(pos)

	first, second := rat(1), rat(2)
	other := &monster{id: 3, name: "Błędny Ognik", hp: 10, maxHP: 10}
	realm.monsters[index] = []*monster{first, other, second}

	if got, want := realm.foeName(pos, second), second.name+" 2"; got != want {
		t.Errorf("second twin = %q, want %q", got, want)
	}

	if got, want := realm.foeName(pos, first), first.name+" 1"; got != want {
		t.Errorf("first twin = %q, want %q", got, want)
	}

	if got := realm.foeName(pos, other); got != other.name {
		t.Errorf("a monster without a twin = %q, want %q", got, other.name)
	}

	// The first falls mid-fight: the second keeps its number.
	realm.fights[pos] = &fight{pos: pos}
	first.hp = 0

	if got, want := realm.foeName(pos, second), second.name+" 2"; got != want {
		t.Errorf("survivor mid-fight = %q, want %q", got, want)
	}

	// The fight is over: the survivor is alone and plain again.
	delete(realm.fights, pos)

	if got := realm.foeName(pos, second); got != second.name {
		t.Errorf("survivor after the fight = %q, want %q", got, second.name)
	}

	// Two that wandered in with the same number are told apart again.
	third := rat(4)
	third.mark, second.mark = 1, 1
	realm.monsters[index] = append(realm.monsters[index], third)

	if a, b := realm.foeName(pos, second), realm.foeName(pos, third); a == b {
		t.Errorf("clashing numbers stay: %q and %q", a, b)
	}
}

// The last round of a won fight is told with the fallen still on the
// master's card, under the numbers they fell with.
func TestVictoryLoreListsTheFallen(t *testing.T) {
	realm := testRealm(t)
	pos := world.Point{X: 1, Y: 1}
	first, second := rat(1), rat(2)
	realm.monsters[realm.world.Index(pos)] = []*monster{first, second}
	realm.foeName(pos, first)
	first.hp, second.hp = 0, 0

	current := &fight{pos: pos, beaten: realm.monsters[realm.world.Index(pos)]}
	realm.monsters[realm.world.Index(pos)] = nil
	lore := realm.sceneLore(current, nil)

	for _, want := range []string{"- Szczur Kablowy 1 (", "- Szczur Kablowy 2 (", "pokonany"} {
		if !strings.Contains(lore, want) {
			t.Errorf("lore misses %q:\n%s", want, lore)
		}
	}
}

// "atakuj szczur 2" picks the second rat.
func TestTargetByNumber(t *testing.T) {
	realm := testRealm(t)
	pos := world.Point{X: 1, Y: 1}
	first, second := rat(1), rat(2)
	realm.monsters[realm.world.Index(pos)] = []*monster{first, second}
	realm.foeName(pos, first)

	current := &fight{pos: pos}
	hero := &player{name: "Ala"}

	if got := realm.target(hero, current, first.name+" 2", 0); got != second {
		t.Errorf("target %q = %v, want the second twin", first.name+" 2", got)
	}

	if got := realm.target(hero, current, first.name, 0); got != first {
		t.Errorf("target by name alone = %v, want the first", got)
	}
}

func rat(id int) *monster {
	return &monster{id: id, name: "Szczur Kablowy", hp: 10, maxHP: 10}
}
