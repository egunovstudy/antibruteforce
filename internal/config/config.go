package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	LoginLimit    int
	PasswordLimit int
	IPLimit       int
	BucketTTL     time.Duration
	RefillPeriod  time.Duration
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		LoginLimit:    getEnvInt("LOGIN_LIMIT", 10),
		PasswordLimit: getEnvInt("PASSWORD_LIMIT", 100),
		IPLimit:       getEnvInt("IP_LIMIT", 1000),
		BucketTTL:     getEnvDuration("BUCKET_TTL", 10*time.Minute),
		RefillPeriod:  getEnvDuration("REFILL_PERIOD", time.Minute),
	}

	if cfg.LoginLimit <= 0 || cfg.PasswordLimit <= 0 || cfg.IPLimit <= 0 {
		return Config{}, fmt.Errorf("limits must be positive")
	}
	if cfg.BucketTTL <= 0 || cfg.RefillPeriod <= 0 {
		return Config{}, fmt.Errorf("bucket ttl and refill period must be positive")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}
