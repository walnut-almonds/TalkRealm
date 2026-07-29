# Discover (For You) Feed — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an algorithmically ranked "For You" timeline (`GET /feed/discover`) to the existing feed module: trending (engagement + time decay) plus light personalization (like/comment affinity), with seeded per-day jitter and offset pagination.

**Architecture:** Extends the `feed` module — no new tables. A pure-function ranking file (`feed_ranking.go`) scores posts; `FeedService.DiscoverTimeline` fetches a bounded candidate pool (last 14 days, excluding the viewer's own posts, cap 500), computes a viewer→author affinity map, scores + sorts in Go, offset-paginates, and reuses the existing `enrich()` for the returned page. Frontend adds a "For You / Following" tab toggle to `FeedView`.

**Tech Stack:** Go + Gin + GORM/PostgreSQL; Vue 3; testify + `testutil` function-field mocks; go-sqlmock for repo tests.

## Global Constraints

- Spec: `docs/specs/2026-07-29-discover-feed-design.md` — authoritative.
- No new database tables. Reuse `FeedPost`/`FeedPostLike`/`FeedComment` and the existing `FeedPostResponse`/`enrich()`.
- Do NOT modify `/feed/timeline` (Following) behavior; discover is additive.
- Ranking parameters live as named constants in ONE place (`feed_ranking.go`), not scattered: `rankWComment=2.0`, `rankWAffinity=3.0`, `rankAffinityCap=10`, `rankGravity=1.5`, `rankJitterFrac=0.10`, `DiscoverWindowDays=14`, `DiscoverPoolSize=500`.
- Discover pagination is offset-based (`?offset=&limit=`), NOT cursor — scores are non-monotonic in id.
- Jitter is deterministic per `(post_id, YYYY-MM-DD)` so a day's ordering is stable across "load more".
- No ML, no impression/dwell logging, no background worker, no materialized timeline.
- Mirror existing feed idioms: service uses the 5-arg `NewFeedService`; handler uses `feedUserID(c)` + `strconv` param parsing + the `feedError`/switch pattern; `testutil` function-field mocks with `var _ Interface = (*Mock)(nil)`.
- After every backend task run: `make check` (fall back to `go build ./... && go vet ./... && go test ./...` if `-race`/mise shims break — no gcc on this Windows box).
- Frontend has no unit-test framework; verify with `npm --prefix web run build` + manual.
- Commit after every task.

---

### Task 1: Ranking pure functions (`feed_ranking.go`)

**Files:**
- Create: `internal/service/feed_ranking.go`
- Test: `internal/service/feed_ranking_test.go`

**Interfaces:**
- Produces (package `service`, unexported except constants used across the module):
```go
const (
	rankWComment       = 2.0
	rankWAffinity      = 3.0
	rankAffinityCap    = 10
	rankGravity        = 1.5
	rankJitterFrac     = 0.10
	DiscoverWindowDays = 14
	DiscoverPoolSize   = 500
)
func jitterFrac(postID uint, day string) float64
func scorePost(postID uint, likeCount, commentCount, affinity int64, createdAt, now time.Time) float64
```

- [ ] **Step 1: Write the failing tests**

Create `internal/service/feed_ranking_test.go`:

```go
package service

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJitterFrac_InRangeAndStable(t *testing.T) {
	j1 := jitterFrac(7, "2026-07-29")
	j2 := jitterFrac(7, "2026-07-29")
	assert.Equal(t, j1, j2, "same (post,day) must be stable")
	assert.LessOrEqual(t, math.Abs(j1), rankJitterFrac+1e-9)
	// different day differs (overwhelmingly likely)
	assert.NotEqual(t, j1, jitterFrac(7, "2026-07-30"))
}

func TestScorePost_CommentWeightAndAffinityAndDecay(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	created := now.Add(-2 * time.Hour)

	// comments weigh more than likes (same post id → same jitter)
	moreLikes := scorePost(1, 4, 0, 0, created, now)
	moreComments := scorePost(1, 0, 4, 0, created, now)
	assert.Greater(t, moreComments, moreLikes)

	// affinity boosts score (same post id, same time → same jitter)
	noAff := scorePost(1, 1, 0, 0, created, now)
	withAff := scorePost(1, 1, 0, 5, created, now)
	assert.Greater(t, withAff, noAff)

	// affinity is capped
	capped := scorePost(1, 1, 0, rankAffinityCap, created, now)
	over := scorePost(1, 1, 0, rankAffinityCap+50, created, now)
	assert.Equal(t, capped, over)

	// older post (same id, same engagement) scores lower — decay
	old := scorePost(1, 5, 0, 0, now.Add(-48*time.Hour), now)
	fresh := scorePost(1, 5, 0, 0, now.Add(-1*time.Hour), now)
	assert.Greater(t, fresh, old)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/ -run "TestJitterFrac|TestScorePost" -v`
Expected: FAIL — undefined `jitterFrac`/`scorePost`.

- [ ] **Step 3: Implement**

Create `internal/service/feed_ranking.go`:

```go
package service

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// Discover ranking parameters — tune here, nowhere else.
const (
	rankWComment       = 2.0  // comments weigh more than likes
	rankWAffinity      = 3.0  // personalization strength
	rankAffinityCap    = 10   // max affinity per author, avoids monopoly
	rankGravity        = 1.5  // time-decay exponent
	rankJitterFrac     = 0.10 // ±10% exploration jitter
	DiscoverWindowDays = 14   // candidate freshness window
	DiscoverPoolSize   = 500  // per-request scoring budget
)

// jitterFrac returns a deterministic value in [-rankJitterFrac, +rankJitterFrac]
// seeded by (postID, day). Stable within a day, changes across days.
func jitterFrac(postID uint, day string) float64 {
	h := sha1.Sum([]byte(fmt.Sprintf("%d:%s", postID, day)))
	unit := float64(binary.BigEndian.Uint32(h[:4])) / float64(math.MaxUint32) // [0,1]
	return (unit*2 - 1) * rankJitterFrac
}

// scorePost computes the discover ranking score for one post.
func scorePost(postID uint, likeCount, commentCount, affinity int64, createdAt, now time.Time) float64 {
	if affinity > rankAffinityCap {
		affinity = rankAffinityCap
	}
	engagement := float64(likeCount) + rankWComment*float64(commentCount)
	ageHours := now.Sub(createdAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	decay := math.Pow(ageHours+2, rankGravity)
	base := (engagement + rankWAffinity*float64(affinity)) / decay
	return base * (1 + jitterFrac(postID, now.Format("2006-01-02")))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/ -run "TestJitterFrac|TestScorePost" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/feed_ranking.go internal/service/feed_ranking_test.go
git commit -m "feat(feed): discover ranking pure functions (score + jitter)"
```

---

### Task 2: Repo — `RecentCandidates` + `AuthorAffinity`

**Files:**
- Modify: `internal/repository/feed_post_repository.go` (interface + impl)
- Modify: `internal/testutil/mocks.go` (extend `MockFeedPostRepository`)
- Test: `internal/repository/feed_post_repository_test.go`

**Interfaces:**
- Produces on `FeedPostRepository`:
```go
RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error) // author_id != excludeAuthorID AND created_at >= since, newest-first, preload Author + Attachments.File
AuthorAffinity(viewerID uint) (map[uint]int64, error) // viewer's (likes + comments) counted per post-author
```
- Produces mock fields: `RecentCandidatesFn`, `AuthorAffinityFn` on `MockFeedPostRepository`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/repository/feed_post_repository_test.go`:

```go
func TestFeedPostRepository_RecentCandidates(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT \* FROM "feed_posts" WHERE \(author_id <> \$1 AND created_at >= \$2\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "content"}).
			AddRow(9, 2, "a").AddRow(8, 3, "b"))
	repo := repository.NewFeedPostRepository(db)
	posts, err := repo.RecentCandidates(5, time.Unix(0, 0), 500)
	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, uint(9), posts[0].ID) // newest-first
}

