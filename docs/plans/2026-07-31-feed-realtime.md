# Feed Realtime (pill + live counts) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the cross-community feed a live feel via lightweight fan-out ping: a "N new posts" pill (tap to load, no auto-inject) plus live like/comment counts on the timeline.

**Architecture:** Extends the `feed` module — no new tables. On post/like/comment writes, the feed service looks up the author's followers and pushes a small WebSocket event to each (via the existing `BroadcastToUser`). The FeedView subscribes to those events: `feed_new_post` increments a pill counter; `feed_post_like`/`feed_comment_count` update the matching loaded post's count. It is fan-out of a *signal*, not timeline materialization.

**Tech Stack:** Go + Gin + GORM/PostgreSQL; Vue 3; existing WebSocket hub (`BroadcastToUser`); testify + `testutil` mocks; go-sqlmock.

## Global Constraints

- Spec: `docs/specs/2026-07-31-feed-realtime-design.md` — authoritative.
- No new tables; no timeline materialization. Push signals only.
- No auto-inject of new posts into the timeline — only a pill the user taps.
- B1: live counts only; do NOT live-insert comments into an expanded thread. FeedProfile live updates are out of scope.
- Events + payloads (exact): `feed_new_post` → `{author_id}` to followers (NOT the author); `feed_post_like` → `{post_id, like_count}` to followers + author; `feed_comment_count` → `{post_id, comment_count}` to followers + author.
- Fan-out is synchronous (a single indexed `FollowerIDs` query + N non-blocking `BroadcastToUser` calls) — deterministic and testable; `ponytail: synchronous fan-out, move to a goroutine/queue only if follower counts get large`.
- Mirror existing idioms: `messageService.SetWebSocketManager` setter pattern; `service.WebSocketManager` interface (`internal/service/message_service.go:31`); `testutil` function-field mocks with interface assertions; FeedArea's `useWebSocket` onMessage/offMessage subscription pattern.
- After every backend task run: `make check` (fall back to `go build ./... && go vet ./... && go test ./...` if `-race`/mise shims break — no gcc on this Windows box).
- Frontend: no unit-test framework; verify with `npm --prefix web run build` + the E2E in the finish step.
- Commit after every task. Do NOT `git add -A`/`git add .` — there are unrelated uncommitted `web/src/components/learn/*.vue` changes; add only your files and leave those alone.

---

### Task 1: `FollowRepository.FollowerIDs` + mock

**Files:**
- Modify: `internal/repository/feed_follow_repository.go` (interface + impl)
- Modify: `internal/testutil/mocks.go` (extend `MockFollowRepository`)
- Test: `internal/repository/feed_follow_repository_test.go`

**Interfaces:**
- Produces: `FollowRepository.FollowerIDs(followeeID uint) ([]uint, error)` — follower ids of a user (for fan-out). Mock field `FollowerIDsFn`.

- [ ] **Step 1: Write the failing test**

Add to `internal/repository/feed_follow_repository_test.go`:

```go
func TestFollowRepository_FollowerIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT "follower_id" FROM "follows" WHERE followee_id = \$1`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"follower_id"}).AddRow(1).AddRow(3))

	repo := repository.NewFollowRepository(db)
	ids, err := repo.FollowerIDs(2)
	require.NoError(t, err)
	assert.Equal(t, []uint{1, 3}, ids)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestFollowRepository_FollowerIDs -v`
Expected: FAIL — `FollowerIDs` undefined.

- [ ] **Step 3: Implement**

Add to the `FollowRepository` interface (next to `FolloweeIDs`):
```go
	FollowerIDs(followeeID uint) ([]uint, error)
```
Impl (mirror `FolloweeIDs`):
```go
func (r *followRepository) FollowerIDs(followeeID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Follow{}).
		Where("followee_id = ?", followeeID).Pluck("follower_id", &ids).Error
	return ids, err
}
```

- [ ] **Step 4: Extend the mock**

In `internal/testutil/mocks.go`, add to `MockFollowRepository`:
```go
	FollowerIDsFn func(followeeID uint) ([]uint, error)
