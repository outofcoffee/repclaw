# Agent picker

The agent picker (`selectModel` in `internal/tui/select.go`) is shown after a connection is established — either directly on startup when the connection is unambiguous, or after the user picks one from the connections picker (see [connections.md](connections.md)). It loads the list of configured agents and either presents a selection UI or auto-selects when only one agent exists. In managed mode the picker also surfaces a **Connections** action (key `c`) so the user can switch connection without first entering chat.

Agents come from the active backend (`backend.Backend.ListAgents`). For OpenClaw, that's the gateway's agent list. For OpenAI-compatible connections, agents are local — see [connections.md](connections.md#openai-agent-storage).

## Loading agents

`loadAgents()` calls `client.ListAgents()` and returns an `agentsLoadedMsg`. Each agent is an `agentItem` wrapping a `protocol.AgentSummary`. The list is displayed using a Bubble Tea list component with a custom delegate that shows the agent name (falling back to ID if no name is set).

## Auto-selection

If exactly one agent is returned, it is selected automatically without user interaction and a session is created immediately. The same auto-select fires after creating a new agent — the picker bypasses the list and proceeds straight to chat.

## Selecting an agent

Pressing Enter on a highlighted agent calls `client.CreateSession(agentID, key)`. On success, `sessionCreatedMsg` carries the new session key and the app transitions to the chat view (`newChatModel(...)`). See [sessions.md](sessions.md) for the session lifecycle from this point.

## Creating an agent

Pressing `n` in the picker switches to a two-field creation form (`subStateCreate`):

- **Name** — must start with a lowercase letter and contain only alphanumeric characters and hyphens. The input is validated on submit; an error is shown inline on failure.
- **Workspace** — a filesystem path that is auto-suggested but editable.

On submit, `client.CreateAgent(name, workspace)` is called. The gateway creates the agent and seeds an `IDENTITY.md` file in the workspace. On success the agent list is reloaded and the new agent is auto-selected (see above). On failure the form stays open and the error is shown so the user can correct and retry.
