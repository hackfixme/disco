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

	storePath := filepath.Join(xdg.DataHome, "disco", "store")
	fs := osfs.New()
	err := fs.MkdirAll(storePath, 0o700)
	handleErr(err)

	var encKey []byte
	if len(c.EncryptionKey) > 0 {
		encKey, err = hex.DecodeString(c.EncryptionKey)
		handleErr(err)
	}
	store, err := badger.Open(storePath, encKey)
	handleErr(err)

	err = ctx.Run(&cli.AppContext{
		FS:     fs,
		Env:    osEnv{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Store:  store,
	})
	ctx.FatalIfErrorf(err)
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type osEnv struct{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
