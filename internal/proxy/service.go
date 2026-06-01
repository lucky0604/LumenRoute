package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lumenroute/internal/apikey"
	"lumenroute/internal/capture"
	"lumenroute/internal/logs"
	"lumenroute/internal/metrics"
	"lumenroute/internal/project"
	"lumenroute/internal/route"
)

const maxProxyBodySize = 10 * 1024 * 1024

type Service struct {
	routeService       *route.Service
	apiKeyService      *apikey.Service
	logService         *logs.Service
	captureService     *capture.Service
	projectService     *project.Service
	recorder           metrics.Recorder
	proxyAuthMode      string
	captureEnabled     bool
	captureMaxBodySize int
	cache              *projCache
	client             *http.Client
}

type ServiceConfig struct {
	ProxyAuthMode      string
	CaptureEnabled     bool
	CaptureMaxBodySize int
}

func NewService(rs *route.Service, aks *apikey.Service, ls *logs.Service, rec metrics.Recorder, cfg ServiceConfig) *Service {
	s := &Service{
		routeService:       rs,
		apiKeyService:      aks,
		logService:         ls,
		recorder:           rec,
		proxyAuthMode:      cfg.ProxyAuthMode,
		captureEnabled:     cfg.CaptureEnabled,
		captureMaxBodySize: cfg.CaptureMaxBodySize,
		cache:              newProjCache(60 * time.Second),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	return s
}

// SetCaptureService wires the capture subsystem after construction.
func (s *Service) SetCaptureService(cs *capture.Service, ps *project.Service) {
	s.captureService = cs
	s.projectService = ps
}

// InvalidateProjectCache evicts a project from the in-memory config cache.
func (s *Service) InvalidateProjectCache(projectID int64) {
	s.cache.invalidate(projectID)
}

func (s *Service) ListModels(w http.ResponseWriter, _ *http.Request) {
	routes, err := s.routeService.ListRoutes()
	if err != nil {
		writeError(w, 500, "internal_error", "Failed to list models")
		return
	}

	models := make([]map[string]interface{}, 0)
	for _, r := range routes {
		if !r.Enabled {
			continue
		}
		targets, _ := s.routeService.GetReadyTargets(r.ID)
		if len(targets) == 0 {
			continue
		}
		models = append(models, map[string]interface{}{
			"id": r.PublicModelName, "object": "model", "created": r.CreatedAt.Unix(), "owned_by": "lumenroute",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"object": "list", "data": models})
}

func (s *Service) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "invalid_request_error", "Method not allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxProxyBodySize))
	if err != nil {
		writeError(w, 400, "invalid_request_error", "Failed to read request body")
		return
	}
	if int64(len(body)) >= maxProxyBodySize {
		writeError(w, 413, "invalid_request_error", "Request body too large")
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, 400, "invalid_request_error", "Invalid JSON body")
		return
	}

	model, _ := req["model"].(string)
	if model == "" {
		writeError(w, 400, "invalid_request_error", "Model field is required")
		return
	}

	rt, err := s.routeService.FindByModelName(model)
	if err != nil {
		writeError(w, 404, "model_not_found", fmt.Sprintf("Model not found: %s", model))
		return
	}

	if s.proxyAuthMode != "disabled" && rt.RequireAuth && s.apiKeyService != nil {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, 401, "invalid_api_key", "API key required for this route")
			return
		}
		rawKey := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := s.apiKeyService.ValidateKey(rawKey); err != nil {
			writeError(w, 401, "invalid_api_key", "Invalid API key")
			return
		}
	}

	target, err := s.routeService.SelectTarget(rt.ID)
	if err != nil {
		writeError(w, 503, "no_available_target", fmt.Sprintf("No available target for model %s", model))
		return
	}

	// Preserve original client body before model rewrite (for capture)
	clientBody := make([]byte, len(body))
	copy(clientBody, body)

	requestID := logs.GenerateRequestID("")

	req["model"] = target.UpstreamModelName
	reqBody, _ := json.Marshal(req)

	streamMode, _ := req["stream"].(bool)
	if streamMode {
		s.proxyStream(w, r, target, reqBody, rt, requestID, clientBody)
		return
	}
	s.proxyNonStream(w, r, target, reqBody, rt, requestID, clientBody)
}

