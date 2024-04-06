package server

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/crypto"
	apiv1 "go.hackfix.me/disco/web/server/api/v1"
)

// Server is a wrapper around http.Server with some custom behavior.
type Server struct {
	*http.Server
	appCtx    *actx.Context
	tlsConfig *tls.Config
}

// New returns a new web Server instance. It creates a self-signed certificate
// for TLS connections over which store data will be transferred.
func New(appCtx *actx.Context, addr string) (*Server, error) {
	tlsCfg := crypto.DefaultTLSConfig()
	cert, certPEM, _, err := appCtx.ServerTLSInfo()
	if err != nil {
		return nil, err
	}
	tlsCfg.Certificates = []tls.Certificate{*cert}
	tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certPEM)
	tlsCfg.ClientCAs = caCertPool

	srv := &Server{
		Server: &http.Server{
			Handler:           setupRouter(appCtx),
			Addr:              addr,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      10 * time.Minute,
		},
		appCtx:    appCtx,
		tlsConfig: tlsCfg,
	}

	return srv, nil
}

// ListenAndServe is a replacement of http.ListenAndServe to ensure we set the
// correct server address to be used in URLs, templates, etc.
// This is needed when starting the server with address ':0'.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	s.Addr = ln.Addr().String()
	s.appCtx.Logger.Info("started web server", "address", s.Addr)

	hl := &HybridListener{
		Listener:  ln,
		tlsConfig: s.tlsConfig,
		logger:    s.appCtx.Logger,
	}
	return s.Serve(hl)
}

func setupRouter(appCtx *actx.Context) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(requestLogger(appCtx.Logger))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(middleware.Recoverer)

	r.Mount("/api/v1", apiv1.Router(appCtx))

	return r
}
