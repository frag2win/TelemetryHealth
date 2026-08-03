package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("ENV")
	os.Unsetenv("PORT")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("CORS_ORIGIN")

	cfg := LoadConfig()

	if cfg.Env != "development" {
		t.Errorf("expected default Env to be 'development', got %s", cfg.Env)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default Port to be 8080, got %d", cfg.Port)
	}
	if cfg.AllowedOrigins != "*" {
		t.Errorf("expected default AllowedOrigins in dev mode to be '*', got %s", cfg.AllowedOrigins)
	}
}

func TestLoadConfig_ProductionDefaults(t *testing.T) {
	os.Setenv("ENV", "production")
	os.Unsetenv("ALLOWED_ORIGINS")
	os.Unsetenv("CORS_ORIGIN")

	cfg := LoadConfig()

	if cfg.AllowedOrigins != "http://localhost:5173" {
		t.Errorf("expected production AllowedOrigins default to be 'http://localhost:5173', got %s", cfg.AllowedOrigins)
	}

	os.Unsetenv("ENV")
}
