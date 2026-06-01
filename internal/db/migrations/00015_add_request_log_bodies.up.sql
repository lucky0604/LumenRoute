ALTER TABLE request_logs ADD COLUMN request_body TEXT NOT NULL DEFAULT '';
ALTER TABLE request_logs ADD COLUMN response_body TEXT NOT NULL DEFAULT '';
