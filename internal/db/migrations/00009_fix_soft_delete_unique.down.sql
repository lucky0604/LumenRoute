-- Revert: drop partial indexes and rebuild with column-level UNIQUE

DROP INDEX IF EXISTS idx_providers_name_active;

DROP INDEX IF EXISTS idx_routes_name_active;

DROP INDEX IF EXISTS idx_routes_model_active;

-- Restore providers with column UNIQUE
CREATE TABLE providers_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
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

INSERT INTO providers_old SELECT * FROM providers;

DROP TABLE providers;

ALTER TABLE providers_old RENAME TO providers;

-- Restore routes with column UNIQUE
CREATE TABLE routes_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  public_model_name TEXT NOT NULL UNIQUE,
  description TEXT,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO routes_old SELECT * FROM routes;

DROP TABLE routes;

ALTER TABLE routes_old RENAME TO routes;
