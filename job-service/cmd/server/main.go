package main

import (
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"job-service/internal/client"
	"job-service/internal/config"
	"job-service/internal/database"
	grpcserver "job-service/internal/grpc"
	"job-service/internal/repository"
	"job-service/internal/service"
)

func main() {
	cfg := config.LoadConfig()

	logger := log.New(os.Stdout, "[job-service] ", log.LstdFlags)

	logger.Printf("Starting job-service on gRPC port %s", cfg.GRPCPort)

	if err := database.Connect(cfg.DatabaseURL); err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	userClient := client.NewUserClient(cfg.UserServiceURL, cfg.InternalAPIKey, 10*time.Second)
	hhClient := client.NewHHClient(cfg.HHHost, cfg.HHAppToken, cfg.HHUserAgent, 10*time.Second, slogLogger)
	favoritesRepo := repository.NewFavoritesRepo()
	grpcSvc := service.NewService(userClient, hhClient, favoritesRepo)
	grpcServer := grpcserver.NewServer(cfg, grpcSvc, slogLogger)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Fatalf("Failed to listen gRPC: %v", err)
	}
	logger.Printf("gRPC server starting on %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")

	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	logger.Println("Server exited")
}
