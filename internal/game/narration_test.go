package game

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type scriptedNarrator struct {
	replies []string
	err     error
	calls   int
	// corrections records the correction each call came with.
	corrections []string
}

func (n *scriptedNarrator) Narrate(_ context.Context, request NarrateRequest) (string, error) {
	n.calls++
	n.corrections = append(n.corrections, request.Correction)

	if n.err != nil {
		return "", n.err
	}

	return n.replies[min(n.calls, len(n.replies))-1], nil
}

func counterRequest() NarrateRequest {
	return NarrateRequest{Decision: Decision{
		Outcome: OutcomeCounter,
		Item:    Ware{Name: "Amulet"},
		Offer:   80,
		Price:   140,
	}}
}

func TestNumbersOK(t *testing.T) {
	tests := []struct {
		text    string
		allowed []int
		want    bool
	}{
		{"Za 80? Daj 140.", []int{80, 140}, true},
		{"Zgoda, 80 i jest twój... albo 120.", []int{80, 140}, false},
		{"Bez żadnych kwot.", nil, true},
		{"Znam go od 3 lat.", nil, false},
		{"Niech będzie pięćdziesiąt.", []int{50}, false},
		{"Sześćdziesięciu złotych nie opuszczę.", []int{60}, false},
		{"Dwanaście lat tu siedzę.", nil, false},
		{"Za sto kapsli jest twoja.", []int{100}, false},
		{"Dwa razy ci mówiłam, pięć minut i po sprawie.", nil, true},
	}

	for _, test := range tests {
		if got := NumbersOK(test.text, test.allowed); got != test.want {
			t.Errorf("NumbersOK(%q) = %v, want %v", test.text, got, test.want)
		}
	}
}

func TestNarrateRetriesThenFallsBack(t *testing.T) {
	request := counterRequest()

	narrator := &scriptedNarrator{replies: []string{"  «Zgoda, 80!»  "}}
	if text, err := Narrate(context.Background(), narrator, request); err != nil || text != "Zgoda, 80!" {
		t.Errorf("consistent reply: got %q, %v", text, err)
	}

	narrator = &scriptedNarrator{replies: []string{"Oddam za 99.", "Daj 140."}}
	if text, err := Narrate(context.Background(), narrator, request); err != nil || text != "Daj 140." {
		t.Errorf("retry: got %q, %v", text, err)
	}

	if narrator.corrections[0] != "" || !strings.Contains(narrator.corrections[1], "tylko cyframi") {
		t.Errorf("the second try was not told what went wrong: %q", narrator.corrections)
	}

	narrator = &scriptedNarrator{replies: []string{"Witaj. [GRA] Gracz zaproponował 80.", "Daj 140."}}
	if text, err := Narrate(context.Background(), narrator, request); err != nil || text != "Daj 140." {
		t.Errorf("a leaked note: got %q, %v", text, err)
	}

	narrator = &scriptedNarrator{replies: []string{"Niech będzie sto czterdzieści.", "Daj 140."}}
	if text, err := Narrate(context.Background(), narrator, request); err != nil || text != "Daj 140." {
		t.Errorf("a spelled-out amount: got %q, %v", text, err)
	}

	// A number that is part of what the merchant knows may be spelled out.
	request.Lore = "Stacja przestała odpowiadać czterdzieści dni temu."
	narrator = &scriptedNarrator{replies: []string{"Od czterdziestu dni nikt tu nie zagląda. Daj 140."}}

	if text, err := Narrate(context.Background(), narrator, request); err != nil || narrator.calls != 1 {
		t.Errorf("a number from the lore: got %q, %v", text, err)
	}

	request.Lore = ""

	narrator = &scriptedNarrator{replies: []string{"Oddam za 99."}}
	if text, err := Narrate(context.Background(), narrator, request); !errors.Is(err, ErrInconsistent) || text != request.Decision.Fallback() {
		t.Errorf("made-up price: got %q, %v", text, err)
	}

	offline := errors.New("offline")
	narrator = &scriptedNarrator{err: offline}

	if _, err := Narrate(context.Background(), narrator, request); !errors.Is(err, offline) || narrator.calls != narrateAttempts {
		t.Errorf("error: %v after %d calls", err, narrator.calls)
	}
}

