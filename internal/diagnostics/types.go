package diagnostics

type ModelSummary struct {
	PublicModelName     string  `json:"public_model_name"`
	UpstreamModelName   string  `json:"upstream_model_name"`
	ProviderName        string  `json:"provider_name"`
	TargetID            int64   `json:"target_id"`
	RequestCount        int     `json:"request_count"`
	ErrorCount          int     `json:"error_count"`
	ErrorRate           float64 `json:"error_rate"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	P95LatencyMs        float64 `json:"p95_latency_ms"`
	TotalTokens         int     `json:"total_tokens"`
	StreamCount         int     `json:"stream_count"`
	StreamCompletedRate float64 `json:"stream_completed_rate"`
	LastErrorCode       string  `json:"last_error_code"`
	LastSeenAt          string  `json:"last_seen_at"`
}

type TargetFacts struct {
	ID                int64  `json:"id"`
	RouteID           int64  `json:"route_id"`
	RouteName         string `json:"route_name"`
	PublicModelName   string `json:"public_model_name"`
	UpstreamModelName string `json:"upstream_model_name"`
	ProviderName      string `json:"provider_name"`
	ProviderBaseURL   string `json:"provider_base_url"`
	ProviderEngine    string `json:"provider_engine"`
	ProviderHealth    string `json:"provider_health"`
	LastCheckAt       string `json:"last_check_at"`
	LastError         string `json:"last_provider_error"`
	Enabled           bool   `json:"enabled"`
}

type TargetDiagnosis struct {
	Target           TargetFacts       `json:"target"`
	Summary          ModelSummary      `json:"summary"`
	RecentFailures   []RequestLogBrief `json:"recent_failures"`
	SlowRequests     []RequestLogBrief `json:"slow_requests"`
	OperatorCommands OperatorCommands  `json:"operator_commands"`
}

type RequestLogBrief struct {
	ID                 int64  `json:"id"`
	StatusCode         int    `json:"status_code"`
	UpstreamStatusCode int    `json:"upstream_status_code"`
	ErrorCode          string `json:"error_code"`
	ErrorMessage       string `json:"error_message"`
	LatencyMs          int    `json:"latency_ms"`
	StreamCompleted    *bool  `json:"stream_completed"`
	PublicModelName    string `json:"public_model_name"`
	CreatedAt          string `json:"created_at"`
}

type OperatorCommands struct {
	ModelsCurl string `json:"models_curl"`
}
