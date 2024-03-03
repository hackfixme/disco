package cli

import "go.hackfix.me/disco/app/ctx"

// The init command initializes the Disco data stores and generates a new
// encryption key.
type Init struct {
	Directory string `default:"" help:"The directory where to store the data (default: $XDG_DATA_HOME/disco). "`
}

// Run the ls command.
func (c *Init) Run(appCtx *ctx.Context) error {
	return nil
}
