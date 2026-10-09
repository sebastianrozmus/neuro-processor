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

	"github.com/sebastianrozmus/neuro-processor/internal/gallery"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
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
	// Picture is the large portrait as a PNG, for terminals that show
	// real images; built-in portraits have none.
	Picture []byte
}

// Art returns the small portrait with the given ID, or an empty string.
func (l *Library) Art(id string) string {
	return l.read(id, "")
}

// Picture returns the PNG of a player-made portrait and the file it is
// in, or nothing for a built-in one.
func (l *Library) Picture(id string) ([]byte, string) {
	if l.dir == "" || id == "" || id != world.Slug(id) {
		return nil, ""
	}

	path := filepath.Join(l.dir, id+".png")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ""
	}

	return data, path
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

	if len(drawing.Picture) > 0 {
		if err := os.WriteFile(filepath.Join(l.dir, id+".png"), drawing.Picture, 0o644); err != nil {
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

// Generate draws a portrait and returns it rendered in block characters.
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
	small := filepath.Join(p.Dir, fmt.Sprintf("draft-%d.px", stamp))
	large := filepath.Join(p.Dir, fmt.Sprintf("draft-%d-big.px", stamp))
	picture := strings.TrimSuffix(large, ".px") + ".png"
	defer os.Remove(small)
	defer os.Remove(large)
	defer os.Remove(picture)
	defer os.Remove(strings.TrimSuffix(small, ".px") + ".png")

	ctx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()

	// The picture is drawn for the block characters the game renders with.
	cellW, cellH := pix.Cell()
	cmd := exec.CommandContext(ctx, p.Python, "-", "--prompt", prompt, "--out", small, "--big", large,
		"--seed", fmt.Sprint(stamp%1000000), "--kind", "portrait", "--cell", fmt.Sprintf("%dx%d", cellW, cellH))
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
		art, err := pix.ReadFile(path)
		if err != nil {
			return Drawing{}, err
		}

		*target = art.Render()
	}

	// The PNG is a bonus for terminals that show real images.
	drawing.Picture, _ = os.ReadFile(picture)

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

// paintTimeout bounds painting a whole world's pictures.
const paintTimeout = 40 * time.Minute

// Paint draws a world's pictures in one run of the image model, each into
// the PX2 file its job names.
func (p Painter) Paint(ctx context.Context, jobs []gallery.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return err
	}

	list, err := os.CreateTemp(p.Dir, "jobs-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(list.Name())

	type job struct {
		Kind   string `json:"kind"`
		Prompt string `json:"prompt"`
		Out    string `json:"out"`
		Cells  [2]int `json:"cells"`
		Seed   int    `json:"seed"`
	}

	wanted := make([]job, 0, len(jobs))
	for _, item := range jobs {
		wanted = append(wanted, job{Kind: item.Kind, Prompt: item.Prompt, Out: item.Out, Cells: item.Cells, Seed: item.Seed})
	}

	if err := json.NewEncoder(list).Encode(wanted); err != nil {
		list.Close()

		return err
	}

	if err := list.Close(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, paintTimeout)
	defer cancel()

	cellW, cellH := pix.Cell()
	cmd := exec.CommandContext(ctx, p.Python, "-", "--jobs", list.Name(), "--cell", fmt.Sprintf("%dx%d", cellW, cellH))
	cmd.Stdin = strings.NewReader(generator)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if message := lastLine(stderr.String()); message != "" {
			return errors.New(message)
		}

		return err
	}

	return nil
}
