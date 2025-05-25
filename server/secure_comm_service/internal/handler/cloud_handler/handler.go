package cloud_handler

import (
	"github.com/1abobik1/SecureComm/internal/service/cloud_service"
	"github.com/1abobik1/SecureComm/internal/service/quota_service"
)

type MinioHandler struct {
	minioService cloud_service.Client
	quotaService *quota_service.QuotaService
}

func NewMinioHandler(minioService cloud_service.Client, quotaService *quota_service.QuotaService) *MinioHandler {
	return &MinioHandler{
		minioService: minioService,
		quotaService: quotaService,
	}
}