func TestFeedPostRepository_AuthorAffinity(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	// likes grouped by post author
	mock.ExpectQuery(`FROM "feed_post_likes"`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "cnt"}).AddRow(2, 3))
	// comments grouped by post author
	mock.ExpectQuery(`FROM "feed_comments"`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{"author_id", "cnt"}).AddRow(2, 1).AddRow(3, 4))

	repo := repository.NewFeedPostRepository(db)
	aff, err := repo.AuthorAffinity(5)
	require.NoError(t, err)
	assert.Equal(t, int64(4), aff[2]) // 3 likes + 1 comment
	assert.Equal(t, int64(4), aff[3]) // 0 likes + 4 comments
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/repository/ -run "TestFeedPostRepository_RecentCandidates|TestFeedPostRepository_AuthorAffinity" -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement**

Add to the `FeedPostRepository` interface in `internal/repository/feed_post_repository.go`:
```go
	RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error)
	AuthorAffinity(viewerID uint) (map[uint]int64, error)
```
Add the import `"time"` if not present. Implementations:

```go
func (r *feedPostRepository) RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error) {
	var posts []*model.FeedPost
	err := r.db.Preload("Author").Preload("Attachments.File").
		Where("author_id <> ? AND created_at >= ?", excludeAuthorID, since).
		Order("id DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

func (r *feedPostRepository) AuthorAffinity(viewerID uint) (map[uint]int64, error) {
	out := make(map[uint]int64)
	type row struct {
		AuthorID uint
		Cnt      int64
	}
	// viewer's likes, grouped by the liked post's author
	var likeRows []row
	if err := r.db.Table("feed_post_likes AS fpl").
		Select("fp.author_id AS author_id, COUNT(*) AS cnt").
		Joins("JOIN feed_posts AS fp ON fp.id = fpl.post_id").
		Where("fpl.user_id = ?", viewerID).
		Group("fp.author_id").Scan(&likeRows).Error; err != nil {
		return nil, err
	}
	for _, x := range likeRows {
		out[x.AuthorID] += x.Cnt
	}
	// viewer's comments, grouped by the commented post's author
	var commentRows []row
	if err := r.db.Table("feed_comments AS fc").
		Select("fp.author_id AS author_id, COUNT(*) AS cnt").
		Joins("JOIN feed_posts AS fp ON fp.id = fc.post_id").
		Where("fc.author_id = ?", viewerID).
		Group("fp.author_id").Scan(&commentRows).Error; err != nil {
		return nil, err
	}
	for _, x := range commentRows {
		out[x.AuthorID] += x.Cnt
	}
	return out, nil
}
```

