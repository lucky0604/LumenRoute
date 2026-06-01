package db

import "testing"

func TestRunMigrations(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	db, err := Open(dsn)
	if err != nil {
		t.Fatalf("Open after migration: %v", err)
	}
	defer db.Close()

	tables := []string{"users", "api_keys", "providers", "routes", "route_targets", "request_logs", "schema_migrations", "projects", "request_captures"}
	for _, name := range tables {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count); err != nil {
			t.Fatalf("check table %s: %v", name, err)
		}
		if count != 1 {
			t.Errorf("table %s not found", name)
		}
	}

	// Verify foreign_keys is ON
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	// Verify WAL mode
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("pragma journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %s, want wal", journalMode)
	}

	// Verify request log indexes exist
	indexes := []string{
		"idx_request_logs_request_id",
		"idx_request_logs_created_at",
		"idx_request_logs_route_created",
		"idx_request_logs_provider_created",
		"idx_request_logs_status_created",
	}
	for _, idx := range indexes {
		var icount int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&icount); err != nil {
			t.Fatalf("check index %s: %v", idx, err)
		}
		if icount != 1 {
			t.Errorf("index %s not found", idx)
		}
	}
}
