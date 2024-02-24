package cli

import (
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"go.hackfix.me/disco/store"
)

// CLI is the command line interface of disco.
type CLI struct {
	Get Get `kong:"cmd,help='Get the value of a key.'"`
	Set Set `kong:"cmd,help='Set the value of a key.'"`
	LS  LS  `kong:"cmd,help='List keys.'"`

	EncryptionKey string `kong:"help='AES private key used for encrypting the local data store.\n It must be either 16, 24, or 32 bytes, for AES-128, AES-192 or AES-256 respectively. '"`
}

// AppContext contains interfaces to external systems, such as the filesystem,
// file descriptors, data stores, process environment, etc. It is passed to all
// commands in order to avoid direct dependencies, and make commands easier to
// test.
type AppContext struct {
	FS  vfs.FileSystem
	Env Environment

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
