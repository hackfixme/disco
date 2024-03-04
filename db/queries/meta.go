package queries

import (
	"context"
	"database/sql"
	"errors"

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

func Version(ctx context.Context, d types.Querier) (sql.Null[string], error) {
	var version sql.Null[string]
	err := d.QueryRowContext(ctx, `SELECT version FROM _meta`).
		Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return version, err
	}

	return version, nil
}
