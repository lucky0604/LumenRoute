package capture

import "testing"

func TestMetrics_InitialKeys(t *testing.T) {
	m := Metrics()
	keys := []string{
		"capture_total",
		"capture_dropped_total",
		"capture_write_errors_total",
		"capture_store_errors_total",
		"capture_export_skipped",
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("Metrics() missing key %q", k)
		}
	}
}

func TestIncrExportSkipped(t *testing.T) {
	before := Metrics()["capture_export_skipped"]
	IncrExportSkipped()
	IncrExportSkipped()
	after := Metrics()["capture_export_skipped"]
	if after-before != 2 {
		t.Errorf("capture_export_skipped delta = %d, want 2", after-before)
	}
}
