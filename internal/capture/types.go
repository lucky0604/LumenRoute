package capture

import (
	"encoding/json"
	"time"
)

// CaptureEntry is the in-flight representation passed from the proxy
// to the capture service via the buffered channel.
type CaptureEntry struct {
	RequestID         string
	ProjectID         int64
	RouteName         string
	PublicModelName   string
	UpstreamModelName string
	ProviderName      string
	Stream            bool
	StreamCompleted   bool
	StatusCode        int
	LatencyMs         int
	TTFCMs            *int
	PromptTokens      *int
	CompletionTokens  *int
	RequestBody       json.RawMessage
	ResponseBody      json.RawMessage
	BodySkipped       bool
}

// CaptureRecord is the metadata persisted in the SQLite index table.
type CaptureRecord struct {
	ID              int64     `json:"id"`
	RequestID       string    `json:"request_id"`
	ProjectID       int64     `json:"project_id"`
	PublicModelName string    `json:"public_model_name"`
	Stream          bool      `json:"stream"`
	StatusCode      int       `json:"status_code"`
	BodySkipped     bool      `json:"body_skipped"`
	FilePath        string    `json:"file_path"`
	FileOffset      int64     `json:"file_offset"`
	RequestSize     int       `json:"request_size"`
	ResponseSize    int       `json:"response_size"`
	CreatedAt       time.Time `json:"created_at"`
}

// jsonlLine is the on-disk JSONL representation.
type jsonlLine struct {
	RequestID         string          `json:"request_id"`
	ProjectID         int64           `json:"project_id"`
	RouteName         string          `json:"route_name"`
	PublicModelName   string          `json:"public_model_name"`
	UpstreamModelName string          `json:"upstream_model_name"`
	ProviderName      string          `json:"provider_name"`
	Stream            bool            `json:"stream"`
	StreamCompleted   *bool           `json:"stream_completed,omitempty"`
	StatusCode        int             `json:"status_code"`
	LatencyMs         int             `json:"latency_ms"`
	TTFCMs            *int            `json:"time_to_first_chunk_ms"`
	PromptTokens      *int            `json:"prompt_tokens"`
	CompletionTokens  *int            `json:"completion_tokens"`
	RequestBody       json.RawMessage `json:"request_body"`
	ResponseBody      json.RawMessage `json:"response_body"`
	CapturedAt        time.Time       `json:"captured_at"`
	SchemaVersion     int             `json:"schema_version"`
}

// CaptureFilter represents query parameters for listing/exporting captures.
type CaptureFilter struct {
	Since      *time.Time
	Until      *time.Time
	Stream     *bool
	Model      string
	StatusCode *int
	Cursor     int64
	PageSize   int
	Format     string
	Download   bool
}

// Config holds capture subsystem configuration.
type Config struct {
	Enabled     bool
	BasePath    string
	MaxBodySize int
	ChannelSize int
	BatchSize   int
}
