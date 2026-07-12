package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"learn-liangliang/backend/internal/db/sqlc"
)

const contentSyncLockID int64 = 2026071201

type CatalogItem struct {
	PublicPath  string
	Title       string
	ContentType string
}

type CatalogCourse struct {
	PublicPath string
	Title      string
}

type CatalogCourseArticle struct {
	CoursePath  string
	ArticlePath string
	Position    int
}

type CatalogSync struct {
	Items          []CatalogItem
	Courses        []CatalogCourse
	CourseArticles []CatalogCourseArticle
}

type CourseSummary struct {
	PublicPath             string    `json:"coursePath"`
	Title                  string    `json:"title"`
	ArticleCount           int       `json:"articleCount"`
	LearnedArticleCount    int       `json:"learnedArticleCount"`
	AverageProgressPercent int       `json:"averageProgressPercent"`
	LatestReadAt           time.Time `json:"latestReadAt,omitempty"`
}

type CourseArticleProgress struct {
	ArticlePath     string     `json:"articlePath"`
	ArticleTitle    string     `json:"articleTitle"`
	Position        int        `json:"position"`
	ProgressPercent *int       `json:"progressPercent"`
	Finished        *bool      `json:"finished"`
	LastReadAt      *time.Time `json:"lastReadAt"`
}

