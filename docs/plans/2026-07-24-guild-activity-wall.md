# Guild Activity Wall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Twitter/FB-style activity wall to each guild as a new `feed` channel type, where members post, like, and single-level comment — reusing the existing `Message`/`Channel` infrastructure.

**Architecture:** A wall post and a comment are both `Message` rows in a `feed`-type channel; a post has `parent_id = nil`, a comment points its `parent_id` at the post. Likes are a new `MessageLike` table shared by posts and comments. Create/edit/delete reuse existing message endpoints (`POST /channels/:id/messages`, `PUT`/`DELETE /messages/:id`); only three new endpoints are added (posts list, comments list, like/unlike). The cross-community personal feed is explicitly out of scope and reserves the `/posts/` namespace.

**Tech Stack:** Go + Gin + GORM/PostgreSQL backend; Vue 3 frontend; existing WebSocket broadcast; testify + `testutil` function-field mocks for service tests; go-sqlmock for repo tests.

## Global Constraints

- Spec: `docs/specs/2026-07-24-guild-activity-wall-design.md` — authoritative.
- Do NOT add or modify any route under `/posts/` — reserved for a future cross-community module.
- Do NOT restructure existing `/messages/:id` routes — the wall follows them, it does not change them.
- Optional service dependencies use the existing setter idiom (`SetFileService`, `SetTranslationService`, `SetWebSocketManager`) — do NOT change `NewMessageService`'s constructor signature.
- Likes are counted with `COUNT(*)`; no denormalized `like_count` column.
- Deleting a post cascades to its comments and all likes, atomically (DB transaction in the repo layer).
- After every backend change run: `make check`.
- Frontend has no unit-test framework; frontend tasks verify manually against a running app.
- Commit after every task.

---

### Task 1: Data model — `Message.ParentID` + `MessageLike` + migration

**Files:**
- Modify: `internal/model/user.go` (Message struct ~line 82; add MessageLike near it)
- Modify: `pkg/database/database.go:85-108` (AutoMigrate list)
- Test: `internal/repository/message_like_repository_test.go` (created in Task 4; no test here — this task is a pure schema change verified by `make check` compiling + migration list)

**Interfaces:**
- Produces: `model.Message.ParentID *uint`; `model.MessageLike{ID, MessageID, UserID, CreatedAt}`.

- [ ] **Step 1: Add `ParentID` to `Message`**

In `internal/model/user.go`, inside `type Message struct`, add after the `Nonce` field:

```go
	ParentID    *uint               `gorm:"index"                                   json:"parent_id"`   // nil = 貼文/一般訊息；有值 = 留言，指向貼文
```

- [ ] **Step 2: Add `MessageLike` model**

In `internal/model/user.go`, add after the `MessageAttachment` struct:

```go
// MessageLike 貼文/留言的按讚（一人對一則只能讚一次）
type MessageLike struct {
	ID        uint      `gorm:"primarykey"                                 json:"id"`
	MessageID uint      `gorm:"not null;uniqueIndex:idx_like_message_user" json:"message_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_like_message_user" json:"user_id"`
	CreatedAt time.Time `                                                  json:"created_at"`
}
```

- [ ] **Step 3: Register in AutoMigrate**

In `pkg/database/database.go`, add to the `db.AutoMigrate(...)` list (after `&model.MessageAttachment{},`):

```go
		&model.MessageLike{},
```

- [ ] **Step 4: Verify build + migration**

Run: `make check`
Expected: build passes, no lint errors. (Migration runs at app startup; the added model is registered.)

- [ ] **Step 5: Commit**

```bash
git add internal/model/user.go pkg/database/database.go
git commit -m "feat(wall): add Message.ParentID and MessageLike model"
```

---

### Task 2: Open the `feed` channel type

**Files:**
- Modify: `internal/service/channel_service.go:96` and `:207` (type guards)
- Test: `internal/service/channel_service_test.go`

**Interfaces:**
- Produces: channel create/update accepts `type == "feed"`.

- [ ] **Step 1: Write the failing test**

Add to `internal/service/channel_service_test.go` (follow the existing create-channel test in that file for mock setup — copy its `mockCh`/`mockMember` wiring, only changing `Type`):

```go
func TestChannelService_CreateChannel_AllowsFeedType(t *testing.T) {
	// arrange mocks exactly like the existing CreateChannel success test,
	// with req.Type = "feed"
	// ... (mock guild membership as owner/admin) ...

	_, err := svc.CreateChannel(ownerID, &service.CreateChannelRequest{
		GuildID: 10,
		Name:    "動態牆",
		Type:    "feed",
	})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestChannelService_CreateChannel_AllowsFeedType -v`
Expected: FAIL — service rejects `"feed"` with an invalid-type error.

- [ ] **Step 3: Update both type guards**

In `internal/service/channel_service.go`, at line 96 and line 207, change:

```go
	if req.Type != "text" && req.Type != "voice" {
```
to:
```go
	if req.Type != "text" && req.Type != "voice" && req.Type != "feed" {
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/ -run TestChannelService_CreateChannel_AllowsFeedType -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/channel_service.go internal/service/channel_service_test.go
git commit -m "feat(wall): allow 'feed' channel type on create/update"
```

---

### Task 3: `parent_id` on message creation (comments)

