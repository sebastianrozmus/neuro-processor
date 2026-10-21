package speech

import "testing"

func TestFeminine(t *testing.T) {
	for name, want := range map[string]bool{
		"Siostra Agnes": true, "Elira Cieniokrąg": true, "Jagna": true, "Bogna Topór": true,
		"Borin Kamienna Pięść": false, "Thorin Żelazna Broda": false, "Ozryk z Akademii": false,
		"Kuba Rozpruwacz": false, "Wędrowny Kupiec": false, "": false,
		"Pani Kaliber": true, "Doktor Lena": true, "Brat Jaromir": false, "Żerca Bogumił": false, "Sanitariusz Kuba": false,
		"Sanitariuszka Mira": true, "Kwatermistrz Vey": false, "Stara Jagna": true, "Sir Gniewomir Nieugięty": false,
	} {
		if got := Feminine(name); got != want {
			t.Errorf("Feminine(%q) = %v", name, got)
		}
	}
}

func TestWomenSpeakWithTheirOwnSample(t *testing.T) {
	settings := VoiceSettings{Default: "man.wav", Narrator: "gm.wav", Female: "woman.wav", ByName: map[string]string{"Jagna": "jagna.wav"}}

	for _, test := range []struct{ name, kind, want string }{
		{"Siostra Agnes", "alchemist", "woman.wav"},
		{"Borin Kamienna Pięść", "towarzysz", "man.wav"},
		{"Jagna", "alchemist", "jagna.wav"},
		{"", "narrator", "gm.wav"},
	} {
		if got := settings.VoiceFor(test.name, test.kind); got != test.want {
			t.Errorf("VoiceFor(%q, %q) = %q, want %q", test.name, test.kind, got, test.want)
		}
	}
}
