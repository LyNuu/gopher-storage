package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	api "storage-service/internal/api/http"
	"storage-service/internal/connections"
	"storage-service/internal/repository"
	"storage-service/internal/s3/pydio"
	"storage-service/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := godotenv.Load(); err != nil {
		slog.Info("Error loading .env file")
	}

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))
	e.Validator = &api.RequestValidator{Validator: validator.New()}
	e.HTTPErrorHandler = api.CustomHTTPErrorHandler

	pool := connections.InitPool(ctx)
	defer pool.Close()

	pydioClient := pydio.NewPydioStorage(
		os.Getenv("PYDIO_BASE_URL"),
		os.Getenv("API_KEY"),
		"gatewaysecret",
	)
	storageRepository := repository.NewStorageRepository(pool)
	storageService := service.NewStorageService(storageRepository, pydioClient)

	publicBaseURL := os.Getenv("PUBLIC_BASE_URL")
	if publicBaseURL == "" {
		publicBaseURL = "http://localhost:8082"
	}

	storageHandler := api.NewStorageHandler([]byte(os.Getenv("JWT_SECRET")), publicBaseURL, storageService)
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

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{"*"}
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
