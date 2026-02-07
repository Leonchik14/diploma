package grpc

import (
	"context"
	"log/slog"

	"job-service/internal/auth"
	"job-service/internal/config"
	"job-service/internal/grpc/handlers"
	"job-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pbjobs "proto/jobs"
)

func NewServer(cfg *config.Config, svc *service.Service, logger *slog.Logger) *grpc.Server {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(internalAuthInterceptor(cfg.InternalAPIKey, logger)),
	)

	handler := handlers.NewJobsHandler(svc, logger)
	pbjobs.RegisterJobsServiceServer(s, handler)

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

		apiKey := md.Get(auth.MetadataInternalAPIKey)
		if len(apiKey) == 0 || apiKey[0] != internalAPIKey {
			logger.Warn("invalid internal API key", "method", info.FullMethod)
			return nil, status.Errorf(codes.Unauthenticated, "invalid internal API key")
		}

		return handler(ctx, req)
	}
}
