package pix

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmbeddedPictures(t *testing.T) {
	scene, ok := Load("forest1")
	if !ok || scene.W != 64 || scene.H != 32 {
		t.Fatalf("forest1 = %dx%d, ok=%v", scene.W, scene.H, ok)
	}

	if _, ok := Load("nope"); ok {
		t.Error("a missing picture loaded")
	}
}

func TestDrawSkipsTransparentAndOutside(t *testing.T) {
	canvas := Image{W: 3, H: 2, Pix: []uint8{1, 1, 1, 1, 1, 1}}
	sprite := Image{W: 2, H: 2, Pix: []uint8{9, 0, 0, 9}}

	drawn := canvas.Clone()
	drawn.Draw(sprite, 2, 0)
	drawn.Draw(sprite, 0, 0)

	want := []uint8{9, 1, 9, 1, 9, 1}
	if string(drawn.Pix) != string(want) {
		t.Errorf("pixels = %v, want %v", drawn.Pix, want)
	}

	if canvas.Pix[0] != 1 {
		t.Error("drawing on a clone changed the original")
	}
}

func TestRender(t *testing.T) {
	image := Image{W: 4, H: 3, Pix: []uint8{
		5, 5, 0, 0,
		7, 7, 0, 3,
		2, 2, 2, 2,
	}}

	lines := strings.Split(image.Render(), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	if want := "\x1b[38;5;5;48;5;7m▀▀\x1b[0m \x1b[38;5;3m▄\x1b[0m"; lines[0] != want {
		t.Errorf("line 0 = %q, want %q", lines[0], want)
	}

	if want := "\x1b[38;5;2m▀▀▀▀\x1b[0m"; lines[1] != want {
		t.Errorf("line 1 = %q, want %q", lines[1], want)
	}
}

func TestRenderQuadrants(t *testing.T) {
	// Red, green and blue in the terminal palette; the cell with three
	// colours keeps the two most frequent, the odd pixel joining the
	// nearer of them (here the dark red, nearer to red than to blue).
	image := Image{W: 6, H: 2, CellW: 2, CellH: 2, Pix: []uint8{
		9, 0, 9, 9, 9, 52,
		0, 0, 12, 12, 12, 12,
	}}

	cells := strings.Split(strings.ReplaceAll(image.Render(), "\x1b[0m", "\x1b[0m\n"), "\n")
	want := []string{
		"\x1b[38;5;9m▘\x1b[0m",
		"\x1b[38;5;9;48;5;12m▀\x1b[0m",
		"\x1b[38;5;12;48;5;9m▄\x1b[0m",
	}

	for index, cell := range want {
		if index >= len(cells) || cells[index] != cell {
			t.Errorf("cell %d = %q, want %q", index, cells[index], cell)
		}
	}

	blank := Image{W: 2, H: 2, CellW: 2, CellH: 2, Pix: []uint8{0, 0, 0, 0}}
	if blank.Render() != " " {
		t.Errorf("empty cell = %q", blank.Render())
	}

	full := Image{W: 2, H: 2, CellW: 2, CellH: 2, Pix: []uint8{4, 4, 4, 4}}
	if want := "\x1b[38;5;4m█\x1b[0m"; full.Render() != want {
		t.Errorf("full cell = %q, want %q", full.Render(), want)
	}
}

func TestRenderSextants(t *testing.T) {
	image := Image{W: 2, H: 3, CellW: 2, CellH: 3, Pix: []uint8{
		1, 0,
		1, 0,
		1, 0,
	}}
	if want := "\x1b[38;5;1m▌\x1b[0m"; image.Render() != want {
		t.Errorf("left column = %q, want %q", image.Render(), want)
	}

	image.Pix = []uint8{1, 1, 0, 0, 0, 0}
	if want := "\x1b[38;5;1m\U0001FB02\x1b[0m"; image.Render() != want {
		t.Errorf("top row = %q, want %q", image.Render(), want)
	}

	// The majority colour 2 sets bits 2, 4, 8 and 16: pattern 30, past the
	// first gap in the block.
	image.Pix = []uint8{1, 2, 2, 2, 2, 1}
	if want := "\x1b[38;5;2;48;5;1m\U0001FB1C\x1b[0m"; image.Render() != want {
		t.Errorf("mixed cell = %q, want %q", image.Render(), want)
	}
}

func TestPaletteColours(t *testing.T) {
	Configure("quad", true)
	defer Configure("quad", true)

	image := Image{W: 2, H: 2, CellW: 2, CellH: 2, Pix: []uint8{1, 1, 2, 2}, Palette: [][3]uint8{{}, {10, 20, 30}, {250, 0, 0}}}
	if want := "\x1b[38;2;10;20;30;48;2;250;0;0m▀\x1b[0m"; image.Render() != want {
		t.Errorf("truecolor = %q, want %q", image.Render(), want)
	}

	Configure("half", false)
	if want := "\x1b[38;5;233;48;5;196m▀\x1b[0m"; image.Render() != want {
		t.Errorf("256 colours = %q, want %q", image.Render(), want)
	}

	if w, h := Cell(); w != 1 || h != 2 {
		t.Errorf("cell = %dx%d after half", w, h)
	}

	Configure("sextant", true)
	if w, h := Cell(); w != 2 || h != 3 {
		t.Errorf("cell = %dx%d after sextant", w, h)
	}
}

func TestDrawOntoPalette(t *testing.T) {
	canvas := Image{W: 2, H: 1, Pix: []uint8{1, 1}, Palette: [][3]uint8{{}, {0, 0, 0}}}
	sprite := Image{W: 1, H: 1, Pix: []uint8{9}}

	canvas.Draw(sprite, 1, 0)
	if canvas.Pix[1] != 2 || canvas.Palette[2] != [3]uint8{255, 0, 0} {
		t.Errorf("red was not added to the palette: %v %v", canvas.Pix, canvas.Palette)
	}

	canvas.Set(0, 0, 9)
	if canvas.Pix[0] != 2 || len(canvas.Palette) != 3 {
		t.Errorf("a known colour was added again: %v %v", canvas.Pix, canvas.Palette)
	}

	plain := New(1, 1)
	plain.Draw(canvas, 0, 0)
	if plain.Pix[0] != 196 {
		t.Errorf("red onto a plain picture = %d, want 196", plain.Pix[0])
	}

	dark := canvas.Dim()
	if dark.Palette[2] != [3]uint8{170, 0, 0} || canvas.Palette[2] != [3]uint8{255, 0, 0} {
		t.Errorf("dim palette = %v, original %v", dark.Palette, canvas.Palette)
	}
}

func TestEncodeDecode(t *testing.T) {
	image := Image{W: 3, H: 2, CellW: 2, CellH: 3, Pix: []uint8{1, 2, 0, 2, 1, 0}, Palette: [][3]uint8{{}, {1, 2, 3}, {4, 5, 6}}}

	path := filepath.Join(t.TempDir(), "picture.px")
	if err := WriteFile(path, image); err != nil {
		t.Fatal(err)
	}

	back, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if back.W != 3 || back.H != 2 || back.CellW != 2 || back.CellH != 3 || string(back.Pix) != string(image.Pix) || len(back.Palette) != 3 || back.Palette[2] != [3]uint8{4, 5, 6} {
		t.Errorf("round trip = %+v", back)
	}

	old, err := Decode([]byte{2, 1, 7, 8})
	if err != nil || old.W != 2 || old.H != 1 || old.Palette != nil || old.CellW != 0 {
		t.Errorf("old layout = %+v, %v", old, err)
	}

	if _, err := Decode([]byte{2, 2, 7}); err == nil {
		t.Error("a truncated picture decoded")
	}
}
