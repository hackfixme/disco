package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/nrednav/cuid2"

	"go.hackfix.me/disco/app/cli"
	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store"
	"go.hackfix.me/disco/db/store/sqlite"
)

// App is the application.
type App struct {
	ctx  *actx.Context
	cli  *cli.CLI
	args []string

	Exit func(int)
}

// New initializes a new application with the given options. dataDir specifies
// the directory where application data will be stored, though this can be
// overriden with the DISCO_DATA_DIR environment variable, and --data-dir CLI
// flag.
func New(dataDir string, opts ...Option) *App {
	defaultCtx := &actx.Context{
		Ctx:     context.Background(),
		Logger:  slog.Default(),
		Version: version,
	}
	app := &App{ctx: defaultCtx, Exit: func(int) {}}

	for _, opt := range opts {
		opt(app)
	}

	uuidgen, err := cuid2.Init(cuid2.WithLength(12))
	if err != nil {
		app.FatalIfErrorf(aerrors.NewRuntimeError(
			"failed creating UUID generation function", err, ""))
	}
	app.ctx.UUIDGen = uuidgen

	slog.SetDefault(app.ctx.Logger)

	app.cli, err = cli.New(app.ctx, app.args, app.Exit, dataDir)
	app.FatalIfErrorf(err)

	return app
}

// Run initializes the application environment and starts execution of the
// application.
func (app *App) Run() {
	app.createDataDir(app.cli.DataDir)
	storeDir := app.cli.DataDir
	if app.ctx.FS.Name() == "MemoryFileSystem" {
		// The SQLite lib will attempt to write directly with the os interface,
		// so prevent it by using SQLite's in-memory support.
		storeDir = ":memory:"
	}
	app.initStores(storeDir)

	err := app.cli.Execute(app.ctx)
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

func (app *App) createDataDir(dir string) {
	err := app.ctx.FS.MkdirAll(dir, 0o700)
	if err != nil {
		app.FatalIfErrorf(aerrors.NewRuntimeError(
			fmt.Sprintf("failed creating app data directory '%s'", dir), err, ""))
	}
}

func (app *App) initStores(dataDir string) {
	var err error
	app.ctx.DB, err = initDB(app.ctx.Ctx, dataDir)
	app.FatalIfErrorf(err)

	version, err := queries.Version(app.ctx.DB.NewContext(), app.ctx.DB)
	if version.Valid {
		app.ctx.VersionInit = version.V
	}

	// Only load the local user if it's not set and we're currrently not
	// initializing. If we're initializing, the migrations haven't been run at
	// this point, so the schema doesn't exist yet.
	cmd := app.cli.Command()
	if app.ctx.User == nil && cmd != "init" {
		// The encryption key is only required for specific commands.
		encKeyCommands := []string{"get", "set", "ls", "serve", "invite user", "remote add"}
		readEncKey := slices.Contains(encKeyCommands, cmd)
		err := app.ctx.LoadLocalUser(readEncKey)
		app.FatalIfErrorf(err)
	}

	app.ctx.Store, err = initKVStore(app.ctx.Ctx, dataDir, app.ctx.User)
	app.FatalIfErrorf(err)
}

func initDB(ctx context.Context, dataDir string) (*db.DB, error) {
	dbPath := dataDir
	if dbPath != ":memory:" {
		dbPath = filepath.Join(dataDir, "disco.db")
	}
	d, err := db.Open(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign key enforcement
	_, err = d.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func initKVStore(ctx context.Context, dataDir string, localUser *models.User) (store.Store, error) {
	storePath := dataDir
	if storePath != ":memory:" {
		storePath = filepath.Join(dataDir, "store.db")
	}

	storeOpts := []sqlite.Option{}
	if localUser != nil {
		storeOpts = append(storeOpts, sqlite.WithEncryptionKey(localUser.PrivateKey))
	}

	store, err := sqlite.Open(ctx, storePath, storeOpts...)
	if err != nil {
		return nil, err
	}

	return store, nil
}
