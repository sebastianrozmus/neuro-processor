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
