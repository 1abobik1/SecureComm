package quota_handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/service/quota_service"

	"github.com/gin-gonic/gin"
)

const (
	bytesInGB = 1024 * 1024 * 1024
	bytesInMB = 1024 * 1024
	bytesInKB = 1024
)

// InitUserPlan создаёт для userID бесплатный тариф 10 GiB
// @Summary      Инициализация бесплатного плана (10 GiB) пользователя
// @Description  Создаёт для пользователя с переданным ID бесплатный план хранения объёмом до 10 GiB. Если план уже существует, операция игнорируется.
// @Tags         Quota
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer {token}"
// @Param        id             path    int     true  "ID пользователя"
// @Success      201            {string}  string               "Created — план успешно инициализирован"
// @Failure      400            {object}  map[string]string    "Некорректный ID пользователя"
// @Failure      401            {object}  map[string]string    "Ошибка авторизации или токен не предоставлен"
// @Failure      500            {object}  map[string]string    "Внутренняя ошибка сервера"
// @Security     BearerAuth
// @Router       /user/{id}/plan/init [post]
func (h *QuotaHandler) InitUserPlan(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.quotaService.InitializeFreePlan(c, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// GetUserUsage выдает текущее кол-во используемой памяти по id пользователя
// @Summary      Получение текущего использования диска пользователя
// @Description  Возвращает, сколько гигабайт, мегабайт и килобайт хранится у пользователя, а также лимит (10 GiB для бесплатного плана) и имя плана.
// @Tags         Quota
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer {token}"
// @Param        id             path    int     true  "ID пользователя"
// @Success      200            {object}  dto.UserUsage  "Информация об использовании дискового пространства"
// @Failure      400            {object}  map[string]string    "Некорректный ID пользователя"
// @Failure      401            {object}  map[string]string    "Ошибка авторизации или токен не предоставлен"
// @Failure      404            {object}  map[string]string    "Пользователь не найден"
// @Failure      500            {object}  map[string]string    "Внутренняя ошибка сервера"
// @Security     BearerAuth
// @Router       /user/{id}/usage [get]
func (h *QuotaHandler) GetUserUsage(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	domainUserUsage, err := h.quotaService.GetUserUsage(c, userID)
	if err != nil {
		if errors.Is(err, quota_service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": quota_service.ErrUserNotFound.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	used := domainUserUsage.CurrentUsed

	// целые гигабайты
	gbCount := used / bytesInGB
	remAfterGB := used % bytesInGB

	// целые мегабайты из остатка после гигабайтов
	mbCount := remAfterGB / bytesInMB
	remAfterMB := remAfterGB % bytesInMB

	// целые килобайты из остатка после мегабайт
	kbCount := remAfterMB / bytesInKB

	resp := dto.UserUsage{
		CurrentUsedGB:  int(gbCount),
		CurrentUsedMB:  int(mbCount),
		CurrentUsedKB:  int(kbCount),
		StorageLimitGB: int(domainUserUsage.StorageLimit / bytesInGB),
		PlanName:       domainUserUsage.PlanName,
	}

	c.JSON(http.StatusOK, resp)
}