**Files:**
- Modify: `internal/service/message_service.go` (`CreateMessageRequest` ~line 138; `CreateMessage` ~line 152)
- Test: `internal/service/message_service_test.go`

**Interfaces:**
- Consumes: `model.Message.ParentID` (Task 1).
- Produces: `CreateMessageRequest.ParentID *uint`; `CreateMessage` persists `ParentID` and rejects invalid parents.

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_CreateMessage_SetsParentID(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}
	post := &model.Message{ID: 7, ChannelID: 1, ParentID: nil}

	var created *model.Message
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) {
			if id == 7 {
				return post, nil
			}
			return created, nil
		},
		CreateFn: func(m *model.Message) error { m.ID = 8; created = m; return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil },
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	parent := uint(7)
	_, err := svc.CreateMessage(5, &service.CreateMessageRequest{
		ChannelID: 1, Content: "nice post", ParentID: &parent,
	})
	require.NoError(t, err)
	require.NotNil(t, created.ParentID)
	assert.Equal(t, uint(7), *created.ParentID)
}

func TestMessageService_CreateMessage_RejectsCommentOnComment(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}
	comment := &model.Message{ID: 7, ChannelID: 1, ParentID: testutil.PtrUint(3)} // already a comment

	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return comment, nil },
		CreateFn:  func(m *model.Message) error { return nil },
	}
	mockCh := &testutil.MockChannelRepository{GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil }}
	mockMember := &testutil.MockGuildMemberRepository{GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil }}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)

	parent := uint(7)
	_, err := svc.CreateMessage(5, &service.CreateMessageRequest{ChannelID: 1, Content: "x", ParentID: &parent})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run TestMessageService_CreateMessage_ -v`
Expected: FAIL — `CreateMessageRequest` has no `ParentID` field / compile error.

- [ ] **Step 3: Add the field and validation**

In `internal/service/message_service.go`, add to `CreateMessageRequest`:

```go
	ParentID  *uint  `json:"parent_id"` // 有值 = 留言，指向貼文（單層）
```

In `CreateMessage`, after `ensureChannelAccess(...)` succeeds and before building `message`, add:

```go
	// 留言：驗證父貼文存在、同頻道、且父本身是貼文（單層留言）
	if req.ParentID != nil {
		parent, perr := s.messageRepo.GetByID(*req.ParentID)
		if perr != nil {
			return nil, errors.New("parent post not found")
		}
		if parent.ChannelID != req.ChannelID {
			return nil, errors.New("parent post is in a different channel")
		}
		if parent.ParentID != nil {
			return nil, errors.New("cannot comment on a comment")
		}
	}
```

Then add `ParentID: req.ParentID,` to the `message := &model.Message{...}` literal.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/ -run TestMessageService_CreateMessage_ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/message_service.go internal/service/message_service_test.go
git commit -m "feat(wall): support parent_id (comments) on message creation"
```

---

### Task 4: `MessageLikeRepository` + testutil mock

**Files:**
- Create: `internal/repository/message_like_repository.go`
- Modify: `internal/testutil/mocks.go`
- Test: `internal/repository/message_like_repository_test.go`

**Interfaces:**
- Consumes: `model.MessageLike` (Task 1).
- Produces:
```go
type MessageLikeRepository interface {
	Create(like *model.MessageLike) error                       // idempotent (ON CONFLICT DO NOTHING)
	Delete(messageID, userID uint) error
	CountByMessageID(messageID uint) (int64, error)
	CountByMessageIDs(ids []uint) (map[uint]int64, error)
	LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error)
	DeleteByMessageIDs(ids []uint) error
}
func NewMessageLikeRepository(db *gorm.DB) MessageLikeRepository
```
- Produces mock: `testutil.MockMessageLikeRepository` with fields `CreateFn`, `DeleteFn`, `CountByMessageIDFn`, `CountByMessageIDsFn`, `LikedMessageIDsFn`, `DeleteByMessageIDsFn`.

- [ ] **Step 1: Write the failing test**

Create `internal/repository/message_like_repository_test.go` (follow the sqlmock harness from `repository_test.go` — same package `repository_test`, reuse `newTestDB`):

```go
func TestMessageLikeRepository_Create_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "message_likes"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestMessageLikeRepository -v`
Expected: FAIL — `NewMessageLikeRepository` undefined.

- [ ] **Step 3: Implement the repository**

Create `internal/repository/message_like_repository.go`:

```go
package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MessageLikeRepository 按讚資料庫操作介面
type MessageLikeRepository interface {
	Create(like *model.MessageLike) error
	Delete(messageID, userID uint) error
	CountByMessageID(messageID uint) (int64, error)
	CountByMessageIDs(ids []uint) (map[uint]int64, error)
	LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error)
	DeleteByMessageIDs(ids []uint) error
}

type messageLikeRepository struct {
	db *gorm.DB
}

// NewMessageLikeRepository 建立按讚 repository
func NewMessageLikeRepository(db *gorm.DB) MessageLikeRepository {
	return &messageLikeRepository{db: db}
}

// Create 冪等建立按讚（重複讚不報錯）
func (r *messageLikeRepository) Create(like *model.MessageLike) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *messageLikeRepository) Delete(messageID, userID uint) error {
	return r.db.
		Where("message_id = ? AND user_id = ?", messageID, userID).
		Delete(&model.MessageLike{}).Error
}

func (r *messageLikeRepository) CountByMessageID(messageID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.MessageLike{}).
		Where("message_id = ?", messageID).Count(&n).Error
	return n, err
}

type likeCountRow struct {
	MessageID uint
	Cnt       int64
}

func (r *messageLikeRepository) CountByMessageIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []likeCountRow
	err := r.db.Model(&model.MessageLike{}).
		Select("message_id, COUNT(*) AS cnt").
		Where("message_id IN ?", ids).
		Group("message_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.MessageID] = row.Cnt
	}
	return out, nil
}

func (r *messageLikeRepository) LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error) {
	out := make(map[uint]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var liked []uint
	err := r.db.Model(&model.MessageLike{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &liked).Error
	if err != nil {
		return nil, err
	}
	for _, id := range liked {
		out[id] = true
	}
	return out, nil
}

func (r *messageLikeRepository) DeleteByMessageIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("message_id IN ?", ids).Delete(&model.MessageLike{}).Error
}
```

