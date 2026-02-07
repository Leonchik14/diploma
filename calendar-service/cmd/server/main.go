package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"calendar-service/internal/config"
	"calendar-service/internal/database"
	"calendar-service/internal/grpc"
	"calendar-service/internal/grpc/handlers"
	"calendar-service/internal/repo/postgres"
	"calendar-service/internal/service"

	"github.com/joho/godotenv"
	"google.golang.org/grpc/reflection"

	pbcalendar "proto/calendar"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found")
	}

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

	repo := postgres.NewEventsRepo(database.DB)
	svc := service.NewCalendarService(repo)
	handler := handlers.NewCalendarHandler(svc, logger)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer(cfg, logger)
	pbcalendar.RegisterCalendarServiceServer(s, handler)
	reflection.Register(s)

	go func() {
		logger.Info("gRPC server starting", "port", cfg.GRPCPort)
		if err := s.Serve(lis); err != nil {
			logger.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	s.GracefulStop()
	logger.Info("Server exited")
}
