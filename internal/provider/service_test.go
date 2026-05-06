package provider

import (
	"testing"

	"lumenroute/internal/db"
)

func TestProviderCRUD(t *testing.T) {
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

	id, err := svc.Create(Provider{Name: "test-provider", BaseURL: "http://10.0.0.1:4000/v1", Engine: "vllm", ProviderType: "openai_compatible", Enabled: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 { t.Error("no id returned") }

	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 { t.Errorf("list len = %d, want 1", len(list)) }

	p, err := svc.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.Name != "test-provider" { t.Errorf("name = %s", p.Name) }

	if err := svc.Update(id, Provider{Name: "updated", BaseURL: "http://10.0.0.2:4000/v1", Engine: "sglang", Enabled: true}); err != nil {
		t.Fatalf("update: %v", err)
	}

	p2, _ := svc.Get(id)
	if p2.Name != "updated" { t.Errorf("update name = %s", p2.Name) }

	if err := svc.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	list2, _ := svc.List()
	if len(list2) != 0 { t.Errorf("list after delete len = %d, want 0", len(list2)) }
}

func TestProviderHealthUpdate(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	if err := db.RunMigrations(dsn); err != nil { t.Fatalf("migrate: %v", err) }
	database, err := db.Open(dsn)
	if err != nil { t.Fatalf("open: %v", err) }
	defer database.Close()

	svc := NewService(database)
	id, err := svc.Create(Provider{Name: "health-test-" + t.TempDir(), BaseURL: "http://10.0.0.1:4000/v1", Engine: "vllm", Enabled: true})
	if err != nil { t.Fatalf("create: %v", err) }

	if err := svc.UpdateHealth(id, "healthy", 200, 15, ""); err != nil {
		t.Fatalf("update health: %v", err)
	}

	p, err := svc.Get(id)
	if err != nil { t.Fatalf("get: %v", err) }
	if p.HealthStatus != "healthy" { t.Errorf("status = %s", p.HealthStatus) }
	if *p.LastStatusCode != 200 { t.Errorf("status code = %d", *p.LastStatusCode) }
}
