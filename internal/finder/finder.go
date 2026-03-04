package finder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UseFd controls whether fd is used for file discovery.
// Set to false in tests or environments where fd is unavailable.
var UseFd = detectFd()

func detectFd() bool {
	_, err := exec.LookPath("fd")
	return err == nil
}

// FindFiles returns all ticket .md file paths under dir.
// It uses fd if available, falling back to filepath.WalkDir.
func FindFiles(dir string) ([]string, error) {
	if UseFd {
		return findWithFd(dir)
	}
	return findWithWalk(dir)
}

func findWithFd(dir string) ([]string, error) {
	cmd := exec.Command("fd", "--type", "f", "--extension", "md", "--glob", "*-st_*.md", dir)
	out, err := cmd.Output()
	if err != nil {
		// fd may fail if dir doesn't exist; fall back to walk
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			return nil, nil
		}
		return findWithWalk(dir)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var paths []string
	for _, line := range lines {
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

func findWithWalk(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".md") && strings.Contains(name, "-st_") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
