package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesConfigFileAndEnvOverrides(t *testing.T) {
	configPath := writeTempConfig(t, `databaseUrl: "postgres://from_file:password@localhost:5432/db?sslmode=disable"
app:
  addr: ":18080"
  cookieName: "file_session"
  cookieSecure: false
  sessionTtlHours: 12
admin:
  username: "file_admin"
  password: "file_admin_password_123"
  displayName: "文件管理员"
`)

	t.Setenv("APP_CONFIG_FILE", configPath)
	t.Setenv("APP_ADDR", ":28080")
	t.Setenv("APP_COOKIE_SECURE", "true")
	t.Setenv("ADMIN_USERNAME", "env_admin")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.DatabaseURL != "postgres://from_file:password@localhost:5432/db?sslmode=disable" {
		t.Fatalf("databaseUrl 未从配置文件读取: %q", cfg.DatabaseURL)
	}
	if cfg.Addr != ":28080" {
		t.Fatalf("环境变量 APP_ADDR 未覆盖配置文件: %q", cfg.Addr)
	}
	if cfg.CookieName != "file_session" {
		t.Fatalf("cookieName 未从配置文件读取: %q", cfg.CookieName)
	}
	if !cfg.CookieSecure {
		t.Fatal("环境变量 APP_COOKIE_SECURE 未覆盖配置文件")
	}
	if cfg.SessionTTL != 12*time.Hour {
		t.Fatalf("sessionTtlHours 未从配置文件读取: %s", cfg.SessionTTL)
	}
	if cfg.AdminUsername != "env_admin" {
		t.Fatalf("环境变量 ADMIN_USERNAME 未覆盖配置文件: %q", cfg.AdminUsername)
	}
	if cfg.AdminPassword != "file_admin_password_123" {
		t.Fatalf("admin.password 未从配置文件读取: %q", cfg.AdminPassword)
	}
	if cfg.AdminDisplayName != "文件管理员" {
		t.Fatalf("admin.displayName 未从配置文件读取: %q", cfg.AdminDisplayName)
	}
}

func TestLoadIgnoresMissingConfigFile(t *testing.T) {
	t.Setenv("APP_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("DATABASE_URL", "postgres://env:password@localhost:5432/db?sslmode=disable")
	t.Setenv("APP_COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("缺失配置文件时不应加载失败: %v", err)
	}

	if cfg.DatabaseURL != "postgres://env:password@localhost:5432/db?sslmode=disable" {
		t.Fatalf("未使用环境变量 DATABASE_URL: %q", cfg.DatabaseURL)
	}
	if cfg.CookieSecure {
		t.Fatal("未使用环境变量 APP_COOKIE_SECURE=false")
	}
}

func TestLoadFailsForInvalidConfigFile(t *testing.T) {
	configPath := writeTempConfig(t, "app:\n  cookieSecure: not-a-bool\n")
	t.Setenv("APP_CONFIG_FILE", configPath)
	t.Setenv("DATABASE_URL", "postgres://env:password@localhost:5432/db?sslmode=disable")

	_, err := Load()
	if err == nil {
		t.Fatal("格式错误的配置文件应加载失败")
	}
	if !strings.Contains(err.Error(), "配置文件") || !strings.Contains(err.Error(), "格式错误") {
		t.Fatalf("错误信息应包含中文配置文件格式提示，实际: %v", err)
	}
}

func TestConfigExampleCanLoad(t *testing.T) {
	examplePath := filepath.Join("..", "..", "config.example.yaml")
	t.Setenv("APP_CONFIG_FILE", examplePath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config.example.yaml 应可被加载: %v", err)
	}

	if cfg.DatabaseURL == "" || cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		t.Fatalf("config.example.yaml 缺少必要示例字段: %+v", cfg)
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}
	return path
}
