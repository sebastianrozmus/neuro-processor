package inline

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func reset() {
	sent = sync.Map{}
}

func TestOffByDefault(t *testing.T) {
	Configure("", &bytes.Buffer{})
	if Enabled() || Show("a.png", []byte("png"), 4, 2) != "" {
		t.Error("pictures should be off unless asked for")
	}

	Configure("kitty", nil)
	if Enabled() {
		t.Error("no writer, no pictures")
	}

	Configure("kitty", &bytes.Buffer{})
	if !Enabled() {
		t.Error("kitty should force pictures on")
	}

	Configure("off", &bytes.Buffer{})
}

func TestAutoDetection(t *testing.T) {
	for _, env := range []string{"TERM", "TMUX", "KITTY_WINDOW_ID", "KONSOLE_VERSION", "TERM_PROGRAM"} {
		t.Setenv(env, "")
	}

	Configure("auto", &bytes.Buffer{})
	if Enabled() {
		t.Error("an unknown terminal should get block art")
	}

	t.Setenv("TERM", "xterm-kitty")
	Configure("auto", &bytes.Buffer{})
	if !Enabled() {
		t.Error("kitty should be recognised")
	}

	t.Setenv("TMUX", "/tmp/tmux-1/default,1,0")
	Configure("auto", &bytes.Buffer{})
	if Enabled() {
		t.Error("tmux does not pass pictures on")
	}

	t.Setenv("TMUX", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")
	Configure("auto", &bytes.Buffer{})
	if !Enabled() {
		t.Error("WezTerm should be recognised")
	}

	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	Configure("auto", &bytes.Buffer{})
	if Enabled() {
		t.Error("Terminal.app has no pictures")
	}

	Configure("off", &bytes.Buffer{})
}

func TestShowTransmitsOnceAndPlaces(t *testing.T) {
	reset()
	var terminal bytes.Buffer
	Configure("kitty", &terminal)
	t.Cleanup(func() { Configure("off", nil) })

	png := bytes.Repeat([]byte{1, 2, 3, 4, 5}, 2000) // 10 000 bytes -> 13 336 base64 characters
	block := Show("art/x/keeper-armory.png", png, 40, 20)

	sequences := strings.Split(strings.TrimSuffix(terminal.String(), "\x1b\\"), "\x1b\\")
	if len(sequences) != 4 {
		t.Fatalf("%d chunks sent", len(sequences))
	}

	id := idOf("art/x/keeper-armory.png")
	if !strings.HasPrefix(sequences[0], "\x1b_Gf=100,t=d,a=T,U=1,q=2,i="+itoa(id)+",c=40,r=20,m=1;") {
		t.Errorf("first chunk = %.80q", sequences[0])
	}

	if !strings.HasPrefix(sequences[1], "\x1b_Gm=1;") || !strings.HasPrefix(sequences[3], "\x1b_Gm=0;") {
		t.Errorf("later chunks = %.20q, %.20q", sequences[1], sequences[3])
	}

	for _, seq := range sequences {
		if payload := seq[strings.Index(seq, ";")+1:]; len(payload) > chunk {
			t.Errorf("chunk of %d bytes", len(payload))
		}
	}

	lines := strings.Split(block, "\n")
	if len(lines) != 20 {
		t.Fatalf("%d rows placed", len(lines))
	}

	for row, line := range lines {
		if width := lipgloss.Width(line); width != 40 {
			t.Errorf("row %d measures %d cells", row, width)
		}

		colour := "\x1b[38;2;" + itoa(id>>16&0xFF) + ";" + itoa(id>>8&0xFF) + ";" + itoa(id&0xFF) + "m"
		if !strings.HasPrefix(line, colour) || !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("row %d = %.40q", row, line)
		}

		if cell := string(placeholder) + string(marks[row]); strings.Count(line, cell) != 40 {
			t.Errorf("row %d lacks its marks", row)
		}
	}

	// The same picture again is placed without being sent again.
	terminal.Reset()
	if again := Show("art/x/keeper-armory.png", png, 40, 20); again != block || terminal.Len() != 0 {
		t.Error("a picture was sent twice")
	}

	if tall := Show("other.png", png, 2, 100); strings.Count(tall, "\n") != MaxRows-1 {
		t.Error("rows are capped at what the marks can number")
	}
}

func TestIDNeverZero(t *testing.T) {
	if idOf("") == 0 || idOf("x") > 0xFFFFFF {
		t.Error("ids must fit 24 bits and never be zero")
	}
}

func itoa(n uint32) string {
	return strconv.FormatUint(uint64(n), 10)
}
