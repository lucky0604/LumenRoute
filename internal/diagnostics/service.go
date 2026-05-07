package diagnostics

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lumenroute/internal/provider"
	"lumenroute/internal/route"
)

var ErrNotFound = errors.New("target not found")

type Service struct {
	db       *sql.DB
	route    *route.Service
	provider *provider.Service
}

func NewService(db *sql.DB, rs *route.Service, ps *provider.Service) *Service {
	return &Service{db: db, route: rs, provider: ps}
}

func (s *Service) GetModelOverview(window string) ([]ModelSummary, error) {
	since, err := parseWindow(window)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT
			COALESCE(target_id, 0),
			COALESCE(public_model_name, ''),
			COALESCE(upstream_model_name, ''),
			COALESCE(provider_name, ''),
			COUNT(*),
			SUM(CASE WHEN error_code IS NOT NULL AND error_code != '' THEN 1 ELSE 0 END),
			COALESCE(AVG(latency_ms), 0),
			COALESCE(SUM(COALESCE(prompt_tokens, 0) + COALESCE(completion_tokens, 0)), 0),
			SUM(CASE WHEN stream = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN stream = 1 AND stream_completed = 1 THEN 1 ELSE 0 END),
			COALESCE(MAX(created_at), '')
		FROM request_logs
		WHERE created_at >= ?
		GROUP BY target_id, public_model_name, upstream_model_name, provider_name
		ORDER BY SUM(CASE WHEN error_code IS NOT NULL AND error_code != '' THEN 1 ELSE 0 END) DESC,
		         AVG(latency_ms) DESC
	`, since.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("query model overview: %w", err)
	}
	defer rows.Close()

	var summaries []ModelSummary
	for rows.Next() {
		var ms ModelSummary
		var errCount int
		var avgLat float64
		var totalToks int
		var streamCount, streamCompleted int
		if err := rows.Scan(
			&ms.TargetID, &ms.PublicModelName, &ms.UpstreamModelName, &ms.ProviderName,
			&ms.RequestCount, &errCount, &avgLat, &totalToks,
			&streamCount, &streamCompleted,
			&ms.LastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("scan model overview: %w", err)
		}
		ms.ErrorCount = errCount
		ms.AvgLatencyMs = avgLat
		ms.TotalTokens = totalToks
		ms.StreamCount = streamCount
		if ms.RequestCount > 0 {
			ms.ErrorRate = float64(errCount) / float64(ms.RequestCount)
		}
		if streamCount > 0 {
			ms.StreamCompletedRate = float64(streamCompleted) / float64(streamCount)
		}
		ms.P95LatencyMs = computeP95(s.db, ms.TargetID, ms.PublicModelName, since)
		ms.LastErrorCode = queryLastErrorCode(s.db, ms.TargetID, ms.PublicModelName, since)
		summaries = append(summaries, ms)
	}
	if summaries == nil {
		summaries = []ModelSummary{}
	}
	return summaries, rows.Err()
}

func (s *Service) GetTargetDiagnosis(targetID int64, window string) (*TargetDiagnosis, error) {
	since, err := parseWindow(window)
	if err != nil {
		return nil, err
	}

	target, err := s.route.GetTarget(targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get target: %w", err)
	}

	rt, _ := s.route.GetRoute(target.RouteID)

	facts := TargetFacts{
		ID:                target.ID,
		RouteID:           target.RouteID,
		UpstreamModelName: target.UpstreamModelName,
		ProviderName:      target.ProviderName,
		ProviderHealth:    target.ProviderHealthStatus,
		Enabled:           target.Enabled,
	}
	if rt != nil {
		facts.RouteName = rt.Name
		facts.PublicModelName = rt.PublicModelName
	}

	prov, err := s.provider.Get(target.ProviderID)
	if err != nil {
		facts.ProviderHealth = "deleted"
	} else {
		facts.ProviderBaseURL = prov.BaseURL
		facts.ProviderEngine = prov.Engine
		facts.ProviderHealth = prov.HealthStatus
		if prov.LastCheckAt != nil {
			facts.LastCheckAt = prov.LastCheckAt.Format(time.RFC3339)
		}
		facts.LastError = prov.LastError
	}

	summary := s.queryTargetSummary(targetID, since)
	summary.PublicModelName = facts.PublicModelName
	summary.UpstreamModelName = facts.UpstreamModelName
	summary.ProviderName = facts.ProviderName
	summary.TargetID = targetID

	recentFailures := s.queryRecentFailures(targetID, since, 20)
	slowRequests := s.querySlowRequests(targetID, since, 20)

	modelsCurl := ""
	if facts.ProviderBaseURL != "" {
		modelsCurl = fmt.Sprintf("curl %s/v1/models", facts.ProviderBaseURL)
	}

	return &TargetDiagnosis{
		Target:           facts,
		Summary:          summary,
		RecentFailures:   recentFailures,
		SlowRequests:     slowRequests,
		OperatorCommands: OperatorCommands{ModelsCurl: modelsCurl},
	}, nil
}

func (s *Service) queryTargetSummary(targetID int64, since time.Time) ModelSummary {
	var ms ModelSummary
	var errCount, streamCount, streamCompleted int
	var avgLat float64
	var totalToks int
	sinceStr := since.Format("2006-01-02 15:04:05")

	err := s.db.QueryRow(`
		SELECT
			COUNT(*),
			SUM(CASE WHEN error_code IS NOT NULL AND error_code != '' THEN 1 ELSE 0 END),
			COALESCE(AVG(latency_ms), 0),
			COALESCE(SUM(COALESCE(prompt_tokens, 0) + COALESCE(completion_tokens, 0)), 0),
			SUM(CASE WHEN stream = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN stream = 1 AND stream_completed = 1 THEN 1 ELSE 0 END),
			COALESCE(MAX(created_at), '')
		FROM request_logs
		WHERE target_id = ? AND created_at >= ?
	`, targetID, sinceStr).Scan(
		&ms.RequestCount, &errCount, &avgLat, &totalToks,
		&streamCount, &streamCompleted,
		&ms.LastSeenAt,
	)
	if err != nil {
		return ms
	}
	ms.ErrorCount = errCount
	ms.AvgLatencyMs = avgLat
	ms.TotalTokens = totalToks
	ms.StreamCount = streamCount
	if ms.RequestCount > 0 {
		ms.ErrorRate = float64(errCount) / float64(ms.RequestCount)
	}
	if streamCount > 0 {
		ms.StreamCompletedRate = float64(streamCompleted) / float64(streamCount)
	}
	ms.P95LatencyMs = computeP95(s.db, targetID, "", since)
	ms.LastErrorCode = queryLastErrorCode(s.db, targetID, "", since)
	return ms
}

func queryLastErrorCode(db *sql.DB, targetID int64, publicModel string, since time.Time) string {
	var code sql.NullString
	sinceStr := since.Format("2006-01-02 15:04:05")
	if targetID > 0 {
		db.QueryRow(`
			SELECT error_code FROM request_logs
			WHERE target_id = ? AND created_at >= ?
			  AND error_code IS NOT NULL AND error_code != ''
			ORDER BY created_at DESC LIMIT 1
		`, targetID, sinceStr).Scan(&code)
	} else if publicModel != "" {
		db.QueryRow(`
			SELECT error_code FROM request_logs
			WHERE public_model_name = ? AND created_at >= ?
			  AND error_code IS NOT NULL AND error_code != ''
			ORDER BY created_at DESC LIMIT 1
		`, publicModel, sinceStr).Scan(&code)
	}
	if code.Valid {
		return code.String
	}
	return ""
}

func (s *Service) queryRecentFailures(targetID int64, since time.Time, limit int) []RequestLogBrief {
	rows, err := s.db.Query(`
		SELECT id, status_code, upstream_status_code,
			COALESCE(error_code, ''), COALESCE(error_message, ''),
			latency_ms, stream_completed, COALESCE(public_model_name, ''), created_at
		FROM request_logs
		WHERE target_id = ? AND created_at >= ?
		  AND error_code IS NOT NULL AND error_code != ''
		ORDER BY created_at DESC
		LIMIT ?
	`, targetID, since.Format("2006-01-02 15:04:05"), limit)
	if err != nil {
		return []RequestLogBrief{}
	}
	defer rows.Close()
	return scanBriefRows(rows)
}

func (s *Service) querySlowRequests(targetID int64, since time.Time, limit int) []RequestLogBrief {
	rows, err := s.db.Query(`
		SELECT id, status_code, upstream_status_code,
			COALESCE(error_code, ''), COALESCE(error_message, ''),
			latency_ms, stream_completed, COALESCE(public_model_name, ''), created_at
		FROM request_logs
		WHERE target_id = ? AND created_at >= ?
		ORDER BY latency_ms DESC
		LIMIT ?
	`, targetID, since.Format("2006-01-02 15:04:05"), limit)
	if err != nil {
		return []RequestLogBrief{}
	}
	defer rows.Close()
	return scanBriefRows(rows)
}

func scanBriefRows(rows *sql.Rows) []RequestLogBrief {
	var results []RequestLogBrief
	for rows.Next() {
		var b RequestLogBrief
		var sc sql.NullBool
		if err := rows.Scan(
			&b.ID, &b.StatusCode, &b.UpstreamStatusCode,
			&b.ErrorCode, &b.ErrorMessage,
			&b.LatencyMs, &sc, &b.PublicModelName, &b.CreatedAt,
		); err != nil {
			continue
		}
		if sc.Valid {
			b.StreamCompleted = &sc.Bool
		}
		results = append(results, b)
	}
	if results == nil {
		results = []RequestLogBrief{}
	}
	return results
}