- [ ] **Step 4: Extend the mock**

Add to `MockFeedPostRepository` in `internal/testutil/mocks.go`:
```go
	RecentCandidatesFn func(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error)
	AuthorAffinityFn   func(viewerID uint) (map[uint]int64, error)
```
```go
func (m *MockFeedPostRepository) RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error) {
	return m.RecentCandidatesFn(excludeAuthorID, since, limit)
}
func (m *MockFeedPostRepository) AuthorAffinity(viewerID uint) (map[uint]int64, error) {
	return m.AuthorAffinityFn(viewerID)
}
```
(Add `"time"` import to mocks.go if not already present.)

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/repository/ -run "TestFeedPostRepository_RecentCandidates|TestFeedPostRepository_AuthorAffinity" -v && go build ./...`
Expected: PASS and build OK. If sqlmock regex doesn't match GORM's emitted SQL (JOIN aliasing, `<>` vs `!=`, arg order), relax the regex/`WithArgs` while keeping the grouping/merge assertions.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_post_repository.go internal/testutil/mocks.go internal/repository/feed_post_repository_test.go
git commit -m "feat(feed): RecentCandidates + AuthorAffinity repo queries"
```

---

### Task 3: Service — `DiscoverTimeline`

**Files:**
- Modify: `internal/service/feed_service.go` (interface + impl)
- Test: `internal/service/feed_service_test.go`

