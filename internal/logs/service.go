package logs

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

type RequestLog struct {
	ID                 int64     `json:"id"`
	RequestID          string    `json:"request_id"`
	APIKeyID           *int64    `json:"api_key_id"`
	RouteID            *int64    `json:"route_id"`
	RouteName          string    `json:"route_name"`
	ProviderID         *int64    `json:"provider_id"`
	ProviderName       string    `json:"provider_name"`
	TargetID           *int64    `json:"target_id"`
	PublicModelName    string    `json:"public_model_name"`
	UpstreamModelName  string    `json:"upstream_model_name"`
	ClientIP           string    `json:"client_ip"`
	Method             string    `json:"method"`
	Path               string    `json:"path"`
	Stream             bool      `json:"stream"`
	StatusCode         int       `json:"status_code"`
	UpstreamStatusCode int       `json:"upstream_status_code"`
	LatencyMs          int       `json:"latency_ms"`
	TimeToFirstChunkMs *int      `json:"time_to_first_chunk_ms"`
	StreamCompleted    *bool     `json:"stream_completed"`
	PromptTokens       *int      `json:"prompt_tokens"`
	CompletionTokens   *int      `json:"completion_tokens"`
	TotalTokens        *int      `json:"total_tokens"`
	ErrorCode          string    `json:"error_code"`
	ErrorMessage       string    `json:"error_message"`
	CreatedAt          time.Time `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Write(log RequestLog) error {
	_, err := s.db.Exec(`
		INSERT INTO request_logs (request_id, api_key_id, route_id, route_name, provider_id, provider_name,
		target_id, public_model_name, upstream_model_name, client_ip, method, path, stream,
		status_code, upstream_status_code, latency_ms, time_to_first_chunk_ms, stream_completed,
		prompt_tokens, completion_tokens, total_tokens, error_code, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, log.RequestID, log.APIKeyID, log.RouteID, log.RouteName, log.ProviderID, log.ProviderName,
		log.TargetID, log.PublicModelName, log.UpstreamModelName, log.ClientIP, log.Method, log.Path, log.Stream,
		log.StatusCode, log.UpstreamStatusCode, log.LatencyMs, log.TimeToFirstChunkMs, log.StreamCompleted,
		log.PromptTokens, log.CompletionTokens, log.TotalTokens, log.ErrorCode, log.ErrorMessage)
	return err
}

