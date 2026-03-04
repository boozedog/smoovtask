package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/spf13/cobra"
)

var migrateDryRun bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate tickets from flat layout to project-based layout",
	Long: `Migrate tickets from <vault>/tickets/ to <vault>/projects/<project>/tickets/YYYY/MM/.

This is a one-time migration for the vault restructuring. It scans the old
flat tickets directory, parses each ticket to determine its project and
creation date, and moves the file to the new nested location.`,
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show migration plan without moving files")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	vault, err := cfg.VaultPath()
	if err != nil {
		return fmt.Errorf("get vault path: %w", err)
	}

	oldTicketsDir := filepath.Join(vault, "tickets")
	projectsDir := filepath.Join(vault, "projects")

	entries, err := os.ReadDir(oldTicketsDir)
	if os.IsNotExist(err) {
		fmt.Println("No tickets/ directory found — nothing to migrate.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read tickets dir: %w", err)
	}

	var (
		moved   int
		skipped int
		errors  int
	)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		oldPath := filepath.Join(oldTicketsDir, entry.Name())
		data, err := os.ReadFile(oldPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR reading %s: %v\n", entry.Name(), err)
			errors++
			continue
		}

		tk, err := ticket.ParseFrontmatter(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR parsing %s: %v\n", entry.Name(), err)
			errors++
			continue
		}

		if tk.Project == "" {
			fmt.Fprintf(os.Stderr, "  SKIP %s: no project set\n", entry.Name())
			skipped++
			continue
		}

		targetDir := filepath.Join(
			projectsDir,
			tk.Project,
			"tickets",
			tk.Created.UTC().Format("2006"),
			tk.Created.UTC().Format("01"),
		)
		targetPath := filepath.Join(targetDir, entry.Name())

		if migrateDryRun {
			fmt.Printf("  %s → %s\n", entry.Name(), relPath(vault, targetPath))
			moved++
			continue
		}

		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR mkdir %s: %v\n", targetDir, err)
			errors++
			continue
		}

		if err := os.Rename(oldPath, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR moving %s: %v\n", entry.Name(), err)
			errors++
			continue
		}

		moved++
	}

	// Clean up empty tickets directory (only if not dry-run)
	if !migrateDryRun {
		remaining, _ := os.ReadDir(oldTicketsDir)
		if len(remaining) == 0 {
			if err := os.Remove(oldTicketsDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remove empty tickets/ dir: %v\n", err)
			}
		}
	}

	action := "Moved"
	if migrateDryRun {
		action = "Would move"
	}
	fmt.Printf("\n%s %d tickets", action, moved)
	if skipped > 0 {
		fmt.Printf(", skipped %d", skipped)
	}
	if errors > 0 {
		fmt.Printf(", %d errors", errors)
	}
	fmt.Println()

	return nil
}

func relPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}
