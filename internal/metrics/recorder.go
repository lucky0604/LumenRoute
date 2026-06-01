package metrics

type ProxyRequest struct {
	StatusCode int
	IsError    bool
	Tokens     int
	IsStream   bool
}

type Recorder interface {
	RecordProxyRequest(r ProxyRequest)
	SetProviderHealthCounts(healthy, unhealthy int64)
	IncActiveStream()
	DecActiveStream()
}
