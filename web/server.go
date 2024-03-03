package web

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/web/api"
)

// Server is a wrapper around http.Server with some custom behavior.
type Server struct {
	*http.Server
	appCtx *actx.Context
}

// NewServer returns a new Server instance.
func NewServer(appCtx *actx.Context, addr string) *Server {
	return &Server{
		appCtx: appCtx,
		Server: &http.Server{
			Handler:           setupRouter(appCtx),
			Addr:              addr,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      10 * time.Minute,
		},
	}
}

// ListenAndServe is a replacement of http.ListenAndServe to ensure we set the
// correct server address to be used in URLs, templates, etc.
// E.g. this is needed when starting the server with ':0'.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	s.Addr = ln.Addr().String()
	s.appCtx.Logger.Info("started web server", "address", s.Addr)

	return s.Serve(ln)
}

func setupRouter(appCtx *actx.Context) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(requestLogger(appCtx.Logger))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(middleware.Recoverer)

	r.Mount("/api", api.Router(appCtx))

	return r
}
