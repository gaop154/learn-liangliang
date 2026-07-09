package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"learn-liangliang/backend/internal/db/sqlc"
)

func (s *Store) UpsertReadingProgress(ctx context.Context, userID int64, articlePath string, articleTitle string, progressPercent int, scrollY int, finished bool) (ReadingProgress, error) {
	progress, err := s.Queries.UpsertReadingProgress(ctx, sqlc.UpsertReadingProgressParams{
		UserID:          userID,
		ArticlePath:     articlePath,
		ArticleTitle:    articleTitle,
		ProgressPercent: int32(progressPercent),
		ScrollY:         int32(scrollY),
		Finished:        finished,
	})
	if err != nil {
		return ReadingProgress{}, err
	}
	return toReadingProgress(progress), nil
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
