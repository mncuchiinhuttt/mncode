package agent

import "strings"

var memoryContextTags = []string{"<memory-context>", "<local_memories>"}
var memoryContextClosers = []string{"</memory-context>", "</local_memories>"}

// MemoryContextScrubber removes private memory prompt fences from streamed
// provider text, including tags split across token boundaries.
type MemoryContextScrubber struct {
	pending string
	inside  bool
}

// NewMemoryContextScrubber creates an empty streaming scrubber.
func NewMemoryContextScrubber() *MemoryContextScrubber { return &MemoryContextScrubber{} }

// Feed consumes one provider chunk and returns only user-visible text.
func (s *MemoryContextScrubber) Feed(chunk string) string {
	if chunk == "" {
		return ""
	}
	s.pending += chunk
	var output strings.Builder
	for {
		if s.inside {
			index, length := firstTag(s.pending, memoryContextClosers)
			if index < 0 {
				s.discardPrivatePrefix()
				return output.String()
			}
			s.pending = s.pending[index+length:]
			s.inside = false
			continue
		}

		index, length := firstTag(s.pending, memoryContextTags)
		if index < 0 {
			output.WriteString(s.emitSafePrefix())
			return output.String()
		}
		output.WriteString(s.pending[:index])
		s.pending = s.pending[index+length:]
		s.inside = true
	}
}

// Flush emits a trailing non-fenced fragment. An unterminated private fence is
// discarded rather than leaking its contents to a terminal or remote client.
func (s *MemoryContextScrubber) Flush() string {
	if s.inside {
		s.pending = ""
		return ""
	}
	output := s.pending
	s.pending = ""
	return output
}

// ScrubMemoryContext removes complete or unterminated memory fences from text.
func ScrubMemoryContext(text string) string {
	scrubber := NewMemoryContextScrubber()
	return scrubber.Feed(text) + scrubber.Flush()
}

func firstTag(value string, tags []string) (int, int) {
	index, length := -1, 0
	for _, tag := range tags {
		if found := strings.Index(value, tag); found >= 0 && (index < 0 || found < index) {
			index, length = found, len(tag)
		}
	}
	return index, length
}

func (s *MemoryContextScrubber) discardPrivatePrefix() {
	keep := longestTagPrefixSuffix(s.pending, memoryContextClosers)
	if keep == 0 {
		s.pending = ""
		return
	}
	s.pending = s.pending[len(s.pending)-keep:]
}

func (s *MemoryContextScrubber) emitSafePrefix() string {
	keep := longestTagPrefixSuffix(s.pending, memoryContextTags)
	if keep == 0 {
		output := s.pending
		s.pending = ""
		return output
	}
	cut := len(s.pending) - keep
	output := s.pending[:cut]
	s.pending = s.pending[cut:]
	return output
}

func longestTagPrefixSuffix(value string, tags []string) int {
	max := 0
	for _, tag := range tags {
		limit := len(value)
		if len(tag) < limit {
			limit = len(tag)
		}
		for size := 1; size <= limit; size++ {
			if strings.HasSuffix(value, tag[:size]) && size > max {
				max = size
			}
		}
	}
	return max
}
