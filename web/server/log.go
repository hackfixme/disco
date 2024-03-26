package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
)

// requestLogger returns a logger HTTP handler using the given logger.
func requestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(next, w, r)
			logger.Info(
				fmt.Sprintf("%s %s", r.Method, r.URL),
				"response_code", m.Code,
				"duration", m.Duration,
				"bytes_sent", m.Written,
				"remote_addr", r.RemoteAddr,
			)
		}
		return http.HandlerFunc(fn)
	}
}
