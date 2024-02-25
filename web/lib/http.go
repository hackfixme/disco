package lib

import (
	"net/http"

	"github.com/go-chi/render"
)

type RenderFunc func(w http.ResponseWriter, r *http.Request) error

type Response struct {
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func (e *Response) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.StatusCode)
	if e.Status == "" {
		e.Status = http.StatusText(e.StatusCode)
	}
	return nil
}

func ErrBadRequest(err error) render.Renderer {
	return &Response{
		StatusCode: http.StatusBadRequest,
		Error:      err.Error(),
	}
}

func ErrInternal(err error) render.Renderer {
	return &Response{
		StatusCode: http.StatusInternalServerError,
		Error:      err.Error(),
	}
}

func ErrNotFound(err error) render.Renderer {
	return &Response{
		StatusCode: http.StatusNotFound,
		Error:      err.Error(),
	}
}
