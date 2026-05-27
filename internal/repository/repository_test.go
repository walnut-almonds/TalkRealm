package repository_test

import (
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestDB creates a GORM DB backed by go-sqlmock.
func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	})
	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	return gdb, mock, sqlDB
}

// ptrU returns a pointer to the given uint value.
func ptrU(v uint) *uint { return &v }

// ---------------------------------------------------------------------------
// UserRepository tests
// ---------------------------------------------------------------------------

func TestUserRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewUserRepository(db)
	user := &model.User{Username: "alice", Email: "alice@test.com", Password: "hashed"}
	err := repo.Create(user)
	require.NoError(t, err)
}

func TestUserRepository_Create_DBError(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	repo := repository.NewUserRepository(db)
	user := &model.User{Username: "alice", Email: "alice@test.com", Password: "hashed"}
	err := repo.Create(user)
	assert.Error(t, err)
}

func TestUserRepository_GetByID_Found(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "nickname", "avatar", "status", "created_at", "updated_at"}).
		AddRow(1, "alice", "alice@test.com", "hashed", "Alice", "", "offline", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(rows)

	repo := repository.NewUserRepository(db)
	user, err := repo.GetByID(1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "alice", user.Username)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}))

	repo := repository.NewUserRepository(db)
	_, err := repo.GetByID(999)
	assert.Error(t, err)
}

func TestUserRepository_GetByEmail_Found(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "nickname", "avatar", "status", "created_at", "updated_at"}).
		AddRow(2, "bob", "bob@test.com", "hashed", "Bob", "", "online", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(rows)

	repo := repository.NewUserRepository(db)
	user, err := repo.GetByEmail("bob@test.com")
	require.NoError(t, err)
	assert.Equal(t, "bob@test.com", user.Email)
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}))

	repo := repository.NewUserRepository(db)
	_, err := repo.GetByEmail("missing@test.com")
	assert.Error(t, err)
}

func TestUserRepository_GetByUsername_Found(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "nickname", "avatar", "status", "created_at", "updated_at"}).
		AddRow(3, "carol", "carol@test.com", "hashed", "Carol", "", "offline", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(rows)

	repo := repository.NewUserRepository(db)
	user, err := repo.GetByUsername("carol")
	require.NoError(t, err)
	assert.Equal(t, "carol", user.Username)
}

func TestUserRepository_GetByUsername_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}))

	repo := repository.NewUserRepository(db)
	_, err := repo.GetByUsername("nobody")
	assert.Error(t, err)
}

func TestUserRepository_Update_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewUserRepository(db)
	user := &model.User{ID: 1, Username: "alice", Email: "alice@test.com", Password: "hashed"}
	err := repo.Update(user)
	require.NoError(t, err)
}

func TestUserRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewUserRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

func TestUserRepository_List_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "nickname", "avatar", "status", "created_at", "updated_at"}).
		AddRow(1, "alice", "alice@test.com", "hashed", "Alice", "", "offline", time.Now(), time.Now()).
		AddRow(2, "bob", "bob@test.com", "hashed", "Bob", "", "online", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(rows)

	repo := repository.NewUserRepository(db)
	users, err := repo.List(0, 10)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUserRepository_UpdateStatus_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewUserRepository(db)
	err := repo.UpdateStatus(1, "online")
	require.NoError(t, err)
}

func TestUserRepository_UpdateStatus_DBError(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	repo := repository.NewUserRepository(db)
	err := repo.UpdateStatus(1, "online")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GuildRepository tests
// ---------------------------------------------------------------------------

func TestGuildRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "guilds"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewGuildRepository(db)
	guild := &model.Guild{Name: "Test Guild", OwnerID: 1}
	err := repo.Create(guild)
	require.NoError(t, err)
}

func TestGuildRepository_GetByID_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	// Preload("Owner") triggers a query for users after guilds query
	mock.ExpectQuery(`SELECT \* FROM "guilds"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner_id"}))

	repo := repository.NewGuildRepository(db)
	_, err := repo.GetByID(999)
	assert.Error(t, err)
}

func TestGuildRepository_Update_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "guilds"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildRepository(db)
	guild := &model.Guild{ID: 1, Name: "Updated Guild", OwnerID: 1}
	err := repo.Update(guild)
	require.NoError(t, err)
}

func TestGuildRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "guilds"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

func TestGuildRepository_GetByOwnerID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "icon", "owner_id", "created_at", "updated_at"}).
		AddRow(1, "My Guild", "", "", 1, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "guilds"`).WillReturnRows(rows)

	repo := repository.NewGuildRepository(db)
	guilds, err := repo.GetByOwnerID(1)
	require.NoError(t, err)
	assert.Len(t, guilds, 1)
}

