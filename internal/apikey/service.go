package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type APIKey struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	KeyPrefix       string     `json:"key_prefix"`
	KeyHash         string     `json:"-"`
	RawKey          string     `json:"raw_key,omitempty"`
	AllowedRouteIDs string     `json:"allowed_route_ids"`
	Enabled         bool       `json:"enabled"`
	ExpiresAt       *time.Time `json:"expires_at"`
	LastUsedAt      *time.Time `json:"last_used_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Service struct {
	db     *sql.DB
	prefix string
}

func NewService(db *sql.DB, prefix string) *Service {
	if prefix == "" {
		prefix = "llmcp_"
	}
	return &Service{db: db, prefix: prefix}
}

func (s *Service) Create(name, description string, allowedRouteIDs string, expiresAt *time.Time) (*APIKey, error) {
	rawKey := s.prefix + randomHex(32)
	keyHash := hashKey(rawKey)
	keyPrefix := rawKey[:12]

	now := time.Now().UTC()
	res, err := s.db.Exec(`
		INSERT INTO api_keys (name, description, key_hash, key_prefix, allowed_route_ids, enabled, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)
	`, name, description, keyHash, keyPrefix, allowedRouteIDs, expiresAt, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	id, _ := res.LastInsertId()

	return &APIKey{
		ID:              id,
		Name:            name,
		Description:     description,
		KeyPrefix:       keyPrefix,
		RawKey:          rawKey,
		AllowedRouteIDs: allowedRouteIDs,
		Enabled:         true,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *Service) List() ([]APIKey, error) {
	return s.query("SELECT id, name, description, key_prefix, allowed_route_ids, enabled, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE deleted_at IS NULL ORDER BY created_at DESC")
}

func (s *Service) Get(id int64) (*APIKey, error) {
	keys, err := s.query("SELECT id, name, description, key_prefix, allowed_route_ids, enabled, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, sql.ErrNoRows
	}
	return &keys[0], nil
}

func (s *Service) Delete(id int64) error {
	_, err := s.db.Exec(`UPDATE api_keys SET deleted_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

func (s *Service) Disable(id int64) error {
	_, err := s.db.Exec(`UPDATE api_keys SET enabled=0, updated_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

func (s *Service) Enable(id int64) error {
	_, err := s.db.Exec(`UPDATE api_keys SET enabled=1, updated_at=datetime('now') WHERE id=? AND deleted_at IS NULL`, id)
	return err
}

func (s *Service) ValidateKey(rawKey string) (*APIKey, error) {
	keyHash := hashKey(rawKey)
	keys, err := s.query("SELECT id, name, description, key_prefix, allowed_route_ids, enabled, expires_at, last_used_at, created_at, updated_at FROM api_keys WHERE key_hash = ? AND deleted_at IS NULL", keyHash)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("invalid api key")
	}
	k := &keys[0]
	if !k.Enabled {
		return nil, fmt.Errorf("api key disabled")
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return nil, fmt.Errorf("api key expired")
	}
	s.db.Exec(`UPDATE api_keys SET last_used_at=datetime('now') WHERE id=?`, k.ID)
	return k, nil
}

func (s *Service) IsModelAllowed(key *APIKey, publicModelName string, routeID int64) bool {
	if key.AllowedRouteIDs == "" {
		return true
	}
	var policy struct {
		Type     string  `json:"type"`
		RouteIDs []int64 `json:"route_ids"`
	}
	if err := json.Unmarshal([]byte(key.AllowedRouteIDs), &policy); err != nil {
		return false
	}
	if policy.Type == "all" {
		return true
	}
	for _, rid := range policy.RouteIDs {
		if rid == routeID {
			return true
		}
	}
	return false
}

func (s *Service) ProxyAuthMiddleware(authMode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authMode == "disabled" {
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				if authMode == "optional" {
					next.ServeHTTP(w, r)
					return
				}
				writeProxyError(w, 401, "invalid_api_key", "Missing API key")
				return
			}
			rawKey := strings.TrimPrefix(authHeader, "Bearer ")
			key, err := s.ValidateKey(rawKey)
			if err != nil {
				if authMode == "optional" {
					next.ServeHTTP(w, r)
					return
				}
				writeProxyError(w, 401, "invalid_api_key", "Invalid API key")
				return
			}
			_ = key
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) query(q string, args ...interface{}) ([]APIKey, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var desc, routeIDs sql.NullString
		var expires, lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.Name, &desc, &k.KeyPrefix, &routeIDs, &k.Enabled,
			&expires, &lastUsed, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid { k.Description = desc.String }
		if routeIDs.Valid { k.AllowedRouteIDs = routeIDs.String }
		if expires.Valid { k.ExpiresAt = &expires.Time }
		if lastUsed.Valid { k.LastUsedAt = &lastUsed.Time }
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func writeProxyError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"message": msg, "type": "authentication_error", "code": code},
	})
}
