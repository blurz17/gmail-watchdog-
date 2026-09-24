package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mostaql-notification/internal/application"
	"github.com/mostaql-notification/internal/infrastructure/postgres"
)

// Server provides the HTTP API for health checks and administration.
type Server struct {
	mux      *http.ServeMux
	server   *http.Server
	db       *postgres.DB
	statusUC *application.GetStatusUseCase
	apiKey   string
	logger   *slog.Logger
}

// NewServer creates a new HTTP server.
func NewServer(
	port string,
	db *postgres.DB,
	statusUC *application.GetStatusUseCase,
	apiKey string,
	logger *slog.Logger,
) *Server {
	mux := http.NewServeMux()

	s := &Server{
		mux:      mux,
		db:       db,
		statusUC: statusUC,
		apiKey:   apiKey,
		logger:   logger,
		server: &http.Server{
			Addr:         ":" + port,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /{$}", s.handleRoot)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ready", s.handleReady)
	s.mux.HandleFunc("GET /status", s.handleStatus)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"service": "Gmail Watchdog",
		"health":  "/health",
		"ready":   "/ready",
		"status":  "/status (requires auth)",
	})
}

// Start starts the HTTP server in the background.
func (s *Server) Start() {
	go func() {
		s.logger.Info("http server starting", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("http server error", "error", err)
		}
	}()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("http server stopping")
	return s.server.Shutdown(ctx)
}

// handleHealth returns 200 if the service is running.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleReady returns 200 if the database is reachable.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.db.HealthCheck(r.Context()); err != nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  "database unreachable",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// handleStatus returns the full system status (requires API key).
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Require API key for status endpoint.
	if s.apiKey != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+s.apiKey {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			return
		}
	}

	status, err := s.statusUC.Execute(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("fetching status: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, status)
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to write json response", "error", err)
	}
}
