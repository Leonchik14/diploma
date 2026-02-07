package config

import (
	"os"
)

type Config struct {
	Port           string
	JWTPublicKey   string
	InternalAPIKey string
	UserServiceURL string
	MaterialsServiceURL string
	CoachServiceURL string
	JobsServiceURL string
	CalendarServiceURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:           getEnv("PORT", "9090"),
		JWTPublicKey:   getEnv("JWT_PUBLIC_KEY", ""),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
		UserServiceURL: getEnv("USER_SERVICE_URL", "user-service:9091"),
		MaterialsServiceURL: getEnv("MATERIALS_SERVICE_URL", "materials-service:9092"),
		CoachServiceURL: getEnv("COACH_SERVICE_URL", "career-coach-service:9093"),
		JobsServiceURL: getEnv("JOBS_SERVICE_URL", "job-service:9094"),
		CalendarServiceURL: getEnv("CALENDAR_SERVICE_URL", "calendar-service:9095"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
