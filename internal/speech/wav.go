package speech

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// windowsPlayer plays a WAV through .NET's SoundPlayer: Windows has no
// afplay, and PowerShell is always there. Killing it stops the sound.
const windowsPlayer = "powershell"

// wavPlayer is the command that plays a WAV file on this system: afplay
// on macOS, aplay on Linux, PowerShell on Windows.
func wavPlayer() string {
	if runtime.GOOS == "windows" {
		return windowsPlayer
	}

	if _, err := exec.LookPath("afplay"); err == nil {
		return "afplay"
	}

	return "aplay"
}

// queued is a sentence waiting for the voice.
type queued struct {
	text string
	mood Emotion
}

// speakWhole gives a whole line to a queue, sentence by sentence, and
// returns what stops it.
func speakWhole(add func(text string, mood Emotion), stop func(), text string, mood Emotion) func() {
	chunks, _ := Sentences(text, true)
	for _, chunk := range chunks {
		add(chunk, mood)
	}

	return stop
}

// queueWavs starts a line whose sentences come one by one: each is
// synthesized into a WAV file by fetch as soon as the one before it is,
// and played when the one before it ends, so the next sentence is being
// synthesized while the previous one is heard. It returns what adds a
// sentence and what stops it all: the fetches are abandoned and the player
// killed.
func queueWavs(player string, fetch func(ctx context.Context, line queued) (string, error)) (func(text string, mood Emotion), func()) {
	ctx, cancel := context.WithCancel(context.Background())
	lines := make(chan queued, 32)
	sounds := make(chan string, 32)

	var (
		mu     sync.Mutex
		cmd    *exec.Cmd
		closed bool
	)

	go func() {
		defer close(sounds)

		for line := range lines {
			if ctx.Err() != nil {
				return
			}

			path, err := fetch(ctx, line)
			if err != nil {
				continue
			}

			sounds <- path
		}
	}()

	go func() {
		for path := range sounds {
			mu.Lock()
			if ctx.Err() == nil {
				cmd = playWav(ctx, player, path)
				if cmd.Start() != nil {
					cmd = nil
				}
			}
			playing := cmd
			mu.Unlock()

			if playing != nil {
				_ = playing.Wait()
			}

			_ = os.Remove(path)
		}
	}()

	add := func(text string, mood Emotion) {
		mu.Lock()
		defer mu.Unlock()

		if closed {
			return
		}

		select {
		case lines <- queued{text, mood}:
		default: // a line this long is not worth hearing to the end
		}
	}

	stop := func() {
		mu.Lock()
		defer mu.Unlock()

		cancel()
		if !closed {
			closed = true
			close(lines)
		}

		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}

	return add, stop
}

// playWav is the command that plays the WAV at path.
func playWav(ctx context.Context, player, path string) *exec.Cmd {
	if player == windowsPlayer {
		script := "(New-Object Media.SoundPlayer '" + strings.ReplaceAll(path, "'", "''") + "').PlaySync()"

		return exec.CommandContext(ctx, player, "-NoProfile", "-NonInteractive", "-Command", script)
	}

	return exec.CommandContext(ctx, player, path)
}
