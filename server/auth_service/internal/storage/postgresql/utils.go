package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/lib/pq"
)

func wrapPostgresErrors(err error, op string) error {
	if err == nil {
		return nil
	}
	// fmt.Errorf("location %s: %w", op, storage.ErrTokenNotFound)
	// Проверяем, является ли ошибка PostgreSQL-ошибкой
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505": // Уникальное ограничение
			return fmt.Errorf("location %s, error: %w", op, storage.ErrUserExists)
		default:
			return fmt.Errorf("location %s, error %s: %w", op, pqErr.Code, err)
		}
	}

	// Проверяем на sql.ErrNoRows
	if errors.Is(err, sql.ErrNoRows) {
		
		if strings.Contains(op, "User") || strings.Contains(op, "user") {
			return fmt.Errorf("location: %s, error: %w", op, storage.ErrUserNotFound)

		} else if strings.Contains(op, "Token") || strings.Contains(op, "token") {
			return fmt.Errorf("location: %s, error: %w", op, storage.ErrTokenNotFound)
		}
	}

	// Общая ошибка, если никакие условия не выполнились
	return fmt.Errorf("location %s: %w", op, err)
}