```
```go
func (m *MockFollowRepository) FollowerIDs(followeeID uint) ([]uint, error) {
	return m.FollowerIDsFn(followeeID)
}
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/repository/ -run TestFollowRepository_FollowerIDs -v && go build ./...`
Expected: PASS and build OK. (Relax the sqlmock regex if GORM's emitted SQL differs; keep the ordering assertion.)

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_follow_repository.go internal/testutil/mocks.go internal/repository/feed_follow_repository_test.go
git commit -m "feat(feed): FollowRepository.FollowerIDs for fan-out"
```

---

### Task 2: feed service WS fan-out (setter + helper + 3 events)

**Files:**
- Modify: `internal/service/feed_service.go` (interface, struct, setter, helper, CreatePost/LikePost/UnlikePost/AddComment)
- Modify: `internal/server/server.go` (wire `SetWebSocketManager`)
- Test: `internal/service/feed_service_test.go`

**Interfaces:**
- Consumes: `FollowRepository.FollowerIDs` (Task 1); `service.WebSocketManager` (existing, `BroadcastToUser(userID uint, msgType string, data any)`).
- Produces on `FeedService`: `SetWebSocketManager(m WebSocketManager)`; fan-out on CreatePost/LikePost/UnlikePost/AddComment.

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/feed_service_test.go`:

```go
func TestFeedService_CreatePost_BroadcastsNewPostToFollowersNotAuthor(t *testing.T) {
	created := &model.FeedPost{ID: 8, AuthorID: 5}
	posts := &testutil.MockFeedPostRepository{
		CreateFn:      func(p *model.FeedPost) error { p.ID = 8; return nil },
		AttachFilesFn: func(postID uint, ids []uint) error { return nil },
		GetByIDFn:     func(id uint) (*model.FeedPost, error) { return created, nil },
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
		LikedPostIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{}, nil },
	}
	comments := &testutil.MockFeedCommentRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
	}
	follow := &testutil.MockFollowRepository{
		FollowerIDsFn: func(followeeID uint) ([]uint, error) { return []uint{2, 3}, nil },
	}
	ws := &testutil.MockWebSocketManager{}
	svc := service.NewFeedService(follow, posts, likes, comments, nil)
	svc.SetWebSocketManager(ws)

	_, err := svc.CreatePost(5, "hi", nil)
	require.NoError(t, err)
	// followers 2 and 3 receive feed_new_post; author 5 does NOT
	got := map[uint]bool{}
	for _, b := range ws.UserBroadcasts {
		if b.MsgType == "feed_new_post" {
			got[b.ID] = true
		}
	}
	assert.True(t, got[2] && got[3])
	assert.False(t, got[5], "author must not receive feed_new_post")
}

