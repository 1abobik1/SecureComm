package cloud_handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/1abobik1/SecureComm/internal/service/cloud_service"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

// CreateOneEncrypted – загружает поток зашифрованного файла в MinIO “на лету”.
// Клиент должен передать:
//   - Authorization: Bearer <token>
//   - Content-Type: application/octet-stream
//   - X-Orig-Filename: <имя файла, напр. photo.jpg>
//   - X-Orig-Mime: <исходный mime, напр. image/jpeg>
//   - X-File-Category: <photo|video|text|unknown>
//
// Тело запроса (body) — это уже полностью зашифрованный поток (будь-то AES-CBC+HMAC по чанкам).
//
// @Summary      Загрузка зашифрованного файла “на лету”
// @Description  Токен авторизации + зашифрованный поток в body + метаданные в заголовках
// @Tags         Files
// @Accept       */*
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        X-Orig-Filename header string true "Оригинальное имя файла (например photo.jpg)"
// @Param        X-Orig-Mime     header string true "Оригинальный MIME-тип (например image/jpeg)"
// @Param        X-File-Category header string true "Категория файла (photo, video, text, unknown)"
// @Success      200  {object}  dto.FileResponse  "Успешно: JSON с metadata + presigned URL"
// @Failure      400  {object}  map[string]string "Некорректные заголовки или body"
// @Failure      401  {object}  map[string]string "Проблема с авторизацией"
// @Failure      500  {object}  map[string]string "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/one/encrypted [post]
func (h *minioHandler) CreateOneEncrypted(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	origName := c.GetHeader("X-Orig-Filename")
	origMime := c.GetHeader("X-Orig-Mime")
	category := strings.ToLower(c.GetHeader("X-File-Category"))
	if origName == "" || origMime == "" || category == "" {
		logrus.Error("missing X-Orig-Filename/X-Orig-Mime/X-File-Category")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing X-Orig-Filename/X-Orig-Mime/X-File-Category"})
		return
	}
	if category != "photo" && category != "video" && category != "text" && category != "unknown" {
		logrus.Error("invalid X-File-Category(video or text or unknown or photo)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-File-Category(video or text or unknown or photo)"})
		return
	}

	objID := cloud_service.GenerateFileID(userID, cloud_service.GetFileExtension(origName))
	metadata := cloud_service.GenerateUserMetaData(userID, origName, time.Now().UTC())
	opts := minio.PutObjectOptions{
		ContentType:  origMime,
		UserMetadata: metadata,
	}

	// c.Request.Body — это io.Reader, из которого MinIO SDK сам делает chunked-upload
	_, err = h.minioService.PutEncryptedObject(
		c.Request.Context(),
		category,
		objID,
		c.Request.Body,
		-1,
		opts,
	)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}

	presignedURL, err := h.minioService.PresignedGetURL(c.Request.Context(), category, objID)
	if err != nil {
		logrus.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate URL"})
		return
	}

	// Собираем ответ и кешируем
	fileResp := dto.FileResponse{
		Name:       origName,
		Created_At: time.Now().UTC().Format(time.RFC3339),
		ObjID:      objID,
		Url:        presignedURL.String(),
		MimeType:   origMime,
	}
	if err := h.minioService.CacheFileResponse(c.Request.Context(), category, objID, fileResp); err != nil {

	}

	c.JSON(http.StatusOK, fileResp)
}