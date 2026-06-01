package proxy

import (
	"net/http"
	"strings"

	"lumenroute/internal/logs"
	"lumenroute/internal/route"
)

type logParams struct {
	RequestID          string
	Request            *http.Request
	Route              *route.Route
	Target             *route.RouteTarget
	UpstreamStatusCode int
	LatencyMs          int64
	Stream             bool
	StreamResult       *StreamResult
	RequestBody        []byte
	ResponseBody       []byte
	ErrorCode          string
	ErrorMessage       string
}

func buildRequestLog(p logParams) logs.RequestLog {
	rid := p.RequestID
	if rid == "" {
		rid = logs.GenerateRequestID("")
	}
	entry := logs.RequestLog{
		RequestID:          rid,
		RouteID:            &p.Route.ID,
		RouteName:          p.Route.Name,
		PublicModelName:    p.Route.PublicModelName,
		UpstreamModelName:  p.Target.UpstreamModelName,
		ProviderID:         &p.Target.ProviderID,
		ProviderName:       p.Target.ProviderName,
		TargetID:           &p.Target.ID,
		ClientIP:           extractClientIP(p.Request),
		Method:             p.Request.Method,
		Path:               p.Request.URL.Path,
		Stream:             p.Stream,
		StatusCode:         p.UpstreamStatusCode,
		UpstreamStatusCode: p.UpstreamStatusCode,
		LatencyMs:          int(p.LatencyMs),
		ErrorCode:          p.ErrorCode,
		ErrorMessage:       p.ErrorMessage,
	}

	if p.Stream && p.StreamResult != nil {
		sr := p.StreamResult
		ttfc := int(sr.TimeToFirstChunkMs)
		entry.TimeToFirstChunkMs = &ttfc
		entry.StreamCompleted = &sr.Completed
		if sr.ErrorCode != "" {
			entry.ErrorCode = sr.ErrorCode
			entry.ErrorMessage = sr.ErrorMessage
		}
		if sr.PromptTokens > 0 {
			entry.PromptTokens = &sr.PromptTokens
		}
		if sr.CompletionTokens > 0 {
			entry.CompletionTokens = &sr.CompletionTokens
		}
		if sr.TotalTokens > 0 {
			entry.TotalTokens = &sr.TotalTokens
		}
	} else if !p.Stream && p.ResponseBody != nil {
		if ti := extractTokenUsage(p.ResponseBody); ti != nil {
			entry.PromptTokens = ti.PromptTokens
			entry.CompletionTokens = ti.CompletionTokens
			entry.TotalTokens = ti.TotalTokens
		}
	}

	const maxBodyLen = 65536
	if len(p.RequestBody) > 0 {
		if len(p.RequestBody) > maxBodyLen {
			entry.RequestBody = string(p.RequestBody[:maxBodyLen]) + "\n... [truncated]"
		} else {
			entry.RequestBody = string(p.RequestBody)
		}
	}
	if len(p.ResponseBody) > 0 {
		if len(p.ResponseBody) > maxBodyLen {
			entry.ResponseBody = string(p.ResponseBody[:maxBodyLen]) + "\n... [truncated]"
		} else {
			entry.ResponseBody = string(p.ResponseBody)
		}
	}

	return entry
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
