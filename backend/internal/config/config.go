package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	DatabaseURL      string
	CookieName       string
	CookieSecure     bool
	SessionTTL       time.Duration
	AdminUsername    string
	AdminPassword    string
	AdminDisplayName string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:             getEnv("APP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		CookieName:       getEnv("APP_COOKIE_NAME", "learn_session"),
		CookieSecure:     getEnvBool("APP_COOKIE_SECURE", true),
		SessionTTL:       time.Duration(getEnvInt("APP_SESSION_TTL_HOURS", 24*30)) * time.Hour,
		AdminUsername:    strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword:    os.Getenv("ADMIN_PASSWORD"),
		AdminDisplayName: getEnv("ADMIN_DISPLAY_NAME", "管理员"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL 不能为空")
	}
	if strings.Contains(cfg.DatabaseURL, "请替换") {
		return Config{}, fmt.Errorf("DATABASE_URL 不能包含示例占位符")
	}
	if cfg.AdminUsername != "" {
		if cfg.AdminPassword == "" {
			return Config{}, fmt.Errorf("ADMIN_PASSWORD 不能为空")
		}
		if strings.Contains(cfg.AdminPassword, "请替换") || len(cfg.AdminPassword) < 12 {
			return Config{}, fmt.Errorf("ADMIN_PASSWORD 必须替换为至少 12 位强密码")
		}
	}
	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
