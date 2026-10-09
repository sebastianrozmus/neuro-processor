package pix

import "strings"

// New returns a transparent picture of the given size.
func New(width, height int) Image {
	return Image{W: width, H: height, Pix: make([]uint8, width*height)}
}

// Tint returns a copy of the picture with every visible pixel in one colour:
// a flash, or a silhouette.
func (img Image) Tint(colour uint8) Image {
	out := img.Clone()
	colour = out.colour(colour)

	for index, pixel := range out.Pix {
		if pixel != 0 {
			out.Pix[index] = colour
		}
	}

	return out
}

// Shifted returns the picture moved right by dx pixels, what falls off
// the edge being lost.
func (img Image) Shifted(dx int) Image {
	out := img.blank(img.W, img.H)
	out.Draw(img, dx, 0)

	return out
}

// Set colours one pixel in a terminal colour, if it lies inside.
func (img *Image) Set(x, y int, colour uint8) {
	if x >= 0 && x < img.W && y >= 0 && y < img.H {
		img.Pix[y*img.W+x] = img.colour(colour)
	}
}

// Frame paints a border of the given thickness around the picture.
func (img *Image) Frame(colour uint8, thickness int) {
	for y := range img.H {
		for x := range img.W {
			if x < thickness || y < thickness || x >= img.W-thickness || y >= img.H-thickness {
				img.Set(x, y, colour)
			}
		}
	}
}

// Line draws a straight line between two points.
func (img *Image) Line(x0, y0, x1, y1 int, colour uint8) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1

	if x0 > x1 {
		sx = -1
	}

	if y0 > y1 {
		sy = -1
	}

	err := dx + dy

	for {
		img.Set(x0, y0, colour)

		if x0 == x1 && y0 == y1 {
			return
		}

		if doubled := 2 * err; doubled >= dy && doubled <= dx {
			err += dy + dx
			x0 += sx
			y0 += sy
		} else if doubled >= dy {
			err += dy
			x0 += sx
		} else {
			err += dx
			y0 += sy
		}
	}
}

// Ring draws a circle of the given radius, two pixels wide so that it reads
// at the half height of a terminal cell.
func (img *Image) Ring(cx, cy, radius int, colour uint8) {
	for y := -radius - 1; y <= radius+1; y++ {
		for x := -radius - 1; x <= radius+1; x++ {
			distance := x*x + y*y
			if distance >= (radius-1)*(radius-1) && distance <= (radius+1)*(radius+1) {
				img.Set(cx+x, cy+y, colour)
			}
		}
	}
}

// Bolt draws a jagged line between two points, like lightning.
func (img *Image) Bolt(x0, y0, x1, y1 int, colour uint8) {
	const steps = 5

	px, py := x0, y0

	for step := 1; step <= steps; step++ {
		nx := x0 + (x1-x0)*step/steps
		ny := y0 + (y1-y0)*step/steps

		// Every other joint is kicked sideways.
		if step < steps {
			kick := 3 - 6*(step%2)
			nx += kick
		}

		img.Line(px, py, nx, ny, colour)
		px, py = nx, ny
	}
}

// glyphs is a 3x5 pixel font for the few signs that damage numbers need.
var glyphs = map[rune][5]string{
	'0': {"###", "# #", "# #", "# #", "###"},
	'1': {" # ", "## ", " # ", " # ", "###"},
	'2': {"###", "  #", "###", "#  ", "###"},
	'3': {"###", "  #", "###", "  #", "###"},
	'4': {"# #", "# #", "###", "  #", "  #"},
	'5': {"###", "#  ", "###", "  #", "###"},
	'6': {"###", "#  ", "###", "# #", "###"},
	'7': {"###", "  #", " # ", " # ", " # "},
	'8': {"###", "# #", "###", "# #", "###"},
	'9': {"###", "# #", "###", "  #", "###"},
	'-': {"   ", "   ", "###", "   ", "   "},
	'+': {"   ", " # ", "###", " # ", "   "},
	'!': {" # ", " # ", " # ", "   ", " # "},
	' ': {"   ", "   ", "   ", "   ", "   "},
}

const (
	glyphWidth  = 3
	glyphHeight = 5
)

// Text renders a short string of digits and signs in one colour, every
// pixel of the font scaled up by the given factor.
func Text(text string, colour uint8, scale int) Image {
	scale = max(scale, 1)
	img := New(len([]rune(text))*(glyphWidth+1)*scale, glyphHeight*scale)

	for index, char := range []rune(text) {
		glyph, ok := glyphs[char]
		if !ok {
			continue
		}

		for row, line := range glyph {
			for column, cell := range line {
				if cell != '#' {
					continue
				}

				for dy := range scale {
					for dx := range scale {
						img.Set((index*(glyphWidth+1)+column)*scale+dx, row*scale+dy, colour)
					}
				}
			}
		}
	}

	return img
}

// Blank reports whether a rendered picture has no visible cell.
func Blank(rendered string) bool {
	return strings.TrimSpace(rendered) == ""
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

// Shrink returns the picture reduced by a whole factor: every block of
// pixels becomes one, in the colour most of the block's visible pixels
// have, so that thin outlines survive better than by sampling.
func (img Image) Shrink(factor int) Image {
	if factor <= 1 {
		return img
	}

	out := img.blank((img.W+factor-1)/factor, (img.H+factor-1)/factor)

	for y := range out.H {
		for x := range out.W {
			counts := map[uint8]int{}

			for dy := range factor {
				for dx := range factor {
					sx, sy := x*factor+dx, y*factor+dy
					if sx < img.W && sy < img.H && img.Pix[sy*img.W+sx] != 0 {
						counts[img.Pix[sy*img.W+sx]]++
					}
				}
			}

			var best uint8
			for colour, count := range counts {
				if count > counts[best] || (count == counts[best] && colour < best) {
					best = colour
				}
			}

			out.Pix[y*out.W+x] = best
		}
	}

	return out
}

// Dim returns the picture as seen at night: every colour of the 256-colour
// cube a step darker, greys darker too, the transparent left alone. A
// picture with its own palette has every palette colour darkened instead.
func (img Image) Dim() Image {
	out := img.Clone()

	if out.Palette != nil {
		for index := 1; index < len(out.Palette); index++ {
			for channel := range 3 {
				out.Palette[index][channel] = uint8(int(out.Palette[index][channel]) * 2 / 3)
			}
		}

		return out
	}

	for index, colour := range out.Pix {
		switch {
		case colour == 0:
		case colour >= 16 && colour <= 231:
			c := int(colour) - 16
			r, g, b := c/36, (c/6)%6, c%6
			r, g, b = max(r-1, 0), max(g-1, 0), max(b-1, 0)
			out.Pix[index] = uint8(16 + r*36 + g*6 + b)
		case colour >= 232:
			out.Pix[index] = uint8(max(int(colour)-5, 232))
		}
	}

	return out
}
