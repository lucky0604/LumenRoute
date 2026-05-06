package route

import (
	"database/sql"
	"math/rand"
	"time"
)

type Route struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	PublicModelName string    `json:"public_model_name"`
	Description     string    `json:"description"`
	Enabled         bool      `json:"enabled"`
	RequireAuth     bool      `json:"require_auth"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RouteTarget struct {
	ID                int64     `json:"id"`
	RouteID           int64     `json:"route_id"`
	ProviderID        int64     `json:"provider_id"`
	UpstreamModelName string    `json:"upstream_model_name"`
	Weight            int       `json:"weight"`
	TimeoutSeconds    int       `json:"timeout_seconds"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ProviderName         string    `json:"provider_name,omitempty"`
	ProviderEnabled      bool      `json:"provider_enabled,omitempty"`
	ProviderHealthy      bool      `json:"provider_healthy,omitempty"`
	ProviderHealthStatus string    `json:"provider_health_status,omitempty"`
	ProviderBaseURL      string    `json:"-"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateRoute(r Route) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO routes (name, description, public_model_name, enabled, require_auth, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, r.Name, r.Description, r.PublicModelName, r.Enabled, r.RequireAuth)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Service) ListRoutes() ([]Route, error) {
	rows, err := s.db.Query(`
		SELECT id, name, public_model_name, description, enabled, require_auth, created_at, updated_at
		FROM routes WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rs []Route
	for rows.Next() {
		var r Route
		var desc sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.PublicModelName, &desc, &r.Enabled, &r.RequireAuth, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid { r.Description = desc.String }
		rs = append(rs, r)
	}
	return rs, rows.Err()
}

