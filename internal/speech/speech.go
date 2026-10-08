// Package speech reads the characters' lines aloud through text-to-speech
// and turns the player's voice into text, so that a conversation can be
// heard and spoken as well as read.
package speech

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

const (
	// polishVoice is the Polish voice shipped with macOS.
	polishVoice = "Zosia"
	polishTag   = "pl_PL"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Emotion is the mood a line is spoken in. The names are the Polish tags a
// narrator puts in front of its lines.
type Emotion string

const (
	Calm       Emotion = "spokój"
	Joy        Emotion = "radość"
	Anger      Emotion = "gniew"
	Sadness    Emotion = "smutek"
	Fear       Emotion = "strach"
	Mockery    Emotion = "kpina"
	Surprise   Emotion = "zdziwienie"
	Tenderness Emotion = "czułość"
)

// Emotions lists the moods a narrator may choose from.
var Emotions = []Emotion{Calm, Joy, Anger, Sadness, Fear, Mockery, Surprise, Tenderness}

// delivery is how the system voice renders a mood: words per minute and
// pitch base, in the units of its embedded commands.
type delivery struct {
	rate  int
	pitch int
}

var deliveries = map[Emotion]delivery{
	Calm: {175, 48}, Joy: {195, 53}, Anger: {205, 42}, Sadness: {150, 44},
	Fear: {210, 56}, Mockery: {170, 50}, Surprise: {190, 57}, Tenderness: {160, 50},
}

// Speaker says lines aloud, one at a time: a new line cuts the previous
// one short, so that the speech keeps up with the game.
type Speaker struct {
	start func(text string, mood Emotion) (stop func())

	mu   sync.Mutex
	stop func()
}

// New returns a speaker on the system voice (macOS `say`), the Polish one
// unless another is named. It returns nil when the system cannot speak.
func New(voice string) *Speaker {
	say, err := exec.LookPath("say")
	if err != nil {
		return nil
	}

	if voice == "" && hasVoice(say, polishVoice) {
		voice = polishVoice
	}

	return &Speaker{start: func(text string, mood Emotion) func() {
		args := []string{}
		if voice != "" {
			args = append(args, "-v", voice)
		}

		// The mood is carried by the voice's own embedded commands.
		if how, ok := deliveries[mood]; ok {
			text = fmt.Sprintf("[[rate %d]] [[pbas %d]] %s", how.rate, how.pitch, text)
		}

		return run(exec.Command(say, append(args, text)...))
	}}
}

// NewCommand returns a speaker that pipes each line into a shell command of
// the player's choosing, for any text-to-speech they have installed. The
// mood reaches the command as the NEURO_TTS_EMOTION variable.
func NewCommand(command string) *Speaker {
	if strings.TrimSpace(command) == "" {
		return nil
	}

	return &Speaker{start: func(text string, mood Emotion) func() {
		cmd := exec.Command("/bin/sh", "-c", command)
		cmd.Stdin = strings.NewReader(text + "\n")
		cmd.Env = append(os.Environ(), "NEURO_TTS_EMOTION="+string(mood))

		return run(cmd)
	}}
}

// run starts a command and returns what stops it.
func run(cmd *exec.Cmd) func() {
	if cmd.Start() != nil {
		return func() {}
	}

	go func() { _ = cmd.Wait() }()

	return func() { _ = cmd.Process.Kill() }
}

// hasVoice reports whether the system has a voice of the given name.
func hasVoice(say, voice string) bool {
	out, err := exec.Command(say, "-v", "?").Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, voice+" ") && strings.Contains(line, polishTag) {
			return true
		}
	}

	return false
}

// Say reads a line aloud in the given mood, cutting short whatever was
// being said. A nil speaker is silent.
func (s *Speaker) Say(text string, mood Emotion) {
	if s == nil {
		return
	}

	text = Clean(text)
	if text == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop != nil {
		s.stop()
	}

	s.stop = s.start(text, mood)
}

// Clean strips what should not be read out: colour codes, quotation marks
// and stray symbols.
func Clean(text string) string {
	text = ansi.ReplaceAllString(text, "")
	text = strings.NewReplacer("«", "", "»", "", "„", "", "”", "", "\"", "", "*", "", "·", ",", "—", ",", "–", ",", "[", "", "]", "").Replace(text)

	return strings.Join(strings.Fields(text), " ")
}

// SplitMood takes the mood tag a narrator puts in front of a line, such as
// "[kpina] Tyle za rurę?", off the line. Lines without one are calm.
func SplitMood(text string) (Emotion, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "[") {
		return Calm, text
	}

	end := strings.Index(text, "]")
	if end < 0 {
		return Calm, text
	}

	tag := Emotion(strings.ToLower(strings.TrimSpace(text[1:end])))
	rest := strings.TrimSpace(text[end+1:])

	for _, known := range Emotions {
		if tag == known {
			return known, rest
		}
	}

	return Calm, rest
}
