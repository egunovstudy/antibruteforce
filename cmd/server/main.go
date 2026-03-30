package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"antibf/internal/config"
	"antibf/internal/httpapi"
	"antibf/internal/ratelimit"
	"antibf/internal/service"
	"antibf/internal/storage/redisstore"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() {
		if errClose := rdb.Close(); errClose != nil {
			log.Printf("close redis: %v", errClose)
		}
	}()

	if err = rdb.Ping(ctx).Err(); err != nil {
		log.Panicf("ping redis: %v", err)
	}

	repo := redisstore.NewNetworkRepository(rdb)
	limiter := ratelimit.NewRedisLimiter(rdb, ratelimit.Config{
		LoginLimit:    cfg.LoginLimit,
		PasswordLimit: cfg.PasswordLimit,
		IPLimit:       cfg.IPLimit,
		BucketTTL:     cfg.BucketTTL,
		RefillPeriod:  cfg.RefillPeriod,
	})

	svc := service.New(repo, limiter)
	handler := httpapi.NewHandler(svc)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if errShutdown := server.Shutdown(shutdownCtx); errShutdown != nil {
			log.Printf("http shutdown: %v", errShutdown)
		}
	}()

	log.Printf("anti-bruteforce service started on %s", server.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Panicf("listen server: %v", err)
	}
}
