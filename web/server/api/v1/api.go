package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/web/server/types"
)

// Handler is the API endpoint handler.
type Handler struct {
	appCtx *actx.Context
}

// Router returns the API router.
func Router(appCtx *actx.Context) chi.Router {
	r := chi.NewRouter()

	r.Use(render.SetContentType(render.ContentTypeJSON))
	// Limit request sizes to 100MB
	r.Use(middleware.RequestSize(100 << (10 * 2)))

	h := Handler{appCtx}
	r.Route("/store", func(r chi.Router) {
		r.Use(tlsOnly)
		r.Get("/value/*", h.StoreGet)
		r.Post("/value/*", h.StoreSet)
		r.Get("/keys/*", h.StoreKeys)
		r.Get("/keys", h.StoreKeys)
	})

	r.Post("/join", h.RemoteJoin)

	return r
}

// tlsOnly ensures that the resource is served exclusively over TLS. It returns
// 401 Unauthorized otherwise.
func tlsOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connType := r.Context().Value(types.ConnTypeKey)
		ct, ok := connType.(types.ConnType)
		if !ok || ct != types.ConnTypeTLS {
			_ = render.Render(w, r, types.ErrUnauthorized("resource must be accessed over TLS"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
