package speech

import (
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// recorder is a voice that remembers what it was given to say.
type recorder struct {
	queued []string
	moods  []Emotion
	whole  []string
	stops  int
}

func (r *recorder) speaker(queues bool) *Speaker {
	speaker := &Speaker{
		start: func(text string, mood Emotion) func() { return r.say(text) },
		startAs: func(_, _, text string, _ Emotion) func() {
			return r.say(text)
		},
	}

	if queues {
		speaker.queueAs = func(_, _ string) (func(string, Emotion), func()) {
			return func(text string, mood Emotion) {
				r.queued = append(r.queued, text)
				r.moods = append(r.moods, mood)
			}, func() { r.stops++ }
		}
	}

	return speaker
}

func (r *recorder) say(text string) func() {
	r.whole = append(r.whole, text)

	return func() { r.stops++ }
}

func TestSentencesWaitForTheEndOfASentence(t *testing.T) {
	for _, tc := range []struct {
		text  string
		final bool
		want  []string
	}{
		// Short sentences go together; the last one waits for what follows.
		{"Dwie korony? Za topór? Chyba żartujesz", false, []string{"Dwie korony? Za topór?"}},
		{"Dwie korony? Za topór? Chyba żartujesz, przybyszu.", false, []string{"Dwie korony? Za topór?"}},
		{"Dwie korony? Za topór? Chyba żartujesz, przybyszu.", true, []string{"Dwie korony? Za topór?", "Chyba żartujesz, przybyszu."}},
		// A number with a dot is no end of a sentence.
		{"Schron 7.5 jest daleko stąd, za rzeką. Idź", false, []string{"Schron 7.5 jest daleko stąd, za rzeką."}},
		{"Hm.", true, []string{"Hm."}},
		{"Hm... no dobrze, niech będzie trzydzieści. A", false, []string{"Hm... no dobrze, niech będzie trzydzieści."}},
	} {
		got, used := Sentences(tc.text, tc.final)
		if !slices.Equal(got, tc.want) {
			t.Errorf("Sentences(%q, %v) = %q, want %q", tc.text, tc.final, got, tc.want)
		}

		if tc.final && used != len(tc.text) {
			t.Errorf("Sentences(%q) used %d of %d", tc.text, used, len(tc.text))
		}
	}
}

func TestStreamSpeaksSentencesAsTheyAreWritten(t *testing.T) {
	voice := &recorder{}
	stream := voice.speaker(true).StreamAs("Borin", "armory")

	for _, piece := range []string{"[kpi", "na] Borin: Dwie korony? Za ", "topór? ", "Chyba żartu", "jesz, przybyszu."} {
		stream.Write(piece)
		if piece == "topór? " && !slices.Equal(voice.queued, []string{"Dwie korony? Za topór?"}) {
			t.Fatalf("the first sentence was not queued at once: %q", voice.queued)
		}
	}

	stream.Close("[kpina] Dwie korony? Za topór? Chyba żartujesz, przybyszu.")
	if !slices.Equal(voice.queued, []string{"Dwie korony? Za topór?", "Chyba żartujesz, przybyszu."}) || voice.moods[0] != Mockery || voice.moods[1] != Mockery || len(voice.whole) != 0 {
		t.Errorf("queued %q in %q, whole %q", voice.queued, voice.moods, voice.whole)
	}
}

func TestStreamStartsOverWhenTheGameSaysSomethingElse(t *testing.T) {
	voice := &recorder{}
	stream := voice.speaker(true).StreamAs("Borin", "armory")
	stream.Write("[radość] Dla ciebie tylko 999 koron, przyjacielu! I ")

	// The rules refused the price and the line was written again.
	stream.Close("[spokój] Sto dwadzieścia koron, ani grosza mniej.")
	if !slices.Equal(voice.whole, []string{"Sto dwadzieścia koron, ani grosza mniej."}) || voice.stops != 1 {
		t.Errorf("whole %q, stops %d", voice.whole, voice.stops)
	}

	// Pieces arriving after the end are not spoken.
	stream.Write("Więcej. ")
	if len(voice.queued) != 1 {
		t.Errorf("queued %q", voice.queued)
	}
}

func TestStreamCutByTheGameDoesNotStartOver(t *testing.T) {
	voice := &recorder{}
	stream := voice.speaker(true).StreamAs("Borin", "armory")
	stream.Write("Tyle mogę dać za ten topór. Bierz albo idź dalej. A tak w ogóle, ")
	stream.Write("słyszałeś o kopalni? ")

	// The game keeps only the sentences a deal may have.
	stream.Close("Tyle mogę dać za ten topór. Bierz albo idź dalej.")
	if len(voice.whole) != 0 || len(voice.queued) != 3 {
		t.Errorf("whole %q, queued %q", voice.whole, voice.queued)
	}
}

func TestStreamOnAVoiceThatSpeaksWholeLines(t *testing.T) {
	voice := &recorder{}
	stream := voice.speaker(false).StreamAs("Borin", "armory")
	stream.Write("[gniew] Precz stąd, złodzieju! ")
	stream.Close("[gniew] Precz stąd, złodzieju!")

	if !slices.Equal(voice.whole, []string{"Precz stąd, złodzieju!"}) || len(voice.queued) != 0 {
		t.Errorf("whole %q, queued %q", voice.whole, voice.queued)
	}

	var silent *Stream
	silent.Write("x")
	silent.Close("x")
	if (*Speaker)(nil).StreamAs("a", "b") != nil {
		t.Error("a nil speaker streams")
	}
}

func TestChatterboxFetchesSentencesInOrder(t *testing.T) {
	box := &fakeBox{}
	web := startBox(t, box)

	dir := t.TempDir()
	speaker := NewChatterbox(web, filepath.Join(dir, "voices.json"))
	chatterbox := &Chatterbox{URL: web, Settings: filepath.Join(dir, "voices.json"), Player: "true", Dir: dir}
	speaker.startAs, speaker.queueAs = chatterbox.start, chatterbox.queue

	speaker.SayAs("Borin", "armory", "Dwie korony? Za topór? Chyba żartujesz, przybyszu.", Mockery)

	deadline := time.Now().Add(5 * time.Second)
	for len(box.texts()) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if got := box.texts(); !slices.Equal(got, []string{"Dwie korony? Za topór?", "Chyba żartujesz, przybyszu."}) {
		t.Errorf("fetched %q", got)
	}
}

// startBox serves the fake Chatterbox for the length of a test.
func startBox(t *testing.T, box *fakeBox) string {
	web := httptest.NewServer(box.handler())
	t.Cleanup(web.Close)

	return web.URL
}

// texts are the lines the fake Chatterbox was asked for.
func (f *fakeBox) texts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	var texts []string
	for _, request := range f.requests {
		texts = append(texts, request.Text)
	}

	return texts
}
