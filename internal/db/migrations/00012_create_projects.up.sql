CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    data_category TEXT NOT NULL DEFAULT 'mixed',
    capture_enabled INTEGER NOT NULL DEFAULT 0,
    sample_rate REAL NOT NULL DEFAULT 0.0,
    retention_days INTEGER NOT NULL DEFAULT 30,
    export_token_hash TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    deleted_at DATETIME
);
CREATE UNIQUE INDEX idx_projects_name ON projects(name) WHERE deleted_at IS NULL
