package app

import (
	"context"
	"log/slog"

	"go.hackfix.me/disco/app/cli"
	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
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
		Ctx:     context.Background(),
		Logger:  slog.Default(),
		Version: version,
	}
	app := &App{ctx: defaultCtx, Exit: func(int) {}}

	for _, opt := range opts {
		opt(app)
	}

	if app.ctx.User == nil && app.ctx.VersionInit != "" {
		app.ctx.User = &models.User{Name: "local"}
		if err := app.ctx.User.Load(app.ctx.DB.NewContext(), app.ctx.DB); err != nil {
			app.FatalIfErrorf(aerrors.NewRuntimeError(
				"failed loading local user", err, ""))
		}
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
func (app *App) FatalIfErrorf(err error, args ...any) {
	if err != nil {
		msg := err.Error()
		if errh, ok := err.(aerrors.WithHint); ok {
			hint := errh.Hint()
			if hint != "" {
				args = append([]any{"hint", hint}, args...)
			}
		}
		if errc, ok := err.(aerrors.WithCause); ok {
			cause := errc.Cause()
			if cause != nil {
				args = append([]any{"cause", cause}, args...)
			}
		}
		app.ctx.Logger.Error(msg, args...)
		app.Exit(1)
	}
}
