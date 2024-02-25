package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"go.hackfix.me/disco/app/ctx"
)

// Handler is the API endpoint handler.
type Handler struct {
	appCtx *ctx.Context
}

// Router returns the API router.
func Router(appCtx *ctx.Context) chi.Router {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))

	h := Handler{appCtx}
	r.Get("/store/value/*", h.StoreGet)
	r.Post("/store/value/*", h.StoreSet)
	r.Get("/store/keys/*", h.StoreKeys)
	r.Get("/store/keys", h.StoreKeys)

	return r
}
