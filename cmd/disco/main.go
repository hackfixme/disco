package main

import (
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
	// NOTE: The order of the passed options is significant, as some options
	// depend on the values set by previous ones.
	a, err := app.New(filepath.Join(xdg.DataHome, "disco"),
		app.WithFDs(
			os.Stdin,
			colorable.NewColorable(os.Stdout),
			colorable.NewColorable(os.Stderr),
		),
		app.WithFS(osfs.New()),
		app.WithLogger(
			isatty.IsTerminal(os.Stdout.Fd()),
			isatty.IsTerminal(os.Stderr.Fd()),
		),
		app.WithEnv(osEnv{}),
	)
	if err != nil {
		app.Errorf(err)
		os.Exit(1)
	}
	if err = a.Run(os.Args[1:]); err != nil {
		app.Errorf(err)
		os.Exit(1)
	}
}

type osEnv struct{}

var _ actx.Environment = &osEnv{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