**Interfaces:**
- Consumes: `scorePost` (Task 1); `RecentCandidates`/`AuthorAffinity` (Task 2); existing `likeRepo.CountByPostIDs`, `commentRepo.CountByPostIDs`, and `enrich()`.
- Produces on `FeedService`:
```go
DiscoverTimeline(viewerID uint, offset, limit int) (*TimelineResponse, error)
```

- [ ] **Step 1: Write the failing test**

Add to `internal/service/feed_service_test.go`:

```go
func TestFeedService_DiscoverTimeline_RanksAndPaginates(t *testing.T) {
	now := time.Now()
	// two candidates: post 8 by author 3 (high affinity), post 9 by author 2 (no affinity)
	cands := []*model.FeedPost{
		{ID: 9, AuthorID: 2, CreatedAt: now.Add(-time.Hour)},
		{ID: 8, AuthorID: 3, CreatedAt: now.Add(-time.Hour)},
	}
	posts := &testutil.MockFeedPostRepository{
		RecentCandidatesFn: func(excl uint, since time.Time, limit int) ([]*model.FeedPost, error) {
			assert.Equal(t, uint(5), excl) // excludes viewer's own posts
			return cands, nil
		},
		AuthorAffinityFn: func(viewerID uint) (map[uint]int64, error) { return map[uint]int64{3: 8}, nil },
		GetByIDFn:        func(id uint) (*model.FeedPost, error) { for _, p := range cands { if p.ID == id { return p, nil } }; return nil, nil },
	}
	likes := &testutil.MockFeedPostLikeRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
		LikedPostIDsFn:   func(uid uint, ids []uint) (map[uint]bool, error) { return map[uint]bool{}, nil },
	}
	comments := &testutil.MockFeedCommentRepository{
		CountByPostIDsFn: func(ids []uint) (map[uint]int64, error) { return map[uint]int64{}, nil },
	}
	svc := service.NewFeedService(nil, posts, likes, comments, nil)

	resp, err := svc.DiscoverTimeline(5, 0, 20)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 2)
	// author 3 has affinity 8 → post 8 outranks post 9 (same time, no engagement)
	assert.Equal(t, uint(8), resp.Posts[0].ID)
	assert.False(t, resp.HasMore)

	// offset pagination
	page2, err := svc.DiscoverTimeline(5, 1, 1)
	require.NoError(t, err)
	require.Len(t, page2.Posts, 1)
	assert.Equal(t, uint(9), page2.Posts[0].ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestFeedService_DiscoverTimeline -v`
Expected: FAIL — `DiscoverTimeline` undefined.

- [ ] **Step 3: Implement**

Add `DiscoverTimeline` to the `FeedService` interface and impl in `internal/service/feed_service.go` (add imports `sort` and `time`):

