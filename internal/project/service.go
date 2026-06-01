package project

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(p Project) (int64, error) {
	if !ValidDataCategories[p.DataCategory] {
		p.DataCategory = "mixed"
	}
	if p.SampleRate < 0 {
		p.SampleRate = 0
	}
	if p.SampleRate > 1 {
		p.SampleRate = 1
	}
	if p.RetentionDays < 0 {
		p.RetentionDays = 30
	}

	res, err := s.db.Exec(`
		INSERT INTO projects (name, description, data_category, capture_enabled, sample_rate, retention_days, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, p.Name, p.Description, p.DataCategory, p.CaptureEnabled, p.SampleRate, p.RetentionDays)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Service) List() ([]Project, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, data_category, capture_enabled, sample_rate, retention_days,
		       COALESCE(export_token_hash, ''), created_at, updated_at
		FROM projects WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (s *Service) Get(id int64) (*Project, error) {
	var p Project
	var desc sql.NullString
	var tokenHash sql.NullString
	if err := s.db.QueryRow(`
		SELECT id, name, description, data_category, capture_enabled, sample_rate, retention_days,
		       export_token_hash, created_at, updated_at
		FROM projects WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.Name, &desc, &p.DataCategory, &p.CaptureEnabled,
		&p.SampleRate, &p.RetentionDays, &tokenHash, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if desc.Valid {
		p.Description = desc.String
	}
	if tokenHash.Valid {
		p.ExportTokenHash = tokenHash.String
	}
	return &p, nil
}

func (s *Service) Update(id int64, p Project) error {
	if !ValidDataCategories[p.DataCategory] {
		p.DataCategory = "mixed"
	}
	if p.SampleRate < 0 {
		p.SampleRate = 0
	}
	if p.SampleRate > 1 {
		p.SampleRate = 1
	}
	_, err := s.db.Exec(`
		UPDATE projects SET name=?, description=?, data_category=?, capture_enabled=?,
		       sample_rate=?, retention_days=?, updated_at=datetime('now')
		WHERE id=? AND deleted_at IS NULL
	`, p.Name, p.Description, p.DataCategory, p.CaptureEnabled, p.SampleRate, p.RetentionDays, id)
	return err
}

func (s *Service) Delete(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE routes SET project_id = NULL WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("disassociate routes: %w", err)
	}
	if _, err := tx.Exec(`UPDATE projects SET deleted_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("soft delete project: %w", err)
	}
	return tx.Commit()
}

func (s *Service) GenerateExportToken(projectID int64) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random: %w", err)
	}
	token := fmt.Sprintf("lrx_%d_%s", projectID, hex.EncodeToString(randomBytes))

	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash token: %w", err)
	}

	_, err = s.db.Exec(`UPDATE projects SET export_token_hash=?, updated_at=datetime('now') WHERE id=? AND deleted_at IS NULL`,
		string(hash), projectID)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) ValidateExportToken(projectID int64, token string) error {
	var hash sql.NullString
	if err := s.db.QueryRow(`
		SELECT export_token_hash FROM projects WHERE id = ? AND deleted_at IS NULL
	`, projectID).Scan(&hash); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	if !hash.Valid || hash.String == "" {
		return fmt.Errorf("no export token configured")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash.String), []byte(token))
}

func (s *Service) GetStats(projectID int64) (*Stats, error) {
	st := &Stats{ProjectID: projectID}

	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(request_size), 0), COALESCE(SUM(response_size), 0),
		       MIN(created_at), MAX(created_at)
		FROM request_captures WHERE project_id = ?
	`, projectID).Scan(&st.TotalCaptures, &st.TotalRequestSizeBytes,
		&st.TotalResponseSizeBytes, &st.EarliestCapture, &st.LatestCapture)
	if err != nil {
		return nil, err
	}

	s.db.QueryRow(`
		SELECT COUNT(*) FROM request_captures
		WHERE project_id = ? AND created_at >= date('now')
	`, projectID).Scan(&st.CapturesToday)

	s.db.QueryRow(`
		SELECT COUNT(*) FROM routes
		WHERE project_id = ? AND deleted_at IS NULL
	`, projectID).Scan(&st.RoutesCount)

	return st, nil
}

func (s *Service) CountAssociatedRoutes(projectID int64) (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM routes WHERE project_id = ? AND deleted_at IS NULL`, projectID).Scan(&count)
	return count, err
}

func (s *Service) ListWithRetention() ([]Project, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, data_category, capture_enabled, sample_rate, retention_days,
		       COALESCE(export_token_hash, ''), created_at, updated_at
		FROM projects WHERE deleted_at IS NULL AND retention_days > 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

func scanProjects(rows *sql.Rows) ([]Project, error) {
	var ps []Project
	for rows.Next() {
		var p Project
		var desc sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &desc, &p.DataCategory, &p.CaptureEnabled,
			&p.SampleRate, &p.RetentionDays, &p.ExportTokenHash, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			p.Description = desc.String
		}
		ps = append(ps, p)
	}
	return ps, rows.Err()
}

// HasExportToken returns whether a project has a configured export token.
func (s *Service) HasExportToken(projectID int64) (bool, error) {
	var hash sql.NullString
	err := s.db.QueryRow(`SELECT export_token_hash FROM projects WHERE id = ? AND deleted_at IS NULL`, projectID).Scan(&hash)
	if err != nil {
		return false, err
	}
	return hash.Valid && hash.String != "", nil
}

// CaptureRecordCount returns total capture count for a project,
// used by delete confirmation UI.
func (s *Service) CaptureRecordCount(projectID int64) (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM request_captures WHERE project_id = ?`, projectID).Scan(&count)
	return count, err
}

// RetentionExpiry calculates when capture data will be cleaned up after project deletion.
func RetentionExpiry(retentionDays int) time.Time {
	if retentionDays <= 0 {
		return time.Time{}
	}
	return time.Now().AddDate(0, 0, retentionDays)
}
