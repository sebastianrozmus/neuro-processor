package game

import "testing"

func TestForHero(t *testing.T) {
	text := "«Wynajmuję {pana|panią}. Ma {pan|pani} trzy dni.» {Byłeś|Byłaś} tam?"

	if got := ForHero(text, false); got != "«Wynajmuję pana. Ma pan trzy dni.» Byłeś tam?" {
		t.Errorf("for a man: %q", got)
	}

	if got := ForHero(text, true); got != "«Wynajmuję panią. Ma pani trzy dni.» Byłaś tam?" {
		t.Errorf("for a woman: %q", got)
	}

	// Braces without a choice are left alone.
	if got := ForHero("{nic} i {", true); got != "{nic} i {" {
		t.Errorf("plain braces: %q", got)
	}
}
