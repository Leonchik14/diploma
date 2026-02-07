package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/email"
	"user-service/internal/repo/postgres"
	"user-service/internal/utils"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PasswordResetService struct {
	cfg    *config.Config
	repo   *postgres.PasswordResetRepo
	sender email.Sender
}

func NewPasswordResetService(cfg *config.Config, sender email.Sender) *PasswordResetService {
	return &PasswordResetService{
		cfg:    cfg,
		repo:   postgres.NewPasswordResetRepo(),
		sender: sender,
	}
}

func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, emailAddr string) error {
	if !isValidEmail(emailAddr) {
		return status.Errorf(codes.InvalidArgument, "invalid email format")
	}

	canRequest, err := s.repo.CheckCooldown(ctx, emailAddr, s.cfg.PasswordResetCooldown)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check cooldown")
	}
	if !canRequest {
		return status.Errorf(codes.ResourceExhausted, "please wait before requesting another code")
	}

	var userID *uint
	var userIDVal uint
	err = database.DB.QueryRow(ctx,
		"SELECT id FROM users WHERE email = $1", emailAddr).Scan(&userIDVal)
	if err == nil {
		userID = &userIDVal
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return status.Errorf(codes.Internal, "database error")
	}

	code := generateCode()
	salt, err := postgres.GenerateSalt()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to generate salt")
	}

	codeHash := postgres.HashCode(code, s.cfg.PasswordResetPepper, salt)
	expiresAt := time.Now().Add(time.Duration(s.cfg.PasswordResetCodeTTL) * time.Minute)

	_, err = s.repo.CreateResetCode(ctx, emailAddr, userID, codeHash, salt, expiresAt, s.cfg.PasswordResetMaxAttempts)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create reset code")
	}

	if err := s.sender.SendPasswordResetCode(emailAddr, code, s.cfg.PasswordResetCodeTTL); err != nil {
		return status.Errorf(codes.Internal, "failed to send email")
	}

	return nil
}

func (s *PasswordResetService) VerifyCode(ctx context.Context, emailAddr, code string) (bool, error) {
	if !isValidEmail(emailAddr) {
		return false, status.Errorf(codes.InvalidArgument, "invalid email format")
	}
	if !isValidCode(code) {
		return false, status.Errorf(codes.InvalidArgument, "invalid code format")
	}

	resetCode, err := s.repo.GetLatestActiveCode(ctx, emailAddr)
	if err != nil {
		return false, nil
	}

	if resetCode.Attempts >= resetCode.MaxAttempts {
		return false, status.Errorf(codes.ResourceExhausted, "too many attempts")
	}

	expectedHash := postgres.HashCode(code, s.cfg.PasswordResetPepper, resetCode.Salt)
	if expectedHash != resetCode.CodeHash {
		if err := s.repo.IncrementAttempts(ctx, resetCode.ID); err != nil {
			return false, status.Errorf(codes.Internal, "failed to increment attempts")
		}
		return false, nil
	}

	return true, nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, emailAddr, code, newPassword string) error {
	if !isValidEmail(emailAddr) {
		return status.Errorf(codes.InvalidArgument, "invalid email format")
	}
	if !isValidCode(code) {
		return status.Errorf(codes.InvalidArgument, "invalid code format")
	}
	if len(newPassword) < 8 {
		return status.Errorf(codes.InvalidArgument, "password must be at least 8 characters")
	}

	resetCode, err := s.repo.GetLatestActiveCode(ctx, emailAddr)
	if err != nil {
		return status.Errorf(codes.PermissionDenied, "invalid or expired code")
	}

	if resetCode.Attempts >= resetCode.MaxAttempts {
		return status.Errorf(codes.ResourceExhausted, "too many attempts")
	}

	expectedHash := postgres.HashCode(code, s.cfg.PasswordResetPepper, resetCode.Salt)
	if expectedHash != resetCode.CodeHash {
		if err := s.repo.IncrementAttempts(ctx, resetCode.ID); err != nil {
			return status.Errorf(codes.Internal, "failed to increment attempts")
		}
		return status.Errorf(codes.PermissionDenied, "invalid code")
	}

	if resetCode.UserID == nil {
		return status.Errorf(codes.NotFound, "user not found")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to hash password")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"UPDATE users SET password = $1 WHERE id = $2",
		hashedPassword, *resetCode.UserID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update password")
	}

	if err := s.repo.MarkAsUsedTx(ctx, tx, resetCode.ID); err != nil {
		return status.Errorf(codes.Internal, "failed to mark code as used: %v", err)
	}

	if err := s.repo.InvalidateOtherCodesTx(ctx, tx, emailAddr, resetCode.ID); err != nil {
		return status.Errorf(codes.Internal, "failed to invalidate other codes")
	}

	if err := tx.Commit(ctx); err != nil {
		return status.Errorf(codes.Internal, "failed to commit transaction")
	}

	return nil
}

func generateCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func isValidCode(code string) bool {
	matched, _ := regexp.MatchString(`^\d{6}$`, code)
	return matched
}

func isValidEmail(email string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, email)
	return matched
}
