package config

import (
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		os.Clearenv()
		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.HTTPPort != "8080" {
			t.Errorf("expected 8080, got %s", cfg.HTTPPort)
		}
	})

	t.Run("env values", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("HTTP_PORT", "9090")
		os.Setenv("REDIS_DB", "5")
		os.Setenv("BUCKET_TTL", "1h")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.HTTPPort != "9090" {
			t.Errorf("expected 9090, got %s", cfg.HTTPPort)
		}
		if cfg.RedisDB != 5 {
			t.Errorf("expected 5, got %d", cfg.RedisDB)
		}
		if cfg.BucketTTL != time.Hour {
			t.Errorf("expected 1h, got %v", cfg.BucketTTL)
		}
	})

	t.Run("invalid values fallback", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("REDIS_DB", "invalid")
		os.Setenv("BUCKET_TTL", "invalid")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.RedisDB != 0 {
			t.Errorf("expected fallback 0, got %d", cfg.RedisDB)
		}
		if cfg.BucketTTL != 10*time.Minute {
			t.Errorf("expected fallback 10m, got %v", cfg.BucketTTL)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("LOGIN_LIMIT", "-1")
		_, err := LoadFromEnv()
		if err == nil {
			t.Fatal("expected error for negative limit")
		}

		os.Clearenv()
		os.Setenv("BUCKET_TTL", "-1m")
		_, err = LoadFromEnv()
		if err == nil {
			t.Fatal("expected error for negative ttl")
		}
	})
}
