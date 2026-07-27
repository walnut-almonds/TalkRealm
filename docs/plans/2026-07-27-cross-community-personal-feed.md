# Cross-Community Personal Feed — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an independent Twitter-style personal feed module: users create personal posts, follow other users one-directionally, and see a chronological timeline of "followees + self", with likes and single-level comments.

**Architecture:** A self-contained `feed` module (files prefixed `feed_`, routes under `/api/v1/feed`), owning 5 new tables (`Follow`, `FeedPost`, `FeedComment`, `FeedPostLike`, `FeedPostAttachment`). It does NOT reuse `Message`/`Channel` and does NOT share schema with the guild wall. Timeline is chronological fan-out-on-read (`WHERE author_id IN (followees ∪ self) ORDER BY id DESC`). No WebSocket — pure pull. Boundary reads only: `User` (author), `Friendship` (follow suggestions), `File` + presign pipeline (post images).

**Tech Stack:** Go + Gin + GORM/PostgreSQL; Vue 3 (fills existing `/feed` route); testify + `testutil` function-field mocks for service tests; go-sqlmock for repo tests.

## Global Constraints

- Spec: `docs/specs/2026-07-27-cross-community-personal-feed-design.md` — authoritative.
- Do NOT modify or share the wall/chat tables (`Message`, `Channel`, `MessageLike`, etc.). Read `User`/`Friendship`/`File` at the boundary only.
- No WebSocket in this module (MVP is pull-only).
- Likes counted with `COUNT(*)`; no denormalized count columns.
- Deleting a post cascades to its comments, likes, and attachments, atomically (DB transaction in the repo layer).
- Follow is one-directional and idempotent; a user cannot follow themselves.
- Edit/delete allowed for the author only (posts and comments).
- Mirror existing codebase idioms established by the guild wall: cursor query = fetch `id DESC` then reverse for chronological callers; batch-count maps to avoid N+1; enrichment DTO embeds the model pointer; optional service deps via setters; `testutil` function-field mocks with `var _ Interface = (*Mock)(nil)` assertions.
- After every backend task run: `make check` (fall back to `go build ./... && go vet ./... && go test ./...` if `-race`/mise shims break on this Windows box — no gcc here).
- Frontend has no unit-test framework; frontend tasks verify with `npm --prefix web run build` + manual checks.
- Commit after every task.

---

### Task 1: Data model (5 tables) + migration

**Files:**
- Create: `internal/model/feed.go`
- Modify: `pkg/database/database.go` (AutoMigrate list)

**Interfaces:**
- Produces: `model.Follow`, `model.FeedPost`, `model.FeedComment`, `model.FeedPostLike`, `model.FeedPostAttachment`.

- [ ] **Step 1: Create the model file**

Create `internal/model/feed.go`:

```go
package model

import "time"

// Follow 單向追蹤關係（feed 模組自有，不共用 Friendship）
type Follow struct {
	ID         uint      `gorm:"primarykey"                              json:"id"`
	FollowerID uint      `gorm:"not null;uniqueIndex:idx_follow_pair;index" json:"follower_id"`
	FolloweeID uint      `gorm:"not null;uniqueIndex:idx_follow_pair;index" json:"followee_id"`
	CreatedAt  time.Time `                                               json:"created_at"`
}

// FeedPost 個人貼文（跨社群，不屬於任何 guild）
type FeedPost struct {
	ID          uint                 `gorm:"primarykey"           json:"id"`
	AuthorID    uint                 `gorm:"not null;index"       json:"author_id"`
	Author      User                 `gorm:"foreignKey:AuthorID"  json:"author"`
	Content     string               `gorm:"not null"             json:"content"`
	IsEdited    bool                 `gorm:"default:false"        json:"is_edited"`
	Attachments []FeedPostAttachment `gorm:"foreignKey:PostID"    json:"attachments"`
	CreatedAt   time.Time            `                            json:"created_at"`
	UpdatedAt   time.Time            `                            json:"updated_at"`
}

// AfterFind 確保 Attachments 序列化為 [] 而非 null
func (p *FeedPost) AfterFind(_ *gorm.DB) error {
	if p.Attachments == nil {
		p.Attachments = []FeedPostAttachment{}
	}
	return nil
}

// FeedComment 貼文的單層留言
type FeedComment struct {
	ID        uint      `gorm:"primarykey"          json:"id"`
	PostID    uint      `gorm:"not null;index"      json:"post_id"`
	AuthorID  uint      `gorm:"not null"            json:"author_id"`
	Author    User      `gorm:"foreignKey:AuthorID" json:"author"`
	Content   string    `gorm:"not null"            json:"content"`
	IsEdited  bool      `gorm:"default:false"       json:"is_edited"`
	CreatedAt time.Time `                           json:"created_at"`
	UpdatedAt time.Time `                           json:"updated_at"`
}

// FeedPostLike 貼文按讚（一人對一貼文只能讚一次）
type FeedPostLike struct {
	ID        uint      `gorm:"primarykey"                                json:"id"`
	PostID    uint      `gorm:"not null;uniqueIndex:idx_feedlike_pair"    json:"post_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_feedlike_pair"    json:"user_id"`
	CreatedAt time.Time `                                                 json:"created_at"`
}

// FeedPostAttachment 貼文附件（連到通用 File）
type FeedPostAttachment struct {
	ID        uint      `gorm:"primarykey"        json:"id"`
	PostID    uint      `gorm:"not null;index"    json:"post_id"`
	FileID    uint      `gorm:"not null"          json:"file_id"`
	File      File      `gorm:"foreignKey:FileID" json:"file"`
	CreatedAt time.Time `                         json:"created_at"`
}
```

Note: `AfterFind` uses `gorm.DB` — add the import. Change the import block to:
```go
import (
	"time"

	"gorm.io/gorm"
)
```

- [ ] **Step 2: Register in AutoMigrate**

