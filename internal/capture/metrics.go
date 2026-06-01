package capture

import "sync/atomic"

var (
	captureTotal            atomic.Int64
	captureDroppedTotal     atomic.Int64
	captureWriteErrorsTotal atomic.Int64
	captureStoreErrorsTotal atomic.Int64
	captureExportSkipped    atomic.Int64
)

// IncrExportSkipped increments the export-skipped counter.
func IncrExportSkipped() { captureExportSkipped.Add(1) }

// Metrics returns a snapshot of capture subsystem counters.
func Metrics() map[string]int64 {
	return map[string]int64{
		"capture_total":              captureTotal.Load(),
		"capture_dropped_total":      captureDroppedTotal.Load(),
		"capture_write_errors_total": captureWriteErrorsTotal.Load(),
		"capture_store_errors_total": captureStoreErrorsTotal.Load(),
		"capture_export_skipped":     captureExportSkipped.Load(),
	}
}
