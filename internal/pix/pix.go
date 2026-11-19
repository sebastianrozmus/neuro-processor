// Package pix composes small palette-indexed pictures and draws them in a
// terminal with block characters: two pixels per character cell for the
// art shipped with the game, four or six for pictures drawn by the image
// model.
package pix

import (
	"strconv"
	"strings"
	"sync"

	"github.com/sebastianrozmus/neuro-processor/assets"
)

var cache sync.Map

// Image is a picture of palette indexes. Index 0 is transparent. Without a
// Palette the indexes are the terminal's 256 colours; with one, index i is
// Palette[i]. CellW and CellH say how many pixels make one character cell
// when the picture is rendered; zero means 1x2, the half blocks of the
// embedded art.
type Image struct {
	W       int
	H       int
	Pix     []uint8
	CellW   int
	CellH   int
	Palette [][3]uint8
}

// Load returns the embedded picture stored under the given name.
func Load(name string) (Image, bool) {
	if cached, ok := cache.Load(name); ok {
		return cached.(Image), true
	}

	data, err := assets.Pixels(name)
	if err != nil {
		return Image{}, false
	}

	image, err := Decode(data)
	if err != nil {
		return Image{}, false
	}

	cache.Store(name, image)

	return image, true
}

// Clone returns a copy that can be drawn on.
func (img Image) Clone() Image {
	img.Pix = append([]uint8(nil), img.Pix...)
	if img.Palette != nil {
		img.Palette = append([][3]uint8(nil), img.Palette...)
	}

	return img
}

// blank returns a transparent picture of the given size with the same
// palette and cell as this one.
func (img Image) blank(width, height int) Image {
	out := New(width, height)
	out.CellW, out.CellH = img.CellW, img.CellH

	if img.Palette != nil {
		out.Palette = append([][3]uint8(nil), img.Palette...)
	}

	return out
}

// cell returns how many pixels wide and high one character cell is.
func (img Image) cell() (int, int) {
	if img.CellW <= 0 || img.CellH <= 0 {
		return 1, 2
	}

	return img.CellW, img.CellH
}

// rgb returns the colour of a palette index.
func (img Image) rgb(index uint8) [3]uint8 {
	if img.Palette == nil {
		return xtermRGB(index)
	}

	if int(index) < len(img.Palette) {
		return img.Palette[index]
	}

	return [3]uint8{}
}

// colour returns the index that paints a terminal colour on this picture:
// the colour itself without a palette, otherwise the palette entry of the
// same colour, added when there is room for it.
func (img *Image) colour(xterm uint8) uint8 {
	if img.Palette == nil || xterm == 0 {
		return xterm
	}

	return img.adopt(xtermRGB(xterm))
}

// adopt returns the palette index of an RGB colour, adding it when missing
// and the palette has room, or the nearest entry otherwise.
func (img *Image) adopt(rgb [3]uint8) uint8 {
	for index := 1; index < len(img.Palette); index++ {
		if img.Palette[index] == rgb {
			return uint8(index)
		}
	}

	if len(img.Palette) < 255 {
		if len(img.Palette) == 0 {
			img.Palette = append(img.Palette, [3]uint8{})
		}

		img.Palette = append(img.Palette, rgb)

		return uint8(len(img.Palette) - 1)
	}

	best, bestDistance := uint8(1), -1
	for index := 1; index < len(img.Palette); index++ {
		if d := distance(img.Palette[index], rgb); bestDistance < 0 || d < bestDistance {
			best, bestDistance = uint8(index), d
		}
	}

	return best
}

// Draw paints src onto the image with its top left corner at x, y.
// Transparent pixels and pixels falling outside are skipped. Colours are
// carried over by value when the two pictures have different palettes.
func (img *Image) Draw(src Image, x, y int) {
	var mapped [256]int16
	for index := range mapped {
		mapped[index] = -1
	}

	convert := func(index uint8) uint8 {
		if index == 0 {
			return 0
		}

		if mapped[index] < 0 {
			switch {
			case img.Palette == nil && src.Palette == nil:
				mapped[index] = int16(index)
			case img.Palette == nil:
				mapped[index] = int16(nearestXterm(src.rgb(index)))
			default:
				mapped[index] = int16(img.adopt(src.rgb(index)))
			}
		}

		return uint8(mapped[index])
	}

	for sy := range src.H {
		for sx := range src.W {
			colour := src.Pix[sy*src.W+sx]
			dx, dy := x+sx, y+sy

			if colour != 0 && dx >= 0 && dx < img.W && dy >= 0 && dy < img.H {
				img.Pix[dy*img.W+dx] = convert(colour)
			}
		}
	}
}

// blockCell is one character of a rendered picture.
type blockCell struct {
	fg, bg uint8
	glyph  string
}

var blank = blockCell{glyph: " "}

