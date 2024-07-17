package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/henvic/vio"
)

// NewServer creates a new API server.
func NewServer(address string, service *vio.Service, log *slog.Logger) *Server {
	return &Server{
		address: address,
		service: service,
		log:     log,
	}
}

// Server for the API.
type Server struct {
	address string
	service *vio.Service
	log     *slog.Logger
	http    *http.Server
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) (err error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/lookup", s.lookupHandler)

	s.http = &http.Server{
		Addr:    s.address,
		Handler: mux,

		ReadHeaderTimeout: 5 * time.Second, // mitigate risk of Slowloris Attack
	}
	s.log.Info("HTTP server listening", slog.Any("address", s.address))
	if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown HTTP server.
func (s *Server) Shutdown(ctx context.Context) {
	s.log.Info("shutting down HTTP server gracefully")
	if s.http != nil {
		if err := s.http.Shutdown(ctx); err != nil {
			s.log.Error("graceful shutdown of HTTP server failed", slog.Any("error", err))
		}
	}
}
