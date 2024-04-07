package api

import (
	"crypto/x509"
	"encoding/pem"
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

	serverCert, err := h.appCtx.ServerTLSCert()
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}
	clientCert, clientKey, err := crypto.NewTLSCert(
		inv.User.Name, []string{"localhost"}, time.Now().Add(24*time.Hour), serverCert,
	)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	var sharedKeyArr [32]byte
	copy(sharedKeyArr[:], sharedKey)

	tlsClientCertEnc, err := crypto.EncryptSymInMemory(clientCert, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	tlsClientKeyEnc, err := crypto.EncryptSymInMemory(clientKey, &sharedKeyArr)
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(err))
		return
	}

	if len(serverCert.Certificate) == 0 {
		_ = render.Render(w, r, types.ErrInternal(errors.New("no certificate data found in parent certificate")))
		return
	}

	x509Cert, err := x509.ParseCertificate(serverCert.Certificate[0])
	if err != nil {
		_ = render.Render(w, r, types.ErrInternal(
			fmt.Errorf("failed parsing server X.509 certificate: %w", err)))
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: x509Cert.Raw})
	if certPEM == nil {
		_ = render.Render(w, r, types.ErrInternal(errors.New("failed encoding TLS certificate to PEM")))
		return
	}

	resp := &types.RemoteJoinResponse{
		Response:         &types.Response{StatusCode: http.StatusOK},
		TLSCACert:        string(certPEM),
		TLSClientCertEnc: base58.Encode(tlsClientCertEnc),
		TLSClientKeyEnc:  base58.Encode(tlsClientKeyEnc),
	}
	_ = render.Render(w, r, resp)
}
