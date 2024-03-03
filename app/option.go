package app

import (
	"encoding/hex"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/lmittmann/tint"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"go.hackfix.me/disco/app/ctx"
	"go.hackfix.me/disco/db/store/badger"
)

// Option is a function that allows configuring the application.
type Option func(*App)

// WithFS sets the filesystem used by the application.
func WithFS(fs vfs.FileSystem) Option {
	return func(app *App) {
		app.ctx.FS = fs
	}
}

// WithEnv sets the process environment used by the application.
func WithEnv(env ctx.Environment) Option {
	return func(app *App) {
		app.ctx.Env = env
	}
}

// WithExit sets the function that stops the application.
func WithExit(fn func(int)) Option {
	return func(app *App) {
		app.Exit = fn
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

// WithStore initializes the key-value store used by the application.
func WithStore() Option {
	return func(app *App) {
		var (
			storePath string
			err       error
		)
		if app.ctx.FS.Name() != "MemoryFileSystem" {
			storePath = filepath.Join(xdg.DataHome, "disco", "store")
			err = app.ctx.FS.MkdirAll(storePath, 0o700)
			app.FatalIfErrorf(err)
		}

		var encKeyDec []byte
		if len(app.cli.EncryptionKey) > 0 {
			encKeyDec, err = hex.DecodeString(app.cli.EncryptionKey)
			app.FatalIfErrorf(err)
		}

		app.ctx.Store, err = badger.Open(storePath, encKeyDec)
		app.FatalIfErrorf(err)
	}
}
