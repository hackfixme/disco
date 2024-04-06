package api

import (
	"crypto/ed25519"
	"errors"
	"fmt"
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
// token encoded as a base 58 string, and its signature. If the token matches an
// existing and valid invitation record, the request body is read, which is
// expected to contain the client's X25519 public key. If successful, ECDH key
// exchange is performed to generate the shared secret key, which is used to
// verify the token signature, and encrypt the generated TLS client certificate
// that is sent in the response.
func (h *Handler) RemoteJoin(w http.ResponseWriter, r *http.Request) {
	// Extract the token signature and data from the Authorization header.
	tokenBundle := r.Header.Get("Authorization")
	tokenSig, tokenData, err := decodeToken(tokenBundle)
	if err != nil {
		_ = render.Render(w, r, types.ErrUnauthorized("invalid invite token"))
		return
	}

	// Lookup the token in the DB.
	inv := &models.Invite{Token: base58.Encode(tokenData)}
	if err := inv.Load(h.appCtx.DB.NewContext(), h.appCtx.DB); err != nil {
		var errNoRes dbtypes.ErrNoResult
		if errors.As(err, &errNoRes) {
			_ = render.Render(w, r, types.ErrUnauthorized("invalid invite token"))
			return
		}

		_ = render.Render(w, r, types.ErrBadRequest(err))
		return
	}

	// Read the client's X25519 pubkey from the request body.
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

	// Perform ECDH key exchange to generate the shared secret key.
	sharedKey, _, err := crypto.ECDHExchange(clientPubKeyData, privKey.Bytes())
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	// Finally, verify the token signature, using the shared key as a seed.
	privSignKey := ed25519.NewKeyFromSeed(sharedKey)
	sigVerified := ed25519.Verify(privSignKey.Public().(ed25519.PublicKey),
		tokenData, tokenSig)
	if !sigVerified {
		_ = render.Render(w, r, types.ErrUnauthorized("invalid invite token"))
		return
	}

	// All good, so generate the response payload.
	serverCert, serverCertPEM, err := h.appCtx.ServerTLSCert()
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}
	clientCert, clientKey, err := crypto.NewTLSCert(
		inv.User.Name, []string{types.ServerName}, time.Now().Add(24*time.Hour), serverCert,
	)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	// Encrypt the client's TLS cert and key with the shared key.
	var sharedKeyArr [32]byte
	copy(sharedKeyArr[:], sharedKey)
	clientCertEnc, err := crypto.EncryptSymInMemory(clientCert, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	clientKeyEnc, err := crypto.EncryptSymInMemory(clientKey, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	resp := &types.RemoteJoinResponse{
		Response:         &types.Response{StatusCode: http.StatusOK},
		TLSCACert:        string(serverCertPEM),
		TLSClientCertEnc: base58.Encode(clientCertEnc),
		TLSClientKeyEnc:  base58.Encode(clientKeyEnc),
	}
	_ = render.Render(w, r, resp)
}

func decodeToken(token string) ([]byte, []byte, error) {
	tokenDec, err := base58.Decode(token)
	if err != nil {
		return nil, nil, fmt.Errorf("failed decoding invite token: %w", err)
	}
	if len(tokenDec) != 96 {
		return nil, nil, errors.New("invalid invite token")
	}

	return tokenDec[:64], tokenDec[64:], nil
}
