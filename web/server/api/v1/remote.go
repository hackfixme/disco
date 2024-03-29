package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/render"
	"github.com/mr-tron/base58"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
	"go.hackfix.me/disco/web/server/lib"
)

type RemoteJoinResponse struct {
	*lib.Response
	TLSClientCertEnc string `json:"tls_client_cert_enc"`
}

// RemoteJoin authenticates a remote Disco node.
// The request is expected to contain an Authorization header with a random
// token encoded as a base 58 string. If the token matches an existing and valid
// invitation record, the request body is read, which is expected to contain the
// client's X25519 public key. If successful, ECDH key exchange is performed to
// generate the shared secret key, used to encrypt the generated TLS client
// certificate that is sent in the response.
func (h *Handler) RemoteJoin(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	inv := &models.Invite{Token: token}
	if err := inv.Load(h.appCtx.DB.NewContext(), h.appCtx.DB); err != nil {
		var errNoRes types.ErrNoResult
		if errors.As(err, &errNoRes) {
			_ = render.Render(w, r, lib.ErrUnauthorized())
			return
		}

		_ = render.Render(w, r, lib.ErrBadRequest(err))
		return
	}

	clientPubKeyEnc, err := io.ReadAll(r.Body)
	if err != nil {
		_ = render.Render(w, r, lib.ErrBadRequest(err))
		return
	}

	clientPubKeyData, err := base58.Decode(string(clientPubKeyEnc))
	if err != nil {
		_ = render.Render(w, r, lib.ErrBadRequest(err))
		return
	}

	privKey, err := inv.PrivateKey(h.appCtx.User.PrivateKey)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	sharedKey, _, err := crypto.ECDHExchange(clientPubKeyData, privKey.Bytes())
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	var sharedKeyArr [32]byte
	copy(sharedKeyArr[:], sharedKey)
	// TODO: Generate TLS client certificate
	tlsClientCertEncR, err := crypto.EncryptSym(bytes.NewBufferString("hello"), &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}
	tlsClientCertEnc, err := io.ReadAll(tlsClientCertEncR)
	if err != nil {
		_ = render.Render(w, r, lib.ErrInternal(err))
		return
	}

	_ = render.Render(w, r, &RemoteJoinResponse{
		Response:         &lib.Response{StatusCode: http.StatusOK},
		TLSClientCertEnc: base58.Encode(tlsClientCertEnc),
	})
}
