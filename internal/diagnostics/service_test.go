package diagnostics

import (
	"database/sql"
	"testing"
	"time"

	"lumenroute/internal/logs"
	"lumenroute/internal/provider"
	"lumenroute/internal/route"
)

func seedOverviewLogs(t *testing.T, logsSvc *logs.Service) {
	t.Helper()
	tid := int64(42)
	rid := int64(1)
	pid := int64(5)
	entries := []logs.RequestLog{
		{
			RequestID: "ok-1", RouteID: &rid, RouteName: "r1",
			PublicModelName: "model-a", UpstreamModelName: "up-a",
			ProviderID: &pid, ProviderName: "prov", TargetID: &tid,
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 100,
			PromptTokens: intPtr(10), CompletionTokens: intPtr(5),
			Stream: true, StreamCompleted: boolPtr(true),
		},
		{
			RequestID: "err-1", RouteID: &rid, RouteName: "r1",
			PublicModelName: "model-a", UpstreamModelName: "up-a",
			ProviderID: &pid, ProviderName: "prov", TargetID: &tid,
			StatusCode: 502, UpstreamStatusCode: 502, LatencyMs: 500,
			ErrorCode: "upstream_error", ErrorMessage: "timeout",
			Stream: false,
		},
		{
			RequestID: "ok-2", RouteID: &rid, RouteName: "r1",
			PublicModelName: "model-b", UpstreamModelName: "up-b",
			ProviderID: &pid, ProviderName: "prov", TargetID: int64Ptr(99),
			StatusCode: 200, UpstreamStatusCode: 200, LatencyMs: 50,
		},
	}
	for _, e := range entries {
		if err := logsSvc.Write(e); err != nil {
			t.Fatalf("write log: %v", err)
		}
	}
}

func intPtr(v int) *int       { return &v }
func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }

func TestGetModelOverview_InvalidWindow(t *testing.T) {
	database, _, _ := openDiagDB(t)
	svc := NewService(database, route.NewService(database), provider.NewService(database))

	_, err := svc.GetModelOverview("7d")
	if err == nil {
		t.Fatal("GetModelOverview(7d) err = nil, want error")
	}
}

func TestGetModelOverview_Empty(t *testing.T) {
	database, _, _ := openDiagDB(t)
	svc := NewService(database, route.NewService(database), provider.NewService(database))

	summaries, err := svc.GetModelOverview("1h")
	if err != nil {
		t.Fatalf("GetModelOverview: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("len(summaries) = %d, want 0", len(summaries))
	}
}

func TestGetModelOverview_Aggregates(t *testing.T) {
	database, logsSvc, _ := openDiagDB(t)
	seedOverviewLogs(t, logsSvc)
	svc := NewService(database, route.NewService(database), provider.NewService(database))

	summaries, err := svc.GetModelOverview("24h")
	if err != nil {
		t.Fatalf("GetModelOverview: %v", err)
	}
	if len(summaries) < 2 {
		t.Fatalf("len(summaries) = %d, want at least 2 groups", len(summaries))
	}

	var modelA *ModelSummary
	for i := range summaries {
		if summaries[i].PublicModelName == "model-a" {
			modelA = &summaries[i]
			break
		}
	}
	if modelA == nil {
		t.Fatal("model-a group missing")
	}
	if modelA.RequestCount != 2 {
		t.Errorf("model-a RequestCount = %d, want 2", modelA.RequestCount)
	}
	if modelA.ErrorCount != 1 {
		t.Errorf("model-a ErrorCount = %d, want 1", modelA.ErrorCount)
	}
	if modelA.ErrorRate < 0.49 || modelA.ErrorRate > 0.51 {
		t.Errorf("model-a ErrorRate = %f, want ~0.5", modelA.ErrorRate)
	}
	if modelA.TotalTokens != 15 {
		t.Errorf("model-a TotalTokens = %d, want 15", modelA.TotalTokens)
	}
	if modelA.StreamCount != 1 {
		t.Errorf("model-a StreamCount = %d, want 1", modelA.StreamCount)
	}
	if modelA.StreamCompletedRate < 0.99 {
		t.Errorf("model-a StreamCompletedRate = %f, want 1", modelA.StreamCompletedRate)
	}
	if modelA.LastErrorCode != "upstream_error" {
		t.Errorf("model-a LastErrorCode = %q, want upstream_error", modelA.LastErrorCode)
	}
}

func TestQueryTargetSummary(t *testing.T) {
	database, logsSvc, _ := openDiagDB(t)
	seedOverviewLogs(t, logsSvc)
	svc := NewService(database, route.NewService(database), provider.NewService(database))
	since := time.Now().UTC().Add(-1 * time.Hour)

	ms := svc.queryTargetSummary(42, since)
	if ms.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2 for target 42", ms.RequestCount)
	}
	if ms.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", ms.ErrorCount)
	}
	if ms.AvgLatencyMs < 200 || ms.AvgLatencyMs > 350 {
		t.Errorf("AvgLatencyMs = %f, want ~300", ms.AvgLatencyMs)
	}
	if ms.P95LatencyMs < 99 {
		t.Errorf("P95LatencyMs = %f, want >= 100", ms.P95LatencyMs)
	}
	if ms.LastErrorCode != "upstream_error" {
		t.Errorf("LastErrorCode = %q, want upstream_error", ms.LastErrorCode)
	}
}

func TestQueryTargetSummary_NoRows(t *testing.T) {
	database, _, _ := openDiagDB(t)
	svc := NewService(database, route.NewService(database), provider.NewService(database))
	since := time.Now().UTC().Add(-1 * time.Hour)

	ms := svc.queryTargetSummary(9999, since)
	if ms.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0 for unknown target", ms.RequestCount)
	}
}

func TestQueryLastErrorCode_ByPublicModel(t *testing.T) {
	database, logsSvc, _ := openDiagDB(t)
	tid := int64(0)
	rid := int64(1)
	entry := logs.RequestLog{
		RequestID: "model-err", RouteID: &rid, RouteName: "r1",
		PublicModelName: "orphan-model", UpstreamModelName: "up",
		TargetID: &tid, StatusCode: 500, UpstreamStatusCode: 500,
		LatencyMs: 10, ErrorCode: "rate_limited",
	}
	if err := logsSvc.Write(entry); err != nil {
		t.Fatalf("write: %v", err)
	}
	since := time.Now().UTC().Add(-1 * time.Hour)
	code := queryLastErrorCode(database, 0, "orphan-model", since)
	if code != "rate_limited" {
		t.Errorf("queryLastErrorCode = %q, want rate_limited", code)
	}
}

func openDiagDB(t *testing.T) (*sql.DB, *logs.Service, *provider.Service) {
	t.Helper()
	database, logsSvc := openHelperDB(t)
	return database, logsSvc, provider.NewService(database)
}
