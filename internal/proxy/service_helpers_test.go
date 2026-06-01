package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lumenroute/internal/db"
	"lumenroute/internal/logs"
	"lumenroute/internal/metrics"
)

func TestOpenAIChatCompletionsURL(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"https://api.example.com", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1/", "https://api.example.com/v1/chat/completions"},
	}
	for _, tt := range tests {
		if got := openAIChatCompletionsURL(tt.base); got != tt.want {
			t.Errorf("openAIChatCompletionsURL(%q) = %q, want %q", tt.base, got, tt.want)
		}
	}
}

func TestWriteLog_PersistsEntry(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/writelog.db"
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	logSvc := logs.NewService(database)
	s := NewService(nil, nil, logSvc, metrics.NewRegistry(), ServiceConfig{})

	rt, target := testRouteTarget()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	s.writeLog(logParams{
		RequestID:          "wl-1",
		Request:            req,
		Route:              rt,
		Target:             target,
		UpstreamStatusCode: 200,
		LatencyMs:          25,
		ResponseBody:       []byte(`{"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`),
	})

	var rid string
	err = database.QueryRow(`SELECT request_id FROM request_logs WHERE request_id = 'wl-1'`).Scan(&rid)
	if err != nil {
		t.Fatalf("writeLog did not persist request log: %v", err)
	}
}

func TestCodeToType(t *testing.T) {
	if codeToType("invalid_api_key") != "authentication_error" {
		t.Error("invalid_api_key type mismatch")
	}
	if codeToType("unknown_code") != "upstream_error" {
		t.Error("default type mismatch")
	}
}
