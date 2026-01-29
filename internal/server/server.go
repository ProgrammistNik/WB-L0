package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"l0/internal/config"
	"l0/internal/interfaces"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	logger     *zerolog.Logger
	service    interfaces.OrderService
	config     *config.Config
}

// New creates a new HTTP server instance
func New(cfg *config.Config, service interfaces.OrderService, logger *zerolog.Logger) *Server {
	server := &Server{
		logger:  logger,
		service: service,
		config:  cfg,
	}

	server.httpServer = &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      server.setupRoutes(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop HTTP server: %w", err)
	}

	return nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /order/{order_uid}", s.handleGetOrder)
	mux.HandleFunc("GET /health", s.handleHealth)

	mux.Handle("GET /", http.FileServer(http.Dir("web/")))

	handler := s.loggingMiddleware(mux)
	handler = s.timeoutMiddleware(handler)
	handler = s.recoveryMiddleware(handler)

	return handler
}

// loggingMiddleware adds request logging
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapper, r)
		},
	)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// timeoutMiddleware adds request timeout handling
func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan struct{})

			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					http.Error(w, `{"error":"Request timeout"}`, http.StatusRequestTimeout)
				}
			}
		},
	)
}

// recoveryMiddleware handles panics and converts them to 500 errors
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					s.logger.Error().
						Interface("panic", err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("Panic recovered in HTTP handler")

					http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		},
	)
}