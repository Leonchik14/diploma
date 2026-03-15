package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort        string
	GRPCPort        string
	DatabaseURL     string
	UserServiceURL  string
	InternalAPIKey  string
	HHAppToken      string
	HHUserAgent     string
	HHHost          string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		HTTPPort:       getEnv("HTTP_PORT", "8083"),
		GRPCPort:       getEnv("GRPC_PORT", "50054"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:CHANGE_ME@localhost:5432/diploma?sslmode=disable"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "user-service:9091"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
		HHAppToken:     getEnv("HH_APP_TOKEN", ""),
		HHUserAgent:    getEnv("HH_USER_AGENT", "InterviewPrepApp/1.0"),
		HHHost:         getEnv("HH_HOST", "api.hh.ru"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
