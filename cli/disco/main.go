package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mandelsoft/vfs/pkg/osfs"

	"go.hackfix.me/disco/cli"
	"go.hackfix.me/disco/store/badger"
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

	store, err := badger.Open("/tmp/badger")
	if err != nil {
		log.Fatal(err)
	}

	err = ctx.Run(&cli.AppContext{
		FS:     osfs.New(),
		Env:    osEnv{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Store:  store,
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
