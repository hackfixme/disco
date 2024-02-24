package main

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"go.hackfix.me/disco/cli"
)

func main() {
	app := cli.NewApp(
		cli.WithFS(osfs.New()),
		cli.WithEnv(osEnv{}),
		cli.WithFDs(os.Stdin, os.Stdout, os.Stderr),
		cli.WithStore(),
	)
	app.Run()
}

type osEnv struct{}

var _ cli.Environment = &osEnv{}

func (e osEnv) Get(key string) string {
	return os.Getenv(key)
}

func (e osEnv) Set(key, val string) error {
	return os.Setenv(key, val)
}
