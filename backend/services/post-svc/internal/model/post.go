package model

import "time"

type Post struct {
	ID            string
	UserID        string
	Content       string
	ViewsCount    int64
	LikesCount    int64
	CommentsCount int64
	RepostsCount  int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	EditedAt      *time.Time
	DeletedAt     *time.Time
}

type Like struct {
	ID        string
	UserID    string
	PostID    string
	CreatedAt time.Time
}

type Bookmark struct {
	ID        string
	UserID    string
	PostID    string
	CreatedAt time.Time
}

type Repost struct {
	ID           string
	UserID       string
	PostID       string
	QuoteContent string
	Type         string
	CreatedAt    time.Time
}

type Hashtag struct {
	ID        string
	Tag       string
	CreatedAt time.Time
}

type TrendingHashtag struct {
	Tag       string
	PostCount int32
}
