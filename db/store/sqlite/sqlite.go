package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"io/fs"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store"
	"go.hackfix.me/disco/db/types"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	*sql.DB
	ctx        context.Context
	encKey     *[32]byte
	migrations []*migrator.Migration
}

var _ store.Store = &Store{}

func Open(ctx context.Context, path string, encKey *[32]byte) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	d := &Store{DB: db, ctx: ctx}
	if !validEncryptionKey(ctx, d.AsQuerier(), encKey) {
		return nil, errors.New("invalid encryption key")
	}
	d.encKey = encKey

	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	migrations, err := migrator.LoadMigrations(migrationsDir)
	if err != nil {
		return nil, err
	}
	d.migrations = migrations

	return d, nil
}

func validEncryptionKey(ctx context.Context, d types.Querier, encKey *[32]byte) bool {
	existingKeyHash, err := queries.GetEncryptionKeyHash(ctx, d)
	if err != nil || !existingKeyHash.Valid {
		return false
	}

	keyHash := crypto.Hash("encryption key hash", encKey[:])
	keyHashHex := hex.EncodeToString(keyHash)

	return existingKeyHash.V == keyHashHex
}

func (s *Store) Get(namespace, key string) (value []byte, err error) {
	return nil, nil
}

func (s *Store) Set(namespace, key string, value []byte) error {
	return nil
}

func (s *Store) List(namespace, key string) map[string][][]byte {
	return nil
}

// Migrations returns all database migrations.
func (s *Store) Migrations() []*migrator.Migration {
	return s.migrations
}

type q struct {
	*sql.DB
	ctx context.Context
}

// NewContext returns a new child context of the main database context.
func (d *q) NewContext() context.Context {
	// TODO: Return cancel func?
	ctx, _ := context.WithCancel(d.ctx)
	return ctx
}

// AsQuerier returns the wrapped store database as a Querier implementation.
// This is only needed if the store is backed by a RDBMS.
func (s *Store) AsQuerier() types.Querier {
	return &q{DB: s.DB, ctx: s.ctx}
}
