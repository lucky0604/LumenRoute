package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

const bootstrapFile = "data/bootstrap-admin-password"

func BootstrapAdmin(db *sql.DB, username string, envPassword string) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("check existing users: %w", err)
	}
	if count > 0 {
		return nil
	}

	password := envPassword
	if password == "" {
		pw, err := generatePassword()
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		password = pw
		if err := os.MkdirAll(filepath.Dir(bootstrapFile), 0700); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
		if err := os.WriteFile(bootstrapFile, []byte(password+"\n"), 0600); err != nil {
			return fmt.Errorf("write bootstrap file: %w", err)
		}
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if username == "" {
		username = "admin"
	}

	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hash)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}
	return nil
}

func generatePassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
