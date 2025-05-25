package quota_handler

import "github.com/1abobik1/SecureComm/internal/service/quota_service"

type QuotaHandler struct {
	quotaService *quota_service.QuotaService
}

func NewQuotaHandler(quotaService *quota_service.QuotaService) *QuotaHandler {
	return &QuotaHandler{quotaService: quotaService}
}
