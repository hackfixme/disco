package queries

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"go.hackfix.me/disco/db/types"
)

func GetEncryptionKeyHash(ctx context.Context, d types.Querier) (sql.Null[string], error) {
	var keyHash sql.Null[string]
	err := d.QueryRowContext(ctx, `SELECT key_hash FROM _meta`).
		Scan(&keyHash)
	if err != nil {
		return keyHash, err
	}

	return keyHash, nil
}

func GetAllTables(ctx context.Context, d types.Querier) (map[string]struct{}, error) {
	allTables := make(map[string]struct{})
	rows, err := d.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, err
		}

		// Exclude internal tables
		if !strings.HasPrefix(name, "_") {
			allTables[name] = struct{}{}
		}
	}

	return allTables, nil
}

func Version(ctx context.Context, d types.Querier) (sql.Null[string], error) {
	var version sql.Null[string]
	err := d.QueryRowContext(ctx, `SELECT version FROM _meta`).
		Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return version, err
	}

	return version, nil
}
