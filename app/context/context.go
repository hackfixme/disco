package context

import (
	"context"
	"io"
	"log/slog"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/store"
)

// Context contains common objects used by the application. It is passed around
// the application to avoid direct dependencies on external systems, and make
// testing easier.
type Context struct {
	Ctx         context.Context
	Version     string // The static app version in the binary
	VersionInit string // The app version the DB was initialized with
	FS          vfs.FileSystem
	Env         Environment
	Logger      *slog.Logger
	UUIDGen     func() string

	// Standard streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	DB    *db.DB
	Store store.Store
	User  *models.User
}

// Environment is the interface to the process environment.
type Environment interface {
	Get(string) string
	Set(string, string) error
}
