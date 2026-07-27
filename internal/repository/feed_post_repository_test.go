package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFeedPostRepository_DeleteCascade(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "feed_comments" WHERE post_id = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM "feed_post_likes" WHERE post_id = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`DELETE FROM "feed_post_attachments" WHERE post_id = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "feed_posts" WHERE "feed_posts"."id" = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := repository.NewFeedPostRepository(db)
	require.NoError(t, repo.DeleteCascade(7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedPostRepository_TimelineCursor(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "feed_posts" WHERE author_id IN`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "content"}).
			AddRow(9, 2, "hi").AddRow(8, 3, "yo"))
	// GORM emits preload sub-queries (Attachments.File, Author) after the main SELECT.
	// Relaxed per plan Step 5: stub them with empty rows so only the main SELECT +
	// newest-first ordering are the load-bearing assertions.
	mock.ExpectQuery(`SELECT \* FROM "feed_post_attachments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewFeedPostRepository(db)
	posts, err := repo.TimelineCursor([]uint{2, 3}, 0, 20)
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, uint(9), posts[0].ID) // newest-first, NOT reversed
}
