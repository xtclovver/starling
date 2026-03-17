package model

import "time"

type Comment struct {
	ID         string
	PostID     string
	UserID     string
	ParentID   *string
	Content    string
	LikesCount int32
	Depth      int32
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	Children   []*Comment
}
