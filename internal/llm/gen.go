package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const worldInstructions = `Projektujesz świat do tekstowej gry RPG w terminalu. Na podstawie opisu gracza zwracasz obiekt JSON zgodny ze schematem. ` +
	`Wszystkie teksty piszesz po polsku. Nie podajesz żadnych liczb poza polami danger i tier: statystyki, ceny i mapę wylicza gra.

Wymagania:
- name: nazwa świata, najwyżej 4 słowa.
- intro: 2-3 zdania wprowadzenia, które mówią, co się dzieje i o co toczy się gra.
- currency: nazwa waluty w formie, która pasuje po liczbie, np. "szt. złota", "kredytów", "kapsli".
- places: 7 do 9 rodzajów miejsc. Dwa mają danger 0: to bezpieczne osady (pierwsza jest startowa). ` +
	`Pozostałe mają danger 1, 2 lub 3 i każdy z tych poziomów występuje co najmniej raz. terrain decyduje o rysunku miejsca. Opis to 1-2 zdania.
- sites: 3 lub 4 miejsca szczególne na pojedynczych polach mapy. kind "treasure" to strzeżona skrytka z cennym przedmiotem, ` +
	`kind "shrine" to miejsce, które przywraca siły. Co najmniej jedno każdego rodzaju.
- monsters: 6 rodzajów przeciwników, po dwa na każdy tier 1, 2 i 3 (3 to najgroźniejsi). shape decyduje o rysunku: ` +
	`beast (zwierzę), undead (nieumarły), humanoid (człowiek), spirit (duch), serpent (wąż), insect (owad lub rój), machine (maszyna), blob (galareta). Opis to jedno zdanie.
- boss: główny przeciwnik, którego pokonanie kończy grę.
- items: dokładnie 17 przedmiotów. Opis to jedno zdanie bez liczb.
  - 3 x kind "weapon" (tier 1, 2, 3) i 3 x kind "armor" (tier 1, 2, 3), effect "none".
  - 3 x kind "potion": effect "heal" tier 1 i 2 oraz effect "mana" tier 1 (przywraca moc).
  - 4 x kind "spell": rzecz, z której bohater uczy się mocy rzucanej w walce (zwój, księga, schemat, program). ` +
	`effect "bolt" (rani jednego wroga) tier 1 i 3, "blast" (rani wszystkich) tier 2, "heal" (leczy drużynę) tier 2.
  - 4 x kind "charm": noszony przedmiot magiczny lub gadżet. effect "ward" (obrona) tier 1, "power" (atak) tier 2, "focus" (moc) tier 2, "vigor" (zdrowie) tier 3.
- companions: 4 towarzysze do wynajęcia w karczmach osad: role "fighter" (wojownik), "healer" (uzdrowiciel) i "mage" (rzuca moce), ` +
	`tier 1-3 (koszt i siła). name to imię lub przydomek, description to 1-2 zdania o charakterze. ` +
	`Dokładnie jeden z nich ma legend true: to słynny najemnik najwyższej klasy (tier 3), który nie bierze zapłaty z góry, tylko żołd za każdy dzień.
- shops: dokładnie 3 budynki, które stoją w każdej osadzie: kind "armory" (broń i pancerze), "alchemist" (środki lecznicze), "magic" (moce i przedmioty magiczne). ` +
	`name to nazwa budynku, keeper to imię gospodarza, persona to 2-3 zdania o jego charakterze i sposobie mówienia, greeting to jego pierwsza kwestia.
- Nazwy własne są krótkie (najwyżej 4 słowa) i spójne z klimatem opisu. W świecie bez magii moce i przedmioty magiczne są techniką, wynalazkami albo umiejętnościami.`

// GenerateWorld asks the model to design a world around the given theme,
// to be played in the given mode. The model only names and describes
// things; the result is cleaned and checked before it is returned.
func (c *Client) GenerateWorld(ctx context.Context, mode world.Mode, theme string) (world.Spec, error) {
	instructions, schema := worldInstructions, worldSchema()
	if mode == world.ModeAdventure {
		instructions, schema = adventureInstructions, adventureSchema()
	}

	content, err := c.chat(ctx, chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: instructions},
			{Role: "user", Content: "Opis świata od gracza:\n" + theme},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "world",
				"strict": true,
				"schema": schema,
			},
		},
	})
	if err != nil {
		return world.Spec{}, err
	}

	var spec world.Spec
	if err := json.Unmarshal([]byte(extractJSON(content)), &spec); err != nil {
		return world.Spec{}, fmt.Errorf("model returned an invalid world: %w", err)
	}

	// Backdrops are embedded assets, which a generated world cannot have.
	spec.Backdrops = nil
	spec.Mode = mode

	if err := spec.Normalize(); err != nil {
		return world.Spec{}, fmt.Errorf("model returned an unplayable world: %w", err)
	}

	return spec, nil
}

