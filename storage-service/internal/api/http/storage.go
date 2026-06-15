package http

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"net/http"
)

var (
	ErrUnauthorized  = errors.New("unauthorized")
	ErrBadRequest    = errors.New("invalid access token")
	ErrBadRequestEOF = errors.New("unexpected EOF")
)

type StorageHandler struct {
	jwtSecret      []byte
	storageService storageService
}

func NewStorageHandler(jwtSecret []byte, service storageService) *StorageHandler {
	return &StorageHandler{
		jwtSecret:      jwtSecret,
		storageService: service,
	}
}

func (h *StorageHandler) CreateStorage(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}
	userIDStr, ok := claims["UserId"].(uuid.UUID)
	if !ok {
		return ErrBadRequest
	}

	err = h.storageService.CreateStorage(c.Request().Context(), userIDStr)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"status": "success",
	})

}

func (h *StorageHandler) RegisterRoute(c *echo.Echo) {
	api := c.Group("/api/v1/storage")
	api.POST("/create", h.CreateStorage)
}

type RequestValidator struct {
	Validator *validator.Validate
}

func (rv *RequestValidator) Validate(i interface{}) error {
	if err := rv.Validator.Struct(i); err != nil {
		return ErrBadRequest
	}
	return nil
}

func CustomHTTPErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
		if resp.Committed {
			return
		}
	}

	switch {
	case errors.Is(err, ErrBadRequest):
		_ = c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	case errors.Is(err, ErrBadRequestEOF):
		_ = c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	case errors.Is(err, ErrUnauthorized):
		_ = c.JSON(http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	_ = c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
}
