package speech

import (
	"encoding/base64"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"strings"
)

// NewSystem returns a speaker on the voice the operating system ships:
// `say` on macOS, the Speech API (SAPI) through PowerShell on Windows. It
// needs nothing installed and no memory of the graphics card, but sounds
// the most mechanical. Speed above 1 speaks faster. It returns nil where
// the system cannot speak.
func NewSystem(voice string, speed float64) *Speaker {
	if speed <= 0 {
		speed = 1
	}

	if runtime.GOOS == "windows" {
		return newSAPI(voice, speed)
	}

	speaker := New(voice)
	if speaker == nil || speed == 1 {
		return speaker
	}

	say, _ := exec.LookPath("say")
	if voice == "" && hasVoice(say, polishVoice) {
		voice = polishVoice
	}

	speaker.start = func(text string, mood Emotion) func() {
		args := []string{}
		if voice != "" {
			args = append(args, "-v", voice)
		}

		how, ok := deliveries[mood]
		if !ok {
			how = deliveries[Calm]
		}

		text = fmt.Sprintf("[[rate %d]] [[pbas %d]] %s", int(float64(how.rate)*speed), how.pitch, text)

		return run(exec.Command(say, append(args, text)...))
	}

	return speaker
}

// newSAPI speaks through the Windows Speech API: every line is a short
// PowerShell script, killed when the next line comes. The mood moves the
// rate a little, the way the macOS voice does.
func newSAPI(voice string, speed float64) *Speaker {
	if _, err := exec.LookPath("powershell"); err != nil {
		return nil
	}

	return &Speaker{start: func(text string, mood Emotion) func() {
		how, ok := deliveries[mood]
		if !ok {
			how = deliveries[Calm]
		}

		// SAPI's rate runs from -10 to 10; about 1 step is 10% faster.
		rate := int(math.Round((speed-1)*10)) + (how.rate-deliveries[Calm].rate)/15
		rate = max(-10, min(10, rate))

		// The text goes in as base64, so no quoting or code page can
		// spoil the Polish letters.
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		script := "Add-Type -AssemblyName System.Speech; $s = New-Object System.Speech.Synthesis.SpeechSynthesizer; "
		if voice != "" {
			script += "try { $s.SelectVoice('" + strings.ReplaceAll(voice, "'", "''") + "') } catch {}; "
		} else {
			script += "$pl = $s.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.Name -eq 'pl-PL' } | Select-Object -First 1; if ($pl) { $s.SelectVoice($pl.VoiceInfo.Name) }; "
		}

		script += fmt.Sprintf("$s.Rate = %d; $s.Speak([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s')))", rate, encoded)

		return run(exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script))
	}}
}

// SystemVoices lists the voices of the operating system, Polish first.
func SystemVoices() []string {
	var (
		out []byte
		err error
	)

	if runtime.GOOS == "windows" {
		out, err = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			"Add-Type -AssemblyName System.Speech; (New-Object System.Speech.Synthesis.SpeechSynthesizer).GetInstalledVoices() | ForEach-Object { $_.VoiceInfo.Name + '|' + $_.VoiceInfo.Culture.Name }").Output()
	} else if say, look := exec.LookPath("say"); look == nil {
		out, err = exec.Command(say, "-v", "?").Output()
	} else {
		return nil
	}

	if err != nil {
		return nil
	}

	var polish, other []string

	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n") {
		name, culture := "", ""
		if runtime.GOOS == "windows" {
			name, culture, _ = strings.Cut(line, "|")
		} else if fields := strings.Fields(line); len(fields) >= 2 {
			name, culture = fields[0], fields[1]
		}

		switch {
		case strings.TrimSpace(name) == "":
		case strings.Contains(culture, "pl"):
			polish = append(polish, strings.TrimSpace(name))
		default:
			other = append(other, strings.TrimSpace(name))
		}
	}

	return append(polish, other...)
}
