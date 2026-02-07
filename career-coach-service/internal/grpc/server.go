package grpc

import (
	"context"
	"log/slog"
	"strconv"

	"career-coach-service/internal/config"
	"career-coach-service/internal/grpc/handlers"
	"career-coach-service/internal/requestctx"
	"career-coach-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pbcoach "proto/coach"
)

func NewServer(cfg *config.Config, coachService *service.CoachService, resumeService *service.ResumeService, logger *slog.Logger) *grpc.Server {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(internalAuthInterceptor(cfg.InternalAPIKey, logger)),
	)

	handler := handlers.NewCoachHandler(coachService, resumeService, logger)
	pbcoach.RegisterCoachServiceServer(s, handler)

	reflection.Register(s)
	return s
}

func internalAuthInterceptor(internalAPIKey string, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("missing metadata in internal call")
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		apiKey := md.Get("x-internal-api-key")
		if len(apiKey) == 0 || apiKey[0] != internalAPIKey {
			logger.Warn("invalid internal API key", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "invalid internal API key")
		}

		userIDStr := md.Get("x-user-id")
		if len(userIDStr) == 0 || userIDStr[0] == "" {
			logger.Warn("missing x-user-id in internal call", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "missing user ID")
		}

		userID, err := strconv.ParseUint(userIDStr[0], 10, 32)
		if err != nil {
			logger.Warn("invalid x-user-id format", "method", info.FullMethod, "user_id_str", userIDStr[0], "error", err)
			return nil, status.Errorf(codes.InvalidArgument, "invalid user ID format")
		}

		ctx = requestctx.WithUserID(ctx, uint(userID))
		return handler(ctx, req)
	}
}
