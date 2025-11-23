package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

func New(port string, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) && err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutting down http server")
		ctxShut, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(ctxShut); err != nil {
			return err
		}

		return nil
	case err := <-errCh:
		return err
	}
}
