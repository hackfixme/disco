package context

import (
	"context"
	"io"
	"log/slog"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/store"
)

// Context contains common objects used by the application. It is passed around
// the application to avoid direct dependencies on external systems, and make
// testing easier.
type Context struct {
	Ctx    context.Context
	FS     vfs.FileSystem
	Env    Environment
	Logger *slog.Logger

	// Standard streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	DB    *db.DB
	Store store.Store
}

// Environment is the interface to the process environment.
type Environment interface {
	Get(string) string
	Set(string, string) error
}
