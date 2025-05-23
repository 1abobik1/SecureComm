package postgresql_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/1abobik1/AuthService/internal/domain/models"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/storage/postgresql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestFindUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	s := postgresql.NewPostgresForTesting(db)
	ctx := context.Background()

	cols := []string{"id", "email", "password", "is_activated"}
	rows := sqlmock.NewRows(cols).AddRow(1, "test_1@mail.ru", "test_pswd_1", false)
	mock.ExpectQuery("SELECT id, email, password, is_activated FROM auth_users WHERE email = \\$1").
		WithArgs("test_1@mail.ru").
		WillReturnRows(rows)

	user, err := s.FindUser(ctx, "test_1@mail.ru")
	assert.NoError(t, err)

	expectedUser := models.UserModel{
		ID:          1,
		Email:       "test_1@mail.ru",
		Password:    []byte("test_pswd_1"),
		IsActivated: false,
	}
	assert.Equal(t, expectedUser, user)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	s := postgresql.NewPostgresForTesting(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email, password, is_activated FROM auth_users WHERE email = \\$1").
		WithArgs("Unknown@mail.ru").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "is_activated"}))

	user, err := s.FindUser(ctx, "Unknown@mail.ru")

	assert.Error(t, err)
	assert.Equal(t, models.UserModel{}, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindUser_ErrorDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	s := postgresql.NewPostgresForTesting(db)
	ctx := context.Background()

	dbErr := errors.New("db error")
	mock.ExpectQuery("SELECT id, email, password, is_activated FROM auth_users WHERE email = \\$1").
		WithArgs("Unknown@mail.ru").
		WillReturnError(dbErr)

	user, err := s.FindUser(ctx, "Unknown@mail.ru")

	assert.Error(t, err)

	assert.Equal(t, models.UserModel{}, user)

	assert.True(t, errors.Is(err, dbErr))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols).AddRow(1)

	query := "INSERT INTO auth_users(email, password) VALUES ($1, $2) RETURNING id"
	query = regexp.QuoteMeta(query)

	mock.ExpectQuery(query).
		WithArgs("test_1@mail.ru", []byte("pswd_1")).
		WillReturnRows(rows)

	stor := postgresql.NewPostgresForTesting(db)

	id, err := stor.SaveUser(context.Background(), "test_1@mail.ru", []byte("pswd_1"))
	assert.NoError(t, err)

	assert.Equal(t, 1, id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveUser_ErrorDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := "INSERT INTO auth_users(email, password) VALUES ($1, $2) RETURNING id"
	query = regexp.QuoteMeta(query)

	dbErr := errors.New("db error")
	mock.ExpectQuery(query).
		WithArgs("test_1@mail.ru", []byte("pswd_1")).
		WillReturnError(dbErr)

	stor := postgresql.NewPostgresForTesting(db)

	id, err := stor.SaveUser(context.Background(), "test_1@mail.ru", []byte("pswd_1"))

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Equal(t, 0, id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertRefreshToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := `
        INSERT INTO refresh_token (token, user_id, platform)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, platform) DO UPDATE
        SET token = EXCLUDED.token;
    `
	query = regexp.QuoteMeta(query)

	mock.ExpectExec(query).WithArgs("refresh_token_1", 1, "web").
		WillReturnResult(sqlmock.NewResult(1, 1))

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.UpsertRefreshToken(context.Background(), "refresh_token_1", 1, "web")

	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertRefreshToken_UpdateExisting(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := `
        INSERT INTO refresh_token (token, user_id, platform)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, platform) DO UPDATE
        SET token = EXCLUDED.token;
    `
	query = regexp.QuoteMeta(query)

	mock.ExpectExec(query).WithArgs("refresh_token_1", 1, "web").
		WillReturnResult(sqlmock.NewResult(0, 1))

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.UpsertRefreshToken(context.Background(), "refresh_token_1", 1, "web")

	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertRefreshToken_ErrorDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := `
        INSERT INTO refresh_token (token, user_id, platform)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, platform) DO UPDATE
        SET token = EXCLUDED.token;
    `
	query = regexp.QuoteMeta(query)

	dbErr := errors.New("db error")
	mock.ExpectExec(query).WithArgs("refresh_token_1", 1, "web").
		WillReturnError(dbErr)

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.UpsertRefreshToken(context.Background(), "refresh_token_1", 1, "web")

	assert.Error(t, err)
	assert.ErrorIs(t, err, dbErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteRefreshToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := "DELETE FROM refresh_token WHERE token = \\$1"
	mock.ExpectExec(query).WithArgs("refresh_token_1").WillReturnResult(sqlmock.NewResult(0, 1))

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.DeleteRefreshToken(context.Background(), "refresh_token_1")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteRefreshToken_TokenNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := "DELETE FROM refresh_token WHERE token = \\$1"
	mock.ExpectExec(query).WithArgs("refresh_token_1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.DeleteRefreshToken(context.Background(), "refresh_token_1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrTokenNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}


func TestCheckRefreshToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := "SELECT user_id FROM refresh_token WHERE token = \\$1"

	cols := []string{"user_id"}
	rows := sqlmock.NewRows(cols).AddRow(1)
	mock.ExpectQuery(query).WithArgs("token_1").WillReturnRows(rows)

	stor := postgresql.NewPostgresForTesting(db)

	user_id, err := stor.CheckRefreshToken("token_1")

	assert.NoError(t, err)
	assert.Equal(t, 1, user_id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckRefreshToken_TokenNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := "SELECT user_id FROM refresh_token WHERE token = \\$1"


	mock.ExpectQuery(query).WithArgs("token_1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

	stor := postgresql.NewPostgresForTesting(db)

	user_id, err := stor.CheckRefreshToken("token_1")

	assert.Error(t, err)
	assert.Equal(t, 0, user_id)
	assert.ErrorIs(t, err, storage.ErrTokenNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateRefreshToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	query := `
		UPDATE refresh_token
		SET token = $1
		WHERE token = $2
		RETURNING id;
	`
	query = regexp.QuoteMeta(query)

	cols := []string{"id"}
	rows := sqlmock.NewRows(cols).AddRow(1)
	mock.ExpectQuery(query).WithArgs("new_token_1", "old_token_1").WillReturnRows(rows)

	stor := postgresql.NewPostgresForTesting(db)
	err = stor.UpdateRefreshToken("old_token_1", "new_token_1")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
