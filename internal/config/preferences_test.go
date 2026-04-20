package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPreferences(t *testing.T) {
	p := DefaultPreferences()
	if !p.CompletionBell {
		t.Error("expected CompletionBell to default to true")
	}
}

func TestSaveAndLoadPreferences(t *testing.T) {
	// Use a temp dir to avoid touching the real config.
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	p := Preferences{CompletionBell: false}
	if err := SavePreferences(p); err != nil {
		t.Fatalf("SavePreferences: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, ".repclaw", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded := LoadPreferences()
	if loaded.CompletionBell != false {
		t.Error("expected CompletionBell to be false after save/load")
	}
}

func TestLoadPreferences_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	p := LoadPreferences()
	if !p.CompletionBell {
		t.Error("expected defaults when file is missing")
	}
}
