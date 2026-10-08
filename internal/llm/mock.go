package llm

import (
	"context"

	"github.com/sebastianrozmus/neuro-processor/internal/game"
)

// Mock is an offline backend: it reads player text with the game's own
// rules and replies with canned lines. It lets the game run without a model.
type Mock struct{}

// Interpret reads the player text with rules alone.
func (Mock) Interpret(_ context.Context, request game.InterpretRequest) (game.Intent, error) {
	return game.Read(request).Alone(), nil
}

// Narrate returns the canned line for the decision.
func (Mock) Narrate(_ context.Context, request game.NarrateRequest) (string, error) {
	return request.Decision.Fallback(), nil
}
