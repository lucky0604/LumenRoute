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
	"lumenroute/internal/logs"
	"lumenroute/internal/route"

	"github.com/pkoukk/tiktoken-go"
)

type Service struct {
	routeService  *route.Service
	apiKeyService *apikey.Service
	logService    *logs.Service
	proxyAuthMode string
	client        *http.Client
}

func NewService(rs *route.Service, aks *apikey.Service, ls *logs.Service, proxyAuthMode string) *Service {
	return &Service{
		routeService:  rs,
		apiKeyService: aks,
		logService:    ls,
		proxyAuthMode: proxyAuthMode,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (s *Service) ListModels(w http.ResponseWriter, _ *http.Request) {
	routes, err := s.routeService.ListRoutes()
	if err != nil {
		writeError(w, 500, "internal_error", "Failed to list models")
		return
	}

	models := make([]map[string]interface{}, 0)
	for _, r := range routes {
		if !r.Enabled { continue }
		targets, _ := s.routeService.GetReadyTargets(r.ID)
		if len(targets) == 0 { continue }
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "invalid_request_error", "Failed to read request body")
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

	req["model"] = target.UpstreamModelName
	reqBody, _ := json.Marshal(req)

	streamMode, _ := req["stream"].(bool)
	if streamMode {
		s.proxyStream(w, r, target, reqBody, rt)
		return
	}
	s.proxyNonStream(w, r, target, reqBody, rt, body)
}

func (s *Service) proxyNonStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte, rt *route.Route, origBody []byte) {
	upstreamReq, err := s.buildUpstreamRequest(target, body, r)
	if err != nil {
		writeError(w, 502, "upstream_error", "Failed to build upstream request")
		return
	}

	start := time.Now()
	resp, err := s.client.Do(upstreamReq)
	if err != nil {
		writeError(w, 502, "upstream_connection_failed", "Failed to connect to upstream provider")
		return
	}
	defer resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	s.recordRequestLog(r, rt, target, resp.StatusCode, latency, false, respBody)
}

func (s *Service) proxyStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte, rt *route.Route) {
	upstreamReq, err := s.buildUpstreamRequest(target, body, r)
	if err != nil {
		writeError(w, 502, "upstream_error", "Failed to build upstream request")
		return
	}

	start := time.Now()
	resp, err := s.client.Do(upstreamReq)
	if err != nil {
		writeError(w, 502, "upstream_connection_failed", "Failed to connect to upstream provider")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.recordStreamLog(r, rt, target, resp.StatusCode, time.Since(start).Milliseconds(), "", body)
		return
	}

	var accumulated strings.Builder
	scanner := NewSSEScanner(resp.Body)
	for scanner.Scan() {
		w.Write(scanner.RawLine())
		w.Write([]byte("\n"))
		flusher.Flush()

		delta := scanner.LastDelta()
		if delta != "" {
			accumulated.WriteString(delta)
		}
	}
	s.recordStreamLog(r, rt, target, resp.StatusCode, time.Since(start).Milliseconds(), accumulated.String(), body)
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

func (s *Service) recordRequestLog(r *http.Request, rt *route.Route, target *route.RouteTarget, upstreamStatusCode int, latencyMs int64, stream bool, respBody []byte) {
	if s.logService == nil {
		return
	}

	logEntry := logs.RequestLog{
		RequestID:         logs.GenerateRequestID(""),
		RouteID:           &rt.ID,
		RouteName:         rt.Name,
		PublicModelName:   rt.PublicModelName,
		UpstreamModelName: target.UpstreamModelName,
		ProviderID:        &target.ProviderID,
		ProviderName:      target.ProviderName,
		TargetID:          &target.ID,
		ClientIP:          extractClientIP(r),
		Method:            r.Method,
		Path:              r.URL.Path,
		Stream:            stream,
		StatusCode:        200,
		UpstreamStatusCode: upstreamStatusCode,
		LatencyMs:          int(latencyMs),
	}

	if tokenInfo := extractTokenUsage(respBody); tokenInfo != nil {
		logEntry.PromptTokens = tokenInfo.PromptTokens
		logEntry.CompletionTokens = tokenInfo.CompletionTokens
		logEntry.TotalTokens = tokenInfo.TotalTokens
	}

	_ = s.logService.Write(logEntry)
}

func (s *Service) recordStreamLog(r *http.Request, rt *route.Route, target *route.RouteTarget, upstreamStatusCode int, latencyMs int64, completionText string, reqBody []byte) {
	if s.logService == nil {
		return
	}

	logEntry := logs.RequestLog{
		RequestID:          logs.GenerateRequestID(""),
		RouteID:            &rt.ID,
		RouteName:          rt.Name,
		PublicModelName:    rt.PublicModelName,
		UpstreamModelName:  target.UpstreamModelName,
		ProviderID:         &target.ProviderID,
		ProviderName:       target.ProviderName,
		TargetID:           &target.ID,
		ClientIP:           extractClientIP(r),
		Method:             r.Method,
		Path:               r.URL.Path,
		Stream:             true,
		StatusCode:         200,
		UpstreamStatusCode: upstreamStatusCode,
		LatencyMs:          int(latencyMs),
	}

	if ti := countStreamTokens(completionText, reqBody); ti != nil {
		logEntry.PromptTokens = ti.PromptTokens
		logEntry.CompletionTokens = ti.CompletionTokens
		logEntry.TotalTokens = ti.TotalTokens
	}

	_ = s.logService.Write(logEntry)
}

func countStreamTokens(completionText string, reqBody []byte) *tokenInfo {
	enc, err := tiktoken.GetEncoding(tiktoken.MODEL_CL100K_BASE)
	if err != nil {
		return nil
	}

	var ti tokenInfo

	completionCount := len(enc.Encode(completionText, nil, nil))
	if completionCount > 0 {
		ti.CompletionTokens = &completionCount
	}

	promptText := extractPromptText(reqBody)
	promptCount := len(enc.Encode(promptText, nil, nil))
	if promptCount > 0 {
		ti.PromptTokens = &promptCount
	}

	total := promptCount + completionCount
	ti.TotalTokens = &total
	return &ti
}

func extractPromptText(reqBody []byte) string {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, msg := range req.Messages {
		sb.WriteString(msg.Role)
		sb.WriteString(": ")
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

type SSEScanner struct {
	reader    io.Reader
	buf       []byte
	rawLine   []byte
	lastDelta string
}

func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: r, buf: make([]byte, 4096)}
}

func (s *SSEScanner) Scan() bool {
	for {
		n, err := s.reader.Read(s.buf)
		if n == 0 {
			return false
		}
		s.lastDelta = ""
		lines := bytes.Split(s.buf[:n], []byte("\n"))
		for _, line := range lines {
			s.rawLine = line
			if bytes.HasPrefix(line, []byte("data: ")) {
				payload := bytes.TrimPrefix(line, []byte("data: "))
				if bytes.Equal(payload, []byte("[DONE]")) {
					return false
				}
				s.lastDelta = parseDelta(payload)
				return true
			}
		}
		if err != nil {
			return false
		}
	}
}

func (s *SSEScanner) RawLine() []byte {
	return s.rawLine
}

func (s *SSEScanner) LastDelta() string {
	return s.lastDelta
}

func parseDelta(payload []byte) string {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return ""
	}
	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}

type tokenInfo struct {
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
}

func extractTokenUsage(body []byte) *tokenInfo {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Usage.PromptTokens == 0 && resp.Usage.TotalTokens == 0 {
		return nil
	}
	var info tokenInfo
	if resp.Usage.PromptTokens > 0 {
		v := resp.Usage.PromptTokens; info.PromptTokens = &v
	}
	if resp.Usage.CompletionTokens > 0 {
		v := resp.Usage.CompletionTokens; info.CompletionTokens = &v
	}
	if resp.Usage.TotalTokens > 0 {
		v := resp.Usage.TotalTokens; info.TotalTokens = &v
	}
	return &info
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
