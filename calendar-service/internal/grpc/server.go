package grpc

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"calendar-service/internal/config"
	"calendar-service/internal/requestctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewServer(cfg *config.Config, logger *slog.Logger) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor(logger), internalAuthInterceptor(cfg, logger)),
	)
}

func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start).Milliseconds()
		code := codes.OK
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		}
		if err != nil {
			logger.Warn("gRPC", "method", info.FullMethod, "code", code.String(), "ms", dur, "error", err.Error())
		} else {
			logger.Info("gRPC", "method", info.FullMethod, "code", code.String(), "ms", dur)
		}
		return resp, err
	}
}

func internalAuthInterceptor(cfg *config.Config, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		internalKey := md.Get("x-internal-api-key")
		if len(internalKey) == 0 || internalKey[0] != cfg.InternalAPIKey {
			return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
		}

		userIDStr := md.Get("x-user-id")
		if len(userIDStr) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing user id")
		}

		userID, err := strconv.ParseUint(userIDStr[0], 10, 32)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user id")
		}

		ctx = requestctx.WithUserID(ctx, uint(userID))
		return handler(ctx, req)
	}
}
