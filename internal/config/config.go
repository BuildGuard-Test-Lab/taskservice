package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port            string
	Environment     string
	Version         string
	Debug           bool
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// Database
	DatabaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		Version:         getEnv("APP_VERSION", "dev"),
		Debug:           getEnvBool("DEBUG", false),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return defaultValue
		}
		return d
	}
	return defaultValue
}
