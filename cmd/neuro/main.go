package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebastianrozmus/neuro-processor/internal/avatars"
	"github.com/sebastianrozmus/neuro-processor/internal/config"
	"github.com/sebastianrozmus/neuro-processor/internal/gallery"
	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/inline"
	"github.com/sebastianrozmus/neuro-processor/internal/llm"
	"github.com/sebastianrozmus/neuro-processor/internal/pix"
	"github.com/sebastianrozmus/neuro-processor/internal/server"
	"github.com/sebastianrozmus/neuro-processor/internal/speech"
	"github.com/sebastianrozmus/neuro-processor/internal/ui"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const dataDirName = "neuro-processor"

func main() {
	listen := flag.String("listen", "", "udostępnij świat innym graczom przez SSH pod tym adresem, np. :2222")
	name := flag.String("name", os.Getenv("USER"), "proponowane imię twojej postaci")
	flag.Parse()

	if err := run(*listen, *name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(listen, name string) error {
	cfg := config.Load()
	pix.Configure(cfg.Blocks, cfg.TrueColor)
	// Real pictures go to the local terminal alone; guests over SSH get block art.
	inline.Configure(cfg.Images, os.Stdout)

	var (
		brain     game.Brain = llm.Mock{}
		generator ui.Generator
		backend   = "brak (tryb offline)"
	)

	var client *llm.Client

	if !cfg.Offline() {
		client = &llm.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model, MaxTokens: cfg.MaxTokens}
		brain, generator, backend = client, client, cfg.Model

		// A second model may do the talking while the first keeps reading.
		if cfg.TalkModel != "" {
			voice := &llm.Client{BaseURL: cfg.TalkBaseURL, APIKey: cfg.TalkAPIKey, Model: cfg.TalkModel, MaxTokens: cfg.MaxTokens}
			split := llm.Split{Mind: client, Voice: voice}
			brain, generator, backend = split, split, cfg.Model+" + "+cfg.TalkModel
		}

		if warm, ok := brain.(interface {
			Local() bool
			Warm(context.Context)
		}); ok && warm.Local() {
			go warm.Warm(context.Background())
		}
	}

	dataDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	dataDir = filepath.Join(dataDir, dataDirName)
	worldsDir := filepath.Join(dataDir, "worlds")
	host := &host{brain: brain, name: name, listen: listen, dataDir: dataDir}

	// A nil speaker would still be a non-nil Voice, so it is only set
	// when there is something to speak with.
	if cfg.Speech {
		speaker := speech.New(cfg.Voice)
		if cfg.SpeechCommand != "" {
			speaker = speech.NewCommand(cfg.SpeechCommand)
		}

		if speaker != nil {
			host.voice = speaker
		}
	}

	if cfg.Talks {
		host.record = recorder(filepath.Join(dataDir, "talks.jsonl"))
	}

	host.library = avatars.Open(filepath.Join(dataDir, "avatars"))

	python := cfg.Python
	if python == "" {
		python = filepath.Join(dataDir, "venv", "bin", "python3")
	}

	painter := avatars.Painter{Python: python, Dir: filepath.Join(dataDir, "avatars")}
	if client != nil {
		painter.Translate = client.Translate
	}

	if painter.Available() {
		host.painter = painter

		// Generated worlds get pictures of their own, painted in the
		// background the first time they are played.
		if cfg.WorldArt {
			host.gallery = gallery.Open(filepath.Join(dataDir, "art"))
			host.gallery.Translate = painter.Translate
			host.brush = painter
			ui.UseGallery(host.gallery)
		}
	}

	if ear := (&speech.Listener{Python: python, Dir: filepath.Join(dataDir, "speech")}); cfg.Hearing && ear.Available() {
		host.ear = ear
	}

	presets, err := world.Presets()
	if err != nil {
		return err
	}

	menu := ui.NewMenu(game.ReadStates(host.savesDir()), presets, world.SavedSpecs(worldsDir), generator, worldsDir, backend, host)
	_, err = tea.NewProgram(menu, tea.WithAltScreen()).Run()

	return err
}

// host runs the worlds picked in the menu: it creates the realm, opens it
// to guests when asked to and lets the local player in.
type host struct {
	brain   game.Brain
	voice   ui.Voice
	record  ui.Record
	library *avatars.Library
	painter avatars.Generator
	gallery *gallery.Gallery
	brush   gallery.Brush
	ear     ui.Ear
	name    string
	listen  string
	dataDir string
}

func (h *host) savesDir() string {
	return filepath.Join(h.dataDir, "saves")
}

// New starts a fresh game and asks the local player who they want to be.
func (h *host) New(spec world.Spec) (tea.Model, tea.Cmd, error) {
	built, err := world.Build(spec)
	if err != nil {
		return nil, nil, err
	}

	realm := game.NewRealm(built, time.Now().UnixNano())
	h.paint(spec)

	invite, err := h.open(realm)
	if err != nil {
		return nil, nil, err
	}

	setup := ui.NewSetup(spec.Name, h.name, h.library, h.painter, func(name, avatar string) (tea.Model, tea.Cmd) {
		return h.play(realm, realm.Join(name, avatar), invite)
	}).Inline(inline.Enabled())

	return setup, setup.Init(), nil
}

// Load resumes a saved game as the hero it was saved by.
func (h *host) Load(state game.State) (tea.Model, tea.Cmd, error) {
	realm, err := game.Restore(state, time.Now().UnixNano())
	if err != nil {
		return nil, nil, err
	}

	invite, err := h.open(realm)
	if err != nil {
		return nil, nil, err
	}

	h.paint(state.Spec)
	scene, cmd := h.play(realm, realm.Join(state.Host, ""), invite)

	return scene, cmd, nil
}

// paint has the pictures of a generated world painted in the background,
// unless they already are. The screens pick them up as they appear.
func (h *host) paint(spec world.Spec) {
	if h.gallery == nil || !gallery.Wanted(spec) || h.gallery.Painted(spec) {
		return
	}

	go func() {
		_ = h.gallery.Paint(context.Background(), spec, h.brush)
	}()
}

// play gives the local player their interface. Only they can save the
// game, and they are told how guests can join. The world's fights get a
// game master when a model is configured; canned narration would add
// nothing to the rules' own lines.
func (h *host) play(realm *game.Realm, hero, invite string) (tea.Model, tea.Cmd) {
	if _, offline := h.brain.(llm.Mock); !offline {
		realm.SetMaster(h.brain, h.record)
	}

	save := func() (string, error) {
		return game.WriteState(h.savesDir(), realm.State(hero))
	}

	if invite != "" {
		realm.Tell(hero, "Świat jest otwarty dla innych graczy: "+invite)
	}

	scene := ui.NewGame(realm, hero, h.brain, h.library, save, h.voice, h.ear, h.record).Inline(inline.Enabled())

	return scene, scene.Init()
}

// open lets guests join the realm over SSH, if the game was started with
// an address to listen on. It returns the command guests join with.
func (h *host) open(realm *game.Realm) (string, error) {
	if h.listen == "" {
		return "", nil
	}

	addr, err := server.Start(h.listen, filepath.Join(h.dataDir, "host_key"), func(user string) (tea.Model, func()) {
		var (
			mu    sync.Mutex
			guest string
		)

		// Guests pick from the host's avatars but cannot have new ones drawn.
		setup := ui.NewSetup(realm.World().Name, user, h.library, nil, func(name, avatar string) (tea.Model, tea.Cmd) {
			mu.Lock()
			defer mu.Unlock()

			guest = realm.Join(name, avatar)
			// Guests are elsewhere: the host's speakers would not reach them.
			scene := ui.NewGame(realm, guest, h.brain, h.library, nil, nil, nil, h.record)

			return scene, scene.Init()
		})

		return setup, func() {
			mu.Lock()
			defer mu.Unlock()

			if guest != "" {
				realm.Leave(guest)
			}
		}
	})
	if err != nil {
		return "", err
	}

	return invitation(addr), nil
}

// recorder appends every well-answered exchange to a JSON Lines file, one
// conversation per line in the shape the model was prompted with. Failures
// to write are ignored: a lost record must never stop a game.
func recorder(path string) ui.Record {
	var mu sync.Mutex

	return func(request game.NarrateRequest, reply string) {
		mu.Lock()
		defer mu.Unlock()

		line, err := json.Marshal(map[string]any{
			"time":     time.Now().Format(time.RFC3339),
			"keeper":   request.Merchant.Name,
			"outcome":  request.Decision.Outcome,
			"messages": llm.Transcript(request, reply),
		})
		if err != nil {
			return
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return
		}

		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		defer file.Close()

		_, _ = file.Write(append(line, '\n'))
	}
}

// invitation words the command guests use to join the world.
func invitation(addr net.Addr) string {
	_, port, _ := net.SplitHostPort(addr.String())
	host := "ADRES-TEGO-KOMPUTERA"

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, candidate := range addrs {
			network, ok := candidate.(*net.IPNet)
			if ok && !network.IP.IsLoopback() && network.IP.To4() != nil {
				host = network.IP.String()

				break
			}
		}
	}

	return fmt.Sprintf("ssh -p %s IMIĘ@%s", port, host)
}
