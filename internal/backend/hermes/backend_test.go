package hermes

import (
	"testing"
)

func TestNew_DefaultsBaseURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	b, err := New(Options{ConnectionID: "test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()
	if b.Backend == nil {
		t.Fatal("inner openai.Backend not constructed")
	}
}

func TestNew_HonoursExplicitBaseURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	b, err := New(Options{
		ConnectionID: "test",
		BaseURL:      "http://example.com:9000/v1",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()
}

func TestNew_RequiresConnectionID(t *testing.T) {
	if _, err := New(Options{BaseURL: "http://x/v1"}); err == nil {
		t.Fatal("expected error for missing ConnectionID")
	}
}