func TestNarrateKeepsRepliesShort(t *testing.T) {
	long := "Raz. Dwa! Trzy? Cztery… Pięć. Sześć."

	request := counterRequest()
	narrator := &scriptedNarrator{replies: []string{long}}

	if text, _ := Narrate(context.Background(), narrator, request); text != "Raz. Dwa! Trzy?" {
		t.Errorf("trade reply = %q", text)
	}

	request.Decision = Decision{Outcome: OutcomeChat}
	if text, _ := Narrate(context.Background(), narrator, request); text != "Raz. Dwa! Trzy? Cztery…" {
		t.Errorf("chat reply = %q", text)
	}

	// A dot inside a number does not end a sentence.
	if got := tidy("Wersja 1.5 działa. Druga też. Trzecia nie.", "Moria", 2); got != "Wersja 1.5 działa. Druga też." {
		t.Errorf("tidy = %q", got)
	}

	for raw, want := range map[string]string{
		"Moria: Z wykopalisk, wędrowcze*.":                                 "Z wykopalisk, wędrowcze.",
		"*Uśmiecha się* Tam kryją się sekrety*</br class=\"tag\"> - Moria": "Uśmiecha się Tam kryją się sekrety",
		"„Nie mam drugiego zwoju.” —Moria":                                 "Nie mam drugiego zwoju.",
		"**Ty...** To była tylko gra!":                                     "Ty... To była tylko gra!",
	} {
		if got := tidy(raw, "Moria", 3); got != want {
			t.Errorf("tidy(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestRead(t *testing.T) {
	wares := []Ware{
		{ID: "rura", Name: "Rura z Gwoździami", Aliases: Aliases("Rura z Gwoździami")},
		{ID: "kurtka", Name: "Kurtka z Opon", Aliases: Aliases("Kurtka z Opon")},
	}

	tests := []struct {
		text  string
		focus string
		want  Intent
		sure  bool
		alone Intent
	}{
		{"dam ci 20 kapsli", "", Intent{Action: ActionOffer, Amount: 20}, true, Intent{Action: ActionOffer, Amount: 20}},
		{"no dobra, 30 i ani kapsla więcej", "rura", Intent{Action: ActionOffer, Amount: 30}, true, Intent{Action: ActionOffer, Amount: 30}},
		{"dam 80... albo nie, 100 za kurtkę", "", Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100}, true, Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100}},
		{"25", "rura", Intent{Action: ActionOffer, Amount: 25}, true, Intent{Action: ActionOffer, Amount: 25}},
		{"Dam trzydzieści pięć", "rura", Intent{Action: ActionOffer, Amount: 35}, true, Intent{Action: ActionOffer, Amount: 35}},
		{"sto dwadzieścia kapsli i ani jednego więcej", "", Intent{Action: ActionOffer, Amount: 120}, true, Intent{Action: ActionOffer, Amount: 120}},
		{"ile chcesz za rurę?", "", Intent{Action: ActionAsk, ItemID: "rura"}, true, Intent{Action: ActionAsk, ItemID: "rura"}},
		{"co masz na sprzedaż?", "", Intent{Action: ActionAsk}, true, Intent{Action: ActionAsk}},
		{"biorę", "rura", Intent{Action: ActionAccept}, true, Intent{Action: ActionAccept}},
		{"No dobra.", "rura", Intent{Action: ActionAccept}, true, Intent{Action: ActionAccept}},
		{"Żegnaj, stary", "", Intent{Action: ActionLeave}, true, Intent{Action: ActionLeave}},
		{"gdzie właściwie jest ta rafineria?", "", Intent{Action: ActionChat}, false, Intent{Action: ActionChat}},
		{"mam dwa pytania do ciebie", "rura", Intent{Action: ActionChat}, false, Intent{Action: ActionChat}},
		{"słyszałem, że w Schronie 7 jest drożej niż u ciebie", "", Intent{Action: ActionChat}, false, Intent{Action: ActionChat}},
		{"słyszałem, że w Schronie 7 jest drożej niż u ciebie", "rura", Intent{Action: ActionChat}, false, Intent{Action: ActionOffer, Amount: 7}},
		{"kupię 5 kurtek za 100 kapsli", "", Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100, Quantity: 5}, true, Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100, Quantity: 5}},
		{"dam 100 za 5 kurtek", "", Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100, Quantity: 5}, true, Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 100, Quantity: 5}},
		{"wezmę trzy rury", "", Intent{Action: ActionAccept, ItemID: "rura", Quantity: 3}, true, Intent{Action: ActionAccept, ItemID: "rura", Quantity: 3}},
		{"ile za 4 kurtki?", "", Intent{Action: ActionAsk, ItemID: "kurtka", Quantity: 4}, true, Intent{Action: ActionAsk, ItemID: "kurtka", Quantity: 4}},
		{"a 10 sztuk za 250?", "rura", Intent{Action: ActionOffer, Amount: 250, Quantity: 10}, true, Intent{Action: ActionOffer, Amount: 250, Quantity: 10}},
		{"2 x rura za 60", "", Intent{Action: ActionOffer, ItemID: "rura", Amount: 60, Quantity: 2}, true, Intent{Action: ActionOffer, ItemID: "rura", Amount: 60, Quantity: 2}},
	}

	// Where money is counted in pieces, "sztuk" after a number means money.
	gold := Read(InterpretRequest{Text: "dam 35 sztuk złota", Wares: wares, FocusID: "rura", Currency: "szt. złota"})
	if want := (Intent{Action: ActionOffer, Amount: 35}); gold.Intent != want || !gold.Sure {
		t.Errorf("an offer in pieces of gold = %+v", gold.Intent)
	}

	lot := Read(InterpretRequest{Text: "kupię 5 rur za 100 sztuk złota", Wares: wares, Currency: "szt. złota"})
	if want := (Intent{Action: ActionOffer, ItemID: "rura", Amount: 100, Quantity: 5}); lot.Intent != want {
		t.Errorf("a lot paid in pieces of gold = %+v", lot.Intent)
	}

	for _, test := range tests {
		reading := Read(InterpretRequest{Text: test.text, Wares: wares, FocusID: test.focus, Currency: "kapsli"})

		if reading.Intent != test.want || reading.Sure != test.sure {
			t.Errorf("Read(%q) = %+v sure=%v, want %+v sure=%v", test.text, reading.Intent, reading.Sure, test.want, test.sure)
		}

		if got := reading.Alone(); got != test.alone {
			t.Errorf("Read(%q).Alone() = %+v, want %+v", test.text, got, test.alone)
		}
	}
}

