package cli

import (
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
	// TODO: Handle graceful shutdown.
	// See https://dev.to/mokiat/proper-http-shutdown-in-go-3fji
	return srv.ListenAndServe()
}
