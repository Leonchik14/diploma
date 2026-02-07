package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"user-service/internal/clients"
	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/utils"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountDeletionService struct {
	cfg            *config.Config
	logger         *slog.Logger
	materialsClient *clients.MaterialsClient
	coachClient    *clients.CoachClient
	jobsClient     *clients.JobsClient
	calendarClient *clients.CalendarClient
}

func NewAccountDeletionService(cfg *config.Config, logger *slog.Logger) *AccountDeletionService {
	return &AccountDeletionService{
		cfg:            cfg,
		logger:         logger,
		materialsClient: clients.NewMaterialsClient(cfg.MaterialsServiceAddr, cfg.InternalAPIKey, logger),
		coachClient:    clients.NewCoachClient(cfg.CoachServiceAddr, cfg.InternalAPIKey, logger),
		jobsClient:     clients.NewJobsClient(cfg.JobsServiceAddr, cfg.InternalAPIKey, logger),
		calendarClient: clients.NewCalendarClient(cfg.CalendarServiceAddr, cfg.InternalAPIKey, logger),
	}
}

func (s *AccountDeletionService) DeleteAccount(ctx context.Context, userID uint, password string) error {
	var hashedPassword string
	var deletedAt sql.NullTime

	err := database.DB.QueryRow(ctx,
		"SELECT password, deleted_at FROM users WHERE id = $1",
		userID).Scan(&hashedPassword, &deletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return status.Errorf(codes.NotFound, "user not found")
		}
		return status.Errorf(codes.Internal, "database error")
	}

	if deletedAt.Valid {
		return nil
	}

	if !utils.CheckPasswordHash(password, hashedPassword) {
		return status.Errorf(codes.PermissionDenied, "invalid password")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	_, err = tx.Exec(ctx,
		`UPDATE users SET deleted_at = $1, email = '<deleted>' || id::text || '@deleted.local', username = 'deleted_' || id::text WHERE id = $2`,
		now, userID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to mark user as deleted")
	}

	_, err = tx.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE user_id = $1",
		userID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to revoke refresh tokens")
	}

	if err := tx.Commit(ctx); err != nil {
		return status.Errorf(codes.Internal, "failed to commit transaction")
	}

	go s.deleteUserDataFromServices(context.Background(), userID)

	return nil
}

func (s *AccountDeletionService) deleteUserDataFromServices(ctx context.Context, userID uint) {
	if s.materialsClient != nil {
		if err := s.materialsClient.DeleteUserData(ctx, userID); err != nil {
			s.logger.Error("Failed to delete user data from materials-service", "user_id", userID, "error", err)
		}
	}

	if s.coachClient != nil {
		if err := s.coachClient.DeleteUserData(ctx, userID); err != nil {
			s.logger.Error("Failed to delete user data from coach-service", "user_id", userID, "error", err)
		}
	}

	if s.jobsClient != nil {
		if err := s.jobsClient.DeleteUserData(ctx, userID); err != nil {
			s.logger.Error("Failed to delete user data from jobs-service", "user_id", userID, "error", err)
		}
	}

	if s.calendarClient != nil {
		if err := s.calendarClient.DeleteUserData(ctx, userID); err != nil {
			s.logger.Error("Failed to delete user data from calendar-service", "user_id", userID, "error", err)
		}
	}
}
