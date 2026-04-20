package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/outofcoffee/repclaw/internal/config"
)

type configItem struct {
	label   string
	key     string
	checked bool
}

type configModel struct {
	items  []configItem
	cursor int
	prefs  config.Preferences
	width  int
	height int
}

func newConfigModel(prefs config.Preferences) configModel {
	return configModel{
		prefs: prefs,
		items: []configItem{
			{label: "Completion notification (terminal bell)", key: "completionBell", checked: prefs.CompletionBell},
		},
	}
}

func (m configModel) Init() tea.Cmd { return nil }

func (m configModel) Update(msg tea.Msg) (configModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "space":
			m.items[m.cursor].checked = !m.items[m.cursor].checked
			m.applyToPrefs()
			prefs := m.prefs
			return m, func() tea.Msg {
				_ = config.SavePreferences(prefs)
				return prefsUpdatedMsg{prefs: prefs}
			}
		case "esc":
			return m, func() tea.Msg { return goBackFromConfigMsg{} }
		}
	}
	return m, nil
}

func (m *configModel) applyToPrefs() {
	for _, item := range m.items {
		switch item.key {
		case "completionBell":
			m.prefs.CompletionBell = item.checked
		}
	}
}

func (m configModel) View() string {
	var b strings.Builder

	header := headerStyle.Render(" Config ")
	b.WriteString(header)
	b.WriteString("\n\n")

	for i, item := range m.items {
		check := "[ ]"
		if item.checked {
			check = "[x]"
		}

		line := fmt.Sprintf("  %s %s", check, item.label)
		if i == m.cursor {
			line = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  space: toggle · esc: back"))

	return b.String()
}

func (m *configModel) setSize(w, h int) {
	m.width = w
	m.height = h
}
