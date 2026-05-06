package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.ServerPort != "8080" {
		t.Errorf("default port = %s, want 8080", cfg.ServerPort)
	}
	if cfg.AdminUser != "admin" {
		t.Errorf("default admin user = %s, want admin", cfg.AdminUser)
	}
	if cfg.ProxyAuthMode != "required" {
		t.Errorf("default proxy auth mode = %s, want required", cfg.ProxyAuthMode)
	}
	if cfg.APIKeyPrefix != "llmcp_" {
		t.Errorf("default api key prefix = %s, want llmcp_", cfg.APIKeyPrefix)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("LUMENROUTE_SERVER_PORT", "9090")
	t.Setenv("LUMENROUTE_PROXY_AUTH_MODE", "optional")
	cfg := Load()
	if cfg.ServerPort != "9090" {
		t.Errorf("env port = %s, want 9090", cfg.ServerPort)
	}
	if cfg.ProxyAuthMode != "optional" {
		t.Errorf("env proxy auth mode = %s, want optional", cfg.ProxyAuthMode)
	}
}
