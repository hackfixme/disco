package main

import (
	"encoding/hex"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"
	"github.com/mandelsoft/vfs/pkg/osfs"

	"go.hackfix.me/disco/cli"
	"go.hackfix.me/disco/store/badger"
)

func main() {
	appCtx := &cli.AppContext{
		FS:     osfs.New(),
		Env:    osEnv{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	var c cli.CLI
	defer setupCLI(&c, appCtx)()

	setupStore(appCtx, c.EncryptionKey)
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func setupCLI(c *cli.CLI, appCtx *cli.AppContext) func() {
	ctx := kong.Parse(c,
		kong.Name("disco"),
		kong.UsageOnError(),
		kong.DefaultEnvars("DISCO"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	return func() {
		err := ctx.Run(appCtx)
		ctx.FatalIfErrorf(err)
	}
}

func setupStore(appCtx *cli.AppContext, encKey string) {
	storePath := filepath.Join(xdg.DataHome, "disco", "store")
	err := appCtx.FS.MkdirAll(storePath, 0o700)
	handleErr(err)

	var encKeyDec []byte
	if len(encKey) > 0 {
		encKeyDec, err = hex.DecodeString(encKey)
		handleErr(err)
	}

	appCtx.Store, err = badger.Open(storePath, encKeyDec)
	handleErr(err)
}

type osEnv struct{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
