package diagnostics

import (
	"testing"
	"time"

	"lumenroute/internal/db"
	"lumenroute/internal/logs"
	"lumenroute/internal/provider"
	"lumenroute/internal/route"
)

func setupTestDB(t *testing.T) (*Service, *logs.Service) {
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

	logsSvc := logs.NewService(database)
	routeSvc := route.NewService(database)
	provSvc := provider.NewService(database)
	diagSvc := NewService(database, routeSvc, provSvc)
	return diagSvc, logsSvc
}

func seedTestData(t *testing.T, logsSvc *logs.Service) {
	t.Helper()
	tid := int64(10)
	rid := int64(1)
	pid := int64(5)
	streamTrue := true
	streamFalse := false
	ttfc := 200
	promptT := 10
	compT := 20
	totalT := 30

	entries := []logs.RequestLog{
		{
			RequestID: "r1", RouteID: &rid, RouteName: "test-route",
			PublicModelName: "qwen-fast", UpstreamModelName: "Qwen3.5-27B",
			ProviderID: &pid, ProviderName: "prov-a", TargetID: &tid,
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 150,
			Stream: false, PromptTokens: &promptT, CompletionTokens: &compT, TotalTokens: &totalT,
		},
		{
			RequestID: "r2", RouteID: &rid, RouteName: "test-route",
			PublicModelName: "qwen-fast", UpstreamModelName: "Qwen3.5-27B",
			ProviderID: &pid, ProviderName: "prov-a", TargetID: &tid,
			StatusCode: 500, UpstreamStatusCode: 500, LatencyMs: 300,
			ErrorCode: "upstream_status_500", ErrorMessage: "internal error",
		},
		{
			RequestID: "r3", RouteID: &rid, RouteName: "test-route",
			PublicModelName: "qwen-fast", UpstreamModelName: "Qwen3.5-27B",
			ProviderID: &pid, ProviderName: "prov-a", TargetID: &tid,
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 5000,
			Stream: true, StreamCompleted: &streamTrue, TimeToFirstChunkMs: &ttfc,
		},
		{
			RequestID: "r4", RouteID: &rid, RouteName: "test-route",
			PublicModelName: "qwen-fast", UpstreamModelName: "Qwen3.5-27B",
			ProviderID: &pid, ProviderName: "prov-a", TargetID: &tid,
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 8000,
			Stream: true, StreamCompleted: &streamFalse,
		},
		{
			RequestID: "r5", RouteID: &rid, RouteName: "test-route",
			PublicModelName: "qwen-fast", UpstreamModelName: "Qwen3.5-27B",
			ProviderID: &pid, ProviderName: "prov-a", TargetID: &tid,
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 200,
		},
	}
	for _, e := range entries {
		if err := logsSvc.Write(e); err != nil {
			t.Fatalf("seed write: %v", err)
		}
	}
}

func TestGetModelOverview_EmptyWindow(t *testing.T) {
	svc, _ := setupTestDB(t)

	summaries, err := svc.GetModelOverview("1h")
	if err != nil {
		t.Fatalf("GetModelOverview: %v", err)
	}
	if summaries == nil {
		t.Fatal("summaries is nil, want empty slice")
	}
	if len(summaries) != 0 {
		t.Errorf("len(summaries) = %d, want 0", len(summaries))
	}
}

func TestGetModelOverview_GroupsCorrectly(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, err := svc.GetModelOverview("1h")
	if err != nil {
		t.Fatalf("GetModelOverview: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}

	ms := summaries[0]
	if ms.PublicModelName != "qwen-fast" {
		t.Errorf("PublicModelName = %q, want qwen-fast", ms.PublicModelName)
	}
	if ms.RequestCount != 5 {
		t.Errorf("RequestCount = %d, want 5", ms.RequestCount)
	}
	if ms.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", ms.ErrorCount)
	}
}

func TestGetModelOverview_ErrorRate(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	expectedRate := float64(1) / float64(5)
	if ms.ErrorRate < expectedRate-0.001 || ms.ErrorRate > expectedRate+0.001 {
		t.Errorf("ErrorRate = %f, want ~%f", ms.ErrorRate, expectedRate)
	}
}

func TestGetModelOverview_AvgLatency(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	// 150 + 300 + 5000 + 8000 + 200 = 13650, avg = 2730
	if ms.AvgLatencyMs < 2700 || ms.AvgLatencyMs > 2800 {
		t.Errorf("AvgLatencyMs = %f, want ~2730", ms.AvgLatencyMs)
	}
}

func TestGetModelOverview_StreamMetrics(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	if ms.StreamCount != 2 {
		t.Errorf("StreamCount = %d, want 2", ms.StreamCount)
	}
	expectedRate := 0.5
	if ms.StreamCompletedRate < expectedRate-0.01 || ms.StreamCompletedRate > expectedRate+0.01 {
		t.Errorf("StreamCompletedRate = %f, want ~0.5", ms.StreamCompletedRate)
	}
}

func TestGetModelOverview_P95(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	// latencies sorted: 150, 200, 300, 5000, 8000; idx = 5*95/100 = 4 -> 8000
	if ms.P95LatencyMs < 7999 || ms.P95LatencyMs > 8001 {
		t.Errorf("P95LatencyMs = %f, want ~8000", ms.P95LatencyMs)
	}
}

func TestGetModelOverview_TotalTokens(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	if ms.TotalTokens < 30 {
		t.Errorf("TotalTokens = %d, want >= 30 (prompt+completion for r1)", ms.TotalTokens)
	}
}

func TestGetModelOverview_LastErrorCode(t *testing.T) {
	svc, logsSvc := setupTestDB(t)
	seedTestData(t, logsSvc)

	summaries, _ := svc.GetModelOverview("1h")
	ms := summaries[0]

	if ms.LastErrorCode != "upstream_status_500" {
		t.Errorf("LastErrorCode = %q, want upstream_status_500", ms.LastErrorCode)
	}
}

func TestGetModelOverview_InvalidWindow(t *testing.T) {
	svc, _ := setupTestDB(t)

	_, err := svc.GetModelOverview("banana")
	if err == nil {
		t.Error("expected error for invalid window, got nil")
	}
}

func TestGetTargetDiagnosis_MissingTarget(t *testing.T) {
	svc, _ := setupTestDB(t)

	_, err := svc.GetTargetDiagnosis(9999, "1h")
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestParseWindow(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"bad", 0, true},
	}
	for _, tt := range tests {
		since, err := parseWindow(tt.input)
		if tt.err && err == nil {
			t.Errorf("parseWindow(%q) expected error", tt.input)
			continue
		}
		if !tt.err && err != nil {
			t.Errorf("parseWindow(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !tt.err {
			actual := time.Since(since)
			if actual < tt.want-time.Second || actual > tt.want+time.Second {
				t.Errorf("parseWindow(%q) = since=%v, want ~%v ago", tt.input, since, tt.want)
			}
		}
	}
}

func TestComputeP95_Empty(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"
	db.RunMigrations(dsn)
	database, _ := db.Open(dsn)
	defer database.Close()

	v := computeP95(database, 999, "", time.Now().Add(-1*time.Hour))
	if v != 0 {
		t.Errorf("computeP95(empty) = %f, want 0", v)
	}
}