- [ ] **Step 4: Add the testutil mock**

Append to `internal/testutil/mocks.go` (follow the `MockMessageRepository` shape already in that file):

```go
// MockMessageLikeRepository is a test double for repository.MessageLikeRepository.
type MockMessageLikeRepository struct {
	CreateFn              func(like *model.MessageLike) error
	DeleteFn              func(messageID, userID uint) error
	CountByMessageIDFn    func(messageID uint) (int64, error)
	CountByMessageIDsFn   func(ids []uint) (map[uint]int64, error)
	LikedMessageIDsFn     func(userID uint, ids []uint) (map[uint]bool, error)
	DeleteByMessageIDsFn  func(ids []uint) error
}

var _ repository.MessageLikeRepository = (*MockMessageLikeRepository)(nil)

func (m *MockMessageLikeRepository) Create(like *model.MessageLike) error { return m.CreateFn(like) }
func (m *MockMessageLikeRepository) Delete(messageID, userID uint) error   { return m.DeleteFn(messageID, userID) }
func (m *MockMessageLikeRepository) CountByMessageID(id uint) (int64, error) {
	return m.CountByMessageIDFn(id)
}
func (m *MockMessageLikeRepository) CountByMessageIDs(ids []uint) (map[uint]int64, error) {
	return m.CountByMessageIDsFn(ids)
}
func (m *MockMessageLikeRepository) LikedMessageIDs(userID uint, ids []uint) (map[uint]bool, error) {
	return m.LikedMessageIDsFn(userID, ids)
}
func (m *MockMessageLikeRepository) DeleteByMessageIDs(ids []uint) error {
	return m.DeleteByMessageIDsFn(ids)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/repository/ -run TestMessageLikeRepository -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/message_like_repository.go internal/repository/message_like_repository_test.go internal/testutil/mocks.go
git commit -m "feat(wall): add MessageLikeRepository and testutil mock"
```

---

### Task 5: Feed cursor queries + cascade delete on `MessageRepository`

**Files:**
- Modify: `internal/repository/message_repository.go` (interface ~line 12; impls)
- Modify: `internal/testutil/mocks.go` (extend `MockMessageRepository`)
- Test: `internal/repository/message_repository_test.go` (add cascade test)

**Interfaces:**
- Produces on `MessageRepository`:
```go
GetPostsByChannelCursor(channelID, before uint, limit int) ([]*model.Message, error) // parent_id IS NULL, newest-first
GetCommentsByPostCursor(postID, before uint, limit int) ([]*model.Message, error)     // parent_id = postID, chronological
CountCommentsByPostIDs(ids []uint) (map[uint]int64, error)                            // group by parent_id
DeletePostCascade(postID uint) error                                                  // tx: likes + comments + post
```
- Produces mock fields on `MockMessageRepository`: `GetPostsByChannelCursorFn`, `GetCommentsByPostCursorFn`, `CountCommentsByPostIDsFn`, `DeletePostCascadeFn`.

- [ ] **Step 1: Write the failing test (cascade in a transaction)**

Add to `internal/repository/message_repository_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestMessageRepository_DeletePostCascade -v`
Expected: FAIL — `DeletePostCascade` undefined.

- [ ] **Step 3: Implement the four methods**

Add to the `MessageRepository` interface in `internal/repository/message_repository.go`:

```go
	GetPostsByChannelCursor(channelID, before uint, limit int) ([]*model.Message, error)
	GetCommentsByPostCursor(postID, before uint, limit int) ([]*model.Message, error)
	CountCommentsByPostIDs(ids []uint) (map[uint]int64, error)
	DeletePostCascade(postID uint) error
```

Add implementations (the cursor helpers mirror `GetByChannelIDCursor`):

