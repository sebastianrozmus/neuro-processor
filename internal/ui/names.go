package ui

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Names of the world stand out in the transcript, each sort in a colour of
// its own: things to buy, people, places and spells.
var (
	itemName   = lipgloss.NewStyle().Foreground(lipgloss.Color("209"))
	personName = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	placeName  = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	spellName  = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

// namer finds the names of a world in a line of text, in any of their
// Polish forms ("Smoczej Góry" is "Smocza Góra"), and paints them.
type namer struct {
	names []name
}

type name struct {
	// words are the folded words of the name; a short one ("pod", "z")
	// must be met whole, a longer one by its stem.
	words []string
	style lipgloss.Style
}

func newNamer(w *world.World) *namer {
	n := &namer{}
	if w == nil {
		return n
	}

	add := func(style lipgloss.Style, names ...string) {
		for _, text := range names {
			if words := strings.Fields(game.Fold(text)); len(words) > 0 {
				n.names = append(n.names, name{words: words, style: style})
			}
		}
	}

	for _, item := range w.Items {
		if item.Kind == world.KindSpell {
			add(spellName, item.Name)
		} else {
			add(itemName, item.Name)
		}
	}

	for _, shop := range w.Shops {
		add(personName, shop.Keeper)
		add(placeName, shop.Name)
	}

	for _, companion := range w.Companions {
		add(personName, companion.Name)
	}

	for _, person := range w.People {
		add(personName, person.Name)
	}

	add(personName, w.Boss.Name)

	for _, place := range w.Places {
		add(placeName, place.Name)
	}

	for _, site := range w.Sites {
		add(placeName, site.Name)
	}

	for _, room := range w.Rooms {
		add(placeName, room.Name)
	}

	add(placeName, w.Name)

	// The longest names first, so that "Kopalnia Durgara" is not cut
	// short by a "Durgar".
	sort.SliceStable(n.names, func(a, b int) bool { return len(n.names[a].words) > len(n.names[b].words) })

	return n
}

// span is a word of the text: where it is and its folded form.
type span struct {
	start, end int
	folded     string
	upper      bool
}

func spans(text string) []span {
	var found []span

	start := -1
	for index, char := range text + " " {
		letter := unicode.IsLetter(char) || unicode.IsDigit(char)
		switch {
		case letter && start < 0:
			start = index
		case !letter && start >= 0:
			word := text[start:index]
			first, _ := utf8.DecodeRuneInString(word)
			found = append(found, span{start: start, end: index, folded: strings.TrimSpace(game.Fold(word)), upper: unicode.IsUpper(first)})
			start = -1
		}
	}

	return found
}

// paint colours every name in a line of plain text. A name counts only
// when it begins with a capital letter, as names do, so that the berries a
// keeper turns away are not mistaken for the "Jagody" of another world.
func (n *namer) paint(text string) string {
	if n == nil || len(n.names) == 0 || strings.Contains(text, "\x1b") {
		return text
	}

	words := spans(text)
	var out strings.Builder
	last := 0

	for index := 0; index < len(words); {
		matched := 0
		var style lipgloss.Style

		if words[index].upper {
			for _, candidate := range n.names {
				if fits(words[index:], candidate.words) {
					matched, style = len(candidate.words), candidate.style
					break
				}
			}
		}

		if matched == 0 {
			index++
			continue
		}

		from, to := words[index].start, words[index+matched-1].end
		out.WriteString(text[last:from])
		out.WriteString(style.Render(text[from:to]))
		last = to
		index += matched
	}

	out.WriteString(text[last:])

	return out.String()
}

// fits reports whether the words of the text begin with the words of a
// name, allowing for declension.
func fits(words []span, nameWords []string) bool {
	if len(words) < len(nameWords) {
		return false
	}

	for index, want := range nameWords {
		got := words[index].folded
		if len(want) < 4 {
			if got != want {
				return false
			}

			continue
		}

		if !strings.HasPrefix(got, game.Stem(want)) || len(got) > len(want)+4 {
			return false
		}
	}

	return true
}
