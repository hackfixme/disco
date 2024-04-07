package queries

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
)

func GetEncryptionPrivKeyHash(ctx context.Context, d types.Querier) (sql.Null[string], error) {
	var keyHash sql.Null[string]
	err := d.QueryRowContext(ctx,
		`SELECT private_key_hash FROM users WHERE type = ?`,
		models.UserTypeLocal).Scan(&keyHash)
	if err != nil {
		return keyHash, err
	}

	return keyHash, nil
}

func GetEncryptionPubKey(ctx context.Context, d types.Querier) (sql.Null[string], error) {
	var pubKey sql.Null[string]
	err := d.QueryRowContext(ctx,
		`SELECT public_key FROM users WHERE type = ?`,
		models.UserTypeLocal).Scan(&pubKey)
	if err != nil {
		return pubKey, err
	}

	return pubKey, nil
}

func GetServerTLSCert(ctx context.Context, d types.Querier) (
	cert sql.Null[string], privKeyEnc sql.Null[[]byte], err error,
) {
	err = d.QueryRowContext(ctx,
		`SELECT server_tls_cert, server_tls_key_enc FROM _meta`).
		Scan(&cert, &privKeyEnc)
	if err != nil {
		return
	}

	if !cert.Valid {
		return cert, privKeyEnc, errors.New("server TLS certificate not found")
	}
	if !privKeyEnc.Valid {
		return cert, privKeyEnc, errors.New("server TLS private key not found")
	}

	return
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
