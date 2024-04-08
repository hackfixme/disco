package cli

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/mr-tron/base58"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
)

// The Init command initializes the Disco data stores and generates a new
// encryption key.
type Init struct{}

// Run the init command.
func (c *Init) Run(appCtx *actx.Context) error {
	if appCtx.VersionInit != "" {
		// TODO: Add --force option?
		return fmt.Errorf("Disco is already initialized with version %s", appCtx.VersionInit)
	}

	rndSANb := make([]byte, 16)
	_, err := rand.Read(rndSANb)
	if err != nil {
		return err
	}
	rndSAN := base58.Encode(rndSANb)
	tlsCert, tlsPrivKey, err := crypto.NewTLSCert(
		"disco server", []string{rndSAN}, time.Now().Add(24*time.Hour), nil,
	)
	if err != nil {
		return fmt.Errorf("failed generating the server TLS certificate: %w", err)
	}

	appCtx.User, err = appCtx.DB.Init(appCtx.Version, tlsCert, tlsPrivKey, rndSAN, appCtx.Logger)
	if err != nil {
		return aerrors.NewRuntimeError("failed initializing database", err, "")
	}

	if err = appCtx.Store.Init(appCtx.Version, appCtx.Logger); err != nil {
		return aerrors.NewRuntimeError("failed initializing store", err, "")
	}

	fmt.Fprintf(appCtx.Stdout, `New encryption key: %s

Make sure to store this key in a secure location, such as a password manager.

It will only be shown once, and you won't be able to access the data on this node without it!
	`, base58.Encode(appCtx.User.PrivateKey[:]))

	return nil
}
