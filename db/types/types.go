package types

import (
	"context"
	"database/sql"
)

// Querier exposes only methods for running SQL queries, and some helper functions.
type Querier interface {
	NewContext() context.Context
	ExecContext(ctx context.Context, sql string, arguments ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
