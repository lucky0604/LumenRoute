package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lumenroute/internal/route"
)

func testRouteTarget() (*route.Route, *route.RouteTarget) {
	tid := int64(20)
	pid := int64(30)
	return &route.Route{
			ID: 1, Name: "main", PublicModelName: "gpt-test",
		}, &route.RouteTarget{
			ID: tid, RouteID: 1, ProviderID: pid,
			UpstreamModelName: "upstream-gpt", ProviderName: "openai",
		}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*http.Request)
		want   string
	}{
		{
			"xff first",
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", " 1.2.3.4 , 5.6.7.8")
			},
			"1.2.3.4",
		},
		{
			"x-real-ip",
			func(r *http.Request) {
				r.Header.Set("X-Real-IP", "9.9.9.9")
			},
			"9.9.9.9",
		},
		{
			"remote addr port stripped",
			func(r *http.Request) {
				r.RemoteAddr = "192.168.0.5:12345"
			},
			"192.168.0.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			tt.setup(req)
			if got := extractClientIP(req); got != tt.want {
				t.Errorf("extractClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildRequestLog_NonStreamWithTokens(t *testing.T) {
	rt, target := testRouteTarget()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.RemoteAddr = "10.0.0.1:8080"

	body := []byte(`{"usage":{"prompt_tokens":3,"completion_tokens":7,"total_tokens":10}}`)
	entry := buildRequestLog(logParams{
		RequestID:          "req-abc",
		Request:            req,
		Route:              rt,
		Target:             target,
		UpstreamStatusCode: 200,
		LatencyMs:          42,
		ResponseBody:       body,
	})

	if entry.RequestID != "req-abc" {
		t.Errorf("RequestID = %q, want req-abc", entry.RequestID)
	}
	if entry.RouteName != "main" || entry.PublicModelName != "gpt-test" {
		t.Errorf("route fields = (%q, %q)", entry.RouteName, entry.PublicModelName)
	}
	if entry.ClientIP != "10.0.0.1" {
		t.Errorf("ClientIP = %q, want 10.0.0.1", entry.ClientIP)
	}
	if entry.PromptTokens == nil || *entry.PromptTokens != 3 {
		t.Errorf("PromptTokens = %v, want 3", entry.PromptTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 10 {
		t.Errorf("TotalTokens = %v, want 10", entry.TotalTokens)
	}
}

func TestBuildRequestLog_StreamResult(t *testing.T) {
	rt, target := testRouteTarget()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	completed := true
	sr := &StreamResult{
		TimeToFirstChunkMs: 15,
		Completed:          completed,
		PromptTokens:       5,
		CompletionTokens:   12,
		TotalTokens:        17,
		ErrorCode:          "upstream_eof",
		ErrorMessage:       "truncated",
	}

	entry := buildRequestLog(logParams{
		Request:            req,
		Route:              rt,
		Target:             target,
		UpstreamStatusCode: 200,
		LatencyMs:          100,
		Stream:             true,
		StreamResult:       sr,
	})

	if entry.Stream != true {
		t.Error("Stream = false, want true")
	}
	if entry.TimeToFirstChunkMs == nil || *entry.TimeToFirstChunkMs != 15 {
		t.Errorf("TimeToFirstChunkMs = %v, want 15", entry.TimeToFirstChunkMs)
	}
	if entry.StreamCompleted == nil || !*entry.StreamCompleted {
		t.Errorf("StreamCompleted = %v, want true", entry.StreamCompleted)
	}
	if entry.ErrorCode != "upstream_eof" {
		t.Errorf("ErrorCode = %q, want upstream_eof", entry.ErrorCode)
	}
	if entry.CompletionTokens == nil || *entry.CompletionTokens != 12 {
		t.Errorf("CompletionTokens = %v, want 12", entry.CompletionTokens)
	}
}

func TestBuildRequestLog_GeneratesRequestID(t *testing.T) {
	rt, target := testRouteTarget()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	entry := buildRequestLog(logParams{
		Request: req, Route: rt, Target: target,
		UpstreamStatusCode: 200, LatencyMs: 1,
	})
	if entry.RequestID == "" {
		t.Error("RequestID empty when not provided")
	}
}
