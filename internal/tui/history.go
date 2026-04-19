package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/outofcoffee/repclaw/internal/client"
)

const historyLimit = 20

// historyResponse is the structure of the chat.history RPC response.
type historyResponse struct {
	Messages []historyMessage `json:"messages"`
}

type historyMessage struct {
	Role    string             `json:"role"`
	Content []chatContentBlock `json:"content"`
}

func (m chatModel) loadHistory() tea.Cmd {
	sessionKey := m.sessionKey
	cl := m.client
	renderer := m.renderer
	return func() tea.Msg {
		msgs, err := fetchHistory(cl, sessionKey, renderer)
		return historyLoadedMsg{messages: msgs, err: err}
	}
}

func (m chatModel) refreshHistory() tea.Cmd {
	sessionKey := m.sessionKey
	cl := m.client
	renderer := m.renderer
	return func() tea.Msg {
		msgs, err := fetchHistory(cl, sessionKey, renderer)
		return historyRefreshMsg{messages: msgs, err: err}
	}
}

func fetchHistory(cl *client.Client, sessionKey string, renderer *glamour.TermRenderer) ([]chatMessage, error) {
	raw, err := cl.ChatHistory(context.Background(), sessionKey, historyLimit)
	if err != nil {
		return nil, err
	}
	var resp historyResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var msgs []chatMessage
	for _, hm := range resp.Messages {
		role := hm.Role
		if role != "user" && role != "assistant" {
			continue
		}
		var parts []string
		for _, block := range hm.Content {
			if block.Type == "text" && block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		text := strings.Join(parts, "\n")
		if text == "" {
			continue
		}
		if role == "user" {
			text = stripSystemLines(text)
			if text == "" {
				continue
			}
		}
		rendered := false
		if role == "assistant" && renderer != nil {
			if out, err := renderer.Render(text); err == nil {
				text = strings.TrimSpace(out)
				rendered = true
			}
		}
		msgs = append(msgs, chatMessage{role: role, content: text, rendered: rendered})
	}
	return msgs, nil
}

// stripSystemLines removes "System:" prefixed lines and leading whitespace
// from user messages, returning only the human-authored portion.
func stripSystemLines(s string) string {
	lines := strings.Split(s, "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "System:") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