```go
// GetPostsByChannelCursor 取得 feed 頻道的貼文（parent_id IS NULL，新到舊 cursor 分頁）
func (r *messageRepository) GetPostsByChannelCursor(
	channelID, before uint, limit int,
) ([]*model.Message, error) {
	var posts []*model.Message
	q := r.db.
		Preload("User").Preload("Attachments.File").
		Where("channel_id = ? AND parent_id IS NULL", channelID).
		Order("id DESC").
		Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	// 貼文列表保持新到舊，不反轉
	return posts, q.Find(&posts).Error
}

// GetCommentsByPostCursor 取得某貼文的留言（parent_id = postID，回傳時間順序）
func (r *messageRepository) GetCommentsByPostCursor(
	postID, before uint, limit int,
) ([]*model.Message, error) {
	var comments []*model.Message
	q := r.db.
		Preload("User").Preload("Attachments.File").
		Where("parent_id = ?", postID).
		Order("id DESC").
		Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Find(&comments).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}
	return comments, nil
}

// CountCommentsByPostIDs 批次計算每則貼文的留言數（避免 N+1）
func (r *messageRepository) CountCommentsByPostIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		ParentID uint
		Cnt      int64
	}
	var rows []row
	err := r.db.Model(&model.Message{}).
		Select("parent_id, COUNT(*) AS cnt").
		Where("parent_id IN ?", ids).
		Group("parent_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.ParentID] = x.Cnt
	}
	return out, nil
}

// DeletePostCascade 於單一 transaction 內刪除貼文、其留言、及全部相關讚
func (r *messageRepository) DeletePostCascade(postID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var commentIDs []uint
		if err := tx.Model(&model.Message{}).
			Where("parent_id = ?", postID).
			Pluck("id", &commentIDs).Error; err != nil {
			return err
		}
		ids := append(commentIDs, postID)
		if err := tx.Where("message_id IN ?", ids).
			Delete(&model.MessageLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("parent_id = ?", postID).
			Delete(&model.Message{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Message{}, postID).Error
	})
}
```

- [ ] **Step 4: Extend the testutil mock**

Add fields to `MockMessageRepository` in `internal/testutil/mocks.go` and their methods:

```go
	GetPostsByChannelCursorFn func(channelID, before uint, limit int) ([]*model.Message, error)
	GetCommentsByPostCursorFn func(postID, before uint, limit int) ([]*model.Message, error)
	CountCommentsByPostIDsFn  func(ids []uint) (map[uint]int64, error)
	DeletePostCascadeFn       func(postID uint) error
```

```go
func (m *MockMessageRepository) GetPostsByChannelCursor(channelID, before uint, limit int) ([]*model.Message, error) {
	return m.GetPostsByChannelCursorFn(channelID, before, limit)
}
func (m *MockMessageRepository) GetCommentsByPostCursor(postID, before uint, limit int) ([]*model.Message, error) {
	return m.GetCommentsByPostCursorFn(postID, before, limit)
}
func (m *MockMessageRepository) CountCommentsByPostIDs(ids []uint) (map[uint]int64, error) {
	return m.CountCommentsByPostIDsFn(ids)
}
func (m *MockMessageRepository) DeletePostCascade(postID uint) error {
	return m.DeletePostCascadeFn(postID)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/repository/ -run TestMessageRepository_DeletePostCascade -v && go build ./...`
Expected: PASS and build OK. (If sqlmock argument-matching on the cascade is brittle, relax the `WithArgs`/regex to match GORM's actual emitted SQL — the goal is verifying order: pluck → delete likes → delete comments → delete post.)

- [ ] **Step 6: Commit**

```bash
git add internal/repository/message_repository.go internal/testutil/mocks.go internal/repository/message_repository_test.go
git commit -m "feat(wall): add feed cursor queries and cascade post delete"
```

---

### Task 6: Like service methods + `post_like` broadcast + wire like repo

**Files:**
- Modify: `internal/service/message_service.go` (interface, struct, setter, methods)
- Modify: `cmd/server/main.go` (or wherever services are constructed — wire `SetLikeRepo`)
- Test: `internal/service/message_service_test.go`

**Interfaces:**
- Consumes: `repository.MessageLikeRepository` (Task 4), `MessageRepository` (Task 5).
- Produces on `MessageService`:
```go
SetLikeRepo(r repository.MessageLikeRepository)
LikePost(messageID, userID uint) (int64, error)    // returns new like count; idempotent
UnlikePost(messageID, userID uint) (int64, error)  // returns new like count
```

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_LikePost_ReturnsCountAndBroadcasts(t *testing.T) {
	msg := &model.Message{ID: 7, ChannelID: 1}
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	mockLike := &testutil.MockMessageLikeRepository{
		CreateFn:           func(l *model.MessageLike) error { return nil },
		CountByMessageIDFn: func(id uint) (int64, error) { return 4, nil },
	}
	ws := &testutil.MockWebSocketManager{} // records BroadcastToChannel calls; see existing usage in this test file

	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)
	svc.SetLikeRepo(mockLike)
	svc.SetWebSocketManager(ws)

	n, err := svc.LikePost(7, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(4), n)
	// assert ws recorded a "post_like" event for channel 1 (adapt to MockWebSocketManager's recording API)
}
```

(If no `MockWebSocketManager` exists in `testutil`, add a minimal one that records `BroadcastToChannel(channelID, msgType, data)` calls, mirroring the `WebSocketManager` interface used by `messageService`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestMessageService_LikePost -v`
Expected: FAIL — `SetLikeRepo`/`LikePost` undefined.

- [ ] **Step 3: Implement**

In `internal/service/message_service.go`:

Add to the `MessageService` interface:
```go
	SetLikeRepo(r repository.MessageLikeRepository)
	LikePost(messageID, userID uint) (int64, error)
	UnlikePost(messageID, userID uint) (int64, error)
```

Add to the `messageService` struct: `likeRepo repository.MessageLikeRepository`.

