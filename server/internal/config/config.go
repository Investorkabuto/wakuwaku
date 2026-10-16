// Package config は環境変数からサーバー設定を読み込む。
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	// Env は "local" / "dev" / "test" / "prod"。
	Env         string
	Port        string
	DatabaseURL string
	// FirebaseProjectID が空かつ Env=local の場合は開発用認証を使う。
	FirebaseProjectID string
	AllowedOrigins    []string
	Version           string
}

func Load() (Config, error) {
	c := Config{
		Env:               getenv("APP_ENV", "local"),
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
		AllowedOrigins:    splitList(getenv("ALLOWED_ORIGINS", "http://localhost:5173")),
		Version:           getenv("APP_VERSION", "dev"),
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if c.Env != "local" && c.FirebaseProjectID == "" {
		return c, errors.New("FIREBASE_PROJECT_ID is required outside local")
	}
	return c, nil
}

// UseDevAuth は開発用認証（"dev:<uid>" トークン）を許可するか。local 以外では常に false。
func (c Config) UseDevAuth() bool {
	return c.Env == "local" && c.FirebaseProjectID == ""
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
