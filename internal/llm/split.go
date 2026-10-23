package llm

import (
	"context"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Split is a brain with two models: Mind reads intents, spots new things
// and facts and generates worlds, all of which needs JSON kept to a
// schema; Voice words the characters' lines, which is what the game's own
// fine-tuned model is trained for. Servers that cannot enforce a schema
// (mlx_lm.server among them) are fine as a Voice. World, when set, takes
// generating worlds over from the Mind.
type Split struct {
	Mind  *Client
	Voice *Client
	World *Client
	// Master, when set, narrates fights instead of the Voice; Images,
	// when set, words the pictures' descriptions in English instead of
	// the Mind.
	Master *Client
	Images *Client
}

// The roles of the models, as the settings and the trace name them.
const (
	RoleTalk   = "rozmowy"
	RoleFight  = "walka"
	RoleMind   = "umysł"
	RoleWorlds = "światy"
	RoleImages = "obrazki"
)

// Roles are the roles in the order the settings list them.
var Roles = []string{RoleTalk, RoleFight, RoleMind, RoleWorlds, RoleImages}

// Choice is the model of one role, as the player picked it.
type Choice struct {
	Model string `json:"model"`
	CPU   bool   `json:"cpu,omitempty"`
}

// Client is the client of a role.
func (s Split) Client(role string) *Client {
	switch role {
	case RoleTalk:
		return s.Voice
	case RoleFight:
		return s.Master
	case RoleMind:
		return s.Mind
	case RoleWorlds:
		return s.World
	case RoleImages:
		return s.Images
	}

	return nil
}

// Choices are the models of all roles.
func (s Split) Choices() map[string]Choice {
	choices := map[string]Choice{}

	for _, role := range Roles {
		if client := s.Client(role); client != nil {
			model, cpu := client.Name()
			choices[role] = Choice{Model: model, CPU: cpu}
		}
	}

	return choices
}

// Use switches the roles to the models picked; roles left out or without
// a model keep theirs.
func (s Split) Use(choices map[string]Choice) {
	for role, choice := range choices {
		if client := s.Client(role); client != nil && choice.Model != "" {
			client.Use(choice.Model, choice.CPU)
		}
	}
}

// Narrator is who narrates fights: the Master, or the Voice.
func (s Split) Narrator() game.Narrator {
	if s.Master != nil {
		return s.Master
	}

	return s.Voice
}

// Interpret reads a player message with the Mind.
func (s Split) Interpret(ctx context.Context, request game.InterpretRequest) (game.Intent, error) {
	return s.Mind.Interpret(ctx, request)
}

// Translate puts a text into English with the Images model, or the Mind.
func (s Split) Translate(ctx context.Context, text string) (string, error) {
	if s.Images != nil {
		return s.Images.Translate(ctx, text)
	}

	return s.Mind.Translate(ctx, text)
}

// Note asks the Mind whether a line brought a new fact.
func (s Split) Note(ctx context.Context, request game.NoteRequest) (game.FactSpec, bool, error) {
	return s.Mind.Note(ctx, request)
}

// SelfNote asks the Mind what a companion said about themselves.
func (s Split) SelfNote(ctx context.Context, request game.SelfNoteRequest) ([]string, error) {
	return s.Mind.SelfNote(ctx, request)
}

// QuestHook asks the Mind whether a keeper called for help.
func (s Split) QuestHook(ctx context.Context, request game.QuestHookRequest) (game.QuestHook, error) {
	return s.Mind.QuestHook(ctx, request)
}

// Rewrite has the Mind word a description again.
func (s Split) Rewrite(ctx context.Context, request game.RewriteRequest) (string, error) {
	return s.Mind.Rewrite(ctx, request)
}

// Invent asks the Mind whether a line offered a new thing.
func (s Split) Invent(ctx context.Context, request game.InventRequest) (world.ItemSpec, bool, error) {
	return s.Mind.Invent(ctx, request)
}

// GenerateWorld makes a world with the World model, or the Mind when
// there is none.
func (s Split) GenerateWorld(ctx context.Context, mode world.Mode, theme string) (world.Spec, error) {
	return s.maker().GenerateWorld(ctx, mode, theme)
}

func (s Split) maker() *Client {
	if s.World != nil {
		return s.World
	}

	return s.Mind
}

// VoiceModel names the model that words the lines.
func (s Split) VoiceModel() string {
	return s.Voice.model()
}

// RewriteModel names the model the inspector rewrites with by default.
func (s Split) RewriteModel() string {
	return s.Mind.model()
}

// RewriterWith rewrites with the named model on the Mind's server.
func (s Split) RewriterWith(model string) game.Rewriter {
	return s.Mind.WithModel(model)
}

// Models lists what the Mind's server can run.
func (s Split) Models(ctx context.Context) ([]string, error) {
	return s.Mind.Models(ctx)
}

// Narrate words a reply with the Voice.
func (s Split) Narrate(ctx context.Context, request game.NarrateRequest) (string, error) {
	return s.Voice.Narrate(ctx, request)
}

// Stream words a reply with the Voice, piece by piece.
func (s Split) Stream(ctx context.Context, request game.NarrateRequest, deliver func(piece string)) (string, error) {
	return s.Voice.Stream(ctx, request, deliver)
}

// Local reports whether any of the models runs on this machine.
func (s Split) Local() bool {
	return s.Mind.Local() || s.Voice.Local() || s.maker().Local()
}

// Warm loads the models, each once.
func (s Split) Warm(ctx context.Context) {
	warmed := map[Choice]bool{}

	// Worlds are made rarely, at the start, and pictures described
	// rarely; loading more models only for that would push the others
	// out of memory.
	for _, client := range []*Client{s.Mind, s.Voice, s.Master} {
		if client == nil {
			continue
		}

		model, cpu := client.Name()
		if !warmed[Choice{model, cpu}] {
			warmed[Choice{model, cpu}] = true
			client.Warm(ctx)
		}
	}
}
