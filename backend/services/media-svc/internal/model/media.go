package model

import "time"

type Media struct {
	ID          string
	UserID      string
	PostID      *string
	Bucket      string
	ObjectKey   string
	ContentType string
	Position    int
	CreatedAt   time.Time
}
