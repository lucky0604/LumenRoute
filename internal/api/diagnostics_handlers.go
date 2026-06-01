package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"lumenroute/internal/diagnostics"
)

type DiagnosticsHandler struct {
	Service *diagnostics.Service
}

func (h *DiagnosticsHandler) HandleGetModelOverview(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "1h"
	}

	summaries, err := h.Service.GetModelOverview(window)
	if err != nil {
		if strings.Contains(err.Error(), "invalid window") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to query model overview")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"window": window,
		"models": summaries,
	})
}

func (h *DiagnosticsHandler) HandleGetTargetDiagnosis(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/diagnostics/targets/")
	idStr = strings.TrimSuffix(idStr, "/")
	parts := strings.Split(idStr, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "target id required")
		return
	}

	targetID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid target id")
		return
	}

	window := r.URL.Query().Get("window")
	if window == "" {
		window = "1h"
	}

	diag, err := h.Service.GetTargetDiagnosis(targetID, window)
	if err != nil {
		if err == diagnostics.ErrNotFound {
			respondError(w, http.StatusNotFound, "target not found")
			return
		}
		if strings.Contains(err.Error(), "invalid window") {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to query target diagnosis")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diag)
}

func RegisterDiagnosticsRoutes(mux *http.ServeMux, h *DiagnosticsHandler) {
	mux.HandleFunc("/api/diagnostics/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.HandleGetModelOverview(w, r)
	})
	mux.HandleFunc("/api/diagnostics/targets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.HandleGetTargetDiagnosis(w, r)
	})
}
