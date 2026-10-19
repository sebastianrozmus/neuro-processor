package speech

import "strings"

// masculineInA are men's names that end in a, as women's names do.
var masculineInA = []string{"kuba", "barnaba", "bonawentura", "kosma", "jarema", "zawisza", "sasza", "misza", "grisza", "kosta", "nikita", "dyzma", "boryna"}

// Feminine reports whether a character's name sounds like a woman's: its
// first word, a name or a title, ends in a in Polish ("Jagna", "Elira
// Cieniokrąg", "Siostra Agnes"), unless it is one of the men's names that
// do too.
func Feminine(name string) bool {
	words := strings.Fields(strings.ToLower(name))
	if len(words) == 0 {
		return false
	}

	first := strings.Trim(words[0], ".,!?;:")

	return strings.HasSuffix(first, "a") && !contains(masculineInA, first)
}
