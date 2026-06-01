package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lumenroute/internal/auth"
	"lumenroute/internal/capture"
	"lumenroute/internal/project"
)

// CaptureHandlers holds service references for capture-related HTTP handlers.
type CaptureHandlers struct {
	CaptureStore   *capture.Store
	Projects       *project.Service
	SessionManager *auth.SessionManager
	CaptureBase    string
}

// ListCaptures handles GET /api/projects/{id}/captures (session-authed)
func (h *CaptureHandlers) ListCaptures(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := h.parseCaptureFilter(r)
	records, err := h.CaptureStore.ListAfterCursor(projectID, filter.Cursor, filter.PageSize, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []capture.CaptureRecord{}
	}

	total, _ := h.CaptureStore.CountByProjectFiltered(projectID, filter)

	var nextCursor *int64
	if len(records) == filter.PageSize {
		last := records[len(records)-1].ID
		nextCursor = &last
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        records,
		"total":       total,
		"next_cursor": nextCursor,
	})
}

// ExportCaptures handles GET /api/projects/{id}/captures/export (dual auth)
// Supports both session cookie and Bearer export token authentication.
// Pass count_only=true to get a size estimate without streaming data.
func (h *CaptureHandlers) ExportCaptures(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r.URL.Path, "/api/projects/")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !h.checkExportAuth(w, r, projectID) {
		return
	}

	if r.URL.Query().Get("count_only") == "true" {
		filter := h.parseCaptureFilter(r)
		total, _ := h.CaptureStore.CountByProjectFiltered(projectID, filter)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"total":      total,
			"project_id": projectID,
		})
		return
	}

	filter := h.parseCaptureFilter(r)
	if filter.PageSize == 0 || filter.PageSize > 1000 {
		filter.PageSize = 100
	}
	download := r.URL.Query().Get("download") == "true"
	if download {
		filter.PageSize = 500
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jsonl"
	}

	if download {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=captures_%d_%s.jsonl",
				projectID, time.Now().Format("20060102_150405")))
	} else {
		w.Header().Set("Content-Type", "application/x-ndjson")
	}
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)
	cursor := filter.Cursor
	totalExported := 0

	for {
		records, err := h.CaptureStore.ListAfterCursor(projectID, cursor, filter.PageSize, filter)
		if err != nil {
			break
		}
		if len(records) == 0 {
			break
		}
		for _, rec := range records {
			line := h.readCaptureLine(rec, format)
			if line == nil {
				capture.IncrExportSkipped()
				continue
			}
			w.Write(line)
			w.Write([]byte("\n"))
			totalExported++
			cursor = rec.ID
		}
		if canFlush {
			flusher.Flush()
		}
		if !download {
			break
		}
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"_meta":         true,
		"exported":      totalExported,
		"next_cursor":   cursor,
		"has_more":      !download,
		"exported_at":   time.Now().UTC().Format(time.RFC3339),
	})
	w.Write(meta)
	w.Write([]byte("\n"))
	if canFlush {
		flusher.Flush()
	}
}

func (h *CaptureHandlers) checkExportAuth(w http.ResponseWriter, r *http.Request, projectID int64) bool {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if err := h.Projects.ValidateExportToken(projectID, token); err == nil {
			return true
		}
	}

	if r.Method == http.MethodPost && h.SessionManager.HasValidSession(r) {
		return true
	}

	respondError(w, http.StatusUnauthorized, "authentication required: Bearer export token, or POST with session cookie")
	return false
}

func (h *CaptureHandlers) parseCaptureFilter(r *http.Request) capture.CaptureFilter {
	q := r.URL.Query()
	f := capture.CaptureFilter{
		Model:  q.Get("model"),
		Format: q.Get("format"),
	}

	if v := q.Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.PageSize = n
		}
	}
	if f.PageSize <= 0 {
		f.PageSize = 50
	}
	if f.PageSize > 1000 {
		f.PageSize = 1000
	}

	if v := q.Get("cursor"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Cursor = n
		}
	}

	if v := q.Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Since = &t
		}
	}
	if v := q.Get("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Until = &t
		}
	}
	if v := q.Get("stream"); v != "" {
		b := v == "true" || v == "1"
		f.Stream = &b
	}
	if v := q.Get("status_code"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.StatusCode = &n
		}
	}
	f.Download = q.Get("download") == "true"

	return f
}

func (h *CaptureHandlers) readCaptureLine(rec capture.CaptureRecord, format string) []byte {
	if rec.BodySkipped || rec.FilePath == "" {
		meta, _ := json.Marshal(map[string]interface{}{
			"request_id":  rec.RequestID,
			"project_id":  rec.ProjectID,
			"model":       rec.PublicModelName,
			"stream":      rec.Stream,
			"status_code": rec.StatusCode,
			"body_skipped": true,
			"created_at":  rec.CreatedAt,
		})
		return meta
	}

	fullPath := filepath.Join(h.CaptureBase, filepath.Clean(rec.FilePath))
	absBase, _ := filepath.Abs(h.CaptureBase)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
		return nil
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	if _, err := f.Seek(rec.FileOffset, 0); err != nil {
		return nil
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
	if !scanner.Scan() {
		return nil
	}

	line := scanner.Bytes()
	if format == "openai" {
		return transformToOpenAI(line)
	}
	result := make([]byte, len(line))
	copy(result, line)
	return result
}

func transformToOpenAI(line []byte) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return line
	}

	out := map[string]json.RawMessage{}
	for _, key := range []string{"request_body", "response_body"} {
		if v, ok := raw[key]; ok {
			if key == "request_body" {
				out["messages"] = extractMessages(v)
			} else {
				out["completion"] = extractCompletion(v)
			}
		}
	}
	if v, ok := raw["public_model_name"]; ok {
		out["model"] = v
	}
	result, _ := json.Marshal(out)
	return result
}

func extractMessages(body json.RawMessage) json.RawMessage {
	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}
	if msgs, ok := req["messages"]; ok {
		return msgs
	}
	return body
}

func extractCompletion(body json.RawMessage) json.RawMessage {
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(body, &resp); err != nil {
		return body
	}
	if choices, ok := resp["choices"]; ok {
		var choiceList []map[string]json.RawMessage
		if err := json.Unmarshal(choices, &choiceList); err == nil && len(choiceList) > 0 {
			if msg, ok := choiceList[0]["message"]; ok {
				var msgMap map[string]json.RawMessage
				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if content, ok := msgMap["content"]; ok {
						return content
					}
				}
			}
		}
	}
	return body
}
