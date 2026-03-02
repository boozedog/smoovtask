package spawn

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	runIDLength = 8
	base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// generateRunID creates a random run ID for a spawned worker.
func generateRunID() string {
	max := big.NewInt(int64(len(base62Chars)))
	var b strings.Builder
	b.WriteString("spawn-")
	for range runIDLength {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Fallback to a simpler format if crypto/rand fails
			return fmt.Sprintf("spawn-%d", big.NewInt(0).Int64())
		}
		b.WriteByte(base62Chars[n.Int64()])
	}
	return b.String()
}
