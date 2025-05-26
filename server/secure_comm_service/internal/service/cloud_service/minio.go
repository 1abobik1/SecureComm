package cloud_service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/domain"
	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const fileMetaOwnerID = "User_id"
const fileMetaFileName = "File_name"
const fileMetaCreatedAt = "Created_At"

var (
	ErrForbiddenResource = errors.New("access to the requested resource is prohibited")
	ErrFileNotFound      = errors.New("file not found")
)

// Client интерфейс для взаимодействия с Minio
type Client interface {
	InitMinio(minioPort, minioRootUser, minioRootPassword string, minioUseSSL bool) error                       // Метод для инициализации подключения к Minio
	CreateOne(ctx context.Context, file domain.FileContent, userID int) (dto.FileResponse, error)               // Метод для создания одного объекта в бакете Minio
	CreateMany(ctx context.Context, data map[string]domain.FileContent, userID int) ([]dto.FileResponse, error) // Метод для создания нескольких объектов в бакете Minio
	GetOne(ctx context.Context, objectID dto.ObjectID, userID int) (dto.FileResponse, error)                    // Метод для получения одного объекта из бакета Minio
	GetMany(ctx context.Context, objectIDs []dto.ObjectID, userID int) ([]dto.FileResponse, []error)            // Метод для получения нескольких объектов из бакета Minio
	GetAll(ctx context.Context, t string, userID int) ([]dto.FileResponse, []error)                             // Метод для получения всех объектов из конкретного бакета Minio для конкретного пользователя
	DeleteOne(ctx context.Context, objectID dto.ObjectID, userID int) (int64, error)                            // Метод для удаления одного объекта из бакета Minio
	DeleteMany(ctx context.Context, objectIDs []dto.ObjectID, userID int) ([]int64, []error)                    // Метод для удаления нескольких объектов из бакета Minio
	PutEncryptedObject(ctx context.Context, bucket, objectKey string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	PresignedGetURL(ctx context.Context, bucket, objectKey string) (*url.URL, error)
	CacheFileResponse(ctx context.Context, bucket, objectKey string, fileResp dto.FileResponse) error
}

type minioClient struct {
	mc          *minio.Client
	cfg         config.Config
	redisClient *redis.Client
}

func NewMinioClient(cfg config.Config, redisClient *redis.Client) Client {
	return &minioClient{cfg: cfg, redisClient: redisClient}
}

func (m *minioClient) InitMinio(minioPort, minioRootUser, minioRootPassword string, minioUseSSL bool) error {
	ctx := context.Background()

	// Подключение к Minio с использованием имени пользователя и пароля
	client, err := minio.New(minioPort, &minio.Options{
		Creds:  credentials.NewStaticV4(minioRootUser, minioRootPassword, ""),
		Secure: minioUseSSL,
	})
	if err != nil {
		return err
	}

	// Установка подключения Minio
	m.mc = client

	buckets := []string{"photo", "video", "text", "unknown"}

	for _, bucket := range buckets {
		exists, err := m.mc.BucketExists(ctx, bucket)
		if err != nil {
			return err
		}
		if !exists {
			err := m.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *minioClient) PutEncryptedObject(
	ctx context.Context,
	bucket, objectKey string,
	reader io.Reader,
	size int64,
	opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	return m.mc.PutObject(ctx, bucket, objectKey, reader, size, opts)
}

func (m *minioClient) PresignedGetURL(ctx context.Context, bucket, objectKey string) (*url.URL, error) {
	return m.mc.PresignedGetObject(ctx, bucket, objectKey, m.cfg.Minio.UrlTTL, nil)
}

func (m *minioClient) CacheFileResponse(ctx context.Context, bucket, objectKey string, fileResp dto.FileResponse) error {
	key := fmt.Sprintf("filemeta:%s:%s", bucket, objectKey)
	data, err := json.Marshal(fileResp)
	if err != nil {
		return err
	}

	return m.redisClient.Set(ctx, key, data, m.cfg.Redis.MinioUrlTTL).Err()
}

// CreateOne создает один объект в бакете Minio.
// Метод принимает структуру FileData, которая содержит имя файла и его данные.
// В случае успешной загрузки данных в бакет, метод возвращает nil, иначе возвращает ошибку.
// Все операции выполняются в контексте задачи.
func (m *minioClient) CreateOne(ctx context.Context, file domain.FileContent, userID int) (dto.FileResponse, error) {
	const op = "location internal.minio.minio.CreateOne"

	// получение расширения файла
	ext := GetFileExtension(file.Name)
	// генерация file_id
	objID := GenerateFileID(userID, ext)
	// создание метаданных для удобного хранения в minio
	metadata := GenerateUserMetaData(userID, file.Name, file.CreatedAt)

	fileCategory := GetCategory(file.Format)

	options := minio.PutObjectOptions{
		ContentType:  file.Format,
		UserMetadata: metadata,
	}

	log.Printf("METADATA: %v", options.UserMetadata)

	// загрузка в объектное хранилище minio
	_, err := m.mc.PutObject(ctx, fileCategory, objID, bytes.NewReader(file.Data), int64(len(file.Data)), options)
	if err != nil {
		return dto.FileResponse{}, fmt.Errorf("error when creating an object %s: %v", file.Name, err)
	}

	// Получение URL для загруженного объекта
	url, err := m.mc.PresignedGetObject(ctx, fileCategory, objID, m.cfg.Minio.UrlTTL, nil)
	if err != nil {
		return dto.FileResponse{}, fmt.Errorf("error when creating the URL for the object %s: %v", file.Name, err)
	}

	// get Metadata in minio
	objInfo, err := m.mc.StatObject(ctx, fileCategory, objID, minio.StatObjectOptions{})
	if err != nil {
		log.Printf("Error: %v, %s \n", err, op)
		return dto.FileResponse{}, fmt.Errorf("error getting information about the object %s: %w", objID, ErrFileNotFound)
	}

	fileResp := dto.FileResponse{
		Name:       file.Name,
		Created_At: file.CreatedAt.Format(time.RFC3339),
		ObjID:      objID,
		Url:        url.String(),
		MimeType:   objInfo.ContentType,
	}
	// в redis храним только json (не поддерживает структуры)
	fileRespJson, err := json.Marshal(fileResp)
	if err != nil {
		return dto.FileResponse{}, fmt.Errorf("error marshaling FileResponse: %w", err)
	}
	// save in redis
	cacheKey := GetRedisKey(objID, fileCategory)
	err = m.redisClient.Set(ctx, cacheKey, fileRespJson, m.cfg.Redis.MinioUrlTTL).Err()
	if err != nil {
		log.Printf("Failed to save redis, file URL: %v, %s", err, op)
	}

	return fileResp, nil
}

// CreateMany создает несколько объектов в хранилище MinIO из переданных данных.
// Если происходит ошибка при создании объекта, метод возвращает ошибку,
// указывающую на неудачные объекты.
func (m *minioClient) CreateMany(ctx context.Context, data map[string]domain.FileContent, userID int) ([]dto.FileResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Гарантирует освобождение контекста

	resCh := make(chan dto.FileResponse, len(data))
	errCh := make(chan error, len(data))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for objectID, file := range data {
		wg.Add(1)
		go func(objectID string, file domain.FileContent) {
			defer wg.Done()

			fileResponse, err := m.CreateOne(ctx, file, userID)
			if err != nil {
				mu.Lock()
				if firstErr == nil { // Запоминаем только первую ошибку
					firstErr = err
					cancel() // Отменяем все горутины
				}
				mu.Unlock()
				errCh <- err
				return
			}

			// Отправляем URL только если контекст не отменён
			select {
			case <-ctx.Done():
				return
			case resCh <- fileResponse:
			}
		}(objectID, file)
	}

	// Ожидаем завершения всех горутин и закрываем каналы
	go func() {
		wg.Wait()
		close(resCh)
		close(errCh)
	}()

	// Собираем результаты
	var urls []dto.FileResponse
	for fileResponse := range resCh {
		urls = append(urls, fileResponse)
	}

	// Если есть ошибка — возвращаем её
	if firstErr != nil {
		return nil, firstErr
	}

	return urls, nil
}

// GetOne получает один объект из бакета Minio по его идентификатору.
// Он принимает строку `objectID` в качестве параметра и возвращает срез байт данных объекта и ошибку, если такая возникает.
func (m *minioClient) GetOne(ctx context.Context, objectID dto.ObjectID, userID int) (dto.FileResponse, error) {
	const op = "location internal.minio.GetOne"

	var fileResp dto.FileResponse

	// search url in Redis
	cacheKey := GetRedisKey(objectID.ObjID, objectID.FileCategory)
	fileRespJsonInRedis, err := m.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		log.Printf("The data is taken from the redis cache, %s.... cacheKey: %v", op, cacheKey)
		// в redis храниться json, нужно десерелизовать в структуру
		if err := json.Unmarshal([]byte(fileRespJsonInRedis), &fileResp); err != nil {
			return dto.FileResponse{}, fmt.Errorf("error unmarshaling FileResponse: %w", err)
		}
		return fileResp, nil
	} else if err != redis.Nil {
		return dto.FileResponse{}, err
	}

	// get Metadata in minio
	objInfo, err := m.mc.StatObject(ctx, objectID.FileCategory, objectID.ObjID, minio.StatObjectOptions{})
	if err != nil {
		log.Printf("Error: %v, %s \n", err, op)
		return dto.FileResponse{}, fmt.Errorf("error getting information about the object %s: %w", objectID.ObjID, ErrFileNotFound)
	}

	userIdStr, ok := objInfo.UserMetadata[fileMetaOwnerID]
	if !ok {
		log.Printf("Error: %v, %s \n", err, op)
		return dto.FileResponse{}, fmt.Errorf("the user_id metadata was not found for the object %s: %w", objectID.ObjID, ErrFileNotFound)
	}

	userIdInt, err := strconv.Atoi(userIdStr)
	if err != nil {
		return dto.FileResponse{}, fmt.Errorf("error converting string number: %s to int", userIdStr)
	}

	if userIdInt != userID {
		return dto.FileResponse{}, fmt.Errorf("you don't have access rights to other people's files: %w", ErrForbiddenResource)
	}

	// generate url in minio if not in redis
	minioURL, err := m.mc.PresignedGetObject(ctx, objectID.FileCategory, objectID.ObjID, m.cfg.Minio.UrlTTL, nil)
	if err != nil {
		log.Printf("Error: %v, %s", err, op)
		return dto.FileResponse{}, OperationError{ObjectID: objectID.ObjID, Err: fmt.Errorf("error when getting the URL for the object %s: %w", objectID.ObjID, ErrFileNotFound)}
	}
	createdAtStr, okDate := objInfo.UserMetadata[fileMetaCreatedAt]
	if !okDate {
		createdAtStr = objInfo.LastModified.Format(time.RFC3339)
	}

	fileResp.Created_At = createdAtStr
	fileResp.Name = utils.Encode([]byte(objInfo.UserMetadata[fileMetaFileName]))
	fileResp.ObjID = objectID.ObjID
	fileResp.Url = minioURL.String()
	fileResp.MimeType = objInfo.ContentType

	// преобразуем структуру в json для удобного хранения в redis
	fileRespJson, errJson := json.Marshal(fileResp)
	if errJson != nil {
		return dto.FileResponse{}, fmt.Errorf("error marshaling FileResponse: %w", err)
	}
	// save in redis
	err = m.redisClient.Set(ctx, cacheKey, fileRespJson, m.cfg.Redis.MinioUrlTTL).Err()
	if err != nil {
		log.Printf("Failed to save redis, file URL: %v, %s", err, op)
	}

	return fileResp, nil
}

// GetAll получает все объекты из указанного бакета (t соответствует типу файла, например, "photo")
// для заданного пользователя. Он использует префикс "<userID>/" для фильтрации объектов.
// Для каждого найденного объекта генерируется предварительно подписанный URL, который кешируется в Redis.
func (m *minioClient) GetAll(ctx context.Context, t string, userID int) ([]dto.FileResponse, []error) {
	const op = "location internal.minio.GetAll"

	prefix := fmt.Sprintf("%d/", userID)

	// Список объектов из бакета t, удовлетворяющих префиксу
	objectCh := m.mc.ListObjects(ctx, t, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var (
		fileResponses []dto.FileResponse
		errs          []error
	)

	// Для каждого объекта получаем предварительно подписанный URL и добавляем его в список.
	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Error listing object: %v", object.Err)
			errs = append(errs, object.Err)
			continue
		}

		// Для каждого объекта пытаемся получить URL из Redis, если его нет — генерируем заново.
		cacheKey := GetRedisKey(object.Key, t)
		fileRespJsonRedis, err := m.redisClient.Get(ctx, cacheKey).Result()
		var fileResp dto.FileResponse
		if err == nil {
			log.Printf("The data is taken from the redis cache, %s.... cacheKey: %v", op, cacheKey)
			if err := json.Unmarshal([]byte(fileRespJsonRedis), &fileResp); err != nil {
				errs = append(errs, err)
			}
			fileResponses = append(fileResponses, fileResp)
		} else {

			// поиск метаданных в minio
			objInfo, err := m.mc.StatObject(ctx, t, object.Key, minio.StatObjectOptions{})
			if err != nil {
				log.Printf("Error: %v, %s \n", err, op)
				errs = append(errs, err)
			}

			// Если в кеше не найдено, генерируем URL через MinIO
			presignedURL, err := m.mc.PresignedGetObject(ctx, t, object.Key, m.cfg.Minio.UrlTTL, nil)
			if err != nil {
				log.Printf("Error generating presigned URL for object %s: %v", object.Key, err)
				errs = append(errs, err)
				continue
			}
			createdAtStr, okDate := objInfo.UserMetadata[fileMetaCreatedAt]
			if !okDate {
				createdAtStr = objInfo.LastModified.Format(time.RFC3339)
			}

			fileResp.Created_At = createdAtStr
			fileResp.Name = utils.Encode([]byte(objInfo.UserMetadata[fileMetaFileName]))
			fileResp.ObjID = object.Key
			fileResp.Url = presignedURL.String()
			fileResp.MimeType = objInfo.ContentType

			// преобразуем структуру в json
			fileRespJson, err := json.Marshal(fileResp)
			if err != nil {
				errs = append(errs, err)
			}
			// Записываем json структуру dto.FileResponse в Redis с заданным TTL
			err = m.redisClient.Set(ctx, cacheKey, fileRespJson, m.cfg.Redis.MinioUrlTTL).Err()
			if err != nil {
				log.Printf("Failed to cache URL for object %s: %v", object.Key, err)
				errs = append(errs, err)
			}
			fileResponses = append(fileResponses, fileResp)
		}
	}

	return fileResponses, errs
}

func (m *minioClient) GetMany(ctx context.Context, objectIDs []dto.ObjectID, userID int) ([]dto.FileResponse, []error) {
	resCh := make(chan dto.FileResponse, len(objectIDs)) // Канал для URL-адресов объектов
	errCh := make(chan OperationError, len(objectIDs))   // Канал для ошибок

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, objectID := range objectIDs {
		wg.Add(1)
		go func(objectID dto.ObjectID) {
			defer wg.Done()

			// Проверка отмены перед выполнением работы
			if ctx.Err() != nil {
				return
			}

			url, err := m.GetOne(ctx, objectID, userID)
			if err != nil {

				// Проверяем, не был ли контекст уже отменён
				select {
				case <-ctx.Done():
					return
				case errCh <- OperationError{
					ObjectID: objectID.ObjID,
					Err:      err,
				}:
				}

				cancel() // Отмена всех горутин
				return
			}

			// Отправка URL, если контекст не отменён
			select {
			case <-ctx.Done():
				return
			case resCh <- url:
			}
		}(objectID)
	}

	// Закрытие каналов после завершения всех горутин.
	go func() {
		wg.Wait()
		close(resCh)
		close(errCh)
	}()

	// Сбор URL-адресов объектов и ошибок.
	var urls []dto.FileResponse
	var errs []error

	for fileResp := range resCh {
		urls = append(urls, fileResp)
	}
	for opErr := range errCh {
		errs = append(errs, opErr.Err)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return urls, nil
}

// DeleteOne удаляет один объект из бакета Minio по его идентификатору.
func (m *minioClient) DeleteOne(ctx context.Context, objectID dto.ObjectID, userID int) (int64, error) {
	const op = "location internal.minio.DeleteOne"

	cacheKey := GetRedisKey(objectID.ObjID, objectID.FileCategory)
	//deleting data in redis
	err := m.redisClient.Del(ctx, cacheKey).Err()
	if err != nil {
		log.Printf("Warning deletion did not work, %s,  details: %v", op, err)
	}

	objInfo, err := m.mc.StatObject(ctx, objectID.FileCategory, objectID.ObjID, minio.StatObjectOptions{})
	if err != nil {
		log.Printf("Error: %v, %s \n", err, op)
		return 0, fmt.Errorf("error getting information about the object %s: %w", objectID.ObjID, ErrFileNotFound)
	}
	size := objInfo.Size

	userIdStr, ok := objInfo.UserMetadata[fileMetaOwnerID]
	if !ok {
		log.Printf("Error: %v, %s \n", err, op)
		return 0, fmt.Errorf("the user_id metadata was not found for the object %s: %w", objectID.ObjID, ErrFileNotFound)
	}

	userIdInt, err := strconv.Atoi(userIdStr)
	if err != nil {
		return 0, fmt.Errorf("error converting string number: %s to int", userIdStr)
	}

	if userIdInt != userID {
		return 0, fmt.Errorf("you don't have access rights to other people's files: %w", ErrForbiddenResource)
	}

	// deleting data in minio if not in redis
	err = m.mc.RemoveObject(ctx, objectID.FileCategory, objectID.ObjID, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("error: %v, %s", err, op)
		return 0, OperationError{ObjectID: objectID.ObjID, Err: fmt.Errorf("couldn't delete selected file: %w", ErrFileNotFound)}
	}
	return size, nil
}

// DeleteMany удаляет сразу несколько объектов, возвращая размеры удалённых и ошибки
func (m *minioClient) DeleteMany(ctx context.Context, objectIDs []dto.ObjectID, userID int) ([]int64, []error) {
	type result struct {
		size int64
		err  error
	}
	resCh := make(chan result, len(objectIDs))
	var wg sync.WaitGroup

	for _, objectID := range objectIDs {
		wg.Add(1)
		go func(obj dto.ObjectID) {
			defer wg.Done()
			size, err := m.DeleteOne(ctx, obj, userID)
			resCh <- result{size: size, err: err}
		}(objectID)
	}

	// Собираем результаты
	go func() {
		wg.Wait()
		close(resCh)
	}()

	var (
		sizes []int64
		errs  []error
	)
	for r := range resCh {
		if r.err != nil {
			errs = append(errs, r.err)
		} else {
			sizes = append(sizes, r.size)
		}
	}
	return sizes, errs
}

func GetRedisKey(ObjID, fileType string) string {
	return fmt.Sprintf("ObjID:%v-file_type:%v", ObjID, fileType)
}

func GetFileExtension(fileName string) string {
	ext := filepath.Ext(fileName)
	return strings.TrimPrefix(ext, ".")
}

func GenerateFileID(userID int, fileExt string) string {
	fileExt = strings.TrimPrefix(fileExt, ".")
	return fmt.Sprintf("%d/%s.%s", userID, uuid.New().String(), fileExt)
}

func GenerateUserMetaData(userID int, origName string, createdAt time.Time) map[string]string {
	return map[string]string{
		fileMetaOwnerID:   fmt.Sprintf("%d", userID),
		fileMetaFileName:  origName,
		fileMetaCreatedAt: createdAt.Format(time.RFC3339),
	}
}

func GetCategory(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/") || contentType == "photo":
		return "photo"
	case strings.HasPrefix(contentType, "video/") || contentType == "video":
		return "video"
	case strings.HasPrefix(contentType, "text/") || contentType == "text":
		return "text"
	default:
		return "unknown"
	}
}
