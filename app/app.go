package app

import (
	"context"
	"log/slog"

	"go.hackfix.me/disco/app/cli"
	actx "go.hackfix.me/disco/app/context"
)

// App is the application.
type App struct {
	ctx  *actx.Context
	cli  *cli.CLI
	args []string

	Exit func(int)
}

// New initializes a new application.
func New(opts ...Option) *App {
	defaultCtx := &actx.Context{
		Ctx:    context.Background(),
		Logger: slog.Default(),
	}
	app := &App{ctx: defaultCtx, Exit: func(int) {}}

	for _, opt := range opts {
		opt(app)
	}

	slog.SetDefault(app.ctx.Logger)

	cli := &cli.CLI{}
	err := cli.Setup(app.ctx, app.args, app.Exit)
	if err != nil {
		app.FatalIfErrorf(err)
	}
	app.cli = cli

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