// Render draws the image as lines of block characters.
func (img Image) Render() string {
	cellW, cellH := img.cell()
	lines := make([]string, 0, (img.H+cellH-1)/cellH)

	for y := 0; y < img.H; y += cellH {
		var line strings.Builder

		var (
			previous blockCell
			run      int
		)

		for x := 0; x <= img.W; x += cellW {
			var current blockCell
			if x < img.W {
				current = img.block(x, y)
			}

			if run > 0 && current == previous {
				run++

				continue
			}

			if run > 0 {
				line.WriteString(img.cells(previous, run))
			}

			previous, run = current, 1
		}

		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// block decides the character and colours of the cell whose top left pixel
// is at x, y.
func (img Image) block(x, y int) blockCell {
	cellW, cellH := img.cell()
	if cellW == 1 && cellH == 2 {
		return img.half(x, y)
	}

	var (
		pixels      [6]uint8
		counts      = map[uint8]int{}
		transparent bool
	)

	for dy := range cellH {
		for dx := range cellW {
			pixel := img.at(x+dx, y+dy)
			pixels[dy*cellW+dx] = pixel

			if pixel == 0 {
				transparent = true
			} else {
				counts[pixel]++
			}
		}
	}

	if len(counts) == 0 {
		return blank
	}

	first, second := leaders(counts)

	var bits int
	for index := range cellW * cellH {
		pixel := pixels[index]

		switch {
		case pixel == first:
			bits |= 1 << index
		case pixel == 0, transparent, second == 0:
		case distance(img.rgb(pixel), img.rgb(first)) <= distance(img.rgb(pixel), img.rgb(second)):
			bits |= 1 << index
		}
	}

	cell := blockCell{fg: first, glyph: glyph(bits, cellW, cellH)}
	if !transparent && second != 0 {
		cell.bg = second
	}

	return cell
}

// half decides a cell of the 1x2 look: the upper pixel paints the
// foreground of a ▀, the lower one its background.
func (img Image) half(x, y int) blockCell {
	top, bottom := img.at(x, y), img.at(x, y+1)

	switch {
	case top == 0 && bottom == 0:
		return blank
	case bottom == 0:
		return blockCell{fg: top, glyph: "▀"}
	case top == 0:
		return blockCell{fg: bottom, glyph: "▄"}
	}

	return blockCell{fg: top, bg: bottom, glyph: "▀"}
}

// leaders returns the two most frequent colours, ties going to the lower
// index; the second is 0 when there is only one.
func leaders(counts map[uint8]int) (uint8, uint8) {
	var first, second uint8

	for colour, count := range counts {
		switch {
		case first == 0 || count > counts[first] || (count == counts[first] && colour < first):
			first, second = colour, first
		case second == 0 || count > counts[second] || (count == counts[second] && colour < second):
			second = colour
		}
	}

	return first, second
}

const quadrants = " ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█"

// glyph returns the block character whose set parts, row by row from the
// top left, are the given bits.
func glyph(bits, cellW, cellH int) string {
	switch cellW * cellH {
	case 4:
		return string([]rune(quadrants)[bits])
	case 6:
		switch bits {
		case 0:
			return " "
		case 21:
			return "▌"
		case 42:
			return "▐"
		case 63:
			return "█"
		}

		code := 0x1FB00 + bits - 1
		if bits > 21 {
			code--
		}

		if bits > 42 {
			code--
		}

		return string(rune(code))
	}

	if bits == 0 {
		return " "
	}

	return "█"
}

func (img Image) at(x, y int) uint8 {
	if x >= img.W || y >= img.H {
		return 0
	}

	return img.Pix[y*img.W+x]
}

// cells draws count copies of a cell.
func (img Image) cells(cell blockCell, count int) string {
	if cell == blank {
		return strings.Repeat(" ", count)
	}

	var out strings.Builder
	out.WriteString("\x1b[")
	out.WriteString(img.code(cell.fg, false))

	if cell.bg != 0 {
		out.WriteString(";")
		out.WriteString(img.code(cell.bg, true))
	}

	out.WriteString("m")
	out.WriteString(strings.Repeat(cell.glyph, count))
	out.WriteString("\x1b[0m")

	return out.String()
}

// code returns the SGR parameters that set an index as the foreground or
// the background colour.
func (img Image) code(index uint8, background bool) string {
	prefix := "38;"
	if background {
		prefix = "48;"
	}

	if img.Palette == nil {
		return prefix + "5;" + strconv.Itoa(int(index))
	}

	rgb := img.rgb(index)
	if !TrueColor() {
		return prefix + "5;" + strconv.Itoa(int(nearestXterm(rgb)))
	}

	return prefix + "2;" + strconv.Itoa(int(rgb[0])) + ";" + strconv.Itoa(int(rgb[1])) + ";" + strconv.Itoa(int(rgb[2]))
}