func worldSchema() map[string]any {
	text := map[string]any{"type": "string"}
	tier := map[string]any{"type": "integer", "enum": []int{1, 2, 3}}
	shape := map[string]any{"type": "string", "enum": world.Shapes}

	return object(map[string]any{
		"name":     text,
		"intro":    text,
		"currency": text,
		"places": list(object(map[string]any{
			"name":        text,
			"description": text,
			"terrain":     map[string]any{"type": "string", "enum": world.Terrains},
			"danger":      map[string]any{"type": "integer", "enum": []int{0, 1, 2, 3}},
		})),
		"sites": list(object(map[string]any{
			"name":        text,
			"description": text,
			"kind":        map[string]any{"type": "string", "enum": world.SiteKinds},
		})),
		"monsters": list(object(map[string]any{"name": text, "description": text, "shape": shape, "tier": tier})),
		"boss":     object(map[string]any{"name": text, "description": text, "shape": shape}),
		"items": list(object(map[string]any{
			"name":        text,
			"description": text,
			"kind":        map[string]any{"type": "string", "enum": world.Kinds},
			"effect":      map[string]any{"type": "string", "enum": world.Effects},
			"tier":        tier,
		})),
		"companions": list(object(map[string]any{
			"name":        text,
			"description": text,
			"role":        map[string]any{"type": "string", "enum": world.Roles},
			"tier":        tier,
			"legend":      map[string]any{"type": "boolean"},
		})),
		"shops": list(object(map[string]any{
			"name":     text,
			"kind":     map[string]any{"type": "string", "enum": world.ShopKinds},
			"keeper":   text,
			"persona":  text,
			"greeting": text,
		})),
	})
}

// object builds a strict JSON schema object: every property is required.
func object(properties map[string]any) map[string]any {
	required := make([]string, 0, len(properties))
	for name := range properties {
		required = append(required, name)
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func list(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

const adventureInstructions = `Projektujesz tekstową grę przygodową w stylu Mózgoprocesora: pomieszczenia, przedmioty, zagadki i ludzie, bez walki. ` +
	`Na podstawie opisu gracza zwracasz obiekt JSON zgodny ze schematem. Wszystkie teksty po polsku. Gra sama sprawdza, co gracz robi; ty podajesz tylko treść i reguły zagadek.

Wymagania:
- name: nazwa gry, najwyżej 4 słowa. intro: 2-3 zdania, jak zaczyna się przygoda. goal: jedno zdanie, co trzeba osiągnąć.
- rooms: 8 do 12 pomieszczeń. Każde ma name, description (2-3 zdania, co widać), terrain (rysunek: station lub street dla wnętrz i miasta, cave dla szybów i piwnic, ` +
	`forest, plains, mountain, swamp, desert, ruins, snow, wasteland, water dla otwartego terenu) oraz x (0-14) i y (0-8): położenie na mapie. ` +
	`Przejście jest możliwe tylko między pomieszczeniami stojącymi obok siebie (różnica 1 w x albo w y), więc układaj je w spójną sieć. Pierwsze pomieszczenie to start. ` +
	`things to nazwy rzeczy, które tam leżą na początku. doors to zamknięte wyjścia: dir (n, e, s, w), text (co widzi gracz, gdy próbuje przejść) i flag (nazwa flagi, która je otwiera).
- things: 10 do 16 rzeczy. fixed true dla tych, których nie da się zabrać (terminal, łóżko, automat). description to 1-2 zdania.
- people: 2 do 4 osoby. room to nazwa pomieszczenia, w którym stoją, talk to, co mówią, gdy nie ma nic ważniejszego, description to 1-2 zdania.
- steps: 10 do 20 kroków fabuły, czyli reguł zagadek. verb: "use" (użyj rzeczy, także "użyj X na Y": thing to X, on to Y; samo "użyj Y": thing puste), ` +
	`"give" (daj thing osobie on), "talk" (rozmowa z osobą on), "open" (otwórz on), "examine" (zbadaj on), "enter" (wejście do pomieszczenia on). ` +
	`room ogranicza krok do pomieszczenia; needs to flaga, bez której krok nie działa; unless to flaga, która go blokuje; once true dla kroków jednorazowych. ` +
	`text to odpowiedź gry. Skutki: flag ustawia flagę (tą samą nazwą otwierasz drzwi), gives daje rzecz do plecaka, takes true zużywa użytą rzecz, reveals sprawia, że rzecz pojawia się w pomieszczeniu, ` +
	`changes zmienia pomieszczenie (room, description, opcjonalnie terrain), win true kończy grę zwycięstwem.
- Zagadki układają się w łańcuch 5-8 kroków od startu do win: każda rzecz potrzebna do kroku musi być do zdobycia (leży gdzieś, ktoś ją daje albo pojawia się po zbadaniu). ` +
	`Dokładnie jeden krok ma win true i wymaga wcześniejszych flag. Nazwy rzeczy i osób w steps muszą być dokładnie takie jak w things i people, a nazwy pomieszczeń jak w rooms.
- Każda osoba ma co najmniej jeden krok talk. Co najmniej jedno pomieszczenie zmienia się w trakcie gry (changes).`

func adventureSchema() map[string]any {
	text := map[string]any{"type": "string"}
	flag := map[string]any{"type": "boolean"}
	terrain := map[string]any{"type": "string", "enum": world.Terrains}

	return object(map[string]any{
		"name":  text,
		"intro": text,
		"goal":  text,
		"rooms": list(object(map[string]any{
			"name":        text,
			"description": text,
			"terrain":     terrain,
			"x":           map[string]any{"type": "integer"},
			"y":           map[string]any{"type": "integer"},
			"things":      list(text),
			"doors": list(object(map[string]any{
				"dir":  map[string]any{"type": "string", "enum": world.Directions},
				"text": text,
				"flag": text,
			})),
		})),
		"things": list(object(map[string]any{"name": text, "description": text, "fixed": flag})),
		"people": list(object(map[string]any{"name": text, "description": text, "room": text, "talk": text})),
		"steps": list(object(map[string]any{
			"verb":    map[string]any{"type": "string", "enum": world.StepVerbs},
			"thing":   text,
			"on":      text,
			"room":    text,
			"needs":   text,
			"unless":  text,
			"once":    flag,
			"text":    text,
			"flag":    text,
			"gives":   text,
			"takes":   flag,
			"reveals": text,
			"changes": object(map[string]any{"room": text, "description": text, "terrain": text}),
			"win":     flag,
		})),
	})
}
