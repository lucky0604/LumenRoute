CREATE TABLE route_targets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  route_id INTEGER NOT NULL,
  provider_id INTEGER NOT NULL,
  upstream_model_name TEXT NOT NULL,
  weight INTEGER NOT NULL DEFAULT 100,
  timeout_seconds INTEGER NOT NULL DEFAULT 120,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  FOREIGN KEY(route_id) REFERENCES routes(id) ON DELETE RESTRICT,
  FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE RESTRICT
);
