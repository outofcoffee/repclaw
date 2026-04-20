package tui

import (
	"encoding/json"
	"testing"

	"github.com/a3tai/openclaw-go/protocol"
	"charm.land/bubbles/v2/viewport"
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

// newTestChatModel creates a minimal chatModel suitable for unit tests.
func newTestChatModel() *chatModel {
	vp := viewport.New()
	return &chatModel{
		viewport:  vp,
		agentName: "test",
		width:     80,
		height:    30,
	}
}

func TestExtractTextFromMessage_DeltaString(t *testing.T) {
	raw := json.RawMessage(`"Hello, world!"`)
	got := extractTextFromMessage(raw)
	if got != "Hello, world!" {
		t.Errorf("got %q, want %q", got, "Hello, world!")
	}
}

func TestExtractTextFromMessage_FinalStructured(t *testing.T) {
	raw := json.RawMessage(`{
		"role": "assistant",
		"content": [
			{"type": "text", "text": "First paragraph."},
			{"type": "text", "text": "Second paragraph."}
		],
		"timestamp": 1776540452625
	}`)
	got := extractTextFromMessage(raw)
	want := "First paragraph.\nSecond paragraph."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractTextFromMessage_FinalWithNonTextBlocks(t *testing.T) {
	raw := json.RawMessage(`{
		"role": "assistant",
		"content": [
			{"type": "tool_use", "text": ""},
			{"type": "text", "text": "Visible text."}
		]
	}`)
	got := extractTextFromMessage(raw)
	if got != "Visible text." {
		t.Errorf("got %q, want %q", got, "Visible text.")
	}
}

func TestExtractTextFromMessage_EmptyContent(t *testing.T) {
	raw := json.RawMessage(`{"role": "assistant", "content": []}`)
	got := extractTextFromMessage(raw)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestExtractTextFromMessage_EmptyInput(t *testing.T) {
	got := extractTextFromMessage(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}

	got = extractTextFromMessage(json.RawMessage{})
	if got != "" {
		t.Errorf("got %q for empty slice, want empty string", got)
	}
}

func TestExtractTextFromMessage_Fallback(t *testing.T) {
	raw := json.RawMessage(`12345`)
	got := extractTextFromMessage(raw)
	if got != "12345" {
		t.Errorf("got %q, want %q", got, "12345")
	}
}

func TestHandleEvent_DeltaCreatesAssistantMessage(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`"First chunk"`)))

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m.messages))
	}
	if m.messages[1].role != "assistant" {
		t.Errorf("role = %q, want assistant", m.messages[1].role)
	}
	if m.messages[1].content != "First chunk" {
		t.Errorf("content = %q, want %q", m.messages[1].content, "First chunk")
	}
	if !m.messages[1].streaming {
		t.Error("expected streaming = true")
	}
}

func TestHandleEvent_DeltasAreCumulative(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`"Hello"`)))
	m.handleEvent(makeChatEvent("delta", "run1", 2, json.RawMessage(`"Hello world"`)))

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m.messages))
	}
	if m.messages[1].content != "Hello world" {
		t.Errorf("content = %q, want %q", m.messages[1].content, "Hello world")
	}
}

func TestHandleEvent_DeltaIgnoredAfterFinalised(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "done", streaming: false},
	}

	m.handleEvent(makeChatEvent("delta", "run1", 5, json.RawMessage(`"late delta"`)))

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
		t.Error("expected a non-nil cmd from final event")
	}
}

