package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/outofcoffee/repclaw/internal/client"
)

type viewState int

const (
	viewSelect viewState = iota
	viewChat
)

// escTimeout is the window in which a key after ESC is treated as an alt+key
// sequence rather than a standalone ESC. 50ms matches typical terminal behaviour.
const escTimeout = 50 * time.Millisecond

// escExpiredMsg is sent when the ESC debounce timer fires without a follow-up key.
type escExpiredMsg struct{}

// AppModel is the root bubbletea model.
type AppModel struct {
	state       viewState
	selectModel selectModel
	chatModel   chatModel
	client      *client.Client
	width       int
	height      int
	escPending  bool // true while waiting to see if ESC is standalone or part of alt+key
}

// NewApp creates the root application model.
func NewApp(c *client.Client) AppModel {
	return AppModel{
		state:       viewSelect,
		selectModel: newSelectModel(c),
		client:      c,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.selectModel.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selectModel.setSize(msg.Width, msg.Height)
		if m.state == viewChat {
			m.chatModel.setSize(msg.Width, msg.Height)
		}
		return m, nil

	case escExpiredMsg:
		if m.escPending {
			m.escPending = false
			if m.state == viewChat {
				m.state = viewSelect
				return m, nil
			}
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state == viewChat {
				// Don't act immediately — wait to see if this is the start
				// of an alt+key sequence (e.g. alt+backspace = ESC + 0x7f).
				m.escPending = true
				return m, tea.Tick(escTimeout, func(time.Time) tea.Msg {
					return escExpiredMsg{}
				})
			}
			return m, tea.Quit
		default:
			if m.escPending {
				// A key arrived after ESC within the timeout — this is an
				// alt+key sequence. Cancel the pending ESC and forward the
				// key with Alt set to the chat model.
				m.escPending = false
				altMsg := tea.KeyMsg{
					Type:  msg.Type,
					Runes: msg.Runes,
					Alt:   true,
				}
				var cmd tea.Cmd
				m.chatModel, cmd = m.chatModel.Update(altMsg)
				return m, cmd
			}
		}
	}

	switch m.state {
	case viewSelect:
		var cmd tea.Cmd
		m.selectModel, cmd = m.selectModel.Update(msg)
		// Check if an agent was selected.
		if m.selectModel.selected {
			m.selectModel.selected = false
			if item, ok := m.selectModel.selectedAgent(); ok {
				name := item.agent.Name
				if name == "" {
					name = item.agent.ID
				}
				m.chatModel = newChatModel(m.client, item.sessionKey, name)
				m.chatModel.setSize(m.width, m.height)
				m.state = viewChat
				return m, m.chatModel.Init()
			}
		}
		return m, cmd

	case viewChat:
		var cmd tea.Cmd
		m.chatModel, cmd = m.chatModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.state {
	case viewSelect:
		return m.selectModel.View()
	case viewChat:
		return m.chatModel.View()
	}
	return ""
}