func (s *Service) proxyNonStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte, rt *route.Route, requestID string, clientBody []byte) {
	upstreamReq, err := s.buildUpstreamRequest(target, body, r)
	if err != nil {
		writeError(w, 502, "upstream_error", "Failed to build upstream request")
		return
	}

	start := time.Now()
	resp, err := s.client.Do(upstreamReq)
	if err != nil {
		s.writeLog(logParams{
			RequestID: requestID,
			Request: r, Route: rt, Target: target,
			UpstreamStatusCode: 502,
			LatencyMs:          time.Since(start).Milliseconds(),
				RequestBody:        clientBody,
			ErrorCode:          "upstream_connection_failed",
			ErrorMessage:       err.Error(),
		})
		writeError(w, 502, "upstream_connection_failed", "Failed to connect to upstream provider")
		return
	}
	defer resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodySize))

	s.writeLog(logParams{
		RequestID: requestID,
		Request: r, Route: rt, Target: target,
		UpstreamStatusCode: resp.StatusCode,
		LatencyMs:          latency,
		RequestBody:        clientBody,
			ResponseBody:       respBody,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	var projectID int64
	if rt.ProjectID != nil {
		projectID = *rt.ProjectID
	}
	if projectID > 0 {
		ti := extractTokenUsage(respBody)
		entry := capture.CaptureEntry{
			RequestID:         requestID,
			ProjectID:         projectID,
			RouteName:         rt.Name,
			PublicModelName:   rt.PublicModelName,
			UpstreamModelName: target.UpstreamModelName,
			ProviderName:      target.ProviderName,
			Stream:            false,
			StatusCode:        resp.StatusCode,
			LatencyMs:         int(latency),
			RequestBody:       json.RawMessage(clientBody),
			ResponseBody:      json.RawMessage(respBody),
		}
		if ti != nil {
			entry.PromptTokens = ti.PromptTokens
			entry.CompletionTokens = ti.CompletionTokens
		}
		s.maybeCapture(entry)
	}
}

func (s *Service) proxyStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte, rt *route.Route, requestID string, clientBody []byte) {
	upstreamReq, err := s.buildUpstreamRequest(target, body, r)
	if err != nil {
		writeError(w, 502, "upstream_error", "Failed to build upstream request")
		return
	}

	var projectID int64
	if rt.ProjectID != nil {
		projectID = *rt.ProjectID
	}

	start := time.Now()
	resp, err := s.client.Do(upstreamReq)
	if err != nil {
		s.writeLog(logParams{
			RequestID: requestID,
			Request: r, Route: rt, Target: target,
			UpstreamStatusCode: 502,
			LatencyMs:          time.Since(start).Milliseconds(),
				RequestBody:        clientBody,
			Stream:             true,
			ErrorCode:          "upstream_connection_failed",
			ErrorMessage:       err.Error(),
		})
		writeError(w, 502, "upstream_connection_failed", "Failed to connect to upstream provider")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodySize))
		latency := time.Since(start).Milliseconds()
		s.writeLog(logParams{
			RequestID: requestID,
			Request: r, Route: rt, Target: target,
			UpstreamStatusCode: resp.StatusCode,
			LatencyMs:          latency,
			Stream:             true,
			RequestBody:        clientBody,
			ResponseBody:       respBody,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)

		if projectID > 0 {
			s.maybeCapture(capture.CaptureEntry{
				RequestID:         requestID,
				ProjectID:         projectID,
				RouteName:         rt.Name,
				PublicModelName:   rt.PublicModelName,
				UpstreamModelName: target.UpstreamModelName,
				ProviderName:      target.ProviderName,
				Stream:            true,
				StatusCode:        resp.StatusCode,
				LatencyMs:         int(latency),
				RequestBody:       json.RawMessage(clientBody),
				ResponseBody:      json.RawMessage(respBody),
			})
		}
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	if s.recorder != nil {
		s.recorder.IncActiveStream()
		defer s.recorder.DecActiveStream()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeLog(logParams{
			RequestID: requestID,
			Request: r, Route: rt, Target: target,
			UpstreamStatusCode: resp.StatusCode,
			LatencyMs:          time.Since(start).Milliseconds(),
				RequestBody:        clientBody,
			Stream:             true,
			ErrorCode:          "no_flusher",
			ErrorMessage:       "ResponseWriter does not support flushing",
		})
		return
	}

	scanner := NewSSEScanner(resp.Body)
	var accumulated strings.Builder
	for scanner.Scan() {
		w.Write(scanner.RawLine())
		w.Write([]byte("\n\n"))
		flusher.Flush()
		delta := scanner.LastDelta()
		if delta != "" {
			accumulated.WriteString(delta)
		}
	}
	if scanner.Completed() {
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}

	sr := &StreamResult{
		LatencyMs:          time.Since(start).Milliseconds(),
		TimeToFirstChunkMs: scanner.TimeToFirstChunkMs(),
		Completed:          scanner.Completed(),
		UpstreamStatusCode: resp.StatusCode,
		CompletionText:     accumulated.String(),
	}
	if !scanner.Completed() {
		sr.ErrorCode = "upstream_eof"
		sr.ErrorMessage = "stream ended before [DONE]"
	}
	if ti := countStreamTokens(sr.CompletionText, body); ti != nil {
		if ti.PromptTokens != nil {
			sr.PromptTokens = *ti.PromptTokens
		}
		if ti.CompletionTokens != nil {
			sr.CompletionTokens = *ti.CompletionTokens
		}
		if ti.TotalTokens != nil {
			sr.TotalTokens = *ti.TotalTokens
		}
	}

	reconstructed := reconstructStreamResponse(accumulated.String(), sr, scanner, clientBody)
	s.writeLog(logParams{
		RequestID: requestID,
		Request: r, Route: rt, Target: target,
		UpstreamStatusCode: resp.StatusCode,
		LatencyMs:          sr.LatencyMs,
		RequestBody:        clientBody,
		ResponseBody:       reconstructed,
		Stream:             true,
		StreamResult:       sr,
	})

	if projectID > 0 {
		ttfc := int(sr.TimeToFirstChunkMs)
		entry := capture.CaptureEntry{
			RequestID:         requestID,
			ProjectID:         projectID,
			RouteName:         rt.Name,
			PublicModelName:   rt.PublicModelName,
			UpstreamModelName: target.UpstreamModelName,
			ProviderName:      target.ProviderName,
			Stream:            true,
			StreamCompleted:   scanner.Completed(),
			StatusCode:        resp.StatusCode,
			LatencyMs:         int(sr.LatencyMs),
			TTFCMs:            &ttfc,
			RequestBody:       json.RawMessage(clientBody),
			ResponseBody:      json.RawMessage(reconstructed),
		}
		if sr.PromptTokens > 0 {
			entry.PromptTokens = &sr.PromptTokens
		}
		if sr.CompletionTokens > 0 {
			entry.CompletionTokens = &sr.CompletionTokens
		}
		s.maybeCapture(entry)
	}
}

