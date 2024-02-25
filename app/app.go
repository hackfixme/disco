package app

import (
	"github.com/mandelsoft/vfs/pkg/memoryfs"

	"go.hackfix.me/disco/app/cli"
	"go.hackfix.me/disco/app/ctx"
)

// App is the application.
type App struct {
	ctx *ctx.Context
	cli *cli.CLI
}

// New initializes a new application.
func New(opts ...Option) *App {
	cli := &cli.CLI{}
	cli.Setup()
	defaultCtx := &ctx.Context{
		FS: memoryfs.New(),
	}
	app := &App{ctx: defaultCtx, cli: cli}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Run starts execution of the application.
func (app *App) Run() {
	err := app.cli.Ctx.Run(app.ctx)
	app.cli.Ctx.FatalIfErrorf(err)
}
