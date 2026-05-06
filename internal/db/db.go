package db

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		dsn = "file:data/lumenroute.db?_foreign_keys=on&_journal_mode=WAL"
	}
	if err := os.MkdirAll("data", 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	return db, nil
}

func RunMigrations(dsn string) error {
	db, err := Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %d: %w", m.version, err)
		}
		if count > 0 {
			continue
		}
		for _, stmt := range strings.Split(m.up, ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
			}
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.version, m.name); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	return nil
}

type migration struct {
	version int
	name    string
	up      string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var ms []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		var version int
		fmt.Sscanf(e.Name(), "%d_", &version)
		name := e.Name()
		data, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		ms = append(ms, migration{version: version, name: name, up: string(data)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	return ms, nil
}
