package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/mr-tron/base58"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/models"
	dbtypes "go.hackfix.me/disco/db/types"
	"go.hackfix.me/disco/web/server/types"
)

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
		var errNoRes dbtypes.ErrNoResult
		if errors.As(err, &errNoRes) {
			_ = render.Render(w, r, types.ErrUnauthorized())
			return
		}

		_ = render.Render(w, r, types.ErrBadRequest(err))
		return
	}

	clientPubKeyEnc, err := io.ReadAll(r.Body)
	if err != nil {
		_ = render.Render(w, r, types.ErrBadRequest(err))
		return
	}

	clientPubKeyData, err := base58.Decode(string(clientPubKeyEnc))
	if err != nil {
		_ = render.Render(w, r, types.ErrBadRequest(err))
		return
	}

	privKey, err := inv.PrivateKey(h.appCtx.User.PrivateKey)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	sharedKey, _, err := crypto.ECDHExchange(clientPubKeyData, privKey.Bytes())
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	clientCert, clientKey, err := crypto.NewTLSCert(
		inv.User.Name, []string{"localhost"}, time.Now().Add(24*time.Hour), nil,
	)

	var sharedKeyArr [32]byte
	copy(sharedKeyArr[:], sharedKey)

	tlsClientCertEnc, err := encrypt(clientCert, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	tlsClientKeyEnc, err := encrypt(clientKey, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	_ = render.Render(w, r, &types.RemoteJoinResponse{
		Response:         &types.Response{StatusCode: http.StatusOK},
		TLSCACert:        string(h.appCtx.TLSCACert),
		TLSClientCertEnc: base58.Encode(tlsClientCertEnc),
		TLSClientKeyEnc:  base58.Encode(tlsClientKeyEnc),
	})
}

func encrypt(data []byte, key *[32]byte) ([]byte, error) {
	dataEncR, err := crypto.EncryptSym(bytes.NewBuffer(data), key)
	if err != nil {
		return nil, err
	}
	dataEnc, err := io.ReadAll(dataEncR)
	if err != nil {
		return nil, err
	}

	return dataEnc, nil
}
