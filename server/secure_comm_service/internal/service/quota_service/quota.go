package quota_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/1abobik1/SecureComm/internal/domain"
	_ "github.com/lib/pq"
)

const (
	FreeLimitBytes = int64(10 * 1024 * 1024 * 1024)       // 10 GiB
	ProLimitBytes  = int64(1 * 1024 * 1024 * 1024 * 1024) // 1 TiB
)

var ErrQuotaExceeded = errors.New("quota exceeded")
var ErrNoActivePlan = errors.New("no active plan for user")
var ErrUserNotFound = errors.New("there is no such user")

type QuotaService struct {
	db *sql.DB
}

func NewQuotaService(storagePath string) (*QuotaService, error) {
	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, err
	}
	return &QuotaService{db: db}, nil
}

func (s *QuotaService) InitializeFreePlan(ctx context.Context, userID int) error {
	// вставляем только один раз: если уже есть — IGNORE
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO user_plans (user_id, plan_id, started_at, expires_at, current_used)
        SELECT
            $1,                  -- user_id
            p.id,                -- plan_id
            NOW(),               -- started_at
            CASE
              WHEN p.period_days > 0
                THEN NOW() + (p.period_days || ' days')::interval
              ELSE 'infinity'::timestamptz
            END,                 -- expires_at: либо сейчас+period, либо бесконечность
            0                    -- current_used
        FROM plans p
        WHERE p.name = 'free'
          AND NOT EXISTS (
              SELECT 1 FROM user_plans WHERE user_id = $1
          )
    `, userID)
	if err != nil {
		return fmt.Errorf("init free plan: %w", err)
	}
	return nil
}

// CheckQuota проверяет, что used + newSize не превысит storage_limit
func (s *QuotaService) CheckQuota(ctx context.Context, userID int, newSize int64) error {
	var used, limit int64

	// попробуем получить активную подписку
	err := s.db.QueryRowContext(ctx, `
        SELECT up.current_used, p.storage_limit
        FROM user_plans up
        JOIN plans p ON up.plan_id = p.id
        WHERE up.user_id = $1
          AND up.expires_at > NOW()
        FOR SHARE
    `, userID).Scan(&used, &limit)

	if err == sql.ErrNoRows {
		// если записи нет — это баг, т.к. при регистрации создаём free-план
		return ErrNoActivePlan
	}
	if err != nil {
		return fmt.Errorf("quota check: %w", err)
	}

	if used+newSize > limit {
		return ErrQuotaExceeded
	}
	return nil
}

// AddUsage прибавляет newSize к current_used в уже существующей активной записи
func (s *QuotaService) AddUsage(ctx context.Context, userID int, newSize int64) error {
	res, err := s.db.ExecContext(ctx, `
        UPDATE user_plans
        SET current_used = current_used + $1
        WHERE user_id = $2
          AND expires_at > NOW()
    `, newSize, userID)
	if err != nil {
		return fmt.Errorf("add usage: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("add usage rows: %w", err)
	}
	if rows == 0 {
		return ErrNoActivePlan
	}
	return nil
}

// RemoveUsage вычитает newSize из current_used в уже существующей активной записи
func (s *QuotaService) RemoveUsage(ctx context.Context, userID int, newSize int64) error {
	// Используем GREATEST, чтобы current_used не стал отрицательным
	res, err := s.db.ExecContext(ctx, `
        UPDATE user_plans
        SET current_used = GREATEST(current_used - $1, 0)
        WHERE user_id = $2
          AND expires_at > NOW()
    `, newSize, userID)
	if err != nil {
		return fmt.Errorf("remove usage: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("remove usage rows: %w", err)
	}
	if rows == 0 {
		return ErrNoActivePlan
	}
	return nil
}

func (s *QuotaService) GetUserUsage(ctx context.Context, userID int) (domain.UserUsage, error) {
	const query = `
    SELECT
      up.current_used,
      p.storage_limit,
      p.name AS plan_name
    FROM user_plans up
    JOIN plans p ON p.id = up.plan_id
    WHERE up.user_id = $1
      AND (p.period_days = 0 OR up.expires_at > NOW())
	`

	var userUsage domain.UserUsage

	err := s.db.QueryRowContext(ctx, query, userID).Scan(&userUsage.CurrentUsed, &userUsage.StorageLimit, &userUsage.PlanName)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.UserUsage{}, ErrUserNotFound
		}

		return domain.UserUsage{}, err
	}

	return userUsage, nil
}
