package ticket

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
)

const (
	// IDPrefix is the prefix for all ticket IDs.
	IDPrefix = "st_"

	// IDLength is the number of random base62 characters after the prefix.
	IDLength = 6

	// base62Chars is the character set for base62 encoding.
	base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// GenerateID creates a new unique ticket ID (st_xxxxxx).
// It checks for collisions against existing files in ticketsDir.
func GenerateID(ticketsDir string) (string, error) {
	existing, err := existingIDs(ticketsDir)
	if err != nil {
		return "", fmt.Errorf("scan existing IDs: %w", err)
	}

	for range 100 {
		id, err := randomID()
		if err != nil {
			return "", err
		}
		if !existing[id] {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique ID after 100 attempts")
}

// randomID generates a random st_xxxxxx ID.
func randomID() (string, error) {
	max := big.NewInt(int64(len(base62Chars)))
	var b strings.Builder
	b.WriteString(IDPrefix)
	for range IDLength {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate random: %w", err)
		}
		b.WriteByte(base62Chars[n.Int64()])
	}
	return b.String(), nil
}

// existingIDs scans the tickets directory and returns a set of existing ticket IDs.
func existingIDs(ticketsDir string) (map[string]bool, error) {
	ids := make(map[string]bool)

	entries, err := os.ReadDir(ticketsDir)
	if os.IsNotExist(err) {
		return ids, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if id := extractIDFromFilename(name); id != "" {
			ids[id] = true
		}
	}

	return ids, nil
}

// extractIDFromFilename extracts the ticket ID from a filename like
// 2026-02-25T10:00-st_a7Kx2m.md
func extractIDFromFilename(name string) string {
	// Find "st_" in the filename
	idx := strings.Index(name, IDPrefix)
	if idx < 0 {
		return ""
	}
	// Extract st_ + 6 chars
	rest := name[idx:]
	if len(rest) < len(IDPrefix)+IDLength {
		return ""
	}
	id := rest[:len(IDPrefix)+IDLength]
	// Validate base62 chars after prefix
	for _, c := range id[len(IDPrefix):] {
		if !strings.ContainsRune(base62Chars, c) {
			return ""
		}
	}
	return id
}
