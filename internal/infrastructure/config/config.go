package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var Version = "1.0.0"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppName      string
	AppVersion   string
	AppEnv       string
	AppPort      string
	MCPTransport string
	LogLevel     string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Security & JWT
	JWTSecret      string
	JWTExpiryHours time.Duration

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Google GenAI / Gemini
	GeminiAPIKey       string
	GeminiDefaultModel string

	// CORS
	AllowedOrigins string
}

// Load reads configuration from environment variables and an optional .env file.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:             getEnv("APP_NAME", "gosling-mcp-server"),
		AppVersion:          getDockerVersion(),
		AppEnv:              getEnv("APP_ENV", "development"),
		AppPort:             getEnv("PORT", "8080"),
		MCPTransport:        getEnv("MCP_TRANSPORT", "http"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "gosling_db"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		JWTSecret:           getEnv("JWT_SECRET", "super-secret-default-jwt-signing-key-change-in-prod"),
		GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:   getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/mcp"),
		GeminiAPIKey:        getEnv("GEMINI_API_KEY", os.Getenv("GOOGLE_API_KEY")),
		GeminiDefaultModel:  getEnv("GEMINI_DEFAULT_MODEL", "gemini-2.5-flash"),
		AllowedOrigins:      getEnv("ALLOWED_ORIGINS", "*"),
	}

	expiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "72")
	hours, err := strconv.Atoi(expiryHoursStr)
	if err != nil {
		hours = 72
	}
	cfg.JWTExpiryHours = time.Duration(hours) * time.Hour

	return cfg, nil
}

// DatabaseDSN constructs the PostgreSQL connection string.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getDockerVersion() string {
	for _, key := range []string{"DOCKER_IMAGE_VERSION", "CONTAINER_VERSION", "IMAGE_TAG", "APP_VERSION", "VERSION"} {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	if Version != "" {
		return Version
	}
	return "1.0.0"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
