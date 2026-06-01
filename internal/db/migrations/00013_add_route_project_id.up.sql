ALTER TABLE routes ADD COLUMN project_id INTEGER REFERENCES projects(id);
CREATE INDEX idx_routes_project_id ON routes(project_id)
