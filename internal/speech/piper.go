package speech

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Piper (Rhasspy, MIT) is a small neural voice that runs on the processor:
// a sentence takes a tenth of a second and no memory of the graphics card,
// where Chatterbox needs several gigabytes of it and seconds per sentence.
// It cannot act out moods, but its Polish voices (gosia, darkman,
// mc_speech) sound natural. tools/speech/setup-piper fetches the program
// and the voices into <data>/piper.
type Piper struct {
	// Exe is the piper program; Voices the folder of *.onnx voices.
	Exe    string
	Voices string
	// Voice is the voice of the characters, Female that of the women among
	// them and Narrator the game master's, all file names in Voices; an
	// empty Female or Narrator speaks with Voice.
	Voice    string
	Female   string
	Narrator string
	// Speed above 1 speaks faster.
	Speed  float64
	Player string
	Dir    string

	mu    sync.Mutex
	procs map[string]*piperProc
}

// piperProc is one voice loaded in a piper process that reads lines of
// JSON and answers each with the path of the WAV it wrote.
type piperProc struct {
	mu  sync.Mutex
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Scanner
}

// PiperExe is where setup-piper puts the program in dir.
func PiperExe(dir string) string {
	name := "piper"
	if runtime.GOOS == "windows" {
		name = "piper.exe"
	}

	return filepath.Join(dir, "piper", name)
}

// PiperVoices lists the voices in the voices folder of dir.
func PiperVoices(dir string) []string {
	paths, _ := filepath.Glob(filepath.Join(dir, "voices", "*.onnx"))
	names := make([]string, 0, len(paths))

	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}

	sort.Strings(names)

	return names
}

// NewPiper returns a speaker on piper.
func NewPiper(p *Piper) *Speaker {
	if p.Player == "" {
		p.Player = wavPlayer()
	}

	if p.Dir == "" {
		p.Dir = os.TempDir()
	}

	queue := func(name, kind string) (func(text string, mood Emotion), func()) {
		return queueWavs(p.Player, func(ctx context.Context, line queued) (string, error) {
			return p.fetch(ctx, name, kind, line)
		})
	}

	start := func(name, kind, text string, mood Emotion) func() {
		add, stop := queue(name, kind)

		return speakWhole(add, stop, text, mood)
	}

	return &Speaker{
		start:   func(text string, mood Emotion) func() { return start("", "", text, mood) },
		startAs: start,
		queueAs: queue,
	}
}

// fetch writes one sentence into a WAV file and returns its path.
func (p *Piper) fetch(ctx context.Context, name, kind string, line queued) (path string, err error) {
	voice := p.Voice

	switch {
	case kind == "narrator" && p.Narrator != "":
		voice = p.Narrator
	case kind != "narrator" && p.Female != "" && Feminine(name):
		voice = p.Female
	}

	started := time.Now()

	// Every sentence goes into the trace, the failed ones with the reason.
	defer func() {
		var wav []byte
		if err == nil {
			wav, _ = os.ReadFile(path)
		}

		traced(EnginePiper, PiperLabel(voice), name, line, started, wav, err)
	}()

	proc, err := p.proc(voice)
	if err != nil {
		return "", err
	}

	path = filepath.Join(p.Dir, fmt.Sprintf("neuro-piper-%d.wav", time.Now().UnixNano()))
	request, _ := json.Marshal(map[string]string{"text": line.text, "output_file": filepath.ToSlash(path)})

	proc.mu.Lock()
	defer proc.mu.Unlock()

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if _, err := proc.in.Write(append(request, '\n')); err != nil {
		p.drop(voice)

		return "", err
	}

	if !proc.out.Scan() {
		p.drop(voice)

		return "", errors.New("piper przestał odpowiadać")
	}

	return path, nil
}

// proc returns the process of a voice, starting it the first time.
func (p *Piper) proc(voice string) (*piperProc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proc, ok := p.procs[voice]; ok {
		return proc, nil
	}

	if voice == "" {
		return nil, errors.New("nie wybrano głosu piper")
	}

	speed := p.Speed
	if speed <= 0 {
		speed = 1
	}

	cmd := exec.Command(p.Exe, "--model", filepath.Join(p.Voices, voice), "--json-input", "--quiet",
		"--length_scale", fmt.Sprintf("%.2f", 1/speed), "--sentence_silence", "0.1")

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	proc := &piperProc{cmd: cmd, in: in, out: bufio.NewScanner(out)}
	if p.procs == nil {
		p.procs = map[string]*piperProc{}
	}

	p.procs[voice] = proc

	go func() { _ = cmd.Wait() }()

	return proc, nil
}

// drop forgets a voice whose process failed, so the next line starts it
// again.
func (p *Piper) drop(voice string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proc, ok := p.procs[voice]; ok {
		_ = proc.cmd.Process.Kill()
		delete(p.procs, voice)
	}
}

// Close ends the voices' processes.
func (p *Piper) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for voice, proc := range p.procs {
		_ = proc.in.Close()
		_ = proc.cmd.Process.Kill()
		delete(p.procs, voice)
	}
}

// PiperLabel is how a voice file is shown to the player: "gosia" for
// pl_PL-gosia-medium.onnx.
func PiperLabel(voice string) string {
	name := strings.TrimSuffix(voice, ".onnx")
	parts := strings.Split(name, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[1:len(parts)-1], "-")
	}

	return name
}
