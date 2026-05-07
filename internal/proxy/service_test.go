package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"lumenroute/internal/route"
)

func testRoute() *route.Route {
	return &route.Route{
		ID:              1,
		Name:            "test-route",
		PublicModelName: "test-model",
	}
}

func testTarget() *route.RouteTarget {
	return &route.RouteTarget{
		ID:                10,
		RouteID:           1,
		ProviderID:        5,
		ProviderName:      "test-provider",
		UpstreamModelName: "upstream-model",
		ProviderBaseURL:   "http://localhost:8080",
	}
}

func testRequest() *http.Request {
	return httptest.NewRequest("POST", "/v1/chat/completions", nil)
}

func TestBuildRequestLog_NonStreamSuccess(t *testing.T) {
	respBody := []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30},"choices":[]}`)
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 200,
		LatencyMs:          150,
		Stream:             false,
		ResponseBody:       respBody,
	})

	if entry.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", entry.StatusCode)
	}
	if entry.UpstreamStatusCode != 200 {
		t.Errorf("UpstreamStatusCode = %d, want 200", entry.UpstreamStatusCode)
	}
	if entry.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d, want 150", entry.LatencyMs)
	}
	if entry.ErrorCode != "" {
		t.Errorf("ErrorCode = %q, want empty", entry.ErrorCode)
	}
	if entry.PromptTokens == nil || *entry.PromptTokens != 10 {
		t.Errorf("PromptTokens = %v, want 10", entry.PromptTokens)
	}
	if entry.CompletionTokens == nil || *entry.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %v, want 20", entry.CompletionTokens)
	}
	if entry.TotalTokens == nil || *entry.TotalTokens != 30 {
		t.Errorf("TotalTokens = %v, want 30", entry.TotalTokens)
	}
	if entry.Stream != false {
		t.Errorf("Stream = %v, want false", entry.Stream)
	}
}

func TestBuildRequestLog_NonStreamUpstream500(t *testing.T) {
	respBody := []byte(`{"error":{"message":"internal error"}}`)
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 500,
		LatencyMs:          200,
		Stream:             false,
		ResponseBody:       respBody,
	})

	if entry.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", entry.StatusCode)
	}
	if entry.UpstreamStatusCode != 500 {
		t.Errorf("UpstreamStatusCode = %d, want 500", entry.UpstreamStatusCode)
	}
}

func TestBuildRequestLog_ConnectionFailure(t *testing.T) {
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 502,
		LatencyMs:          5,
		ErrorCode:          "upstream_connection_failed",
		ErrorMessage:       "dial tcp: connection refused",
	})

	if entry.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", entry.StatusCode)
	}
	if entry.ErrorCode != "upstream_connection_failed" {
		t.Errorf("ErrorCode = %q, want upstream_connection_failed", entry.ErrorCode)
	}
	if entry.ErrorMessage != "dial tcp: connection refused" {
		t.Errorf("ErrorMessage = %q", entry.ErrorMessage)
	}
}

func TestBuildRequestLog_StreamCompleted(t *testing.T) {
	sr := &StreamResult{
		LatencyMs:          3000,
		TimeToFirstChunkMs: 200,
		Completed:          true,
		UpstreamStatusCode: 200,
		PromptTokens:       15,
		CompletionTokens:   25,
		TotalTokens:        40,
	}
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 200,
		LatencyMs:          3000,
		Stream:             true,
		StreamResult:       sr,
	})

	if entry.Stream != true {
		t.Errorf("Stream = %v, want true", entry.Stream)
	}
	if entry.StreamCompleted == nil || *entry.StreamCompleted != true {
		t.Errorf("StreamCompleted = %v, want true", entry.StreamCompleted)
	}
	if entry.TimeToFirstChunkMs == nil || *entry.TimeToFirstChunkMs != 200 {
		t.Errorf("TimeToFirstChunkMs = %v, want 200", entry.TimeToFirstChunkMs)
	}
	if entry.PromptTokens == nil || *entry.PromptTokens != 15 {
		t.Errorf("PromptTokens = %v, want 15", entry.PromptTokens)
	}
	if entry.ErrorCode != "" {
		t.Errorf("ErrorCode = %q, want empty for completed stream", entry.ErrorCode)
	}
}

func TestBuildRequestLog_StreamIncomplete(t *testing.T) {
	sr := &StreamResult{
		LatencyMs:          5000,
		TimeToFirstChunkMs: 300,
		Completed:          false,
		ErrorCode:          "upstream_eof",
		ErrorMessage:       "stream ended before [DONE]",
		UpstreamStatusCode: 200,
	}
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 200,
		LatencyMs:          5000,
		Stream:             true,
		StreamResult:       sr,
	})

	if entry.StreamCompleted == nil || *entry.StreamCompleted != false {
		t.Errorf("StreamCompleted = %v, want false", entry.StreamCompleted)
	}
	if entry.ErrorCode != "upstream_eof" {
		t.Errorf("ErrorCode = %q, want upstream_eof", entry.ErrorCode)
	}
	if entry.TimeToFirstChunkMs == nil || *entry.TimeToFirstChunkMs != 300 {
		t.Errorf("TimeToFirstChunkMs = %v, want 300", entry.TimeToFirstChunkMs)
	}
}

func TestBuildRequestLog_FieldMapping(t *testing.T) {
	entry := buildRequestLog(logParams{
		Request:            testRequest(),
		Route:              testRoute(),
		Target:             testTarget(),
		UpstreamStatusCode: 200,
		LatencyMs:          100,
	})

	if entry.RouteName != "test-route" {
		t.Errorf("RouteName = %q, want test-route", entry.RouteName)
	}
	if entry.PublicModelName != "test-model" {
		t.Errorf("PublicModelName = %q, want test-model", entry.PublicModelName)
	}
	if entry.UpstreamModelName != "upstream-model" {
		t.Errorf("UpstreamModelName = %q, want upstream-model", entry.UpstreamModelName)
	}
	if entry.ProviderName != "test-provider" {
		t.Errorf("ProviderName = %q, want test-provider", entry.ProviderName)
	}
	if entry.TargetID == nil || *entry.TargetID != 10 {
		t.Errorf("TargetID = %v, want 10", entry.TargetID)
	}
	if entry.RouteID == nil || *entry.RouteID != 1 {
		t.Errorf("RouteID = %v, want 1", entry.RouteID)
	}
	if entry.ProviderID == nil || *entry.ProviderID != 5 {
		t.Errorf("ProviderID = %v, want 5", entry.ProviderID)
	}
}

func TestExtractClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	ip := extractClientIP(r)
	if ip != "1.2.3.4" {
		t.Errorf("extractClientIP(X-Forwarded-For) = %q, want 1.2.3.4", ip)
	}
}

func TestExtractClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "10.0.0.1")
	ip := extractClientIP(r)
	if ip != "10.0.0.1" {
		t.Errorf("extractClientIP(X-Real-IP) = %q, want 10.0.0.1", ip)
	}
}

func TestSSEScanner_DoneMarker(t *testing.T) {
	data := "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n\n"
	chunks := bytes.SplitAfter([]byte(data), []byte("\n\n"))
	scanner := NewSSEScanner(&chunkReader{chunks: chunks})

	var texts []string
	for scanner.Scan() {
		texts = append(texts, scanner.LastDelta())
	}

	if !scanner.Completed() {
		t.Error("scanner.Completed() = false, want true after [DONE]")
	}
	if len(texts) != 1 || texts[0] != "hello" {
		t.Errorf("got deltas %v, want [hello]", texts)
	}
	if scanner.TimeToFirstChunkMs() < 0 {
		t.Errorf("TimeToFirstChunkMs = %d, want >= 0", scanner.TimeToFirstChunkMs())
	}
}

type chunkReader struct {
	chunks [][]byte
	pos    int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.chunks) {
		return 0, nil
	}
	n := copy(p, r.chunks[r.pos])
	r.pos++
	return n, nil
}

func TestSSEScanner_EarlyEOF(t *testing.T) {
	data := "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"
	scanner := NewSSEScanner(bytes.NewBufferString(data))

	for scanner.Scan() {
	}

	if scanner.Completed() {
		t.Error("scanner.Completed() = true, want false on early EOF")
	}
}

func TestSSEScanner_EmptyStream(t *testing.T) {
	scanner := NewSSEScanner(bytes.NewBufferString(""))

	for scanner.Scan() {
	}

	if scanner.Completed() {
		t.Error("scanner.Completed() = true, want false on empty stream")
	}
	if scanner.TimeToFirstChunkMs() != 0 {
		t.Errorf("TimeToFirstChunkMs = %d, want 0 for empty stream", scanner.TimeToFirstChunkMs())
	}
}

func TestParseDelta_Valid(t *testing.T) {
	payload := []byte(`{"choices":[{"delta":{"content":"world"}}]}`)
	d := parseDelta(payload)
	if d != "world" {
		t.Errorf("parseDelta = %q, want world", d)
	}
}

func TestParseDelta_Empty(t *testing.T) {
	payload := []byte(`{"choices":[]}`)
	d := parseDelta(payload)
	if d != "" {
		t.Errorf("parseDelta(empty) = %q, want empty", d)
	}
}

func TestParseDelta_Invalid(t *testing.T) {
	d := parseDelta([]byte(`not json`))
	if d != "" {
		t.Errorf("parseDelta(invalid) = %q, want empty", d)
	}
}