In `pkg/database/database.go`, add to the `db.AutoMigrate(...)` list (after the wall's `&model.MessageLike{},` or at the end before the closing paren):

```go
		&model.Follow{},
		&model.FeedPost{},
		&model.FeedComment{},
		&model.FeedPostLike{},
		&model.FeedPostAttachment{},
```

- [ ] **Step 3: Verify build + migration**

Run: `make check` (or the non-race fallback).
Expected: build passes, no lint errors.

- [ ] **Step 4: Commit**

```bash
git add internal/model/feed.go pkg/database/database.go
git commit -m "feat(feed): add Follow/FeedPost/FeedComment/FeedPostLike/FeedPostAttachment models"
```

---

### Task 2: FollowRepository + testutil mock

**Files:**
- Create: `internal/repository/feed_follow_repository.go`
- Modify: `internal/testutil/mocks.go`
- Test: `internal/repository/feed_follow_repository_test.go`

**Interfaces:**
- Produces:
```go
type FollowRepository interface {
	Follow(followerID, followeeID uint) error            // idempotent (ON CONFLICT DO NOTHING)
	Unfollow(followerID, followeeID uint) error
	IsFollowing(followerID, followeeID uint) (bool, error)
	FolloweeIDs(followerID uint) ([]uint, error)          // for timeline source
	ListFollowing(userID uint) ([]*model.Follow, error)   // preload Followee User
	ListFollowers(userID uint) ([]*model.Follow, error)   // preload Follower User
	CountFollowing(userID uint) (int64, error)
	CountFollowers(userID uint) (int64, error)
}
func NewFollowRepository(db *gorm.DB) FollowRepository
```
- Produces mock: `testutil.MockFollowRepository` with a `Fn` field per method.

- [ ] **Step 1: Write the failing test**

Create `internal/repository/feed_follow_repository_test.go` (same `repository_test` package + `newTestDB` harness):

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestFollowRepository -v`
Expected: FAIL — `NewFollowRepository` undefined.

- [ ] **Step 3: Implement the repository**

Create `internal/repository/feed_follow_repository.go`:

```go
package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository interface {
	Follow(followerID, followeeID uint) error
	Unfollow(followerID, followeeID uint) error
	IsFollowing(followerID, followeeID uint) (bool, error)
	FolloweeIDs(followerID uint) ([]uint, error)
	ListFollowing(userID uint) ([]*model.Follow, error)
	ListFollowers(userID uint) ([]*model.Follow, error)
	CountFollowing(userID uint) (int64, error)
	CountFollowers(userID uint) (int64, error)
}

type followRepository struct{ db *gorm.DB }

func NewFollowRepository(db *gorm.DB) FollowRepository { return &followRepository{db: db} }

func (r *followRepository) Follow(followerID, followeeID uint) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.Follow{FollowerID: followerID, FolloweeID: followeeID}).Error
}

func (r *followRepository) Unfollow(followerID, followeeID uint) error {
	return r.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&model.Follow{}).Error
}

func (r *followRepository) IsFollowing(followerID, followeeID uint) (bool, error) {
	var n int64
	err := r.db.Model(&model.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Count(&n).Error
	return n > 0, err
}

func (r *followRepository) FolloweeIDs(followerID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Follow{}).
		Where("follower_id = ?", followerID).Pluck("followee_id", &ids).Error
	return ids, err
}

func (r *followRepository) ListFollowing(userID uint) ([]*model.Follow, error) {
	var out []*model.Follow
	err := r.db.Preload("Followee").Where("follower_id = ?", userID).
		Order("id DESC").Find(&out).Error
	return out, err
}

func (r *followRepository) ListFollowers(userID uint) ([]*model.Follow, error) {
	var out []*model.Follow
	err := r.db.Preload("Follower").Where("followee_id = ?", userID).
		Order("id DESC").Find(&out).Error
	return out, err
}

func (r *followRepository) CountFollowing(userID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Follow{}).Where("follower_id = ?", userID).Count(&n).Error
	return n, err
}

