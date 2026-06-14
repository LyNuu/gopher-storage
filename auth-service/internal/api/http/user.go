package http

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	"auth-service/internal/model"
	"auth-service/internal/service"
)

type AuthHandler struct {
	svc authService
}

func NewAuthHandler(svc authService) *AuthHandler {
	return &AuthHandler{
		svc: svc,
	}
}

func (h *AuthHandler) Register(c *echo.Context) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=7"`
	}
	if err := c.Bind(&req); err != nil {
		return ErrBadRequestEOF
	}
	if err := c.Validate(&req); err != nil {
		return ErrBadRequest
	}
	user := model.User{
		Email:    req.Email,
		Password: req.Password,
	}
	if err := h.svc.Create(c.Request().Context(), user); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "success"})
}

func (h *AuthHandler) Login(c *echo.Context) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=7"`
	}
	if err := c.Bind(&req); err != nil {
		return ErrBadRequestEOF
	}
	if err := c.Validate(&req); err != nil {
		return ErrBadRequest
	}
	tokens, err := h.svc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(c *echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return ErrBadRequest
	}

	if err := c.Validate(&req); err != nil {
		return ErrBadRequest
	}

	tokens, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
	})
}

func (h *AuthHandler) RegisterRoute(e *echo.Echo) {
	api := e.Group("/api/v1/auth")
	api.POST("/register", h.Register)
	api.POST("/login", h.Login)
	api.POST("/refresh", h.Refresh)
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
	case errors.Is(err, service.ErrEmailAlreadyExists):
		_ = c.JSON(http.StatusConflict, map[string]any{"error": err.Error()})
		return
	case errors.Is(err, service.ErrUserNotFound):
		_ = c.JSON(http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	case errors.Is(err, service.ErrInvalidCreds):
		_ = c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	_ = c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
}

var (
	ErrBadRequest    = errors.New("invalid email or password")
	ErrBadRequestEOF = errors.New("unexpected EOF")
)
