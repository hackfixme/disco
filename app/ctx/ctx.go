package ctx

import (
	"io"
	"log/slog"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"go.hackfix.me/disco/db/store"
)

// Context contains common objects used by the application. Most of them are
// interfaces to external systems, such as the filesystem, file descriptors,
// data stores, process environment, etc. It is passed around the application
// to avoid direct dependencies, and make testing easier.
type Context struct {
	FS     vfs.FileSystem
	Env    Environment
	Logger *slog.Logger

	// Standard streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Store store.Store
}

// Environment is the interface to the process environment.
type Environment interface {
	Get(string) string
	Set(string, string) error
}
