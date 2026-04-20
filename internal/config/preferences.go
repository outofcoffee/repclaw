package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences holds user-configurable settings persisted to disk.
type Preferences struct {
	CompletionBell bool `json:"completionBell"`
}

// DefaultPreferences returns the default preference values.
func DefaultPreferences() Preferences {
	return Preferences{
		CompletionBell: true,
	}
}

// PreferencesPath returns the path to the preferences file,
// creating the parent directory if necessary.
func PreferencesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".repclaw")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadPreferences reads preferences from disk.
// Returns defaults if the file is missing or unreadable.
func LoadPreferences() Preferences {
	path, err := PreferencesPath()
	if err != nil {
		return DefaultPreferences()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultPreferences()
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return DefaultPreferences()
	}
	return p
}

// SavePreferences writes preferences to disk.
func SavePreferences(p Preferences) error {
	path, err := PreferencesPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