func TestMergeNeverLetsTheModelSpend(t *testing.T) {
	wares := []Ware{{ID: "rura", Name: "Rura z Gwoździami", Aliases: Aliases("Rura z Gwoździami")}}
	read := func(text, focus string) Reading {
		return Read(InterpretRequest{Text: text, Wares: wares, FocusID: focus, Currency: "kapsli"})
	}

	tests := []struct {
		name    string
		reading Reading
		model   Intent
		want    Intent
	}{
		{"a question is not a purchase", read("ile chcesz za rurę?", ""),
			Intent{Action: ActionOffer, ItemID: "rura", Amount: 40}, Intent{Action: ActionAsk, ItemID: "rura"}},
		{"the amount comes from the text", read("dam ci 20 kapsli, bo jest zardzewiała", "rura"),
			Intent{Action: ActionOffer, ItemID: "kurtka", Amount: 200, Persuasion: 1}, Intent{Action: ActionOffer, Amount: 20, Persuasion: 1}},
		{"no accepting on a hunch", read("niech pomyślę chwilę", "rura"),
			Intent{Action: ActionAccept, ItemID: "rura"}, Intent{Action: ActionChat}},
		{"no leaving on a hunch", read("ciężko się z tobą rozmawia", ""),
			Intent{Action: ActionLeave, Persuasion: -1}, Intent{Action: ActionChat, Persuasion: -1}},
		{"a bare number needs the model's word", read("słyszałem, że w Schronie 7 jest drożej", "rura"),
			Intent{Action: ActionChat}, Intent{Action: ActionChat}},
		{"a bare number confirmed as an offer", read("no to może jednak 25 wystarczy tym razem", "rura"),
			Intent{Action: ActionOffer, Amount: 99}, Intent{Action: ActionOffer, Amount: 25}},
		{"the model may spot an unnamed item", read("pokaż to coś ostrego, co wisi na ścianie obok drzwi", ""),
			Intent{Action: ActionAsk, ItemID: "rura"}, Intent{Action: ActionAsk, ItemID: "rura"}},
	}

	for _, test := range tests {
		if got := test.reading.Merge(test.model); got != test.want {
			t.Errorf("%s: got %+v, want %+v", test.name, got, test.want)
		}
	}
}

