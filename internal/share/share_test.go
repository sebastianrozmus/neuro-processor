package share

import (
	"strings"
	"testing"
)

func valid() Payload {
	return Payload{
		Version: Version, Game: "test", Kind: KindTalk, Consent: true,
		Turns: []Turn{{Who: WhoPlayer, Text: "Dzień dobry."}, {Who: WhoNPC, Name: "Borin", Text: "Witaj.", Model: "run23"}},
	}
}

func TestValidate(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}

	cases := map[string]func(*Payload){
		"version":     func(p *Payload) { p.Version = 2 },
		"consent":     func(p *Payload) { p.Consent = false },
		"kind":        func(p *Payload) { p.Kind = "x" },
		"no turns":    func(p *Payload) { p.Turns = nil },
		"who":         func(p *Payload) { p.Turns[0].Who = "admin" },
		"empty":       func(p *Payload) { p.Turns[0].Text = "  " },
		"long":        func(p *Payload) { p.Turns[0].Text = strings.Repeat("ż", MaxText+1) },
		"control":     func(p *Payload) { p.Turns[0].Text = "a\x1b[2Jb" },
		"bad utf8":    func(p *Payload) { p.Turns[0].Text = "a\xffb" },
		"name line":   func(p *Payload) { p.Turns[1].Name = "Bo\nrin" },
		"rating kind": func(p *Payload) { p.Rating = &Rating{Turn: 1, Value: 1} },
		"no rating":   func(p *Payload) { p.Kind = KindRating },
		"rating turn": func(p *Payload) { p.Kind, p.Rating = KindRating, &Rating{Turn: 5, Value: 1} },
		"rating val":  func(p *Payload) { p.Kind, p.Rating = KindRating, &Rating{Turn: 1, Value: 3} },
	}

	for name, change := range cases {
		p := valid()
		change(&p)

		if p.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	p := valid()
	p.Kind, p.Rating = KindRating, &Rating{Turn: 1, Value: -1, Note: "za drogo\nnie po polsku"}
	p.Turns[0].Text = "linia\n\tdruga"

	if err := p.Validate(); err != nil {
		t.Fatalf("rating: %v", err)
	}
}

func TestSolveReachesDifficulty(t *testing.T) {
	nonce, ok := Solve("c", "k", 12, nil)
	if !ok || Work("c", "k", nonce) < 12 || !ValidNonce(nonce) {
		t.Fatalf("nonce %q", nonce)
	}

	if _, ok := Solve("c", "k", 64, func() bool { return true }); ok {
		t.Fatal("did not stop")
	}

	if Work("c", "k2", nonce) >= 12 && Work("c2", "k", nonce) >= 12 {
		t.Log("work happens to carry over; unlikely but possible")
	}
}

func TestFlags(t *testing.T) {
	p := valid()
	if flags := p.Flags(map[string]bool{"run23": true}); len(flags) != 0 {
		t.Fatalf("clean: %v", flags)
	}

	p.Turns = []Turn{{Who: WhoNPC, Text: "a", Model: "x"}}
	flags := strings.Join(p.Flags(map[string]bool{"run23": true}), ",")

	for _, want := range []string{"krotka", "bez-gracza", "nieznany-model"} {
		if !strings.Contains(flags, want) {
			t.Errorf("missing %s in %s", want, flags)
		}
	}
}

func TestIdentityPersists(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}

	again, err := LoadIdentity(dir)
	if err != nil || again.Key() != first.Key() {
		t.Fatalf("reload: %v", err)
	}

	other, _ := NewIdentity()
	if err := ForgetIdentity(dir, other); err != nil {
		t.Fatal(err)
	}

	if kept, _ := LoadIdentity(dir); kept.Key() != first.Key() {
		t.Fatal("a key that was not confirmed got removed")
	}

	if err := ForgetIdentity(dir, first); err != nil {
		t.Fatal(err)
	}

	fresh, _ := LoadIdentity(dir)
	if fresh.Key() == first.Key() {
		t.Fatal("key kept after forget")
	}
}
