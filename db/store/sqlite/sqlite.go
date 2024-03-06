package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"

	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	*sql.DB
	ctx              context.Context
	encKey           *[32]byte
	migrations       []*migrator.Migration
	validTableNameRx *regexp.Regexp
}

var _ store.Store = &Store{}

func Open(ctx context.Context, path string, opts ...Option) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	d := &Store{DB: db, ctx: ctx, validTableNameRx: regexp.MustCompile(`^[a-zA-Z0-9-_/.]+$`)}

	var optErr error
	for _, opt := range opts {
		optErr = opt(d)
		if optErr != nil {
			return nil, optErr
		}
	}

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

// Get returns the value associated with a key within a specific namespace.
func (s *Store) Get(namespace, key string) (value []byte, err error) {
	// Validate the table name to ensure it actually exists. This prevents
	// possible SQL injection attacks, since we parametrize the table name below.
	allTables, err := queries.GetAllTables(s.NewContext(), s)
	if err != nil {
		return nil, err
	}
	if _, ok := allTables[namespace]; !ok {
		return nil, nil
	}

	// Namespaces are stored in different tables, but parameterization is not
	// supported for table names, so template it manually.
	err = s.QueryRowContext(s.ctx, fmt.Sprintf(`SELECT value
		FROM "%s"
		WHERE key = ?`, namespace), key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	decValue, err := crypto.Decrypt(value, s.encKey)
	if err != nil {
		return nil, aerrors.NewRuntimeError("failed decrypting value", err, "")
	}

	return decValue, nil
}

func (s *Store) Set(namespace, key string, value []byte) error {
	// Validate the table name to ensure it actually exists. This prevents
	// possible SQL injection attacks, since we parametrize the table name below.
	allTables, err := queries.GetAllTables(s.NewContext(), s)
	if err != nil {
		return err
	}

	if _, ok := allTables[namespace]; !ok {
		// The namespace/table doesn't exist, so sanitize it before creating it.
		if s.validTableNameRx.Match([]byte(namespace)) {
			_, err = s.ExecContext(s.NewContext(), fmt.Sprintf(`CREATE TABLE "%s" (
				key VARCHAR UNIQUE NOT NULL,
				value BLOB
			)`, namespace))
			if err != nil {
				return aerrors.NewRuntimeError("failed creating namespace", err, "")
			}
		} else {
			return fmt.Errorf("invalid namespace: '%s'", namespace)
		}
	}

	encValue, err := crypto.Encrypt(value, s.encKey)
	if err != nil {
		return aerrors.NewRuntimeError("failed encrypting value", err, "")
	}
	_, err = s.ExecContext(s.NewContext(), fmt.Sprintf(
		`INSERT INTO "%s" (key, value)
		VALUES (:key, :value)
		ON CONFLICT(key) DO UPDATE SET value = :value`, namespace),
		sql.Named("key", key), sql.Named("value", encValue))
	if err != nil {
		return aerrors.NewRuntimeError("failed setting key", err, "")
	}

	return nil
}

func (s *Store) List(namespace, keyPrefix string) (map[string][]string, error) {
	// Validate the table name to ensure it actually exists. This prevents
	// possible SQL injection attacks, since we parametrize the table name below.
	allTables, err := queries.GetAllTables(s.NewContext(), s)
	if err != nil {
		return nil, err
	}

	keysPerNS := make(map[string][]string)

	listNamespace := func(ns string) error {
		rows, err := s.QueryContext(s.ctx, fmt.Sprintf(`SELECT key
			FROM "%s"
			ORDER BY key ASC`, ns))
		if err != nil {
			return err
		}

		for rows.Next() {
			var key string
			err = rows.Scan(&key)
			if nsKeys, ok := keysPerNS[ns]; !ok {
				keysPerNS[ns] = []string{key}
			} else {
				nsKeys = append(nsKeys, key)
			}
		}

		return nil
	}

	if namespace == "*" {
		for ns := range allTables {
			if err = listNamespace(ns); err != nil {
				return nil, err
			}
		}
	} else if _, ok := allTables[namespace]; !ok {
		return keysPerNS, nil
	} else if err = listNamespace(namespace); err != nil {
		return nil, err
	}

	return keysPerNS, nil
}

// Migrations returns all database migrations.
func (s *Store) Migrations() []*migrator.Migration {
	return s.migrations
}

// NewContext returns a new child context of the main database context.
func (s *Store) NewContext() context.Context {
	// TODO: Return cancel func?
	ctx, _ := context.WithCancel(s.ctx)
	return ctx
}
