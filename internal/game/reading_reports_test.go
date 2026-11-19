package game

import (
	"strings"
	"testing"
)

// The cases come from the talk packs of run 14: lines the rules read
// otherwise than the people who wrote them meant.
func TestInsultsFromRun14(t *testing.T) {
	for _, text := range []string{
		"ty stary chuju", "ty stary skurwielu, za drogo to wszystko", "ty grubasie, zejdź mi z drogi", "ty stary pierniku wiesz w ogóle cokolwiek",
		"ty stary zdzierco", "hej frajerze", "podpalaczko", "to twój towar to i tak śmieci", "babo z toporem",
	} {
		if !Read(InterpretRequest{Text: text}).rude {
			t.Errorf("%q is no insult", text)
		}
	}

	for _, text := range []string{"wybacz, głupio mi", "przepraszam, głupio wyszło", "jesteś czarownica?", "ty jesteś czarownica?", "ale śmiech"} {
		if Read(InterpretRequest{Text: text}).rude {
			t.Errorf("%q is an insult", text)
		}
	}
}

func TestThreatsFromRun14(t *testing.T) {
	for _, text := range []string{
		"jak mnie oszukasz, to cię zastrzelę", "albo dasz mi topór albo dostaniesz w zęby", "spokojnie babka, bo ci tę chatę spalę",
		"oddaj złoto albo pogadamy inaczej", "a jak cię zbiję to oddasz towar", "a jak cię walnę i zabiorę klucze do magazynu?",
	} {
		if !threatening(Fold(text)) {
			t.Errorf("%q is no threat", text)
		}
	}

	for _, text := range []string{"zabijemy go razem", "spałem w karczmie", "zastrzelę tego dzika"} {
		if threatening(Fold(text)) {
			t.Errorf("%q is a threat", text)
		}
	}
}

func TestGoodbyesFromRun14(t *testing.T) {
	for _, text := range []string{
		"nara", "pa", "dobranoc mira", "dzięki, trzymaj się", "ok, trzymaj się", "haha dobra, pa", "no trudno, cześć", "spoko, nara",
		"no to nara, cześć", "nie było tematu. idę", "no dobra, nie to nie, hej", "trzymaj się babciu", "idę się przespać",
	} {
		if got := Read(InterpretRequest{Text: text}).Intent.Action; got != ActionLeave {
			t.Errorf("%q = %s, want leave", text, got)
		}
	}

	for _, text := range []string{"cześć", "no hej", "cześć, masz miecz?", "siema mordo"} {
		if got := Read(InterpretRequest{Text: text}).Intent.Action; got == ActionLeave {
			t.Errorf("%q is a goodbye", text)
		}
	}
}

func TestAskAfterApology(t *testing.T) {
	if got := Read(InterpretRequest{Text: "ok ok, przepraszam. ile bierzesz"}).Intent.Action; got != ActionAsk {
		t.Errorf("an apology with a question = %s, want ask", got)
	}

	if greeting([]string{"no"}) {
		t.Error("a bare „no” is a greeting")
	}
}

func TestWordsThatNameNoWare(t *testing.T) {
	wares := []Ware{
		{ID: "book", Name: "Księga Wielkiej Fali"}, {ID: "drone", Name: "Dron Sprzątający"}, {ID: "pay", Name: "Żołd dzienny"},
		{ID: "mail", Name: "Kolczuga"}, {ID: "coat", Name: "Kurtka z Opon"},
	}

	for text, want := range map[string]string{
		"sprzedasz mi księżyc?": "", "co masz na sprzedaż": "", "dzień dobry": "", "dzień dobry pani": "",
		"pokaż kolczugę": "mail", "co z tą kolczugą": "mail", "5 kurtek": "coat", "księgę poproszę": "book", "ile za żołd": "pay",
	} {
		if got := Read(InterpretRequest{Text: text, Wares: wares}).Intent.ItemID; got != want {
			t.Errorf("%q names %q, want %q", text, got, want)
		}
	}

	if quantity, _ := quantityIn(strings.Fields(Fold("a za 50 dziennie?")), wares, "kapsle"); quantity != 0 {
		t.Errorf("a daily wage of 50 counts %d pieces", quantity)
	}
}
