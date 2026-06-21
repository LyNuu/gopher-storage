package main

import (
	"context"
	echojwt "github.com/labstack/echo-jwt/v5"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"storage-service/internal/connections"
	"storage-service/internal/repository"
	"storage-service/internal/s3/pydio"
	"storage-service/internal/service"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	api "storage-service/internal/api/http"
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
	e.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
	}))
	e.Validator = &api.RequestValidator{Validator: validator.New()}
	e.HTTPErrorHandler = api.CustomHTTPErrorHandler
	pool := connections.InitPool(ctx)
	defer pool.Close()

	pydioClient := pydio.NewPydioStorage(
		os.Getenv("PYDIO_BASE_URL"),
		os.Getenv("API_KEY"),
		"gatewaysecret", // фиксированная константа протокола Pydio S3-gateway, не секрет в смысле безопасности
	)
	storageRepository := repository.NewStorageRepository(pool)
	storageService := service.NewStorageService(storageRepository, pydioClient)

	storageHandler := api.NewStorageHandler([]byte(os.Getenv("JWT_SECRET")), storageService)
	storageHandler.RegisterRoute(e)

	srv := &http.Server{
		Addr:    ":8082",
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
