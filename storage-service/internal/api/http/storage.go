package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"storage-service/internal/model"
	"storage-service/internal/service"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrBadRequest   = errors.New("bad request")
)

type StorageHandler struct {
	jwtSecret      []byte
	publicBaseURL  string
	storageService storageService
}

func NewStorageHandler(jwtSecret []byte, publicBaseURL string, svc storageService) *StorageHandler {
	return &StorageHandler{
		jwtSecret:      jwtSecret,
		publicBaseURL:  publicBaseURL,
		storageService: svc,
	}
}

type grantAccessRequest struct {
	UserID string            `json:"user_id" validate:"required,uuid"`
	Level  model.AccessLevel `json:"level" validate:"required,oneof=read write"`
}

type shareRequest struct {
	TTLSeconds int64 `json:"ttl_seconds"`
}

func (h *StorageHandler) CreateStorage(c *echo.Context) error {
	userID := userIDFromCtx(c)

	var in model.CreateStorageInput
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&in); err != nil {
		return err
	}

	st, err := h.storageService.CreateStorage(c.Request().Context(), userID, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, st)
}

func (h *StorageHandler) ListStorages(c *echo.Context) error {
	userID := userIDFromCtx(c)

	list, err := h.storageService.ListStorages(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	if list == nil {
		list = []model.Storage{}
	}
	return c.JSON(http.StatusOK, list)
}

func (h *StorageHandler) GetStorage(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}

	st, err := h.storageService.GetStorage(c.Request().Context(), userID, storageID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, st)
}

func (h *StorageHandler) UploadFile(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
	}
	files := form.File["file"]
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required, use 'file' key")
	}
	header := files[0]

	src, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer src.Close()

	f, err := h.storageService.UploadFile(
		c.Request().Context(), userID, storageID, header.Filename, src, header.Size, header.Header.Get("Content-Type"),
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, f)
}

func (h *StorageHandler) DownloadFile(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}
	name := c.Param("name")

	stream, size, contentType, err := h.storageService.DownloadFile(c.Request().Context(), userID, storageID, name)
	if err != nil {
		return err
	}
	defer stream.Close()
	return h.streamFile(c, stream, size, contentType, name)
}

func (h *StorageHandler) ListFiles(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}

	files, err := h.storageService.ListFiles(c.Request().Context(), userID, storageID)
	if err != nil {
		return err
	}
	if files == nil {
		files = []model.File{}
	}
	return c.JSON(http.StatusOK, files)
}

func (h *StorageHandler) DeleteFile(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}
	name := c.Param("name")

	if err := h.storageService.DeleteFile(c.Request().Context(), userID, storageID, name); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *StorageHandler) ShareFile(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}
	name := c.Param("name")

	var req shareRequest
	_ = c.Bind(&req)
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}

	sh, err := h.storageService.ShareFile(c.Request().Context(), userID, storageID, name, ttl)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"token":      sh.Token,
		"url":        h.publicBaseURL + "/api/v1/shared/" + sh.Token,
		"expires_at": sh.ExpiresAt,
	})
}

func (h *StorageHandler) DownloadShared(c *echo.Context) error {
	token := c.Param("token")

	stream, size, contentType, name, err := h.storageService.GetSharedFile(c.Request().Context(), token)
	if err != nil {
		return err
	}
	defer stream.Close()
	return h.streamFile(c, stream, size, contentType, name)
}

func (h *StorageHandler) RevokeShare(c *echo.Context) error {
	userID := userIDFromCtx(c)
	token := c.Param("token")

	if err := h.storageService.RevokeShare(c.Request().Context(), userID, token); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *StorageHandler) CreateUploadLink(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}
	name := c.Param("name")

	link, err := h.storageService.CreateUploadLink(c.Request().Context(), userID, storageID, name, 24*time.Hour)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"upload_url": link})
}

