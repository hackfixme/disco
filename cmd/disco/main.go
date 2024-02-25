package main

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"

	"go.hackfix.me/disco/app"
	"go.hackfix.me/disco/app/ctx"
)

func main() {
	app.New(
		app.WithExit(os.Exit),
		app.WithFDs(
			os.Stdin,
			colorable.NewColorable(os.Stdout),
			colorable.NewColorable(os.Stderr),
		),
		app.WithLogger(
			isatty.IsTerminal(os.Stdout.Fd()),
			isatty.IsTerminal(os.Stderr.Fd()),
		),
		app.WithFS(osfs.New()),
		app.WithEnv(osEnv{}),
		app.WithStore(),
	).Run()
}

type osEnv struct{}

var _ ctx.Environment = &osEnv{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