func (r *followRepository) CountFollowers(userID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Follow{}).Where("followee_id = ?", userID).Count(&n).Error
	return n, err
}
```

Note: `ListFollowing`/`ListFollowers` preload `Followee`/`Follower` — add those association fields to the `Follow` model if you want them populated:
```go
	Follower User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee User `gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
```
Add these two fields to `model.Follow` in `internal/model/feed.go` now (amend Task 1's struct).

- [ ] **Step 4: Add the testutil mock**

Append `MockFollowRepository` to `internal/testutil/mocks.go` (mirror the existing mock shape — one `Fn` field per method, `var _ repository.FollowRepository = (*MockFollowRepository)(nil)`).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/repository/ -run TestFollowRepository -v && go build ./...`
Expected: PASS and build OK. (Relax sqlmock regex if GORM's emitted SQL differs — keep the operation assertion.)

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_follow_repository.go internal/repository/feed_follow_repository_test.go internal/testutil/mocks.go internal/model/feed.go
git commit -m "feat(feed): add FollowRepository + mock"
```

---

### Task 3: FeedPostRepository (CRUD + timeline/profile cursor + cascade)

**Files:**
- Create: `internal/repository/feed_post_repository.go`
- Modify: `internal/testutil/mocks.go`
- Test: `internal/repository/feed_post_repository_test.go`

**Interfaces:**
- Produces:
```go
type FeedPostRepository interface {
	Create(p *model.FeedPost) error
	GetByID(id uint) (*model.FeedPost, error)             // preload Author, Attachments.File
	Update(p *model.FeedPost) error
	AttachFiles(postID uint, fileIDs []uint) error        // bulk create FeedPostAttachment
	TimelineCursor(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error) // WHERE author_id IN, newest-first
	ByAuthorCursor(authorID, before uint, limit int) ([]*model.FeedPost, error)
	DeleteCascade(postID uint) error                      // tx: comments + likes + attachments + post
}
func NewFeedPostRepository(db *gorm.DB) FeedPostRepository
```
- Produces mock: `testutil.MockFeedPostRepository`.

- [ ] **Step 1: Write the failing test (cascade + timeline)**

Create `internal/repository/feed_post_repository_test.go`:

```go
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
	// preload Author + Attachments queries follow; expectations may need relaxing (see step 5)

	repo := repository.NewFeedPostRepository(db)
	posts, err := repo.TimelineCursor([]uint{2, 3}, 0, 20)
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, uint(9), posts[0].ID) // newest-first, NOT reversed
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestFeedPostRepository -v`
Expected: FAIL — `NewFeedPostRepository` undefined.

- [ ] **Step 3: Implement**

Create `internal/repository/feed_post_repository.go`:

```go
package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

type FeedPostRepository interface {
	Create(p *model.FeedPost) error
	GetByID(id uint) (*model.FeedPost, error)
	Update(p *model.FeedPost) error
	AttachFiles(postID uint, fileIDs []uint) error
	TimelineCursor(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error)
	ByAuthorCursor(authorID, before uint, limit int) ([]*model.FeedPost, error)
	DeleteCascade(postID uint) error
}

type feedPostRepository struct{ db *gorm.DB }

func NewFeedPostRepository(db *gorm.DB) FeedPostRepository { return &feedPostRepository{db: db} }

func (r *feedPostRepository) Create(p *model.FeedPost) error { return r.db.Create(p).Error }

func (r *feedPostRepository) GetByID(id uint) (*model.FeedPost, error) {
	var p model.FeedPost
	err := r.db.Preload("Author").Preload("Attachments.File").First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *feedPostRepository) Update(p *model.FeedPost) error { return r.db.Save(p).Error }

func (r *feedPostRepository) AttachFiles(postID uint, fileIDs []uint) error {
	if len(fileIDs) == 0 {
		return nil
	}
	rows := make([]model.FeedPostAttachment, len(fileIDs))
	for i, fid := range fileIDs {
		rows[i] = model.FeedPostAttachment{PostID: postID, FileID: fid}
	}
	return r.db.Create(&rows).Error
}

func (r *feedPostRepository) TimelineCursor(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	if len(authorIDs) == 0 {
		return posts, nil
	}
	q := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id IN ?", authorIDs).
		Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	return posts, q.Find(&posts).Error // timeline stays newest-first, no reverse
}

func (r *feedPostRepository) ByAuthorCursor(authorID, before uint, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	q := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id = ?", authorID).
		Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	return posts, q.Find(&posts).Error
}

func (r *feedPostRepository) DeleteCascade(postID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedPostLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.FeedPostAttachment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.FeedPost{}, postID).Error
	})
}
```

- [ ] **Step 4: Add the testutil mock**

Append `MockFeedPostRepository` to `internal/testutil/mocks.go` (one `Fn` per method + interface assertion).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/repository/ -run TestFeedPostRepository -v && go build ./...`
Expected: PASS and build OK. If preload sub-queries make sqlmock strict-matching fail on `TimelineCursor`, relax to assert only the main SELECT + ordering (the cascade ordering test is the load-bearing one).

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_post_repository.go internal/repository/feed_post_repository_test.go internal/testutil/mocks.go
git commit -m "feat(feed): add FeedPostRepository (timeline/profile cursor + cascade)"
```

---

### Task 4: FeedPostLikeRepository + FeedCommentRepository + testutil mocks

**Files:**
- Create: `internal/repository/feed_like_repository.go`, `internal/repository/feed_comment_repository.go`
- Modify: `internal/testutil/mocks.go`
- Test: `internal/repository/feed_like_repository_test.go`, `internal/repository/feed_comment_repository_test.go`

**Interfaces:**
- Produces:
```go
type FeedPostLikeRepository interface {
	Create(like *model.FeedPostLike) error                 // idempotent
	Delete(postID, userID uint) error
	CountByPostID(postID uint) (int64, error)
	CountByPostIDs(ids []uint) (map[uint]int64, error)
	LikedPostIDs(userID uint, ids []uint) (map[uint]bool, error)
}
type FeedCommentRepository interface {
	Create(c *model.FeedComment) error
	GetByID(id uint) (*model.FeedComment, error)           // preload Author
	Update(c *model.FeedComment) error
	Delete(id uint) error
	ByPostCursor(postID, before uint, limit int) ([]*model.FeedComment, error) // chronological
	CountByPostIDs(ids []uint) (map[uint]int64, error)
}
func NewFeedPostLikeRepository(db *gorm.DB) FeedPostLikeRepository
func NewFeedCommentRepository(db *gorm.DB) FeedCommentRepository
```
- Produces mocks: `testutil.MockFeedPostLikeRepository`, `testutil.MockFeedCommentRepository`.

- [ ] **Step 1: Write the failing tests**

Create both test files. Like repo (mirror the wall's `MessageLikeRepository` tests — idempotent Create is a Query with `RETURNING id`, count is a `SELECT count(*)`):

```go
func TestFeedPostLikeRepository_Create_Idempotent(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "feed_post_likes"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	repo := repository.NewFeedPostLikeRepository(db)
	require.NoError(t, repo.Create(&model.FeedPostLike{PostID: 7, UserID: 5}))
	require.NoError(t, mock.ExpectationsWereMet())
}
```
Comment repo (mirror `GetByChannelIDCursor` DESC-then-reverse):
```go
func TestFeedCommentRepository_ByPostCursor_Chronological(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()
	mock.ExpectQuery(`SELECT \* FROM "feed_comments" WHERE post_id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "content"}).
			AddRow(9, 7, "b").AddRow(8, 7, "a")) // fetched DESC
	repo := repository.NewFeedCommentRepository(db)
	cs, err := repo.ByPostCursor(7, 0, 50)
	require.NoError(t, err)
	assert.Equal(t, uint(8), cs[0].ID) // reversed to chronological
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/repository/ -run "TestFeedPostLikeRepository|TestFeedCommentRepository" -v`
Expected: FAIL — constructors undefined.

- [ ] **Step 3: Implement both repositories**

`internal/repository/feed_like_repository.go` — copy the structure of `message_like_repository.go` verbatim, renaming: table `feed_post_likes`, model `model.FeedPostLike`, filter column `post_id` (not `message_id`), methods `CountByPostID`/`CountByPostIDs`/`LikedPostIDs`. Idempotent `Create` via `clause.OnConflict{DoNothing: true}`.

`internal/repository/feed_comment_repository.go`:
```go
package repository

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

type FeedCommentRepository interface {
	Create(c *model.FeedComment) error
	GetByID(id uint) (*model.FeedComment, error)
	Update(c *model.FeedComment) error
	Delete(id uint) error
	ByPostCursor(postID, before uint, limit int) ([]*model.FeedComment, error)
	CountByPostIDs(ids []uint) (map[uint]int64, error)
}

type feedCommentRepository struct{ db *gorm.DB }

func NewFeedCommentRepository(db *gorm.DB) FeedCommentRepository { return &feedCommentRepository{db: db} }

func (r *feedCommentRepository) Create(c *model.FeedComment) error { return r.db.Create(c).Error }

func (r *feedCommentRepository) GetByID(id uint) (*model.FeedComment, error) {
	var c model.FeedComment
	if err := r.db.Preload("Author").First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *feedCommentRepository) Update(c *model.FeedComment) error { return r.db.Save(c).Error }

