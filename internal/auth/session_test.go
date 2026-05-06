package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lumenroute/internal/db"
)

func TestSessionLoginLogout(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := BootstrapAdmin(database, "admin", "testpass"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	sm := NewSessionManager(database, false, 24*time.Hour)

	// Login
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("username=admin&password=testpass"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := sm.Login(w, "admin", "testpass"); err != nil {
			t.Fatalf("login: %v", err)
		}
	})
	handler.ServeHTTP(w, r)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set")
	}
	cookie := cookies[0]
	if cookie.HttpOnly != true {
		t.Error("cookie not HttpOnly")
	}
	if cookie.Name != sessionCookieName {
		t.Errorf("cookie name = %s, want %s", cookie.Name, sessionCookieName)
	}

	// Logout
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/api/auth/logout", nil)
	r2.AddCookie(cookie)
	sm.Logout(w2, r2)
	cookies2 := w2.Result().Cookies()
	if len(cookies2) == 0 || cookies2[0].MaxAge >= 0 {
		t.Error("logout did not clear session cookie")
	}
}

func TestAdminAPIBypass(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := BootstrapAdmin(database, "admin", "testpass"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	sm := NewSessionManager(database, false, 24*time.Hour)
	protected := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/health", nil)
	protected.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Errorf("expected 401 without session, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("no error message in 401 response")
	}
}

func TestAuthEndpointBypassMiddleware(t *testing.T) {
	dsn := "file:" + t.TempDir() + "/test.db"

	if err := db.RunMigrations(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	sm := NewSessionManager(database, false, 24*time.Hour)
	protected := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/auth/login", nil)
	protected.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("auth endpoint should bypass middleware, got %d", w.Code)
	}
}
