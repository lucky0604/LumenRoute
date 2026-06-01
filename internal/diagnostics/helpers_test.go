package diagnostics

import (
	"database/sql"
	"testing"
	"time"

	"lumenroute/internal/db"
	"lumenroute/internal/logs"
)

func openHelperDB(t *testing.T) (*sql.DB, *logs.Service) {
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
	return database, logs.NewService(database)
}

func seedLatencies(t *testing.T, logsSvc *logs.Service, targetID int64, model string, latencies []int) {
	t.Helper()
	tid := targetID
	rid := int64(1)
	pid := int64(5)
	for i, lat := range latencies {
		entry := logs.RequestLog{
			RequestID:         "lat-" + string(rune('a'+i)),
			RouteID:           &rid,
			RouteName:         "test-route",
			PublicModelName:   model,
			UpstreamModelName: "upstream",
			ProviderID:        &pid,
			ProviderName:      "prov",
			TargetID:          &tid,
			StatusCode:        200,
			UpstreamStatusCode: 200,
			LatencyMs:         lat,
		}
		if err := logsSvc.Write(entry); err != nil {
			t.Fatalf("seed write: %v", err)
		}
	}
}

func TestComputePercentile_ByTarget(t *testing.T) {
	database, logsSvc := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	seedLatencies(t, logsSvc, 42, "model-a", []int{100, 200, 300, 400, 500})

	p95 := computePercentile(database, 42, "", since, 95)
	// sorted: 100,200,300,400,500; idx = 5*95/100 = 4 -> 500
	if p95 < 499 || p95 > 501 {
		t.Errorf("computePercentile(95) = %f, want ~500", p95)
	}
}

func TestComputePercentile_ByModel(t *testing.T) {
	database, logsSvc := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	seedLatencies(t, logsSvc, 0, "qwen-fast", []int{10, 20, 30})

	v := computePercentile(database, 0, "qwen-fast", since, 50)
	// idx = 3*50/100 = 1 -> 20
	if v < 19 || v > 21 {
		t.Errorf("computePercentile(50) by model = %f, want ~20", v)
	}
}

func TestComputePercentile_NoFilter(t *testing.T) {
	database, _ := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)

	v := computePercentile(database, 0, "", since, 95)
	if v != 0 {
		t.Errorf("computePercentile(no filter) = %f, want 0", v)
	}
}

func TestComputePercentile_ZeroLatencyExcluded(t *testing.T) {
	database, logsSvc := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	seedLatencies(t, logsSvc, 7, "m", []int{0, 100})

	v := computeP95(database, 7, "", since)
	if v < 99 || v > 101 {
		t.Errorf("computeP95 with zero latency = %f, want ~100", v)
	}
}

func TestComputeP99(t *testing.T) {
	database, logsSvc := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	seedLatencies(t, logsSvc, 99, "m", []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100})

	v := computeP99(database, 99, "", since)
	// idx = 10*99/100 = 9 -> 100
	if v < 99 || v > 101 {
		t.Errorf("computeP99 = %f, want ~100", v)
	}
}

func TestComputePercentile_SingleValue(t *testing.T) {
	database, logsSvc := openHelperDB(t)
	since := time.Now().UTC().Add(-1 * time.Hour)
	seedLatencies(t, logsSvc, 1, "solo", []int{777})

	v := computeP95(database, 1, "", since)
	if v < 776 || v > 778 {
		t.Errorf("computeP95(single) = %f, want ~777", v)
	}
}
