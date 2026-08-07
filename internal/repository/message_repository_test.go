package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

func TestMessageRepository_GetMessagesAround(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	// older-or-equal half (id <= around, DESC)
	mock.ExpectQuery(`SELECT \* FROM "messages" WHERE channel_id = \$1 AND id <= \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel_id"}).AddRow(10, 1).AddRow(9, 1))
	// Attachments preload for the older half (rows carry no user_id, so the User
	// preload is skipped; only Attachments keys off the present id column).
	mock.ExpectQuery(`SELECT \* FROM "message_attachments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	// newer half (id > around, ASC)
	mock.ExpectQuery(`SELECT \* FROM "messages" WHERE channel_id = \$1 AND id > \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel_id"}).AddRow(11, 1))
	// Attachments preload for the newer half
	mock.ExpectQuery(`SELECT \* FROM "message_attachments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewMessageRepository(db)
	msgs, err := repo.GetMessagesAround(1, 10, 6)
	require.NoError(t, err)
	// chronological ascending, target 10 present, window around it
	require.Len(t, msgs, 3)
	assert.Equal(t, uint(9), msgs[0].ID)
	assert.Equal(t, uint(10), msgs[1].ID)
	assert.Equal(t, uint(11), msgs[2].ID)
}

func TestMessageRepository_DeletePostCascade(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	// collect comment ids
	mock.ExpectQuery(`SELECT "id" FROM "messages" WHERE parent_id = \$1`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(8).AddRow(9))
	// delete likes for post + comments
	mock.ExpectExec(`DELETE FROM "message_likes" WHERE message_id IN`).
		WillReturnResult(sqlmock.NewResult(0, 3))
	// delete comments
	mock.ExpectExec(`DELETE FROM "messages" WHERE parent_id = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 2))
	// delete post
	mock.ExpectExec(`DELETE FROM "messages" WHERE "messages"."id" = \$1`).
		WithArgs(7).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := repository.NewMessageRepository(db)
	require.NoError(t, repo.DeletePostCascade(7))
	require.NoError(t, mock.ExpectationsWereMet())
}
