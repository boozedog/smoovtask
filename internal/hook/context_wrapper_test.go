package hook

import "testing"

func TestWrapAdditionalContextWrapsPlainText(t *testing.T) {
	got := wrapAdditionalContext("hello")
	want := "<smoovtask>\nhello\n</smoovtask>"
	if got != want {
		t.Fatalf("wrapAdditionalContext() = %q, want %q", got, want)
	}
}

func TestWrapAdditionalContextIsIdempotent(t *testing.T) {
	in := "<smoovtask>\nhello\n</smoovtask>"
	got := wrapAdditionalContext(in)
	if got != in {
		t.Fatalf("wrapAdditionalContext() = %q, want %q", got, in)
	}
}

func TestWrapAdditionalContextEmpty(t *testing.T) {
	got := wrapAdditionalContext("   \n\t")
	if got != "" {
		t.Fatalf("wrapAdditionalContext() = %q, want empty", got)
	}
}
