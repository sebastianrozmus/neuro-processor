// Package inline shows real pictures in terminals that speak the kitty
// graphics protocol. A picture is sent to the terminal once, and the
// screen then places it with Unicode placeholders: ordinary printable
// characters whose colour names the picture, which survive being cut,
// measured and redrawn line by line like any other text. Terminals
// without the protocol get block art instead.
package inline

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	// placeholder is the character the protocol reserves for image cells.
	placeholder = '\U0010EEEE'
	// chunk is the most bytes of base64 one escape sequence may carry.
	chunk = 4096
	// MaxRows is as many rows as the placeholders can number.
	MaxRows = 40
)

// marks are the combining marks that number the rows of a placed picture;
// the column follows from the position in the line.
var marks = [MaxRows]rune{
	0x305, 0x30D, 0x30E, 0x310, 0x312, 0x33D, 0x33E, 0x33F, 0x346, 0x34A,
	0x34B, 0x34C, 0x350, 0x351, 0x352, 0x357, 0x35B, 0x363, 0x364, 0x365,
	0x366, 0x367, 0x368, 0x369, 0x36A, 0x36B, 0x36C, 0x36D, 0x36E, 0x36F,
	0x483, 0x484, 0x485, 0x486, 0x487, 0x592, 0x593, 0x594, 0x595, 0x597,
}

var (
	mu      sync.Mutex
	enabled bool
	out     io.Writer
	sent    sync.Map
)

// Configure turns inline pictures on or off: "kitty" forces them, "auto"
// turns them on when the terminal is known to draw them, anything else
// leaves them off. Pictures are sent to out, the terminal of the local
// player.
func Configure(mode string, writer io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "kitty":
		enabled = true
	case "auto":
		enabled = supported()
	default:
		enabled = false
	}

	out = writer
	if out == nil {
		enabled = false
	}
}

// supported guesses from the environment whether the terminal places
// pictures by placeholders. Multiplexers sit between the game and the
// terminal and do not pass the pictures on.
func supported() bool {
	term := os.Getenv("TERM")
	if os.Getenv("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return false
	}

	switch {
	case term == "xterm-kitty", os.Getenv("KITTY_WINDOW_ID") != "", os.Getenv("KONSOLE_VERSION") != "":
		return true
	}

	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "wezterm", "ghostty":
		return true
	}

	return false
}

// Enabled reports whether pictures are shown inline.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()

	return enabled
}

// Show sends the PNG picture to the terminal, the first time it is asked
// for, and returns the block of placeholders that puts it on the screen
// at cols by rows characters. The path only names the picture; the same
// path is the same picture for the rest of the run.
func Show(path string, png []byte, cols, rows int) string {
	if !Enabled() || len(png) == 0 || cols <= 0 || rows <= 0 {
		return ""
	}

	rows = min(rows, MaxRows)
	id := idOf(path)

	if _, done := sent.LoadOrStore(id, true); !done {
		mu.Lock()
		_, _ = io.WriteString(out, transmit(id, png, cols, rows))
		mu.Unlock()
	}

	return place(id, cols, rows)
}

// idOf names a picture for the terminal: 24 bits, never zero.
func idOf(path string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(path))

	if id := h.Sum32() & 0xFFFFFF; id != 0 {
		return id
	}

	return 1
}

// transmit words the escape sequences that hand the terminal the PNG for
// a virtual placement of cols by rows cells.
func transmit(id uint32, png []byte, cols, rows int) string {
	data := base64.StdEncoding.EncodeToString(png)

	var seq strings.Builder
	first := true

	for len(data) > 0 {
		size := min(chunk, len(data))
		part, more := data[:size], 0
		data = data[size:]

		if len(data) > 0 {
			more = 1
		}

		if first {
			fmt.Fprintf(&seq, "\x1b_Gf=100,t=d,a=T,U=1,q=2,i=%d,c=%d,r=%d,m=%d;%s\x1b\\", id, cols, rows, more, part)
			first = false
		} else {
			fmt.Fprintf(&seq, "\x1b_Gm=%d;%s\x1b\\", more, part)
		}
	}

	return seq.String()
}

// place builds the placeholder block: every line names the picture in
// its colour and its row in the mark on each cell.
func place(id uint32, cols, rows int) string {
	lines := make([]string, 0, rows)

	for row := range rows {
		var line strings.Builder
		fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%dm", id>>16&0xFF, id>>8&0xFF, id&0xFF)

		cell := string(placeholder) + string(marks[row])
		line.WriteString(strings.Repeat(cell, cols))
		line.WriteString("\x1b[0m")

		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}
