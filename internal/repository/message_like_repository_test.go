package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestMessageLikeRepository_Create_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	// GORM (postgres) emits INSERT ... ON CONFLICT DO NOTHING RETURNING "id",
	// which is a Query, not an Exec — mirror the other insert tests in this pkg.
	mock.ExpectQuery(`INSERT INTO "message_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewMessageLikeRepository(db)
	err := repo.Create(&model.MessageLike{MessageID: 7, UserID: 5})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageLikeRepository_CountByMessageID(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "message_likes"`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	repo := repository.NewMessageLikeRepository(db)
	n, err := repo.CountByMessageID(7)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}
