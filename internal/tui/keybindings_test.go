package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestNewChatModel_DeleteWordBackwardBinding(t *testing.T) {
	m := newChatModel(nil, "main", "test", "")

	// ctrl+w should match DeleteWordBackward.
	ctrlW := tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl}
	if !key.Matches(ctrlW, m.textarea.KeyMap.DeleteWordBackward) {
		t.Errorf("ctrl+w should match DeleteWordBackward, got string=%q", ctrlW.String())
	}

	// alt+backspace should also match DeleteWordBackward.
	altBS := tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModAlt}
	if !key.Matches(altBS, m.textarea.KeyMap.DeleteWordBackward) {
		t.Errorf("alt+backspace should match DeleteWordBackward, got string=%q", altBS.String())
	}

	// Plain backspace should NOT match DeleteWordBackward.
	plainBS := tea.KeyPressMsg{Code: tea.KeyBackspace}
	if key.Matches(plainBS, m.textarea.KeyMap.DeleteWordBackward) {
		t.Error("plain backspace should not match DeleteWordBackward")
	}
}

func TestNewChatModel_InsertNewlineBinding(t *testing.T) {
	m := newChatModel(nil, "main", "test", "")

	// Plain enter should NOT match InsertNewline.
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	if key.Matches(enter, m.textarea.KeyMap.InsertNewline) {
		t.Error("plain enter should not match InsertNewline")
	}

	// Shift+enter SHOULD match InsertNewline.
	shiftEnter := tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	if !key.Matches(shiftEnter, m.textarea.KeyMap.InsertNewline) {
		t.Errorf("shift+enter should match InsertNewline, got string=%q", shiftEnter.String())
	}
}
