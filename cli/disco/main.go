package main

import (
	"github.com/alecthomas/kong"
	"go.hackfix.me/disco/cli"
)

func main() {
	var cli cli.CLI
	ctx := kong.Parse(&cli,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
