package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"learn-liangliang/backend/internal/auth"
	"learn-liangliang/backend/internal/config"
	"learn-liangliang/backend/internal/db"
	"learn-liangliang/backend/internal/reading"
	"learn-liangliang/backend/internal/response"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("连接数据库失败", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(ctx, "migrations"); err != nil {
		slog.Error("执行数据库迁移失败", "error", err)
		os.Exit(1)
	}
	if err := auth.EnsureAdmin(ctx, store, cfg); err != nil {
		slog.Error("预置管理员账号失败", "error", err)
		os.Exit(1)
	}

	router := buildRouter(store, cfg)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("API 服务已启动", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("API 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("关闭 API 服务失败", "error", err)
		os.Exit(1)
	}
	slog.Info("API 服务已关闭")
}

func buildRouter(store *db.Store, cfg config.Config) http.Handler {
	authHandler := auth.NewHandler(store, cfg)
	readingHandler := reading.NewHandler(store)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(authHandler.AttachUser)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Get("/me", authHandler.Me)
		r.Post("/logout", authHandler.Logout)
	})

	r.Group(func(r chi.Router) {
		r.Use(authHandler.RequireAuth)
		r.Put("/api/reading-progress", readingHandler.Upsert)
		r.Get("/api/reading-progress", readingHandler.Get)
		r.Get("/api/reading-progress/recent", readingHandler.Recent)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
	})

	return r
}
