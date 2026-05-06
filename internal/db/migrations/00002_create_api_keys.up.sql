CREATE TABLE api_keys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  description TEXT,
  key_hash TEXT NOT NULL UNIQUE,
  key_prefix TEXT NOT NULL,
  allowed_route_ids TEXT NOT NULL DEFAULT 'all',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  expires_at DATETIME NULL,
  last_used_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
