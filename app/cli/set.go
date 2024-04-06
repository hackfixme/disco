package cli

import (
	"bytes"
	"errors"
	"io"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/web/client"
)

// The Set command stores the value of a key.
type Set struct {
	Key   string `arg:"" help:"The unique key that identifies the value."`
	Value string `arg:"" help:"The value."`

	Namespace string `default:"default" help:"The namespace to store the value in."`
	Remote    string `help:"The remote Disco node to store the value in."`
}

// Run the set command.
func (c *Set) Run(appCtx *actx.Context) error {
	if c.Namespace == "*" {
		// TODO: Add support for the wildcard namespace. I.e. set the value in
		// all existing namespaces. This would require keeping a registry of
		// existing namespaces.
		return errors.New("namespace '*' is not supported for the set command")
	}

	var value io.Reader = bytes.NewReader([]byte(c.Value))
	if c.Value == "-" {
		value = appCtx.Stdin
	}

	var setErr error
	if c.Remote != "" {
		r := &models.Remote{Name: c.Remote}
		if err := r.Load(appCtx.DB.NewContext(), appCtx.DB); err != nil {
			return err
		}

		tlsConfig, err := r.ClientTLSConfig(appCtx.User.PrivateKey)
		if err != nil {
			return err
		}

		client := client.New(r.Address, tlsConfig)
		setErr = client.StoreSet(appCtx.Ctx, c.Namespace, c.Key, value)
		if err != nil {
			return err
		}
	} else {
		setErr = appCtx.Store.Set(c.Namespace, c.Key, value)
	}

	return setErr
}
