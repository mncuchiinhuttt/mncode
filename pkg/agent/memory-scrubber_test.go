package agent

import "testing"

func TestMemoryContextScrubberHandlesSplitTags(t *testing.T) {
	scrubber := NewMemoryContextScrubber()
	chunks := []string{"before <memory-", "context>secret", " data</memory-con", "text> after"}
	var visible string
	for _, chunk := range chunks {
		visible += scrubber.Feed(chunk)
	}
	visible += scrubber.Flush()
	if visible != "before  after" {
		t.Fatalf("visible = %q, want %q", visible, "before  after")
	}
}

func TestMemoryContextScrubberDropsUnterminatedFence(t *testing.T) {
	if got := ScrubMemoryContext("visible<local_memories>private without close"); got != "visible" {
		t.Fatalf("scrubbed = %q, want visible", got)
	}
}

func TestMemoryContextScrubberPreservesNonMemoryText(t *testing.T) {
	input := "Use <identity> tags and normal <code>content</code>."
	if got := ScrubMemoryContext(input); got != input {
		t.Fatalf("scrubbed unrelated tags: %q", got)
	}
}
