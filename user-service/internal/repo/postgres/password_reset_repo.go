package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"user-service/internal/database"
	"user-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PasswordResetRepo struct{}

func NewPasswordResetRepo() *PasswordResetRepo {
	return &PasswordResetRepo{}
}

func (r *PasswordResetRepo) CreateResetCode(ctx context.Context, email string, userID *uint, codeHash, salt string, expiresAt time.Time, maxAttempts int) (string, error) {
	id := uuid.New().String()

	var userIDVal interface{}
	if userID != nil {
		userIDVal = *userID
	}

	_, err := database.DB.Exec(ctx,
		`INSERT INTO password_reset_codes (id, user_id, email, code_hash, salt, expires_at, max_attempts, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		id, userIDVal, email, codeHash, salt, expiresAt, maxAttempts)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *PasswordResetRepo) GetLatestActiveCode(ctx context.Context, email string) (*model.PasswordResetCode, error) {
	var code model.PasswordResetCode
	var userID sql.NullInt64

	err := database.DB.QueryRow(ctx,
		`SELECT id, user_id, email, code_hash, salt, attempts, max_attempts, expires_at, used_at, created_at
		 FROM password_reset_codes
		 WHERE email = $1 AND used_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`,
		email).Scan(
		&code.ID, &userID, &code.Email, &code.CodeHash, &code.Salt,
		&code.Attempts, &code.MaxAttempts, &code.ExpiresAt, &code.UsedAt, &code.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("code not found")
		}
		return nil, err
	}

	if userID.Valid {
		uid := uint(userID.Int64)
		code.UserID = &uid
	}

	return &code, nil
}

func (r *PasswordResetRepo) IncrementAttempts(ctx context.Context, codeID string) error {
	_, err := database.DB.Exec(ctx,
		"UPDATE password_reset_codes SET attempts = attempts + 1 WHERE id = $1",
		codeID)
	return err
}

func (r *PasswordResetRepo) MarkAsUsed(ctx context.Context, codeID string) error {
	_, err := database.DB.Exec(ctx,
		"UPDATE password_reset_codes SET used_at = NOW() WHERE id = $1",
		codeID)
	if err != nil {
		return fmt.Errorf("failed to mark as used: %w", err)
	}
	return nil
}

func (r *PasswordResetRepo) MarkAsUsedTx(ctx context.Context, tx pgx.Tx, codeID string) error {
	_, err := tx.Exec(ctx,
		"UPDATE password_reset_codes SET used_at = NOW() WHERE id = $1",
		codeID)
	if err != nil {
		return fmt.Errorf("failed to mark as used: %w", err)
	}
	return nil
}

func (r *PasswordResetRepo) InvalidateOtherCodes(ctx context.Context, email string, excludeID string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE password_reset_codes SET used_at = NOW()
		 WHERE email = $1 AND id != $2 AND used_at IS NULL`,
		email, excludeID)
	return err
}

func (r *PasswordResetRepo) InvalidateOtherCodesTx(ctx context.Context, tx pgx.Tx, email string, excludeID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE password_reset_codes SET used_at = NOW()
		 WHERE email = $1 AND id != $2 AND used_at IS NULL`,
		email, excludeID)
	return err
}

func (r *PasswordResetRepo) CheckCooldown(ctx context.Context, email string, cooldownSeconds int) (bool, error) {
	var lastCreated time.Time
	err := database.DB.QueryRow(ctx,
		`SELECT created_at FROM password_reset_codes
		 WHERE email = $1 ORDER BY created_at DESC LIMIT 1`,
		email).Scan(&lastCreated)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}

	elapsed := time.Since(lastCreated)
	return elapsed >= time.Duration(cooldownSeconds)*time.Second, nil
}

func HashCode(code, pepper, salt string) string {
	h := sha256.New()
	h.Write([]byte(pepper + code + salt))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
