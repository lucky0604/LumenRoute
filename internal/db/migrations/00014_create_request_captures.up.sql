CREATE TABLE request_captures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id TEXT NOT NULL,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    public_model_name TEXT NOT NULL DEFAULT '',
    stream INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL DEFAULT 200,
    body_skipped INTEGER NOT NULL DEFAULT 0,
    file_path TEXT NOT NULL DEFAULT '',
    file_offset INTEGER NOT NULL DEFAULT 0,
    request_size INTEGER NOT NULL DEFAULT 0,
    response_size INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_captures_project_date ON request_captures(project_id, created_at);
CREATE INDEX idx_captures_request_id ON request_captures(request_id);
CREATE INDEX idx_captures_filter ON request_captures(project_id, public_model_name, stream, status_code)