func (r *feedCommentRepository) Delete(id uint) error {
	return r.db.Delete(&model.FeedComment{}, id).Error
}

func (r *feedCommentRepository) ByPostCursor(postID, before uint, limit int) ([]*model.FeedComment, error) {
	var cs []*model.FeedComment
	q := r.db.Preload("Author").Where("post_id = ?", postID).Order("id DESC").Limit(limit)
	if before > 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Find(&cs).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(cs)-1; i < j; i, j = i+1, j-1 {
		cs[i], cs[j] = cs[j], cs[i]
	}
	return cs, nil
}

func (r *feedCommentRepository) CountByPostIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		PostID uint
		Cnt    int64
	}
	var rows []row
	err := r.db.Model(&model.FeedComment{}).
		Select("post_id, COUNT(*) AS cnt").
		Where("post_id IN ?", ids).Group("post_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.PostID] = x.Cnt
	}
	return out, nil
}
```

- [ ] **Step 4: Add the testutil mocks**

Append `MockFeedPostLikeRepository` and `MockFeedCommentRepository` to `internal/testutil/mocks.go` (one `Fn` per method + interface assertions).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/repository/ -run "TestFeedPostLikeRepository|TestFeedCommentRepository" -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_like_repository.go internal/repository/feed_comment_repository.go internal/repository/feed_like_repository_test.go internal/repository/feed_comment_repository_test.go internal/testutil/mocks.go
git commit -m "feat(feed): add FeedPostLike and FeedComment repositories + mocks"
```

---

### Task 5: Follow service (follow/unfollow/suggestions/lists)

**Files:**
- Create: `internal/service/feed_service.go` (start the service here; struct + constructor + follow methods)
- Modify: `internal/testutil/mocks.go` (add `MockFeedService` only if a handler test needs it later — otherwise skip)
- Test: `internal/service/feed_service_test.go`

**Interfaces:**
- Consumes: `FollowRepository` (Task 2), `repository.FriendshipRepository` (existing, `ListFriends(userID) ([]*model.Friendship, error)`), `repository.UserRepository` (existing).
- Produces on `FeedService`:
```go
type FollowListResponse struct {
	Users []*model.User `json:"users"`
	Count int64         `json:"count"`
}
Follow(followerID, followeeID uint) error         // errors on self-follow
Unfollow(followerID, followeeID uint) error
Suggestions(userID uint) ([]*model.User, error)    // friends not yet followed, excluding self
ListFollowing(userID uint) (*FollowListResponse, error)
ListFollowers(userID uint) (*FollowListResponse, error)
```

- [ ] **Step 1: Write the failing tests**

Create `internal/service/feed_service_test.go`:

```go
func TestFeedService_Follow_RejectsSelf(t *testing.T) {
	svc := service.NewFeedService(
		&testutil.MockFollowRepository{}, nil, nil, nil, nil,
	)
	err := svc.Follow(5, 5)
	require.Error(t, err)
}

func TestFeedService_Suggestions_ExcludesAlreadyFollowed(t *testing.T) {
	follow := &testutil.MockFollowRepository{
		FolloweeIDsFn: func(uid uint) ([]uint, error) { return []uint{2}, nil }, // already follows 2
	}
	friends := &testutil.MockFriendshipRepository{
		ListFriendsFn: func(uid uint) ([]*model.Friendship, error) {
			// friends: users 2 and 3 (adapt to the real Friendship shape: RequesterID/AddresseeID + preloaded Users)
			return []*model.Friendship{
				{RequesterID: 5, AddresseeID: 2, Addressee: model.User{ID: 2, Username: "b"}},
				{RequesterID: 3, AddresseeID: 5, Requester: model.User{ID: 3, Username: "c"}},
			}, nil
		},
	}
	svc := service.NewFeedService(follow, nil, nil, nil, friends)
	sugg, err := svc.Suggestions(5)
	require.NoError(t, err)
	// user 2 already followed → only user 3 suggested
	require.Len(t, sugg, 1)
	assert.Equal(t, uint(3), sugg[0].ID)
}
```

Before implementing, INSPECT the real `model.Friendship` struct and `FriendshipRepository.ListFriends` return shape (fields for the two sides + whether Users are preloaded). Adapt the test's friend extraction to reality; the assertion (exclude already-followed + self, resolve to the "other" user) stays.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run TestFeedService_ -v`
Expected: FAIL — `NewFeedService` undefined.

- [ ] **Step 3: Implement follow service**

Create `internal/service/feed_service.go`. Constructor signature (fixed for the whole module; later tasks add post/comment methods to the SAME service):

```go
package service

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrFeedPostNotFound = errors.New("feed post not found")
	ErrNotFeedOwner     = errors.New("not the owner")
)

type FollowListResponse struct {
	Users []*model.User `json:"users"`
	Count int64         `json:"count"`
}

type FeedService interface {
	Follow(followerID, followeeID uint) error
	Unfollow(followerID, followeeID uint) error
	Suggestions(userID uint) ([]*model.User, error)
	ListFollowing(userID uint) (*FollowListResponse, error)
	ListFollowers(userID uint) (*FollowListResponse, error)
	// post/comment methods added in Tasks 6-7
}

type feedService struct {
	followRepo  repository.FollowRepository
	postRepo    repository.FeedPostRepository
	likeRepo    repository.FeedPostLikeRepository
	commentRepo repository.FeedCommentRepository
	friendRepo  repository.FriendshipRepository
}

func NewFeedService(
	followRepo repository.FollowRepository,
	postRepo repository.FeedPostRepository,
	likeRepo repository.FeedPostLikeRepository,
	commentRepo repository.FeedCommentRepository,
	friendRepo repository.FriendshipRepository,
) FeedService {
	return &feedService{followRepo, postRepo, likeRepo, commentRepo, friendRepo}
}

func (s *feedService) Follow(followerID, followeeID uint) error {
	if followerID == followeeID {
		return ErrCannotFollowSelf
	}
	return s.followRepo.Follow(followerID, followeeID)
}

func (s *feedService) Unfollow(followerID, followeeID uint) error {
	return s.followRepo.Unfollow(followerID, followeeID)
}

func (s *feedService) Suggestions(userID uint) ([]*model.User, error) {
	friends, err := s.friendRepo.ListFriends(userID)
	if err != nil {
		return nil, err
	}
	followeeIDs, err := s.followRepo.FolloweeIDs(userID)
	if err != nil {
		return nil, err
	}
	followed := make(map[uint]bool, len(followeeIDs))
	for _, id := range followeeIDs {
		followed[id] = true
	}
	var out []*model.User
	for _, f := range friends {
		other := friendCounterpart(f, userID) // returns the *model.User that is NOT userID
		if other == nil || other.ID == userID || followed[other.ID] {
			continue
		}
		out = append(out, other)
	}
	return out, nil
}

func (s *feedService) ListFollowing(userID uint) (*FollowListResponse, error) {
	rows, err := s.followRepo.ListFollowing(userID)
	if err != nil {
		return nil, err
	}
	cnt, _ := s.followRepo.CountFollowing(userID)
	users := make([]*model.User, 0, len(rows))
	for _, r := range rows {
		u := r.Followee
		users = append(users, &u)
	}
	return &FollowListResponse{Users: users, Count: cnt}, nil
}

func (s *feedService) ListFollowers(userID uint) (*FollowListResponse, error) {
	rows, err := s.followRepo.ListFollowers(userID)
	if err != nil {
		return nil, err
	}
	cnt, _ := s.followRepo.CountFollowers(userID)
	users := make([]*model.User, 0, len(rows))
	for _, r := range rows {
		u := r.Follower
		users = append(users, &u)
	}
	return &FollowListResponse{Users: users, Count: cnt}, nil
}
```

