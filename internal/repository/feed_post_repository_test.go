package repository_test

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFeedPostRepository_DeleteCascade(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "feed_comment_likes" WHERE comment_id IN \(SELECT id FROM feed_comments WHERE post_id = \$1\)`).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 4))
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

func TestFeedPostRepository_RecentCandidates(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "feed_posts" WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "content"}).
			AddRow(9, 2, "a").AddRow(8, 3, "b"))
	// GORM fires Author + Attachments.File preload sub-queries after the main SELECT;
	// stub them with empty rows (same relaxation as TimelineCursor above) so the
	// load-bearing assertion stays newest-first ordering.
	mock.ExpectQuery(`SELECT \* FROM "feed_post_attachments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewFeedPostRepository(db)
	posts, err := repo.RecentCandidates(5, time.Unix(0, 0), 500)
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, uint(9), posts[0].ID) // newest-first
}

func TestFeedPostRepository_AuthorAffinity(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	// likes grouped by post author.
	// Relaxed from plan's `FROM "feed_post_likes"`: GORM's Table("... AS fpl")
	// emits the table name UNQUOTED (`FROM feed_post_likes AS fpl`), so the
	// quoted regex never matched. Grouping/merge assertions below are unchanged.
	mock.ExpectQuery(`FROM feed_post_likes`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "cnt"}).AddRow(2, 3))
	// comments grouped by post author
	mock.ExpectQuery(`FROM feed_comments`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "cnt"}).AddRow(2, 1).AddRow(3, 4))

	repo := repository.NewFeedPostRepository(db)
	aff, err := repo.AuthorAffinity(5)
	require.NoError(t, err)
	assert.Equal(t, int64(4), aff[2]) // 3 likes + 1 comment
	assert.Equal(t, int64(4), aff[3]) // 0 likes + 4 comments
}
