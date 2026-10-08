package world

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed presets/*.json
var presetFiles embed.FS

var transliteration = strings.NewReplacer(
	"ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n", "ó", "o", "ś", "s", "ź", "z", "ż", "z",
)

// Presets returns the worlds shipped with the game.
func Presets() ([]Spec, error) {
	entries, err := presetFiles.ReadDir("presets")
	if err != nil {
		return nil, err
	}

	specs := make([]Spec, 0, len(entries))
	for _, entry := range entries {
		data, err := presetFiles.ReadFile("presets/" + entry.Name())
		if err != nil {
			return nil, err
		}

		spec, err := parse(data)
		if err != nil {
			return nil, err
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// SavedSpecs returns the worlds stored in dir, skipping files that are not valid.
func SavedSpecs(dir string) []Spec {
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(paths)

	specs := make([]Spec, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if spec, err := parse(data); err == nil {
			specs = append(specs, spec)
		}
	}

	return specs
}

// SaveSpec stores a world in dir under a name derived from the world's name.
func SaveSpec(dir string, spec Spec) (string, error) {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, Slug(spec.Name)+".json")

	return path, os.WriteFile(path, data, 0o644)
}

func parse(data []byte) (Spec, error) {
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return Spec{}, err
	}

	return spec, spec.Normalize()
}

// Slug turns a name into a file name made of ASCII letters, digits and dashes.
func Slug(name string) string {
	var out strings.Builder

	for _, char := range transliteration.Replace(strings.ToLower(name)) {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			out.WriteRune(char)
		case out.Len() > 0 && !strings.HasSuffix(out.String(), "-"):
			out.WriteByte('-')
		}
	}

	if result := strings.Trim(out.String(), "-"); result != "" {
		return result
	}

	return "world"
}
