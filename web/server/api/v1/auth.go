package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/web/server/types"
)

// authnUser authenticates the Disco user from the received TLS client
// certificate, and loads the User record in the request context given that the
// Subject Common Name matches an existing User name. For this to be reached,
// the resource needs to have been accessed with a valid client certificate,
// which is validated in the Go runtime, before reaching Disco HTTP endpoints.
//
// If this fails, a response with status 401 Unauthorized is returned. Otherwise
// the request is allowed to continue, and authorization to access individual
// resources is done later in each handler.
func authnUser(appCtx *actx.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
				_ = render.Render(w, r, types.ErrUnauthorized("failed TLS authentication"))
				return
			}

			subjectCN := r.TLS.VerifiedChains[0][0].Subject.CommonName
			user := &models.User{Name: subjectCN}
			if err := user.Load(appCtx.DB.NewContext(), appCtx.DB); err != nil {
				appCtx.Logger.Warn(
					"failed loading user with the received TLS client certificate",
					"subjectCommonName", subjectCN, "error", err.Error())
				_ = render.Render(w, r, types.ErrUnauthorized(
					"failed loading user identified in the client TLS certificate"))
				return
			}

			ctx := context.WithValue(r.Context(), types.ConnTLSUserKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// authzUser checks whether the user is authorized to perform the given action
// on the given resource in the given namespace. An error is returned if
// authorization fails, or nil otherwise.
func authzUser(
	req *http.Request, action models.Action, resource models.Resource,
	namespace, target string,
) error {
	user, ok := req.Context().Value(types.ConnTLSUserKey).(*models.User)
	if !ok {
		return errors.New("user object not found in the request context")
	}

	target = fmt.Sprintf("%s:%s:%s", namespace, resource, target)

	if ok, err := user.Can(string(action), target); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("user '%s' is not authorized to %s %s", user.Name, action, target)
	}

	return nil
}
