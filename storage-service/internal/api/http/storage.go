package http

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"net/http"
	"time"
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
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}

	err = h.storageService.CreateStorage(c.Request().Context(), userUUID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"status": "success",
	})

}

func (h *StorageHandler) UploadFile(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
	}

	formFiles := form.File["file"]
	if len(formFiles) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to get file from request, check 'file' key")
	}
	fileHeader := formFiles[0]

	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open file")
	}
	defer file.Close()

	fileName := fileHeader.Filename
	fileSize := fileHeader.Size
	contentType := fileHeader.Header.Get("Content-Type")

	err = h.storageService.UploadUserFile(c.Request().Context(), userUUID, fileName, file, fileSize, contentType)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":    "success",
		"file_name": fileName,
	})
}

func (h *StorageHandler) DownloadFile(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}
	fileName := c.QueryParam("filename")

	// 1. Получаем поток из Pydio
	stream, size, contentType, err := h.storageService.DownloadUserFile(c.Request().Context(), userUUID, fileName)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	defer stream.Close()

	// 2. Устанавливаем заголовки для скачивания
	c.Response().Header().Set(echo.HeaderContentType, contentType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", size))

	// 3. Отправляем поток клиенту
	return c.Stream(http.StatusOK, contentType, stream)
}

func (h *StorageHandler) DeleteFile(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}
	fileName := c.Param("filename") // Получаем имя из URL: /api/v1/storage/delete/:filename

	if fileName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename is required")
	}

	err = h.storageService.DeleteUserFile(c.Request().Context(), userUUID, fileName)
	if err != nil {
		// Если ошибка в S3, отдаем 500, или 404 если файл не найден
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete file: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "file deleted successfully",
	})
}

func (h *StorageHandler) ShareFile(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}
	fileName := c.QueryParam("filename")

	// Генерируем ссылку, которая будет жить 1 час
	link, err := h.storageService.GetShareableLink(c.Request().Context(), userUUID, fileName, 1*time.Hour)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate link")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"url": link,
	})
}

func (h *StorageHandler) CreateUploadLink(c *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrUnauthorized
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return ErrBadRequest
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrBadRequest
	}

	link, err := h.storageService.CreateUploadLink(
		c.Request().Context(),
		userUUID,
		"go-way.jpg",
		24*time.Hour,
	)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			err.Error(),
		)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"upload_url": link,
	})
}

func (h *StorageHandler) RegisterRoute(c *echo.Echo) {
	api := c.Group("/api/v1/storage")
	api.POST("/create", h.CreateStorage)
	api.POST("/upload", h.UploadFile, middleware.BodyLimit(100<<20))
	api.GET("/download", h.DownloadFile)
	api.DELETE("/delete/:filename", h.DeleteFile)
	api.GET("/share", h.ShareFile)
	api.POST("/share-folder", h.CreateUploadLink)
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

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		_ = c.JSON(echoErr.Code, map[string]any{"error": echoErr.Message})
		return
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
