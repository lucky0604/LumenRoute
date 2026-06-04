package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"lumenroute/internal/apikey"
	"lumenroute/internal/logs"
	"lumenroute/internal/provider"
	"lumenroute/internal/route"
)

// AdminHandlers holds service references for all admin HTTP handlers.
type AdminHandlers struct {
	Providers *provider.Service
	Routes    *route.Service
	APIKeys   *apikey.Service
	Logs      *logs.Service
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

// parseID extracts a trailing numeric ID from a URL path after the given prefix.
func parseID(path, prefix string) (int64, error) {
	s := strings.TrimPrefix(path, prefix)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimRight(s, "/")
	// For paths like /api/providers/5/check, extract the "5"
	parts := strings.Split(s, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
			return id, nil
		}
	}
	return 0, fmt.Errorf("id not found in path: %s", path)
}

// --- Providers ---

// ListProviders handles GET /api/providers
func (h *AdminHandlers) ListProviders(w http.ResponseWriter, r *http.Request) {
	data, err := h.Providers.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []provider.Provider{}
	}
	respondJSON(w, http.StatusOK, data)
}

// CreateProvider handles POST /api/providers
func (h *AdminHandlers) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req provider.CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p := provider.Provider{
		Name:            req.Name,
		Description:     req.Description,
		ProviderType:    req.ProviderType,
		Engine:          req.Engine,
		BaseURL:         req.BaseURL,
		AuthMode:        req.AuthMode,
		CustomHeaders:   string(req.CustomHeaders),
		HealthCheckPath: req.HealthCheckPath,
		Enabled:         req.Enabled,
	}
	id, err := h.Providers.Create(p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// GetProvider handles GET /api/providers/{id}
func (h *AdminHandlers) GetProvider(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/providers/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	v, err := h.Providers.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "provider not found")
		return
	}
	respondJSON(w, http.StatusOK, v)
}

// UpdateProvider handles PUT /api/providers/{id}
func (h *AdminHandlers) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/providers/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req provider.CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p := provider.Provider{
		Name:            req.Name,
		Description:     req.Description,
		ProviderType:    req.ProviderType,
		Engine:          req.Engine,
		BaseURL:         req.BaseURL,
		AuthMode:        req.AuthMode,
		CustomHeaders:   string(req.CustomHeaders),
		HealthCheckPath: req.HealthCheckPath,
		Enabled:         req.Enabled,
	}
	if err := h.Providers.Update(id, p); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteProvider handles DELETE /api/providers/{id}
func (h *AdminHandlers) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/providers/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Providers.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// CheckProvider handles POST /api/providers/{id}/check
func (h *AdminHandlers) CheckProvider(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/providers/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.Providers.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "provider not found")
		return
	}
	healthPath := p.HealthCheckPath
	if healthPath == "" {
		healthPath = provider.GetDefaultHealthPath(p.Engine)
	}
	checkURL := p.BaseURL + healthPath
	resp, err := http.Get(checkURL)
	if err != nil {
		h.Providers.UpdateHealth(id, "unhealthy", 0, 0, err.Error())
		respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "healthy": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	status := "healthy"
	errMsg := ""
	if !healthy {
		status = "unhealthy"
		errMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}
	h.Providers.UpdateHealth(id, status, resp.StatusCode, 0, errMsg)
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "healthy": healthy, "status_code": resp.StatusCode})
}

// --- Routes ---

// ListRoutes handles GET /api/routes
func (h *AdminHandlers) ListRoutes(w http.ResponseWriter, r *http.Request) {
	data, err := h.Routes.ListRoutes()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []route.Route{}
	}
	respondJSON(w, http.StatusOK, data)
}

// CreateRoute handles POST /api/routes
func (h *AdminHandlers) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		PublicModelName string `json:"public_model_name"`
		Description     string `json:"description"`
		Enabled         bool   `json:"enabled"`
		RequireAuth     *bool  `json:"require_auth"`
		ProjectID       *int64 `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	requireAuth := true
	if req.RequireAuth != nil {
		requireAuth = *req.RequireAuth
	}
	rt := route.Route{
		Name:            req.Name,
		PublicModelName: req.PublicModelName,
		Description:     req.Description,
		Enabled:         req.Enabled,
		RequireAuth:     requireAuth,
		ProjectID:       req.ProjectID,
	}
	id, err := h.Routes.CreateRoute(rt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// GetRoute handles GET /api/routes/{id}
func (h *AdminHandlers) GetRoute(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/routes/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	v, err := h.Routes.GetRoute(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "route not found")
		return
	}
	respondJSON(w, http.StatusOK, v)
}

// UpdateRoute handles PUT /api/routes/{id}
func (h *AdminHandlers) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/routes/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
		RequireAuth *bool  `json:"require_auth"`
		ProjectID   *int64 `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	requireAuth := true
	if req.RequireAuth != nil {
		requireAuth = *req.RequireAuth
	}
	rt := route.Route{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		RequireAuth: requireAuth,
		ProjectID:   req.ProjectID,
	}
	if err := h.Routes.UpdateRoute(id, rt); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteRoute handles DELETE /api/routes/{id}
