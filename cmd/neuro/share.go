package main

import (
	"context"

	"github.com/sebastianrozmus/neuro-processor/internal/settings"
	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// sharing sends conversations to the collecting server for the pause menu
// (Esc → Udostępnij rozmowy). The consent and a deletion still owed live
// in settings.json, the installation's key in the data directory, made at
// the first send.
type sharing struct {
	client  *share.Client
	dataDir string
	path    string
}

type shareSettings struct {
	Consent bool `json:"consent"`
	// ForgetPending is a withdrawal the server has not confirmed yet: the
	// menu offers to try again, without asking for consent anew.
	ForgetPending bool `json:"forget_pending,omitempty"`
}

func (s *sharing) state() shareSettings {
	var state shareSettings

	settings.Load(s.path, "share", &state)

	return state
}

func (s *sharing) Consented() bool { return s.state().Consent }

func (s *sharing) ForgetPending() bool { return s.state().ForgetPending }

func (s *sharing) SetConsent(yes bool) error {
	state := s.state()
	state.Consent = yes

	if !yes {
		// The withdrawal is owed from this moment until the server confirms.
		state.ForgetPending = true
	}

	return settings.Save(s.path, "share", state)
}

func (s *sharing) Game() string { return "neuro " + version }

func (s *sharing) Send(ctx context.Context, payload share.Payload) error {
	// Nothing goes with a key whose withdrawal is still owed.
	if state := s.state(); !state.Consent || state.ForgetPending {
		return share.ErrRejected
	}

	id, err := share.LoadIdentity(s.dataDir)
	if err != nil {
		return err
	}

	return s.client.Send(ctx, id, payload)
}

// Forget asks the server to delete what this installation sent and, once
// it confirmed, drops the key, so that anything sent later is not linked
// to it. Without a key nothing was ever sent.
func (s *sharing) Forget(ctx context.Context) error {
	id, ok, err := share.ReadIdentity(s.dataDir)
	if err != nil {
		return err
	}

	if ok {
		if err := s.client.Forget(ctx, id); err != nil {
			return err
		}

		if err := share.ForgetIdentity(s.dataDir, id); err != nil {
			return err
		}
	}

	state := s.state()
	state.ForgetPending = false

	return settings.Save(s.path, "share", state)
}
