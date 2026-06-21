package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"

	api "auth-service/internal/api/http"
	"auth-service/internal/connections"
	auth_redis "auth-service/internal/redis"
	"auth-service/internal/repository"
	"auth-service/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	err := godotenv.Load()
	if err != nil {
		slog.Info("Error loading .env file")
	}

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Validator = &api.RequestValidator{Validator: validator.New()}
	e.HTTPErrorHandler = api.CustomHTTPErrorHandler
	pool := connections.InitPool(ctx)
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	authCache := auth_redis.NewAuthCacheRepository(rdb)

	authRepository := repository.NewAuthRepository(pool)
	authService := service.NewAuthService(authRepository, authCache,
		[]byte(os.Getenv("JWT_SECRET")), 24*60*time.Minute, 2*24*60*time.Minute)

	authHandler := api.NewAuthHandler(authService)
	authHandler.RegisterRoute(e)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: e,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			slog.Warn("service error", "error", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("shutdown error", "error", err)
	}
}
