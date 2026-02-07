package main

import (
	"log/slog"
	"os"
	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/grpc"
)

func main() {
	cfg := config.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := database.Connect(cfg.DatabaseURL); err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	grpcServer := grpc.NewServer(cfg, logger)
	if err := grpcServer.Start(); err != nil {
		logger.Error("Failed to start gRPC server", "error", err)
		os.Exit(1)
	}
}
