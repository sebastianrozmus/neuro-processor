// Package pix composes small palette-indexed pictures and draws them in a
// terminal with half-block characters, two pixels per character cell.
package pix

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sebastianrozmus/neuro-processor/assets"
)

const headerSize = 2

var cache sync.Map

// Image is a picture of 256-colour palette indexes. Index 0 is transparent.
type Image struct {
	W   int
	H   int
	Pix []uint8
}

// Load returns the embedded picture stored under the given name.
func Load(name string) (Image, bool) {
	if cached, ok := cache.Load(name); ok {
		return cached.(Image), true
	}

	data, err := assets.Pixels(name)
	if err != nil || len(data) < headerSize || len(data) != headerSize+int(data[0])*int(data[1]) {
		return Image{}, false
	}

	image := Image{W: int(data[0]), H: int(data[1]), Pix: data[headerSize:]}
	cache.Store(name, image)

	return image, true
}

// Clone returns a copy that can be drawn on.
func (img Image) Clone() Image {
	img.Pix = append([]uint8(nil), img.Pix...)

	return img
}

// Draw paints src onto the image with its top left corner at x, y.
// Transparent pixels and pixels falling outside are skipped.
func (img *Image) Draw(src Image, x, y int) {
	for sy := range src.H {
		for sx := range src.W {
			colour := src.Pix[sy*src.W+sx]
			dx, dy := x+sx, y+sy

			if colour != 0 && dx >= 0 && dx < img.W && dy >= 0 && dy < img.H {
				img.Pix[dy*img.W+dx] = colour
			}
		}
	}
}

// Render draws the image as lines of half-block characters.
func (img Image) Render() string {
	lines := make([]string, 0, (img.H+1)/2)

	for y := 0; y < img.H; y += 2 {
		var line strings.Builder

		for x := 0; x < img.W; {
			top, bottom := img.at(x, y), img.at(x, y+1)

			run := 1
			for x+run < img.W && img.at(x+run, y) == top && img.at(x+run, y+1) == bottom {
				run++
			}

			line.WriteString(cells(top, bottom, run))
			x += run
		}

		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

func (img Image) at(x, y int) uint8 {
	if y >= img.H {
		return 0
	}

	return img.Pix[y*img.W+x]
}

// cells draws count character cells with the given upper and lower pixel.
func cells(top, bottom uint8, count int) string {
	switch {
	case top == 0 && bottom == 0:
		return strings.Repeat(" ", count)
	case bottom == 0:
		return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", top, strings.Repeat("▀", count))
	case top == 0:
		return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", bottom, strings.Repeat("▄", count))
	}

	return fmt.Sprintf("\x1b[38;5;%d;48;5;%dm%s\x1b[0m", top, bottom, strings.Repeat("▀", count))
}
