package shareserver

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/share"
)

// Store keeps accepted submissions as JSON lines, one file a day. Each
// line carries the envelope it came in (so its signature can be checked
// again at any time) and the hash of the line before it in the same file
// (so a line removed or changed inside a file shows). Verify checks both.
//
// A withdrawal of consent is written to forget.log first, durably, and
// refuses the key at once; a janitor then removes the key's records file
// by file, holding the lock for one file at a time. At every start the
// janitor makes a full pass, so a removal cut short by a crash is finished.
//
// Memory stays bounded: repeats are looked for among the last two days
// only (a retry after a lost answer comes within minutes), counters are
// kept for today only, and no index of the whole archive is held.
type Store struct {
	dir    string
	now    func() time.Time
	limits Limits

	mu        sync.Mutex
	seen      map[[16]byte]struct{} // key + payload hash of today
	seenPrev  map[[16]byte]struct{} // and of yesterday
	forgotten map[[16]byte]bool     // keys that withdrew consent
	pending   map[string]bool       // keys whose records the janitor still removes
	last      map[string]string     // file -> hash of its last line
	day       string
	daily     map[string]int // key -> accepted today
	dayCount  int
	dayBytes  int64
	forgets   int // withdrawals today
	total     int64
	broken    error // set when the files may no longer match the memory

	wake chan struct{}
	lock *os.File
}

// Limits bound what the store takes.
type Limits struct {
	PerKeyDay     int   // records of one key a day
	PerDay        int   // records of everyone a day
	PerDayBytes   int64 // bytes of everyone a day
	Total         int64 // bytes on disk in all
	ForgetsPerDay int   // withdrawals of everyone a day
}

