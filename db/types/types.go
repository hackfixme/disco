package types

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
)

// Querier exposes only methods for running SQL queries, and some helper functions.
type Querier interface {
	NewContext() context.Context
	ExecContext(ctx context.Context, sql string, arguments ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Filter is used to dynamically modify queries.
type Filter struct {
	Where string
	Args  []any
}

func NewFilter(where string, args []any) *Filter {
	return &Filter{Where: where, Args: args}
}

func (f1 *Filter) And(f2 *Filter) *Filter {
	return &Filter{
		Where: fmt.Sprintf("%s AND %s", f1.Where, f2.Where),
		Args:  slices.Concat(f1.Args, f2.Args),
	}
}

func (f1 *Filter) Or(f2 *Filter) *Filter {
	return &Filter{
		Where: fmt.Sprintf("%s OR %s", f1.Where, f2.Where),
		Args:  slices.Concat(f1.Args, f2.Args),
	}
}