```go
func (s *feedService) DiscoverTimeline(viewerID uint, offset, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	since := time.Now().AddDate(0, 0, -DiscoverWindowDays)
	candidates, err := s.postRepo.RecentCandidates(viewerID, since, DiscoverPoolSize)
	if err != nil {
		return nil, err
	}
	affinity, err := s.postRepo.AuthorAffinity(viewerID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(candidates))
	for i, p := range candidates {
		ids[i] = p.ID
	}
	likeCounts, _ := s.likeRepo.CountByPostIDs(ids)
	commentCounts, _ := s.commentRepo.CountByPostIDs(ids)

	now := time.Now()
	type scored struct {
		p *model.FeedPost
		s float64
	}
	arr := make([]scored, len(candidates))
	for i, p := range candidates {
		arr[i] = scored{p, scorePost(p.ID, likeCounts[p.ID], commentCounts[p.ID], affinity[p.AuthorID], p.CreatedAt, now)}
	}
	sort.SliceStable(arr, func(i, j int) bool { return arr[i].s > arr[j].s })

	hasMore := offset+limit < len(arr)
	if offset > len(arr) {
		offset = len(arr)
	}
	end := offset + limit
	if end > len(arr) {
		end = len(arr)
	}
	page := make([]*model.FeedPost, 0, end-offset)
	for _, x := range arr[offset:end] {
		page = append(page, x.p)
	}
	return &TimelineResponse{Posts: s.enrich(page, viewerID), HasMore: hasMore}, nil
}
```

Note: `enrich()` re-queries counts for the returned page (≤limit posts) — a small, acceptable double-count vs the pool-wide counts fetched for scoring. `ponytail: reuse enrich for consistency; inline the already-fetched counts only if this ever shows up in a profile`.

- [ ] **Step 4: Run test + build**

