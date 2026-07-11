package db

import (
	"os"
	"strings"
	"testing"
)

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
		t.Fatalf("SQL 缺少预期片段 %q\n实际 SQL:\n%s", want, content)
	}
}
