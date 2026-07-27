package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestFeedCommentRepository_ByPostCursor_Chronological(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	// fetched id DESC, then reversed to chronological by the repo.
	mock.ExpectQuery(`SELECT \* FROM "feed_comments" WHERE post_id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "content"}).
			AddRow(9, 7, "b").AddRow(8, 7, "a"))

	repo := repository.NewFeedCommentRepository(db)
	cs, err := repo.ByPostCursor(7, 0, 50)
	require.NoError(t, err)
	require.Len(t, cs, 2)
	assert.Equal(t, uint(8), cs[0].ID) // reversed to chronological
	assert.Equal(t, uint(9), cs[1].ID)
}

func TestFeedCommentRepository_CountByPostIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT post_id, COUNT\(\*\) AS cnt FROM "feed_comments"`).
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "cnt"}).
			AddRow(7, 2).AddRow(8, 5))

	repo := repository.NewFeedCommentRepository(db)
	counts, err := repo.CountByPostIDs([]uint{7, 8})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[7])
	assert.Equal(t, int64(5), counts[8])
}
