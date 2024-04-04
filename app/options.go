package app

import (
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
	"github.com/mandelsoft/vfs/pkg/vfs"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/db/models"
)

// Option is a function that allows configuring the application.
type Option func(*App)

// WithEnv sets the process environment used by the application.
func WithEnv(env actx.Environment) Option {
	return func(app *App) {
		app.ctx.Env = env
	}
}

// WithFDs sets the file descriptors used by the application.
func WithFDs(stdin io.Reader, stdout, stderr io.Writer) Option {
	return func(app *App) {
		app.ctx.Stdin = stdin
		app.ctx.Stdout = stdout
		app.ctx.Stderr = stderr
	}
}

// WithFS sets the filesystem used by the application.
func WithFS(fs vfs.FileSystem) Option {
	return func(app *App) {
		app.ctx.FS = fs
	}
}

// WithLogger initializes the logger used by the application.
func WithLogger(isStdoutTTY, isStderrTTY bool) Option {
	return func(app *App) {
		logger := slog.New(
			tint.NewHandler(app.ctx.Stderr, &tint.Options{
				Level:      slog.LevelDebug, // TODO: Make configurable
				NoColor:    !isStderrTTY,
				TimeFormat: "2006-01-02 15:04:05.000",
			}),
		)
		app.ctx.Logger = logger
		slog.SetDefault(logger)
	}
}

// WithUser sets the local user of the app.
func WithUser(user *models.User) Option {
	return func(app *App) {
		app.ctx.User = user
	}
}
