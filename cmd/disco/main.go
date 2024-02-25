package main

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"

	"go.hackfix.me/disco/app"
	"go.hackfix.me/disco/app/ctx"
)

func main() {
	app.New(
		app.WithFS(osfs.New()),
		app.WithEnv(osEnv{}),
		app.WithFDs(os.Stdin, os.Stdout, os.Stderr),
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
