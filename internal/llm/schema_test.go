package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
)

// The server makes the model write the fields in the order the schema lists
// them, so the Mind's schemas must keep the order it was taught: the
// decision first.
func TestMindSchemasKeepTheirOrder(t *testing.T) {
	cases := map[string]struct {
		schema map[string]any
		order  []string
	}{
		"intent":    {intentSchema(nil), []string{"action", "item_id", "amount", "persuasion"}},
		"invent":    {inventSchema(), []string{"offered", "name", "description", "kind", "effect", "tier"}},
		"note":      {noteSchema(), []string{"found", "subject", "fact", "quote"}},
		"questhook": {questHookSchema(game.QuestHookRequest{}), []string{"found", "person", "relation", "monster", "place", "quote"}},
	}

	for name, c := range cases {
		data, err := json.Marshal(c.schema)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		text := string(data)
		properties := text[strings.Index(text, `"properties":`):]
		last := -1

		for _, key := range c.order {
			at := strings.Index(properties, `"`+key+`":`)
			if at < 0 || at < last {
				t.Errorf("%s: %q out of order in %s", name, key, properties)
			}

			last = at
		}

		var back map[string]any
		if err := json.Unmarshal(data, &back); err != nil {
			t.Errorf("%s: schema is not valid JSON: %v", name, err)
		}
	}
}