func TestGuildRepository_GetMemberGuilds_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "name", "description", "icon", "owner_id", "created_at", "updated_at"}).
		AddRow(1, "Member Guild", "", "", 2, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	repo := repository.NewGuildRepository(db)
	guilds, err := repo.GetMemberGuilds(1, 0, 10)
	require.NoError(t, err)
	assert.Len(t, guilds, 1)
}

func TestGuildRepository_List_Empty(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "guilds"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner_id"}))

	repo := repository.NewGuildRepository(db)
	guilds, err := repo.List(0, 10)
	require.NoError(t, err)
	assert.Empty(t, guilds)
}

// ---------------------------------------------------------------------------
// ChannelRepository tests
// ---------------------------------------------------------------------------

func TestChannelRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "channels"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewChannelRepository(db)
	ch := &model.Channel{GuildID: ptrU(1), Name: "general", Type: "text"}
	err := repo.Create(ch)
	require.NoError(t, err)
}

func TestChannelRepository_GetByID_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "channels"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	repo := repository.NewChannelRepository(db)
	_, err := repo.GetByID(999)
	assert.Error(t, err)
}

func TestChannelRepository_Update_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "channels"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewChannelRepository(db)
	ch := &model.Channel{ID: 1, GuildID: ptrU(1), Name: "updated", Type: "text"}
	err := repo.Update(ch)
	require.NoError(t, err)
}

func TestChannelRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "channels"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewChannelRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

func TestChannelRepository_GetByGuildID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "guild_id", "name", "type", "topic", "position", "created_at", "updated_at"}).
		AddRow(1, 1, "general", "text", "", 0, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "channels"`).WillReturnRows(rows)

	repo := repository.NewChannelRepository(db)
	channels, err := repo.GetByGuildID(1)
	require.NoError(t, err)
	assert.Len(t, channels, 1)
}

func TestChannelRepository_GetByType_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "guild_id", "name", "type", "topic", "position", "created_at", "updated_at"}).
		AddRow(2, 1, "voice-1", "voice", "", 1, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "channels"`).WillReturnRows(rows)

	repo := repository.NewChannelRepository(db)
	channels, err := repo.GetByType(1, "voice")
	require.NoError(t, err)
	assert.Len(t, channels, 1)
}

// ---------------------------------------------------------------------------
// GuildMemberRepository tests
// ---------------------------------------------------------------------------

func TestGuildMemberRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "guild_members"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewGuildMemberRepository(db)
	m := &model.GuildMember{GuildID: 1, UserID: 1, Role: "member"}
	err := repo.Create(m)
	require.NoError(t, err)
}

func TestGuildMemberRepository_GetByID_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "guild_members"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewGuildMemberRepository(db)
	_, err := repo.GetByID(999)
	assert.Error(t, err)
}

func TestGuildMemberRepository_Update_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "guild_members"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildMemberRepository(db)
	m := &model.GuildMember{ID: 1, GuildID: 1, UserID: 1, Role: "admin"}
	err := repo.Update(m)
	require.NoError(t, err)
}

func TestGuildMemberRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "guild_members"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildMemberRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