Run: `go test ./internal/service/ -run TestFeedService_DiscoverTimeline -v && go build ./...`
Expected: PASS and build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/feed_service.go internal/service/feed_service_test.go
git commit -m "feat(feed): DiscoverTimeline service (score + sort + offset paginate)"
```

---

### Task 4: Handler + route — `GET /feed/discover`

**Files:**
- Modify: `internal/handler/feed_handler.go` (add `Discover` handler)
- Modify: `internal/server/server.go` (register route)
- Test: manual via `make check` compile + the finish-step E2E

**Interfaces:**
- Consumes: `FeedService.DiscoverTimeline` (Task 3).
- Produces HTTP: `GET /api/v1/feed/discover?offset=&limit=` → `TimelineResponse`.

- [ ] **Step 1: Add the handler**

In `internal/handler/feed_handler.go`, mirroring the existing `Timeline`/`ProfilePosts` handlers (userID via `feedUserID(c)`, error via `feedError`):

```go
// Discover GET /feed/discover?offset=&limit=
func (h *FeedHandler) Discover(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	resp, err := h.feedService.DiscoverTimeline(userID, offset, limit)
	if err != nil {
		feedError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
```
(Confirm `strconv`, `net/http` are already imported in the file; they are used by sibling handlers.)

- [ ] **Step 2: Register the route**

In `internal/server/server.go`, inside the existing `feed := protected.Group("/feed")` block, add next to `feed.GET("/timeline", ...)`:
```go
		feed.GET("/discover", s.feedHandler.Discover)
```

- [ ] **Step 3: Full check**

Run: `make check` (or the non-race fallback). Expected: build + lint + tests pass; swagger regenerates (include regenerated `docs/openapi/*` in the commit).

- [ ] **Step 4: Commit**

```bash
git add internal/handler/feed_handler.go internal/server/server.go docs/openapi/
git commit -m "feat(feed): GET /feed/discover route + handler"
```

---

### Task 5: Frontend — API + FeedView tabs

**Files:**
- Modify: `web/src/api/index.js` (EP + method)
- Modify: `web/src/views/FeedView.vue` (tab toggle + per-tab load)
- Modify: `web/src/i18n/locales/{en,zh,zh-tw,ja}.js` (tab labels)
- Verify: `npm --prefix web run build` + manual

**Interfaces:**
- Consumes: Task 4 endpoint.

- [ ] **Step 1: Add API method**

In `web/src/api/index.js` `EP` map:
```js
    FEED_DISCOVER: '/api/v1/feed/discover',
```
Method (near `getTimeline`):
```js
    getDiscover(offset = 0, limit = 20) {
      const q = new URLSearchParams({ offset, limit })
      return this.get(`${EP.FEED_DISCOVER}?${q}`)
    },
```

- [ ] **Step 2: Add i18n tab labels**

Add to the `feed` section of ALL FOUR locales in `web/src/i18n/locales/`:
```js
        tabForYou: '為你推薦',   // en: 'For You'  ja: 'おすすめ'  zh: '为你推荐'
        tabFollowing: '追蹤中',  // en: 'Following' ja: 'フォロー中' zh: '关注中'
```

- [ ] **Step 3: Add the tab toggle to FeedView**

Modify `web/src/views/FeedView.vue`:
- Add `const tab = ref('following')` (default Following, per spec).
- Replace `load()`/`loadOlder()` so they branch on `tab.value`:
  - `following` → `api.getTimeline()` / `api.getTimeline(20, lastId)` (existing cursor).
  - `discover` → `api.getDiscover(0, 20)` / `api.getDiscover(posts.value.length, 20)` (offset = current count).
- Add a `switchTab(next)` that sets `tab.value`, clears `posts.value = []`, resets `hasMore`, and calls `load()`.
- Render a tab bar above the composer:
```html
<div class="feed-tabs">
  <button class="feed-tab" :class="{ active: tab === 'discover' }" @click="switchTab('discover')">{{ t('feed.tabForYou') }}</button>
  <button class="feed-tab" :class="{ active: tab === 'following' }" @click="switchTab('following')">{{ t('feed.tabFollowing') }}</button>
</div>
```
- In `onScroll`, `loadOlder()` already branches via `tab`. Concrete `load`/`loadOlder`:
```js
async function load() {
  loading.value = true
  try {
    const res = tab.value === 'discover' ? await api.getDiscover(0, 20) : await api.getTimeline()
    posts.value = res.posts || []
    hasMore.value = !!res.has_more
  } catch (e) {
    store.showNotification(e.message || 'Failed to load feed', 'error')
  } finally { loading.value = false }
}

async function loadOlder() {
  if (!hasMore.value || loading.value || posts.value.length === 0) return
  loading.value = true
  try {
    const res = tab.value === 'discover'
      ? await api.getDiscover(posts.value.length, 20)
      : await api.getTimeline(20, posts.value[posts.value.length - 1].id)
    posts.value.push(...(res.posts || []))
    hasMore.value = !!res.has_more
  } catch (e) {
    store.showNotification(e.message || 'Failed to load feed', 'error')
  } finally { loading.value = false }
}

function switchTab(next) {
  if (tab.value === next) return
  tab.value = next
  posts.value = []
  hasMore.value = false
  load()
}
```
Style `.feed-tabs`/`.feed-tab` with the existing Kinetic Noir tokens (a simple two-button row with an `.active` underline/accent). The composer stays visible on both tabs (you can still post from For You).

- [ ] **Step 4: Verify build**

Run: `npm --prefix web run build`
Expected: exit 0.

- [ ] **Step 5: Manual check**

Run the app, open `/feed`: default is Following; click "為你推薦" → discover loads (algorithmically ranked); scroll to load more (offset); switch back to Following works.
Expected: both tabs load and paginate independently.

- [ ] **Step 6: Commit**

```bash
git add web/src/api/index.js web/src/views/FeedView.vue web/src/i18n/locales/
git commit -m "feat(feed): For You / Following tab toggle + discover API"
```

---

## Notes for the implementer

- **Order:** Tasks 1→4 backend, strictly ordered. Task 5 frontend, needs Task 4 live.
- **`make check` after each backend task** (non-race fallback on this Windows box — no gcc).
- **Ranking params live only in `feed_ranking.go`** — do not sprinkle magic numbers elsewhere.
- **Don't touch `/feed/timeline`** or any Following behavior — discover is purely additive.
- **sqlmock brittleness (Task 2):** relax regex/`WithArgs` to match GORM's real emitted SQL; keep the grouping/merge and newest-first assertions. Behavioral guarantees live in Tasks 1 and 3.
```
