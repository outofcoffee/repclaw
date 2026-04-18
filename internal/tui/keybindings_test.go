package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewChatModel_DeleteWordBackwardBinding(t *testing.T) {
	m := newChatModel(nil, "main", "test")

	// ctrl+w should match DeleteWordBackward.
	ctrlW := tea.KeyMsg(tea.Key{Type: tea.KeyCtrlW})
	if !key.Matches(ctrlW, m.textarea.KeyMap.DeleteWordBackward) {
		t.Errorf("ctrl+w should match DeleteWordBackward, got string=%q", ctrlW.String())
	}

	// alt+backspace should also match DeleteWordBackward.
	altBS := tea.KeyMsg(tea.Key{Type: tea.KeyBackspace, Alt: true})
	if !key.Matches(altBS, m.textarea.KeyMap.DeleteWordBackward) {
		t.Errorf("alt+backspace should match DeleteWordBackward, got string=%q", altBS.String())
	}

	// Plain backspace should NOT match DeleteWordBackward.
	plainBS := tea.KeyMsg(tea.Key{Type: tea.KeyBackspace})
	if key.Matches(plainBS, m.textarea.KeyMap.DeleteWordBackward) {
		t.Error("plain backspace should not match DeleteWordBackward")
	}
}

func TestNewChatModel_InsertNewlineBinding(t *testing.T) {
	m := newChatModel(nil, "main", "test")

	// Plain enter should NOT match InsertNewline.
	enter := tea.KeyMsg(tea.Key{Type: tea.KeyEnter})
	if key.Matches(enter, m.textarea.KeyMap.InsertNewline) {
		t.Error("plain enter should not match InsertNewline")
	}
}
