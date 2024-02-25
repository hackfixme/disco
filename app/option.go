package app

import (
	"encoding/hex"
	"io"
	"log"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"go.hackfix.me/disco/app/ctx"
	"go.hackfix.me/disco/store/badger"
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

// WithFDs sets the file descriptors used by the application.
func WithFDs(stdin io.Reader, stdout, stderr io.Writer) Option {
	return func(app *App) {
		app.ctx.Stdin = stdin
		app.ctx.Stdout = stdout
		app.ctx.Stderr = stderr
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
			handleErr(err)
		}

		var encKeyDec []byte
		if len(app.cli.EncryptionKey) > 0 {
			encKeyDec, err = hex.DecodeString(app.cli.EncryptionKey)
			handleErr(err)
		}

		app.ctx.Store, err = badger.Open(storePath, encKeyDec)
		handleErr(err)
	}
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