func TestFallbackOnlyUsesAllowedNumbers(t *testing.T) {
	merchant := testMerchant()
	merchant.Currency = "kapsli"
	merchant.Items[0].Description = "Broń, atak +3."

	session := NewSession(merchant, 300)
	intents := []Intent{
		{Action: ActionAsk},
		{Action: ActionAsk, ItemID: "a"},
		{Action: ActionOffer, ItemID: "a", Amount: 80},
		{Action: ActionOffer, ItemID: "a", Amount: 20},
		{Action: ActionOffer, ItemID: "a", Amount: 900},
		{Action: ActionChat},
		{Action: ActionOffer},
		{Action: ActionAccept, ItemID: "b"},
		{Action: ActionLeave},
	}

	for _, intent := range intents {
		decision := session.Apply(intent)
		if !NumbersOK(decision.Fallback(), decision.Numbers()) {
			t.Errorf("%s: fallback %q uses numbers outside %v", decision.Outcome, decision.Fallback(), decision.Numbers())
		}

		if decision.Currency != "kapsli" {
			t.Errorf("%s: currency = %q", decision.Outcome, decision.Currency)
		}
	}
}

func TestSanitize(t *testing.T) {
	request := InterpretRequest{Wares: []Ware{{ID: "a"}}}

	got := request.Sanitize(Intent{Action: "steal", ItemID: "zzz", Amount: 5, Persuasion: 9})
	want := Intent{Action: ActionChat, ItemID: "", Amount: 5, Persuasion: 2}

	if got != want {
		t.Errorf("Sanitize = %+v, want %+v", got, want)
	}
}

func TestForbiddenAmountsEarnASecondTry(t *testing.T) {
	narrator := &scriptedNarrator{replies: []string{"[spokój] Dołączam za 85 koron.", "[spokój] Topór mówi za mnie."}}
	request := NarrateRequest{
		Decision:  Decision{Outcome: OutcomeChat},
		Lore:      "Twoja służba kosztuje: «Umowa najmu» 85 koron.",
		Role:      RoleMercenary,
		Forbidden: []int{85},
	}

	text, err := Narrate(context.Background(), narrator, request)
	if err != nil || text != "[spokój] Topór mówi za mnie." || narrator.calls != 2 || narrator.corrections[1] != correction {
		t.Errorf("got %q, %v after %d calls (%q)", text, err, narrator.calls, narrator.corrections)
	}

	// Without the ban the fee from the lore passes as before.
	narrator = &scriptedNarrator{replies: []string{"[spokój] Dołączam za 85 koron."}}
	request.Forbidden = nil
	if text, _ := Narrate(context.Background(), narrator, request); narrator.calls != 1 || text != "[spokój] Dołączam za 85 koron." {
		t.Errorf("the fee from the lore was refused: %q after %d calls", text, narrator.calls)
	}
}

