package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	ServerHost              string
	ServerPort              string
	DBDriver                string
	DBDSN                   string
	AdminUser               string
	AdminPassword           string
	APIKeyPrefix            string
	ProxyAuthMode           string
	SessionSecret           string
	RequestTimeoutSeconds   int
	HealthCheckIntervalSec  int
	RequestLogRetentionDays int
	MetricsEnabled          bool
	MetricsPath             string
	CaptureEnabled          bool
	CaptureMaxBodySize      int
	CaptureBasePath         string
}

func init() {
	loadDotEnv(".env.local")
	loadDotEnv(".env")
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"`)

		if os.Getenv(k) == "" && v != "" {
			os.Setenv(k, v)
		}
	}
	if err := s.Err(); err != nil {
		log.Printf("warning: reading %s: %v", path, err)
	}
}

func Load() *Config {
	return &Config{
		ServerHost:              env("LUMENROUTE_SERVER_HOST", "0.0.0.0"),
		ServerPort:              env("LUMENROUTE_SERVER_PORT", "8080"),
		DBDriver:                env("LUMENROUTE_DB_DRIVER", "sqlite"),
		DBDSN:                   env("LUMENROUTE_DB_DSN", "file:data/lumenroute.db?_foreign_keys=on&_journal_mode=WAL"),
		AdminUser:               env("LUMENROUTE_ADMIN_USER", "admin"),
		AdminPassword:           env("LUMENROUTE_ADMIN_PASSWORD", ""),
		APIKeyPrefix:             env("LUMENROUTE_API_KEY_PREFIX", "llmcp_"),
		ProxyAuthMode:            env("LUMENROUTE_PROXY_AUTH_MODE", "required"),
		SessionSecret:            env("LUMENROUTE_SESSION_SECRET", ""),
		RequestTimeoutSeconds:    120,
		HealthCheckIntervalSec:   30,
		RequestLogRetentionDays:  7,
		MetricsEnabled:           true,
		MetricsPath:              env("LUMENROUTE_METRICS_PATH", "/metrics"),
		CaptureEnabled:          env("LUMENROUTE_CAPTURE_ENABLED", "false") == "true",
		CaptureMaxBodySize:      envInt("LUMENROUTE_CAPTURE_MAX_BODY_SIZE", 1048576),
		CaptureBasePath:         env("LUMENROUTE_CAPTURE_BASE_PATH", "data/captures"),
	}
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
