package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "lumenroute_session"

type SessionManager struct {
	mu           sync.RWMutex
	db           *sql.DB
	cookieDomain string
	secure       bool
	maxAge       time.Duration
}

func NewSessionManager(database *sql.DB, secure bool, maxAge time.Duration) *SessionManager {
	return &SessionManager{
		db:     database,
		secure: secure,
		maxAge: maxAge,
	}
}

func (sm *SessionManager) Login(w http.ResponseWriter, username, password string) error {
	var hash string
	if err := sm.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash); err != nil {
		return fmt.Errorf("invalid credentials")
	}
	if !CheckPassword(hash, password) {
		return fmt.Errorf("invalid credentials")
	}

	token, err := generateToken()
	if err != nil {
		return err
	}
	tokenHash := hashToken(token)

	if _, err := sm.db.Exec("INSERT INTO sessions (token_hash, username, created_at, expires_at) VALUES (?, ?, ?, ?)",
		tokenHash, username, time.Now(), time.Now().Add(sm.maxAge)); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.maxAge.Seconds()),
	})
	return nil
}

func (sm *SessionManager) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		tokenHash := hashToken(cookie.Value)
		sm.db.Exec("DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (sm *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJ(w, 401, `{"error":"authentication required"}`)
			return
		}

		tokenHash := hashToken(cookie.Value)
		var expiresAt time.Time
		if err := sm.db.QueryRow("SELECT expires_at FROM sessions WHERE token_hash = ?", tokenHash).Scan(&expiresAt); err != nil {
			writeJ(w, 401, `{"error":"invalid session"}`)
			return
		}
		if time.Now().After(expiresAt) {
			sm.db.Exec("DELETE FROM sessions WHERE token_hash = ?", tokenHash)
			writeJ(w, 401, `{"error":"session expired"}`)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func writeJ(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(body))
}