Add the setter and methods:
```go
// SetLikeRepo 設定按讚 repository（動態牆功能所需）
func (s *messageService) SetLikeRepo(r repository.MessageLikeRepository) { s.likeRepo = r }

// likeAccessCheck 確認訊息存在、使用者是該頻道所屬 guild 的成員，回傳訊息
func (s *messageService) likeAccessCheck(messageID, userID uint) (*model.Message, error) {
	msg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	channel, err := s.channelRepo.GetByID(msg.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}
	if channel.GuildID != nil {
		if member, merr := s.guildMemberRepo.GetMember(*channel.GuildID, userID); merr != nil || member == nil {
			return nil, ErrNotChannelMemberMsg
		}
	}
	return msg, nil
}

func (s *messageService) broadcastLikeCount(msg *model.Message) (int64, error) {
	n, err := s.likeRepo.CountByMessageID(msg.ID)
	if err != nil {
		return 0, err
	}
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(msg.ChannelID, "post_like", map[string]any{
			"message_id": msg.ID,
			"like_count": n,
		})
	}
	return n, nil
}

// LikePost 按讚（idempotent）
func (s *messageService) LikePost(messageID, userID uint) (int64, error) {
	msg, err := s.likeAccessCheck(messageID, userID)
	if err != nil {
		return 0, err
	}
	if err := s.likeRepo.Create(&model.MessageLike{MessageID: messageID, UserID: userID}); err != nil {
		return 0, err
	}
	return s.broadcastLikeCount(msg)
}

// UnlikePost 收回讚
func (s *messageService) UnlikePost(messageID, userID uint) (int64, error) {
	msg, err := s.likeAccessCheck(messageID, userID)
	if err != nil {
		return 0, err
	}
	if err := s.likeRepo.Delete(messageID, userID); err != nil {
		return 0, err
	}
	return s.broadcastLikeCount(msg)
}
```

- [ ] **Step 4: Wire the like repo where services are built**

Find where `NewMessageService(...)` is called and the file repo/etc. are wired (grep `SetFileService` for the spot). Next to that, add:
```go
	messageService.SetLikeRepo(repository.NewMessageLikeRepository(db))
```
(Use the same `db` handle and message-service variable already in scope there.)

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/service/ -run TestMessageService_LikePost -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/service/message_service.go internal/testutil/mocks.go cmd/server/main.go internal/service/message_service_test.go
git commit -m "feat(wall): like/unlike service with post_like broadcast"
```

---

### Task 7: Cascade wiring in `DeleteMessage`

**Files:**
- Modify: `internal/service/message_service.go` (`DeleteMessage`, ~line 457 where `messageRepo.Delete` is called)
- Test: `internal/service/message_service_test.go`

**Interfaces:**
- Consumes: `DeletePostCascade` (Task 5), `likeRepo.DeleteByMessageIDs` (Task 4).

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_DeleteMessage_FeedPostCascades(t *testing.T) {
	post := &model.Message{ID: 7, ChannelID: 1, UserID: 5, ParentID: nil}
	cascaded := false
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn:           func(id uint) (*model.Message, error) { return post, nil },
		DeletePostCascadeFn: func(postID uint) error { cascaded = true; return nil },
		DeleteFn:            func(id uint) error { t.Fatalf("plain Delete must not be used for a feed post"); return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	svc := service.NewMessageService(mockMsg, mockCh, &testutil.MockGuildMemberRepository{}, nil)

	require.NoError(t, svc.DeleteMessage(7, 5))
	assert.True(t, cascaded)
}

func TestMessageService_DeleteMessage_CommentClearsOwnLikes(t *testing.T) {
	comment := &model.Message{ID: 8, ChannelID: 1, UserID: 5, ParentID: testutil.PtrUint(7)}
	var clearedIDs []uint
	deleted := false
	mockMsg := &testutil.MockMessageRepository{
		GetByIDFn: func(id uint) (*model.Message, error) { return comment, nil },
		DeleteFn:  func(id uint) error { deleted = true; return nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockLike := &testutil.MockMessageLikeRepository{
		DeleteByMessageIDsFn: func(ids []uint) error { clearedIDs = ids; return nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, &testutil.MockGuildMemberRepository{}, nil)
	svc.SetLikeRepo(mockLike)

	require.NoError(t, svc.DeleteMessage(8, 5))
	assert.True(t, deleted)
	assert.Equal(t, []uint{8}, clearedIDs)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run TestMessageService_DeleteMessage_ -v`
Expected: FAIL — cascade branch not implemented (plain `Delete` called for the post).

- [ ] **Step 3: Implement the branch**

In `DeleteMessage`, replace the existing delete block:
```go
	// 刪除訊息
	if err := s.messageRepo.Delete(messageID); err != nil {
		return err
	}
```
with:
```go
	// feed 頻道的貼文（parent_id = nil）→ cascade 刪留言與所有讚
	if channel != nil && channel.Type == "feed" && message.ParentID == nil {
		if err := s.messageRepo.DeletePostCascade(messageID); err != nil {
			return err
		}
	} else {
		if s.likeRepo != nil {
			_ = s.likeRepo.DeleteByMessageIDs([]uint{messageID})
		}
		if err := s.messageRepo.Delete(messageID); err != nil {
			return err
		}
	}
```

