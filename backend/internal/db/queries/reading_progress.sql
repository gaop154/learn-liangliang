-- name: GetActiveContentItem :one
SELECT id, public_path, title, content_type
FROM content_items
WHERE public_path = $1
  AND is_active
  AND content_type IN ('course_article', 'article', 'geektime_article', 'love_course_article');

-- name: UpsertReadingProgress :one
INSERT INTO reading_progress (user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (user_id, article_path)
DO UPDATE SET
    article_title = EXCLUDED.article_title,
    progress_percent = GREATEST(reading_progress.progress_percent, EXCLUDED.progress_percent),
    scroll_y = GREATEST(reading_progress.scroll_y, EXCLUDED.scroll_y),
    finished = reading_progress.finished OR EXCLUDED.finished,
    last_read_at = NOW(),
    updated_at = NOW()
RETURNING id, user_id, article_path, article_title, progress_percent, scroll_y, finished, last_read_at, created_at, updated_at;

-- name: GetReadingProgress :one
SELECT reading_progress.id, reading_progress.user_id, reading_progress.article_path, reading_progress.article_title,
       reading_progress.progress_percent, reading_progress.scroll_y, reading_progress.finished,
       reading_progress.last_read_at, reading_progress.created_at, reading_progress.updated_at
FROM reading_progress
JOIN content_items ON content_items.public_path = reading_progress.article_path
WHERE reading_progress.user_id = $1
  AND reading_progress.article_path = $2
  AND content_items.is_active
  AND content_items.content_type IN ('course_article', 'article', 'geektime_article', 'love_course_article');

-- name: ListReadingProgressByArticlePaths :many
SELECT reading_progress.id, reading_progress.user_id, reading_progress.article_path, reading_progress.article_title,
       reading_progress.progress_percent, reading_progress.scroll_y, reading_progress.finished,
       reading_progress.last_read_at, reading_progress.created_at, reading_progress.updated_at
FROM reading_progress
JOIN content_items ON content_items.public_path = reading_progress.article_path
WHERE reading_progress.user_id = $1
  AND reading_progress.article_path = ANY(sqlc.arg(article_paths)::text[])
  AND content_items.is_active
  AND content_items.content_type IN ('course_article', 'article', 'geektime_article', 'love_course_article');

-- name: ListLatestReadingProgressByCoursePaths :many
SELECT courses.public_path AS course_path,
       reading_progress.id, reading_progress.user_id, reading_progress.article_path, reading_progress.article_title,
       reading_progress.progress_percent, reading_progress.scroll_y, reading_progress.finished,
       reading_progress.last_read_at, reading_progress.created_at, reading_progress.updated_at
FROM user_course_progress
JOIN courses ON courses.id = user_course_progress.course_id
JOIN content_items ON content_items.id = user_course_progress.latest_content_item_id
JOIN reading_progress
  ON reading_progress.user_id = user_course_progress.user_id
 AND reading_progress.article_path = content_items.public_path
WHERE user_course_progress.user_id = $1
  AND courses.public_path = ANY(sqlc.arg(course_paths)::text[])
  AND courses.is_active
  AND content_items.is_active
  AND content_items.content_type = 'course_article'
ORDER BY courses.public_path;

-- name: ListRecentReadingProgress :many
SELECT reading_progress.id, reading_progress.user_id, reading_progress.article_path, reading_progress.article_title,
       reading_progress.progress_percent, reading_progress.scroll_y, reading_progress.finished,
       reading_progress.last_read_at, reading_progress.created_at, reading_progress.updated_at
FROM reading_progress
JOIN content_items ON content_items.public_path = reading_progress.article_path
WHERE reading_progress.user_id = $1
  AND content_items.is_active
  AND content_items.content_type IN ('course_article', 'article', 'geektime_article', 'love_course_article')
ORDER BY reading_progress.last_read_at DESC
LIMIT $2;

-- name: RefreshUserCourseProgress :exec
INSERT INTO user_course_progress (
    user_id, course_id, learned_article_count, average_progress_percent, latest_content_item_id, latest_read_at, updated_at
)
SELECT
    $1,
    $2,
    COUNT(reading_progress.id)::int,
    COALESCE(ROUND(AVG(COALESCE(reading_progress.progress_percent, 0)))::int, 0),
    (
        SELECT course_articles.content_item_id
        FROM course_articles
        JOIN content_items ON content_items.id = course_articles.content_item_id
        JOIN reading_progress latest_progress
          ON latest_progress.article_path = content_items.public_path
         AND latest_progress.user_id = $1
        WHERE course_articles.course_id = $2
          AND course_articles.is_active
          AND content_items.is_active
        ORDER BY latest_progress.last_read_at DESC, latest_progress.id DESC
        LIMIT 1
    ),
    (
        SELECT latest_progress.last_read_at
        FROM course_articles
        JOIN content_items ON content_items.id = course_articles.content_item_id
        JOIN reading_progress latest_progress
          ON latest_progress.article_path = content_items.public_path
         AND latest_progress.user_id = $1
        WHERE course_articles.course_id = $2
          AND course_articles.is_active
          AND content_items.is_active
        ORDER BY latest_progress.last_read_at DESC, latest_progress.id DESC
        LIMIT 1
    ),
    NOW()
FROM course_articles
JOIN content_items ON content_items.id = course_articles.content_item_id
LEFT JOIN reading_progress
  ON reading_progress.article_path = content_items.public_path
 AND reading_progress.user_id = $1
WHERE course_articles.course_id = $2
  AND course_articles.is_active
  AND content_items.is_active
HAVING COUNT(reading_progress.id) > 0
ON CONFLICT (user_id, course_id)
DO UPDATE SET
    learned_article_count = EXCLUDED.learned_article_count,
    average_progress_percent = EXCLUDED.average_progress_percent,
    latest_content_item_id = EXCLUDED.latest_content_item_id,
    latest_read_at = EXCLUDED.latest_read_at,
    updated_at = NOW();

-- name: RebuildCourseProgressForCourses :exec
DELETE FROM user_course_progress
WHERE course_id = ANY(sqlc.arg(course_ids)::bigint[]);

-- name: CreateCourseProgressForCourses :exec
WITH affected_user_courses AS (
    SELECT DISTINCT reading_progress.user_id, course_articles.course_id
    FROM course_articles
    JOIN content_items ON content_items.id = course_articles.content_item_id
    JOIN reading_progress ON reading_progress.article_path = content_items.public_path
    WHERE course_articles.course_id = ANY(sqlc.arg(course_ids)::bigint[])
      AND course_articles.is_active
      AND content_items.is_active
), course_metrics AS (
    SELECT
        affected_user_courses.user_id,
        affected_user_courses.course_id,
        COUNT(reading_progress.id)::int AS learned_article_count,
        ROUND(AVG(COALESCE(reading_progress.progress_percent, 0)))::int AS average_progress_percent
    FROM affected_user_courses
    JOIN course_articles ON course_articles.course_id = affected_user_courses.course_id AND course_articles.is_active
    JOIN content_items ON content_items.id = course_articles.content_item_id AND content_items.is_active
    LEFT JOIN reading_progress
      ON reading_progress.user_id = affected_user_courses.user_id
     AND reading_progress.article_path = content_items.public_path
    GROUP BY affected_user_courses.user_id, affected_user_courses.course_id
)
INSERT INTO user_course_progress (
    user_id, course_id, learned_article_count, average_progress_percent, latest_content_item_id, latest_read_at, updated_at
)
SELECT
    course_metrics.user_id,
    course_metrics.course_id,
    course_metrics.learned_article_count,
    course_metrics.average_progress_percent,
    latest.content_item_id,
    latest.last_read_at,
    NOW()
FROM course_metrics
LEFT JOIN LATERAL (
    SELECT course_articles.content_item_id, reading_progress.last_read_at
    FROM course_articles
    JOIN content_items ON content_items.id = course_articles.content_item_id
    JOIN reading_progress
      ON reading_progress.article_path = content_items.public_path
     AND reading_progress.user_id = course_metrics.user_id
    WHERE course_articles.course_id = course_metrics.course_id
      AND course_articles.is_active
      AND content_items.is_active
    ORDER BY reading_progress.last_read_at DESC, reading_progress.id DESC
    LIMIT 1
) AS latest ON TRUE;

-- name: ListCourseSummaries :many
SELECT courses.public_path, courses.title, courses.article_count,
       COALESCE(user_course_progress.learned_article_count, 0)::int AS learned_article_count,
       COALESCE(user_course_progress.average_progress_percent, 0)::int AS average_progress_percent,
       user_course_progress.latest_read_at
FROM courses
LEFT JOIN user_course_progress
  ON user_course_progress.course_id = courses.id
 AND user_course_progress.user_id = $1
WHERE courses.is_active
ORDER BY courses.public_path;

-- name: GetCourseSummary :one
SELECT courses.id, courses.public_path, courses.title, courses.article_count,
       COALESCE(user_course_progress.learned_article_count, 0)::int AS learned_article_count,
       COALESCE(user_course_progress.average_progress_percent, 0)::int AS average_progress_percent,
       user_course_progress.latest_read_at
FROM courses
LEFT JOIN user_course_progress
  ON user_course_progress.course_id = courses.id
 AND user_course_progress.user_id = $1
WHERE courses.public_path = $2
  AND courses.is_active;

-- name: ListCourseArticlesWithProgress :many
SELECT content_items.public_path, content_items.title, course_articles.position,
       reading_progress.progress_percent, reading_progress.finished, reading_progress.last_read_at
FROM course_articles
JOIN content_items ON content_items.id = course_articles.content_item_id
LEFT JOIN reading_progress
  ON reading_progress.article_path = content_items.public_path
 AND reading_progress.user_id = $1
WHERE course_articles.course_id = $2
  AND course_articles.is_active
  AND content_items.is_active
ORDER BY course_articles.position;