func (s *Store) UpsertActiveReadingProgress(ctx context.Context, userID int64, articlePath string, articleTitle string, progressPercent int, scrollY int, finished bool) (ReadingProgress, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ReadingProgress{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockContentSync(ctx, tx); err != nil {
		return ReadingProgress{}, err
	}

	queries := s.Queries.WithTx(tx)
	item, err := queries.GetActiveContentItem(ctx, articlePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReadingProgress{}, ErrNotFound
	}
	if err != nil {
		return ReadingProgress{}, err
	}
	progress, err := queries.UpsertReadingProgress(ctx, sqlc.UpsertReadingProgressParams{
		UserID: userID, ArticlePath: articlePath, ArticleTitle: articleTitle,
		ProgressPercent: int32(progressPercent), ScrollY: int32(scrollY), Finished: finished,
	})
	if err != nil {
		return ReadingProgress{}, err
	}
	if item.ContentType == "course_article" {
		var courseID int64
		err = tx.QueryRow(ctx, `
SELECT course_articles.course_id
FROM course_articles
WHERE course_articles.content_item_id = $1 AND course_articles.is_active
`, item.ID).Scan(&courseID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return ReadingProgress{}, err
		}
		if err == nil {
			if err := queries.RefreshUserCourseProgress(ctx, sqlc.RefreshUserCourseProgressParams{UserID: userID, CourseID: courseID}); err != nil {
				return ReadingProgress{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ReadingProgress{}, err
	}
	return toReadingProgress(progress), nil
}

func (s *Store) ListCourseSummaries(ctx context.Context, userID int64) ([]CourseSummary, error) {
	rows, err := s.Queries.ListCourseSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]CourseSummary, 0, len(rows))
	for _, row := range rows {
		item := CourseSummary{
			PublicPath: row.PublicPath, Title: row.Title, ArticleCount: int(row.ArticleCount),
			LearnedArticleCount: int(row.LearnedArticleCount), AverageProgressPercent: int(row.AverageProgressPercent),
		}
		if row.LatestReadAt.Valid {
			item.LatestReadAt = row.LatestReadAt.Time
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) GetCourseSummary(ctx context.Context, userID int64, coursePath string) (CourseSummary, []CourseArticleProgress, error) {
	course, err := s.Queries.GetCourseSummary(ctx, sqlc.GetCourseSummaryParams{UserID: userID, PublicPath: coursePath})
	if errors.Is(err, pgx.ErrNoRows) {
		return CourseSummary{}, nil, ErrNotFound
	}
	if err != nil {
		return CourseSummary{}, nil, err
	}
	summary := CourseSummary{
		PublicPath: course.PublicPath, Title: course.Title, ArticleCount: int(course.ArticleCount),
		LearnedArticleCount: int(course.LearnedArticleCount), AverageProgressPercent: int(course.AverageProgressPercent),
	}
	if course.LatestReadAt.Valid {
		summary.LatestReadAt = course.LatestReadAt.Time
	}
	rows, err := s.Queries.ListCourseArticlesWithProgress(ctx, sqlc.ListCourseArticlesWithProgressParams{UserID: userID, CourseID: course.ID})
	if err != nil {
		return CourseSummary{}, nil, err
	}
	articles := make([]CourseArticleProgress, 0, len(rows))
	for _, row := range rows {
		item := CourseArticleProgress{ArticlePath: row.PublicPath, ArticleTitle: row.Title, Position: int(row.Position)}
		if row.ProgressPercent.Valid {
			value := int(row.ProgressPercent.Int32)
			item.ProgressPercent = &value
		}
		if row.Finished.Valid {
			value := row.Finished.Bool
			item.Finished = &value
		}
		if row.LastReadAt.Valid {
			value := row.LastReadAt.Time
			item.LastReadAt = &value
		}
		articles = append(articles, item)
	}
	return summary, articles, nil
}

func (s *Store) SyncCatalog(ctx context.Context, source CatalogSync) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockContentSync(ctx, tx); err != nil {
		return err
	}
	if err := createCatalogStaging(ctx, tx); err != nil {
		return err
	}
	if err := copyCatalogStaging(ctx, tx, source); err != nil {
		return err
	}
	if err := syncCatalogRows(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func lockContentSync(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", contentSyncLockID)
	return err
}

func createCatalogStaging(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
CREATE TEMP TABLE sync_items (public_path TEXT PRIMARY KEY, title TEXT NOT NULL, content_type TEXT NOT NULL) ON COMMIT DROP;
CREATE TEMP TABLE sync_courses (public_path TEXT PRIMARY KEY, title TEXT NOT NULL) ON COMMIT DROP;
CREATE TEMP TABLE sync_course_articles (course_path TEXT NOT NULL, article_path TEXT NOT NULL, position INT NOT NULL, PRIMARY KEY (course_path, article_path)) ON COMMIT DROP;
CREATE TEMP TABLE sync_known_courses (public_path TEXT PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE sync_affected_courses (course_id BIGINT PRIMARY KEY) ON COMMIT DROP;
INSERT INTO sync_known_courses (public_path) SELECT public_path FROM courses;
`)
	return err
}

func copyCatalogStaging(ctx context.Context, tx pgx.Tx, source CatalogSync) error {
	itemRows := make([][]any, 0, len(source.Items))
	for _, item := range source.Items {
		itemRows = append(itemRows, []any{item.PublicPath, item.Title, item.ContentType})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sync_items"}, []string{"public_path", "title", "content_type"}, pgx.CopyFromRows(itemRows)); err != nil {
		return fmt.Errorf("暂存内容索引失败: %w", err)
	}
	courseRows := make([][]any, 0, len(source.Courses))
	for _, course := range source.Courses {
		courseRows = append(courseRows, []any{course.PublicPath, course.Title})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sync_courses"}, []string{"public_path", "title"}, pgx.CopyFromRows(courseRows)); err != nil {
		return fmt.Errorf("暂存课程索引失败: %w", err)
	}
	articleRows := make([][]any, 0, len(source.CourseArticles))
	for _, article := range source.CourseArticles {
		articleRows = append(articleRows, []any{article.CoursePath, article.ArticlePath, article.Position})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"sync_course_articles"}, []string{"course_path", "article_path", "position"}, pgx.CopyFromRows(articleRows)); err != nil {
		return fmt.Errorf("暂存课程章节失败: %w", err)
	}
	return nil
}

func syncCatalogRows(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
INSERT INTO sync_affected_courses (course_id)
SELECT courses.id
FROM courses
LEFT JOIN sync_courses ON sync_courses.public_path = courses.public_path
WHERE sync_courses.public_path IS NULL
   OR courses.title <> sync_courses.title
   OR NOT courses.is_active
ON CONFLICT DO NOTHING;

INSERT INTO sync_affected_courses (course_id)
SELECT DISTINCT courses.id
FROM courses
JOIN course_articles ON course_articles.course_id = courses.id
JOIN content_items ON content_items.id = course_articles.content_item_id
LEFT JOIN sync_course_articles
  ON sync_course_articles.course_path = courses.public_path
 AND sync_course_articles.article_path = content_items.public_path
WHERE sync_course_articles.article_path IS NULL
   OR sync_course_articles.position <> course_articles.position
   OR NOT course_articles.is_active
   OR NOT content_items.is_active
ON CONFLICT DO NOTHING;

INSERT INTO sync_affected_courses (course_id)
SELECT DISTINCT courses.id
FROM sync_course_articles
JOIN courses ON courses.public_path = sync_course_articles.course_path
LEFT JOIN course_articles
  ON course_articles.course_id = courses.id
 AND course_articles.content_item_id = (
     SELECT id FROM content_items WHERE public_path = sync_course_articles.article_path
 )
WHERE course_articles.content_item_id IS NULL
   OR course_articles.position <> sync_course_articles.position
   OR NOT course_articles.is_active
ON CONFLICT DO NOTHING;

UPDATE content_items SET is_active = FALSE, updated_at = NOW() WHERE is_active;
INSERT INTO content_items (public_path, title, content_type, is_active, updated_at)
SELECT public_path, title, content_type, TRUE, NOW() FROM sync_items
ON CONFLICT (public_path) DO UPDATE SET
    title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    is_active = TRUE,
    updated_at = NOW();

UPDATE courses SET is_active = FALSE, article_count = 0, updated_at = NOW() WHERE is_active;
INSERT INTO courses (public_path, title, is_active, updated_at)
SELECT public_path, title, TRUE, NOW() FROM sync_courses
ON CONFLICT (public_path) DO UPDATE SET
    title = EXCLUDED.title,
    is_active = TRUE,
    updated_at = NOW();

INSERT INTO sync_affected_courses (course_id)
SELECT courses.id
FROM courses
JOIN sync_courses ON sync_courses.public_path = courses.public_path
LEFT JOIN sync_known_courses ON sync_known_courses.public_path = courses.public_path
WHERE sync_known_courses.public_path IS NULL
ON CONFLICT DO NOTHING;

UPDATE course_articles SET is_active = FALSE, updated_at = NOW() WHERE is_active;
INSERT INTO course_articles (course_id, content_item_id, position, is_active, updated_at)
SELECT courses.id, content_items.id, sync_course_articles.position, TRUE, NOW()
FROM sync_course_articles
JOIN courses ON courses.public_path = sync_course_articles.course_path
JOIN content_items ON content_items.public_path = sync_course_articles.article_path
ON CONFLICT (course_id, content_item_id) DO UPDATE SET
    position = EXCLUDED.position,
    is_active = TRUE,
    updated_at = NOW();

UPDATE courses
SET article_count = course_article_counts.article_count, updated_at = NOW()
FROM (
    SELECT courses.id, COUNT(course_articles.content_item_id)::int AS article_count
    FROM courses
    LEFT JOIN course_articles ON course_articles.course_id = courses.id AND course_articles.is_active
    LEFT JOIN content_items ON content_items.id = course_articles.content_item_id AND content_items.is_active
    GROUP BY courses.id
) AS course_article_counts
WHERE courses.id = course_article_counts.id;

DELETE FROM user_course_progress
WHERE course_id IN (SELECT course_id FROM sync_affected_courses);
`)
	if err != nil {
		return fmt.Errorf("同步内容目录失败: %w", err)
	}
	queries := sqlc.New(tx)
	rows, err := tx.Query(ctx, "SELECT course_id FROM sync_affected_courses")
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := queries.CreateCourseProgressForCourses(ctx, ids); err != nil {
			return fmt.Errorf("重算课程进度失败: %w", err)
		}
	}
	return nil
}
