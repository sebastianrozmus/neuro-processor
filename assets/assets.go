// Package assets embeds the art shipped with the game.
package assets

import (
	"embed"
	"strings"
)

//go:embed ascii px
var files embed.FS

// Art returns the ANSI-coloured art stored under the given name.
func Art(name string) (string, error) {
	data, err := files.ReadFile("ascii/" + name + ".txt")
	if err != nil {
		return "", err
	}

	// A checkout with Windows line endings must not leave a carriage return
	// at the end of every row of art.
	return strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), nil
}

// Pixels returns the raw pixel art stored under the given name: a width
// byte, a height byte and one palette index per pixel, 0 meaning transparent.
func Pixels(name string) ([]byte, error) {
	return files.ReadFile("px/" + name + ".px")
}
