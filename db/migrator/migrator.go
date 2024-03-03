package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"go.hackfix.me/disco/db/types"
)

// MigrationType is the type of the migration.
type MigrationType string

// Migration types.
const (
	MigrationUp   MigrationType = "up"
	MigrationDown MigrationType = "down"
)

// MigrationEvent is a record of a migration being applied or rolled back.
type MigrationEvent struct {
	Type MigrationType
	Time time.Time
}

// Migration is a database migration.
type Migration struct {
	ID      int64
	Name    string
	Applied bool
	History []MigrationEvent
	Up      sql.Null[string]
	Down    sql.Null[string]
}

// Save the migration to the database.
func (m *Migration) Save(ctx context.Context, d types.Querier) error {
	var id int64
	err := d.QueryRowContext(ctx, `
        INSERT INTO _migrations (name, up, down)
        VALUES ($1, $2, $3);
		`, m.Name, m.Up.V, m.Down.V).Scan(&id)
	if err != nil {
		return err
	}

	m.ID = id

	return nil
}

// LoadMigrations reads SQL files from the embedded directory, and returns a
// slice of Migration sorted by migration name.
func LoadMigrations(dir fs.ReadDirFS) ([]*Migration, error) {
	fnameRx := regexp.MustCompile(`^(?P<name>\d{1,}-[a-z0-9-_]+)\.(?P<type>up|down)\.sql$`)
	migrationMap := make(map[string]*Migration)

	fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.Type().IsRegular() || filepath.Ext(d.Name()) != ".sql" {
			return nil
		}

		matched := fnameRx.FindStringSubmatch(d.Name())
		if len(matched) == 0 {
			// TODO: Log warning
			return nil
		}
		data, err := os.ReadFile(d.Name())
		if err != nil {
			return err
		}
		name := matched[fnameRx.SubexpIndex("name")]
		typ := matched[fnameRx.SubexpIndex("type")]
		m, ok := migrationMap[name]
		if !ok {
			m = &Migration{Name: name}
			migrationMap[name] = m
		}
		val := sql.Null[string]{V: string(data), Valid: true}
		if typ == "up" {
			m.Up = val
		} else {
			m.Down = val
		}

		return nil
	})

	mKeys := make([]string, 0, len(migrationMap))
	for mName := range migrationMap {
		mKeys = append(mKeys, mName)
	}
	sort.Strings(mKeys)
	migrations := make([]*Migration, 0, len(migrationMap))
	for _, mName := range mKeys {
		migrations = append(migrations, migrationMap[mName])
	}

	return migrations, nil
}

// GetMigrations returns all database migrations.
func GetMigrations(d types.Querier) ([]*Migration, error) {
	ctx, cancel := context.WithCancel(d.NewContext())
	defer cancel()
	if err := createMigrationSchema(ctx, d); err != nil {
		return nil, fmt.Errorf("failed creating migrations schema: %w", err)
	}

	rows, err := d.QueryContext(ctx, `
		SELECT m.id, m.name, m.up, m.down, mh.type, mh.time
		FROM _migrations AS m
		LEFT JOIN _migration_history AS mh ON m.id = mh.migration_id
		ORDER BY m.name, mh.time;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed retrieving migrations: %w", err)
	}

	migrations := []*Migration{}
	type mRow struct {
		ID       int64
		Name     string
		Up, Down sql.Null[string]
		Type     sql.Null[string]
		Time     sql.Null[time.Time]
	}

	var migration *Migration
	for rows.Next() {
		row := mRow{}
		err := rows.Scan(&row.ID, &row.Name, &row.Up, &row.Down, &row.Type, &row.Time)
		if err != nil {
			return nil, fmt.Errorf("failed reading from database: %w", err)
		}
		if migration == nil || migration.Name != row.Name {
			migration = &Migration{
				ID:   row.ID,
				Name: row.Name,
				Up:   row.Up,
				Down: row.Down,
			}
			migrations = append(migrations, migration)
		}

		if row.Type.V != "" {
			evt := MigrationEvent{
				Type: MigrationType(row.Type.V),
				Time: row.Time.V,
			}
			migration.History = append(migration.History, evt)
			if evt.Type == MigrationUp {
				migration.Applied = true
			} else {
				migration.Applied = false
			}
		}
	}

	return migrations, nil
}

// RunMigrations applies or rolls back migrations.
func RunMigrations(d types.Querier, typ MigrationType, to string) error {
	migrations, err := GetMigrations(d)
	if err != nil {
		return err
	}

	runPlan, err := createMigrationPlan(migrations, typ, to)
	if err != nil {
		return err
	}

	// TODO: Ask user for confirmation before running the plan.

	if len(runPlan) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(d.NewContext())
	defer cancel()

	for _, run := range runPlan {
		_, err := d.ExecContext(ctx, run.sql)
		if err != nil {
			return err
		}
		msg := "applied"
		if run.typ == MigrationDown {
			msg = "rolled back"
		}
		_, err = d.ExecContext(ctx, `
			INSERT INTO _migration_history (migration_id, type, time)
			VALUES ($1, $2, $3);
			`, run.id, string(run.typ), time.Now().UTC())
		if err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("%s DB migration", msg), "name", run.name)
	}

	return nil
}

type migrationRun struct {
	id   int64
	name string
	typ  MigrationType
	sql  string
}

func createMigrationPlan(
	migrations []*Migration, typ MigrationType, to string,
) ([]migrationRun, error) {
	runPlan := []migrationRun{}

	toIdx := -1
	for i, m := range migrations {
		if m.Name == to {
			toIdx = i
			break
		}
	}

	if toIdx < 0 && to != "all" {
		return nil, fmt.Errorf("migration '%s' doesn't exist", to)
	}

	for idx, m := range migrations {
		if !m.Applied && typ == MigrationUp && (toIdx >= idx || to == "all") {
			runPlan = append(runPlan, migrationRun{
				id:   m.ID,
				name: m.Name,
				typ:  MigrationUp,
				sql:  m.Up.V,
			})
		} else if m.Applied && typ == MigrationDown && (toIdx < idx || to == "all") {
			// Run the down migration in reverse order
			runPlan = append([]migrationRun{{
				id:   m.ID,
				name: m.Name,
				typ:  MigrationDown,
				sql:  m.Down.V,
			}}, runPlan...)
		}
	}

	return runPlan, nil
}

func createMigrationSchema(ctx context.Context, q types.Querier) error {
	_, err := q.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS _migrations (
			  id      INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY
			, name    VARCHAR(128) UNIQUE
            , up      TEXT
            , down    TEXT
		);
		-- Postgres doesn't support CREATE TYPE IF NOT EXISTS,
		-- or CREATE OR REPLACE TYPE.
		DO $$ BEGIN
			CREATE TYPE _migration_type AS ENUM ('up', 'down');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
		CREATE TABLE IF NOT EXISTS _migration_history (
            migration_id   INTEGER NOT NULL REFERENCES _migrations,
            type           _migration_type NOT NULL,
			time           timestamp NOT NULL
		);`)
	return err
}
