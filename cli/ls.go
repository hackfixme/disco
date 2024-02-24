package cli

import "fmt"

// The LS command prints keys.
type LS struct {
	KeyPrefix string `arg:"" optional:"" help:"An optional key prefix."`
}

// Run the ls command.
func (c *LS) Run(appCtx *AppContext) error {
	keys := appCtx.Store.List([]byte(c.KeyPrefix))
	for _, key := range keys {
		fmt.Fprintf(appCtx.Stdout, "%s\n", key)
	}

	return nil
}
