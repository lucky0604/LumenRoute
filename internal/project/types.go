package project

import "time"

type Project struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	DataCategory    string     `json:"data_category"`
	CaptureEnabled  bool       `json:"capture_enabled"`
	SampleRate      float64    `json:"sample_rate"`
	RetentionDays   int        `json:"retention_days"`
	ExportTokenHash string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

var ValidDataCategories = map[string]bool{
	"qa":     true,
	"rag":    true,
	"mixed":  true,
	"custom": true,
}

type Stats struct {
	ProjectID              int64      `json:"project_id"`
	TotalCaptures          int64      `json:"total_captures"`
	TotalRequestSizeBytes  int64      `json:"total_request_size_bytes"`
	TotalResponseSizeBytes int64      `json:"total_response_size_bytes"`
	EarliestCapture        *time.Time `json:"earliest_capture"`
	LatestCapture          *time.Time `json:"latest_capture"`
	CapturesToday          int64      `json:"captures_today"`
	RoutesCount            int64      `json:"routes_count"`
}
