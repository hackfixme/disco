package db

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	_ "github.com/glebarez/go-sqlite"
	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/types"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*sql.DB
	ctx        context.Context
	migrations []*migrator.Migration
}

var _ types.Querier = &DB{}

func Open(ctx context.Context, path string) (*DB, error) {
	sqliteDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	d := &DB{DB: sqliteDB, ctx: ctx}

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

// NewContext returns a new child context of the main database context.
func (d *DB) NewContext() context.Context {
	// TODO: Return cancel func?
	ctx, _ := context.WithCancel(d.ctx)
	return ctx
}
