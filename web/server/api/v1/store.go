package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"go.hackfix.me/disco/web/server/types"
)

// StoreGet returns the value associated to the received key.
func (h *Handler) StoreGet(w http.ResponseWriter, r *http.Request) {
	req := &types.StoreGetRequest{Key: chi.URLParam(r, "*"), Namespace: "default"}
	if req.Key == "" {
		_ = render.Render(w, r, types.ErrBadRequest(errors.New("key not provided")))
		return
	}

	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	ok, val, err := h.appCtx.Store.Get(req.Namespace, req.Key)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	// TODO: Infer Content-Type from the value
	w.Header().Del("Content-Type")

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = io.Copy(w, val)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
	}
}

// StoreSet stores the provided value associated to the provided key.
func (h *Handler) StoreSet(w http.ResponseWriter, r *http.Request) {
	req := &types.StoreSetRequest{Key: chi.URLParam(r, "*"), Namespace: "default"}
	if req.Key == "" {
		_ = render.Render(w, r, types.ErrBadRequest(errors.New("key not provided")))
		return
	}

	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	err := h.appCtx.Store.Set(req.Namespace, req.Key, r.Body)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	_ = render.Render(w, r, &types.StoreSetResponse{
		Response: &types.Response{StatusCode: http.StatusOK},
	})
}

// StoreKeys returns the keys in the data store.
func (h *Handler) StoreKeys(w http.ResponseWriter, r *http.Request) {
	req := &types.StoreKeysRequest{Namespace: "default", Prefix: chi.URLParam(r, "*")}
	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	nsKeys, err := h.appCtx.Store.List(req.Namespace, req.Prefix)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	resp := &types.StoreKeysResponse{
		Response: &types.Response{StatusCode: http.StatusOK},
		Data:     make(map[string][]string),
	}

	for ns, keys := range nsKeys {
		var strKeys []string
		for _, v := range keys {
			strKeys = append(strKeys, string(v))
		}
		resp.Data[ns] = strKeys
	}

	_ = render.Render(w, r, resp)
}
