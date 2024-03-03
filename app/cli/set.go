package cli

import (
	"errors"

	actx "go.hackfix.me/disco/app/context"
)

// The Set command stores the value of a key.
type Set struct {
	Key   string `arg:"" help:"The unique key that identifies the value."`
	Value string `arg:"" help:"The value."`

	Namespace string `default:"default" help:"The namespace to store the value in."`
}

// Run the set command.
func (c *Set) Run(appCtx *actx.Context) error {
	if c.Namespace == "*" {
		// TODO: Add support for the wildcard namespace. I.e. set the value in
		// all existing namespaces. This would require keeping a registry of
		// existing namespaces.
		return errors.New("namespace '*' is not supported for the set command")
	}
	return appCtx.Store.Set(c.Namespace, []byte(c.Key), []byte(c.Value))
}