Note: `channel` is already fetched earlier in `DeleteMessage` only inside the non-owner branch. Ensure `channel` is loaded for all paths — fetch it once near the top of `DeleteMessage` (after loading `message`): `channel, _ := s.channelRepo.GetByID(message.ChannelID)` and reuse it in both the auth check and this branch. Adjust the existing auth code to use the pre-loaded `channel`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/ -run TestMessageService_DeleteMessage_ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/message_service.go internal/service/message_service_test.go
git commit -m "feat(wall): cascade delete feed posts (comments + likes)"
```

---

### Task 8: HTTP handlers + routes (posts list, comments list, like/unlike)

**Files:**
- Modify: `internal/handler/message_handler.go` (add handlers + `PostResponse` enrichment; the handler already holds `messageService`)
- Modify: `internal/service/message_service.go` (add `ListPosts`/`ListComments` returning enriched DTOs)
- Modify: `internal/server/server.go` (routes ~line 353 channels group, ~line 372 messages group)
- Test: `internal/service/message_service_test.go` (ListPosts enrichment)

**Interfaces:**
- Consumes: Task 5 cursor queries, Task 4 like counts.
- Produces on `MessageService`:
```go
type PostResponse struct {
	*model.Message
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	LikedByMe    bool  `json:"liked_by_me"`
}
type PostListResponse struct {
	Posts   []*PostResponse `json:"posts"`
	HasMore bool            `json:"has_more"`
}
ListPosts(channelID, userID uint, limit int, before uint) (*PostListResponse, error)
ListComments(postID, userID uint, limit int, before uint) (*MessageListResponse, error)
```
- Produces HTTP:
  - `GET  /api/v1/channels/:id/posts` → `PostListResponse`
  - `GET  /api/v1/messages/:id/comments` → `MessageListResponse`
  - `PUT  /api/v1/messages/:id/like` → `{ "message_id", "like_count" }`
  - `DELETE /api/v1/messages/:id/like` → `{ "message_id", "like_count" }`

- [ ] **Step 1: Write the failing test (ListPosts enrichment)**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_ListPosts_EnrichesCounts(t *testing.T) {
	posts := []*model.Message{{ID: 7, ChannelID: 1}, {ID: 6, ChannelID: 1}}
	mockMsg := &testutil.MockMessageRepository{
		GetPostsByChannelCursorFn: func(cid, before uint, limit int) ([]*model.Message, error) { return posts, nil },
		CountCommentsByPostIDsFn:  func(ids []uint) (map[uint]int64, error) { return map[uint]int64{7: 2}, nil },
	}
	mockCh := &testutil.MockChannelRepository{
		GetByIDFn: func(id uint) (*model.Channel, error) {
			return &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "feed"}, nil
		},
	}
	mockMember := &testutil.MockGuildMemberRepository{
		GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil },
	}
	mockLike := &testutil.MockMessageLikeRepository{
		CountByMessageIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{7: 4}, nil },
		LikedMessageIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{7: true}, nil },
	}
	svc := service.NewMessageService(mockMsg, mockCh, mockMember, nil)
	svc.SetLikeRepo(mockLike)

	resp, err := svc.ListPosts(1, 5, 20, 0)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 2)
	assert.Equal(t, int64(4), resp.Posts[0].LikeCount)
	assert.Equal(t, int64(2), resp.Posts[0].CommentCount)
	assert.True(t, resp.Posts[0].LikedByMe)
	assert.Equal(t, int64(0), resp.Posts[1].LikeCount)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestMessageService_ListPosts -v`
Expected: FAIL — `ListPosts` undefined.

- [ ] **Step 3: Implement service methods**

Add the DTOs (near `MessageListResponse`) and methods to `internal/service/message_service.go`. Add both to the `MessageService` interface too.

```go
type PostResponse struct {
	*model.Message
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	LikedByMe    bool  `json:"liked_by_me"`
}

type PostListResponse struct {
	Posts   []*PostResponse `json:"posts"`
	HasMore bool            `json:"has_more"`
}

// ListPosts 回傳 feed 頻道貼文（新到舊），每筆帶讚數/留言數/我是否讚過
func (s *messageService) ListPosts(channelID, userID uint, limit int, before uint) (*PostListResponse, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}
	if err := s.ensureChannelAccess(channel, channelID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	posts, err := s.messageRepo.GetPostsByChannelCursor(channelID, before, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := false
	if len(posts) > limit {
		hasMore = true
		posts = posts[:limit]
	}

	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	var likeCounts, commentCounts map[uint]int64
	var liked map[uint]bool
	if s.likeRepo != nil {
		likeCounts, _ = s.likeRepo.CountByMessageIDs(ids)
		liked, _ = s.likeRepo.LikedMessageIDs(userID, ids)
	}
	commentCounts, _ = s.messageRepo.CountCommentsByPostIDs(ids)

	out := make([]*PostResponse, len(posts))
	for i, p := range posts {
		out[i] = &PostResponse{
			Message:      p,
			LikeCount:    likeCounts[p.ID],
			CommentCount: commentCounts[p.ID],
			LikedByMe:    liked[p.ID],
		}
	}
	return &PostListResponse{Posts: out, HasMore: hasMore}, nil
}

// ListComments 回傳某貼文的留言（時間順序）
func (s *messageService) ListComments(postID, userID uint, limit int, before uint) (*MessageListResponse, error) {
	post, err := s.messageRepo.GetByID(postID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	channel, err := s.channelRepo.GetByID(post.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}
	if err := s.ensureChannelAccess(channel, post.ChannelID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	comments, err := s.messageRepo.GetCommentsByPostCursor(postID, before, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := false
	if len(comments) > limit {
		hasMore = true
		comments = comments[len(comments)-limit:]
	}
	return &MessageListResponse{Messages: comments, HasMore: hasMore}, nil
}
```

