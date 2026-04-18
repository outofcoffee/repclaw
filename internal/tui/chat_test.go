package tui

import (
	"encoding/json"
	"testing"
)

func TestExtractTextFromMessage_DeltaString(t *testing.T) {
	raw := json.RawMessage(`"Hello, world!"`)
	got := extractTextFromMessage(raw)
	if got != "Hello, world!" {
		t.Errorf("got %q, want %q", got, "Hello, world!")
	}
}

func TestExtractTextFromMessage_FinalStructured(t *testing.T) {
	raw := json.RawMessage(`{
		"role": "assistant",
		"content": [
			{"type": "text", "text": "First paragraph."},
			{"type": "text", "text": "Second paragraph."}
		],
		"timestamp": 1776540452625
	}`)
	got := extractTextFromMessage(raw)
	want := "First paragraph.\nSecond paragraph."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractTextFromMessage_FinalWithNonTextBlocks(t *testing.T) {
	raw := json.RawMessage(`{
		"role": "assistant",
		"content": [
			{"type": "tool_use", "text": ""},
			{"type": "text", "text": "Visible text."}
		]
	}`)
	got := extractTextFromMessage(raw)
	if got != "Visible text." {
		t.Errorf("got %q, want %q", got, "Visible text.")
	}
}

func TestExtractTextFromMessage_EmptyContent(t *testing.T) {
	raw := json.RawMessage(`{"role": "assistant", "content": []}`)
	got := extractTextFromMessage(raw)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestExtractTextFromMessage_EmptyInput(t *testing.T) {
	got := extractTextFromMessage(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}

	got = extractTextFromMessage(json.RawMessage{})
	if got != "" {
		t.Errorf("got %q for empty slice, want empty string", got)
	}
}

func TestExtractTextFromMessage_Fallback(t *testing.T) {
	// Neither a JSON string nor a structured message — should return raw bytes.
	raw := json.RawMessage(`12345`)
	got := extractTextFromMessage(raw)
	if got != "12345" {
		t.Errorf("got %q, want %q", got, "12345")
	}
}

func TestStripSystemLines_OnlySystemLines(t *testing.T) {
	input := "System: [2026-04-18] Node connected\nSystem: [2026-04-18] reason launch"
	got := stripSystemLines(input)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestStripSystemLines_MixedContent(t *testing.T) {
	input := "System: [2026-04-18] Node connected\n\n[Sat 2026-04-18 20:27] hello there"
	got := stripSystemLines(input)
	want := "[Sat 2026-04-18 20:27] hello there"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripSystemLines_NoSystemLines(t *testing.T) {
	input := "just a normal message"
	got := stripSystemLines(input)
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestStripSystemLines_EmptyInput(t *testing.T) {
	got := stripSystemLines("")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestStripSystemLines_IndentedSystemLine(t *testing.T) {
	input := "  System: indented system line\nuser text"
	got := stripSystemLines(input)
	if got != "user text" {
		t.Errorf("got %q, want %q", got, "user text")
	}
}

func TestWordWrap_ShortText(t *testing.T) {
	got := wordWrap("hello", 80)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestWordWrap_ZeroWidth(t *testing.T) {
	got := wordWrap("hello world", 0)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestWordWrap_NegativeWidth(t *testing.T) {
	got := wordWrap("hello world", -5)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestWordWrap_WrapsLongLine(t *testing.T) {
	got := wordWrap("the quick brown fox jumps", 15)
	// "the quick brown" = 15 chars, then "fox jumps" wraps
	if got == "the quick brown fox jumps" {
		t.Error("expected text to be wrapped, but got original string")
	}
	// Verify it contains a newline.
	lines := 0
	for _, c := range got {
		if c == '\n' {
			lines++
		}
	}
	if lines == 0 {
		t.Error("expected at least one newline in wrapped output")
	}
}

func TestWordWrap_PreservesExistingNewlines(t *testing.T) {
	got := wordWrap("line one\nline two", 80)
	if got != "line one\nline two" {
		t.Errorf("got %q", got)
	}
}
