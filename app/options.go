package app

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/lmittmann/tint"
	"github.com/mandelsoft/vfs/pkg/vfs"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/store/sqlite"
)

// Option is a function that allows configuring the application.
type Option func(*App)

// WithArgs sets the command arguments passed to the CLI parser.
func WithArgs(args []string) Option {
	return func(app *App) {
		app.args = args
	}
}

// WithDB initializes the database used by the application.
func WithDB(dataDir string) Option {
	return func(app *App) {
		dbCtx, _ := context.WithCancel(app.ctx.Ctx)
		dbPath := dataDir
		if dbPath != ":memory:" {
			dbPath = filepath.Join(dataDir, "disco.db")
		}
		var err error
		app.ctx.DB, err = db.Open(dbCtx, dbPath)
		app.FatalIfErrorf(err)
	}
}

// WithEnv sets the process environment used by the application.
func WithEnv(env actx.Environment) Option {
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

// WithStore initializes the key-value store used by the application.
func WithStore(dataDir string) Option {
	return func(app *App) {
		storeCtx, _ := context.WithCancel(app.ctx.Ctx)
		storePath := dataDir
		if storePath != ":memory:" {
			storePath = filepath.Join(dataDir, "store.db")
		}

		mustValidEncKey := len(app.args) > 0 && (app.args[0] == "get" ||
			app.args[0] == "set" || app.args[0] == "serve")

		storeOpts := []sqlite.Option{}
		if mustValidEncKey {
			storeOpts = append(storeOpts, sqlite.WithEncryptionKey(
				app.ctx.Env.Get("DISCO_ENCRYPTION_KEY")))
		}

		var err error
		app.ctx.Store, err = sqlite.Open(storeCtx, storePath, storeOpts...)
		if err != nil {
			app.FatalIfErrorf(err)
		}
	}
}
