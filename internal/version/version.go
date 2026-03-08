// Package version holds the build version for st, injected via ldflags.
package version

// Version is set at build time via -ldflags "-X ...Version=v1.2.3".
// When unset (local dev builds), it defaults to "dev".
var Version = "dev"
