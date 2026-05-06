-- Fix soft-delete + UNIQUE conflict: rebuild providers and routes tables
-- to use partial unique indexes (WHERE deleted_at IS NULL) instead of
-- column-level UNIQUE constraints that block re-creating soft-deleted names.

-- 1. Rebuild providers table without column-level UNIQUE on name
CREATE TABLE providers_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  description TEXT,
  provider_type TEXT NOT NULL DEFAULT 'openai_compatible',
  engine TEXT NOT NULL DEFAULT 'unknown',
  base_url TEXT NOT NULL,
  auth_mode TEXT NOT NULL DEFAULT 'none',
  upstream_api_key_encrypted TEXT NULL,
  custom_headers TEXT NULL,
  health_check_path TEXT NOT NULL DEFAULT '/models',
  health_status TEXT NOT NULL DEFAULT 'unknown',
  last_check_at DATETIME NULL,
  last_status_code INTEGER NULL,
  last_latency_ms INTEGER NULL,
  last_error TEXT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO providers_new SELECT * FROM providers;

DROP TABLE providers;

ALTER TABLE providers_new RENAME TO providers;

CREATE UNIQUE INDEX idx_providers_name_active ON providers(name) WHERE deleted_at IS NULL;

-- 2. Rebuild routes table without column-level UNIQUE on name and public_model_name
CREATE TABLE routes_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  public_model_name TEXT NOT NULL,
  description TEXT,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO routes_new SELECT * FROM routes;

DROP TABLE routes;

ALTER TABLE routes_new RENAME TO routes;

CREATE UNIQUE INDEX idx_routes_name_active ON routes(name) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_routes_model_active ON routes(public_model_name) WHERE deleted_at IS NULL;
