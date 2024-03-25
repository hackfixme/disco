package models

import (
	"context"
	"fmt"

	"go.hackfix.me/disco/db/types"
)

func filterCount(ctx context.Context, d types.Querier, table string, filter *types.Filter) (int, error) {
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE %s`, table, filter.Where)
	var count int
	err := d.QueryRowContext(ctx, countQ, filter.Args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed scanning %s count query: %w", table, err)
	}

	return count, nil
}
