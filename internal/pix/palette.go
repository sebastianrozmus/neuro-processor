package pix

import (
	"strings"
	"sync"
)

// settings is how pictures are rendered: the cell new art is drawn for and
// whether palette colours are sent as they are or as the nearest of the
// terminal's 256.
var settings = struct {
	sync.RWMutex
	cellW, cellH int
	truecolor    bool
}{cellW: 2, cellH: 2, truecolor: true}

// Configure picks the block characters new pictures are drawn with:
// "half" (1x2 pixels per cell, the look of the embedded art), "quad" (2x2,
// standard block elements) or "sextant" (2x3, Symbols for Legacy
// Computing, which needs a font that has them). Anything else means quad.
// truecolor false maps palette colours onto the 256-colour terminal palette.
func Configure(blocks string, truecolor bool) {
	settings.Lock()
	defer settings.Unlock()

	switch strings.ToLower(strings.TrimSpace(blocks)) {
	case "half":
		settings.cellW, settings.cellH = 1, 2
	case "sextant":
		settings.cellW, settings.cellH = 2, 3
	default:
		settings.cellW, settings.cellH = 2, 2
	}

	settings.truecolor = truecolor
}

// Cell returns how many pixels wide and high a character cell is in
// pictures made now.
func Cell() (int, int) {
	settings.RLock()
	defer settings.RUnlock()

	return settings.cellW, settings.cellH
}

// TrueColor reports whether palette colours are sent as 24-bit values.
func TrueColor() bool {
	settings.RLock()
	defer settings.RUnlock()

	return settings.truecolor
}

// xterm is the terminal's 256-colour palette.
var xterm = func() [256][3]uint8 {
	var palette [256][3]uint8

	base := [16][3]uint8{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0}, {0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	copy(palette[:], base[:])

	steps := [6]uint8{0, 95, 135, 175, 215, 255}
	for index := range 216 {
		palette[16+index] = [3]uint8{steps[index/36], steps[index/6%6], steps[index%6]}
	}

	for index := range 24 {
		grey := uint8(8 + index*10)
		palette[232+index] = [3]uint8{grey, grey, grey}
	}

	return palette
}()

// xtermRGB returns the colour of a terminal palette index.
func xtermRGB(index uint8) [3]uint8 {
	return xterm[index]
}

var nearest sync.Map

// nearestXterm returns the terminal palette index closest to a colour. The
// first 16 are skipped: terminal themes repaint them, the rest are fixed.
func nearestXterm(rgb [3]uint8) uint8 {
	if cached, ok := nearest.Load(rgb); ok {
		return cached.(uint8)
	}

	best, bestDistance := uint8(16), -1
	for index := 16; index < 256; index++ {
		if d := distance(xterm[index], rgb); bestDistance < 0 || d < bestDistance {
			best, bestDistance = uint8(index), d
		}
	}

	nearest.Store(rgb, best)

	return best
}

// distance is the squared difference between two colours.
func distance(a, b [3]uint8) int {
	total := 0
	for channel := range 3 {
		diff := int(a[channel]) - int(b[channel])
		total += diff * diff
	}

	return total
}
