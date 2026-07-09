package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"learn-liangliang/backend/internal/db/sqlc"
)

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.Queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	return err
}

func (s *Store) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (SessionUser, error) {
	session, err := s.Queries.GetActiveSessionByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionUser{}, ErrNotFound
	}
	if err != nil {
		return SessionUser{}, err
	}
	return SessionUser{
		SessionID:   session.ID,
		UserID:      session.UserID,
		Username:    session.Username,
		DisplayName: session.DisplayName,
		IsActive:    session.IsActive,
		ExpiresAt:   session.ExpiresAt.Time,
	}, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	return s.Queries.RevokeSession(ctx, tokenHash)
}
