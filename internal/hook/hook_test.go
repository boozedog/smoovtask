package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReadInput(t *testing.T) {
	raw := `{"session_id":"sess-123","cwd":"/tmp/proj","hook_event_name":"SessionStart","source":"startup"}`
	input, err := ReadInputFrom(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadInputFrom: %v", err)
	}

	if input.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", input.SessionID, "sess-123")
	}
	if input.CWD != "/tmp/proj" {
		t.Errorf("CWD = %q, want %q", input.CWD, "/tmp/proj")
	}
	if input.HookEventName != "SessionStart" {
		t.Errorf("HookEventName = %q, want %q", input.HookEventName, "SessionStart")
	}
	if input.Source != "startup" {
		t.Errorf("Source = %q, want %q", input.Source, "startup")
	}
}

func TestReadInputEmpty(t *testing.T) {
	input, err := ReadInputFrom(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ReadInputFrom: %v", err)
	}
	if input.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", input.SessionID)
	}
}

func TestWriteOutput(t *testing.T) {
	var buf bytes.Buffer
	out := Output{
		AdditionalContext: "test context",
	}

	if err := WriteOutputTo(&buf, out); err != nil {
		t.Fatalf("WriteOutputTo: %v", err)
	}

	var got Output
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if got.AdditionalContext != "test context" {
		t.Errorf("AdditionalContext = %q, want %q", got.AdditionalContext, "test context")
	}
}