// Record is one stored line.
type Record struct {
	Received  string          `json:"received"`
	Key       string          `json:"key"`
	Challenge string          `json:"challenge"`
	Nonce     string          `json:"nonce"`
	Sig       string          `json:"sig"`
	SHA256    string          `json:"sha256"`
	Prev      string          `json:"prev"`
	Flags     []string        `json:"flags,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// Errors of Add and Forget.
var (
	ErrDuplicate = errors.New("duplicate")
	ErrQuota     = errors.New("quota")
	ErrFull      = errors.New("full")
	ErrForgotten = errors.New("forgotten")
)

const (
	dayLayout  = "2006-01-02"
	forgetFile = "forget.log"
	lockFile   = ".lock"
	maxLine    = 4 * share.MaxBody
)

// OpenStore takes the directory for this process alone, repairs a line cut
// short by a crash at the end of any file, checks the chains and reads the
// counters. Call Janitor to start removing what was withdrawn.
func OpenStore(dir string, limits Limits, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	lock, err := lockDir(filepath.Join(dir, lockFile))
	if err != nil {
		return nil, fmt.Errorf("katalog %s jest zajęty przez inny proces: %w", dir, err)
	}

	s := &Store{
		dir: dir, now: now, limits: limits, seen: map[[16]byte]struct{}{}, seenPrev: map[[16]byte]struct{}{},
		forgotten: map[[16]byte]bool{}, pending: map[string]bool{}, last: map[string]string{}, daily: map[string]int{},
		day: now().UTC().Format(dayLayout), wake: make(chan struct{}, 1), lock: lock,
	}

	leftovers, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	for _, leftover := range leftovers {
		_ = os.Remove(leftover)
	}

	if err := s.readForgotten(true); err != nil {
		return nil, err
	}

	files, err := s.list()
	if err != nil {
		return nil, err
	}

	yesterday := now().UTC().AddDate(0, 0, -1).Format(dayLayout)

	for _, file := range files {
		if err := repairTail(file); err != nil {
			return nil, err
		}

		day := strings.TrimSuffix(filepath.Base(file), ".jsonl")
		prev := ""

		err := eachLine(file, func(line []byte, record Record) error {
			if record.Prev != prev && s.broken == nil {
				s.broken = fmt.Errorf("przerwany łańcuch w %s: uruchom verify", filepath.Base(file))
				log.Print(s.broken)
			}

			prev = lineHash(line)
			size := int64(len(line) + 1)
			s.total += size

			switch day {
			case s.day:
				s.seen[seenKey(record.Key, record.SHA256)] = struct{}{}
				s.daily[record.Key]++
				s.dayCount++
				s.dayBytes += size
			case yesterday:
				s.seenPrev[seenKey(record.Key, record.SHA256)] = struct{}{}
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}

		s.last[file] = prev
	}

	return s, nil
}

// Close lets another process take the directory.
func (s *Store) Close() error {
	return s.lock.Close()
}

func seenKey(key, hash string) [16]byte {
	sum := sha256.Sum256([]byte(key + " " + hash))

	return [16]byte(sum[:16])
}

func keyHash(key string) [16]byte {
	sum := sha256.Sum256([]byte(key))

	return [16]byte(sum[:16])
}

// readForgotten reads forget.log. Its torn last line is cut (that
// withdrawal was never confirmed); any other bad line stops the start, as
// it may be a confirmed withdrawal.
func (s *Store) readForgotten(repair bool) error {
	path := filepath.Join(s.dir, forgetFile)

	if !repair {
		// A torn last line is no withdrawal; verify reads without writing.
	} else if err := repairTail(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}
	defer f.Close()

	today := s.day
	scanner := bufio.NewScanner(f)

	for number := 1; scanner.Scan(); number++ {
		var note struct {
			Time string `json:"time"`
			Key  string `json:"key_sha256"`
		}

		err := json.Unmarshal(scanner.Bytes(), &note)

		var raw []byte
		if err == nil {
			raw, err = hex.DecodeString(note.Key)
		}

		if err != nil || len(raw) < 16 {
			return fmt.Errorf("%s:%d: uszkodzony wpis wycofania zgody, popraw go ręcznie", forgetFile, number)
		}

		s.forgotten[[16]byte(raw[:16])] = true
		s.total += int64(len(scanner.Bytes()) + 1)

		if strings.HasPrefix(note.Time, today) {
			s.forgets++
		}
	}

	return scanner.Err()
}

func (s *Store) list() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(s.dir, "*.jsonl"))
	sort.Strings(files)

	return files, err
}

// repairTail cuts a last line that has no newline: a write the process did
// not finish. Every complete line ends with one.
func repairTail(file string) error {
	f, err := os.OpenFile(file, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return err
	}

	tail := min(info.Size(), int64(maxLine)+1)
	buffer := make([]byte, tail)

	if _, err := f.ReadAt(buffer, info.Size()-tail); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	if buffer[len(buffer)-1] == '\n' {
		return nil
	}

	cut := info.Size() - tail + int64(bytes.LastIndexByte(buffer, '\n')+1)
	log.Printf("%s: obcinam niedokończoną linię (%d bajtów)", filepath.Base(file), info.Size()-cut)

	if err := f.Truncate(cut); err != nil {
		return err
	}

	return f.Sync()
}

func lineHash(line []byte) string {
	sum := sha256.Sum256(line)

	return hex.EncodeToString(sum[:])
}

func eachLine(file string, each func(line []byte, record Record) error) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), maxLine)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var record Record
		if err := json.Unmarshal(line, &record); err != nil {
			return err
		}

		if err := each(line, record); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// encode writes a record as one line, compact and without HTML escaping,
// so the payload bytes stay exactly those that were signed.
func encode(record Record) ([]byte, error) {
	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(record); err != nil {
		return nil, err
	}

	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

// rollDay starts today's counters when the day changed.
func (s *Store) rollDay(day string) {
	if day == s.day {
		return
	}

	s.seenPrev, s.seen = s.seen, map[[16]byte]struct{}{}
	s.day, s.daily, s.dayCount, s.dayBytes, s.forgets = day, map[string]int{}, 0, 0, 0
}

// Add stores a verified envelope, refusing a payload the key sent lately,
// a key that withdrew consent, and anything over the limits.
func (s *Store) Add(envelope share.Envelope, flags []string) error {
	// Only compact payloads are taken, so storing them cannot change a byte.
	var compact bytes.Buffer
	if err := json.Compact(&compact, envelope.Payload); err != nil || !bytes.Equal(compact.Bytes(), envelope.Payload) {
		return errBad{"dane nie są zwartym JSON-em"}
	}

	sum := sha256.Sum256(envelope.Payload)
	hash := hex.EncodeToString(sum[:])
	seen := seenKey(envelope.Key, hash)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broken != nil {
		return s.broken
	}

	now := s.now().UTC()
	day := now.Format(dayLayout)
	file := filepath.Join(s.dir, day+".jsonl")
	s.rollDay(day)

	_, today := s.seen[seen]
	_, yesterday := s.seenPrev[seen]

	switch {
	case s.forgotten[keyHash(envelope.Key)]:
		return ErrForgotten
	case today || yesterday:
		return ErrDuplicate
	case s.daily[envelope.Key] >= s.limits.PerKeyDay:
		return ErrQuota
	}

	line, err := encode(Record{
		Received: now.Format(time.RFC3339), Key: envelope.Key, Challenge: envelope.Challenge, Nonce: envelope.Nonce,
		Sig: envelope.Sig, SHA256: hash, Prev: s.last[file], Flags: flags, Payload: envelope.Payload,
	})
	if err != nil {
		return err
	}

	size := int64(len(line) + 1)

	if s.dayCount >= s.limits.PerDay || s.dayBytes+size > s.limits.PerDayBytes || s.total+size > s.limits.Total {
		return ErrFull
	}

	if err := s.append(file, append(line, '\n')); err != nil {
		return err
	}

	// Counters count what was taken today: a later removal frees disk, not
	// today's budget.
	s.seen[seen] = struct{}{}
	s.daily[envelope.Key]++
	s.dayCount++
	s.dayBytes += size
	s.total += size
	s.last[file] = lineHash(line)

	return nil
}

// append writes a line durably. When the write fails part way, the file is
// cut back to where it was, so the next line does not glue onto a broken
// one; if even that fails, the store stops taking writes.
func (s *Store) append(file string, line []byte) error {
	_, statErr := os.Stat(file)
	created := errors.Is(statErr, os.ErrNotExist)

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = f.Write(line)
	if err == nil {
		err = f.Sync()
	}

	if err != nil {
		cutErr := f.Truncate(size)
		if cutErr == nil {
			cutErr = f.Sync()
		}

		if cutErr != nil {
			s.broken = fmt.Errorf("%s nie zgadza się z pamięcią: %w", filepath.Base(file), cutErr)
		}

		return err
	}

	// The line itself is on disk; a directory that cannot be synced only
	// risks the new file's name in a crash, which the next start sees.
	if created {
		if err := syncDir(s.dir); err != nil {
			log.Printf("sync katalogu: %v", err)
		}
	}

	return nil
}

// syncDir makes a new or renamed file in dir survive a crash. Windows
// cannot sync a directory and does not need to.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Sync()
}

// Forget refuses everything from key from now on and queues its records
// for removal. It returns once the withdrawal is on disk; the janitor does
// the rest. forget.log keeps only the hash of the key.
func (s *Store) Forget(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// A key already withdrawn is in the janitor's hands (and in every full
	// pass): asking again costs nothing and starts no scan.
	hash := keyHash(key)
	if s.forgotten[hash] {
		return nil
	}

	s.rollDay(s.now().UTC().Format(dayLayout))

	if s.forgets >= s.limits.ForgetsPerDay {
		return ErrFull
	}

	note, _ := json.Marshal(map[string]string{
		"time": s.now().UTC().Format(time.RFC3339), "key_sha256": hex.EncodeToString(hash[:]),
	})

	if err := s.append(filepath.Join(s.dir, forgetFile), append(note, '\n')); err != nil {
		return err
	}

	s.forgotten[hash] = true
	s.forgets++
	s.total += int64(len(note) + 1)
	s.queue(key)

	return nil
}

func (s *Store) queue(key string) {
	s.pending[key] = true

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Pacing of the janitor: withdrawals arriving close together are removed
// in one pass, and a failed pass is tried again after a pause.
const (
	janitorGap   = 5 * time.Second
	janitorRetry = 30 * time.Second
)

// Janitor removes the records of withdrawn keys until stop is closed: one
// full pass at once (finishing what a crash cut short), then a pass for
// the keys queued since. A pass that fails is tried again, full again if
// it was full, until it succeeds. done, if not nil, is told after each.
func (s *Store) Janitor(stop <-chan struct{}, done func(error)) {
	full := true

	for {
		err := s.sweep(full)
		if err != nil {
			log.Printf("usuwanie: %v (ponowię za %v)", err, janitorRetry)
		}

		if done != nil {
			done(err)
		}

		full = full && err != nil

		wait := janitorGap
		if err != nil {
			wait = janitorRetry
		}

		select {
		case <-stop:
			return
		case <-time.After(wait):
		}

		if err == nil {
			select {
			case <-stop:
				return
			case <-s.wake:
			}
		}
	}
}

// sweep makes one pass over the files, removing the records of the keys
// queued (or of every withdrawn key, when full). A file is looked through
// without the lock when nothing but the janitor changes it (any day but
// today), and rewritten under the lock only when it holds such a record.
// A file that fails does not stop the others; its keys stay queued.
func (s *Store) sweep(full bool) error {
	s.mu.Lock()

	keys := s.pending
	s.pending = map[string]bool{}

	withdrawn := map[[16]byte]bool{}
	if full {
		for hash := range s.forgotten {
			withdrawn[hash] = true
		}
	} else {
		for key := range keys {
			withdrawn[keyHash(key)] = true
		}
	}

	today := filepath.Join(s.dir, s.day+".jsonl")
	s.mu.Unlock()

	if !full && len(keys) == 0 {
		return nil
	}

	gone := func(record Record) bool { return withdrawn[keyHash(record.Key)] }

	files, err := s.list()

	var failed []error

	if err != nil {
		failed = append(failed, err)
	}

	for _, file := range files {
		if file != today && !holds(file, gone) {
			continue
		}

		if err := s.dropFrom(file, gone); err != nil {
			failed = append(failed, err)
		}
	}

	if len(failed) > 0 {
		s.mu.Lock()
		for key := range keys {
			s.pending[key] = true
		}
		s.mu.Unlock()

		return errors.Join(failed...)
	}

	return nil
}

// holds reports whether file has a record gone picks; an unreadable file
// counts as holding one, so that dropFrom reports why.
func holds(file string, gone func(Record) bool) bool {
	found := false

	err := eachLine(file, func(_ []byte, record Record) error {
		found = found || gone(record)

		return nil
	})

	return found || err != nil && !errors.Is(err, os.ErrNotExist)
}

// dropFrom rewrites file without the records gone picks, with its chain
// made again. A file with none of them is left untouched; a broken chain
// stops the rewrite, so that it cannot hide what broke it.
func (s *Store) dropFrom(file string, gone func(Record) bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found, prev := false, ""

	err := eachLine(file, func(line []byte, record Record) error {
		if record.Prev != prev {
			return fmt.Errorf("przerwany łańcuch w %s: uruchom verify", filepath.Base(file))
		}

		prev = lineHash(line)
		found = found || gone(record)

		return nil
	})
	if errors.Is(err, os.ErrNotExist) || err == nil && !found {
		return nil
	}

	if err != nil {
		return err
	}

	tmp := file + ".tmp"

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(out)
	prev, kept := "", 0

	// Lines kept can only shrink (the first one loses its prev), so total
	// is lowered by the dropped lines alone and stays on the safe side.
	var dropped int64

	err = eachLine(file, func(line []byte, record Record) error {
		if s.forgotten[keyHash(record.Key)] && gone(record) {
			dropped += int64(len(line) + 1)

			return nil
		}

		record.Prev = prev

		encoded, err := encode(record)
		if err != nil {
			return err
		}

		if _, err := writer.Write(append(encoded, '\n')); err != nil {
			return err
		}

		prev = lineHash(encoded)
		kept++

		return nil
	})
	if err == nil {
		err = writer.Flush()
	}

	if err == nil {
		err = out.Sync()
	}

	if closeErr := out.Close(); err == nil {
		err = closeErr
	}

	if err == nil {
		if kept == 0 {
			err = os.Remove(file)
		} else {
			err = os.Rename(tmp, file)
		}
	}

	_ = os.Remove(tmp)

	if err != nil {
		return err
	}

	if kept == 0 {
		delete(s.last, file)
	} else {
		s.last[file] = prev
	}

	s.total -= dropped

	if err := syncDir(s.dir); err != nil {
		log.Printf("sync katalogu: %v", err)
	}

	return nil
}
