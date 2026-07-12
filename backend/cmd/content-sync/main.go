package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"learn-liangliang/backend/internal/catalog"
	"learn-liangliang/backend/internal/config"
	"learn-liangliang/backend/internal/db"
)

func main() {
	contentRoot := flag.String("content-root", "../content", "内容物理根目录")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	root, err := filepath.Abs(*contentRoot)
	if err != nil {
		logger.Error("解析内容目录失败", "error", err)
		os.Exit(1)
	}
	source, err := catalog.BuildSync(root)
	if err != nil {
		logger.Error("扫描内容目录失败", "root", root, "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("连接数据库失败", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx, "migrations"); err != nil {
		logger.Error("执行数据库迁移失败", "error", err)
		os.Exit(1)
	}
	if err := store.SyncCatalog(ctx, toDBCatalogSync(source)); err != nil {
		logger.Error("同步内容目录失败", "error", err)
		os.Exit(1)
	}
	logger.Info("内容目录同步完成", "contentRoot", root, "items", len(source.Items), "courses", len(source.Courses), "courseArticles", len(source.CourseArticles))
}

func toDBCatalogSync(source catalog.SyncSource) db.CatalogSync {
	items := make([]db.CatalogItem, 0, len(source.Items))
	for _, item := range source.Items {
		items = append(items, db.CatalogItem{PublicPath: item.PublicPath, Title: item.Title, ContentType: string(item.ContentType)})
	}
	courses := make([]db.CatalogCourse, 0, len(source.Courses))
	for _, course := range source.Courses {
		courses = append(courses, db.CatalogCourse{PublicPath: course.PublicPath, Title: course.Title})
	}
	articles := make([]db.CatalogCourseArticle, 0, len(source.CourseArticles))
	for _, article := range source.CourseArticles {
		articles = append(articles, db.CatalogCourseArticle{CoursePath: article.CoursePath, ArticlePath: article.ArticlePath, Position: article.Position})
	}
	return db.CatalogSync{Items: items, Courses: courses, CourseArticles: articles}
}
