package web

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/boozedog/smoovtask/internal/config"
	"github.com/boozedog/smoovtask/internal/project"
	"github.com/boozedog/smoovtask/internal/web/handler"
	"github.com/boozedog/smoovtask/internal/web/middleware"
	"github.com/boozedog/smoovtask/internal/web/sse"
	"github.com/boozedog/smoovtask/internal/web/static"
)

// Server is the web UI server for smoovtask.
type Server struct {
	cfg    *config.Config
	port   int
	broker *sse.Broker
	srv    *http.Server
}

// NewServer creates a new web server.
func NewServer(cfg *config.Config, port int) *Server {
	return &Server{
		cfg:  cfg,
		port: port,
	}
}

// ListenAndServe starts the server and blocks until the context is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	eventsDir, err := s.cfg.EventsDir()
	if err != nil {
		return fmt.Errorf("get events dir: %w", err)
	}

	ticketsDir, err := s.cfg.TicketsDir()
	if err != nil {
		return fmt.Errorf("get tickets dir: %w", err)
	}

	// Start SSE broker and file watcher.
	s.broker = sse.NewBroker()
	watcher, err := sse.NewWatcher(eventsDir, s.broker)
	if err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	// Set up handlers.
	cwd, _ := os.Getwd()
	proj := project.Detect(s.cfg, cwd)
	h := handler.New(ticketsDir, eventsDir, s.broker, proj)

	mux := http.NewServeMux()

	// Static assets.
	staticFS, err := fs.Sub(static.Assets, "dist")
	if err != nil {
		return fmt.Errorf("static fs: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Pages.
	mux.HandleFunc("GET /{$}", h.Board)
	mux.HandleFunc("GET /new", h.NewTicket)
	mux.HandleFunc("POST /new", h.CreateTicket)
	mux.HandleFunc("GET /list", h.List)
	mux.HandleFunc("GET /ticket/{id}", h.Ticket)
	mux.HandleFunc("GET /ticket/{id}/edit", h.EditTicket)
	mux.HandleFunc("POST /ticket/{id}/edit", h.UpdateTicket)
	mux.HandleFunc("GET /activity", h.Activity)
	mux.HandleFunc("GET /critical-path", h.CriticalPath)

	// SSE endpoint.
	mux.HandleFunc("GET /events", h.Events)

	// Partials for htmx.
	mux.HandleFunc("GET /partials/board", h.PartialBoard)
	mux.HandleFunc("GET /partials/list", h.PartialList)
	mux.HandleFunc("GET /partials/list-content", h.PartialListContent)
	mux.HandleFunc("GET /partials/ticket/{id}", h.PartialTicket)
	mux.HandleFunc("GET /partials/activity", h.PartialActivity)
	mux.HandleFunc("GET /partials/activity-content", h.PartialActivityContent)
	mux.HandleFunc("GET /partials/critical-path", h.PartialCriticalPath)

	s.srv = &http.Server{
		Addr: fmt.Sprintf(":%d", s.port),
		Handler: middleware.Chain(mux,
			middleware.CORS(),
			middleware.RateLimit(ctx, middleware.DefaultRateLimitConfig()),
		),
		ReadTimeout: 5 * time.Second,
		// WriteTimeout is deliberately unset (0 = no timeout) because SSE
		// connections are long-lived. A per-handler write timeout would
		// kill the /events stream and trigger aggressive browser reconnects
		// that exhaust the HTTP/1.1 connection pool.
		IdleTimeout: 120 * time.Second,
	}

	// Graceful shutdown on context cancellation.
	go func() {
		<-ctx.Done()
		slog.Info("shutting down web server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutdownCtx)
	}()

	slog.Info("listening", "addr", fmt.Sprintf("http://localhost:%d", s.port))
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
