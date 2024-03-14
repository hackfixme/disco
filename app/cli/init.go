package cli

import (
	"fmt"

	"github.com/mr-tron/base58"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
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

	var err error
	appCtx.User, err = appCtx.DB.Init(appCtx.Version)
	if err != nil {
		return aerrors.NewRuntimeError("failed initializing database", err, "")
	}

	if err = appCtx.Store.Init(appCtx.Version); err != nil {
		return aerrors.NewRuntimeError("failed initializing store", err, "")
	}

	fmt.Fprintf(appCtx.Stdout, `New encryption key: %s

Make sure to store this key in a secure location, such as a password manager.

It will only be shown once, and you won't be able to access the data on this node without it!
	`, base58.Encode(appCtx.User.PrivateKey[:]))

	return nil
}
