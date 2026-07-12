package db

import (
	"os"
	"strings"
	"testing"
)

func TestReadingProgressQueryContracts(t *testing.T) {
	paths := []string{
		"queries/reading_progress.sql",
		"sqlc/reading_progress.sql.go",
	}

	for _, filePath := range paths {
		t.Run(filePath, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("读取阅读进度 SQL 失败: %v", err)
			}
			query := string(content)
			for _, expected := range []string{
				"GetActiveContentItem",
				"ListReadingProgressByArticlePaths",
				"ListLatestReadingProgressByCoursePaths",
				"ListRecentReadingProgress",
				"content_items.is_active",
				"courses.public_path = ANY",
				"user_course_progress",
				"RefreshUserCourseProgress",
				"CreateCourseProgressForCourses",
			} {
				assertContains(t, query, expected)
			}
			if strings.Contains(query, " LIKE ") {
				t.Fatal("课程和内容查询不得使用 LIKE")
			}
		})
	}
}

func TestReadingProgressUpsertSQLKeepsHighestProgress(t *testing.T) {
	paths := []string{
		"queries/reading_progress.sql",
		"sqlc/reading_progress.sql.go",
	}
	for _, filePath := range paths {
		t.Run(filePath, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("读取阅读进度 SQL 失败: %v", err)
			}
			query := string(content)
			assertContains(t, query, "progress_percent = GREATEST(reading_progress.progress_percent, EXCLUDED.progress_percent)")
			assertContains(t, query, "scroll_y = GREATEST(reading_progress.scroll_y, EXCLUDED.scroll_y)")
			assertContains(t, query, "finished = reading_progress.finished OR EXCLUDED.finished")
		})
	}
}

func assertContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("SQL 缺少预期片段 %q", want)
	}
}
