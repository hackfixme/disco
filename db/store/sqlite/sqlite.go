package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"regexp"

	aerrors "go.hackfix.me/disco/app/errors"
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
// The returned boolean indicates whether the value was found or not.
func (s *Store) Get(namespace, key string) (ok bool, value io.Reader, err error) {
	// Validate the table name to ensure it actually exists. This prevents
	// possible SQL injection attacks, since we parametrize the table name below.
	allTables, err := queries.GetAllTables(s.NewContext(), s)
	if err != nil {
		return false, nil, err
	}
	if _, ok := allTables[namespace]; !ok {
		return false, nil, nil
	}

	// Namespaces are stored in different tables, but parameterization is not
	// supported for table names, so template it manually.
	var encValue []byte
	err = s.QueryRowContext(s.ctx, fmt.Sprintf(`SELECT value
		FROM "%s"
		WHERE key = ?`, namespace), key).Scan(&encValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, err
	}

	decValue, err := crypto.DecryptSym(bytes.NewReader(encValue), s.encKey)
	if err != nil {
		return true, nil, aerrors.NewRuntimeError("failed decrypting value", err, "")
	}

	return true, decValue, nil
}

func (s *Store) Set(namespace, key string, value io.Reader) error {
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

	encData, err := crypto.EncryptSym(value, s.encKey)
	if err != nil {
		return aerrors.NewRuntimeError("failed encrypting value", err, "")
	}

	encValue, err := io.ReadAll(encData)
	if err != nil {
		return aerrors.NewRuntimeError("failed reading encrypted data", err, "")
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

// Delete a key within a specific namespace. An error is returned if the key
// doesn't exist.
func (s *Store) Delete(namespace, key string) error {
	// Validate the table name to ensure it actually exists. This prevents
	// possible SQL injection attacks, since we parametrize the table name below.
	allTables, err := queries.GetAllTables(s.NewContext(), s)
	if err != nil {
		return err
	}
	if _, ok := allTables[namespace]; !ok {
		return fmt.Errorf("namespace doesn't exist: %s", namespace)
	}

	// Namespaces are stored in different tables, but parameterization is not
	// supported for table names, so template it manually.
	res, err := s.ExecContext(s.ctx,
		fmt.Sprintf(`DELETE FROM "%s" WHERE key = ?`, namespace), key)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("key doesn't exist: %s", key)
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

	filter := types.NewFilter("1=1", []any{})
	if keyPrefix != "" {
		filter = types.NewFilter("key LIKE ? || '%'", []any{keyPrefix})
	}

	listNamespace := func(ns string) error {
		query := fmt.Sprintf(
			`SELECT key
			FROM "%s"
			WHERE %s
			ORDER BY key ASC`, ns, filter.Where)
		rows, err := s.QueryContext(s.ctx, query, filter.Args...)
		if err != nil {
			return err
		}

		for rows.Next() {
			var key string
			err = rows.Scan(&key)
			if err != nil {
				return err
			}
			if nsKeys, ok := keysPerNS[ns]; !ok {
				keysPerNS[ns] = []string{key}
			} else {
				nsKeys = append(nsKeys, key)
				keysPerNS[ns] = nsKeys
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

// NewContext returns a new child context of the main database context.
func (s *Store) NewContext() context.Context {
	// TODO: Return cancel func?
	ctx, _ := context.WithCancel(s.ctx)
	return ctx
}

// Init creates the database schema and initial records.
func (s *Store) Init(appVersion string, logger *slog.Logger) error {
	err := migrator.RunMigrations(s, s.migrations, migrator.MigrationUp, "all", logger)
	if err != nil {
		return err
	}

	dbCtx := s.NewContext()
	_, err = s.ExecContext(dbCtx,
		`INSERT INTO _meta (version) VALUES (?)`, appVersion)
	if err != nil {
		return err
	}

	return nil
}
