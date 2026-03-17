package model

import "time"

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	Bio          string
	AvatarURL    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type Follow struct {
	ID          string
	FollowerID  string
	FollowingID string
	CreatedAt   time.Time
}
