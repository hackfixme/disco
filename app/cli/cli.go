package cli

import "github.com/alecthomas/kong"

// CLI is the command line interface of disco.
type CLI struct {
	Ctx *kong.Context

	Get Get `kong:"cmd,help='Get the value of a key.'"`
	Set Set `kong:"cmd,help='Set the value of a key.'"`
	LS  LS  `kong:"cmd,help='List keys.'"`

	EncryptionKey string `kong:"help='AES private key used for encrypting the local data store.\n It must be either 16, 24, or 32 bytes, for AES-128, AES-192 or AES-256 respectively. '"`
}

// Setup the command-line interface.
func (c *CLI) Setup() {
	c.Ctx = kong.Parse(c,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
}
