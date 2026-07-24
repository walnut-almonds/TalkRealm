package repository_test

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

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
