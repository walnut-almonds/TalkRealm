package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFollowRepository_Follow_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "follows"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewFollowRepository(db)
	require.NoError(t, repo.Follow(1, 2))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_FolloweeIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT "followee_id" FROM "follows" WHERE follower_id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"followee_id"}).AddRow(2).AddRow(3))

	repo := repository.NewFollowRepository(db)
	ids, err := repo.FolloweeIDs(1)
	require.NoError(t, err)
	assert.Equal(t, []uint{2, 3}, ids)
}
