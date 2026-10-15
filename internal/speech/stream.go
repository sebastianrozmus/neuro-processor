package speech

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// shortest is how many letters a piece of a line has at least before it
// goes to the voice on its own: a lone "Hm." sounds odd and costs a whole
// request, so it waits for the next sentence.
const shortest = 20

// Stream speaks a line while the model is still writing it. Every finished
// sentence goes to the voice at once, so the first words are heard while
// the rest is being written and the next sentence is synthesized while the
// previous one plays. Voices that only speak whole lines hear the line
// when it is finished. A nil stream is silent.
type Stream struct {
	speaker    *Speaker
	name, kind string
	// add queues a sentence after the ones before it; nil for voices that
	// only speak whole lines.
	add func(text string, mood Emotion)

	written string
	mood    Emotion
	// queued are the sentences already given to the voice, cleaned; used
	// is how much of the line after its mood tag they cover.
	queued []string
	used   int

	// Fit, when set, corrects the mood the model wrote before the voice
	// takes it: the game knows how the speaker feels better than the model.
	Fit func(Emotion) Emotion
}

// StreamAs starts a line spoken as it is written, as a named character of
// a kind, cutting short whatever was being said.
func (s *Speaker) StreamAs(name, kind string) *Stream {
	if s == nil {
		return nil
	}

	stream := &Stream{speaker: s, name: name, kind: kind}
	if s.queueAs == nil {
		return stream
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop != nil {
		s.stop()
	}

	stream.add, s.stop = s.queueAs(name, kind)

	return stream
}

// Write takes the next piece of the line as the model writes it. The
// pieces are raw: the game has not checked them yet.
func (st *Stream) Write(piece string) {
	if st == nil || st.add == nil {
		return
	}

	st.written += piece

	// The mood tag comes first; until it is closed, the mood is unknown.
	text := strings.TrimLeftFunc(st.written, unicode.IsSpace)
	if text == "" || strings.HasPrefix(text, "[") && !strings.Contains(text, "]") {
		return
	}

	mood, rest := SplitMood(text)
	if st.Fit != nil {
		mood = st.Fit(mood)
	}

	// The game takes a "Name:" in front of a line away; so does the voice.
	rest = strings.TrimSpace(strings.TrimPrefix(rest, st.name+":"))
	// A space after the last stop is what tells a finished sentence.
	if last, _ := utf8.DecodeLastRuneInString(st.written); unicode.IsSpace(last) {
		rest += " "
	}

	if st.queued == nil {
		st.mood = mood
	}

	chunks, used := Sentences(rest[min(st.used, len(rest)):], false)
	st.used += used

	for _, chunk := range chunks {
		st.queue(chunk)
	}
}

// Close takes the line as the game has checked it. When the sentences
// already spoken are in it, the voice goes on with the rest; when the game
// threw the streamed line away and said something else, the voice starts
// over with that.
func (st *Stream) Close(final string) {
	if st == nil {
		return
	}

	mood, line := SplitMood(final)
	if st.add == nil {
		st.speaker.SayAs(st.name, st.kind, line, mood)

		return
	}

	if st.queued == nil {
		st.mood = mood
	}

	// The game may cut a line to its last allowed sentence, so sentences
	// spoken past the end of the checked line are no reason to start over.
	chunks, _ := Sentences(Clean(line), true)
	for index, spoken := range st.queued {
		if index == len(chunks) {
			break
		}

		if key(spoken) != key(chunks[index]) {
			st.add = nil
			st.speaker.SayAs(st.name, st.kind, line, mood)

			return
		}
	}

	for _, chunk := range chunks[min(len(st.queued), len(chunks)):] {
		st.queue(chunk)
	}

	st.add = nil
}

// queue gives a sentence to the voice.
func (st *Stream) queue(chunk string) {
	chunk = Clean(chunk)
	if chunk == "" {
		return
	}

	st.queued = append(st.queued, chunk)
	st.add(chunk, st.mood)
}

// key is a sentence reduced to its letters and digits, so that a stray
// quotation mark or star the game tidied away still matches.
func key(sentence string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}

		return -1
	}, sentence)
}

// Sentences cuts text into pieces worth speaking on their own: whole
// sentences of at least a few words. Unless the text is final, the last
// sentence only counts once something follows its stop, since "7." may
// still become "7.5" and "Tak" may go on. It returns the pieces and how
// many bytes of the text they cover.
func Sentences(text string, final bool) ([]string, int) {
	var (
		chunks []string
		start  int
	)

	for at, r := range text {
		if !strings.ContainsRune(".!?…", r) {
			continue
		}

		end := at + utf8.RuneLen(r)
		next, _ := utf8.DecodeRuneInString(text[end:])
		if end < len(text) && !unicode.IsSpace(next) {
			continue
		}

		if end == len(text) && !final {
			continue
		}

		if utf8.RuneCountInString(strings.TrimSpace(text[start:end])) < shortest {
			continue
		}

		chunks = append(chunks, strings.TrimSpace(text[start:end]))
		start = end
	}

	if final && strings.TrimSpace(text[start:]) != "" {
		chunks = append(chunks, strings.TrimSpace(text[start:]))
		start = len(text)
	}

	return chunks, start
}
