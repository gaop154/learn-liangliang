package db

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SessionUser struct {
	SessionID   int64
	UserID      int64
	Username    string
	DisplayName string
	IsActive    bool
	ExpiresAt   time.Time
}

type ReadingProgress struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"-"`
	ArticlePath     string    `json:"articlePath"`
	ArticleTitle    string    `json:"articleTitle"`
	ProgressPercent int       `json:"progressPercent"`
	ScrollY         int       `json:"scrollY"`
	Finished        bool      `json:"finished"`
	LastReadAt      time.Time `json:"lastReadAt"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
