package auth

import (
	"context"
	"log/slog"

	"learn-liangliang/backend/internal/config"
	"learn-liangliang/backend/internal/db"
)

func EnsureAdmin(ctx context.Context, store *db.Store, cfg config.Config) error {
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		slog.Info("未配置 ADMIN_USERNAME / ADMIN_PASSWORD，跳过管理员账号预置")
		return nil
	}

	hash, err := HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	_, err = store.CreateUser(ctx, cfg.AdminUsername, hash, cfg.AdminDisplayName)
	if err != nil {
		return err
	}
	slog.Info("管理员账号已预置或更新", "username", cfg.AdminUsername)
	return nil
}
