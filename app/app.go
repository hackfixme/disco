package app

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nrednav/cuid2"

	"go.hackfix.me/disco/app/cli"
	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store"
	"go.hackfix.me/disco/db/store/sqlite"
)

// App is the application.
type App struct {
	name string
	ctx  *actx.Context
	cli  *cli.CLI
	args []string
	// the logging level is set via the CLI, if the app was initialized with the
	// WithLogger option.
	logLevel *slog.LevelVar
}

// New initializes a new application with the given options. dataDir specifies
// the directory where application data will be stored, though this can be
// overriden with the DISCO_DATA_DIR environment variable, and --data-dir CLI
// flag.
func New(name, dataDir string, opts ...Option) (*App, error) {
	version, err := actx.GetVersion()
	if err != nil {
		return nil, err
	}
	defaultCtx := &actx.Context{
		Ctx:     context.Background(),
		Logger:  slog.Default(),
		Version: version,
	}
	app := &App{name: name, ctx: defaultCtx}

	for _, opt := range opts {
		opt(app)
	}

	uuidgen, err := cuid2.Init(cuid2.WithLength(12))
	if err != nil {
		return nil, aerrors.NewRuntimeError(
			"failed creating UUID generation function", err, "")
	}
	app.ctx.UUIDGen = uuidgen

	ver := fmt.Sprintf("%s %s", app.name, app.ctx.Version.String())
	app.cli, err = cli.New(dataDir, ver)
	if err != nil {
		return nil, err
	}

	return app, nil
}

// Run initializes the application environment and starts execution of the
// application.
func (app *App) Run(args []string) error {
	if err := app.cli.Parse(args); err != nil {
		return err
	}

	if app.logLevel != nil {
		app.logLevel.Set(app.cli.Log.Level)
		slog.SetLogLoggerLevel(app.cli.Log.Level)
	}

	if err := app.createDataDir(app.cli.DataDir); err != nil {
		return err
	}
	storeDir := app.cli.DataDir
	if app.ctx.FS.Name() == "MemoryFileSystem" {
		// The SQLite lib will attempt to write directly with the os interface,
		// so prevent it by using SQLite's in-memory support.
		storeDir = ":memory:"
	}
	if err := app.initStores(storeDir); err != nil {
		return err
	}

	if err := app.cli.Execute(app.ctx); err != nil {
		return err
	}

	return nil
}

func (app *App) createDataDir(dir string) error {
	err := app.ctx.FS.MkdirAll(dir, 0o700)
	if err != nil {
		return aerrors.NewRuntimeError(
			fmt.Sprintf("failed creating app data directory '%s'", dir), err, "")
	}
	return nil
}

func (app *App) initStores(dataDir string) error {
	var err error
	if app.ctx.DB == nil {
		app.ctx.DB, err = initDB(app.ctx.Ctx, dataDir)
		if err != nil {
			return err
		}
	}

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
		err = app.ctx.LoadLocalUser(readEncKey)
		if err != nil {
			return err
		}
	}

	if app.ctx.Store == nil {
		var encKey *[32]byte
		if app.ctx.User != nil {
			encKey = app.ctx.User.PrivateKey
		}
		app.ctx.Store, err = initKVStore(app.ctx.Ctx, dataDir, encKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func initDB(ctx context.Context, dataDir string) (*db.DB, error) {
	var d *db.DB
	dbPath := dataDir
	if strings.Contains(dbPath, "mode=memory") {
		defer func() {
			if d != nil {
				// See https://github.com/mattn/go-sqlite3#faq
				d.SetMaxIdleConns(10)
				d.SetConnMaxLifetime(time.Duration(math.Inf(1)))
			}
		}()
	} else {
		dbPath = filepath.Join(dataDir, "disco.db")
	}

	var err error
	d, err = db.Open(ctx, dbPath)
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

func initKVStore(ctx context.Context, dataDir string, encKey *[32]byte) (store.Store, error) {
	var s *sqlite.Store
	storePath := dataDir
	if strings.Contains(storePath, "mode=memory") {
		defer func() {
			if s != nil {
				// See https://github.com/mattn/go-sqlite3#faq
				s.SetMaxIdleConns(10)
				s.SetConnMaxLifetime(time.Duration(math.Inf(1)))
			}
		}()
	} else {
		storePath = filepath.Join(dataDir, "store.db")
	}

	storeOpts := []sqlite.Option{}
	if encKey != nil {
		storeOpts = append(storeOpts, sqlite.WithEncryptionKey(encKey))
	}

	var err error
	s, err = sqlite.Open(ctx, storePath, storeOpts...)
	if err != nil {
		return nil, err
	}

	return s, nil
}