func TestMangledNamesEarnASecondTry(t *testing.T) {
	narrator := &scriptedNarrator{replies: []string{"[spokój] O tym pytaj Wędronowego Kupuca.", "[spokój] O tym pytaj: Wędrowny Kupiec, on wie."}}
	request := NarrateRequest{Decision: Decision{Outcome: OutcomeChat}, Required: []string{"Wędrowny Kupiec"}}

	text, err := Narrate(context.Background(), narrator, request)
	if err != nil || text != "[spokój] O tym pytaj: Wędrowny Kupiec, on wie." || narrator.calls != 2 || !strings.HasPrefix(narrator.corrections[1], nameCorrection) || !strings.HasSuffix(narrator.corrections[1], "Wędrowny Kupiec.") {
		t.Errorf("got %q, %v after %d calls (%q)", text, err, narrator.calls, narrator.corrections)
	}

	// A name in another case is the name; so is one with its fleeting e
	// gone. Half a name or a mangled one is not.
	for _, line := range []string{"Pytaj u Wędrownego Kupca w karczmie.", "Wędrownemu Kupcowi to zostaw.", "Wędrowny Kupiec wie."} {
		if !namesOK(line, []string{"Wędrowny Kupiec"}) {
			t.Errorf("%q refused", line)
		}
	}

	if !namesOK("Ozryka z Akademii pytaj, Jagny nie.", []string{"Ozryk", "Jagna"}) || namesOK("Oźyrka pytaj.", []string{"Ozryk"}) || namesOK("Kupca pytaj.", []string{"Wędrowny Kupiec"}) {
		t.Error("name check is off")
	}
}

func TestChatBriefFollowsTheRole(t *testing.T) {
	chat := Decision{Outcome: OutcomeChat, Mood: MoodPleased}
	if brief := chat.BriefFor(""); brief != chat.Brief() || !strings.Contains(brief, "polecić coś ze swojego towaru") {
		t.Errorf("shopkeeper brief = %q", brief)
	}

	// The mood closes a role's brief as it closes the shopkeeper's: the
	// model was trained on briefs that end this way.
	for role, want := range map[string]string{RoleMercenary: "O cenie służby mów tylko", RoleCompanion: "Jesteś już w drużynie", RoleWitness: "CO WIESZ"} {
		brief := chat.BriefFor(role)
		if !strings.Contains(brief, want) || strings.Contains(brief, "towaru") || !strings.HasSuffix(brief, " Nastrój twojej postaci: zadowolona.") {
			t.Errorf("brief for %q = %q", role[:20], brief)
		}
	}

	// A mercenary's closed deal is a hire the player pays for, not a sale.
	hired := Decision{Outcome: OutcomeSold, Item: Ware{Name: "Umowa najmu"}, Price: 85, Currency: "koron", Mood: MoodPleased}
	if brief := hired.BriefFor(RoleMercenary); !strings.Contains(brief, "płaci ci 85 koron za «Umowa najmu»") || strings.Contains(brief, "sprzedaż") || !strings.HasSuffix(brief, "zadowolona.") {
		t.Errorf("hire brief = %q", brief)
	}

	if hired.BriefFor("") != hired.Brief() {
		t.Errorf("shopkeeper's sale briefed as a hire: %q", hired.BriefFor(""))
	}

	// A legend's wage is agreed, not paid: nothing changes hands until dawn.
	wage := Decision{Outcome: OutcomeSold, Item: Ware{Name: WageName}, Price: 15, Currency: "koron", Mood: MoodCalm}
	if brief := wage.BriefFor(RoleMercenary); !strings.HasPrefix(brief, "Gracz przyjmuje twoją stawkę «Żołd dzienny» 15 koron za każdy dzień, płatne o świcie; nic z góry.") || !strings.Contains(brief, "Teraz nie bierzesz pieniędzy.") {
		t.Errorf("wage brief = %q", brief)
	}

	// Other decisions keep their brief whatever the role.
	farewell := Decision{Outcome: OutcomeFarewell}
	if farewell.BriefFor(RoleMercenary) != farewell.Brief() {
		t.Errorf("a farewell briefed by role: %q", farewell.BriefFor(RoleMercenary))
	}
}