func TestFeedService_LikePost_BroadcastsLikeCount(t *testing.T) {
	posts := &testutil.MockFeedPostRepository{
		GetByIDFn: func(id uint) (*model.FeedPost, error) { return &model.FeedPost{ID: 7, AuthorID: 5}, nil },
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CreateFn:        func(l *model.FeedPostLike) error { return nil },
		CountByPostIDFn: func(id uint) (int64, error) { return 3, nil },
	}
	follow := &testutil.MockFollowRepository{
		FollowerIDsFn: func(followeeID uint) ([]uint, error) { return []uint{2}, nil },
	}
	ws := &testutil.MockWebSocketManager{}
	svc := service.NewFeedService(follow, posts, likes, nil, nil)
	svc.SetWebSocketManager(ws)

	n, err := svc.LikePost(7, 9)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
	// follower 2 AND author 5 receive feed_post_like
	recipients := map[uint]bool{}
	for _, b := range ws.UserBroadcasts {
		if b.MsgType == "feed_post_like" {
			recipients[b.ID] = true
		}
	}
	assert.True(t, recipients[2] && recipients[5])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run "TestFeedService_CreatePost_Broadcasts|TestFeedService_LikePost_Broadcasts" -v`
Expected: FAIL — `SetWebSocketManager` undefined.

- [ ] **Step 3: Implement the setter + helper**

In `internal/service/feed_service.go`:
- Add field to `feedService` struct: `wsManager WebSocketManager`.
- Add to the `FeedService` interface: `SetWebSocketManager(m WebSocketManager)`.
- Add:
```go
func (s *feedService) SetWebSocketManager(m WebSocketManager) { s.wsManager = m }

// broadcastToFollowers pushes an event to every follower of authorID (and to the
// author too when includeAuthor). Fan-out of a signal only — not materialization.
// ponytail: synchronous fan-out; move to a goroutine/queue only if follower counts get large.
func (s *feedService) broadcastToFollowers(authorID uint, includeAuthor bool, event string, data any) {
	if s.wsManager == nil {
		return
	}
	ids, err := s.followRepo.FollowerIDs(authorID)
	if err != nil {
		return
	}
	for _, uid := range ids {
		s.wsManager.BroadcastToUser(uid, event, data)
	}
	if includeAuthor {
		s.wsManager.BroadcastToUser(authorID, event, data)
	}
}
```

- [ ] **Step 4: Wire the three events**

In `CreatePost`, after the post is created and enriched (right before `return`), add:
```go
	s.broadcastToFollowers(authorID, false, "feed_new_post", map[string]any{"author_id": authorID})
```

In `LikePost` and `UnlikePost`: capture the post's author. Change the leading `GetByID` so the post is kept, and after computing the new count broadcast before returning. For `LikePost`:
```go
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return 0, ErrFeedPostNotFound
	}
	if err := s.likeRepo.Create(&model.FeedPostLike{PostID: postID, UserID: userID}); err != nil {
		return 0, err
	}
	n, err := s.likeRepo.CountByPostID(postID)
	if err != nil {
		return 0, err
	}
	s.broadcastToFollowers(post.AuthorID, true, "feed_post_like", map[string]any{"post_id": postID, "like_count": n})
	return n, nil
```
For `UnlikePost`, it currently may not fetch the post — add a `GetByID(postID)` to get `post.AuthorID` (ignore not-found → skip broadcast), delete the like, get count, broadcast `feed_post_like`, return. Keep the existing delete/count behavior; only add the author lookup + broadcast.

In `AddComment`, it already does `s.postRepo.GetByID(postID)` — capture that `post`. After creating the comment, compute the count and broadcast:
```go
	cnt := int64(0)
	if counts, cerr := s.commentRepo.CountByPostIDs([]uint{postID}); cerr == nil {
		cnt = counts[postID]
	}
	s.broadcastToFollowers(post.AuthorID, true, "feed_comment_count", map[string]any{"post_id": postID, "comment_count": cnt})
```
(Return the created comment as before.)

- [ ] **Step 5: Wire the WS manager in server.go**

In `internal/server/server.go`, after `feedService.SetCommentLikeRepo(...)`, add:
```go
	feedService.SetWebSocketManager(wsManager)
```

- [ ] **Step 6: Run tests + full check**

Run: `go test ./internal/service/ -run TestFeedService_ -v` (all feed service tests still pass) then `make check` (or the non-race fallback).
Expected: PASS. Swagger unaffected (no route/DTO change).

- [ ] **Step 7: Commit**

```bash
git add internal/service/feed_service.go internal/server/server.go internal/service/feed_service_test.go
git commit -m "feat(feed): fan-out feed_new_post / feed_post_like / feed_comment_count to followers"
```

---

### Task 3: Frontend — WS dispatch + FeedView pill + live counts

**Files:**
- Modify: `web/src/composables/useWebSocket.js` (3 event cases)
- Modify: `web/src/views/FeedView.vue` (subscribe + pill + live counts)
- Modify: `web/src/i18n/locales/{en,zh,zh-tw,ja}.js` (`feed.newPosts`)
- Verify: `npm --prefix web run build` + manual

**Interfaces:**
- Consumes: Task 2 events.

- [ ] **Step 1: Dispatch the events**

In `web/src/composables/useWebSocket.js`, next to the existing `message`/`post_like` cases, add:
```js
            case 'feed_new_post':      notify('feed_new_post', msg.d); break
            case 'feed_post_like':     notify('feed_post_like', msg.d); break
            case 'feed_comment_count': notify('feed_comment_count', msg.d); break