Add a helper `friendCounterpart(f *model.Friendship, self uint) *model.User` in the same file that returns the preloaded User on the opposite side of `self` — WRITE IT AGAINST the real `Friendship` field names you inspected in Step 1 (e.g. if fields are `RequesterID/Requester` and `AddresseeID/Addressee`, return `&f.Requester` when `f.RequesterID != self` else `&f.Addressee`).

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/service/ -run TestFeedService_ -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/feed_service.go internal/service/feed_service_test.go
git commit -m "feat(feed): follow service (follow/unfollow/suggestions/lists)"
```

---

### Task 6: Feed post service — create / timeline / profile + enrichment

**Files:**
- Modify: `internal/service/feed_service.go` (add DTOs + methods to the interface and impl)
- Modify: `internal/service/feed_service_test.go`

**Interfaces:**
- Consumes: `FeedPostRepository`, `FeedPostLikeRepository`, `FeedCommentRepository` (Tasks 3-4).
- Produces on `FeedService`:
```go
type FeedPostResponse struct {
	*model.FeedPost
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	LikedByMe    bool  `json:"liked_by_me"`
}
type TimelineResponse struct {
	Posts   []*FeedPostResponse `json:"posts"`
	HasMore bool                `json:"has_more"`
}
CreatePost(authorID uint, content string, fileIDs []uint) (*FeedPostResponse, error)
Timeline(userID uint, before uint, limit int) (*TimelineResponse, error)
ProfilePosts(targetID, viewerID uint, before uint, limit int) (*TimelineResponse, error)
```

- [ ] **Step 1: Write the failing test (timeline enrichment + includes self)**

Add to `internal/service/feed_service_test.go`:

```go
func TestFeedService_Timeline_IncludesSelfAndEnriches(t *testing.T) {
	var gotAuthorIDs []uint
	post := &model.FeedPost{ID: 9, AuthorID: 2}
	follow := &testutil.MockFollowRepository{
		FolloweeIDsFn: func(uid uint) ([]uint, error) { return []uint{2, 3}, nil },
	}
	posts := &testutil.MockFeedPostRepository{
		TimelineCursorFn: func(ids []uint, before uint, limit int) ([]*model.FeedPost, error) {
			gotAuthorIDs = ids
			return []*model.FeedPost{post}, nil
		},
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{9: 4}, nil },
		LikedPostIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{9: true}, nil },
	}
	comments := &testutil.MockFeedCommentRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{9: 2}, nil },
	}
	svc := service.NewFeedService(follow, posts, likes, comments, nil)

	resp, err := svc.Timeline(5, 0, 20)
	require.NoError(t, err)
	assert.Contains(t, gotAuthorIDs, uint(5)) // self included
	require.Len(t, resp.Posts, 1)
	assert.Equal(t, int64(4), resp.Posts[0].LikeCount)
	assert.Equal(t, int64(2), resp.Posts[0].CommentCount)
	assert.True(t, resp.Posts[0].LikedByMe)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestFeedService_Timeline -v`
Expected: FAIL — `Timeline` undefined.

- [ ] **Step 3: Implement**

Add the DTOs and methods to `internal/service/feed_service.go` (and the interface). Add a private `enrich(posts, viewerID)` helper reused by Timeline and ProfilePosts:

```go
func (s *feedService) CreatePost(authorID uint, content string, fileIDs []uint) (*FeedPostResponse, error) {
	if content == "" && len(fileIDs) == 0 {
		return nil, errors.New("empty post")
	}
	p := &model.FeedPost{AuthorID: authorID, Content: content}
	if err := s.postRepo.Create(p); err != nil {
		return nil, err
	}
	if err := s.postRepo.AttachFiles(p.ID, fileIDs); err != nil {
		return nil, err
	}
	full, err := s.postRepo.GetByID(p.ID)
	if err != nil {
		return nil, err
	}
	out := s.enrich([]*model.FeedPost{full}, authorID)
	return out[0], nil
}

