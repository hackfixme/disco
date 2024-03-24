package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"go.hackfix.me/disco/app"
	actx "go.hackfix.me/disco/app/context"
)

func main() {
	fs := osfs.New()
	dataDir := filepath.Join(xdg.DataHome, "disco")
	err := fs.MkdirAll(dataDir, 0o700)
	if err != nil {
		panic(fmt.Sprintf("failed creating app directory '%s': %s", dataDir, err))
	}

	// NOTE: The order of the passed options is significant, as some options depend
	// on the values set by previous ones.
	app.New(
		app.WithExit(os.Exit),
		app.WithArgs(os.Args[1:]),
		app.WithEnv(osEnv{}),
		app.WithFDs(
			os.Stdin,
			colorable.NewColorable(os.Stdout),
			colorable.NewColorable(os.Stderr),
		),
		app.WithFS(fs),
		app.WithLogger(
			isatty.IsTerminal(os.Stdout.Fd()),
			isatty.IsTerminal(os.Stderr.Fd()),
		),
		app.WithDB(dataDir),
		app.WithLocalUser(nil),
		app.WithStore(dataDir),
	).Run()
}

type osEnv struct{}

var _ actx.Environment = &osEnv{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
