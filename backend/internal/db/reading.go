package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"learn-liangliang/backend/internal/db/sqlc"
)

func (s *Store) UpsertReadingProgress(ctx context.Context, userID int64, articlePath string, articleTitle string, progressPercent int, scrollY int, finished bool) (ReadingProgress, error) {
	return s.UpsertActiveReadingProgress(ctx, userID, articlePath, articleTitle, progressPercent, scrollY, finished)
}

func (s *Store) GetReadingProgress(ctx context.Context, userID int64, articlePath string) (ReadingProgress, error) {
	progress, err := s.Queries.GetReadingProgress(ctx, sqlc.GetReadingProgressParams{UserID: userID, ArticlePath: articlePath})
	if errors.Is(err, pgx.ErrNoRows) {
		return ReadingProgress{}, ErrNotFound
	}
	if err != nil {
		return ReadingProgress{}, err
	}
	return toReadingProgress(progress), nil
}

func (s *Store) ListReadingProgressByArticlePaths(ctx context.Context, userID int64, articlePaths []string) ([]ReadingProgress, error) {
	items, err := s.Queries.ListReadingProgressByArticlePaths(ctx, sqlc.ListReadingProgressByArticlePathsParams{
		UserID:       userID,
		ArticlePaths: articlePaths,
	})
	if err != nil {
		return nil, err
	}
	progresses := make([]ReadingProgress, 0, len(items))
	for _, item := range items {
		progresses = append(progresses, toReadingProgress(item))
	}
	return progresses, nil
}

type CourseResume struct {
	CoursePath string `json:"coursePath"`
	ReadingProgress
}

func (s *Store) ListLatestReadingProgressByCoursePaths(ctx context.Context, userID int64, coursePaths []string) ([]CourseResume, error) {
	items, err := s.Queries.ListLatestReadingProgressByCoursePaths(ctx, sqlc.ListLatestReadingProgressByCoursePathsParams{
		UserID:      userID,
		CoursePaths: coursePaths,
	})
	if err != nil {
		return nil, err
	}
	resumes := make([]CourseResume, 0, len(items))
	for _, item := range items {
		resumes = append(resumes, CourseResume{
			CoursePath: item.CoursePath,
			ReadingProgress: ReadingProgress{
				ID:              item.ID,
				UserID:          item.UserID,
				ArticlePath:     item.ArticlePath,
				ArticleTitle:    item.ArticleTitle,
				ProgressPercent: int(item.ProgressPercent),
				ScrollY:         int(item.ScrollY),
				Finished:        item.Finished,
				LastReadAt:      item.LastReadAt.Time,
				CreatedAt:       item.CreatedAt.Time,
				UpdatedAt:       item.UpdatedAt.Time,
			},
		})
	}
	return resumes, nil
}

func (s *Store) ListRecentReadingProgress(ctx context.Context, userID int64, limit int32) ([]ReadingProgress, error) {
	items, err := s.Queries.ListRecentReadingProgress(ctx, sqlc.ListRecentReadingProgressParams{UserID: userID, Limit: limit})
	if err != nil {
		return nil, err
	}
	progresses := make([]ReadingProgress, 0, len(items))
	for _, item := range items {
		progresses = append(progresses, toReadingProgress(item))
	}
	return progresses, nil
}

func toReadingProgress(progress sqlc.ReadingProgress) ReadingProgress {
	return ReadingProgress{
		ID:              progress.ID,
		UserID:          progress.UserID,
		ArticlePath:     progress.ArticlePath,
		ArticleTitle:    progress.ArticleTitle,
		ProgressPercent: int(progress.ProgressPercent),
		ScrollY:         int(progress.ScrollY),
		Finished:        progress.Finished,
		LastReadAt:      progress.LastReadAt.Time,
		CreatedAt:       progress.CreatedAt.Time,
		UpdatedAt:       progress.UpdatedAt.Time,
	}
}