func (s *Service) List(filter LogFilter) ([]RequestLog, error) {
	query := `SELECT id, request_id, api_key_id, route_id, route_name, provider_id, provider_name,
		target_id, public_model_name, upstream_model_name, client_ip, method, path, stream,
		status_code, upstream_status_code, latency_ms, time_to_first_chunk_ms, stream_completed,
		prompt_tokens, completion_tokens, total_tokens, error_code, error_message, created_at
		FROM request_logs WHERE 1=1`
	var args []interface{}

	if filter.Model != "" {
		query += " AND (public_model_name LIKE ? OR upstream_model_name LIKE ?)"
		args = append(args, "%"+filter.Model+"%", "%"+filter.Model+"%")
	}
	if filter.Provider != "" {
		query += " AND provider_name LIKE ?"
		args = append(args, "%"+filter.Provider+"%")
	}
	if filter.StatusCode > 0 {
		query += " AND status_code = ?"
		args = append(args, filter.StatusCode)
	}
	if filter.Stream != nil {
		query += " AND stream = ?"
		args = append(args, *filter.Stream)
	}
	if filter.ErrorOnly {
		query += " AND error_code IS NOT NULL AND error_code != ''"
	}
	if filter.RequestID != "" {
		query += " AND request_id = ?"
		args = append(args, filter.RequestID)
	}
	if filter.TargetID != nil {
		query += " AND target_id = ?"
		args = append(args, *filter.TargetID)
	}
	if filter.RouteID != nil {
		query += " AND route_id = ?"
		args = append(args, *filter.RouteID)
	}
	if filter.Since != nil {
		query += " AND created_at >= ?"
		args = append(args, filter.Since.Format("2006-01-02 15:04:05"))
	}
	if filter.MinLatencyMs != nil {
		query += " AND latency_ms >= ?"
		args = append(args, *filter.MinLatencyMs)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (s *Service) Get(id int64) (*RequestLog, error) {
	row := s.db.QueryRow(`SELECT id, request_id, api_key_id, route_id, route_name, provider_id, provider_name,
		target_id, public_model_name, upstream_model_name, client_ip, method, path, stream,
		status_code, upstream_status_code, latency_ms, time_to_first_chunk_ms, stream_completed,
		prompt_tokens, completion_tokens, total_tokens, error_code, error_message, created_at
		FROM request_logs WHERE id = ?`, id)
	var log RequestLog
	if err := scanLogRow(row, &log); err != nil {
		return nil, err
	}
	return &log, nil
}

func GenerateRequestID(clientID string) string {
	if clientID != "" {
		return clientID
	}
	return fmt.Sprintf("req_%x", rand.Uint64())
}

type LogFilter struct {
	Model        string
	Provider     string
	StatusCode   int
	Stream       *bool
	ErrorOnly    bool
	RequestID    string
	TargetID     *int64
	RouteID      *int64
	Since        *time.Time
	MinLatencyMs *int
	Limit        int
}

func scanLogRow(row *sql.Row, log *RequestLog) error {
	var apiKeyID, routeID, providerID, targetID sql.NullInt64
	var routeName, providerName, clientIP, method, path, publicModel, upstreamModel sql.NullString
	var upstreamSC sql.NullInt64
	var ttfc sql.NullInt64
	var streamComp sql.NullBool
	var promptT, compT, totalT sql.NullInt64
	var errorCode, errorMsg sql.NullString

	if err := row.Scan(&log.ID, &log.RequestID, &apiKeyID, &routeID, &routeName, &providerID, &providerName,
		&targetID, &publicModel, &upstreamModel, &clientIP, &method, &path, &log.Stream,
		&log.StatusCode, &upstreamSC, &log.LatencyMs, &ttfc, &streamComp,
		&promptT, &compT, &totalT, &errorCode, &errorMsg, &log.CreatedAt); err != nil {
		return err
	}
	if apiKeyID.Valid { v := apiKeyID.Int64; log.APIKeyID = &v }
	if routeID.Valid { v := routeID.Int64; log.RouteID = &v }
	if routeName.Valid { log.RouteName = routeName.String }
	if providerID.Valid { v := providerID.Int64; log.ProviderID = &v }
	if providerName.Valid { log.ProviderName = providerName.String }
	if targetID.Valid { v := targetID.Int64; log.TargetID = &v }
	if publicModel.Valid { log.PublicModelName = publicModel.String }
	if upstreamModel.Valid { log.UpstreamModelName = upstreamModel.String }
	if clientIP.Valid { log.ClientIP = clientIP.String }
	if method.Valid { log.Method = method.String }
	if path.Valid { log.Path = path.String }
	if upstreamSC.Valid { log.UpstreamStatusCode = int(upstreamSC.Int64) }
	if ttfc.Valid { v := int(ttfc.Int64); log.TimeToFirstChunkMs = &v }
	if streamComp.Valid { log.StreamCompleted = &streamComp.Bool }
	if promptT.Valid { v := int(promptT.Int64); log.PromptTokens = &v }
	if compT.Valid { v := int(compT.Int64); log.CompletionTokens = &v }
	if totalT.Valid { v := int(totalT.Int64); log.TotalTokens = &v }
	if errorCode.Valid { log.ErrorCode = errorCode.String }
	if errorMsg.Valid { log.ErrorMessage = errorMsg.String }
	return nil
}

func scanLogs(rows *sql.Rows) ([]RequestLog, error) {
	var logs []RequestLog
	for rows.Next() {
		var log RequestLog
		var apiKeyID, routeID, providerID, targetID sql.NullInt64
		var routeName, providerName, clientIP, method, path, publicModel, upstreamModel sql.NullString
		var upstreamSC sql.NullInt64
		var ttfc sql.NullInt64
		var streamComp sql.NullBool
		var promptT, compT, totalT sql.NullInt64
		var errorCode, errorMsg sql.NullString

		if err := rows.Scan(&log.ID, &log.RequestID, &apiKeyID, &routeID, &routeName, &providerID, &providerName,
			&targetID, &publicModel, &upstreamModel, &clientIP, &method, &path, &log.Stream,
			&log.StatusCode, &upstreamSC, &log.LatencyMs, &ttfc, &streamComp,
			&promptT, &compT, &totalT, &errorCode, &errorMsg, &log.CreatedAt); err != nil {
			return nil, err
		}
		if apiKeyID.Valid { v := apiKeyID.Int64; log.APIKeyID = &v }
		if routeID.Valid { v := routeID.Int64; log.RouteID = &v }
		if routeName.Valid { log.RouteName = routeName.String }
		if providerID.Valid { v := providerID.Int64; log.ProviderID = &v }
		if providerName.Valid { log.ProviderName = providerName.String }
		if targetID.Valid { v := targetID.Int64; log.TargetID = &v }
		if publicModel.Valid { log.PublicModelName = publicModel.String }
		if upstreamModel.Valid { log.UpstreamModelName = upstreamModel.String }
		if clientIP.Valid { log.ClientIP = clientIP.String }
		if method.Valid { log.Method = method.String }
		if path.Valid { log.Path = path.String }
		if upstreamSC.Valid { log.UpstreamStatusCode = int(upstreamSC.Int64) }
		if ttfc.Valid { v := int(ttfc.Int64); log.TimeToFirstChunkMs = &v }
		if streamComp.Valid { log.StreamCompleted = &streamComp.Bool }
		if promptT.Valid { v := int(promptT.Int64); log.PromptTokens = &v }
		if compT.Valid { v := int(compT.Int64); log.CompletionTokens = &v }
		if totalT.Valid { v := int(totalT.Int64); log.TotalTokens = &v }
		if errorCode.Valid { log.ErrorCode = errorCode.String }
		if errorMsg.Valid { log.ErrorMessage = errorMsg.String }
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
