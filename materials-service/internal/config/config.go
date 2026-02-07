package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBDSN string

	// MinIO
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool

	// HTTP
	HTTPPort string

	// gRPC
	GRPCPort string

	// Logging
	LogLevel string

	// Internal API
	InternalAPIKey string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		DBDSN:          getEnv("DB_DSN", "postgres://postgres:CHANGE_ME@localhost:5432/materials_db?sslmode=disable"),
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
      MinIOSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:    getEnv("MINIO_BUCKET", "materials"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),
		HTTPPort:       getEnv("HTTP_PORT", "8081"),
		GRPCPort:       getEnv("GRPC_PORT", "50052"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
