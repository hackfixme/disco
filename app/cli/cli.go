package cli

import (
	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
)

// CLI is the command line interface of disco.
type CLI struct {
	Ctx *kong.Context `kong:"-"`

	Init   Init   `kong:"cmd,help='Initialize the data stores and generate the encryption key.'"`
	Get    Get    `kong:"cmd,help='Get the value of a key.'"`
	Set    Set    `kong:"cmd,help='Set the value of a key.'"`
	Rm     Rm     `kong:"cmd,help='Delete a key.'"`
	Ls     Ls     `kong:"cmd,help='List keys.'"`
	Role   Role   `kong:"cmd,help='Manage roles.'"`
	Serve  Serve  `kong:"cmd,help='Start the web server.'"`
	User   User   `kong:"cmd,help='Manage users.'"`
	Invite Invite `kong:"cmd,help='Manage invitations for remote users.'"`

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
			Compact:             true,
			Summary:             true,
			NoExpandSubcommands: true,
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
