package cli

import (
	"log/slog"
	"strings"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
)

// CLI is the command line interface of disco.
type CLI struct {
	kong *kong.Kong
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

	Version       kong.VersionFlag `kong:"help='Output Disco version and exit.'"`
	DataDir       string           `kong:"default='${dataDir}',help='Directory to store Disco data in.'"`
	EncryptionKey string           `kong:"help='32-byte private key used for encrypting and decrypting the local data store, encoded in base 58. '"`
	Log           struct {
		Level slog.Level `enum:"DEBUG,INFO,WARN,ERROR" default:"INFO" help:"Set the app logging level."`
	} `embed:"" prefix:"log-"`
}

// New initializes the command-line interface.
func New(dataDir, version string) (*CLI, error) {
	c := &CLI{}
	kparser, err := kong.New(c,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			Summary:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"dataDir": dataDir,
			"version": version,
		},
	)
	if err != nil {
		return nil, err
	}

	c.kong = kparser

	return c, nil
}

// Execute starts the command execution. Parse must be called before this method.
func (c *CLI) Execute(appCtx *actx.Context) error {
	if c.kctx == nil {
		panic("the CLI wasn't initialized properly")
	}
	c.kong.Stdout = appCtx.Stdout
	c.kong.Stderr = appCtx.Stderr

	return c.kctx.Run(appCtx)
}

// Parse the given command line arguments. This method must be called before
// Execute.
func (c *CLI) Parse(args []string) error {
	kctx, err := c.kong.Parse(args)
	if err != nil {
		return err
	}
	c.kctx = kctx

	return nil
}

// Command returns the full path of the executed command.
func (c *CLI) Command() string {
	if c.kctx == nil {
		panic("the CLI wasn't initialized properly")
	}
	cmdPath := []string{}
	for _, p := range c.kctx.Path {
		if p.Command != nil {
			cmdPath = append(cmdPath, p.Command.Name)
		}
	}

	return strings.Join(cmdPath, " ")
}
