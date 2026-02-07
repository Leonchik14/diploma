package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"career-coach-service/internal/client"
	"career-coach-service/internal/config"
	"career-coach-service/internal/database"
	"career-coach-service/internal/extractor"
	grpcserver "career-coach-service/internal/grpc"
	"career-coach-service/internal/llm"
	"career-coach-service/internal/parser"
	"career-coach-service/internal/repository"
	"career-coach-service/internal/service"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.OpenRouterAPIKey == "" {
		slog.Error("OPENROUTER_API_KEY is required")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting career-coach-service",
		"grpc_port", cfg.GRPCPort,
		"parse_model", cfg.ParseModel,
		"coach_model", cfg.CoachModel,
		"timeout", cfg.RequestTimeout,
	)

	if err := database.Connect(cfg.DatabaseURL); err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	llmClient := llm.NewClient(cfg.OpenRouterAPIKey, cfg.OpenRouterEndpoint, cfg.RequestTimeout)

	repo := repository.NewRepository()

	materialsClient := client.NewMaterialsClient(cfg.MaterialsServiceURL, cfg.InternalAPIKey, cfg.RequestTimeout)
	userClient := client.NewUserClient(cfg.UserServiceGRPC, cfg.InternalAPIKey, cfg.RequestTimeout)

	parser := parser.NewParser(llmClient, cfg.ParseModel, cfg.MaxResumeChars)
	extractor := extractor.NewExtractor(cfg.MaxResumeChars)
	resumeService := service.NewResumeService(parser, repo, extractor, materialsClient, userClient)

	coachService := service.NewCoachService(llmClient, repo, cfg.CoachModel, cfg.ChatHistoryLimit)

	grpcServer := grpcserver.NewServer(cfg, coachService, resumeService, logger)

	lis, err := net.Listen("tcp", "0.0.0.0:"+cfg.GRPCPort)
	if err != nil {
		logger.Error("Failed to listen gRPC", "error", err)
		os.Exit(1)
	}
	logger.Info("gRPC server starting", "addr", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	logger.Info("server exited")
}
