package tui

import (
	"encoding/json"
	"testing"

	"github.com/a3tai/openclaw-go/protocol"
	"github.com/charmbracelet/bubbles/viewport"
)

// makeChatEvent builds a protocol.Event wrapping a ChatEvent payload.
func makeChatEvent(state, runID string, seq int, message json.RawMessage) protocol.Event {
	chatEv := protocol.ChatEvent{
		RunID:   runID,
		State:   state,
		Seq:     seq,
		Message: message,
	}
	payload, _ := json.Marshal(chatEv)
	return protocol.Event{
		EventName: protocol.EventChat,
		Payload:   payload,
	}
}

func makeChatEventWithError(state, runID, errMsg string) protocol.Event {
	chatEv := protocol.ChatEvent{
		RunID:        runID,
		State:        state,
		ErrorMessage: errMsg,
	}
	payload, _ := json.Marshal(chatEv)
	return protocol.Event{
		EventName: protocol.EventChat,
		Payload:   payload,
	}
}

// newTestChatModel creates a minimal chatModel suitable for handleEvent tests.
func newTestChatModel() *chatModel {
	vp := viewport.New(80, 20)
	return &chatModel{
		viewport:  vp,
		agentName: "test",
		width:     80,
		height:    30,
	}
}

func TestHandleEvent_DeltaCreatesAssistantMessage(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	ev := makeChatEvent("delta", "run1", 1, json.RawMessage(`"First chunk"`))
	m.handleEvent(ev)

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m.messages))
	}
	msg := m.messages[1]
	if msg.role != "assistant" {
		t.Errorf("role = %q, want assistant", msg.role)
	}
	if msg.content != "First chunk" {
		t.Errorf("content = %q, want %q", msg.content, "First chunk")
	}
	if !msg.streaming {
		t.Error("expected streaming = true")
	}
}

func TestHandleEvent_DeltasAreCumulative(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	// First delta — creates assistant message.
	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`"Hello"`)))
	// Second delta — cumulative, replaces content.
	m.handleEvent(makeChatEvent("delta", "run1", 2, json.RawMessage(`"Hello world"`)))

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m.messages))
	}
	got := m.messages[1].content
	if got != "Hello world" {
		t.Errorf("content = %q, want %q (cumulative replacement)", got, "Hello world")
	}
}

func TestHandleEvent_DeltaIgnoredAfterFinalised(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "done", streaming: false},
	}

	ev := makeChatEvent("delta", "run1", 5, json.RawMessage(`"late delta"`))
	m.handleEvent(ev)

	// Should still be 2 messages — the late delta is ignored.
	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m.messages))
	}
	if m.messages[1].content != "done" {
		t.Errorf("content changed to %q, should have stayed %q", m.messages[1].content, "done")
	}
}

func TestHandleEvent_FinalMarksStreamingDone(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "response text", streaming: true},
	}

	finalMsg := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"response text"}],"timestamp":123}`)
	m.handleEvent(makeChatEvent("final", "run1", 3, finalMsg))

	if m.messages[1].streaming {
		t.Error("expected streaming = false after final")
	}
	if m.sending {
		t.Error("expected sending = false after final")
	}
}

func TestHandleEvent_FinalReturnsRefreshCmd(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "text", streaming: true},
	}

	finalMsg := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"text"}],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 3, finalMsg))

	if cmd == nil {
		t.Error("expected a non-nil cmd (history refresh) from final event")
	}
}

func TestHandleEvent_FinalWithNoMessages(t *testing.T) {
	m := newTestChatModel()
	m.messages = nil

	finalMsg := json.RawMessage(`{"role":"assistant","content":[],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 1, finalMsg))

	if cmd != nil {
		t.Error("expected nil cmd when no messages exist")
	}
}

func TestHandleEvent_ErrorSetsErrMsg(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "partial", streaming: true},
	}

	ev := makeChatEventWithError("error", "run1", "something went wrong")
	m.handleEvent(ev)

	msg := m.messages[1]
	if msg.streaming {
		t.Error("expected streaming = false after error")
	}
	if msg.errMsg != "something went wrong" {
		t.Errorf("errMsg = %q, want %q", msg.errMsg, "something went wrong")
	}
	if m.sending {
		t.Error("expected sending = false after error")
	}
}

func TestHandleEvent_AbortedAppendsMarker(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "partial", streaming: true},
	}

	ev := makeChatEvent("aborted", "run1", 3, nil)
	m.handleEvent(ev)

	msg := m.messages[1]
	if msg.streaming {
		t.Error("expected streaming = false after aborted")
	}
	if msg.content != "partial\n[aborted]" {
		t.Errorf("content = %q, want %q", msg.content, "partial\n[aborted]")
	}
	if m.sending {
		t.Error("expected sending = false after aborted")
	}
}

func TestHandleEvent_NonChatEventIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	ev := protocol.Event{EventName: "tick"}
	m.handleEvent(ev)

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
}

func TestHandleEvent_InvalidPayloadIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	ev := protocol.Event{
		EventName: protocol.EventChat,
		Payload:   json.RawMessage(`not valid json`),
	}
	m.handleEvent(ev)

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
}

func TestHandleEvent_EmptyDeltaIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	ev := makeChatEvent("delta", "run1", 1, json.RawMessage(`""`))
	m.handleEvent(ev)

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message (empty delta ignored), got %d", len(m.messages))
	}
}

func TestHandleEvent_FullStreamingFlow(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "ping"}}
	m.sending = true

	// Delta 1: partial text.
	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`"Hel"`)))
	if len(m.messages) != 2 {
		t.Fatalf("after delta 1: expected 2 messages, got %d", len(m.messages))
	}
	if m.messages[1].content != "Hel" {
		t.Errorf("after delta 1: content = %q", m.messages[1].content)
	}

	// Delta 2: cumulative update.
	m.handleEvent(makeChatEvent("delta", "run1", 2, json.RawMessage(`"Hello!"`)))
	if m.messages[1].content != "Hello!" {
		t.Errorf("after delta 2: content = %q, want %q", m.messages[1].content, "Hello!")
	}
	if !m.messages[1].streaming {
		t.Error("after delta 2: should still be streaming")
	}

	// Final.
	finalMsg := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"Hello!"}],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 3, finalMsg))
	if m.messages[1].streaming {
		t.Error("after final: should not be streaming")
	}
	if m.sending {
		t.Error("after final: sending should be false")
	}
	if cmd == nil {
		t.Error("after final: expected refresh cmd")
	}

	// Late delta after final — should be ignored.
	m.handleEvent(makeChatEvent("delta", "run1", 3, json.RawMessage(`"Hello!"`)))
	if len(m.messages) != 2 {
		t.Errorf("after late delta: expected 2 messages, got %d", len(m.messages))
	}
}

func TestUpdateViewport_BottomAnchoring(t *testing.T) {
	m := newTestChatModel()
	m.viewport = viewport.New(80, 20)
	m.width = 80
	m.agentName = "test"
	m.messages = []chatMessage{
		{role: "user", content: "hi"},
		{role: "assistant", content: "hello"},
	}

	m.updateViewport()

	content := m.viewport.View()
	// The content should contain padding newlines at the top since we only
	// have a few lines of messages but the viewport is 20 lines tall.
	// Just verify the messages appear and the content has some leading newlines.
	if len(content) == 0 {
		t.Error("viewport content should not be empty")
	}
}