func (s *feedService) Timeline(userID uint, before uint, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	ids, err := s.followRepo.FolloweeIDs(userID)
	if err != nil {
		return nil, err
	}
	ids = append(ids, userID) // include self
	posts, err := s.postRepo.TimelineCursor(ids, before, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	return &TimelineResponse{Posts: s.enrich(posts, userID), HasMore: hasMore}, nil
}

func (s *feedService) ProfilePosts(targetID, viewerID uint, before uint, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	posts, err := s.postRepo.ByAuthorCursor(targetID, before, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	return &TimelineResponse{Posts: s.enrich(posts, viewerID), HasMore: hasMore}, nil
}

func (s *feedService) enrich(posts []*model.FeedPost, viewerID uint) []*FeedPostResponse {
	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	var likeCounts, commentCounts map[uint]int64
	var liked map[uint]bool
	if s.likeRepo != nil {
		likeCounts, _ = s.likeRepo.CountByPostIDs(ids)
		liked, _ = s.likeRepo.LikedPostIDs(viewerID, ids)
	}
	if s.commentRepo != nil {
		commentCounts, _ = s.commentRepo.CountByPostIDs(ids)
	}
	out := make([]*FeedPostResponse, len(posts))
	for i, p := range posts {
		out[i] = &FeedPostResponse{
			FeedPost:     p,
			LikeCount:    likeCounts[p.ID],
			CommentCount: commentCounts[p.ID],
			LikedByMe:    liked[p.ID],
		}
	}
	return out
}
```

- [ ] **Step 4: Run test + build**

Run: `go test ./internal/service/ -run TestFeedService_Timeline -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/feed_service.go internal/service/feed_service_test.go
git commit -m "feat(feed): post create + timeline/profile with enrichment"
```

---

### Task 7: Feed post service — like/unlike, comments, edit/delete, cascade

**Files:**
- Modify: `internal/service/feed_service.go`
- Modify: `internal/service/feed_service_test.go`

**Interfaces:**
- Produces on `FeedService`:
```go
type CommentListResponse struct {
	Comments []*model.FeedComment `json:"comments"`
	HasMore  bool                 `json:"has_more"`
}
UpdatePost(postID, userID uint, content string) (*FeedPostResponse, error)
DeletePost(postID, userID uint) error
LikePost(postID, userID uint) (int64, error)
UnlikePost(postID, userID uint) (int64, error)
ListComments(postID uint, before uint, limit int) (*CommentListResponse, error)
AddComment(postID, authorID uint, content string) (*model.FeedComment, error)
UpdateComment(commentID, userID uint, content string) (*model.FeedComment, error)
DeleteComment(commentID, userID uint) error
```

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/feed_service_test.go`:

```go
func TestFeedService_DeletePost_OwnerCascades(t *testing.T) {
	cascaded := false
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn:       func(id uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7, AuthorID: 5}, nil },
		DeleteCascadeFn: func(id uint) error { cascaded = true; return nil },
	}
	svc := service.NewFeedService(nil, posts, nil, nil, nil)
	require.NoError(t, svc.DeletePost(7, 5))
	assert.True(t, cascaded)
}

func TestFeedService_DeletePost_NonOwnerRejected(t *testing.T) {
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn: func(id uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7, AuthorID: 99}, nil },
	}
	svc := service.NewFeedService(nil, posts, nil, nil, nil)
	require.ErrorIs(t, svc.DeletePost(7, 5), service.ErrNotFeedOwner)
}

func TestFeedService_LikePost_ReturnsCount(t *testing.T) {
	posts := &testutil.MockFeedPostRepository{GetByIDFn: func(id uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7}, nil }}
	likes := &testutil.MockFeedPostLikeRepository{
		CreateFn:           func(l *model.FeedPostLike) error { return nil },
		CountByPostIDFn:    func(id uint) (int64, error) { return 3, nil },
	}
	svc := service.NewFeedService(nil, posts, likes, nil, nil)
	n, err := svc.LikePost(7, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run "TestFeedService_DeletePost|TestFeedService_LikePost" -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement**

Add to `internal/service/feed_service.go` (and interface):

```go
type CommentListResponse struct {
	Comments []*model.FeedComment `json:"comments"`
	HasMore  bool                 `json:"has_more"`
}

func (s *feedService) getOwnedPost(postID, userID uint) (*model.FeedPost, error) {
	p, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, ErrFeedPostNotFound
	}
	if p.AuthorID != userID {
		return nil, ErrNotFeedOwner
	}
	return p, nil
}

func (s *feedService) UpdatePost(postID, userID uint, content string) (*FeedPostResponse, error) {
	p, err := s.getOwnedPost(postID, userID)
	if err != nil {
		return nil, err
	}
	p.Content = content
	p.IsEdited = true
	if err := s.postRepo.Update(p); err != nil {
		return nil, err
	}
	full, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, err
	}
	return s.enrich([]*model.FeedPost{full}, userID)[0], nil
}

func (s *feedService) DeletePost(postID, userID uint) error {
	if _, err := s.getOwnedPost(postID, userID); err != nil {
		return err
	}
	return s.postRepo.DeleteCascade(postID)
}

func (s *feedService) LikePost(postID, userID uint) (int64, error) {
	if _, err := s.postRepo.GetByID(postID); err != nil {
		return 0, ErrFeedPostNotFound
	}
	if err := s.likeRepo.Create(&model.FeedPostLike{PostID: postID, UserID: userID}); err != nil {
		return 0, err
	}
	return s.likeRepo.CountByPostID(postID)
}

func (s *feedService) UnlikePost(postID, userID uint) (int64, error) {
	if err := s.likeRepo.Delete(postID, userID); err != nil {
		return 0, err
	}
	return s.likeRepo.CountByPostID(postID)
}

func (s *feedService) ListComments(postID uint, before uint, limit int) (*CommentListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	cs, err := s.commentRepo.ByPostCursor(postID, before, limit+1)
	if err != nil {
		return nil, err
	}
	hasMore := false
	if len(cs) > limit {
		hasMore = true
		cs = cs[len(cs)-limit:]
	}
	return &CommentListResponse{Comments: cs, HasMore: hasMore}, nil
}

func (s *feedService) AddComment(postID, authorID uint, content string) (*model.FeedComment, error) {
	if content == "" {
		return nil, errors.New("empty comment")
	}
	if _, err := s.postRepo.GetByID(postID); err != nil {
		return nil, ErrFeedPostNotFound
	}
	c := &model.FeedComment{PostID: postID, AuthorID: authorID, Content: content}
	if err := s.commentRepo.Create(c); err != nil {
		return nil, err
	}
	return s.commentRepo.GetByID(c.ID)
}

func (s *feedService) UpdateComment(commentID, userID uint, content string) (*model.FeedComment, error) {
	c, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return nil, errors.New("comment not found")
	}
	if c.AuthorID != userID {
		return nil, ErrNotFeedOwner
	}
	c.Content = content
	c.IsEdited = true
	if err := s.commentRepo.Update(c); err != nil {
		return nil, err
	}
	return s.commentRepo.GetByID(commentID)
}

func (s *feedService) DeleteComment(commentID, userID uint) error {
	c, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return errors.New("comment not found")
	}
	if c.AuthorID != userID {
		return ErrNotFeedOwner
	}
	return s.commentRepo.Delete(commentID)
}
```

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/service/ -run TestFeedService_ -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/feed_service.go internal/service/feed_service_test.go
git commit -m "feat(feed): like/unlike, comments CRUD, owner-guarded edit/delete + cascade"
```

---

