package game

import (
	"strings"
	"testing"
)

// A fact that only retells the card in other words is the card: on SN23 the
// Mind noted such paraphrases and the engine let all of them through.
func TestCardSaysRetold(t *testing.T) {
	eufemia := "Zielarka Eufemia, Izba Zielarki. Eufemia to czterdziestoletnia zielarka, która przyszła z miasta nad rzeką i nauczyła się ziół u siostry Scholastyki w klasztorze. Nie wyszła za mąż. Jej brat Józef jest księdzem w sąsiedniej parafii."
	hipolit := "Hipolit Prasa, Kuźnia przy Tłoczni. Hipolit to pięćdziesięciopięcioletni kowal. Jego żona Marianna zmarła na febrę, a bliźniacy Jaś i Staś pracują z nim przy tłoczni."
	seweryn := "Przewodnik Seweryn, wojownik. Trzydziestopięcioletni przewodnik z blizną na przedramieniu od pękniętej liny. Fachu nauczył go ojciec, też przewodnik."

	cases := []struct {
		fact, card, speaker string
		retold              bool
	}{
		{"Eufemia pochodzi z miasta nad rzeką.", eufemia, "Zielarka Eufemia", true},
		{"Eufemię uczyła siostra Scholastyka w klasztorze.", eufemia, "Zielarka Eufemia", true},
		{"Marianna, żona Hipolita, nie żyje.", hipolit, "Hipolit Prasa", true},
		{"Hipolit ma dwóch synów, bliźniaków, którzy pracują z nim przy tłoczni.", hipolit, "Hipolit Prasa", true},
		{"Seweryn ma bliznę na przedramieniu od pękniętej liny na żlebie.", seweryn, "Przewodnik Seweryn", true},
		// New: life where the card tells of none, a new name, new words.
		{"Ojciec Seweryna żyje.", seweryn, "Przewodnik Seweryn", false},
		{"Seweryna wyciągnął ojciec, gdy pękła lina.", seweryn, "Przewodnik Seweryn", false},
		{"Hipolit ma kuźnię po ojcu.", hipolit, "Hipolit Prasa", false},
		{"Eufemia ma siostrzenicę Agatę w mieście.", eufemia, "Zielarka Eufemia", false},
	}

	for _, c := range cases {
		if got := cardSays(c.fact, c.card, strings.Fields(Fold(c.speaker))); got != c.retold {
			t.Errorf("cardSays(%q) = %v, want %v", c.fact, got, c.retold)
		}
	}
}
