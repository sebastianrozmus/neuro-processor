package shareserver

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// Report is the outcome of Verify.
type Report struct {
	Files, Records int
	Problems       []string
	// Heads are the count and the hash of each file's last line: kept
	// somewhere else (with a backup, say), they show a later cut at the end
	// of a file or a file gone.
	Heads []string
}

// Verify checks every stored line: the hash chain of each file, the hash
// of each payload, the key, the signature, the proof of work, the payload
// schema, and that no record of a withdrawn key is left. The challenge's
// MAC cannot be checked: its key lived only in the process that took it.
// Run it on a stopped server or a copy (docker compose exec works too, it
// only reads).
func Verify(dir string) (Report, error) {
	var report Report

	if info, err := os.Stat(dir); err != nil {
		return report, err
	} else if !info.IsDir() {
		return report, fmt.Errorf("%s nie jest katalogiem", dir)
	}

	store := &Store{dir: dir, forgotten: map[[16]byte]bool{}}
	if err := store.readForgotten(false); err != nil {
		report.Problems = append(report.Problems, err.Error())
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return report, err
	}

	sort.Strings(files)

	for _, file := range files {
		report.Files++

		prev, number := "", 0
		name := filepath.Base(file)

		err := eachLine(file, func(line []byte, record Record) error {
			number++
			report.Records++

			problem := func(text string) {
				report.Problems = append(report.Problems, fmt.Sprintf("%s:%d: %s", name, number, text))
			}

			if record.Prev != prev {
				problem("przerwany łańcuch (brakuje lub zmieniono poprzednią linię)")
			}

			prev = lineHash(line)
			sum := sha256.Sum256(record.Payload)

			if hex.EncodeToString(sum[:]) != record.SHA256 {
				problem("skrót danych się nie zgadza")
			}

			key, keyOK := decode(record.Key)
			sig, sigOK := decode(record.Sig)

			if !keyOK || !sigOK || len(key) != ed25519.PublicKeySize || share.WeakKey(key) ||
				!ed25519.Verify(key, share.Signed(share.ActionSubmit, record.Key, record.Challenge, record.Nonce, record.Payload), sig) {
				problem("podpis lub klucz się nie zgadza")
			}

			challenge, err := base64.RawURLEncoding.DecodeString(record.Challenge)
			if _, canonical := decode(record.Challenge); err != nil || !canonical || len(challenge) != challengeLen ||
				challenge[24] < 1 || challenge[24] > 32 || !share.ValidNonce(record.Nonce) ||
				share.Work(record.Challenge, record.Key, record.Nonce) < int(challenge[24]) {
				problem("brak dowodu pracy")
			}

			if _, err := share.ParseCanonical(record.Payload); err != nil {
				problem("dane: " + err.Error())
			}

			if store.forgotten[keyHash(record.Key)] {
				problem("wpis klucza, który wycofał zgodę (usunie go działający serwer)")
			}

			return nil
		})
		if err != nil {
			report.Problems = append(report.Problems, fmt.Sprintf("%s: %v", name, err))
		}

		report.Heads = append(report.Heads, fmt.Sprintf("%s %d %s", name, number, prev))
	}

	return report, nil
}
