package logs

import (
	"testing"

	"lumenroute/internal/db"
)

func TestWriteAndList(t *testing.T) {
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

	if err := svc.Write(RequestLog{
		RequestID:         "req-123",
		PublicModelName:   "test-model",
		ProviderName:      "test-prov",
		StatusCode:        200,
		LatencyMs:         150,
		Stream:            false,
		UpstreamStatusCode: 200,
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	logs, err := svc.List(LogFilter{Model: "test-model"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("list len = %d, want 1", len(logs))
	}

	log, err := svc.Get(logs[0].ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if log.RequestID != "req-123" {
		t.Errorf("request_id = %s", log.RequestID)
	}
}

func TestLogFilters(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()

	svc := NewService(database)
	svc.Write(RequestLog{RequestID: "r1", StatusCode: 200, ProviderName: "p1"})
	svc.Write(RequestLog{RequestID: "r2", StatusCode: 500, ProviderName: "p2", ErrorCode: "timeout"})
	svc.Write(RequestLog{RequestID: "r3", StatusCode: 200, Stream: true})

	errorOnly := true
	logs, _ := svc.List(LogFilter{ErrorOnly: true})
	if len(logs) != 1 {
		t.Errorf("error filter len = %d, want 1", len(logs))
	}

	tstream := true
	logs, _ = svc.List(LogFilter{Stream: &tstream})
	if len(logs) != 1 {
		t.Errorf("stream filter len = %d, want 1", len(logs))
	}

	logs, _ = svc.List(LogFilter{Provider: "p2"})
	if len(logs) != 1 {
		t.Errorf("provider filter len = %d, want 1", len(logs))
	}
	_ = errorOnly
}

func TestGenerateRequestID(t *testing.T) {
	id := GenerateRequestID("client-123")
	if id != "client-123" {
		t.Errorf("should reuse client id, got %s", id)
	}
	id2 := GenerateRequestID("")
	if id2 == "" || len(id2) < 8 {
		t.Errorf("generated id too short: %s", id2)
	}
}
