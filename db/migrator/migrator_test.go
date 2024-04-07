package migrator

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMigrationPlan(t *testing.T) {
	t.Parallel()

	up := sql.Null[string]{V: "up", Valid: true}
	down := sql.Null[string]{V: "down", Valid: true}

	testCases := []struct {
		migrations []*Migration
		typ        MigrationType
		to         string
		expected   []migrationRun
		expErr     string
	}{
		// Up
		{
			migrations: []*Migration{
				{Name: "a", Applied: false, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "all",
			expected: []migrationRun{
				{name: "a", typ: MigrationUp, sql: up.V},
				{name: "b", typ: MigrationUp, sql: up.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "all",
			expected: []migrationRun{
				{name: "b", typ: MigrationUp, sql: up.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: false, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "a",
			expected: []migrationRun{
				{name: "a", typ: MigrationUp, sql: up.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: false, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "b",
			expected: []migrationRun{
				{name: "a", typ: MigrationUp, sql: up.V},
				{name: "b", typ: MigrationUp, sql: up.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "a",
			expected: []migrationRun{},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
				{Name: "c", Applied: false, Up: up, Down: down},
			}, typ: MigrationUp, to: "c",
			expected: []migrationRun{
				{name: "c", typ: MigrationUp, sql: up.V},
			},
		},
		// Down
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
			}, typ: MigrationDown, to: "all",
			expected: []migrationRun{
				{name: "a", typ: MigrationDown, sql: down.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
			}, typ: MigrationDown, to: "all",
			expected: []migrationRun{
				{name: "b", typ: MigrationDown, sql: down.V},
				{name: "a", typ: MigrationDown, sql: down.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
			}, typ: MigrationDown, to: "a",
			expected: []migrationRun{
				{name: "b", typ: MigrationDown, sql: down.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
				{Name: "c", Applied: true, Up: up, Down: down},
			}, typ: MigrationDown, to: "a",
			expected: []migrationRun{
				{name: "c", typ: MigrationDown, sql: down.V},
				{name: "b", typ: MigrationDown, sql: down.V},
			},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
				{Name: "c", Applied: false, Up: up, Down: down},
			}, typ: MigrationDown, to: "c",
			expected: []migrationRun{},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: true, Up: up, Down: down},
				{Name: "c", Applied: false, Up: up, Down: down},
			}, typ: MigrationDown, to: "b",
			expected: []migrationRun{},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
				{Name: "b", Applied: false, Up: up, Down: down},
				{Name: "c", Applied: false, Up: up, Down: down},
			}, typ: MigrationDown, to: "a",
			expected: []migrationRun{},
		},
		{
			migrations: []*Migration{
				{Name: "a", Applied: true, Up: up, Down: down},
			}, typ: MigrationDown, to: "none",
			expErr: "migration 'none' doesn't exist",
		},
		{
			migrations: []*Migration{},
			typ:        MigrationDown, to: "none",
			expErr: "migration 'none' doesn't exist",
		},
	}

	for idx, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			runPlan, err := createMigrationPlan(tc.migrations, tc.typ, tc.to)
			if tc.expErr != "" {
				require.EqualError(t, err, tc.expErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, runPlan)
		})
	}
}
