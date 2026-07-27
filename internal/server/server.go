package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gnrfan/quickproxy/internal/config"
	"github.com/gnrfan/quickproxy/internal/logging"
	"github.com/gnrfan/quickproxy/internal/proxy"
)

type Server struct {
	config  *config.Config
	logger  *logging.Logger
	log     *slog.Logger
	http    *http.Server
}

func New(cfg *config.Config, logger *logging.Logger, log *slog.Logger) *Server {
	p := proxy.New(cfg, logger, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.ServeHTTP)

	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: cfg.GetReadHeaderTimeout(),
		IdleTimeout:       cfg.GetIdleTimeout(),
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	return &Server{
		config: cfg,
		logger: logger,
		log:    log,
		http:   srv,
	}
}

func (s *Server) Start() error {
	s.log.Info("starting server",
		"listen", s.config.ListenAddress,
		"domains", s.config.Domains,
		"target_host", s.config.TargetHost,
	)

	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	s.log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), s.config.GetShutdownTimeout())
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	if err := s.logger.Close(); err != nil {
		return fmt.Errorf("logger close error: %w", err)
	}

	s.log.Info("server stopped")
	return nil
}

func (s *Server) ListenAddress() string {
	return s.config.ListenAddress
}
