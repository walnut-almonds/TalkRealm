package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFeedCommentLikeRepository_Create_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	// GORM (postgres) emits INSERT ... ON CONFLICT DO NOTHING RETURNING "id",
	// which is a Query, not an Exec — mirror feed_like_repository_test.go.
	mock.ExpectQuery(`INSERT INTO "feed_comment_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewFeedCommentLikeRepository(db)
	err := repo.Create(&model.FeedCommentLike{CommentID: 7, UserID: 5})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedCommentLikeRepository_CountByCommentID(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "feed_comment_likes"`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	repo := repository.NewFeedCommentLikeRepository(db)
	n, err := repo.CountByCommentID(7)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestFeedCommentLikeRepository_CountByCommentIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT comment_id, COUNT\(\*\) AS cnt FROM "feed_comment_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"comment_id", "cnt"}).
			AddRow(7, 3).AddRow(8, 1))

	repo := repository.NewFeedCommentLikeRepository(db)
	counts, err := repo.CountByCommentIDs([]uint{7, 8})
	require.NoError(t, err)
	assert.Equal(t, int64(3), counts[7])
	assert.Equal(t, int64(1), counts[8])
}

func TestFeedCommentLikeRepository_LikedCommentIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT "comment_id" FROM "feed_comment_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"comment_id"}).AddRow(7))

	repo := repository.NewFeedCommentLikeRepository(db)
	liked, err := repo.LikedCommentIDs(5, []uint{7, 8})
	require.NoError(t, err)
	assert.True(t, liked[7])
	assert.False(t, liked[8])
}
