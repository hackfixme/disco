package cli

import (
	"strings"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
)

// CLI is the command line interface of disco.
type CLI struct {
	kctx *kong.Context

	Init   Init   `kong:"cmd,help='Initialize the data stores and generate the encryption key.'"`
	Get    Get    `kong:"cmd,help='Get the value of a key.'"`
	Set    Set    `kong:"cmd,help='Set the value of a key.'"`
	Rm     Rm     `kong:"cmd,help='Delete a key.'"`
	Ls     Ls     `kong:"cmd,help='List keys.'"`
	Role   Role   `kong:"cmd,help='Manage roles.'"`
	Serve  Serve  `kong:"cmd,help='Start the web server.'"`
	User   User   `kong:"cmd,help='Manage users.'"`
	Invite Invite `kong:"cmd,help='Manage invitations for remote users.'"`
	Remote Remote `kong:"cmd,help='Manage remote Disco nodes.'"`

	DataDir       string `kong:"default='${dataDir}',help='Directory to store Disco data in.'"`
	EncryptionKey string `kong:"help='32-byte private key used for encrypting and decrypting the local data store, encoded in base 58. '"`
}

// New initializes the command-line interface.
func New(appCtx *actx.Context, args []string, exitFn func(int), dataDir string) (*CLI, error) {
	c := &CLI{}
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
		kong.Vars{"dataDir": dataDir},
	)
	if err != nil {
		return nil, err
	}

	kparser.Stdout = appCtx.Stdout
	kparser.Stderr = appCtx.Stderr

	kctx, err := kparser.Parse(args)
	if err != nil {
		return nil, err
	}

	c.kctx = kctx

	return c, nil
}

// Execute starts the command execution.
func (c *CLI) Execute(appCtx *actx.Context) error {
	return c.kctx.Run(appCtx)
}

// Command returns the full path of the executed command.
func (c *CLI) Command() string {
	cmdPath := []string{}
	for _, p := range c.kctx.Path {
		if p.Command != nil {
			cmdPath = append(cmdPath, p.Command.Name)
		}
	}

	return strings.Join(cmdPath, " ")
}
