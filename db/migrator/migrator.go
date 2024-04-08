package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
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

// LoadMigrations reads SQL files from the embedded directory, and returns a
// slice of Migration sorted by migration name.
func LoadMigrations(dir fs.FS) ([]*Migration, error) {
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
		data, err := fs.ReadFile(dir, d.Name())
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

func loadHistory(d types.Querier, migrations []*Migration) error {
	migrationMap := make(map[string]*Migration)
	for _, m := range migrations {
		migrationMap[m.Name] = m
	}

	ctx, cancel := context.WithCancel(d.NewContext())
	defer cancel()

	rows, err := d.QueryContext(ctx, `SELECT name, type, time
		FROM _migration_history
		ORDER BY name, time;`)
	if err != nil {
		return fmt.Errorf("failed retrieving migration history: %w", err)
	}

	for rows.Next() {
		var (
			name, typ string
			time      sql.Null[time.Time]
		)
		err := rows.Scan(&name, &typ, &time)
		if err != nil {
			return fmt.Errorf("failed reading from database: %w", err)
		}

		migration, ok := migrationMap[name]
		if !ok {
			return fmt.Errorf("found unknown migration in history: '%s'", name)
		}
		evt := MigrationEvent{
			Type: MigrationType(typ),
			Time: time.V,
		}
		migration.History = append(migration.History, evt)
		if evt.Type == MigrationUp {
			migration.Applied = true
		} else {
			migration.Applied = false
		}
	}

	return nil
}

// RunMigrations applies or rolls back migrations.
// to can either be a migration name, or "all".
func RunMigrations(
	d types.Querier, migrations []*Migration, typ MigrationType, to string,
	logger *slog.Logger,
) error {
	ctx, cancel := context.WithCancel(d.NewContext())
	defer cancel()
	if err := createMigrationSchema(ctx, d); err != nil {
		return fmt.Errorf("failed creating migrations schema: %w", err)
	}

	loadHistory(d, migrations)

	runPlan, err := createMigrationPlan(migrations, typ, to)
	if err != nil {
		return err
	}

	// TODO: Ask user for confirmation before running the plan.

	if len(runPlan) == 0 {
		return nil
	}

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
			INSERT INTO _migration_history (name, type, time)
			VALUES ($1, $2, $3);
			`, run.name, string(run.typ), time.Now().UTC())
		if err != nil {
			return err
		}
		logger.Debug(fmt.Sprintf("%s DB migration", msg), "name", run.name)
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
		CREATE TABLE IF NOT EXISTS _migration_history (
			name   VARCHAR(128),
            type   VARCHAR(32) CHECK( type IN ('up','down') ) NOT NULL,
			time   TIMESTAMP NOT NULL
		);`)
	return err
}
