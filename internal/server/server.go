// Package server runs an HTTP server as a disposable process: it starts fast,
// stops accepting work when told to, and drains in-flight requests before exit.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func Address(port int) string {
	return fmt.Sprintf(addressFormat, port)
}

func New(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}
}

// Run takes an open listener rather than an address so callers, tests
// included, bind the port themselves and learn it before serving starts.
func Run(ctx context.Context, srv *http.Server, listener net.Listener, shutdownTimeout time.Duration, logger *slog.Logger) error {
	served := make(chan error, 1)
	go func() {
		served <- srv.Serve(listener)
	}()
	logger.LogAttrs(ctx, slog.LevelInfo, MsgListening, slog.String(LogKeyAddress, listener.Addr().String()))

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	logger.LogAttrs(ctx, slog.LevelInfo, MsgShuttingDown)
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf(shutdownErrorFormat, err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