func (s *Service) GetRoute(id int64) (*Route, error) {
	var r Route
	var desc sql.NullString
	if err := s.db.QueryRow(`
		SELECT id, name, public_model_name, description, enabled, require_auth, created_at, updated_at
		FROM routes WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&r.ID, &r.Name, &r.PublicModelName, &desc, &r.Enabled, &r.RequireAuth, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	if desc.Valid { r.Description = desc.String }
	return &r, nil
}

func (s *Service) FindByModelName(modelName string) (*Route, error) {
	var r Route
	var desc sql.NullString
	if err := s.db.QueryRow(`
		SELECT id, name, public_model_name, description, enabled, require_auth, created_at, updated_at
		FROM routes WHERE public_model_name = ? AND deleted_at IS NULL
	`, modelName).Scan(&r.ID, &r.Name, &r.PublicModelName, &desc, &r.Enabled, &r.RequireAuth, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	if desc.Valid { r.Description = desc.String }
	return &r, nil
}

func (s *Service) UpdateRoute(id int64, r Route) error {
	_, err := s.db.Exec(`
		UPDATE routes SET name=?, description=?, enabled=?, require_auth=?, updated_at=datetime('now')
		WHERE id=? AND deleted_at IS NULL
	`, r.Name, r.Description, r.Enabled, r.RequireAuth, id)
	return err
}

func (s *Service) DeleteRoute(id int64) error {
	_, err := s.db.Exec(`UPDATE routes SET deleted_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

// RouteTarget operations

func (s *Service) CreateTarget(t RouteTarget) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO route_targets (route_id, provider_id, upstream_model_name, weight, timeout_seconds, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, t.RouteID, t.ProviderID, t.UpstreamModelName, t.Weight, t.TimeoutSeconds, t.Enabled)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Service) ListTargets(routeID int64) ([]RouteTarget, error) {
	rows, err := s.db.Query(`
		SELECT rt.id, rt.route_id, rt.provider_id, rt.upstream_model_name, rt.weight, rt.timeout_seconds,
		       rt.enabled, rt.created_at, rt.updated_at,
		       COALESCE(p.enabled, 0), COALESCE(p.health_status='healthy', 0),
		       COALESCE(p.name, ''), COALESCE(p.health_status, 'unknown')
		FROM route_targets rt
		LEFT JOIN providers p ON p.id = rt.provider_id AND p.deleted_at IS NULL
		WHERE rt.route_id = ? AND rt.deleted_at IS NULL
		ORDER BY rt.created_at DESC
	`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ts []RouteTarget
	for rows.Next() {
		var t RouteTarget
		if err := rows.Scan(&t.ID, &t.RouteID, &t.ProviderID, &t.UpstreamModelName, &t.Weight, &t.TimeoutSeconds,
			&t.Enabled, &t.CreatedAt, &t.UpdatedAt, &t.ProviderEnabled, &t.ProviderHealthy,
			&t.ProviderName, &t.ProviderHealthStatus); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}

func (s *Service) GetTarget(id int64) (*RouteTarget, error) {
	var t RouteTarget
	if err := s.db.QueryRow(`
		SELECT rt.id, rt.route_id, rt.provider_id, rt.upstream_model_name, rt.weight, rt.timeout_seconds,
		       rt.enabled, rt.created_at, rt.updated_at,
		       COALESCE(p.enabled, 0), COALESCE(p.health_status='healthy', 0),
		       COALESCE(p.name, ''), COALESCE(p.health_status, 'unknown')
		FROM route_targets rt
		LEFT JOIN providers p ON p.id = rt.provider_id AND p.deleted_at IS NULL
		WHERE rt.id = ? AND rt.deleted_at IS NULL
	`, id).Scan(&t.ID, &t.RouteID, &t.ProviderID, &t.UpstreamModelName, &t.Weight, &t.TimeoutSeconds,
		&t.Enabled, &t.CreatedAt, &t.UpdatedAt, &t.ProviderEnabled, &t.ProviderHealthy,
		&t.ProviderName, &t.ProviderHealthStatus); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) UpdateTarget(id int64, t RouteTarget) error {
	_, err := s.db.Exec(`
		UPDATE route_targets SET provider_id=?, upstream_model_name=?, weight=?, timeout_seconds=?, enabled=?, updated_at=datetime('now')
		WHERE id=? AND deleted_at IS NULL
	`, t.ProviderID, t.UpstreamModelName, t.Weight, t.TimeoutSeconds, t.Enabled, id)
	return err
}

func (s *Service) DeleteTarget(id int64) error {
	_, err := s.db.Exec(`UPDATE route_targets SET deleted_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

func (s *Service) GetReadyTargets(routeID int64) ([]RouteTarget, error) {
	rows, err := s.db.Query(`
		SELECT rt.id, rt.route_id, rt.provider_id, rt.upstream_model_name, rt.weight, rt.timeout_seconds,
		       rt.enabled, rt.created_at, rt.updated_at,
		       p.base_url
		FROM route_targets rt
		INNER JOIN providers p ON p.id = rt.provider_id
		WHERE rt.route_id = ? AND rt.enabled = 1 AND p.enabled = 1 AND p.health_status = 'healthy'
		  AND rt.deleted_at IS NULL AND p.deleted_at IS NULL
	`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ts []RouteTarget
	for rows.Next() {
		var t RouteTarget
		if err := rows.Scan(&t.ID, &t.RouteID, &t.ProviderID, &t.UpstreamModelName, &t.Weight, &t.TimeoutSeconds,
			&t.Enabled, &t.CreatedAt, &t.UpdatedAt, &t.ProviderBaseURL); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}

func (s *Service) SelectTarget(routeID int64) (*RouteTarget, error) {
	targets, err := s.GetReadyTargets(routeID)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(targets) == 1 {
		return &targets[0], nil
	}
	// Weighted random selection
	totalWeight := 0
	for _, t := range targets {
		totalWeight += t.Weight
	}
	pick := rand.Intn(totalWeight)
	running := 0
	for _, t := range targets {
		running += t.Weight
		if pick < running {
			return &t, nil
		}
	}
	return &targets[0], nil
}
