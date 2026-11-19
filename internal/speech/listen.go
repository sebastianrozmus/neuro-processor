package speech

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed listen.py
var listener string

const (
	recognizeTimeout = 2 * time.Minute
	// startTimeout is how long the recorder may take to load its libraries
	// and open the microphone.
	startTimeout = 20 * time.Second
	ready        = "recording"
)

// Listener turns the player's voice into text: it records while asked to
// and recognises the speech with a local model when told to stop. It runs
// the Python of an environment that has the model's libraries installed.
type Listener struct {
	Python string
	Dir    string

	mu  sync.Mutex
	cmd *exec.Cmd
	out string
	err *bytes.Buffer
}

// Available reports whether the listener's interpreter exists.
func (l *Listener) Available() bool {
	info, err := os.Stat(l.Python)

	return err == nil && !info.IsDir()
}

// Start begins recording. It reports an error when a recording is already
// going on or the recorder cannot be started.
func (l *Listener) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cmd != nil {
		return errors.New("nagrywanie już trwa")
	}

	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		return err
	}

	l.out = filepath.Join(l.Dir, fmt.Sprintf("heard-%d.txt", time.Now().UnixNano()))
	l.err = &bytes.Buffer{}

	cmd := exec.Command(l.Python, "-", "--out", l.out)
	cmd.Stdin = strings.NewReader(listener)
	cmd.Stderr = l.err

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Stopping before the microphone is open would only interrupt the
	// loading of the libraries, so Start waits until recording really
	// goes on.
	if err := l.await(cmd, stdout); err != nil {
		return err
	}

	l.cmd = cmd

	return nil
}

// await reads the recorder's output until it reports that it records, or
// gives up when it dies or takes too long.
func (l *Listener) await(cmd *exec.Cmd, stdout io.Reader) error {
	lines := make(chan string, 1)
	failed := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == ready {
				lines <- ready

				// The rest of the output is drained so the recorder never blocks.
				for scanner.Scan() {
				}

				return
			}
		}

		failed <- cmd.Wait()
	}()

	select {
	case <-lines:
		return nil
	case err := <-failed:
		if message := lastLine(l.err.String()); message != "" {
			return errors.New(message)
		}

		if err == nil {
			err = errors.New("nagrywanie nie ruszyło")
		}

		return err
	case <-time.After(startTimeout):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		return errors.New("mikrofon nie odpowiada; sprawdź zgodę na mikrofon dla Terminala w Ustawieniach systemowych")
	}
}

// Recording reports whether the microphone is on.
func (l *Listener) Recording() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.cmd != nil
}

// Stop ends the recording and returns what was recognised. It blocks
// while the model works, so callers run it off the interface's loop.
func (l *Listener) Stop() (string, error) {
	l.mu.Lock()
	cmd, out, stderr := l.cmd, l.out, l.err
	l.cmd = nil
	l.mu.Unlock()

	if cmd == nil {
		return "", errors.New("nic nie jest nagrywane")
	}

	defer os.Remove(out)
	_ = cmd.Process.Signal(syscall.SIGINT)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			if message := lastLine(stderr.String()); message != "" {
				return "", errors.New(message)
			}

			return "", err
		}
	case <-time.After(recognizeTimeout):
		_ = cmd.Process.Kill()

		return "", errors.New("rozpoznawanie trwało za długo")
	}

	text, err := os.ReadFile(out)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(text)), nil
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	return strings.TrimSpace(lines[len(lines)-1])
}
