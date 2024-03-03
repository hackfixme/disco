package cli

import (
	"errors"
	"fmt"

	actx "go.hackfix.me/disco/app/context"
)

// The Get command retrieves and prints the value of a key.
type Get struct {
	Key string `arg:"" help:"The unique key associated with the value."`

	Namespace string `default:"default" help:"The namespace to retrieve the value from."`
}

// Run the get command.
func (c *Get) Run(appCtx *actx.Context) error {
	if c.Namespace == "*" {
		// TODO: Think about how the wildcard namespace could work for the get
		// command. Output values for the given key in all namespaces, separated
		// by \0?
		return errors.New("namespace '*' is not supported for the get command")
	}

	val, err := appCtx.Store.Get(c.Namespace, []byte(c.Key))
	if err != nil {
		return err
	}

	fmt.Fprintf(appCtx.Stdout, "%s\n", val)

	return nil
}