Note: if `ensureChannelAccess` is not already a method usable here, reuse whatever access-check the existing `ListChannelMessages` uses (mirror that code path exactly).

- [ ] **Step 4: Run service test to verify it passes**

Run: `go test ./internal/service/ -run TestMessageService_ListPosts -v`
Expected: PASS

- [ ] **Step 5: Add HTTP handlers**

In `internal/handler/message_handler.go`, add (mirror `ListChannelMessages` for param parsing — `c.Param("id")`, `before`/`limit` query, `userID` from context):

```go
// ListPosts GET /channels/:id/posts
func (h *MessageHandler) ListPosts(c *gin.Context) {
	channelID := parseUintParam(c, "id")
	userID := c.GetUint("userID")
	limit, before := parseListQuery(c) // reuse the same helper ListChannelMessages uses
	resp, err := h.messageService.ListPosts(channelID, userID, limit, before)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListComments GET /messages/:id/comments
func (h *MessageHandler) ListComments(c *gin.Context) {
	postID := parseUintParam(c, "id")
	userID := c.GetUint("userID")
	limit, before := parseListQuery(c)
	resp, err := h.messageService.ListComments(postID, userID, limit, before)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// LikePost PUT /messages/:id/like
func (h *MessageHandler) LikePost(c *gin.Context) {
	messageID := parseUintParam(c, "id")
	userID := c.GetUint("userID")
	n, err := h.messageService.LikePost(messageID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message_id": messageID, "like_count": n})
}

// UnlikePost DELETE /messages/:id/like
func (h *MessageHandler) UnlikePost(c *gin.Context) {
	messageID := parseUintParam(c, "id")
	userID := c.GetUint("userID")
	n, err := h.messageService.UnlikePost(messageID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message_id": messageID, "like_count": n})
}
```

Use the exact param/query parsing idioms already present in `message_handler.go` (replace `parseUintParam`/`parseListQuery` with whatever that file already does — do NOT invent new helpers if the file inlines the parsing).

- [ ] **Step 6: Wire routes**

In `internal/server/server.go`, in the `channels` group (near line 353) add:
```go
				channels.GET("/:id/posts", s.messageHandler.ListPosts)
```
In the `messages` group (near line 372) add:
```go
				messages.GET("/:id/comments", s.messageHandler.ListComments)
				messages.PUT("/:id/like", s.messageHandler.LikePost)
				messages.DELETE("/:id/like", s.messageHandler.UnlikePost)
```

- [ ] **Step 7: Full check**

Run: `make check`
Expected: build, lint, and all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/service/message_service.go internal/handler/message_handler.go internal/server/server.go internal/service/message_service_test.go
git commit -m "feat(wall): posts/comments list + like endpoints"
```

---

### Task 9: Frontend — API client, create-channel option, sidebar icon

**Files:**
- Modify: `web/src/api/index.js` (endpoints ~line 25-49; methods ~line 204-217)
- Modify: `web/src/components/modals/CreateChannelModal.vue` (add feed radio)
- Modify: `web/src/components/ChannelSidebar.vue` (feed icon)
- Verify: manual (no frontend test framework)

**Interfaces:**
- Consumes: Task 8 endpoints.
- Produces: `api.getPosts`, `api.getComments`, `api.createPost`, `api.createComment`, `api.likePost`, `api.unlikePost`; a `feed` option in the create-channel modal.

- [ ] **Step 1: Add endpoints + methods**

In `web/src/api/index.js` `EP` map add:
```js
    CHANNEL_POSTS: (channelId) => `/api/v1/channels/${channelId}/posts`,
    MESSAGE_COMMENTS: (id) => `/api/v1/messages/${id}/comments`,
    MESSAGE_LIKE: (id) => `/api/v1/messages/${id}/like`,
```

In the methods block (near `sendMessage`) add:
```js
    getPosts(channelId, limit = 20, before = null) {
      const q = new URLSearchParams({ limit }); if (before) q.set('before', before)
      return this.get(`${EP.CHANNEL_POSTS(channelId)}?${q}`)
    },
    createPost(channelId, content, fileIds = []) {
      return this.post(EP.CHANNEL_MESSAGES(channelId), { content, file_ids: fileIds })
    },
    getComments(postId, limit = 50, before = null) {
      const q = new URLSearchParams({ limit }); if (before) q.set('before', before)
      return this.get(`${EP.MESSAGE_COMMENTS(postId)}?${q}`)
    },
    createComment(channelId, postId, content) {
      return this.post(EP.CHANNEL_MESSAGES(channelId), { content, parent_id: postId })
    },
    likePost(id) { return this.put(EP.MESSAGE_LIKE(id)) },
    unlikePost(id) { return this.del(EP.MESSAGE_LIKE(id)) },