```

- [ ] **Step 2: Subscribe + pill + live counts in FeedView**

In `web/src/views/FeedView.vue`:
- Imports: add `onUnmounted` to the vue import; `import { useWebSocket } from '@/composables/useWebSocket.js'` and `const ws = useWebSocket()`.
- Add `const newCount = ref(0)`.
- Handler:
```js
function onWSFeed(type, data) {
  if (type === 'feed_new_post') {
    newCount.value++
    return
  }
  const post = posts.value.find(p => p.id === data.post_id)
  if (!post) return
  if (type === 'feed_post_like' && typeof data.like_count === 'number') post.like_count = data.like_count
  else if (type === 'feed_comment_count' && typeof data.comment_count === 'number') post.comment_count = data.comment_count
}
onMounted(() => ws.onMessage(onWSFeed))
onUnmounted(() => ws.offMessage(onWSFeed))
```
  (Keep the existing `onMounted(load)` — either combine into one onMounted that calls both `load()` and `ws.onMessage(onWSFeed)`, or add a second onMounted; both work.)
- Reset the counter on manual load / tab switch: in `switchTab` and at the start of `load()`, set `newCount.value = 0`.
- Pill: a `loadNew()` that resets and reloads:
```js
function loadNew() { newCount.value = 0; load() }
```
- Template: add the pill at the top of `.feed-column`, above the tab bar or just under it (fixed-feel, centered), shown only when `newCount > 0`:
```html
<button v-if="newCount > 0" class="feed-newposts-pill" @click="loadNew">
  {{ t('feed.newPosts', { count: newCount }) }} ↑
</button>
```

- [ ] **Step 3: i18n**

Add to the `feed` section of ALL FOUR locales:
```js
        newPosts: '{count} 則新貼文',   // en: '{count} new posts'  ja: '{count}件の新着'  zh: '{count} 条新动态'
```

- [ ] **Step 4: Pill styling**

Add scoped CSS to FeedView, Kinetic Noir accent pill:
```css
.feed-newposts-pill {
  align-self: center;
  background: var(--accent, #5865f2);
  color: #fff;
  border: none;
  border-radius: 9999px;
  padding: 8px 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  transition: filter 0.1s;
}
.feed-newposts-pill:hover { filter: brightness(1.08); }
```

- [ ] **Step 5: Verify build**

Run: `npm --prefix web run build`
Expected: exit 0.

- [ ] **Step 6: Manual check**

Two browser sessions, A follows B. B posts → A sees the pill (no auto-inject); A taps → loads. B likes/comments a post in A's timeline → A's count updates live.

- [ ] **Step 7: Commit**

```bash
git add web/src/composables/useWebSocket.js web/src/views/FeedView.vue web/src/i18n/locales/
git commit -m "feat(feed): N-new-posts pill + live like/comment counts"
```

---

## Notes for the implementer

- **Order:** Task 1 → 2 backend (strict), Task 3 frontend (needs Task 2 live).
- **`make check` after each backend task** (non-race fallback on this Windows box — no gcc).
- **Fan-out is synchronous** and testable via `MockWebSocketManager.UserBroadcasts`.
- **Do not** auto-inject posts, live-insert comments, or touch FeedProfile realtime — all out of scope (B1).
- **Do not `git add -A`** — leave the unrelated `web/src/components/learn/*.vue` changes uncommitted.
```
