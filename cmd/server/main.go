package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aleka7sk/booking-os/internal/app"
	"github.com/aleka7sk/booking-os/internal/httpapi"
	"github.com/aleka7sk/booking-os/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	addr := env("BOOKING_OS_ADDR", ":8080")
	dataPath := env("BOOKING_OS_DATA_PATH", "./data/booking-os.json")
	adminEmail := env("BOOKING_OS_ADMIN_EMAIL", "owner@booking.local")
	adminPassword := env("BOOKING_OS_ADMIN_PASSWORD", "demo1234")
	sessionSecret := env("BOOKING_OS_SESSION_SECRET", "development-change-me-before-production")
	publicURL := env("BOOKING_OS_PUBLIC_URL", "http://localhost:8080")

	if sessionSecret == "development-change-me-before-production" || sessionSecret == "development-change-me" {
		logger.Warn("default session secret is active; replace BOOKING_OS_SESSION_SECRET before external access")
	}
	if adminPassword == "demo1234" {
		logger.Warn("default owner password is active; replace BOOKING_OS_ADMIN_PASSWORD before external access")
	}

	st, err := store.Open(dataPath, store.DemoSeed(adminEmail, adminPassword))
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	service := app.New(st)
	api := httpapi.New(service, httpapi.Config{Addr: addr, SessionSecret: sessionSecret, PublicURL: publicURL}, logger)

	server := &http.Server{Addr: addr, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if count, err := service.ExpireHolds(); err != nil {
				logger.Error("expire holds", "error", err)
			} else if count > 0 {
				logger.Info("expired holds", "count", count)
			}
		}
	}()
	go func() {
		logger.Info("booking-os started", "addr", addr, "public_url", publicURL, "data_path", dataPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "error", err)
	}
	logger.Info("booking-os stopped")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
