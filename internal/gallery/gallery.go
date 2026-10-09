// Package gallery keeps the pictures painted for a generated world: the
// backdrops behind the menu, a scene for every kind of place and the
// portraits of the keepers, all drawn by the local image model in the
// style of the world's card. The game shows them as soon as they appear
// and falls back to the embedded art until then.
package gallery

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Picture sizes in character cells: the menu backdrop fills a terminal,
// a scene is what the map screen shows, a keeper's portrait is the side
// panel of the shop.
const (
	BackdropCols, BackdropRows = 120, 40
	SceneCols, SceneRows       = 64, 16
	KeeperCols, KeeperRows     = 40, 20

	// Backdrops is how many pictures the menu cycles through; Variants
	// how many scenes each kind of place gets.
	Backdrops = 2
	Variants  = 2

	doneFile = "done"
)

// Job is one picture to paint: kind picks the composition the painter
// prompts for (portrait, scene or backdrop), Cells its size in characters.
type Job struct {
	Kind   string
	Prompt string
	Out    string
	Cells  [2]int
	Seed   int
}

// Brush paints jobs one after another, each into a PX2 file at Job.Out.
type Brush interface {
	Paint(ctx context.Context, jobs []Job) error
}

// Gallery is the directory the pictures of all worlds are kept in, one
// subdirectory per world.
type Gallery struct {
	dir string
	// Translate turns the world's Polish descriptions into English, which
	// image models understand far better; nil paints from the Polish.
	Translate func(ctx context.Context, text string) (string, error)

	images sync.Map
}

// Open returns the gallery kept in dir.
func Open(dir string) *Gallery {
	return &Gallery{dir: dir}
}

// Dir is where the pictures of a world live.
func (g *Gallery) Dir(spec world.Spec) string {
	return filepath.Join(g.dir, world.Slug(spec.Name))
}

// Painted reports whether every picture of the world has been painted.
func (g *Gallery) Painted(spec world.Spec) bool {
	if g == nil {
		return false
	}

	_, err := os.Stat(filepath.Join(g.Dir(spec), doneFile))

	return err == nil
}

// Wanted reports whether the world is one to paint pictures for: a
// generated one, which has no embedded art of its own.
func Wanted(spec world.Spec) bool {
	return len(spec.Backdrops) == 0 && len(spec.Places) > 0 && !spec.Mode.Rooms()
}

// Scene returns the painted picture of a kind of place, if there is one.
func (g *Gallery) Scene(spec world.Spec, place, variant int) (pix.Image, bool) {
	return g.image(spec, sceneFile(place, variant))
}

// Backdrop returns a painted menu backdrop, rendered, or an empty string.
func (g *Gallery) Backdrop(spec world.Spec, index int) string {
	return g.rendered(spec, backdropFile(index))
}

// Keeper returns the painted portrait of the keeper of a kind of shop,
// rendered, or an empty string.
func (g *Gallery) Keeper(spec world.Spec, kind world.ShopKind) string {
	return g.rendered(spec, keeperFile(kind))
}

func (g *Gallery) rendered(spec world.Spec, name string) string {
	image, ok := g.image(spec, name)
	if !ok {
		return ""
	}

	return image.Render()
}

// image reads a picture once; a file that is still being written fails to
// decode and is simply tried again next time.
func (g *Gallery) image(spec world.Spec, name string) (pix.Image, bool) {
	if g == nil {
		return pix.Image{}, false
	}

	path := filepath.Join(g.Dir(spec), name)
	if cached, ok := g.images.Load(path); ok {
		return cached.(pix.Image), true
	}

	image, err := pix.ReadFile(path)
	if err != nil {
		return pix.Image{}, false
	}

	g.images.Store(path, image)

	return image, true
}

// Paint draws every picture of the world that is still missing and marks
// the world done. Descriptions are translated first, then painted in one
// go, so that the image model is loaded once.
func (g *Gallery) Paint(ctx context.Context, spec world.Spec, brush Brush) error {
	if g == nil || brush == nil || !Wanted(spec) {
		return nil
	}

	dir := g.Dir(spec)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var jobs []Job
	for _, job := range g.Jobs(ctx, spec) {
		job.Out = filepath.Join(dir, job.Out)
		if _, err := os.Stat(job.Out); err != nil {
			jobs = append(jobs, job)
		}
	}

	if len(jobs) > 0 {
		if err := brush.Paint(ctx, jobs); err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(dir, doneFile), nil, 0o644)
}

// Jobs lists the pictures a world gets, with their prompts in English and
// file names relative to the world's directory.
func (g *Gallery) Jobs(ctx context.Context, spec world.Spec) []Job {
	style := ""
	if spec.Style != nil {
		style = spec.Style.Art
	}

	seed := int(hash(spec.Name) % 100000)
	jobs := make([]Job, 0, Backdrops+Variants*len(spec.Places)+len(spec.Shops))

	intro := g.english(ctx, spec.Intro)
	lair := g.english(ctx, spec.Boss.Description)
	for index, subject := range []string{intro, lair} {
		jobs = append(jobs, Job{
			Kind:   "backdrop",
			Prompt: join(subject, style),
			Out:    backdropFile(index),
			Cells:  [2]int{BackdropCols, BackdropRows},
			Seed:   seed + index,
		})
	}

	for index, place := range spec.Places {
		subject := join(string(place.Terrain), g.english(ctx, place.Name+". "+place.Description))

		for variant := 1; variant <= Variants; variant++ {
			jobs = append(jobs, Job{
				Kind:   "scene",
				Prompt: join(subject, style),
				Out:    sceneFile(index, variant),
				Cells:  [2]int{SceneCols, SceneRows},
				Seed:   seed + 10 + index*Variants + variant,
			})
		}
	}

	for index, shop := range spec.Shops {
		jobs = append(jobs, Job{
			Kind:   "portrait",
			Prompt: join(trade(shop.Kind), g.english(ctx, shop.Keeper+". "+shop.Persona), style),
			Out:    keeperFile(shop.Kind),
			Cells:  [2]int{KeeperCols, KeeperRows},
			Seed:   seed + 100 + index,
		})
	}

	return jobs
}

func (g *Gallery) english(ctx context.Context, text string) string {
	text = strings.TrimSpace(text)
	if g.Translate == nil || text == "" {
		return text
	}

	if translated, err := g.Translate(ctx, text); err == nil && strings.TrimSpace(translated) != "" {
		return strings.TrimSpace(translated)
	}

	return text
}

// trade names the keeper's line of work for the painter.
func trade(kind world.ShopKind) string {
	switch kind {
	case world.ShopArmory:
		return "armorer"
	case world.ShopAlchemist:
		return "herbalist healer"
	}

	return "mysterious merchant of arcane wares"
}

func backdropFile(index int) string         { return fmt.Sprintf("backdrop-%d.px", index+1) }
func sceneFile(place, variant int) string   { return fmt.Sprintf("place-%d-%d.px", place+1, variant) }
func keeperFile(kind world.ShopKind) string { return fmt.Sprintf("keeper-%s.px", kind) }

func join(parts ...string) string {
	kept := parts[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, ", ")
}

func hash(text string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(text))

	return h.Sum32()
}