func (h *AdminHandlers) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/routes/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Routes.DeleteRoute(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Route Targets ---

// ListTargets handles GET /api/routes/{routeId}/targets
func (h *AdminHandlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	routeID, err := parseID(r.URL.Path, "/api/routes/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.Routes.ListTargets(routeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []route.RouteTarget{}
	}
	respondJSON(w, http.StatusOK, data)
}

// CreateTarget handles POST /api/routes/{routeId}/targets
func (h *AdminHandlers) CreateTarget(w http.ResponseWriter, r *http.Request) {
	routeID, err := parseID(r.URL.Path, "/api/routes/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		ProviderID        int64  `json:"provider_id"`
		UpstreamModelName string `json:"upstream_model_name"`
		Weight            int    `json:"weight"`
		TimeoutSeconds    int    `json:"timeout_seconds"`
		Enabled           bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t := route.RouteTarget{
		RouteID:           routeID,
		ProviderID:        req.ProviderID,
		UpstreamModelName: req.UpstreamModelName,
		Weight:            req.Weight,
		TimeoutSeconds:    req.TimeoutSeconds,
		Enabled:           req.Enabled,
	}
	id, err := h.Routes.CreateTarget(t)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateTarget handles PUT /api/route-targets/{id}
func (h *AdminHandlers) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/route-targets/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		ProviderID        int64  `json:"provider_id"`
		UpstreamModelName string `json:"upstream_model_name"`
		Weight            int    `json:"weight"`
		TimeoutSeconds    int    `json:"timeout_seconds"`
		Enabled           bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t := route.RouteTarget{
		ProviderID:        req.ProviderID,
		UpstreamModelName: req.UpstreamModelName,
		Weight:            req.Weight,
		TimeoutSeconds:    req.TimeoutSeconds,
		Enabled:           req.Enabled,
	}
	if err := h.Routes.UpdateTarget(id, t); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteTarget handles DELETE /api/route-targets/{id}
func (h *AdminHandlers) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/route-targets/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Routes.DeleteTarget(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- API Keys ---

// ListAPIKeys handles GET /api/api-keys
func (h *AdminHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	data, err := h.APIKeys.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []apikey.APIKey{}
	}
	respondJSON(w, http.StatusOK, data)
}

// CreateAPIKey handles POST /api/api-keys
func (h *AdminHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string  `json:"name"`
		Description     string  `json:"description"`
		AllowedRouteIDs string  `json:"allowed_route_ids"`
		ExpiresAt       *string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	key, err := h.APIKeys.Create(req.Name, req.Description, req.AllowedRouteIDs, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       key.ID,
		"raw_key":  key.RawKey,
		"prefix":   key.KeyPrefix,
	})
}

// DeleteAPIKey handles DELETE /api/api-keys/{id}
func (h *AdminHandlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/api-keys/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.APIKeys.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// DisableAPIKey handles POST /api/api-keys/{id}/disable
func (h *AdminHandlers) DisableAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/api-keys/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.APIKeys.Disable(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "enabled": false})
}

// EnableAPIKey handles POST /api/api-keys/{id}/enable
func (h *AdminHandlers) EnableAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/api-keys/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.APIKeys.Enable(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "enabled": true})
}

// --- Request Logs ---

// ListLogs handles GET /api/request-logs
func (h *AdminHandlers) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter logs.LogFilter
	filter.Model = q.Get("model")
	filter.Provider = q.Get("provider")
	filter.RequestID = q.Get("request_id")
	if v := q.Get("status_code"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.StatusCode = n
		}
	}
	if v := q.Get("stream"); v != "" {
		b := v == "true" || v == "1"
		filter.Stream = &b
	}
	if v := q.Get("error_only"); v == "true" || v == "1" {
		filter.ErrorOnly = true
	}
	data, err := h.Logs.List(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []logs.RequestLog{}
	}
	respondJSON(w, http.StatusOK, data)
}

// GetLog handles GET /api/request-logs/{id}
func (h *AdminHandlers) GetLog(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/request-logs/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	v, err := h.Logs.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "log not found")
		return
	}
	respondJSON(w, http.StatusOK, v)
}
