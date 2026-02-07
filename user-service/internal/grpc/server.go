package grpc

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"user-service/internal/config"
	"user-service/internal/grpc/handlers"
	"user-service/internal/requestctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pbauth "proto/auth"
	pbuser "proto/user"
)

type Server struct {
	cfg       *config.Config
	logger    *slog.Logger
	authHandler *handlers.AuthHandler
	userHandler *handlers.UserHandler
}

func NewServer(cfg *config.Config, logger *slog.Logger) *Server {
	authHandler := handlers.NewAuthHandler(cfg, logger)
	userHandler := handlers.NewUserHandler(cfg, logger)
	return &Server{
		cfg:        cfg,
		logger:     logger,
		authHandler: authHandler,
		userHandler: userHandler,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.cfg.GRPCPort)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(s.internalAuthInterceptor),
	)

	pbauth.RegisterAuthServiceServer(grpcServer, s.authHandler)
	pbuser.RegisterUserServiceServer(grpcServer, s.userHandler)

	reflection.Register(grpcServer)

	go func() {
		s.logger.Info("gRPC server starting", "port", s.cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			s.logger.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	return nil
}

var internalOnlyResumeMethods = map[string]bool{
	"/user.UserService/GetResumeProfileInternal":   true,
	"/user.UserService/UpsertResumeProfileInternal": true,
	"/user.UserService/PatchResumeProfileInternal": true,
}

func (s *Server) internalAuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if info.FullMethod == "/auth.AuthService/Login" ||
		info.FullMethod == "/auth.AuthService/Refresh" ||
		info.FullMethod == "/auth.AuthService/Register" ||
		info.FullMethod == "/user.UserService/RequestPasswordReset" ||
		info.FullMethod == "/user.UserService/VerifyPasswordResetCode" ||
		info.FullMethod == "/user.UserService/ResetPassword" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	internalKey := md.Get("x-internal-api-key")
	if len(internalKey) == 0 || internalKey[0] != s.cfg.InternalAPIKey {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
	}

	if internalOnlyResumeMethods[info.FullMethod] {
		return handler(ctx, req)
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
