// Package server lets other players join a running world over SSH. Every
// connection gets its own interface onto the same shared realm, so guests
// need nothing but an SSH client.
package server

import (
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
)

// Join admits a connecting user. It returns the interface to show them and
// a function called when they disconnect.
type Join func(user string) (tea.Model, func())

type contextKey struct{}

// Start begins accepting players on addr and returns the address it listens
// on. The host key is created at keyPath on first use. Anyone who can reach
// the address may join: there is no authentication.
func Start(addr, keyPath string, join Join) (net.Addr, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	server, err := wish.NewServer(
		wish.WithHostKeyPath(keyPath),
		wish.WithMiddleware(
			bubbletea.Middleware(func(session ssh.Session) (tea.Model, []tea.ProgramOption) {
				model, _ := session.Context().Value(contextKey{}).(tea.Model)

				return model, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			func(next ssh.Handler) ssh.Handler {
				return func(session ssh.Session) {
					model, leave := join(session.User())
					defer leave()

					session.Context().SetValue(contextKey{}, model)
					next(session)
				}
			},
			activeterm.Middleware(),
		),
	)
	if err != nil {
		listener.Close()

		return nil, err
	}

	// Serve returns only when the listener closes, which is at process exit.
	go func() { _ = server.Serve(listener) }()

	return listener.Addr(), nil
}
