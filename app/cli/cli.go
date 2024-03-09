package cli

import (
	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
)

// CLI is the command line interface of disco.
type CLI struct {
	Ctx *kong.Context `kong:"-"`

	Init  Init  `kong:"cmd,help='Initialize the data stores and generate the encryption key.'"`
	Get   Get   `kong:"cmd,help='Get the value of a key.'"`
	Set   Set   `kong:"cmd,help='Set the value of a key.'"`
	LS    LS    `kong:"cmd,help='List keys.'"`
	Serve Serve `kong:"cmd,help='Start the web server.'"`

	EncryptionKey string `kong:"help='32-byte private key used for encrypting and decrypting the local data store, encoded in base 58. '"`
}

// Setup the command-line interface.
func (c *CLI) Setup(appCtx *actx.Context, args []string, exitFn func(int)) error {
	kparser, err := kong.New(c,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.Exit(exitFn),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
	if err != nil {
		return err
	}

	kparser.Stdout = appCtx.Stdout
	kparser.Stderr = appCtx.Stderr

	ctx, err := kparser.Parse(args)
	if err != nil {
		return err
	}

	c.Ctx = ctx

	return nil
}
