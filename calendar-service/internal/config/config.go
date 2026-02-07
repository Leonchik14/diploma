package config

import (
	"os"
)

type Config struct {
	DatabaseURL    string
	GRPCPort       string
	InternalAPIKey string
	LogLevel       string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:CHANGE_ME@localhost:5432/diploma?sslmode=disable"),
		GRPCPort:       getEnv("GRPC_PORT", "9095"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
