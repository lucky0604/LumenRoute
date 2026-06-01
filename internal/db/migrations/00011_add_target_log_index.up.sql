CREATE INDEX IF NOT EXISTS idx_request_logs_target_created
    ON request_logs(target_id, created_at);
