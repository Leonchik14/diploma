package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"materials-service/internal/config"
	"materials-service/internal/database"
	grpcserver "materials-service/internal/grpc"
	"materials-service/internal/repository"
	"materials-service/internal/service"
	"materials-service/internal/storage"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	var logger *zap.Logger
	var err error
	if cfg.LogLevel == "debug" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("Starting materials-service",
		zap.String("grpc_port", cfg.GRPCPort),
		zap.String("log_level", cfg.LogLevel),
	)

	// Connect to database
	if err := database.Connect(cfg.DBDSN); err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	// Setup gRPC server
	repo := repository.NewRepository()
	stor, err := storage.NewStorage(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
	if err != nil {
		logger.Fatal("Failed to create storage", zap.Error(err))
	}
	svc := service.NewService(repo, stor)
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	grpcSrv := grpcserver.NewServer(cfg, svc, slogLogger)

	// Start gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Fatal("Failed to listen gRPC", zap.Error(err))
	}
	logger.Info("gRPC server starting", zap.String("addr", lis.Addr().String()))

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	if grpcSrv != nil {
		grpcSrv.GracefulStop()
	}

	logger.Info("Server exited")
}
