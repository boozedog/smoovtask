package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/web"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the web UI",
	Long:  `Starts a local web server with a kanban board, ticket list, detail views, and live activity feed.`,
	RunE:  runWeb,
}

var webPort int

func init() {
	webCmd.Flags().IntVar(&webPort, "port", 8080, "port to listen on")
	rootCmd.AddCommand(webCmd)
}

func runWeb(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := web.NewServer(cfg, webPort)
	return srv.ListenAndServe(ctx)
}
