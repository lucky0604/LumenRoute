CREATE TABLE sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash TEXT NOT NULL UNIQUE,
  username TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  expires_at DATETIME NOT NULL
);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
