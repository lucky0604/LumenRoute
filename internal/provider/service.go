package provider

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Provider struct {
	ID                      int64            `json:"id"`
	Name                    string           `json:"name"`
	Description             string           `json:"description"`
	ProviderType            string           `json:"provider_type"`
	Engine                  string           `json:"engine"`
	BaseURL                 string           `json:"base_url"`
	AuthMode                string           `json:"auth_mode"`
	CustomHeaders           string           `json:"-"`
	HealthCheckPath         string           `json:"health_check_path"`
	HealthStatus            string           `json:"health_status"`
	LastCheckAt             *time.Time       `json:"last_check_at"`
	LastStatusCode          *int             `json:"last_status_code"`
	LastLatencyMs           *int             `json:"last_latency_ms"`
	LastError               string           `json:"last_error"`
	Enabled                 bool             `json:"enabled"`
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(p Provider) (int64, error) {
	r, err := s.db.Exec(`
		INSERT INTO providers (name, description, provider_type, engine, base_url, auth_mode, custom_headers, health_check_path, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, p.Name, p.Description, p.ProviderType, p.Engine, p.BaseURL, p.AuthMode, p.CustomHeaders, p.HealthCheckPath, p.Enabled)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (s *Service) List() ([]Provider, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, provider_type, engine, base_url, auth_mode, health_check_path, health_status,
		       last_check_at, last_status_code, last_latency_ms, last_error, enabled, created_at, updated_at
		FROM providers WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (s *Service) Get(id int64) (*Provider, error) {
	var p Provider
	var desc, lastErr sql.NullString
	var lastCheck sql.NullTime
	var lastCode, lastLat sql.NullInt64
	if err := s.db.QueryRow(`
		SELECT id, name, description, provider_type, engine, base_url, auth_mode, health_check_path, health_status,
		       last_check_at, last_status_code, last_latency_ms, last_error, enabled, created_at, updated_at
		FROM providers WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.Name, &desc, &p.ProviderType, &p.Engine, &p.BaseURL, &p.AuthMode,
		&p.HealthCheckPath, &p.HealthStatus, &lastCheck, &lastCode, &lastLat, &lastErr,
		&p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if desc.Valid { p.Description = desc.String }
	if lastCheck.Valid { p.LastCheckAt = &lastCheck.Time }
	if lastCode.Valid { v := int(lastCode.Int64); p.LastStatusCode = &v }
	if lastLat.Valid { v := int(lastLat.Int64); p.LastLatencyMs = &v }
	if lastErr.Valid { p.LastError = lastErr.String }
	return &p, nil
}

func (s *Service) Update(id int64, p Provider) error {
	_, err := s.db.Exec(`
		UPDATE providers SET name=?, description=?, provider_type=?, engine=?, base_url=?, auth_mode=?,
		custom_headers=?, health_check_path=?, enabled=?, updated_at=datetime('now')
		WHERE id=? AND deleted_at IS NULL
	`, p.Name, p.Description, p.ProviderType, p.Engine, p.BaseURL, p.AuthMode,
		p.CustomHeaders, p.HealthCheckPath, p.Enabled, id)
	return err
}

func (s *Service) Delete(id int64) error {
	_, err := s.db.Exec(`UPDATE providers SET deleted_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

func (s *Service) UpdateHealth(id int64, status string, statusCode int, latencyMs int, lastError string) error {
	_, err := s.db.Exec(`
		UPDATE providers SET health_status=?, last_status_code=?, last_latency_ms=?, last_error=?,
		last_check_at=datetime('now'), updated_at=datetime('now')
		WHERE id=?
	`, status, statusCode, latencyMs, lastError, id)
	return err
}

func scanProviders(rows *sql.Rows) ([]Provider, error) {
	var ps []Provider
	for rows.Next() {
		var p Provider
		var desc, lastErr sql.NullString
		var lastCheck sql.NullTime
		var lastCode, lastLat sql.NullInt64
		if err := rows.Scan(&p.ID, &p.Name, &desc, &p.ProviderType, &p.Engine, &p.BaseURL, &p.AuthMode,
			&p.HealthCheckPath, &p.HealthStatus, &lastCheck, &lastCode, &lastLat, &lastErr, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid { p.Description = desc.String }
		if lastCheck.Valid { p.LastCheckAt = &lastCheck.Time }
		if lastCode.Valid { v := int(lastCode.Int64); p.LastStatusCode = &v }
		if lastLat.Valid { v := int(lastLat.Int64); p.LastLatencyMs = &v }
		if lastErr.Valid { p.LastError = lastErr.String }
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

type CreateProviderRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ProviderType    string `json:"provider_type"`
	Engine          string `json:"engine"`
	BaseURL         string `json:"base_url"`
	AuthMode        string `json:"auth_mode"`
	CustomHeaders   json.RawMessage `json:"custom_headers"`
	HealthCheckPath string `json:"health_check_path"`
	Enabled         bool   `json:"enabled"`
}
