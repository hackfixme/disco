package app

import (
	"log/slog"

	"github.com/mandelsoft/vfs/pkg/memoryfs"

	"go.hackfix.me/disco/app/cli"
	"go.hackfix.me/disco/app/ctx"
)

// App is the application.
type App struct {
	ctx *ctx.Context
	cli *cli.CLI

	Exit func(int)
}

// New initializes a new application.
func New(opts ...Option) *App {
	cli := &cli.CLI{}
	cli.Setup()

	defaultCtx := &ctx.Context{
		FS:     memoryfs.New(),
		Logger: slog.Default(),
	}
	app := &App{ctx: defaultCtx, cli: cli, Exit: func(int) {}}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Run starts execution of the application.
func (app *App) Run() {
	err := app.cli.Ctx.Run(app.ctx)
	app.FatalIfErrorf(err)
}

// FatalIfErrorf terminates the application with an error message if err != nil.
func (app *App) FatalIfErrorf(err error, args ...interface{}) {
	if err != nil {
		app.ctx.Logger.Error(err.Error(), args...)
		app.Exit(1)
	}
}
