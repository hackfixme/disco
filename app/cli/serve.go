package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/web/server"
)

// Serve starts the web server.
type Serve struct {
	Address string `help:"[host]:port to listen on" default:":2020"`
}

// Run the serve command.
func (s *Serve) Run(appCtx *actx.Context) error {
	srv, err := server.New(appCtx, s.Address)
	if err != nil {
		return err
	}

	// Gracefully shutdown the server if a process signal is received, or the
	// main context is done.
	// See https://dev.to/mokiat/proper-http-shutdown-in-go-3fji
	srvDone := make(chan error)
	go func() {
		srvErr := srv.ListenAndServe()
		slog.Debug("web server shutdown")
		srvDone <- srvErr
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sigCh:
		slog.Debug("process received signal", "signal", s)
	case <-appCtx.Ctx.Done():
		slog.Debug("app context is done")
	case srvErr := <-srvDone:
		if srvErr != nil && !errors.Is(srvErr, http.ErrServerClosed) {
			return fmt.Errorf("web server error: %w", srvErr)
		}
		return nil
	}

	if err := srv.Shutdown(appCtx.Ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("failed shutting down web server: %w", err)
	}

	return nil
}
