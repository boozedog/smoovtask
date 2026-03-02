package hook

import "strings"

const (
	contextEnvelopeStart = "<smoovtask>"
	contextEnvelopeEnd   = "</smoovtask>"
)

func wrapAdditionalContext(ctx string) string {
	trimmed := strings.TrimSpace(ctx)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, contextEnvelopeStart) && strings.HasSuffix(trimmed, contextEnvelopeEnd) {
		return trimmed
	}
	return contextEnvelopeStart + "\n" + trimmed + "\n" + contextEnvelopeEnd
}
