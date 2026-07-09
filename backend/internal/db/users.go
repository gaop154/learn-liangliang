package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"learn-liangliang/backend/internal/db/sqlc"
)

var ErrNotFound = errors.New("记录不存在")

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	user, err := s.Queries.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}, nil
}

func (s *Store) CreateUser(ctx context.Context, username string, passwordHash string, displayName string) (User, error) {
	user, err := s.Queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
	})
	if err != nil {
		return User{}, err
	}
	return User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}, nil
}
