# Message rendering

## Message roles

Each chat message has a role that determines how it is displayed in the TUI:

- `user` — sent by the local user; shown right-aligned.
- `assistant` — returned by the agent; rendered as markdown via Glamour.
- `system` — local-only notices (errors, status, command output); shown in a muted style and never sent to the gateway.
- `tool` — inline status card for an in-flight or completed tool call from the agent. See [Tool call cards](#tool-call-cards).

## The System: prefix convention

Some content needs to be sent to the gateway as part of a user message (so the agent sees it) but must not be shown in the local chat history when it is loaded back. The convention is to prefix every such line with `System: `:

```
System: Available agent skills (activate with /skill-name):
System:   - review: Perform a code review
```

`prefixAllLines()` in `internal/tui/skills.go` applies this prefix to a block of text. Content that uses this convention includes:

- The skill catalog prepended to the first user message (see [skills.md](skills.md)).
- Injected skill bodies sent when the user activates a skill.

## History cleanup

When session history is fetched from the gateway, each user message may contain `System:` prefixed lines that were injected at send time. `stripSystemLines()` in `internal/tui/history.go` removes those lines before the messages are added to the display, so the user only sees what they actually typed.

`isSystemLine()` matches two forms:

- `System: ...` — the standard local prefix.
- `System (<qualifier>): ...` — a variant the gateway may substitute (e.g. `System (untrusted):`) when rewriting message content for safety reasons.

## Markdown rendering

Assistant messages are conditionally rendered with Glamour. `looksLikeMarkdown()` in `internal/tui/history.go` checks for code fences, bold markers, pipe tables, list prefixes, headings, and numbered lists. If none are found the text is shown as plain text, avoiding unwanted paragraph indentation on short replies.

The Glamour renderer is created in `setSize()` with a wrap width equal to the terminal width minus 4. `wordWrap()` in `render.go` is applied after rendering and preserves lines containing box-drawing characters (table borders) so they are not split.

## Tool call cards

When the agent invokes a tool, lucinate renders a single-line status card inline in the chat scrollback so the user can see what's running. Cards are driven by `agent` events with `stream == "tool"` from the gateway — declared via the `tool-events` capability on connect (see `internal/client/client.go`).

Each card shows a state glyph, the tool name in bold, and a one-line argument summary:

```
⠋ search (query="hello world")
✓ search (query="hello world")
✖ read (path="/missing") — file not found
```

The state glyph cycles through the same braille spinner as the streaming cursor (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) while the tool is running, then flips to `✓` on success or `✖` on error. Errors append a one-line message extracted from the tool result.

The mapping from event payload to card lives in `handleAgentEvent` (`internal/tui/events.go`):

- `phase: "start"` — freezes any currently streaming assistant row, then appends a new `chatMessage` with `role: "tool"` and `toolState: "running"`. If the streaming assistant is still the empty pre-delta placeholder, it's dropped instead of frozen so we don't render a blank assistant block above the card.
- `phase: "update"` — currently a no-op. Partial result streaming is deferred (see issue for expand/collapse output).
- `phase: "result"` — finds the matching tool row by `toolCallId` and flips `toolState` to `"success"` or `"error"`.

The `summariseArgs` helper picks a human-readable key from the args object (priority order: `command`, `path`, `file`, `filePath`, `query`, `url`, `name`, `message`, `text`) or falls back to compact JSON, truncated to 80 runes.

The full output of a tool call is not currently rendered; only the header line is. An expand/collapse affordance similar to the official OpenClaw TUI's Ctrl+O is tracked separately.

## Streaming placeholder

When the user sends a message, an empty assistant message with `streaming: true` is appended immediately so there is always something to animate (see [chat-ux.md](chat-ux.md#streaming-animation)). As delta events arrive, the message content is built up incrementally. If the final event arrives before any delta (e.g. an error), the placeholder message is removed from the display.
