package api

import (
	"encoding/json"
	"net/http"

	"lumenroute/internal/project"
)

// ProjectHandlers holds service references for project-related HTTP handlers.
type ProjectHandlers struct {
	Projects         *project.Service
	OnProjectChanged func(projectID int64)
}

// ListProjects handles GET /api/projects
func (h *ProjectHandlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	data, err := h.Projects.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []project.Project{}
	}

	type item struct {
		project.Project
		HasExportToken bool  `json:"has_export_token"`
		RoutesCount    int64 `json:"routes_count"`
	}
	out := make([]item, len(data))
	for i, p := range data {
		out[i].Project = p
		out[i].HasExportToken = p.ExportTokenHash != ""
		out[i].Project.ExportTokenHash = ""
		cnt, _ := h.Projects.CountAssociatedRoutes(p.ID)
		out[i].RoutesCount = cnt
	}
	respondJSON(w, http.StatusOK, out)
}

// CreateProject handles POST /api/projects
func (h *ProjectHandlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		DataCategory   string  `json:"data_category"`
		CaptureEnabled bool    `json:"capture_enabled"`
		SampleRate     float64 `json:"sample_rate"`
		RetentionDays  int     `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	p := project.Project{
		Name:           req.Name,
		Description:    req.Description,
		DataCategory:   req.DataCategory,
		CaptureEnabled: req.CaptureEnabled,
		SampleRate:     req.SampleRate,
		RetentionDays:  req.RetentionDays,
	}
	id, err := h.Projects.Create(p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// GetProject handles GET /api/projects/{id}
func (h *ProjectHandlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.Projects.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}
	type resp struct {
		*project.Project
		HasExportToken bool `json:"has_export_token"`
	}
	out := resp{Project: p, HasExportToken: p.ExportTokenHash != ""}
	out.Project.ExportTokenHash = ""
	respondJSON(w, http.StatusOK, out)
}

// UpdateProject handles PUT /api/projects/{id}
func (h *ProjectHandlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		DataCategory   string  `json:"data_category"`
		CaptureEnabled bool    `json:"capture_enabled"`
		SampleRate     float64 `json:"sample_rate"`
		RetentionDays  int     `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p := project.Project{
		Name:           req.Name,
		Description:    req.Description,
		DataCategory:   req.DataCategory,
		CaptureEnabled: req.CaptureEnabled,
		SampleRate:     req.SampleRate,
		RetentionDays:  req.RetentionDays,
	}
	if err := h.Projects.Update(id, p); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.OnProjectChanged != nil {
		h.OnProjectChanged(id)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteProject handles DELETE /api/projects/{id}
func (h *ProjectHandlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Projects.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.OnProjectChanged != nil {
		h.OnProjectChanged(id)
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetProjectStats handles GET /api/projects/{id}/stats
func (h *ProjectHandlers) GetProjectStats(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	stats, err := h.Projects.GetStats(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// GenerateExportToken handles POST /api/projects/{id}/export-token
func (h *ProjectHandlers) GenerateExportToken(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, err := h.Projects.GenerateExportToken(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"export_token": token})
}
