package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"go.hackfix.me/disco/web/lib"
)

type StoreGetRequest struct {
	Key       string
	Namespace string
}

// StoreGet returns the value associated to the received key.
func (h *Handler) StoreGet(w http.ResponseWriter, r *http.Request) {
	req := &StoreGetRequest{Key: chi.URLParam(r, "*"), Namespace: "default"}
	if req.Key == "" {
		_ = render.Render(w, r, lib.ErrBadRequest(errors.New("key not provided")))
		return
	}

	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	val, err := h.appCtx.Store.Get(req.Namespace, req.Key)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	// TODO: Infer Content-Type from the value
	w.Header().Del("Content-Type")

	_, err = w.Write(val)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
	}
}

type StoreSetRequest struct {
	Key       string
	Value     []byte
	Namespace string
}

type StoreSetResponse struct {
	*lib.Response
}

// StoreSet stores the provided value associated to the provided key.
func (h *Handler) StoreSet(w http.ResponseWriter, r *http.Request) {
	req := &StoreSetRequest{Key: chi.URLParam(r, "*"), Namespace: "default"}
	if req.Key == "" {
		_ = render.Render(w, r, lib.ErrBadRequest(errors.New("key not provided")))
		return
	}

	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	var err error
	req.Value, err = io.ReadAll(r.Body)

	err = h.appCtx.Store.Set(req.Namespace, req.Key, req.Value)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	_ = render.Render(w, r, &StoreSetResponse{
		Response: &lib.Response{StatusCode: http.StatusOK},
	})
}

type StoreKeysRequest struct {
	Namespace string
	Prefix    string
}

type StoreKeysResponse struct {
	*lib.Response
	Data map[string][]string `json:"keys"`
}

// StoreKeys returns the keys in the data store.
func (h *Handler) StoreKeys(w http.ResponseWriter, r *http.Request) {
	req := &StoreKeysRequest{Namespace: "default", Prefix: chi.URLParam(r, "*")}
	if ns := r.URL.Query().Get("namespace"); ns != "" {
		req.Namespace = ns
	}

	nsKeys, err := h.appCtx.Store.List(req.Namespace, req.Prefix)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	resp := &StoreKeysResponse{
		Response: &lib.Response{StatusCode: http.StatusOK},
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
