package cli

import (
	"encoding/hex"
	"io"
	"log"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"go.hackfix.me/disco/store"
	"go.hackfix.me/disco/store/badger"
)

// App is the CLI application.
type App struct {
	ctx *appContext
	cli *CLI
}

// Environment is the interface to the process environment.
type Environment interface {
	Get(string) string
	Set(string, string) error
}

// appContext contains interfaces to external systems, such as the filesystem,
// file descriptors, data stores, process environment, etc. It is passed to all
// commands in order to avoid direct dependencies, and make commands easier to
// test.
type appContext struct {
	fs  vfs.FileSystem
	env Environment

	// Standard streams
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	store store.Store
}

// NewApp initializes a new CLI application.
func NewApp(opts ...AppOption) *App {
	cli := &CLI{}
	cli.setup()
	defaultCtx := &appContext{
		fs: memoryfs.New(),
	}

	app := &App{ctx: defaultCtx, cli: cli}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Run starts execution of the application.
func (app *App) Run() {
	err := app.cli.ctx.Run(app.ctx)
	app.cli.ctx.FatalIfErrorf(err)
}

// AppOption is a function that allows configuring the application.
type AppOption func(*App)

// WithFS sets the filesystem used by the application.
func WithFS(fs vfs.FileSystem) AppOption {
	return func(app *App) {
		app.ctx.fs = fs
	}
}

// WithEnv sets the process environment used by the application.
func WithEnv(env Environment) AppOption {
	return func(app *App) {
		app.ctx.env = env
	}
}

// WithFDs sets the file descriptors used by the application.
func WithFDs(stdin io.Reader, stdout, stderr io.Writer) AppOption {
	return func(app *App) {
		app.ctx.stdin = stdin
		app.ctx.stdout = stdout
		app.ctx.stderr = stderr
	}
}

// WithStore initializes the key-value store used by the application.
func WithStore() AppOption {
	return func(app *App) {
		var (
			storePath string
			err       error
		)
		if app.ctx.fs.Name() != "MemoryFileSystem" {
			storePath = filepath.Join(xdg.DataHome, "disco", "store")
			err = app.ctx.fs.MkdirAll(storePath, 0o700)
			handleErr(err)
		}

		var encKeyDec []byte
		if len(app.cli.EncryptionKey) > 0 {
			encKeyDec, err = hex.DecodeString(app.cli.EncryptionKey)
			handleErr(err)
		}

		app.ctx.store, err = badger.Open(storePath, encKeyDec)
		handleErr(err)
	}
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type memEnv struct {
	mx  sync.RWMutex
	env map[string]string
}

var _ Environment = &memEnv{}

func (e *memEnv) Get(key string) string {
	e.mx.RLock()
	defer e.mx.RUnlock()
	return e.env[key]
}

func (e *memEnv) Set(key, val string) error {
	e.mx.Lock()
	defer e.mx.Unlock()
	e.env[key] = val
	return nil
}
