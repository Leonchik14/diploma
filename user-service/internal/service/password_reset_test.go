package service

import (
	"context"
	"testing"
	"time"

	"user-service/internal/config"
	"user-service/internal/email"
)

type mockSender struct {
	sentCodes []string
}

func (m *mockSender) SendPasswordResetCode(email, code string, ttlMinutes int) error {
	m.sentCodes = append(m.sentCodes, code)
	return nil
}

func TestGenerateCode(t *testing.T) {
	code := generateCode()
	if len(code) != 6 {
		t.Errorf("Expected code length 6, got %d", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("Expected numeric code, got %c", c)
		}
	}
}

func TestIsValidCode(t *testing.T) {
	tests := []struct {
		code   string
		valid  bool
	}{
		{"123456", true},
		{"000000", true},
		{"999999", true},
		{"12345", false},
		{"1234567", false},
		{"abc123", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isValidCode(tt.code); got != tt.valid {
			t.Errorf("isValidCode(%q) = %v, want %v", tt.code, got, tt.valid)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email  string
		valid  bool
	}{
		{"test@example.com", true},
		{"user.name@example.co.uk", true},
		{"invalid", false},
		{"@example.com", false},
		{"test@", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isValidEmail(tt.email); got != tt.valid {
			t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.valid)
		}
	}
}

func TestPasswordResetService_RequestPasswordReset(t *testing.T) {
	cfg := &config.Config{
		PasswordResetCodeTTL:     10,
		PasswordResetCooldown:    60,
		PasswordResetMaxAttempts: 5,
		PasswordResetPepper:       "test-pepper",
	}
	mockSender := &mockSender{}
	svc := &PasswordResetService{
		cfg:    cfg,
		sender: mockSender,
	}

	ctx := context.Background()

	err := svc.RequestPasswordReset(ctx, "invalid-email")
	if err == nil {
		t.Error("Expected error for invalid email")
	}

	err = svc.RequestPasswordReset(ctx, "")
	if err == nil {
		t.Error("Expected error for empty email")
	}
}

func TestPasswordResetService_VerifyCode(t *testing.T) {
	cfg := &config.Config{
		PasswordResetCodeTTL:     10,
		PasswordResetCooldown:    60,
		PasswordResetMaxAttempts: 5,
		PasswordResetPepper:       "test-pepper",
	}
	mockSender := &mockSender{}
	svc := &PasswordResetService{
		cfg:    cfg,
		sender: mockSender,
	}

	ctx := context.Background()

	_, err := svc.VerifyCode(ctx, "invalid-email", "123456")
	if err == nil {
		t.Error("Expected error for invalid email")
	}

	_, err = svc.VerifyCode(ctx, "test@example.com", "invalid")
	if err == nil {
		t.Error("Expected error for invalid code format")
	}
}

func TestPasswordResetService_ResetPassword(t *testing.T) {
	cfg := &config.Config{
		PasswordResetCodeTTL:     10,
		PasswordResetCooldown:    60,
		PasswordResetMaxAttempts: 5,
		PasswordResetPepper:       "test-pepper",
	}
	mockSender := &mockSender{}
	svc := &PasswordResetService{
		cfg:    cfg,
		sender: mockSender,
	}

	ctx := context.Background()

	err := svc.ResetPassword(ctx, "invalid-email", "123456", "newpass123")
	if err == nil {
		t.Error("Expected error for invalid email")
	}

	err = svc.ResetPassword(ctx, "test@example.com", "invalid", "newpass123")
	if err == nil {
		t.Error("Expected error for invalid code format")
	}

	err = svc.ResetPassword(ctx, "test@example.com", "123456", "short")
	if err == nil {
		t.Error("Expected error for short password")
	}
}
