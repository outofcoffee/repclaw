package hermes

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestPromptLog_AppendThenLookup is the round-trip happy path: a
// prompt logged under a response ID is recoverable by ID, and a miss
// returns ok=false rather than an error. ChatHistory relies on this
// to render the user side of the transcript when the upstream GET
// /v1/responses/{id} only returns the assistant output.
func TestPromptLog_AppendThenLookup(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, err := newPromptLog("conn")
	if err != nil {
		t.Fatalf("newPromptLog: %v", err)
	}
	if err := log.Append("resp_a", "first prompt", 1000); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := log.Append("resp_b", "second prompt", 2000); err != nil {
		t.Fatalf("Append: %v", err)
	}

	rec, ok, err := log.Lookup("resp_b")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !ok {
		t.Fatal("Lookup ok=false for known ID")
	}
	if rec.Prompt != "second prompt" {
		t.Errorf("rec.Prompt = %q, want %q", rec.Prompt, "second prompt")
	}
	if rec.Time != 2000 {
		t.Errorf("rec.Time = %d, want 2000", rec.Time)
	}

	if _, ok, err := log.Lookup("missing"); err != nil || ok {
		t.Errorf("Lookup(missing) ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

// TestPromptLog_LookupMissingFile is the first-run case: with no log
// file on disk yet, Lookup should return ok=false without surfacing
// the missing-file error to the caller.
func TestPromptLog_LookupMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, err := newPromptLog("conn")
	if err != nil {
		t.Fatalf("newPromptLog: %v", err)
	}
	rec, ok, err := log.Lookup("anything")
	if err != nil {
		t.Errorf("Lookup on missing file should not error, got %v", err)
	}
	if ok {
		t.Errorf("Lookup ok=true on missing file, rec=%+v", rec)
	}
}

// TestPromptLog_LookupReturnsLatestForDuplicateIDs guards against the
// retry-write case noted in lookupLocked: the same ID can appear
// twice if Append is retried; the latest entry wins so a recovered
// write replaces the prior one in the transcript.
func TestPromptLog_LookupReturnsLatestForDuplicateIDs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, err := newPromptLog("conn")
	if err != nil {
		t.Fatalf("newPromptLog: %v", err)
	}
	_ = log.Append("resp_dup", "first write", 1)
	_ = log.Append("resp_dup", "second write", 2)

	rec, ok, _ := log.Lookup("resp_dup")
	if !ok {
		t.Fatal("Lookup ok=false")
	}
	if rec.Prompt != "second write" {
		t.Errorf("Prompt = %q, want %q", rec.Prompt, "second write")
	}
	if rec.Time != 2 {
		t.Errorf("Time = %d, want 2", rec.Time)
	}
}

// TestPromptLog_TrimsToCap covers the soft-cap rotation: once the log
// exceeds promptLogCap the file is rewritten with only the most-recent
// promptLogCap lines so the on-disk size stays bounded as Hermes'
// own server-side LRU does. Critical because Lookup linear-scans the
// file, so an unbounded log would eventually slow first-render.
func TestPromptLog_TrimsToCap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, err := newPromptLog("conn")
	if err != nil {
		t.Fatalf("newPromptLog: %v", err)
	}

	total := promptLogCap + 5
	for i := 0; i < total; i++ {
		if err := log.Append("id"+strings.Repeat("x", 0)+itoa(i), "p", int64(i)); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	f, err := os.Open(log.path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()
	lines := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines++
	}
	if lines > promptLogCap {
		t.Errorf("trim left %d lines on disk, want <= %d", lines, promptLogCap)
	}

	// The oldest entries should have been dropped. Picking an early ID
	// that must be outside the retained window proves the rotation
	// actually took effect rather than the file just being capped on
	// a future Append.
	if _, ok, _ := log.Lookup("id0"); ok {
		t.Error("expected id0 to be trimmed away once log exceeded cap")
	}
	// The newest entry must still be present.
	if _, ok, _ := log.Lookup("id" + itoa(total-1)); !ok {
		t.Error("expected the most recent id to survive the trim")
	}
}

// TestPromptLog_ClearRemovesFile covers the SessionDelete code path:
// Clear should remove the file (so the next Append starts a fresh
// log) without erroring when there's nothing to remove.
func TestPromptLog_ClearRemovesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, err := newPromptLog("conn")
	if err != nil {
		t.Fatalf("newPromptLog: %v", err)
	}
	_ = log.Append("resp_a", "p", 1)
	if _, err := os.Stat(log.path); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
	if err := log.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(log.path); !os.IsNotExist(err) {
		t.Errorf("expected log file removed, got err=%v", err)
	}
	// Idempotent — clearing an absent file is fine.
	if err := log.Clear(); err != nil {
		t.Errorf("Clear on absent file should not error, got %v", err)
	}
}

// itoa is a small helper kept local to the test file so the trim test
// doesn't import strconv just for the loop counter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
