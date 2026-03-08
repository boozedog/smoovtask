package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
)

func testProjectPath(t *testing.T, cfg *config.Config) string {
	t.Helper()
	vaultPath, err := cfg.VaultPath()
	if err != nil {
		t.Fatalf("get vault path: %v", err)
	}
	meta, err := project.LoadMeta(vaultPath, "testproject")
	if err != nil || meta == nil {
		t.Fatalf("load project meta: %v", err)
	}
	return meta.Path
}

func TestRunHook_PIEventSessionStart(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	cwd := testProjectPath(t, env.Config)

	payload := fmt.Sprintf(`{"type":"session_start","session_id":"run-pi-1","cwd":%q}`, cwd)
	out, err := runHookWithInput(t, "pi-event", payload)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	if !strings.Contains(out, "project called testproject") {
		t.Fatalf("output = %q, want project intro", out)
	}
	if !strings.Contains(out, "Your run ID is `run-pi-1`") {
		t.Fatalf("output = %q, want run id", out)
	}
}

func TestRunHook_PIEventToolCallBlocked(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	cwd := testProjectPath(t, env.Config)

	payload := fmt.Sprintf(`{"type":"tool_call","session_id":"run-pi-2","cwd":%q,"tool_name":"Edit"}`, cwd)
	out, err := runHookWithInput(t, "pi-event", payload)
	if err != nil {
		t.Fatalf("run hook: %v", err)
	}

	if !strings.Contains(out, "\"permissionDecision\":\"deny\"") {
		t.Fatalf("output = %q, want deny decision", out)
	}
	if !strings.Contains(out, "st pick") || !strings.Contains(out, "--run-id run-pi-2") {
		t.Fatalf("output = %q, want remediation guidance", out)
	}
}

func TestRunHook_PreToolBlocksOnDeniedDecision(t *testing.T) {
	env := newTestEnv(t)
	cwd := testProjectPath(t, env.Config)

	_, err := runHookWithInput(t, "pre-tool", fmt.Sprintf(`{"session_id":"run-claude-1","cwd":%q,"tool_name":"Edit"}`, cwd))
	if err == nil {
		t.Fatal("expected pre-tool hook to hard-block Edit without active ticket")
	}
	if !strings.Contains(err.Error(), "st pick <ticket-id> --run-id run-claude-1") {
		t.Fatalf("error = %q, want remediation guidance", err.Error())
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
