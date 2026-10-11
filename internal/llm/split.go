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
// (mlx_lm.server among them) are fine as a Voice.
type Split struct {
	Mind  *Client
	Voice *Client
}

// Interpret reads a player message with the Mind.
func (s Split) Interpret(ctx context.Context, request game.InterpretRequest) (game.Intent, error) {
	return s.Mind.Interpret(ctx, request)
}

// Translate puts a text into English with the Mind.
func (s Split) Translate(ctx context.Context, text string) (string, error) {
	return s.Mind.Translate(ctx, text)
}

// Note asks the Mind whether a line brought a new fact.
func (s Split) Note(ctx context.Context, request game.NoteRequest) (game.FactSpec, bool, error) {
	return s.Mind.Note(ctx, request)
}

// Rewrite has the Mind word a description again.
func (s Split) Rewrite(ctx context.Context, request game.RewriteRequest) (string, error) {
	return s.Mind.Rewrite(ctx, request)
}

// Invent asks the Mind whether a line offered a new thing.
func (s Split) Invent(ctx context.Context, request game.InventRequest) (world.ItemSpec, bool, error) {
	return s.Mind.Invent(ctx, request)
}

// GenerateWorld makes a world with the Mind.
func (s Split) GenerateWorld(ctx context.Context, mode world.Mode, theme string) (world.Spec, error) {
	return s.Mind.GenerateWorld(ctx, mode, theme)
}

// Narrate words a reply with the Voice.
func (s Split) Narrate(ctx context.Context, request game.NarrateRequest) (string, error) {
	return s.Voice.Narrate(ctx, request)
}

// Stream words a reply with the Voice, piece by piece.
func (s Split) Stream(ctx context.Context, request game.NarrateRequest, deliver func(piece string)) (string, error) {
	return s.Voice.Stream(ctx, request, deliver)
}

// Local reports whether either model runs on this machine.
func (s Split) Local() bool {
	return s.Mind.Local() || s.Voice.Local()
}

// Warm loads both models.
func (s Split) Warm(ctx context.Context) {
	s.Mind.Warm(ctx)
	s.Voice.Warm(ctx)
}
