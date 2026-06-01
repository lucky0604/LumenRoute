package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	"lumenroute/internal/capture"
)

func TestRegistry_Handler_Output(t *testing.T) {
	reg := NewRegistry()
	reg.IncRequests()
	reg.IncErrors()
	reg.AddTokens(42)
	reg.IncActiveStream()
	reg.SetProviderHealthCounts(3, 1)
	capture.IncrExportSkipped()

	rr := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))

	body := rr.Body.String()
	if ct := rr.Header().Get("Content-Type"); ct != "text/plain; version=0.0.4" {
		t.Errorf("Content-Type = %q, want text/plain; version=0.0.4", ct)
	}

	checks := map[string]string{
		"lumenroute_requests_total 1":              "requests counter",
		"lumenroute_request_errors_total 1":        "errors counter",
		"lumenroute_request_tokens_total 42":       "tokens counter",
		"lumenroute_active_streams 1":              "active streams gauge",
		"lumenroute_provider_healthy 3":            "healthy providers",
		"lumenroute_provider_unhealthy 1":          "unhealthy providers",
		"lumenroute_capture_export_skipped":        "capture export skipped",
		"# TYPE lumenroute_requests_total counter": "requests type",
		"# HELP lumenroute_capture_total":          "capture help",
	}
	for needle, label := range checks {
		if !strings.Contains(body, needle) {
			t.Errorf("metrics body missing %s (%q)", label, needle)
		}
	}
}

func TestRecordProxyRequest(t *testing.T) {
	reg := NewRegistry()
	reg.RecordProxyRequest(ProxyRequest{StatusCode: 500, IsError: true, Tokens: 100})
	reg.RecordProxyRequest(ProxyRequest{StatusCode: 200, IsError: false, Tokens: 0})

	rr := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()

	if !strings.Contains(body, "lumenroute_requests_total 2") {
		t.Error("expected requests_total 2")
	}
	if !strings.Contains(body, "lumenroute_request_errors_total 1") {
		t.Error("expected request_errors_total 1")
	}
	if !strings.Contains(body, "lumenroute_request_tokens_total 100") {
		t.Error("expected request_tokens_total 100")
	}
}

func TestRegistry_StreamGauge(t *testing.T) {
	reg := NewRegistry()
	reg.IncActiveStream()
	reg.IncActiveStream()
	reg.DecActiveStream()

	rr := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(rr.Body.String(), "lumenroute_active_streams 1") {
		t.Error("expected active_streams 1 after inc/inc/dec")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{12345, "12345"},
		{-99, "-99"},
	}
	for _, tt := range tests {
		if got := itoa(tt.in); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
