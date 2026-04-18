package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/a3tai/openclaw-go/protocol"
	tea "github.com/charmbracelet/bubbletea"
)

// slashCommands is the list of available slash commands for autocomplete.
var slashCommands = []string{"/back", "/clear", "/exit", "/help", "/model", "/quit", "/stats"}

// completeSlashCommand returns the first matching slash command for the given
// prefix, or "" if no match.
func completeSlashCommand(prefix string) string {
	lower := strings.ToLower(prefix)
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, lower) {
			return cmd
		}
	}
	return ""
}

// slashCommandHint returns the completion hint to display after the current
// input, or "" if no hint applies.
func slashCommandHint(input string) string {
	if !strings.HasPrefix(input, "/") || strings.Contains(input, " ") || input == "" {
		return ""
	}
	match := completeSlashCommand(input)
	if match == "" || match == strings.ToLower(input) {
		return ""
	}
	return match[len(input):]
}

// handleSlashCommand processes local slash commands. Returns (true, cmd) if
// the input was handled as a command, or (false, nil) if it should be sent
// to the gateway.
func (m *chatModel) handleSlashCommand(text string) (handled bool, cmd tea.Cmd) {
	command := strings.ToLower(strings.TrimSpace(text))
	switch command {
	case "/quit", "/exit":
		return true, tea.Quit
	case "/back":
		return true, func() tea.Msg { return goBackMsg{} }
	case "/clear":
		m.messages = nil
		m.updateViewport()
		return true, nil
	case "/help":
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: "/quit, /exit — quit repclaw\n/back — return to agent list\n/clear — clear chat display\n/model — list available models\n/model <name> — switch model\n/stats — show session statistics\n/help — show this help\n\n!<command> — run command on gateway host",
		})
		m.updateViewport()
		return true, nil
	case "/stats":
		if m.stats == nil {
			m.messages = append(m.messages, chatMessage{role: "system", content: "Stats not yet loaded..."})
			m.updateViewport()
			return true, m.loadStats()
		}
		m.messages = append(m.messages, chatMessage{role: "system", content: m.formatStatsTable()})
		m.updateViewport()
		return true, nil
	}

	// /model with optional argument.
	if command == "/model" || strings.HasPrefix(command, "/model ") {
		return m.handleModelCommand(text)
	}

	// Unknown slash command.
	if strings.HasPrefix(text, "/") {
		m.messages = append(m.messages, chatMessage{
			role:   "system",
			errMsg: fmt.Sprintf("unknown command: %s (try /help)", command),
		})
		m.updateViewport()
		return true, nil
	}

	return false, nil
}

// handleModelCommand handles `/model` and `/model <name>`.
func (m *chatModel) handleModelCommand(text string) (bool, tea.Cmd) {
	parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
	if len(parts) == 1 {
		cl := m.client
		return true, func() tea.Msg {
			result, err := cl.ModelsList(context.Background())
			if err != nil {
				return modelListMsg{err: err}
			}
			return modelListMsg{models: result.Models}
		}
	}

	query := strings.ToLower(strings.TrimSpace(parts[1]))
	cl := m.client
	sessionKey := m.sessionKey
	return true, func() tea.Msg {
		result, err := cl.ModelsList(context.Background())
		if err != nil {
			return modelSwitchedMsg{err: err}
		}
		var match *protocol.ModelChoice
		for i, mc := range result.Models {
			lower := strings.ToLower(mc.ID)
			if lower == query || strings.ToLower(mc.Name) == query {
				match = &result.Models[i]
				break
			}
			if strings.Contains(lower, query) || strings.Contains(strings.ToLower(mc.Name), query) {
				match = &result.Models[i]
			}
		}
		if match == nil {
			return modelSwitchedMsg{err: fmt.Errorf("no model matching %q", query)}
		}
		if err := cl.SessionPatchModel(context.Background(), sessionKey, match.ID); err != nil {
			return modelSwitchedMsg{err: err}
		}
		return modelSwitchedMsg{modelID: match.ID}
	}
}

// execCommand submits a command for remote execution on the gateway host.
func (m *chatModel) execCommand(command string) tea.Cmd {
	cl := m.client
	sessionKey := m.sessionKey
	return func() tea.Msg {
		ctx := context.Background()

		result, err := cl.ExecRequest(ctx, command, sessionKey)
		if err != nil {
			return execSubmittedMsg{err: err}
		}

		decision := ""
		if result.Decision != nil {
			decision = *result.Decision
		}
		logEvent("EXEC request id=%s status=%s decision=%q", result.ID, result.Status, decision)

		switch decision {
		case "deny":
			return execSubmittedMsg{err: fmt.Errorf("command execution denied by gateway")}
		case "":
			_, err := cl.ExecResolve(ctx, result.ID, "approve")
			if err != nil {
				return execSubmittedMsg{err: fmt.Errorf("approval failed: %w", err)}
			}
			logEvent("EXEC auto-approved id=%s", result.ID)
		default:
			logEvent("EXEC decision=%q — waiting for exec.finished event", decision)
		}

		return execSubmittedMsg{}
	}
}
