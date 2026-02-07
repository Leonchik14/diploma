package model

import "time"

type PasswordResetCode struct {
	ID          string
	UserID      *uint
	Email       string
	CodeHash    string
	Salt        string
	Attempts    int
	MaxAttempts int
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}
