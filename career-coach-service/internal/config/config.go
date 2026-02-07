package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenRouterAPIKey   string
	ParseModel          string
	CoachModel          string
	HTTPPort            string
	GRPCPort            string
	RequestTimeout      time.Duration
	OpenRouterEndpoint  string
	MaxResumeChars      int
	ChatHistoryLimit    int
	DatabaseURL         string
	InternalAPIKey      string
	MaterialsServiceURL string
	UserServiceURL      string
	UserServiceGRPC     string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	cfg := &Config{
		OpenRouterAPIKey:   getEnv("OPENROUTER_API_KEY", ""),
		ParseModel:         getEnv("PARSE_MODEL", "google/gemma-3-27b-instruct"),
		CoachModel:         getEnv("COACH_MODEL", "google/gemma-3-27b-instruct"),
		HTTPPort:           getEnv("HTTP_PORT", "8082"),
		GRPCPort:           getEnv("GRPC_PORT", "9093"),
		OpenRouterEndpoint: "https://openrouter.ai/api/v1/chat/completions",
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:CHANGE_ME@localhost:5432/career_coach?sslmode=disable"),
		InternalAPIKey:     getEnv("INTERNAL_API_KEY", ""),
		MaterialsServiceURL: getEnv("MATERIALS_SERVICE_URL", "materials-service:9092"),
		UserServiceURL:      getEnv("USER_SERVICE_URL", "http://user-service:8080"),
		UserServiceGRPC:     getEnv("USER_SERVICE_GRPC", "user-service:9091"),
	}

	timeoutMs := getEnv("REQUEST_TIMEOUT_MS", "30000")
	if ms, err := strconv.Atoi(timeoutMs); err == nil {
		cfg.RequestTimeout = time.Duration(ms) * time.Millisecond
	} else {
		cfg.RequestTimeout = 30 * time.Second
	}

	maxChars := getEnv("MAX_RESUME_CHARS", "50000")
	if chars, err := strconv.Atoi(maxChars); err == nil {
		cfg.MaxResumeChars = chars
	} else {
		cfg.MaxResumeChars = 50000
	}

	historyLimit := getEnv("CHAT_HISTORY_LIMIT", "20")
	if limit, err := strconv.Atoi(historyLimit); err == nil {
		cfg.ChatHistoryLimit = limit
	} else {
		cfg.ChatHistoryLimit = 20
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
