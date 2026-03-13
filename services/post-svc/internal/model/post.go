package model

import "time"

type Post struct {
	ID            string
	UserID        string
	Content       string
	MediaURL      string
	LikesCount    int64
	CommentsCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type Like struct {
	ID        string
	UserID    string
	PostID    string
	CreatedAt time.Time
}
