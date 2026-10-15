package game

import "testing"

func TestOffered(t *testing.T) {
	cases := []struct {
		line, name string
		want       bool
	}{
		{"[spokój] Mam też hełm z rondem.", "Hełm z Rondem", true},
		{"Polecam Eliksir Gromu, mocna rzecz.", "Eliksir Gromu", true},
		{"Jagody, 3 sztuki złota za garść.", "Jagody", true},
		{"[spokój] Jagody? Krasnoludowi tego nie nałożę na widelec. Jeśli chcesz jagody, idź po nie w Kopalni Durgara.", "Jagody", false},
		{"Wracaj bez ran i bez jagód, bo zielska nie trzymam na swojej półce.", "Jagody", false},
		{"Tarczy nie mam, ale Kolczuga jest za 95.", "Tarcza", false},
		{"Kupcy mówią o czerwonym świetle w kopalni.", "Czerwone Światło", false},
	}

	for _, test := range cases {
		if got := Offered(test.line, test.name); got != test.want {
			t.Errorf("Offered(%q, %q) = %v, want %v", test.line, test.name, got, test.want)
		}
	}
}

func TestFitMood(t *testing.T) {
	cases := []struct {
		text string
		mood Mood
		want string
	}{
		{"[gniew] Czego?", MoodCalm, "[spokój] Czego?"},
		{"[kpina] Jagody? Krasnoludowi?", MoodCalm, "[kpina] Jagody? Krasnoludowi?"},
		{"[radość] Miło cię widzieć.", MoodOffended, "[gniew] Miło cię widzieć."},
		{"[smutek] Szkoda.", MoodPleased, "[radość] Szkoda."},
		{"Bez znacznika.", MoodCalm, "Bez znacznika."},
		{"[gniew] Mów.", "", "[gniew] Mów."},
	}

	for _, test := range cases {
		if got := fitMood(test.text, test.mood); got != test.want {
			t.Errorf("fitMood(%q, %q) = %q, want %q", test.text, test.mood, got, test.want)
		}
	}
}
