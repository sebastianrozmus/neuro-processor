// Package avatars keeps the heroes' portraits: the ones shipped with the game
// and the ones players make themselves with a local image model.
package avatars

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

//go:embed builtin/*.txt
var builtin embed.FS

//go:embed generate.py
var generator string

const (
	indexFile       = "index.json"
	generateTimeout = 5 * time.Minute
)

// Avatar is a portrait a player can pick for their hero.
type Avatar struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Builtin lists the portraits shipped with the game.
var Builtin = []Avatar{
	{"wojownik", "Wojownik"},
	{"czarodziej", "Czarodziej"},
	{"lotrzyk", "Łotrzyk"},
	{"lowczyni", "Łowczyni"},
	{"szabrownik", "Szabrownik"},
	{"pilot", "Pilot"},
}

// Library is the built-in portraits together with those stored in a
// directory, which players add to.
type Library struct {
	dir string

	mu     sync.Mutex
	custom []Avatar
}

// Open returns the library kept in dir, creating the directory on first
// use. An empty dir gives the built-in portraits alone.
func Open(dir string) *Library {
	library := &Library{dir: dir}

	if dir != "" {
		if data, err := os.ReadFile(filepath.Join(dir, indexFile)); err == nil {
			_ = json.Unmarshal(data, &library.custom)
		}
	}

	return library
}

// Dir returns where the library keeps the players' own portraits.
func (l *Library) Dir() string {
	return l.dir
}

// List returns every portrait, the built-in ones first.
func (l *Library) List() []Avatar {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append(append([]Avatar(nil), Builtin...), l.custom...)
}

// Drawing is a portrait in the two sizes the game shows: small beside the
// hero's statistics, large on the character screen. Built-in portraits
// have only the small one.
type Drawing struct {
	Small string
	Large string
}

// Art returns the small portrait with the given ID, or an empty string.
func (l *Library) Art(id string) string {
	return l.read(id, "")
}

// Portrait returns the large portrait with the given ID, or the small one
// when there is no large.
func (l *Library) Portrait(id string) string {
	if large := l.read(id, "-big"); large != "" {
		return large
	}

	return l.read(id, "")
}

func (l *Library) read(id, suffix string) string {
	if data, err := builtin.ReadFile("builtin/" + id + suffix + ".txt"); err == nil {
		return strings.TrimRight(string(data), "\n")
	}

	if l.dir == "" || id == "" || id != world.Slug(id) {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(l.dir, id+suffix+".txt"))
	if err != nil {
		return ""
	}

	return strings.TrimRight(string(data), "\n")
}

// Add stores a portrait under a name and returns its ID. A name already in
// use gets a number appended.
func (l *Library) Add(name string, drawing Drawing) (Avatar, error) {
	if l.dir == "" {
		return Avatar{}, errors.New("nie ma gdzie zapisać awatara")
	}

	name = strings.Join(strings.Fields(name), " ")
	if name == "" || strings.TrimSpace(drawing.Small) == "" {
		return Avatar{}, errors.New("awatar potrzebuje nazwy i obrazka")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	id := world.Slug(name)
	for suffix := 2; l.taken(id); suffix++ {
		id = fmt.Sprintf("%s-%d", world.Slug(name), suffix)
	}

	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return Avatar{}, err
	}

	if err := os.WriteFile(filepath.Join(l.dir, id+".txt"), []byte(drawing.Small+"\n"), 0o644); err != nil {
		return Avatar{}, err
	}

	if drawing.Large != "" {
		if err := os.WriteFile(filepath.Join(l.dir, id+"-big.txt"), []byte(drawing.Large+"\n"), 0o644); err != nil {
			return Avatar{}, err
		}
	}

	avatar := Avatar{ID: id, Name: name}
	l.custom = append(l.custom, avatar)

	data, err := json.MarshalIndent(l.custom, "", " ")
	if err != nil {
		return Avatar{}, err
	}

	return avatar, os.WriteFile(filepath.Join(l.dir, indexFile), data, 0o644)
}

func (l *Library) taken(id string) bool {
	for _, avatar := range Builtin {
		if avatar.ID == id {
			return true
		}
	}

	for _, avatar := range l.custom {
		if avatar.ID == id {
			return true
		}
	}

	return false
}

// Generator draws a portrait from a description.
type Generator interface {
	Generate(ctx context.Context, prompt string) (Drawing, error)
}

// Painter draws portraits with the local image model, through the Python
// interpreter of an environment that has the model's libraries installed.
type Painter struct {
	Python string
	// Dir holds the files the generator writes.
	Dir string
	// Translate, when set, turns the player's description into English,
	// which image models understand far better. Failures are ignored and
	// the description is used as typed.
	Translate func(ctx context.Context, text string) (string, error)
}

// Generate draws a portrait and returns it as half-block art.
func (p Painter) Generate(ctx context.Context, prompt string) (Drawing, error) {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return Drawing{}, err
	}

	if p.Translate != nil {
		if english, err := p.Translate(ctx, prompt); err == nil && strings.TrimSpace(english) != "" {
			prompt = strings.TrimSpace(english)
		}
	}

	stamp := time.Now().UnixNano()
	small := filepath.Join(p.Dir, fmt.Sprintf("draft-%d.txt", stamp))
	large := filepath.Join(p.Dir, fmt.Sprintf("draft-%d-big.txt", stamp))
	defer os.Remove(small)
	defer os.Remove(large)

	ctx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.Python, "-", "--prompt", prompt, "--out", small, "--big", large, "--seed", fmt.Sprint(stamp%1000000))
	cmd.Stdin = strings.NewReader(generator)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if message := lastLine(stderr.String()); message != "" {
			return Drawing{}, errors.New(message)
		}

		return Drawing{}, err
	}

	var drawing Drawing

	for path, target := range map[string]*string{small: &drawing.Small, large: &drawing.Large} {
		art, err := os.ReadFile(path)
		if err != nil {
			return Drawing{}, err
		}

		*target = strings.TrimRight(string(art), "\n")
	}

	return drawing, nil
}

// Available reports whether the painter's interpreter exists.
func (p Painter) Available() bool {
	info, err := os.Stat(p.Python)

	return err == nil && !info.IsDir()
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	return strings.TrimSpace(lines[len(lines)-1])
}
