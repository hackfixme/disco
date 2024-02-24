package cli

import "fmt"

// The Get command retrieves and prints the value of a key.
type Get struct {
	Key string `arg:"" help:"The unique key associated with the value."`
}

// Run the get command.
func (c *Get) Run(appCtx *AppContext) error {
	val, err := appCtx.Store.Get([]byte(c.Key))
	if err != nil {
		return err
	}

	fmt.Fprintf(appCtx.Stdout, "%s\n", val)

	return nil
}