func TestHandleEvent_FinalWithNoMessages(t *testing.T) {
	m := newTestChatModel()
	m.messages = nil

	finalMsg := json.RawMessage(`{"role":"assistant","content":[],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 1, finalMsg))

	// No streaming assistant message to finalise — treated as a gateway ack.
	if cmd != nil {
		t.Error("expected nil cmd for final with no streaming assistant message")
	}
}

func TestHandleEvent_ErrorSetsErrMsg(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "partial", streaming: true},
	}

	m.handleEvent(makeChatEventWithError("error", "run1", "something went wrong"))

	if m.messages[1].streaming {
		t.Error("expected streaming = false after error")
	}
	if m.messages[1].errMsg != "something went wrong" {
		t.Errorf("errMsg = %q", m.messages[1].errMsg)
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

	m.handleEvent(makeChatEvent("aborted", "run1", 3, nil))

	if m.messages[1].streaming {
		t.Error("expected streaming = false after aborted")
	}
	if m.messages[1].content != "partial\n[aborted]" {
		t.Errorf("content = %q", m.messages[1].content)
	}
}

func TestDrainQueue_SendsFirstPendingMessage(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.pendingMessages = []string{"msg1", "msg2", "msg3"}
	m.messages = []chatMessage{
		{role: "user", content: "original"},
		{role: "assistant", content: "response", streaming: false},
	}

	cmd := m.drainQueue()

	if cmd == nil {
		t.Fatal("expected non-nil cmd from drainQueue")
	}
	if len(m.pendingMessages) != 2 {
		t.Errorf("expected 2 remaining pending messages, got %d", len(m.pendingMessages))
	}
	if m.pendingMessages[0] != "msg2" {
		t.Errorf("expected next pending to be %q, got %q", "msg2", m.pendingMessages[0])
	}
	if !m.sending {
		t.Error("expected sending to remain true while queue is non-empty")
	}
	// User message should be appended to m.messages when dequeued.
	last := m.messages[len(m.messages)-1]
	if last.role != "user" || last.content != "msg1" {
		t.Errorf("expected last message to be user 'msg1', got %s %q", last.role, last.content)
	}
}

func TestDrainQueue_EmptyQueueSetsSendingFalse(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.pendingMessages = nil

	cmd := m.drainQueue()

	if cmd == nil {
		t.Fatal("expected non-nil cmd (refresh) from drainQueue on empty queue")
	}
	if m.sending {
		t.Error("expected sending = false when queue is empty")
	}
}

func TestQueuedMessages_NotInMessagesUntilDrained(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "original"},
	}
	m.pendingMessages = []string{"queued1", "queued2"}

	// Pending messages should not be in m.messages.
	for _, msg := range m.messages {
		if msg.content == "queued1" || msg.content == "queued2" {
			t.Error("queued messages should not be in m.messages before draining")
		}
	}

	// Drain first message.
	m.drainQueue()
	found := false
	for _, msg := range m.messages {
		if msg.role == "user" && msg.content == "queued1" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'queued1' in m.messages after draining")
	}

	// Second queued message should still not be in m.messages.
	for _, msg := range m.messages {
		if msg.content == "queued2" {
			t.Error("'queued2' should not be in m.messages yet")
		}
	}
}

func TestFinalEvent_EmptyAckDoesNotDrain(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.pendingMessages = []string{"queued"}
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
	}

	// An empty final with no streaming assistant message (gateway ack).
	finalMsg := json.RawMessage(`{"role":"assistant","content":[],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 1, finalMsg))

	if cmd != nil {
		t.Error("expected nil cmd for empty ack final")
	}
	if len(m.pendingMessages) != 1 {
		t.Errorf("queue should be unchanged, got %d pending", len(m.pendingMessages))
	}
	if !m.sending {
		t.Error("sending should remain true — ack should not reset it")
	}
}

func TestFinalEvent_DrainAfterStreamingResponse(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.pendingMessages = []string{"next msg"}
	m.messages = []chatMessage{
		{role: "user", content: "hello"},
		{role: "assistant", content: "response", streaming: true},
	}

	finalMsg := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"response"}],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 3, finalMsg))

	if cmd == nil {
		t.Fatal("expected non-nil cmd to drain the queue")
	}
	if m.messages[1].streaming {
		t.Error("assistant message should be finalised")
	}
	if len(m.pendingMessages) != 0 {
		t.Errorf("expected queue to be drained, got %d pending", len(m.pendingMessages))
	}
	// The dequeued message should now be in m.messages.
	last := m.messages[len(m.messages)-1]
	if last.role != "user" || last.content != "next msg" {
		t.Errorf("expected dequeued user message, got %s %q", last.role, last.content)
	}
}

func TestDeltaAfterDrain_CreatesNewAssistant(t *testing.T) {
	m := newTestChatModel()
	m.sending = true
	m.messages = []chatMessage{
		{role: "user", content: "first"},
		{role: "assistant", content: "reply", streaming: false},
		{role: "user", content: "second"}, // appended by drainQueue
	}

	// A delta for the response to "second" should create a new assistant message,
	// not be ignored because of the earlier finalised assistant.
	m.handleEvent(makeChatEvent("delta", "run2", 1, json.RawMessage(`"new reply"`)))

	if len(m.messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(m.messages))
	}
	last := m.messages[3]
	if last.role != "assistant" || last.content != "new reply" || !last.streaming {
		t.Errorf("expected new streaming assistant, got role=%s content=%q streaming=%v",
			last.role, last.content, last.streaming)
	}
}

func TestHandleEvent_NonChatEventIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	m.handleEvent(protocol.Event{EventName: "tick"})

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
}

func TestHandleEvent_InvalidPayloadIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	m.handleEvent(protocol.Event{
		EventName: protocol.EventChat,
		Payload:   json.RawMessage(`not valid json`),
	})

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
}

func TestHandleEvent_EmptyDeltaIgnored(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "hello"}}

	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`""`)))

	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
}

func TestHandleEvent_FullStreamingFlow(t *testing.T) {
	m := newTestChatModel()
	m.messages = []chatMessage{{role: "user", content: "ping"}}
	m.sending = true

	m.handleEvent(makeChatEvent("delta", "run1", 1, json.RawMessage(`"Hel"`)))
	if m.messages[1].content != "Hel" {
		t.Errorf("after delta 1: content = %q", m.messages[1].content)
	}

	m.handleEvent(makeChatEvent("delta", "run1", 2, json.RawMessage(`"Hello!"`)))
	if m.messages[1].content != "Hello!" {
		t.Errorf("after delta 2: content = %q", m.messages[1].content)
	}

	finalMsg := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"Hello!"}],"timestamp":123}`)
	cmd := m.handleEvent(makeChatEvent("final", "run1", 3, finalMsg))
	if m.messages[1].streaming {
		t.Error("after final: should not be streaming")
	}
	if cmd == nil {
		t.Error("after final: expected refresh cmd")
	}

	m.handleEvent(makeChatEvent("delta", "run1", 3, json.RawMessage(`"Hello!"`)))
	if len(m.messages) != 2 {
		t.Errorf("after late delta: expected 2 messages, got %d", len(m.messages))
	}
}
