package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all validated configuration for the TelemetryHealth control plane services.
type Config struct {
	Env                   string
	Port                  int
	AllowedOrigins        string
	OIDCIssuer            string
	InsecureDevMode       bool
	ClickHouseHost        string
	ClickHousePort        string
	ClickHouseUser        string
	ClickHousePassword    string
	ClickHouseDatabase    string
	IngestGatewayEndpoint string
}

// LoadConfig loads configuration from environment variables with sensible production defaults.
func LoadConfig() *Config {
	env := getEnv("ENV", "development")
	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 8080
	}

	allowedOrigins := getEnv("ALLOWED_ORIGINS", "")
	if allowedOrigins == "" {
		allowedOrigins = getEnv("CORS_ORIGIN", "")
	}
	if allowedOrigins == "" {
		if strings.ToLower(env) == "production" {
			allowedOrigins = "http://localhost:5173"
		} else {
			allowedOrigins = "*"
		}
	}

	return &Config{
		Env:                   env,
		Port:                  port,
		AllowedOrigins:        allowedOrigins,
		OIDCIssuer:            getEnv("OIDC_ISSUER", ""),
		InsecureDevMode:       getEnv("INSECURE_DEV_MODE", "false") == "true",
		ClickHouseHost:        getEnv("CH_HOST", "127.0.0.1"),
		ClickHousePort:        getEnv("CH_PORT", "9000"),
		ClickHouseUser:        getEnv("CH_USER", "default"),
		ClickHousePassword:    getEnv("CH_PASSWORD", ""),
		ClickHouseDatabase:    getEnv("CH_DATABASE", "telemetry_health"),
		IngestGatewayEndpoint: getEnv("INGEST_GATEWAY_ENDPOINT", "127.0.0.1:4317"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
