package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/1abobik1/AuthService/internal/domain/models"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/lib/pq"
)

type PostgesStorage struct {
	db *sql.DB
}

func NewPostgresStorageProd(storagePath string) (*PostgesStorage, error) {
	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, err
	}

	return &PostgesStorage{db: db}, nil
}

func NewPostgresForTesting(db *sql.DB) *PostgesStorage {
	return &PostgesStorage{db: db}
}

func (p *PostgesStorage) SaveUser(ctx context.Context, email string, password []byte) (int, error) {
	const op = "storage.postgresql.SaveUser"

	query := "INSERT INTO auth_users(email, password) VALUES ($1, $2) RETURNING id"

	var id int
	err := p.db.QueryRowContext(ctx, query, email, password).Scan(&id)
	if err != nil {
		return 0, wrapPostgresErrors(err, op)
	}

	return id, nil
}

func (p *PostgesStorage) SaveUserKey(ctx context.Context, userID int, userKey string) error {
	_, err := p.db.ExecContext(ctx, `
        INSERT INTO user_keys (user_id, user_key)
        VALUES ($1, $2)
    `, userID, userKey)

	if err != nil {
		// ошибка уникальности
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("user key already exists for user_id %d", userID)
		}
		return fmt.Errorf("failed to insert user key: %w", err)
	}

	return nil
}

func (p *PostgesStorage) GetUserKey(ctx context.Context, userID int) (string, error) {
	var userKey string

	err := p.db.QueryRowContext(ctx, `
        SELECT user_key
        FROM user_keys
        WHERE user_id = $1
    `, userID).Scan(&userKey)

	if err == sql.ErrNoRows {
		return "", storage.ErrUserNotFound
	} else if err != nil {
		return "", fmt.Errorf("failed to get user key: %w", err)
	}

	return userKey, nil
}

// adding a new token, if there is already a token with such a platform, then simply update it in the database.
func (p *PostgesStorage) UpsertRefreshToken(ctx context.Context, refreshToken string, userID int, platform string) error {
	const op = "storage.postgresql.UpsertRefreshToken"

	query := `
        INSERT INTO refresh_token (token, user_id, platform)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, platform) DO UPDATE
        SET token = EXCLUDED.token;
    `

	_, err := p.db.ExecContext(ctx, query, refreshToken, userID, platform)
	if err != nil {
		log.Printf("Error saving in postgresql: %s, %v \n", op, err)
		return err
	}

	return nil
}

func (p *PostgesStorage) FindUser(ctx context.Context, email string) (models.UserModel, error) {
	const op = "storage.postgresql.FindUser"

	var userModel models.UserModel
	query := "SELECT id, email, password, is_activated FROM auth_users WHERE email = $1"
	err := p.db.QueryRowContext(ctx, query, email).Scan(&userModel.ID, &userModel.Email, &userModel.Password, &userModel.IsActivated)
	if err != nil {
		return models.UserModel{}, wrapPostgresErrors(err, op)
	}

	return userModel, nil
}

func (p *PostgesStorage) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	const op = "storage.postgresql.DeleteRefreshToken"

	query := "DELETE FROM refresh_token WHERE token = $1"
	res, err := p.db.ExecContext(ctx, query, refreshToken)
	if err != nil {
		log.Printf("Error deleting refresh_token where token = %s: %v location: %s", refreshToken, err, op)
		return wrapPostgresErrors(err, op)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error getting affected rows: %v location: %s", err, op)
		return wrapPostgresErrors(err, op)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("location %s: %w", op, storage.ErrTokenNotFound)
	}

	return nil
}

func (p *PostgesStorage) CheckRefreshToken(refreshToken string) (int, error) {
	const op = "storage.postgresql.CheckRefreshToken"

	query := "SELECT user_id FROM refresh_token WHERE token = $1"

	var userID int
	err := p.db.QueryRow(query, refreshToken).Scan(&userID)
	if err != nil {
		return 0, wrapPostgresErrors(err, op)
	}

	return userID, nil
}

func (p *PostgesStorage) UpdateRefreshToken(oldRefreshToken, newRefreshToken string) error {
	const op = "storage.postgresql.UpdateRefreshToken"

	// SQL-запрос для обновления
	query := `
		UPDATE refresh_token
		SET token = $1
		WHERE token = $2
		RETURNING id;
	`

	var id int
	err := p.db.QueryRow(query, newRefreshToken, oldRefreshToken).Scan(&id)
	if err != nil {
		return wrapPostgresErrors(err, op)
	}

	return nil
}