```
(If the client has no `put` helper, use whatever verb helper exists — match `updateMessage`'s style.)

- [ ] **Step 2: Add the feed option to CreateChannelModal**

In `web/src/components/modals/CreateChannelModal.vue`, after the voice `<label>` (~line 54-57) add:
```html
            <label class="type-option" :class="{ active: channelType === 'feed' }">
              <input type="radio" v-model="channelType" value="feed" />
              <i class="fas fa-stream"></i> {{ t('createChannel.feedChannel') }}
            </label>
```
Add the i18n key `createChannel.feedChannel` (e.g. "動態牆") to each locale file under `web/src/locales/` (or wherever `createChannel.textChannel` is defined — run the repo's `web/scripts/check-i18n-keys.mjs` to confirm all locales covered).

- [ ] **Step 3: Add a feed icon in ChannelSidebar**

In `web/src/components/ChannelSidebar.vue`, wherever the channel icon is chosen by `channel.type` (the `fa-hashtag`/`fa-volume-up` ternary), extend it so `type === 'feed'` renders `fa-stream`.

- [ ] **Step 4: Verify manually**

Run the app (per the repo's run instructions). Create a channel with type "動態牆"; confirm it appears in the sidebar with the stream icon and that the backend accepts it (Task 2).
Expected: feed channel is created and listed.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/index.js web/src/components/modals/CreateChannelModal.vue web/src/components/ChannelSidebar.vue web/src/locales/
git commit -m "feat(wall): frontend api client + feed channel create option"
```

---

### Task 10: Frontend — `FeedArea` component + channel-type switch + WS

**Files:**
- Create: `web/src/components/FeedArea.vue`
- Modify: `web/src/views/ChatView.vue:105` (switch ChatArea ↔ FeedArea by channel type)
- Modify: `web/src/composables/useWebSocket.js:120-123` (handle `post_like`; `message_create` already dispatched)
- Verify: manual

**Interfaces:**
- Consumes: Task 9 API methods; existing `message_create`/`message_update`/`message_delete` WS events; new `post_like` event.

- [ ] **Step 1: Switch content area by channel type**

In `web/src/views/ChatView.vue`, import `FeedArea` and replace the `<ChatArea .../>` in the guild branch (line ~105) with:
```html
      <FeedArea
        v-if="store.currentChannel?.type === 'feed'"
        @open-sidebar="mobileChannelSidebarOpen = true"
      />
      <ChatArea
        v-else
        @toggle-members="toggleMembers"
        @open-sidebar="mobileChannelSidebarOpen = true"
      />
```

- [ ] **Step 2: Create `FeedArea.vue`**

Create `web/src/components/FeedArea.vue`. Mirror `ChatArea.vue`'s shell (header, `useAppStore`, `useI18n`) and reuse `MessageInput`/attachment upload where practical. Minimum behavior:
- On mount and on `store.currentChannel.id` change: `api.getPosts(channelId)` → `posts` ref (array, newest-first).
- Compose box at top: text + optional image (reuse the existing upload flow used by `MessageInput`); submit → `api.createPost(channelId, content, fileIds)`.
- Each post card: author, time, content, attachments, a like button showing `like_count` (toggles `api.likePost`/`api.unlikePost`, optimistic), and a comment toggle showing `comment_count`.
- Expanding a post: `api.getComments(postId)` → render single-level comments + a comment box → `api.createComment(channelId, postId, content)`.
- Infinite scroll up: when scrolled to top, `api.getPosts(channelId, 20, oldestLoadedPostId)` and append older.
- Subscribe to WS events for the current channel (use the same subscription mechanism `MessageList`/`ChatArea` use — grep how they consume `notify('message', ...)`):
  - `message` (create) with `parent_id == null` and matching `channel_id` → prepend to `posts`.
  - `message` (create) with `parent_id` → if that post's comments are loaded, append.
  - `message_update` → replace matching post/comment content.
  - `message_delete` → remove matching post (and its comment block) or comment.
  - `post_like` → set `like_count` on the matching post.

Follow `DESIGN.md` (Kinetic Noir) for styling; reuse existing message/card CSS classes rather than inventing a new design language.

- [ ] **Step 3: Dispatch `post_like` in the WS composable**

In `web/src/composables/useWebSocket.js`, alongside the existing message cases (line ~120), add:
```js
            case 'post_like': notify('post_like', msg.d); break
```

- [ ] **Step 4: Verify manually**

Run the app. In a feed channel: create a post, like/unlike it (count updates live), comment on it, and confirm a second browser session sees the post/like/comment appear in real time. Delete a post and confirm its comments disappear too.
Expected: full wall flow works end to end with live updates.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/FeedArea.vue web/src/views/ChatView.vue web/src/composables/useWebSocket.js
git commit -m "feat(wall): FeedArea view with realtime posts, likes, comments"
```

---

## Notes for the implementer

- **Order matters:** Tasks 1→8 are backend and strictly ordered (each consumes the previous). Tasks 9-10 are frontend and depend on Task 8's endpoints being live.
- **`make check` after each backend task.** If go-sqlmock argument matching in Tasks 4-5 proves brittle against GORM's emitted SQL, relax the regex/`WithArgs` — the behavioral guarantees live in the service-layer tests (Tasks 3, 6, 7, 8).
- **Do not touch `/posts/`** anywhere — it is reserved (Global Constraints).
- **Pre-existing orphan behavior** (chat message delete not clearing attachments/mentions) is explicitly out of scope; do not "fix" it here.
