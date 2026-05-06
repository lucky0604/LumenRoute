package apikey

import (
	"testing"

	"lumenroute/internal/db"
)

func TestAPIKeyCRUD(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	svc := NewService(database, "llmcp_")

	key, err := svc.Create("test-key", "desc", `{"type":"all"}`, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if key.RawKey == "" {
		t.Error("raw key not returned")
	}
	if len(key.RawKey) < 38 {
		t.Errorf("raw key too short: %s (%d)", key.RawKey, len(key.RawKey))
	}

	keys, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("list len = %d", len(keys))
	}

	validated, err := svc.ValidateKey(key.RawKey)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validated.ID != key.ID {
		t.Errorf("validated id mismatch")
	}

	if err := svc.Disable(key.ID); err != nil {
		t.Fatalf("disable: %v", err)
	}
	_, err = svc.ValidateKey(key.RawKey)
	if err == nil {
		t.Error("expected error validating disabled key")
	}

	if err := svc.Enable(key.ID); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := svc.Delete(key.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	keys2, _ := svc.List()
	if len(keys2) != 0 {
		t.Errorf("list after delete len = %d", len(keys2))
	}
}

func TestInvalidKey(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()

	svc := NewService(database, "llmcp_")
	_, err := svc.ValidateKey("nonsense_key")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestIsModelAllowed(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()

	svc := NewService(database, "llmcp_")

	tests := []struct {
		policy   string
		routeID  int64
		expected bool
	}{
		{`{"type":"all"}`, 1, true},
		{`{"type":"selected","route_ids":[1,2]}`, 1, true},
		{`{"type":"selected","route_ids":[2,3]}`, 1, false},
		{`{"type":"selected","route_ids":[]}`, 1, false},
	}
	for _, tc := range tests {
		key := &APIKey{AllowedRouteIDs: tc.policy}
		if got := svc.IsModelAllowed(key, "model", tc.routeID); got != tc.expected {
			t.Errorf("policy %s route %d: got %v, want %v", tc.policy, tc.routeID, got, tc.expected)
		}
	}
}
