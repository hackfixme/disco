package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/mandelsoft/vfs/pkg/osfs"

	"go.hackfix.me/disco/cli"
)

func main() {
	var c cli.CLI
	ctx := kong.Parse(&c,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	err := ctx.Run(&cli.AppContext{
		FS:     osfs.New(),
		Env:    osEnv{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	ctx.FatalIfErrorf(err)
}

type osEnv struct{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
