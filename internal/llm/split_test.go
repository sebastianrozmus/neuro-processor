package llm

import "testing"

func TestSplitMakesWorldsWithTheWorldModel(t *testing.T) {
	mind, voice, maker := &Client{Model: "mind"}, &Client{Model: "voice"}, &Client{Model: "maker"}

	if got := (Split{Mind: mind, Voice: voice}).maker(); got != mind {
		t.Fatalf("without a World model worlds come from %q, want the Mind", got.Model)
	}

	split := Split{Mind: mind, Voice: voice, World: maker}
	if got := split.maker(); got != maker {
		t.Fatalf("worlds come from %q, want the World model", got.Model)
	}

	if split.VoiceModel() != "voice" || split.RewriteModel() != "mind" {
		t.Fatalf("VoiceModel %q, RewriteModel %q", split.VoiceModel(), split.RewriteModel())
	}

	if other := split.RewriterWith("other").(*Client); other.Model != "other" || other.BaseURL != mind.BaseURL {
		t.Fatalf("RewriterWith gave %+v", other)
	}
}
