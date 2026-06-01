package main

import (
	"net/http"
	"strings"

	"lumenroute/internal/api"
)

func registerProjectRoutes(mux *http.ServeMux, ph *api.ProjectHandlers, ch *api.CaptureHandlers) {
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ph.ListProjects(w, r)
		case http.MethodPost:
			ph.CreateProject(w, r)
		default:
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/stats") && r.Method == http.MethodGet:
			ph.GetProjectStats(w, r)
		case strings.HasSuffix(path, "/export-token") && r.Method == http.MethodPost:
			ph.GenerateExportToken(w, r)
		case strings.HasSuffix(path, "/captures/export") && r.Method == http.MethodGet:
			ch.ExportCaptures(w, r)
		case strings.Contains(path, "/captures") && r.Method == http.MethodGet:
			ch.ListCaptures(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				ph.GetProject(w, r)
			case http.MethodPut:
				ph.UpdateProject(w, r)
			case http.MethodDelete:
				ph.DeleteProject(w, r)
			default:
				methodNotAllowed(w)
			}
		}
	})
}

func registerAdminRoutes(mux *http.ServeMux, h *api.AdminHandlers) {
	mux.HandleFunc("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListProviders(w, r)
		case http.MethodPost:
			h.CreateProvider(w, r)
		default:
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/providers/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/check") && r.Method == http.MethodPost {
			h.CheckProvider(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.GetProvider(w, r)
		case http.MethodPut:
			h.UpdateProvider(w, r)
		case http.MethodDelete:
			h.DeleteProvider(w, r)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/routes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListRoutes(w, r)
		case http.MethodPost:
			h.CreateRoute(w, r)
		default:
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/routes/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/targets") {
			switch r.Method {
			case http.MethodGet:
				h.ListTargets(w, r)
			case http.MethodPost:
				h.CreateTarget(w, r)
			default:
				methodNotAllowed(w)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.GetRoute(w, r)
		case http.MethodPut:
			h.UpdateRoute(w, r)
		case http.MethodDelete:
			h.DeleteRoute(w, r)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/route-targets/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			h.UpdateTarget(w, r)
		case http.MethodDelete:
			h.DeleteTarget(w, r)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/api-keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListAPIKeys(w, r)
		case http.MethodPost:
			h.CreateAPIKey(w, r)
		default:
			methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/api-keys/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/disable") && r.Method == http.MethodPost {
			h.DisableAPIKey(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/enable") && r.Method == http.MethodPost {
			h.EnableAPIKey(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			h.DeleteAPIKey(w, r)
			return
		}
		methodNotAllowed(w)
	})

	mux.HandleFunc("/api/request-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ListLogs(w, r)
			return
		}
		methodNotAllowed(w)
	})
	mux.HandleFunc("/api/request-logs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetLog(w, r)
			return
		}
		methodNotAllowed(w)
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(405)
	w.Write([]byte(`{"error":"method not allowed"}`))
}