func (h *StorageHandler) GrantAccess(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}

	var req grantAccessRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	target, err := uuid.Parse(req.UserID)
	if err != nil {
		return ErrBadRequest
	}

	if err := h.storageService.GrantAccess(c.Request().Context(), userID, storageID, target, req.Level); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *StorageHandler) ListAccess(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}

	list, err := h.storageService.ListAccess(c.Request().Context(), userID, storageID)
	if err != nil {
		return err
	}
	if list == nil {
		list = []model.StorageAccess{}
	}
	return c.JSON(http.StatusOK, list)
}

func (h *StorageHandler) RevokeAccess(c *echo.Context) error {
	userID := userIDFromCtx(c)

	storageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrBadRequest
	}
	target, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return ErrBadRequest
	}

	if err := h.storageService.RevokeAccess(c.Request().Context(), userID, storageID, target); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *StorageHandler) streamFile(c *echo.Context, stream io.Reader, size int64, contentType, name string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	if size > 0 {
		c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", size))
	}
	return c.Stream(http.StatusOK, contentType, stream)
}

func (h *StorageHandler) authUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		token, err := echo.ContextGet[*jwt.Token](c, "user")
		if err != nil {
			return ErrUnauthorized
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return ErrUnauthorized
		}
		idStr, ok := claims["user_id"].(string)
		if !ok {
			return ErrBadRequest
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			return ErrBadRequest
		}
		c.Set("user_id", id)
		return next(c)
	}
}

func userIDFromCtx(c *echo.Context) uuid.UUID {
	id, _ := c.Get("user_id").(uuid.UUID)
	return id
}

func (h *StorageHandler) RegisterRoute(e *echo.Echo) {
	e.GET("/api/v1/shared/:token", h.DownloadShared)

	jwtMW := echojwt.WithConfig(echojwt.Config{
		SigningKey: h.jwtSecret,
	})
	api := e.Group("/api/v1", jwtMW, h.authUser)

	api.POST("/storages", h.CreateStorage)
	api.GET("/storages", h.ListStorages)
	api.GET("/storages/:id", h.GetStorage)
	api.POST("/storages/:id/files", h.UploadFile, middleware.BodyLimit(5<<30))
	api.GET("/storages/:id/files", h.ListFiles)
	api.GET("/storages/:id/files/:name", h.DownloadFile)
	api.DELETE("/storages/:id/files/:name", h.DeleteFile)
	api.POST("/storages/:id/files/:name/share", h.ShareFile)
	api.POST("/storages/:id/files/:name/upload-link", h.CreateUploadLink)
	api.POST("/storages/:id/access", h.GrantAccess)
	api.GET("/storages/:id/access", h.ListAccess)
	api.DELETE("/storages/:id/access/:userId", h.RevokeAccess)
	api.DELETE("/shares/:token", h.RevokeShare)
}

type RequestValidator struct {
	Validator *validator.Validate
}

func (rv *RequestValidator) Validate(i any) error {
	if err := rv.Validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func CustomHTTPErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
		if resp.Committed {
			return
		}
	}

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		_ = c.JSON(echoErr.Code, map[string]any{"error": echoErr.Message})
		return
	}

	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, service.ErrAlreadyExists):
		code = http.StatusConflict
	case errors.Is(err, service.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, service.ErrQuotaExceeded), errors.Is(err, service.ErrFileTooLarge):
		code = http.StatusRequestEntityTooLarge
	case errors.Is(err, service.ErrMimeNotAllowed), errors.Is(err, service.ErrInvalidFileName):
		code = http.StatusBadRequest
	case errors.Is(err, service.ErrShareExpired):
		code = http.StatusGone
	case errors.Is(err, ErrBadRequest):
		code = http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		code = http.StatusUnauthorized
	}

	if code == http.StatusInternalServerError {
		_ = c.JSON(code, map[string]any{"error": "internal server error"})
		return
	}
	_ = c.JSON(code, map[string]any{"error": err.Error()})
}
