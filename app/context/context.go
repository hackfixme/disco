package context

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mr-tron/base58"

	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store"
	"go.hackfix.me/disco/db/types"
)

// Context contains common objects used by the application. It is passed around
// the application to avoid direct dependencies on external systems, and make
// testing easier.
type Context struct {
	Ctx         context.Context
	Version     string // The static app version in the binary
	VersionInit string // The app version the DB was initialized with
	FS          vfs.FileSystem
	DataDir     string
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

// LoadLocalUser loads the local user from the database into c.User.
// If readEncKey is true, it also reads the private encryption key from the
// environment and validates it against its stored hash.
// Note that this *must* load a single user. Currently only a single local user
// is created, but in the future this might change.
func (c *Context) LoadLocalUser(readEncKey bool) error {
	users, err := models.Users(c.DB.NewContext(), c.DB,
		types.NewFilter("u.type = ?", []any{models.UserTypeLocal}))
	if err != nil {
		return aerrors.NewRuntimeError("failed loading local user", err, "")
	}

	switch len(users) {
	case 0:
		return aerrors.NewRuntimeError("local user not found", nil,
			"Did you forget to run 'disco init'?")
	case 1:
		c.User = users[0]
	default:
		return aerrors.NewRuntimeError(
			fmt.Sprintf("found more than 1 local user: %d", len(users)), nil, "")
	}

	if readEncKey {
		privKeyHash, privKeyErr := queries.GetEncryptionPrivKeyHash(c.DB.NewContext(), c.DB)
		if privKeyErr != nil || !privKeyHash.Valid {
			return aerrors.NewRuntimeError("missing encryption key hash",
				privKeyErr, "Did you forget to run 'disco init'?")
		}

		privKeyEnc := c.Env.Get("DISCO_ENCRYPTION_KEY")
		privKey, err := crypto.DecodeKey(privKeyEnc)
		if err != nil {
			return aerrors.NewRuntimeError("invalid encryption key", err, "")
		}

		inPrivKeyHash := crypto.Hash("encryption key hash", privKey[:])
		inPrivKeyHashEnc := base58.Encode(inPrivKeyHash)
		if privKeyHash.V != inPrivKeyHashEnc {
			return aerrors.NewRuntimeError("invalid encryption key", errors.New("hash mismatch"), "")
		}

		c.User.PrivateKey = privKey
	}

	return nil
}

// ServerTLSInfo returns the TLS certificate, private key and Subject
// Alternative Name used by the server.
func (c *Context) ServerTLSInfo() (
	cert *tls.Certificate, certPEM []byte, san string, err error,
) {
	certPEMNull, privKeyEncNull, sanNull, err := queries.GetServerTLSInfo(c.DB.NewContext(), c.DB)
	if err != nil {
		return nil, nil, "", err
	}

	privKey, err := crypto.DecryptSymInMemory(privKeyEncNull.V, c.User.PrivateKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed decrypting server TLS private key: %w", err)
	}

	certPair, err := tls.X509KeyPair([]byte(certPEMNull.V), privKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed parsing PEM encoded TLS certificate: %w", err)
	}

	return &certPair, []byte(certPEMNull.V), sanNull.V, nil
}