### Task 8: HTTP handlers + routes + wiring

**Files:**
- Create: `internal/handler/feed_handler.go`
- Modify: `internal/server/server.go` (routes + service/handler construction; grep how `messageHandler`/`messageService` are built and mirror)
- Test: manual via `make check` compile + the E2E in the finish step

**Interfaces:**
- Consumes: `FeedService` (Tasks 5-7).
- Produces HTTP endpoints under `/api/v1/feed` exactly as the spec §2 lists.

- [ ] **Step 1: Create the handler**

Create `internal/handler/feed_handler.go`. Mirror `message_handler.go` idioms for param/query/userID parsing (`c.Get("user_id")` → `.(uint)`, `strconv.ParseUint(c.Param("id"), ...)`, the `before`/`limit` query parse — reuse the same helper if one is shared, else inline). One handler method per endpoint:

```go
type FeedHandler struct{ feedService service.FeedService }

func NewFeedHandler(feedService service.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}
```
Methods: `Follow`, `Unfollow`, `Suggestions`, `ListFollowing`, `ListFollowers`, `Timeline`, `CreatePost`, `ProfilePosts`, `UpdatePost`, `DeletePost`, `LikePost`, `UnlikePost`, `ListComments`, `AddComment`, `UpdateComment`, `DeleteComment`. Each: parse ids + userID from context, bind JSON body where needed (create/update take `{content, file_ids}` / `{content}`), call the service, return `c.JSON`. Error shape `gin.H{"error": err.Error()}` (400 for domain errors, 404 for not-found sentinels — mirror `message_handler.go`'s `switch errors.Is`).

Request bodies:
```go
type createFeedPostBody struct { Content string `json:"content"`; FileIDs []uint `json:"file_ids"` }
type feedContentBody   struct { Content string `json:"content" binding:"required"` }
```

- [ ] **Step 2: Wire construction + routes**

In `internal/server/server.go`, where repos/services/handlers are constructed, add (using the shared `db`, mirroring existing wiring):
```go
feedService := service.NewFeedService(
	repository.NewFollowRepository(db),
	repository.NewFeedPostRepository(db),
	repository.NewFeedPostLikeRepository(db),
	repository.NewFeedCommentRepository(db),
	friendshipRepo, // reuse the FriendshipRepository already constructed for friends feature
)
feedHandler := handler.NewFeedHandler(feedService)
```
Add `feedHandler *handler.FeedHandler` to the Server struct if handlers are stored there (match the pattern). Then register routes inside the `protected` group:
```go
feed := protected.Group("/feed")
{
	feed.GET("/suggestions", s.feedHandler.Suggestions)
	feed.POST("/follows/:userId", s.feedHandler.Follow)
	feed.DELETE("/follows/:userId", s.feedHandler.Unfollow)
	feed.GET("/users/:userId/following", s.feedHandler.ListFollowing)
	feed.GET("/users/:userId/followers", s.feedHandler.ListFollowers)
	feed.GET("/users/:userId/posts", s.feedHandler.ProfilePosts)
	feed.GET("/timeline", s.feedHandler.Timeline)
	feed.POST("/posts", s.feedHandler.CreatePost)
	feed.PUT("/posts/:id", s.feedHandler.UpdatePost)
	feed.DELETE("/posts/:id", s.feedHandler.DeletePost)
	feed.PUT("/posts/:id/like", s.feedHandler.LikePost)
	feed.DELETE("/posts/:id/like", s.feedHandler.UnlikePost)
	feed.GET("/posts/:id/comments", s.feedHandler.ListComments)
	feed.POST("/posts/:id/comments", s.feedHandler.AddComment)
	feed.PUT("/comments/:id", s.feedHandler.UpdateComment)
	feed.DELETE("/comments/:id", s.feedHandler.DeleteComment)
}
```
Note the gin route-conflict rule: `/users/:userId/...` and `/posts/...` and `/timeline` are distinct top segments — fine. `:userId` vs `:id` param names differ across groups but never within the same path — fine.

- [ ] **Step 3: Full check**

Run: `make check` (or non-race fallback). Expected: build + lint + tests pass; swagger may regenerate.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/feed_handler.go internal/server/server.go
git commit -m "feat(feed): HTTP handlers + /api/v1/feed routes + wiring"
```

---

### Task 9: Frontend — API client + i18n

**Files:**
- Modify: `web/src/api/index.js` (EP entries + methods)
- Modify: `web/src/i18n/locales/{en,zh,zh-tw,ja}.js` (feed UI keys)
- Verify: `npm --prefix web run build`

**Interfaces:**
- Consumes: Task 8 endpoints.
- Produces: `api.feed*` methods used by Task 10-11 components.

- [ ] **Step 1: Add EP entries + methods**

In `web/src/api/index.js` `EP` map:
```js
    FEED_TIMELINE: '/api/v1/feed/timeline',
    FEED_POSTS: '/api/v1/feed/posts',
    FEED_POST: (id) => `/api/v1/feed/posts/${id}`,
    FEED_POST_LIKE: (id) => `/api/v1/feed/posts/${id}/like`,
    FEED_POST_COMMENTS: (id) => `/api/v1/feed/posts/${id}/comments`,
    FEED_COMMENT: (id) => `/api/v1/feed/comments/${id}`,
    FEED_SUGGESTIONS: '/api/v1/feed/suggestions',
    FEED_FOLLOW: (userId) => `/api/v1/feed/follows/${userId}`,
    FEED_USER_POSTS: (userId) => `/api/v1/feed/users/${userId}/posts`,
    FEED_FOLLOWING: (userId) => `/api/v1/feed/users/${userId}/following`,
    FEED_FOLLOWERS: (userId) => `/api/v1/feed/users/${userId}/followers`,
```
Methods (match the class's existing style with the `get/post/put/del` helpers):
```js
    getTimeline(limit = 20, before = null) {
      const q = new URLSearchParams({ limit }); if (before) q.set('before', before)
      return this.get(`${EP.FEED_TIMELINE}?${q}`)
    },
    createFeedPost(content, fileIds = []) { return this.post(EP.FEED_POSTS, { content, file_ids: fileIds }) },
    updateFeedPost(id, content) { return this.put(EP.FEED_POST(id), { content }) },
    deleteFeedPost(id) { return this.del(EP.FEED_POST(id)) },
    likeFeedPost(id) { return this.put(EP.FEED_POST_LIKE(id), {}) },
    unlikeFeedPost(id) { return this.del(EP.FEED_POST_LIKE(id)) },
    getFeedComments(id, limit = 50, before = null) {
      const q = new URLSearchParams({ limit }); if (before) q.set('before', before)
      return this.get(`${EP.FEED_POST_COMMENTS(id)}?${q}`)
    },
    addFeedComment(id, content) { return this.post(EP.FEED_POST_COMMENTS(id), { content }) },
    getUserPosts(userId, limit = 20, before = null) {
      const q = new URLSearchParams({ limit }); if (before) q.set('before', before)
      return this.get(`${EP.FEED_USER_POSTS(userId)}?${q}`)
    },
    getFollowSuggestions() { return this.get(EP.FEED_SUGGESTIONS) },
    follow(userId) { return this.put ? this.post(EP.FEED_FOLLOW(userId), {}) : null }, // POST per spec
    unfollow(userId) { return this.del(EP.FEED_FOLLOW(userId)) },
    getFollowing(userId) { return this.get(EP.FEED_FOLLOWING(userId)) },
    getFollowers(userId) { return this.get(EP.FEED_FOLLOWERS(userId)) },
```
(`follow` uses POST — simplify to `follow(userId) { return this.post(EP.FEED_FOLLOW(userId), {}) }`.)

- [ ] **Step 2: Add i18n keys**

The stub `FeedView.vue` uses `views.feed.*`. Add a `feed.*` UI section (compose placeholder, publish, like, comment, follow, unfollow, followers, following, suggestions title, empty states) to ALL FOUR locale files under `web/src/i18n/locales/`. Run `node web/scripts/check-i18n-keys.mjs` to confirm parity.

- [ ] **Step 3: Verify build**

Run: `npm --prefix web run build`
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add web/src/api/index.js web/src/i18n/locales/
git commit -m "feat(feed): frontend api client + i18n keys"
```

---

### Task 10: Frontend — FeedView + FeedComposer + FeedPostCard (timeline core)

**Files:**
- Rewrite: `web/src/views/FeedView.vue` (replace stub)
- Create: `web/src/components/feed/FeedComposer.vue`, `web/src/components/feed/FeedPostCard.vue`
- Verify: `npm --prefix web run build` + manual

**Interfaces:**
- Consumes: Task 9 API methods.

- [ ] **Step 1: Build the components**

- `FeedComposer.vue`: text area + image attach (reuse the same upload composable/flow the wall's compose used — grep how `FeedArea.vue`/`MessageInput.vue` collect `fileIds`) + a publish button; emits `posted(post)` after `api.createFeedPost`.
- `FeedPostCard.vue`: props `post` (a `FeedPostResponse`); renders author (reuse avatar/timestamp components), content, attachments (reuse Lightbox), a like button (`like_count`, toggles `api.likeFeedPost`/`unlikeFeedPost` optimistically via `liked_by_me`), and a comment toggle (`comment_count`) that lazy-loads `api.getFeedComments` and shows single-level comments + a comment box (`api.addFeedComment`). Own-post edit/delete via `api.updateFeedPost`/`deleteFeedPost`. Emit `deleted(id)` upward.
- `FeedView.vue`: replace the stub template. On mount call `api.getTimeline()` → `posts` ref; `FeedComposer` at top prepends new posts; infinite scroll up uses `has_more` + last post id as `before`. Wide-screen right column hosts `FeedFollowSuggestions` (Task 11). Follow the Kinetic Noir styling and reuse existing card/message CSS classes.

- [ ] **Step 2: Verify build**

Run: `npm --prefix web run build`
Expected: exit 0.

- [ ] **Step 3: Manual check**

Run the app (see `docs`/local-run notes), open `/feed`: compose a post, see it appear, like it, comment on it.
Expected: timeline core works.

- [ ] **Step 4: Commit**

```bash
git add web/src/views/FeedView.vue web/src/components/feed/
git commit -m "feat(feed): FeedView timeline + composer + post card"
```

---

### Task 11: Frontend — FeedProfile + FeedFollowSuggestions (follow flow)

**Files:**
- Create: `web/src/components/feed/FeedProfile.vue`, `web/src/components/feed/FeedFollowSuggestions.vue`
- Modify: `web/src/views/FeedView.vue` (mount suggestions in the right column; open profile on author click)
- Verify: `npm --prefix web run build` + manual

- [ ] **Step 1: Build the components**

- `FeedFollowSuggestions.vue`: on mount `api.getFollowSuggestions()` → list of friend users each with a Follow button (`api.follow(userId)`, then remove from list / mark followed). Empty state when none.
- `FeedProfile.vue`: props `userId`; loads `api.getUserPosts(userId)` (their timeline), `api.getFollowers`/`getFollowing` counts, and a Follow/Unfollow toggle (hide when it's the current user). Reuse `FeedPostCard` for the post list.
- `FeedView.vue`: render `FeedFollowSuggestions` in the wide-screen right column; clicking a post author (in `FeedPostCard`) opens `FeedProfile` (overlay/panel or a `?user=` state within the feed view — keep it lazy, no new router route needed).

- [ ] **Step 2: Verify build**

Run: `npm --prefix web run build`
Expected: exit 0.

- [ ] **Step 3: Manual check**

Open `/feed`: see friend suggestions, follow a user, confirm their posts now appear in your timeline; open a profile, follow/unfollow toggles and post list render.
Expected: full follow → timeline flow works.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/feed/ web/src/views/FeedView.vue
git commit -m "feat(feed): follow suggestions + profile view"
```

---

## Notes for the implementer

- **Order:** Tasks 1→8 backend, strictly ordered (each consumes the previous). Tasks 9-11 frontend, need Task 8 live.
- **`make check` after each backend task** (non-race fallback on this Windows box — no gcc).
- **sqlmock brittleness:** if GORM's emitted SQL doesn't match a test regex in Tasks 2-4, relax the regex/`WithArgs` while keeping the operation/ordering assertion. Behavioral guarantees live in the service tests (Tasks 5-7).
- **Boundary discipline:** never import or modify wall/chat tables. `Friendship`, `User`, `File` are read-only at the boundary.
- **Inspect the real `Friendship` shape** before writing `friendCounterpart` (Task 5) — field names must match.
- **Reuse, don't reinvent, on the frontend:** avatar, timestamp, attachment upload, and Lightbox components already exist (used by the wall's `FeedArea`/`MessageItem`). Grep and reuse them.
```
