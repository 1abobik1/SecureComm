package cloud_handler

import "github.com/1abobik1/SecureComm/internal/service/cloud_service"

type minioHandler struct {
	minioService cloud_service.Client
}

func NewMinioHandler(minioService cloud_service.Client) *minioHandler {
	return &minioHandler{
		minioService: minioService,
	}
}
