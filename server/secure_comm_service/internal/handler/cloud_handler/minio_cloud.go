package cloud_handler

import (
	"errors"
	"fmt"
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

// CreateMany загружает несколько файлов
// @Summary      Загрузка нескольких файлов
// @Description  Загружает несколько файлов в MinIO через form-data.
// @Tags         Files
// @Accept       mpfd
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        files formData []file true "Массив файлов для загрузки" collectionFormat(multi)
// @Param        mime_type formData []string false "Массив MIME-типов (по индексу)" collectionFormat(multi)
// @Success      200  {array}   dto.FileResponse             "Файлы успешно загружены + данные о файлах"
// @Failure      400  {object}  ErrorResponse            "Некорректная форма или отсутствуют файлы"
// @Failure      403  {object}  ErrorResponse            "Превышена квота"
// @Failure      413  {object}  map[string]string        "Один из файлов слишком большой"
// @Failure      500  {object}  ErrorResponse            "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/many [post]
// func (h *minioHandler) CreateMany(c *gin.Context) {
// 	const op = "location internal.handler.minio_handler.minio.CreateMany"

// 	userID, err := utils.GetUserID(c)
// 	if err != nil {
// 		logrus.Errorf("Errors: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
// 		return
// 	}

// 	form, err := c.MultipartForm()
// 	if err != nil {
// 		logrus.Errorf("Error: %v, %s", err, op)
// 		c.JSON(http.StatusBadRequest, ErrorResponse{
// 			Status:  http.StatusBadRequest,
// 			Error:   "Invalid form",
// 			Details: err,
// 		})
// 		return
// 	}

// 	files := form.File["files"]
// 	if files == nil {
// 		logrus.Errorf("Error: %v, %s", err, op)
// 		c.JSON(http.StatusBadRequest, ErrorResponse{
// 			Status:  http.StatusBadRequest,
// 			Error:   "No files are received",
// 			Details: err,
// 		})
// 		return
// 	}

// 	var totalSize int64
// 	for _, fh := range files {
// 		totalSize += fh.Size
// 	}

// 	mimeTypes := c.PostFormArray("mime_type")
// 	data := make(map[string]domain.FileContent)

// 	for i, file := range files {
// 		f, err := file.Open()
// 		if err != nil {
// 			logrus.Errorf("Error: %v, %s", err, op)
// 			c.JSON(http.StatusInternalServerError, ErrorResponse{
// 				Status:  http.StatusInternalServerError,
// 				Error:   "Unable to open the file",
// 				Details: err,
// 			})
// 			return
// 		}

// 		fileBytes, err := io.ReadAll(f)
// 		if err != nil {
// 			if errors.Is(err, middleware.ErrFileTooLarge) {
// 				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
// 					"error": fmt.Sprintf("file %q is too large: limit is %d bytes", file.Filename, middleware.MaxFileSize),
// 				})
// 			} else {
// 				logrus.Errorf("Error: %v, %s", err, op)
// 				c.JSON(http.StatusInternalServerError, ErrorResponse{
// 					Status:  http.StatusInternalServerError,
// 					Error:   "Unable to read the file",
// 					Details: err,
// 				})
// 			}
// 			f.Close()
// 			return
// 		}

// 		var fileFormat string
// 		if i < len(mimeTypes) && mimeTypes[i] != "" {
// 			fileFormat = mimeTypes[i]
// 		} else {
// 			fileFormat = http.DetectContentType(fileBytes)
// 		}

// 		now := time.Now().UTC()
// 		data[file.Filename] = domain.FileContent{
// 			Name:      file.Filename,
// 			Format:    fileFormat,
// 			CreatedAt: now,
// 			Data:      fileBytes,
// 		}

// 		logrus.Infof("USER-ID:%d FILE DATA... fileFormat:%s, fileName: %s, CreatedAt: %v", userID, fileFormat, file.Filename, now)
// 	}

// 	fileRespes, err := h.minioService.CreateMany(c, data, userID)
// 	if err != nil {
// 		logrus.Errorf("Error: %v, %s", err, op)
// 		c.JSON(http.StatusInternalServerError, ErrorResponse{
// 			Status:  http.StatusInternalServerError,
// 			Error:   "Unable to save the files",
// 			Details: err,
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":    http.StatusOK,
// 		"message":   "Files uploaded successfully",
// 		"file_data": fileRespes,
// 	})
// }

// GetOne возвращает предварительно подписанную ссылку на скачивание одного файла
// @Summary      Получение одного файла
// @Description  Возвращает пре‐подписанную ссылку на скачивание одного файла по ID и типу.
// @Tags         Files
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        id query string true "Идентификатор объекта"
// @Param        type query string true "Категория файла (photo, unknown, video, text)"
// @Success      200  {object}  dto.FileResponse   "Ссылка на скачивание файла"
// @Failure      400  {object}  ErrorResponse  "Некорректный запрос"
// @Failure      403  {object}  ErrorResponse  "Доступ запрещён"
// @Failure      404  {object}  ErrorResponse  "Файл не найден"
// @Failure      500  {object}  ErrorResponse  "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/one [get]
// func (h *minioHandler) GetOne(c *gin.Context) {
// 	const op = "location internal.handler.minio_handler.minio.GetOne"

// 	userID, err := utils.GetUserID(c)
// 	if err != nil {
// 		logrus.Errorf("Errors: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
// 		return
// 	}

// 	objectID := dto.ObjectID{
// 		ObjID:        c.Query("id"),
// 		FileCategory: c.Query("type"),
// 	}

// 	logrus.Infof("objectID... ID:%s, userID:%d, FileCategory:%s", objectID.ObjID, userID, objectID.FileCategory)

// 	fileResp, err := h.minioService.GetOne(c, objectID, userID)
// 	if err != nil {
// 		logrus.Errorf("Error: %v,  %s", err, op)

// 		if errors.Is(err, cloud_service.ErrFileNotFound) {
// 			c.JSON(http.StatusNotFound, ErrorResponse{
// 				Status:  http.StatusNotFound,
// 				Error:   "File not found",
// 				Details: fmt.Sprintf("%v, file category: %s", err.Error(), objectID.FileCategory),
// 			})
// 			return
// 		}

// 		if errors.Is(err, cloud_service.ErrForbiddenResource) {
// 			c.JSON(http.StatusForbidden, ErrorResponse{
// 				Status:  http.StatusForbidden,
// 				Error:   "access to the requested resource is prohibited",
// 				Details: err.Error(),
// 			})
// 			return
// 		}

// 		c.JSON(http.StatusInternalServerError, ErrorResponse{
// 			Status:  http.StatusInternalServerError,
// 			Error:   "Enable to get the object",
// 			Details: err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":    http.StatusOK,
// 		"message":   "File received successfully",
// 		"file_data": fileResp,
// 	})
// }

// GetMany возвращает несколько файлов (список ID) в виде ссылок
// @Summary      Получение нескольких файлов
// @Description  Возвращает пре‐подписанные ссылки на скачивание нескольких файлов.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        objectIDs body dto.ObjectIDs true "Массив идентификаторов объектов"
// @Success      200  {array}   dto.FileResponse    "Ссылки на скачивание файлов"
// @Failure      400  {object}  ErrorResponse   "Некорректный JSON в теле запроса"
// @Failure      403  {object}  ErrorResponse   "Доступ запрещён"
// @Failure      404  {object}  ErrorResponse   "Один из файлов не найден"
// @Failure      500  {object}  ErrorResponse   "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/many [get]
// func (h *minioHandler) GetMany(c *gin.Context) {
// 	const op = "location internal.handler.minio_handler.minio.GetMany"

// 	userID, err := utils.GetUserID(c)
// 	if err != nil {
// 		logrus.Errorf("Errors: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
// 		return
// 	}

// 	var objectIDs dto.ObjectIDs
// 	if err := c.ShouldBindJSON(&objectIDs); err != nil {
// 		logrus.Errorf("Error: %v,  %s", err, op)
// 		c.JSON(http.StatusBadRequest, ErrorResponse{
// 			Status:  http.StatusBadRequest,
// 			Error:   "Invalid request body",
// 			Details: err,
// 		})
// 		return
// 	}

// 	logrus.Infof("ObjectIDsDto: %v \n", objectIDs)

// 	fileResp, errs := h.minioService.GetMany(c, objectIDs.ObjectIDs, userID)
// 	for _, err := range errs {
// 		if err != nil {
// 			logrus.Errorf("Error: %v,  %s", err, op)

// 			if errors.Is(err, cloud_service.ErrFileNotFound) {
// 				c.JSON(http.StatusNotFound, ErrorResponse{
// 					Status:  http.StatusNotFound,
// 					Error:   "File not found",
// 					Details: fmt.Sprintf("%v", err.Error()),
// 				})
// 				return
// 			}

// 			if errors.Is(err, cloud_service.ErrForbiddenResource) {
// 				c.JSON(http.StatusForbidden, ErrorResponse{
// 					Status:  http.StatusForbidden,
// 					Error:   "access to the requested resource is prohibited",
// 					Details: err.Error(),
// 				})
// 				return
// 			}

// 			c.JSON(http.StatusInternalServerError, ErrorResponse{
// 				Status:  http.StatusInternalServerError,
// 				Error:   "Enable to get many objects",
// 				Details: err,
// 			})
// 			return
// 		}
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":    http.StatusOK,
// 		"message":   "Files received successfully",
// 		"file_data": fileResp,
// 	})
// }

// GetAll возвращает все файлы указанной категории
// @Summary      Получение всех файлов категории
// @Description  Возвращает пре‐подписанные ссылки на скачивание всех файлов заданной категории (photo, unknown, video, text).
// @Tags         Files
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        type query string true "Категория файлов (photo, unknown, video, text)"
// @Success      200  {array}   dto.FileResponse    "Список ссылок на все файлы категории"
// @Failure      400  {object}  map[string]string "Некорректная категория"
// @Failure      403  {object}  ErrorResponse   "Доступ запрещён"
// @Failure      404  {object}  ErrorResponse   "Файлы не найдены"
// @Failure      500  {object}  ErrorResponse   "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/all [get]
func (h *minioHandler) GetAll(c *gin.Context) {
	const op = "location internal.handler.minio_handler.minio.GetAll"

	userID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("Errors: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
		return
	}

	t := c.Query("type")
	if t != "photo" && t != "unknown" && t != "video" && t != "text" {
		logrus.Infof("Error: the passed type in the query parameter. It can only be one of these types {photo, unknown, video, text}")
		c.JSON(http.StatusBadRequest, gin.H{"error": "the passed type in the query parameter. It can only be one of these types {photo, unknown, video, text}"})
		return
	}

	fileResp, errs := h.minioService.GetAll(c, t, userID)
	for _, err := range errs {
		if err != nil {
			logrus.Errorf("Error: %v,  %s", err, op)

			if errors.Is(err, cloud_service.ErrFileNotFound) {
				c.JSON(http.StatusNotFound, ErrorResponse{
					Status:  http.StatusNotFound,
					Error:   "File not found",
					Details: fmt.Sprintf("%v", err.Error()),
				})
				return
			}

			if errors.Is(err, cloud_service.ErrForbiddenResource) {
				c.JSON(http.StatusForbidden, ErrorResponse{
					Status:  http.StatusForbidden,
					Error:   "access to the requested resource is prohibited",
					Details: err.Error(),
				})
				return
			}

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Status:  http.StatusInternalServerError,
				Error:   "Enable to get many objects",
				Details: err,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    http.StatusOK,
		"message":   "All Files received successfully",
		"file_data": fileResp,
	})
}

// DeleteOne удаляет один файл
// @Summary      Удаление одного файла
// @Description  Удаляет один объект из MinIO и снижает использование квоты.
// @Tags         Files
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        id query string true "Идентификатор объекта"
// @Param        type query string true "Категория файла (photo, unknown, video, text)"
// @Success      200  {object}  map[string]string   "Файл успешно удалён"
// @Failure      400  {object}  ErrorResponse       "Некорректный запрос"
// @Failure      403  {object}  ErrorResponse       "Доступ запрещён"
// @Failure      404  {object}  ErrorResponse       "Файл не найден"
// @Failure      500  {object}  ErrorResponse       "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/one [delete]
// func (h *minioHandler) DeleteOne(c *gin.Context) {
// 	const op = "location internal.handler.minio_handler.minio.DeleteOne"

// 	userID, err := utils.GetUserID(c)
// 	if err != nil {
// 		logrus.Errorf("Errors: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
// 		return
// 	}

// 	objectID := dto.ObjectID{
// 		ObjID:        c.Query("id"),
// 		FileCategory: c.Query("type"),
// 	}

// 	logrus.Infof("objectID... ID:%s, userID:%d, FileCategory:%s", objectID.ObjID, userID, objectID.FileCategory)

// 	_, err = h.minioService.DeleteOne(c, objectID, userID)
// 	if err != nil {
// 		logrus.Errorf("Error: %v,  %s", err, op)

// 		if errors.Is(err, cloud_service.ErrFileNotFound) {
// 			c.JSON(http.StatusNotFound, ErrorResponse{
// 				Status:  http.StatusNotFound,
// 				Error:   "File not found",
// 				Details: fmt.Sprintf("%v", err.Error()),
// 			})
// 			return
// 		}

// 		if errors.Is(err, cloud_service.ErrForbiddenResource) {
// 			c.JSON(http.StatusForbidden, ErrorResponse{
// 				Status:  http.StatusForbidden,
// 				Error:   "access to the requested resource is prohibited",
// 				Details: err.Error(),
// 			})
// 			return
// 		}

// 		c.JSON(http.StatusInternalServerError, ErrorResponse{
// 			Status:  http.StatusInternalServerError,
// 			Error:   "Cannot delete the object",
// 			Details: err,
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  http.StatusOK,
// 		"message": "File deleted successfully",
// 	})
// }

// DeleteMany удаляет несколько файлов
// @Summary      Удаление нескольких файлов
// @Description  Удаляет несколько объектов из MinIO, переданных в JSON-массиве, и снижает использование квоты.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {token}"
// @Param        objectIDs body dto.ObjectIDs true "Массив идентификаторов объектов"
// @Success      200  {object}  map[string]string   "Файлы успешно удалены"
// @Failure      400  {object}  ErrorResponse       "Некорректный JSON в теле запроса"
// @Failure      403  {object}  ErrorResponse       "Доступ запрещён"
// @Failure      404  {object}  ErrorResponse       "Один из файлов не найден"
// @Failure      500  {object}  ErrorResponse       "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /files/many [delete]
// func (h *minioHandler) DeleteMany(c *gin.Context) {
// 	const op = "location internal.handler.minio_handler.minio.DeleteMany"

// 	userID, err := utils.GetUserID(c)
// 	if err != nil {
// 		logrus.Errorf("Errors: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
// 		return
// 	}

// 	var objectIDs dto.ObjectIDs
// 	if err := c.ShouldBindJSON(&objectIDs); err != nil {
// 		logrus.Errorf("Error: %v,  %s", err, op)
// 		c.JSON(http.StatusBadRequest, ErrorResponse{
// 			Status:  http.StatusBadRequest,
// 			Error:   "Invalid request body",
// 			Details: err,
// 		})
// 		return
// 	}

// 	logrus.Infof("ObjectIDsDto: %v \n", objectIDs)

// 	sizes, errs := h.minioService.DeleteMany(c, objectIDs.ObjectIDs, userID)
// 	for _, err := range errs {
// 		if err != nil {
// 			logrus.Errorf("Error: %v,  %s", err, op)

// 			if errors.Is(err, cloud_service.ErrFileNotFound) {
// 				c.JSON(http.StatusNotFound, ErrorResponse{
// 					Status:  http.StatusNotFound,
// 					Error:   "File not found",
// 					Details: fmt.Sprintf("%v", err.Error()),
// 				})
// 				return
// 			}

// 			if errors.Is(err, cloud_service.ErrForbiddenResource) {
// 				c.JSON(http.StatusForbidden, ErrorResponse{
// 					Status:  http.StatusForbidden,
// 					Error:   "access to the requested resource is prohibited",
// 					Details: err.Error(),
// 				})
// 				return
// 			}

// 			c.JSON(http.StatusInternalServerError, ErrorResponse{
// 				Status:  http.StatusInternalServerError,
// 				Error:   "Enable to get many objects",
// 				Details: err,
// 			})
// 			return
// 		}
// 	}

// 	var totalRemoved int64
// 	for _, sz := range sizes {
// 		totalRemoved += sz
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  http.StatusOK,
// 		"message": "Files deleted successfully",
// 	})
// }
