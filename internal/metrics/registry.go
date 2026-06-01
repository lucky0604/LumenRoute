package metrics

import (
	"net/http"
	"sync/atomic"

	"lumenroute/internal/capture"
)

type Registry struct {
	requestsTotal    atomic.Int64
	requestErrors    atomic.Int64
	requestTokens    atomic.Int64
	activeStreams    atomic.Int64
	providerHealthy  atomic.Int64
	providerUnhealthy atomic.Int64
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (m *Registry) IncRequests()   { m.requestsTotal.Add(1) }
func (m *Registry) IncErrors()     { m.requestErrors.Add(1) }
func (m *Registry) AddTokens(n int64) { m.requestTokens.Add(n) }
func (m *Registry) IncStreams()    { m.activeStreams.Add(1) }
func (m *Registry) DecStreams()    { m.activeStreams.Add(-1) }
func (m *Registry) SetHealthy(n int64)     { m.providerHealthy.Store(n) }
func (m *Registry) SetUnhealthy(n int64)   { m.providerUnhealthy.Store(n) }

func (m *Registry) RecordProxyRequest(r ProxyRequest) {
	m.requestsTotal.Add(1)
	if r.IsError {
		m.requestErrors.Add(1)
	}
	if r.Tokens > 0 {
		m.requestTokens.Add(int64(r.Tokens))
	}
}

func (m *Registry) SetProviderHealthCounts(healthy, unhealthy int64) {
	m.providerHealthy.Store(healthy)
	m.providerUnhealthy.Store(unhealthy)
}

func (m *Registry) IncActiveStream() { m.activeStreams.Add(1) }
func (m *Registry) DecActiveStream() { m.activeStreams.Add(-1) }

func (m *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte("# HELP lumenroute_requests_total Total number of proxy requests.\n"))
		w.Write([]byte("# TYPE lumenroute_requests_total counter\n"))
		w.Write([]byte("lumenroute_requests_total " + itoa(m.requestsTotal.Load()) + "\n"))

		w.Write([]byte("# HELP lumenroute_request_errors_total Total number of proxy errors.\n"))
		w.Write([]byte("# TYPE lumenroute_request_errors_total counter\n"))
		w.Write([]byte("lumenroute_request_errors_total " + itoa(m.requestErrors.Load()) + "\n"))

		w.Write([]byte("# HELP lumenroute_request_tokens_total Total tokens processed.\n"))
		w.Write([]byte("# TYPE lumenroute_request_tokens_total counter\n"))
		w.Write([]byte("lumenroute_request_tokens_total " + itoa(m.requestTokens.Load()) + "\n"))

		w.Write([]byte("# HELP lumenroute_active_streams Current active streaming connections.\n"))
		w.Write([]byte("# TYPE lumenroute_active_streams gauge\n"))
		w.Write([]byte("lumenroute_active_streams " + itoa(m.activeStreams.Load()) + "\n"))

		w.Write([]byte("# HELP lumenroute_provider_healthy Number of healthy providers.\n"))
		w.Write([]byte("# TYPE lumenroute_provider_healthy gauge\n"))
		w.Write([]byte("lumenroute_provider_healthy " + itoa(m.providerHealthy.Load()) + "\n"))

		w.Write([]byte("# HELP lumenroute_provider_unhealthy Number of unhealthy providers.\n"))
		w.Write([]byte("# TYPE lumenroute_provider_unhealthy gauge\n"))
		w.Write([]byte("lumenroute_provider_unhealthy " + itoa(m.providerUnhealthy.Load()) + "\n"))

		cm := capture.Metrics()
		w.Write([]byte("# HELP lumenroute_capture_total Total captures submitted.\n"))
		w.Write([]byte("# TYPE lumenroute_capture_total counter\n"))
		w.Write([]byte("lumenroute_capture_total " + itoa(cm["capture_total"]) + "\n"))
		w.Write([]byte("# HELP lumenroute_capture_dropped_total Captures dropped due to full channel.\n"))
		w.Write([]byte("# TYPE lumenroute_capture_dropped_total counter\n"))
		w.Write([]byte("lumenroute_capture_dropped_total " + itoa(cm["capture_dropped_total"]) + "\n"))
		w.Write([]byte("# HELP lumenroute_capture_write_errors_total Capture file write errors.\n"))
		w.Write([]byte("# TYPE lumenroute_capture_write_errors_total counter\n"))
		w.Write([]byte("lumenroute_capture_write_errors_total " + itoa(cm["capture_write_errors_total"]) + "\n"))
		w.Write([]byte("# HELP lumenroute_capture_store_errors_total Capture DB store errors.\n"))
		w.Write([]byte("# TYPE lumenroute_capture_store_errors_total counter\n"))
		w.Write([]byte("lumenroute_capture_store_errors_total " + itoa(cm["capture_store_errors_total"]) + "\n"))
		w.Write([]byte("# HELP lumenroute_capture_export_skipped Captures skipped during export.\n"))
		w.Write([]byte("# TYPE lumenroute_capture_export_skipped counter\n"))
		w.Write([]byte("lumenroute_capture_export_skipped " + itoa(cm["capture_export_skipped"]) + "\n"))
	})
}

func itoa(n int64) string {
	if n == 0 { return "0" }
	s := ""
	neg := false
	if n < 0 { neg = true; n = -n }
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg { s = "-" + s }
	return s
}
