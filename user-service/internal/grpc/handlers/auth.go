package handlers

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/utils"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbauth "proto/auth"
)

type AuthHandler struct {
	pbauth.UnimplementedAuthServiceServer
	cfg    *config.Config
	logger *slog.Logger
	privKey *rsa.PrivateKey
}

var emailFormatRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", status.Errorf(codes.InvalidArgument, "email is required")
	}
	if !emailFormatRegex.MatchString(email) {
		return "", status.Errorf(codes.InvalidArgument, "invalid email format")
	}
	return email, nil
}

func NewAuthHandler(cfg *config.Config, logger *slog.Logger) *AuthHandler {
	privKey, err := utils.LoadPrivateKey(cfg.JWTPrivateKey)
	if err != nil {
		logger.Error("Failed to load private key", "error", err)
		panic(err)
	}
	return &AuthHandler{
		cfg:    cfg,
		logger: logger,
		privKey: privKey,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *pbauth.RegisterRequest) (*pbauth.RegisterResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Password == "" || len(req.Password) < 8 {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 8 characters")
	}

	// Проверяем, существует ли пользователь с таким email
	var existingUserID uint
	err = database.DB.QueryRow(ctx,
		"SELECT id FROM users WHERE email = $1 LIMIT 1",
		email).Scan(&existingUserID)

	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "user with this email already exists")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "database error")
	}

	// Хешируем пароль
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password")
	}

	// username для логина = email
	username := email
	now := time.Now()
	var userID uint
	err = database.DB.QueryRow(ctx,
		`INSERT INTO users (username, email, password, first_name, last_name, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		username, email, hashedPassword, req.FirstName, req.LastName, now, now).Scan(&userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return nil, status.Errorf(codes.AlreadyExists, "user with this email or username already exists")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user")
	}

	// Генерируем access токен (username = email для логина)
	accessToken, err := utils.GenerateAccessTokenRSA(userID, email, username, h.privKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token")
	}

	// Создаем refresh token
	refreshToken, err := h.createRefreshToken(ctx, userID, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token")
	}

	return &pbauth.RegisterResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &pbauth.UserProfile{
			Id:         uint32(userID),
			FirstName:  req.FirstName,
			LastName:   req.LastName,
			Email:      email,
			Username:   username,
		},
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pbauth.LoginRequest) (*pbauth.LoginResponse, error) {
	emailInput, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}

	var userID uint
	var email, username, password string
	var deletedAt sql.NullTime

	err = database.DB.QueryRow(ctx,
		"SELECT id, email, username, password, deleted_at FROM users WHERE email = $1",
		emailInput).Scan(&userID, &email, &username, &password, &deletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if deletedAt.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	if !utils.CheckPasswordHash(req.Password, password) {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	accessToken, err := utils.GenerateAccessTokenRSA(userID, email, username, h.privKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token")
	}

	deviceID := req.DeviceId
	if deviceID == "" {
		deviceID = "unknown"
	}

	refreshToken, err := h.createRefreshToken(ctx, userID, deviceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create refresh token")
	}

	var firstName, lastName sql.NullString
	_ = database.DB.QueryRow(ctx, "SELECT first_name, last_name FROM users WHERE id = $1", userID).Scan(&firstName, &lastName)
	fn, ln := "", ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	return &pbauth.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &pbauth.UserProfile{
			Id:         uint32(userID),
			FirstName:  fn,
			LastName:   ln,
			Email:      email,
			Username:   username,
		},
	}, nil
}

func (h *AuthHandler) Refresh(ctx context.Context, req *pbauth.RefreshRequest) (*pbauth.RefreshResponse, error) {
	var userID uint
	var expiresAt time.Time

	err := database.DB.QueryRow(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens 
		 WHERE token = $1 AND used = FALSE AND expires_at > NOW()`,
		req.RefreshToken).Scan(&userID, &expiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}

	var email, username string
	var deletedAt sql.NullTime
	var firstName, lastName sql.NullString
	err = database.DB.QueryRow(ctx,
		"SELECT email, username, deleted_at, first_name, last_name FROM users WHERE id = $1", userID).Scan(&email, &username, &deletedAt, &firstName, &lastName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	if deletedAt.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token")
	}

	accessToken, err := utils.GenerateAccessTokenRSA(userID, email, username, h.privKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token")
	}

	fn, ln := "", ""
	if firstName.Valid {
		fn = firstName.String
	}
	if lastName.Valid {
		ln = lastName.String
	}

	return &pbauth.RefreshResponse{
		AccessToken: accessToken,
		User: &pbauth.UserProfile{
			Id:         uint32(userID),
			FirstName:  fn,
			LastName:   ln,
			Email:      email,
			Username:   username,
		},
	}, nil
}

func (h *AuthHandler) CheckPasswordResetEmail(ctx context.Context, req *pbauth.PasswordResetCheckEmailRequest) (*pbauth.PasswordResetCheckEmailResponse, error) {
	var exists bool
	err := database.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return &pbauth.PasswordResetCheckEmailResponse{Exists: exists}, nil
}

func (h *AuthHandler) SendPasswordResetCode(ctx context.Context, req *pbauth.PasswordResetSendCodeRequest) (*pbauth.PasswordResetSendCodeResponse, error) {
	var exists bool
	err := database.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if !exists {
		return &pbauth.PasswordResetSendCodeResponse{Sent: false}, nil
	}

	code := utils.GenerateResetCode()
	expiresAt := time.Now().Add(time.Duration(h.cfg.CodeExpiration) * time.Minute)

	_, err = database.DB.Exec(ctx,
		`INSERT INTO password_resets (email, code, expires_at) VALUES ($1, $2, $3)`,
		req.Email, code, expiresAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save code")
	}

	if err := utils.SendPasswordResetCode(req.Email, code, h.cfg.SMTPHost, h.cfg.SMTPPort, h.cfg.SMTPUser, h.cfg.SMTPPassword, h.cfg.SMTPFromEmail, h.cfg.SMTPFromName); err != nil {
		h.logger.Warn("Failed to send email", "error", err)
	}

	return &pbauth.PasswordResetSendCodeResponse{Sent: true}, nil
}

func (h *AuthHandler) VerifyPasswordReset(ctx context.Context, req *pbauth.PasswordResetVerifyRequest) (*pbauth.PasswordResetVerifyResponse, error) {
	if len(req.Password) < 8 {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 8 characters")
	}

	var expiresAt time.Time
	var used bool

	err := database.DB.QueryRow(ctx,
		`SELECT expires_at, used FROM password_resets 
		 WHERE email = $1 AND code = $2 ORDER BY created_at DESC LIMIT 1`,
		req.Email, req.Code).Scan(&expiresAt, &used)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid code")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}

	if used || time.Now().After(expiresAt) {
		return nil, status.Errorf(codes.InvalidArgument, "code expired or used")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"UPDATE users SET password = $1 WHERE email = $2", hashedPassword, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update password")
	}

	_, err = tx.Exec(ctx,
		"UPDATE password_resets SET used = TRUE WHERE email = $1 AND code = $2",
		req.Email, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark code as used")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction")
	}

	return &pbauth.PasswordResetVerifyResponse{Success: true}, nil
}

func (h *AuthHandler) createRefreshToken(ctx context.Context, userID uint, deviceID string) (string, error) {
	refreshToken := utils.GenerateRefreshToken()
	expiresAt := time.Now().Add(time.Duration(h.cfg.RefreshTokenExpDays) * 24 * time.Hour)

	_, err := database.DB.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token, device_id, expires_at, created_at) 
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, refreshToken, deviceID, expiresAt, time.Now())
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}
