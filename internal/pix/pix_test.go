package pix

import (
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
