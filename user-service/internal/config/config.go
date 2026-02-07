package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL              string
	JWTSecret                string
	JWTPrivateKey             string
	Port                     string
	GRPCPort                 string
	SMTPHost                 string
	SMTPPort                 string
	SMTPUser                 string
	SMTPPassword             string
	SMTPFromEmail            string
	SMTPFromName             string
	SMTPTLS                  bool
	CodeExpiration           int
	RefreshTokenExpDays      int
	InternalAPIKey           string
	PasswordResetCodeTTL        int
	PasswordResetCooldown       int
	PasswordResetMaxAttempts    int
	PasswordResetPepper         string
	MaterialsServiceAddr        string
	CoachServiceAddr            string
	JobsServiceAddr             string
	CalendarServiceAddr         string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
DatabaseURL:              getEnv("DATABASE_URL", "postgres://user:CHANGE_ME@localhost/diploma?sslmode=disable"),
      JWTSecret:                getEnv("JWT_SECRET", ""),
		JWTPrivateKey:             getEnv("JWT_PRIVATE_KEY", ""),
		Port:                     getEnv("PORT", "8080"),
		GRPCPort:                 getEnv("GRPC_PORT", "9091"),
		SMTPHost:                 getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:                 getEnv("SMTP_PORT", "587"),
		SMTPUser:                 getEnv("SMTP_USER", ""),
		SMTPPassword:             getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:            getEnv("SMTP_FROM_EMAIL", ""),
		SMTPFromName:             getEnv("SMTP_FROM_NAME", "Interview Prep App"),
		SMTPTLS:                  getEnvBool("SMTP_TLS", true),
		CodeExpiration:           getIntEnv("CODE_EXPIRATION_MINUTES", 15),
		RefreshTokenExpDays:      getIntEnv("REFRESH_TOKEN_EXPIRATION_DAYS", 30),
		InternalAPIKey:           getEnv("INTERNAL_API_KEY", ""),
		PasswordResetCodeTTL:        getIntEnv("PASSWORD_RESET_CODE_TTL_MINUTES", 10),
		PasswordResetCooldown:       getIntEnv("PASSWORD_RESET_COOLDOWN_SECONDS", 60),
		PasswordResetMaxAttempts:    getIntEnv("PASSWORD_RESET_MAX_ATTEMPTS", 5),
		PasswordResetPepper:         getEnv("PASSWORD_RESET_PEPPER", ""),
		MaterialsServiceAddr:        getEnv("MATERIALS_SERVICE_ADDR", "materials-service:50052"),
		CoachServiceAddr:            getEnv("COACH_SERVICE_ADDR", "career-coach-service:50053"),
		JobsServiceAddr:             getEnv("JOBS_SERVICE_ADDR", "job-service:50054"),
		CalendarServiceAddr:         getEnv("CALENDAR_SERVICE_ADDR", "calendar-service:50055"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
