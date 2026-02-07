package models

import (
	"time"
)

type User struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PasswordReset struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Code      string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

type RefreshToken struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Token     string    `json:"-"`
	DeviceID  string    `json:"device_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type PasswordResetCheckEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetSendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetVerifyRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
	Password string `json:"password" binding:"required,min=8"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	DeviceID     string `json:"device_id"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         UserProfile `json:"user"`
}

type RefreshTokenResponse struct {
	AccessToken string      `json:"access_token"`
	User        UserProfile `json:"user"`
}

type UserProfile struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type ResumeProfile struct {
	TargetRoles     []string    `json:"target_roles"`
	ExperienceLevel *string     `json:"experience_level"`
	Areas           []Area      `json:"areas"`
	SalaryMin       *float64    `json:"salary_min"`
	Currency        *string     `json:"currency"`
	WorkFormat      []string    `json:"work_format"`
	SkillsTop       []string    `json:"skills_top"`
}

type Area struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
