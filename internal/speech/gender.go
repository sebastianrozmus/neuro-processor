package speech

import "strings"

// masculineInA are men's names that end in a, as women's names do.
var masculineInA = []string{"kuba", "barnaba", "bonawentura", "kosma", "jarema", "zawisza", "sasza", "misza", "grisza", "kosta", "nikita", "dyzma", "boryna"}

// Titles tell more than the name after them: "Pani Kaliber" is a woman and
// "Brat Jaromir" a man, while "Doktor Lena" is whatever Lena is.
var (
	feminineTitles  = []string{"pani", "siostra", "matka", "ciotka", "babka", "lady", "madame", "panna"}
	masculineTitles = []string{"pan", "brat", "sir", "ojciec", "wuj", "wujek", "dziadek", "mistrz"}
	neutralTitles   = []string{"doktor", "dr", "doktorka", "sierżant", "kapitan", "kapral", "porucznik", "profesor", "inżynier", "medyk", "kwatermistrz", "sanitariusz", "żerca", "kapłan"}
)

// Feminine reports whether a character's name sounds like a woman's: its
// first word, a name or a title, ends in a in Polish ("Jagna", "Elira
// Cieniokrąg", "Siostra Agnes"), unless it is one of the men's names that
// do too. A title that says nothing of who bears it is passed over.
func Feminine(name string) bool {
	words := strings.Fields(strings.ToLower(name))

	for index, word := range words {
		word = strings.Trim(word, ".,!?;:")

		switch {
		case contains(feminineTitles, word):
			return true
		case contains(masculineTitles, word):
			return false
		case contains(neutralTitles, word) && index+1 < len(words):
			continue
		}

		return strings.HasSuffix(word, "a") && !contains(masculineInA, word)
	}

	return false
}
