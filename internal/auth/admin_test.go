package auth

import (
	"os"
	"testing"

	"lumenroute/internal/db"
)

func TestBootstrapAdminWithEnvPassword(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := BootstrapAdmin(database, "admin", "secret123"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	var hash string
	if err := database.QueryRow("SELECT password_hash FROM users WHERE username = ?", "admin").Scan(&hash); err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if !CheckPassword(hash, "secret123") {
		t.Error("password hash does not match env password")
	}
}

func TestBootstrapAdminGeneratedPassword(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + dir + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := BootstrapAdmin(database, "admin", ""); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	b, err := os.ReadFile("data/bootstrap-admin-password")
	if err != nil {
		t.Fatalf("read bootstrap file: %v", err)
	}
	generated := string(b)
	if len(generated) < 32 {
		t.Fatalf("generated password too short: %d chars", len(generated))
	}

	var hash string
	if err := database.QueryRow("SELECT password_hash FROM users WHERE username = ?", "admin").Scan(&hash); err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if !CheckPassword(hash, generated[:len(generated)-1]) {
		t.Error("password hash does not match generated password")
	}
}

func TestBootstrapAdminIdempotent(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := BootstrapAdmin(database, "admin", "first"); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if err := BootstrapAdmin(database, "admin", "second"); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}
