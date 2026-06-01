package capture

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store persists capture metadata to SQLite.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) InsertBatch(records []CaptureRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO request_captures
			(request_id, project_id, public_model_name, stream, status_code, body_skipped,
			 file_path, file_offset, request_size, response_size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.Exec(r.RequestID, r.ProjectID, r.PublicModelName,
			r.Stream, r.StatusCode, r.BodySkipped,
			r.FilePath, r.FileOffset, r.RequestSize, r.ResponseSize); err != nil {
			return fmt.Errorf("exec insert: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) ListAfterCursor(projectID int64, cursor int64, pageSize int, filter CaptureFilter) ([]CaptureRecord, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "project_id = ?")
	args = append(args, projectID)

	if cursor > 0 {
		conditions = append(conditions, "id > ?")
		args = append(args, cursor)
	}

	if filter.Since != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339))
	}
	if filter.Until != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.Until.UTC().Format(time.RFC3339))
	}
	if filter.Stream != nil {
		conditions = append(conditions, "stream = ?")
		if *filter.Stream {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.Model != "" {
		conditions = append(conditions, "public_model_name = ?")
		args = append(args, filter.Model)
	}
	if filter.StatusCode != nil {
		conditions = append(conditions, "status_code = ?")
		args = append(args, *filter.StatusCode)
	}

	where := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT id, request_id, project_id, public_model_name, stream, status_code,
		       body_skipped, file_path, file_offset, request_size, response_size, created_at
		FROM request_captures
		WHERE %s
		ORDER BY id ASC
		LIMIT ?
	`, where)
	args = append(args, pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRecords(rows)
}

func (s *Store) CountByProject(projectID int64) (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM request_captures WHERE project_id = ?`, projectID).Scan(&count)
	return count, err
}

func (s *Store) CountByProjectFiltered(projectID int64, filter CaptureFilter) (int64, error) {
	var conditions []string
	var args []interface{}
	conditions = append(conditions, "project_id = ?")
	args = append(args, projectID)

	if filter.Since != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339))
	}
	if filter.Until != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.Until.UTC().Format(time.RFC3339))
	}
	if filter.Stream != nil {
		conditions = append(conditions, "stream = ?")
		if *filter.Stream {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.Model != "" {
		conditions = append(conditions, "public_model_name = ?")
		args = append(args, filter.Model)
	}
	if filter.StatusCode != nil {
		conditions = append(conditions, "status_code = ?")
		args = append(args, *filter.StatusCode)
	}

	where := strings.Join(conditions, " AND ")
	var count int64
	err := s.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM request_captures WHERE %s`, where), args...).Scan(&count)
	return count, err
}

func (s *Store) DeleteByProjectBefore(projectID int64, before time.Time) ([]string, int64, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT file_path FROM request_captures
		WHERE project_id = ? AND created_at < ?
	`, projectID, before.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var filePaths []string
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			return nil, 0, err
		}
		filePaths = append(filePaths, fp)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	res, err := s.db.Exec(`
		DELETE FROM request_captures WHERE project_id = ? AND created_at < ?
	`, projectID, before.UTC().Format(time.RFC3339))
	if err != nil {
		return filePaths, 0, err
	}
	deleted, _ := res.RowsAffected()
	return filePaths, deleted, nil
}

func scanRecords(rows *sql.Rows) ([]CaptureRecord, error) {
	var rs []CaptureRecord
	for rows.Next() {
		var r CaptureRecord
		if err := rows.Scan(&r.ID, &r.RequestID, &r.ProjectID, &r.PublicModelName,
			&r.Stream, &r.StatusCode, &r.BodySkipped, &r.FilePath, &r.FileOffset,
			&r.RequestSize, &r.ResponseSize, &r.CreatedAt); err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, rows.Err()
}
