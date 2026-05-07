package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/interceptor"
	"api-gateway/internal/proxy"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pbgateway "proto/gateway"
)

const maxGRPCMessageSizeBytes = 32 * 1024 * 1024

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found")
	}

	cfg := config.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	pubKey, err := loadPublicKey(cfg.JWTPublicKey)
	if err != nil {
		logger.Error("Failed to load JWT public key", "error", err)
		os.Exit(1)
	}

	authInterceptor := interceptor.NewAuthInterceptor(pubKey, cfg.InternalAPIKey, logger)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxGRPCMessageSizeBytes),
		grpc.MaxSendMsgSize(maxGRPCMessageSizeBytes),
		grpc.ChainUnaryInterceptor(loggingInterceptor(logger), authInterceptor.Unary()),
		grpc.StreamInterceptor(authInterceptor.Stream()),
	)

	gatewayProxy := proxy.NewGatewayProxy(cfg, logger)
	pbgateway.RegisterBackendGatewayServer(s, gatewayProxy)

	reflection.Register(s)

	go func() {
		logger.Info("gRPC server starting", "port", cfg.Port)
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

func loadPublicKey(keyData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(keyData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}
