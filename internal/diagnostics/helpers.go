package diagnostics

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

func computePercentile(db *sql.DB, targetID int64, publicModel string, since time.Time, pct int) float64 {
	var rows *sql.Rows
	var err error
	sinceStr := since.Format("2006-01-02 15:04:05")

	if targetID > 0 {
		rows, err = db.Query(`
			SELECT latency_ms FROM request_logs
			WHERE target_id = ? AND created_at >= ? AND latency_ms > 0
			ORDER BY latency_ms ASC LIMIT 10000
		`, targetID, sinceStr)
	} else if publicModel != "" {
		rows, err = db.Query(`
			SELECT latency_ms FROM request_logs
			WHERE public_model_name = ? AND created_at >= ? AND latency_ms > 0
			ORDER BY latency_ms ASC LIMIT 10000
		`, publicModel, sinceStr)
	} else {
		return 0
	}
	if err != nil {
		return 0
	}
	defer rows.Close()

	var values []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err == nil {
			values = append(values, v)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	idx := len(values) * pct / 100
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return float64(values[idx])
}

func computeP95(db *sql.DB, targetID int64, publicModel string, since time.Time) float64 {
	return computePercentile(db, targetID, publicModel, since, 95)
}

func computeP99(db *sql.DB, targetID int64, publicModel string, since time.Time) float64 {
	return computePercentile(db, targetID, publicModel, since, 99)
}

func parseWindow(window string) (time.Time, error) {
	var d time.Duration
	switch window {
	case "5m":
		d = 5 * time.Minute
	case "1h":
		d = 1 * time.Hour
	case "24h":
		d = 24 * time.Hour
	default:
		return time.Time{}, fmt.Errorf("invalid window %q: must be 5m, 1h, or 24h", window)
	}
	return time.Now().UTC().Add(-d), nil
}