func TestGuildMemberRepository_GetByGuildID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "guild_id", "user_id", "role", "joined_at", "created_at", "updated_at"}).
		AddRow(1, 1, 1, "owner", time.Now(), time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "guild_members"`).WillReturnRows(rows)
	// Preload("User") — second query
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewGuildMemberRepository(db)
	members, err := repo.GetByGuildID(1)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestGuildMemberRepository_GetMember_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "guild_members"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "guild_id", "user_id"}))

	repo := repository.NewGuildMemberRepository(db)
	_, err := repo.GetMember(1, 999)
	assert.Error(t, err)
}

func TestGuildMemberRepository_IsMember_True(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "guild_members"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	repo := repository.NewGuildMemberRepository(db)
	ok, err := repo.IsMember(1, 1)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGuildMemberRepository_IsMember_False(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "guild_members"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	repo := repository.NewGuildMemberRepository(db)
	ok, err := repo.IsMember(1, 99)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestGuildMemberRepository_GetUserGuildIDs_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"guild_id"}).AddRow(1).AddRow(2))

	repo := repository.NewGuildMemberRepository(db)
	ids, err := repo.GetUserGuildIDs(1)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
}

// ---------------------------------------------------------------------------
// RefreshTokenRepository tests
// ---------------------------------------------------------------------------

func TestRefreshTokenRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "refresh_tokens"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewRefreshTokenRepository(db)
	rt := &model.RefreshToken{UserID: 1, Token: "tok123", ExpiresAt: time.Now().Add(time.Hour)}
	err := repo.Create(rt)
	require.NoError(t, err)
}

func TestRefreshTokenRepository_GetByToken_Found(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "user_id", "token", "expires_at", "revoked", "created_at"}).
		AddRow(1, 1, "tok123", time.Now().Add(time.Hour), false, time.Now())
	mock.ExpectQuery(`SELECT \* FROM "refresh_tokens"`).WillReturnRows(rows)

	repo := repository.NewRefreshTokenRepository(db)
	rt, err := repo.GetByToken("tok123")
	require.NoError(t, err)
	assert.Equal(t, "tok123", rt.Token)
}

func TestRefreshTokenRepository_GetByToken_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "refresh_tokens"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "token"}))

	repo := repository.NewRefreshTokenRepository(db)
	_, err := repo.GetByToken("missing")
	assert.Error(t, err)
}

func TestRefreshTokenRepository_RevokeByToken_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewRefreshTokenRepository(db)
	err := repo.RevokeByToken("tok123")
	require.NoError(t, err)
}

func TestRefreshTokenRepository_RevokeAllByUserID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(1, 3))
	mock.ExpectCommit()

	repo := repository.NewRefreshTokenRepository(db)
	err := repo.RevokeAllByUserID(1)
	require.NoError(t, err)
}

func TestRefreshTokenRepository_DeleteExpired_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	repo := repository.NewRefreshTokenRepository(db)
	err := repo.DeleteExpired()
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GuildInviteRepository tests
// ---------------------------------------------------------------------------

func TestGuildInviteRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "guild_invites"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewGuildInviteRepository(db)
	inv := &model.GuildInvite{GuildID: 1, CreatorID: 1, Code: "ABC123"}
	err := repo.Create(inv)
	require.NoError(t, err)
}

func TestGuildInviteRepository_GetByCode_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "guild_invites"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code"}))

	repo := repository.NewGuildInviteRepository(db)
	_, err := repo.GetByCode("NOTEXIST")
	assert.Error(t, err)
}

func TestGuildInviteRepository_ListByGuildID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "guild_id", "creator_id", "code", "max_uses", "uses", "expires_at", "created_at", "updated_at"}).
		AddRow(1, 1, 1, "ABC123", 0, 0, nil, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "guild_invites"`).WillReturnRows(rows)
	// Preload("Creator")
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewGuildInviteRepository(db)
	invites, err := repo.ListByGuildID(1)
	require.NoError(t, err)
	assert.Len(t, invites, 1)
}

func TestGuildInviteRepository_IncrementUses_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "guild_invites"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildInviteRepository(db)
	err := repo.IncrementUses(1)
	require.NoError(t, err)
}

func TestGuildInviteRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "guild_invites"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewGuildInviteRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// MessageRepository tests
// ---------------------------------------------------------------------------

func TestMessageRepository_Create_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "messages"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	repo := repository.NewMessageRepository(db)
	msg := &model.Message{ChannelID: 1, UserID: 1, Content: "hello", Type: "text"}
	err := repo.Create(msg)
	require.NoError(t, err)
}

func TestMessageRepository_GetByID_NotFound(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "messages"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content"}))

	repo := repository.NewMessageRepository(db)
	_, err := repo.GetByID(999)
	assert.Error(t, err)
}

func TestMessageRepository_Update_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "messages"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewMessageRepository(db)
	msg := &model.Message{ID: 1, ChannelID: 1, UserID: 1, Content: "edited", Type: "text"}
	err := repo.Update(msg)
	require.NoError(t, err)
}

func TestMessageRepository_Delete_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "messages"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := repository.NewMessageRepository(db)
	err := repo.Delete(1)
	require.NoError(t, err)
}

func TestMessageRepository_GetByChannelID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "channel_id", "user_id", "content", "type", "is_edited", "created_at", "updated_at"}).
		AddRow(1, 1, 1, "hello", "text", false, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "messages"`).WillReturnRows(rows)
	// Preload Attachments (GORM executes in reverse preload declaration order)
	mock.ExpectQuery(`SELECT \* FROM "message_attachments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	// Preload User
	mock.ExpectQuery(`SELECT \* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewMessageRepository(db)
	msgs, err := repo.GetByChannelID(1, 0, 20)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

func TestMessageRepository_GetByUserID_Success(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)

	defer func() { _ = sqlDB.Close() }()

	rows := sqlmock.NewRows([]string{"id", "channel_id", "user_id", "content", "type", "is_edited", "created_at", "updated_at"}).
		AddRow(1, 1, 1, "hello", "text", false, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "messages"`).WillReturnRows(rows)
	// Preload Channel
	mock.ExpectQuery(`SELECT \* FROM "channels"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	repo := repository.NewMessageRepository(db)
	msgs, err := repo.GetByUserID(1, 0, 20)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}
