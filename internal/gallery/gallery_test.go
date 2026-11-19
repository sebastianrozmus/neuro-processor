package gallery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// brush paints every job as a two-pixel picture and remembers the jobs.
type brush struct {
	jobs []Job
	fail bool
}

func (b *brush) Paint(_ context.Context, jobs []Job) error {
	if b.fail {
		return errors.New("no model")
	}

	b.jobs = append(b.jobs, jobs...)
	for _, job := range jobs {
		image := pix.New(1, 2)
		image.Set(0, 0, 196)
		image.Set(0, 1, 21)

		if err := pix.WriteFile(job.Out, image); err != nil {
			return err
		}
	}

	return nil
}

func testSpec() world.Spec {
	return world.Spec{
		Name:  "Czerwona Mgła",
		Intro: "Miasto w deszczu.",
		Style: &world.StyleSpec{Look: "Neon.", Art: "neon, rain, 90s anime"},
		Boss:  world.BossSpec{Name: "Baron", Description: "Siedzi w porcie."},
		Places: []world.PlaceSpec{
			{Name: "Doki", Description: "Dźwigi i mgła.", Terrain: "street"},
			{Name: "Stacja", Description: "Puste perony.", Terrain: "station", Danger: 1},
		},
		Shops: []world.ShopSpec{
			{Kind: world.ShopArmory, Keeper: "Rudy", Persona: "Milczek."},
			{Kind: world.ShopAlchemist, Keeper: "Lena", Persona: "Gada bez przerwy."},
		},
	}
}

func TestJobsFollowTheStyleCard(t *testing.T) {
	g := Open(t.TempDir())
	g.Translate = func(_ context.Context, text string) (string, error) {
		return "EN(" + text + ")", nil
	}

	jobs := g.Jobs(context.Background(), testSpec())
	if len(jobs) != Backdrops+Variants*2+2 {
		t.Fatalf("%d jobs", len(jobs))
	}

	for _, job := range jobs {
		if !strings.HasSuffix(job.Prompt, ", neon, rain, 90s anime") || job.Cells[0] == 0 || job.Out == "" {
			t.Errorf("job %+v", job)
		}
	}

	if jobs[0].Kind != "backdrop" || jobs[0].Prompt != "EN(Miasto w deszczu.), neon, rain, 90s anime" || jobs[0].Out != "backdrop-1.px" {
		t.Errorf("backdrop job %+v", jobs[0])
	}

	if scene := jobs[Backdrops]; scene.Kind != "scene" || scene.Prompt != "street, EN(Doki. Dźwigi i mgła.), neon, rain, 90s anime" || scene.Out != "place-1-1.px" || scene.Cells != [2]int{SceneCols, SceneRows} {
		t.Errorf("scene job %+v", scene)
	}

	if keeper := jobs[len(jobs)-1]; keeper.Kind != "portrait" || !strings.HasPrefix(keeper.Prompt, "herbalist healer, EN(Lena. Gada bez przerwy.)") || keeper.Out != "keeper-alchemist.px" {
		t.Errorf("keeper job %+v", keeper)
	}

	if seeds := map[int]bool{}; true {
		for _, job := range jobs {
			if seeds[job.Seed] {
				t.Errorf("seed %d used twice", job.Seed)
			}

			seeds[job.Seed] = true
		}
	}

	// Without a translator the Polish goes to the painter as it is.
	plain := Open(t.TempDir()).Jobs(context.Background(), testSpec())
	if plain[0].Prompt != "Miasto w deszczu., neon, rain, 90s anime" {
		t.Errorf("untranslated prompt %q", plain[0].Prompt)
	}
}

func TestPaintFillsTheGallery(t *testing.T) {
	spec := testSpec()
	g := Open(t.TempDir())

	if g.Painted(spec) || g.Backdrop(spec, 0) != "" || g.Keeper(spec, world.ShopArmory) != "" {
		t.Fatal("an empty gallery should have nothing")
	}

	if _, ok := g.Scene(spec, 0, 1); ok {
		t.Fatal("an empty gallery should have no scene")
	}

	b := &brush{}
	if err := g.Paint(context.Background(), spec, b); err != nil {
		t.Fatal(err)
	}

	if len(b.jobs) != Backdrops+Variants*2+2 || !g.Painted(spec) {
		t.Fatalf("%d jobs painted, done %v", len(b.jobs), g.Painted(spec))
	}

	if g.Dir(spec) != filepath.Join(g.dir, "czerwona-mgla") || filepath.Dir(b.jobs[0].Out) != g.Dir(spec) {
		t.Errorf("pictures went to %s", b.jobs[0].Out)
	}

	scene, ok := g.Scene(spec, 1, 2)
	if !ok || scene.W != 1 || scene.H != 2 {
		t.Errorf("scene = %+v, %v", scene, ok)
	}

	if art := g.Backdrop(spec, 1); !strings.Contains(art, "▀") {
		t.Errorf("backdrop = %q", art)
	}

	if art := g.Keeper(spec, world.ShopAlchemist); !strings.Contains(art, "▀") {
		t.Errorf("keeper = %q", art)
	}

	if art := g.Keeper(spec, world.ShopMagic); art != "" {
		t.Errorf("a shop the world lacks got a portrait: %q", art)
	}

	// Painting again has nothing left to do.
	again := &brush{fail: true}
	if err := g.Paint(context.Background(), spec, again); err != nil {
		t.Errorf("a finished gallery asked the brush again: %v", err)
	}
}

func TestPaintSkipsWhatHasArt(t *testing.T) {
	spec := testSpec()
	spec.Backdrops = []string{"forest1"}

	if Wanted(spec) {
		t.Error("a world with embedded art should not be painted")
	}

	g := Open(t.TempDir())
	b := &brush{fail: true}
	if err := g.Paint(context.Background(), spec, b); err != nil || g.Painted(spec) {
		t.Errorf("painting a preset: %v, done %v", err, g.Painted(spec))
	}

	var none *Gallery
	if none.Painted(spec) || none.Backdrop(spec, 0) != "" || none.Paint(context.Background(), testSpec(), b) != nil {
		t.Error("a nil gallery should be inert")
	}
}

func TestPicturesBesideThePixels(t *testing.T) {
	spec := testSpec()
	g := Open(t.TempDir())

	if png, path := g.ScenePicture(spec, 0, 1); png != nil || path != "" {
		t.Error("an empty gallery should have no picture")
	}

	dir := g.Dir(spec)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "place-1-1.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "keeper-armory.png"), []byte("PNG2"), 0o644); err != nil {
		t.Fatal(err)
	}

	if png, path := g.ScenePicture(spec, 0, 1); string(png) != "PNG" || path != filepath.Join(dir, "place-1-1.png") {
		t.Errorf("scene picture = %q at %q", png, path)
	}

	if png, path := g.KeeperPicture(spec, world.ShopArmory); string(png) != "PNG2" || path != filepath.Join(dir, "keeper-armory.png") {
		t.Errorf("keeper picture = %q at %q", png, path)
	}

	if png, _ := g.KeeperPicture(spec, world.ShopMagic); png != nil {
		t.Error("a shop without a picture got one")
	}
}
