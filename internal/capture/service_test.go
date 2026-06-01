package capture

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"lumenroute/internal/db"
)

func TestService_SubmitAndProcess(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()
	ensureTestProject(t, database, 1)

	basePath := filepath.Join(t.TempDir(), "captures")
	store := NewStore(database)
	svc := NewService(Config{BasePath: basePath, BatchSize: 1, ChannelSize: 4}, store)
	svc.Start()
	defer svc.Close()

	before := Metrics()["capture_total"]
	svc.Submit(CaptureEntry{
		RequestID:       "cap-1",
		ProjectID:       1,
		PublicModelName: "test-model",
		StatusCode:      200,
		RequestBody:     json.RawMessage(`{"prompt":"hi"}`),
		ResponseBody:    json.RawMessage(`{"ok":true}`),
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		count, _ := store.CountByProject(1)
		if count >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	count, err := store.CountByProject(1)
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if count != 1 {
		t.Errorf("stored count = %d, want 1", count)
	}

	after := Metrics()["capture_total"]
	if after-before != 1 {
		t.Errorf("capture_total delta = %d, want 1", after-before)
	}
}

func TestService_SubmitDropsWhenFull(t *testing.T) {
	store := setupCaptureStore(t)
	svc := NewService(Config{
		BasePath:    t.TempDir(),
		ChannelSize: 1,
		BatchSize:   100,
	}, store)

	// Fill channel without starting processor.
	svc.Submit(CaptureEntry{RequestID: "block"})
	beforeDrop := Metrics()["capture_dropped_total"]
	svc.Submit(CaptureEntry{RequestID: "drop-me"})
	afterDrop := Metrics()["capture_dropped_total"]

	if afterDrop-beforeDrop != 1 {
		t.Errorf("capture_dropped_total delta = %d, want 1", afterDrop-beforeDrop)
	}
	if svc.Dropped() != 1 {
		t.Errorf("Dropped() = %d, want 1", svc.Dropped())
	}
	if svc.QueueLength() != 1 {
		t.Errorf("QueueLength() = %d, want 1", svc.QueueLength())
	}
}

func TestService_BodySkippedRecord(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()
	ensureTestProject(t, database, 7)

	store := NewStore(database)
	svc := NewService(Config{BasePath: t.TempDir(), BatchSize: 1, ChannelSize: 2}, store)
	svc.Start()
	defer svc.Close()

	svc.Submit(CaptureEntry{
		RequestID:   "skip-1",
		ProjectID:   7,
		StatusCode:  413,
		BodySkipped: true,
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		records, _ := store.ListAfterCursor(7, 0, 1, CaptureFilter{})
		if len(records) == 1 && records[0].BodySkipped {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("body-skipped record not persisted in time")
}
