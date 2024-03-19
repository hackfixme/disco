package cli

import (
	"errors"

	actx "go.hackfix.me/disco/app/context"
)

// The Rm command deletes a key.
type Rm struct {
	Key       string `arg:"" help:"The key to delete."`
	Namespace string `default:"default" help:"The namespace to key exists in."`
}

// Run the rm command.
func (c *Rm) Run(appCtx *actx.Context) error {
	if c.Namespace == "*" {
		// TODO: Add support for the wildcard namespace. I.e. remove the value
		// from all existing namespaces.
		return errors.New("namespace '*' is not supported for the rm command")
	}

	return appCtx.Store.Delete(c.Namespace, c.Key)
}
