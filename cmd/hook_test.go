package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunHook_PIEventSessionStart(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	cwd := env.Config.Projects["testproject"].Path

	payload := fmt.Sprintf(`{"type":"session_start","session_id":"run-pi-1","cwd":%q}`, cwd)
	out, err := runHookWithInput(t, "pi-event", payload)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	if !strings.Contains(out, "smoovtask — testproject") {
		t.Fatalf("output = %q, want board summary", out)
	}
	if !strings.Contains(out, "Run: run-pi-1") {
		t.Fatalf("output = %q, want run id", out)
	}
}

func TestRunHook_PIEventToolCallWarning(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	cwd := env.Config.Projects["testproject"].Path

	payload := fmt.Sprintf(`{"type":"tool_call","session_id":"run-pi-2","cwd":%q,"tool_name":"Edit"}`, cwd)
	out, err := runHookWithInput(t, "pi-event", payload)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	if !strings.Contains(out, "WARNING: You are editing code without an active smoovtask ticket") {
		t.Fatalf("output = %q, want warning context", out)
	}
}

func runHookWithInput(t *testing.T, eventType, input string) (string, error) {
	t.Helper()

	in, err := os.CreateTemp(t.TempDir(), "hook-input-*.json")
	if err != nil {
		t.Fatalf("create temp input: %v", err)
	}
	if _, err := in.WriteString(input); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek input: %v", err)
	}

	origStdin := os.Stdin
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})
	os.Stdin = in

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	err = runHook(nil, []string{eventType})

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("close stdout writer: %v", closeErr)
	}
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}
	if closeErr := r.Close(); closeErr != nil {
		t.Fatalf("close stdout reader: %v", closeErr)
	}

	return string(outBytes), err
}
