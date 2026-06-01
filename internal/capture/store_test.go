package capture

import (
	"database/sql"
	"testing"
	"time"

	"lumenroute/internal/db"
)

func setupCaptureStore(t *testing.T) *Store {
	t.Helper()
	dsn := "file:" + t.TempDir() + "/test.db"
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	for _, id := range []int64{1, 2, 3, 5, 7, 9} {
		ensureTestProject(t, database, id)
	}
	return NewStore(database)
}

func ensureTestProject(t *testing.T, database *sql.DB, id int64) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO projects (id, name, description, data_category, capture_enabled)
		VALUES (?, ?, '', 'mixed', 1)
	`, id, "test-project-"+itoaProject(id))
	if err != nil {
		t.Fatalf("insert project %d: %v", id, err)
	}
}

func itoaProject(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestStore_InsertBatch_Empty(t *testing.T) {
	store := setupCaptureStore(t)
	if err := store.InsertBatch(nil); err != nil {
		t.Errorf("InsertBatch(nil) = %v, want nil", err)
	}
}

func TestStore_InsertBatch_AndList(t *testing.T) {
	store := setupCaptureStore(t)
	records := []CaptureRecord{
		{
			RequestID: "req-1", ProjectID: 1, PublicModelName: "gpt",
			Stream: false, StatusCode: 200, FilePath: "/tmp/a.jsonl",
			FileOffset: 0, RequestSize: 10, ResponseSize: 20,
		},
		{
			RequestID: "req-2", ProjectID: 1, PublicModelName: "gpt",
			Stream: true, StatusCode: 200, BodySkipped: true,
		},
	}
	if err := store.InsertBatch(records); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}

	got, err := store.ListAfterCursor(1, 0, 10, CaptureFilter{})
	if err != nil {
		t.Fatalf("ListAfterCursor: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(got))
	}
	if got[0].RequestID != "req-1" || got[1].RequestID != "req-2" {
		t.Errorf("request IDs = %q, %q", got[0].RequestID, got[1].RequestID)
	}
	if !got[1].BodySkipped {
		t.Error("second record BodySkipped = false, want true")
	}
}

func TestStore_CountByProject(t *testing.T) {
	store := setupCaptureStore(t)
	store.InsertBatch([]CaptureRecord{
		{RequestID: "a", ProjectID: 5, PublicModelName: "m", StatusCode: 200},
		{RequestID: "b", ProjectID: 5, PublicModelName: "m", StatusCode: 200},
		{RequestID: "c", ProjectID: 9, PublicModelName: "m", StatusCode: 200},
	})

	count, err := store.CountByProject(5)
	if err != nil {
		t.Fatalf("CountByProject: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestStore_ListAfterCursor_Filters(t *testing.T) {
	store := setupCaptureStore(t)
	streamTrue := true
	status200 := 200
	store.InsertBatch([]CaptureRecord{
		{RequestID: "a", ProjectID: 2, PublicModelName: "m1", Stream: true, StatusCode: 200},
		{RequestID: "b", ProjectID: 2, PublicModelName: "m2", Stream: false, StatusCode: 500},
	})

	got, err := store.ListAfterCursor(2, 0, 10, CaptureFilter{
		Stream:     &streamTrue,
		Model:      "m1",
		StatusCode: &status200,
	})
	if err != nil {
		t.Fatalf("ListAfterCursor: %v", err)
	}
	if len(got) != 1 || got[0].RequestID != "a" {
		t.Errorf("filtered records = %+v, want [a]", got)
	}
}

func TestStore_DeleteByProjectBefore(t *testing.T) {
	store := setupCaptureStore(t)
	store.InsertBatch([]CaptureRecord{
		{RequestID: "old", ProjectID: 3, PublicModelName: "m", StatusCode: 200, FilePath: "/data/old.jsonl"},
	})

	before := time.Now().Add(1 * time.Hour)
	paths, deleted, err := store.DeleteByProjectBefore(3, before)
	if err != nil {
		t.Fatalf("DeleteByProjectBefore: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if len(paths) != 1 || paths[0] != "/data/old.jsonl" {
		t.Errorf("paths = %v, want [/data/old.jsonl]", paths)
	}

	count, _ := store.CountByProject(3)
	if count != 0 {
		t.Errorf("count after delete = %d, want 0", count)
	}
}
