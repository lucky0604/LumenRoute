CREATE INDEX idx_request_logs_request_id ON request_logs(request_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX idx_request_logs_route_created ON request_logs(route_id, created_at);
CREATE INDEX idx_request_logs_provider_created ON request_logs(provider_id, created_at);
CREATE INDEX idx_request_logs_status_created ON request_logs(status_code, created_at);
