package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultConfigFile = "config.yaml"

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

type fileConfig struct {
	DatabaseURL *string         `yaml:"databaseUrl"`
	App         fileAppConfig   `yaml:"app"`
	Admin       fileAdminConfig `yaml:"admin"`
}

type fileAppConfig struct {
	Addr            *string `yaml:"addr"`
	CookieName      *string `yaml:"cookieName"`
	CookieSecure    *bool   `yaml:"cookieSecure"`
	SessionTTLHours *int    `yaml:"sessionTtlHours"`
}

type fileAdminConfig struct {
	Username    *string `yaml:"username"`
	Password    *string `yaml:"password"`
	DisplayName *string `yaml:"displayName"`
}

func Load() (Config, error) {
	cfg := defaultConfig()
	if err := applyConfigFile(&cfg, configFilePath()); err != nil {
		return Config{}, err
	}
	applyEnv(&cfg)

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

func defaultConfig() Config {
	return Config{
		Addr:             ":8080",
		CookieName:       "learn_session",
		CookieSecure:     true,
		SessionTTL:       24 * 30 * time.Hour,
		AdminDisplayName: "管理员",
	}
}

func configFilePath() string {
	path := strings.TrimSpace(os.Getenv("APP_CONFIG_FILE"))
	if path == "" {
		return defaultConfigFile
	}
	return path
}

func applyConfigFile(cfg *Config, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var fileCfg fileConfig
	if err := yaml.Unmarshal(content, &fileCfg); err != nil {
		return fmt.Errorf("配置文件 %s 格式错误: %w", path, err)
	}

	if fileCfg.DatabaseURL != nil {
		cfg.DatabaseURL = strings.TrimSpace(*fileCfg.DatabaseURL)
	}
	if fileCfg.App.Addr != nil {
		cfg.Addr = strings.TrimSpace(*fileCfg.App.Addr)
	}
	if fileCfg.App.CookieName != nil {
		cfg.CookieName = strings.TrimSpace(*fileCfg.App.CookieName)
	}
	if fileCfg.App.CookieSecure != nil {
		cfg.CookieSecure = *fileCfg.App.CookieSecure
	}
	if fileCfg.App.SessionTTLHours != nil {
		cfg.SessionTTL = time.Duration(*fileCfg.App.SessionTTLHours) * time.Hour
	}
	if fileCfg.Admin.Username != nil {
		cfg.AdminUsername = strings.TrimSpace(*fileCfg.Admin.Username)
	}
	if fileCfg.Admin.Password != nil {
		cfg.AdminPassword = *fileCfg.Admin.Password
	}
	if fileCfg.Admin.DisplayName != nil {
		cfg.AdminDisplayName = strings.TrimSpace(*fileCfg.Admin.DisplayName)
	}
	return nil
}

func applyEnv(cfg *Config) {
	cfg.Addr = getEnv("APP_ADDR", cfg.Addr)
	cfg.DatabaseURL = strings.TrimSpace(getEnv("DATABASE_URL", cfg.DatabaseURL))
	cfg.CookieName = getEnv("APP_COOKIE_NAME", cfg.CookieName)
	cfg.CookieSecure = getEnvBool("APP_COOKIE_SECURE", cfg.CookieSecure)
	cfg.SessionTTL = time.Duration(getEnvInt("APP_SESSION_TTL_HOURS", int(cfg.SessionTTL/time.Hour))) * time.Hour
	cfg.AdminUsername = strings.TrimSpace(getEnv("ADMIN_USERNAME", cfg.AdminUsername))
	cfg.AdminPassword = getEnv("ADMIN_PASSWORD", cfg.AdminPassword)
	cfg.AdminDisplayName = getEnv("ADMIN_DISPLAY_NAME", cfg.AdminDisplayName)
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