func (s *Service) writeLog(p logParams) {
	if s.logService != nil {
		entry := buildRequestLog(p)
		_ = s.logService.Write(entry)
	}
	if s.recorder != nil {
		totalTokens := 0
		if p.Stream && p.StreamResult != nil {
			totalTokens = p.StreamResult.TotalTokens
		} else if !p.Stream && p.ResponseBody != nil {
			if ti := extractTokenUsage(p.ResponseBody); ti != nil && ti.TotalTokens != nil {
				totalTokens = *ti.TotalTokens
			}
		}
		s.recorder.RecordProxyRequest(metrics.ProxyRequest{
			StatusCode: p.UpstreamStatusCode,
			IsError:    p.UpstreamStatusCode >= 400 || p.ErrorCode != "",
			Tokens:     totalTokens,
			IsStream:   p.Stream,
		})
	}
}

func (s *Service) buildUpstreamRequest(target *route.RouteTarget, body []byte, r *http.Request) (*http.Request, error) {
	upstreamURL := openAIChatCompletionsURL(target.ProviderBaseURL)
	req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, vs := range r.Header {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Host") ||
			strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Connection") ||
			strings.EqualFold(k, "Transfer-Encoding") {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func openAIChatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"message": msg, "type": codeToType(code), "code": code},
	})
}

func codeToType(code string) string {
	switch code {
	case "invalid_api_key", "authentication_error":
		return "authentication_error"
	case "model_not_allowed":
		return "permission_error"
	case "model_not_found", "invalid_request_error":
		return "invalid_request_error"
	case "no_available_target":
		return "service_unavailable"
	default:
		return "upstream_error"
	}
}
