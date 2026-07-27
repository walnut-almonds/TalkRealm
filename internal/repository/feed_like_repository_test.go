package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFeedPostLikeRepository_Create_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	// GORM (postgres) emits INSERT ... ON CONFLICT DO NOTHING RETURNING "id",
	// which is a Query, not an Exec — mirror message_like_repository_test.go.
	mock.ExpectQuery(`INSERT INTO "feed_post_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewFeedPostLikeRepository(db)
	err := repo.Create(&model.FeedPostLike{PostID: 7, UserID: 5})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedPostLikeRepository_CountByPostID(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "feed_post_likes"`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	repo := repository.NewFeedPostLikeRepository(db)
	n, err := repo.CountByPostID(7)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestFeedPostLikeRepository_CountByPostIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT post_id, COUNT\(\*\) AS cnt FROM "feed_post_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "cnt"}).
			AddRow(7, 3).AddRow(8, 1))

	repo := repository.NewFeedPostLikeRepository(db)
	counts, err := repo.CountByPostIDs([]uint{7, 8})
	require.NoError(t, err)
	assert.Equal(t, int64(3), counts[7])
	assert.Equal(t, int64(1), counts[8])
}

func TestFeedPostLikeRepository_LikedPostIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT "post_id" FROM "feed_post_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"post_id"}).AddRow(7))

	repo := repository.NewFeedPostLikeRepository(db)
	liked, err := repo.LikedPostIDs(5, []uint{7, 8})
	require.NoError(t, err)
	assert.True(t, liked[7])
	assert.False(t, liked[8])
}
