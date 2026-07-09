-- name: UpsertReadingProgress :one
INSERT INTO reading_progress (user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (user_id, article_path)
DO UPDATE SET
    article_title = EXCLUDED.article_title,
    progress_percent = EXCLUDED.progress_percent,
    scroll_y = EXCLUDED.scroll_y,
    finished = EXCLUDED.finished,
    last_read_at = NOW(),
    updated_at = NOW()
RETURNING id, user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, created_at, updated_at;

-- name: GetReadingProgress :one
SELECT id, user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, created_at, updated_at
FROM reading_progress
WHERE user_id = $1 AND article_path = $2;

-- name: ListRecentReadingProgress :many
SELECT id, user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, created_at, updated_at
FROM reading_progress
WHERE user_id = $1
ORDER BY last_read_at DESC
LIMIT $2;
