package proxy

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"lumenroute/internal/capture"
	"lumenroute/internal/db"
	"lumenroute/internal/project"
)

func TestProjCache_SetGetInvalidate(t *testing.T) {
	c := newProjCache(time.Minute)
	p := &project.Project{ID: 1, Name: "alpha", CaptureEnabled: true}

	c.set(1, p)
	got, ok := c.get(1)
	if !ok || got.Name != "alpha" {
		t.Fatalf("get(1) = (%v, %v), want alpha true", got, ok)
	}

	c.invalidate(1)
	if _, ok := c.get(1); ok {
		t.Error("get(1) after invalidate = hit, want miss")
	}
}

func TestProjCache_ExpiredEntry(t *testing.T) {
	c := newProjCache(10 * time.Millisecond)
	c.set(2, &project.Project{ID: 2, Name: "stale"})
	time.Sleep(15 * time.Millisecond)
	if _, ok := c.get(2); ok {
		t.Error("get(2) after TTL = hit, want miss")
	}
}

func TestShouldCapture(t *testing.T) {
	tests := []struct {
		name string
		p    project.Project
		want bool
	}{
		{"disabled", project.Project{CaptureEnabled: false, SampleRate: 1}, false},
		{"full sample", project.Project{CaptureEnabled: true, SampleRate: 1}, true},
		{"over one", project.Project{CaptureEnabled: true, SampleRate: 1.5}, true},
		{"zero rate", project.Project{CaptureEnabled: true, SampleRate: 0}, false},
		{"negative rate", project.Project{CaptureEnabled: true, SampleRate: -0.1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldCapture(&tt.p); got != tt.want {
				t.Errorf("shouldCapture() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidateProjectCache(t *testing.T) {
	s := NewService(nil, nil, nil, nil, ServiceConfig{})
	s.cache.set(9, &project.Project{ID: 9, Name: "cached"})
	s.InvalidateProjectCache(9)
	if _, ok := s.cache.get(9); ok {
		t.Error("cache still has project 9 after InvalidateProjectCache")
	}
}

func setupProxyCapture(t *testing.T) (*Service, *capture.Service, *capture.Store, *sql.DB) {
	t.Helper()
	dsn := "file:" + t.TempDir() + "/proxy_cap.db"
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	_, err = database.Exec(`
		INSERT INTO projects (id, name, description, data_category, capture_enabled, sample_rate)
		VALUES (1, 'cap-proj', '', 'mixed', 1, 1.0)
	`)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	store := capture.NewStore(database)
	capSvc := capture.NewService(capture.Config{
		BasePath:    filepath.Join(t.TempDir(), "captures"),
		BatchSize:   1,
		ChannelSize: 8,
	}, store)
	capSvc.Start()
	t.Cleanup(func() { capSvc.Close() })

	proxy := NewService(nil, nil, nil, nil, ServiceConfig{
		CaptureEnabled:     true,
		CaptureMaxBodySize: 50,
	})
	proxy.SetCaptureService(capSvc, project.NewService(database))
	return proxy, capSvc, store, database
}

func TestMaybeCapture_NoOpWhenDisabled(t *testing.T) {
	s := NewService(nil, nil, nil, nil, ServiceConfig{CaptureEnabled: false})
	before := capture.Metrics()["capture_total"]
	s.maybeCapture(capture.CaptureEntry{ProjectID: 1, RequestID: "noop"})
	after := capture.Metrics()["capture_total"]
	if after != before {
		t.Errorf("capture_total changed when capture disabled")
	}
}

func TestMaybeCapture_SkipsZeroProject(t *testing.T) {
	proxy, capSvc, _, _ := setupProxyCapture(t)
	before := capture.Metrics()["capture_total"]
	proxy.maybeCapture(capture.CaptureEntry{ProjectID: 0, RequestID: "zero"})
	if capSvc.QueueLength() != 0 {
		t.Error("maybeCapture with ProjectID 0 should not enqueue")
	}
	after := capture.Metrics()["capture_total"]
	if after != before {
		t.Error("capture_total changed for zero project")
	}
}

func TestMaybeCapture_SubmitsWhenEligible(t *testing.T) {
	proxy, capSvc, store, _ := setupProxyCapture(t)
	before := capture.Metrics()["capture_total"]
	proxy.maybeCapture(capture.CaptureEntry{
		RequestID:    "cap-eligible",
		ProjectID:    1,
		StatusCode:   200,
		RequestBody:  json.RawMessage(`{"x":1}`),
		ResponseBody: json.RawMessage(`{"y":2}`),
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if capSvc.QueueLength() == 0 {
			count, _ := store.CountByProject(1)
			if count >= 1 {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	after := capture.Metrics()["capture_total"]
	if after-before < 1 {
		t.Errorf("capture_total delta = %d, want >= 1", after-before)
	}
}

func TestMaybeCapture_StripsOversizedBodies(t *testing.T) {
	proxy, capSvc, store, _ := setupProxyCapture(t)
	large := json.RawMessage(`{"payload":"` + string(make([]byte, 80)) + `"}`)
	proxy.maybeCapture(capture.CaptureEntry{
		RequestID:    "cap-big",
		ProjectID:    1,
		RequestBody:  large,
		ResponseBody: large,
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		records, _ := store.ListAfterCursor(1, 0, 5, capture.CaptureFilter{})
		for _, r := range records {
			if r.BodySkipped {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("oversized capture not marked BodySkipped")
	_ = capSvc
}

func TestMaybeCapture_SkipsWhenProjectCaptureDisabled(t *testing.T) {
	proxy, capSvc, _, database := setupProxyCapture(t)
	if _, err := database.Exec(`UPDATE projects SET capture_enabled = 0 WHERE id = 1`); err != nil {
		t.Fatalf("update project: %v", err)
	}
	proxy.cache.invalidate(1)

	before := capture.Metrics()["capture_total"]
	proxy.maybeCapture(capture.CaptureEntry{ProjectID: 1, RequestID: "disabled"})
	if capSvc.QueueLength() != 0 {
		t.Error("maybeCapture enqueued when project capture disabled")
	}
	after := capture.Metrics()["capture_total"]
	if after != before {
		t.Error("capture_total changed when project capture disabled")
	}
}

func TestGetProjectCached_UsesCache(t *testing.T) {
	proxy, _, _, _ := setupProxyCapture(t)
	proxy.cache.set(1, &project.Project{ID: 1, Name: "from-cache", CaptureEnabled: true, SampleRate: 1})

	// DB has name cap-proj; cache should win.
	p := proxy.getProjectCached(1)
	if p == nil || p.Name != "from-cache" {
		t.Fatalf("getProjectCached() = %+v, want from-cache", p)
	}
}
