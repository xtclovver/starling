package model

import "time"

type Media struct {
	ID          string
	UserID      string
	PostID      *string
	Bucket      string
	ObjectKey   string
	ContentType string
	CreatedAt   time.Time
}
