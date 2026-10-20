package game

import (
	"fmt"
	"slices"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Artifacts are the legendary pieces of a world. No shop sells one: each
// boss that falls leaves one behind, picked at random among those nobody
// has had yet, so that the first ones won are all different.

// artifact picks the piece a fallen boss leaves, and remembers it was given.
func (r *Realm) artifact() (world.ItemSpec, bool) {
	all := r.world.Artifacts
	if len(all) == 0 {
		return world.ItemSpec{}, false
	}

	var fresh []world.ItemSpec

	for _, item := range all {
		if !slices.Contains(r.dropped, item.ID) {
			fresh = append(fresh, item)
		}
	}

	// Once every one is out, they come round again.
	if len(fresh) == 0 {
		fresh = all
	}

	item := fresh[r.rng.Intn(len(fresh))]
	if !slices.Contains(r.dropped, item.ID) {
		r.dropped = append(r.dropped, item.ID)
	}

	return item, true
}

// dropArtifact lays a fallen boss's artifact on the battlefield.
func (r *Realm) dropArtifact(current *fight, index int, boss string) {
	item, ok := r.artifact()
	if !ok {
		return
	}

	r.ground[index] = append(r.ground[index], item)
	r.tellFight(current, fmt.Sprintf("%s zostawia artefakt: %s. %s", boss, item.Name, Effect(item)))
}
