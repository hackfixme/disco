package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	actx "go.hackfix.me/disco/app/context"
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
	r.Get("/store/value/*", h.StoreGet)
	r.Post("/store/value/*", h.StoreSet)
	r.Get("/store/keys/*", h.StoreKeys)
	r.Get("/store/keys", h.StoreKeys)

	r.Post("/join", h.RemoteJoin)

	return r
}
