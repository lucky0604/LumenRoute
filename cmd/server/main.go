package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"lumenroute/internal/api"
	"lumenroute/internal/apikey"
	"lumenroute/internal/auth"
	"lumenroute/internal/config"
	"lumenroute/internal/db"
	"lumenroute/internal/diagnostics"
	"lumenroute/internal/logs"
	"lumenroute/internal/metrics"
	"lumenroute/internal/provider"
	"lumenroute/internal/proxy"
	"lumenroute/internal/route"
	"lumenroute/internal/scheduler"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(cfg.DBDSN); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := auth.BootstrapAdmin(database, cfg.AdminUser, cfg.AdminPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	providerSvc := provider.NewService(database)
	routeSvc := route.NewService(database)
	apiKeySvc := apikey.NewService(database, cfg.APIKeyPrefix)
	logsSvc := logs.NewService(database)
	metricsReg := metrics.NewRegistry()
	proxySvc := proxy.NewService(routeSvc, apiKeySvc, logsSvc, cfg.ProxyAuthMode, metricsReg)
	diagSvc := diagnostics.NewService(database, routeSvc, providerSvc)

	handlers := &api.AdminHandlers{
		Providers: providerSvc,
		Routes:    routeSvc,
		APIKeys:   apiKeySvc,
		Logs:      logsSvc,
	}

	diagHandler := &api.DiagnosticsHandler{Service: diagSvc}

	sessionManager := auth.NewSessionManager(database, cfg.ServerHost != "0.0.0.0", 24*time.Hour)
	mux := http.NewServeMux()

	// Public auth endpoints
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var username, password string
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"invalid request body"}`))
				return
			}
			username = body.Username
			password = body.Password
		} else {
			username = r.FormValue("username")
			password = r.FormValue("password")
		}
		if err := sessionManager.Login(w, username, password); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		sessionManager.Logout(w, r)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Admin CRUD endpoints (session-protected)
	adminMux := http.NewServeMux()
	registerAdminRoutes(adminMux, handlers)
	api.RegisterDiagnosticsRoutes(adminMux, diagHandler)
	adminHandler := sessionManager.Middleware(adminMux)
	mux.Handle("/api/", adminHandler)

	// Proxy endpoints (API key auth checked per-route)
	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/v1/models", proxySvc.ListModels)
	proxyMux.HandleFunc("/v1/chat/completions", proxySvc.ChatCompletions)
	mux.Handle("/v1/", proxyMux)

	// Metrics
	if cfg.MetricsEnabled {
		mux.Handle(cfg.MetricsPath, metricsReg.Handler())
	}

	// Static SPA frontend
	webDist := filepath.Join("web", "dist")
	if fi, err := os.Stat(webDist); err == nil && fi.IsDir() {
		fs := http.FileServer(http.Dir(webDist))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/v1") || strings.HasPrefix(r.URL.Path, "/metrics") {
				http.NotFound(w, r)
				return
			}
			fp := filepath.Join(webDist, filepath.Clean(r.URL.Path))
			if _, err := os.Stat(fp); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(webDist, "index.html"))
		})
	}

	// Start background schedulers
	quit := make(chan struct{})
	scheduler.StartHealthChecker(providerSvc, cfg.HealthCheckIntervalSec, quit, metricsReg)
	scheduler.StartLogCleanup(database, cfg.RequestLogRetentionDays, quit)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      mux,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("LumenRoute server starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-sig
	log.Println("shutting down server...")
	close(quit)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}

func registerAdminRoutes(mux *http.ServeMux, h *api.AdminHandlers) {
	// Providers
	mux.HandleFunc("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListProviders(w, r)
		case http.MethodPost:
			h.CreateProvider(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
		}
	})

	// Routes
	mux.HandleFunc("/api/routes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListRoutes(w, r)
		case http.MethodPost:
			h.CreateRoute(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(405)
				w.Write([]byte(`{"error":"method not allowed"}`))
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
		}
	})

	// Route Targets (direct)
	mux.HandleFunc("/api/route-targets/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			h.UpdateTarget(w, r)
		case http.MethodDelete:
			h.DeleteTarget(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
		}
	})

	// API Keys
	mux.HandleFunc("/api/api-keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListAPIKeys(w, r)
		case http.MethodPost:
			h.CreateAPIKey(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"method not allowed"}`))
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(405)
		w.Write([]byte(`{"error":"method not allowed"}`))
	})

	// Request Logs
	mux.HandleFunc("/api/request-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ListLogs(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(405)
		w.Write([]byte(`{"error":"method not allowed"}`))
	})
	mux.HandleFunc("/api/request-logs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetLog(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(405)
		w.Write([]byte(`{"error":"method not allowed"}`))
	})
}
