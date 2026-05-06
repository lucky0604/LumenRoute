CREATE TABLE routes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  public_model_name TEXT NOT NULL UNIQUE,
  description TEXT,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  deleted_at DATETIME NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
