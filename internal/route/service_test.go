package route

import (
	"testing"

	"lumenroute/internal/db"
)

func TestRouteTargetCRUD(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	svc := NewService(database)

	// Need a provider first
	database.Exec(`INSERT INTO providers (name, base_url, engine, enabled, health_status, created_at, updated_at)
		VALUES ('test-prov', 'http://10.0.0.1:4000/v1', 'vllm', 1, 'healthy', datetime('now'), datetime('now'))`)

	// Create route
	rid, err := svc.CreateRoute(Route{Name: "test-route", PublicModelName: "test-model", Enabled: true})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}

	// Create target
	tid, err := svc.CreateTarget(RouteTarget{RouteID: rid, ProviderID: 1, UpstreamModelName: "test-model", Weight: 100, Enabled: true})
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	if tid == 0 { t.Error("no target id") }

	// List targets
	targets, err := svc.ListTargets(rid)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	if len(targets) != 1 { t.Errorf("targets len = %d", len(targets)) }

	// Select target
	selected, err := svc.SelectTarget(rid)
	if err != nil {
		t.Fatalf("select target: %v", err)
	}
	if selected.ID != tid { t.Errorf("selected id = %d, want %d", selected.ID, tid) }

	// Delete route
	if err := svc.DeleteRoute(rid); err != nil {
		t.Fatalf("delete route: %v", err)
	}
}

func TestSelectTargetNoReady(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()

	svc := NewService(database)
	// Route with no targets
	rid, _ := svc.CreateRoute(Route{Name: "empty", PublicModelName: "empty-model", Enabled: true})

	_, err := svc.SelectTarget(rid)
	if err == nil {
		t.Error("expected error selecting target with no ready targets")
	}
}
