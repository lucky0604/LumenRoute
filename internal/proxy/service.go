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
	"lumenroute/internal/route"
)

type Service struct {
	routeService  *route.Service
	apiKeyService *apikey.Service
	proxyAuthMode string
	client        *http.Client
}

func NewService(rs *route.Service, aks *apikey.Service, proxyAuthMode string) *Service {
	return &Service{
		routeService:  rs,
		apiKeyService: aks,
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
		s.proxyStream(w, r, target, reqBody)
		return
	}
	s.proxyNonStream(w, r, target, reqBody)
}

func (s *Service) proxyNonStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte) {
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
	_ = latency
}

func (s *Service) proxyStream(w http.ResponseWriter, r *http.Request, target *route.RouteTarget, body []byte) {
	upstreamReq, err := s.buildUpstreamRequest(target, body, r)
	if err != nil {
		writeError(w, 502, "upstream_error", "Failed to build upstream request")
		return
	}

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
		return
	}

	done := false
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
			if bytes.Contains(buf[:n], []byte("[DONE]")) {
				done = true
			}
		}
		if readErr != nil {
			break
		}
		if r.Context().Err() != nil {
			break
		}
	}
	_ = done
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
