# Discover Ranking Refinement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two signals to the discover (For You) ranking — downrank posts the viewer already liked (×0.3) and boost posts by 2nd-degree connections (+3), i.e. people your followees follow that you don't yet follow.

**Architecture:** Ranking-layer only, no new tables/endpoints. `scorePost` gains `likedByMe` and `secondDegree` booleans; a new `FollowRepository.SecondDegreeAuthorIDs` query supplies the 2nd-degree author set; `DiscoverTimeline` computes the liked set (existing `LikedPostIDs`) and the 2nd-degree set and passes them into `scorePost`.

**Tech Stack:** Go + Gin + GORM/PostgreSQL; testify + `testutil` mocks; go-sqlmock.

## Global Constraints

- Spec: `docs/specs/2026-08-03-discover-ranking-refinement-design.md` — authoritative.
- No new tables, no new endpoints, no candidate-pool change — ranking layer only.
- New ranking constants live with the others in `internal/service/feed_ranking.go`: `rankLikedPenalty = 0.3`, `rankWSecondDegree = 3.0`.
- Formula: `base = (engagement + rankWAffinity×min(affinity,cap) + (secondDegree?rankWSecondDegree:0)) / decay`; `score = base×(1+jitter)`; then `if likedByMe: score ×= rankLikedPenalty`.
- 2nd-degree set excludes the viewer's direct followees and the viewer themselves.
- After each backend task run `make check` (fall back to `go build ./... && go vet ./... && go test ./...` if `-race`/mise shims break — no gcc on this Windows box).
- Commit after each task. Do NOT `git add -A`/`git add .` — there are unrelated uncommitted `web/src/components/learn/*.vue` changes; add only your files.

---

### Task 1: `FollowRepository.SecondDegreeAuthorIDs` + mock

**Files:**
- Modify: `internal/repository/feed_follow_repository.go` (interface + impl)
- Modify: `internal/testutil/mocks.go` (extend `MockFollowRepository`)
- Test: `internal/repository/feed_follow_repository_test.go`

**Interfaces:**
- Produces: `FollowRepository.SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error)` — set of authors followed by the viewer's followees, excluding the viewer's direct followees and the viewer. Mock field `SecondDegreeAuthorIDsFn`.

- [ ] **Step 1: Write the failing test**

Add to `internal/repository/feed_follow_repository_test.go`:

```go
func TestFollowRepository_SecondDegreeAuthorIDs(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(`SELECT DISTINCT f2\.followee_id`).
		WithArgs(1, 1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"followee_id"}).AddRow(7).AddRow(9))

	repo := repository.NewFollowRepository(db)
	set, err := repo.SecondDegreeAuthorIDs(1)
	require.NoError(t, err)
	assert.True(t, set[7] && set[9])
	assert.False(t, set[1])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestFollowRepository_SecondDegreeAuthorIDs -v`
Expected: FAIL — `SecondDegreeAuthorIDs` undefined.

- [ ] **Step 3: Implement**

Add to the `FollowRepository` interface (next to `FollowerIDs`):
```go
	SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error)
```
Impl (raw SQL self-join; exclude direct follows + self):
```go
func (r *followRepository) SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error) {
	var ids []uint
	err := r.db.Raw(`
		SELECT DISTINCT f2.followee_id
		FROM follows f1
		JOIN follows f2 ON f2.follower_id = f1.followee_id
		WHERE f1.follower_id = ?
		  AND f2.followee_id <> ?
		  AND f2.followee_id NOT IN (SELECT followee_id FROM follows WHERE follower_id = ?)
	`, viewerID, viewerID, viewerID).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
```

- [ ] **Step 4: Extend the mock**

In `internal/testutil/mocks.go`, add to `MockFollowRepository`:
```go
	SecondDegreeAuthorIDsFn func(viewerID uint) (map[uint]bool, error)
```
```go
func (m *MockFollowRepository) SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error) {
	return m.SecondDegreeAuthorIDsFn(viewerID)
}
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/repository/ -run TestFollowRepository_SecondDegreeAuthorIDs -v && go build ./...`
Expected: PASS and build OK. (Relax the sqlmock regex/`WithArgs` if GORM's emitted SQL differs; keep the set-membership assertions.)

- [ ] **Step 6: Commit**

```bash
git add internal/repository/feed_follow_repository.go internal/testutil/mocks.go internal/repository/feed_follow_repository_test.go
git commit -m "feat(feed): FollowRepository.SecondDegreeAuthorIDs for discover boost"
```

---

### Task 2: `scorePost` signals + `DiscoverTimeline` wiring

**Files:**
- Modify: `internal/service/feed_ranking.go` (constants + `scorePost`)
- Modify: `internal/service/feed_service.go` (`DiscoverTimeline` — compute liked/2nd-degree sets, pass to scorePost)
- Test: `internal/service/feed_ranking_test.go`

**Interfaces:**
- Consumes: `FollowRepository.SecondDegreeAuthorIDs` (Task 1); existing `likeRepo.LikedPostIDs`.
- Produces: `scorePost(postID uint, likeCount, commentCount, affinity int64, likedByMe, secondDegree bool, createdAt, now time.Time) float64`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/feed_ranking_test.go`:

```go
func TestScorePost_LikedPenaltyAndSecondDegree(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	created := now.Add(-2 * time.Hour)

	base := scorePost(1, 4, 0, 0, false, false, created, now)

	// liked-by-me multiplies the score by rankLikedPenalty (same post/time → same jitter)
	liked := scorePost(1, 4, 0, 0, true, false, created, now)
	assert.InDelta(t, base*rankLikedPenalty, liked, 1e-9)

	// second-degree adds a boost → strictly higher than base
	second := scorePost(1, 4, 0, 0, false, true, created, now)
	assert.Greater(t, second, base)

	// both: boost applied then penalty
	both := scorePost(1, 4, 0, 0, true, true, created, now)
	assert.InDelta(t, second*rankLikedPenalty, both, 1e-9)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestScorePost_LikedPenaltyAndSecondDegree -v`
Expected: FAIL — too many arguments to `scorePost` (signature mismatch / won't compile).

- [ ] **Step 3: Update `scorePost` + constants**

In `internal/service/feed_ranking.go`, add to the const block:
```go
	rankLikedPenalty  = 0.3 // already-liked posts are downranked, not excluded
	rankWSecondDegree = 3.0 // boost for 2nd-degree authors (friends-of-followees)
```
Replace `scorePost` with:
```go
// scorePost computes the discover ranking score for one post.
func scorePost(
	postID uint,
	likeCount, commentCount, affinity int64,
	likedByMe, secondDegree bool,
	createdAt, now time.Time,
) float64 {
	if affinity > rankAffinityCap {
		affinity = rankAffinityCap
	}
	engagement := float64(likeCount) + rankWComment*float64(commentCount)
	ageHours := now.Sub(createdAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	secondBoost := 0.0
	if secondDegree {
		secondBoost = rankWSecondDegree
	}
	decay := math.Pow(ageHours+2, rankGravity)
	base := (engagement + rankWAffinity*float64(affinity) + secondBoost) / decay
	score := base * (1 + jitterFrac(postID, now.Format("2006-01-02")))
	if likedByMe {
		score *= rankLikedPenalty
	}
	return score
}
```

- [ ] **Step 4: Wire `DiscoverTimeline`**

In `internal/service/feed_service.go` `DiscoverTimeline`, after `likeCounts`/`commentCounts` are fetched and before the scoring loop, add:
```go
	likedSet, _ := s.likeRepo.LikedPostIDs(viewerID, ids)
	secondSet, _ := s.followRepo.SecondDegreeAuthorIDs(viewerID)
```
Change the `scorePost(...)` call inside the loop to pass the two new args:
```go
	arr[i] = scored{p, scorePost(
		p.ID,
		likeCounts[p.ID], commentCounts[p.ID], affinity[p.AuthorID],
		likedSet[p.ID], secondSet[p.AuthorID],
		p.CreatedAt, now,
	)}
```
(`likedSet`/`secondSet` are nil-safe: indexing a nil map returns the zero value `false`.)

- [ ] **Step 5: Run tests + full check**

Run: `go test ./internal/service/ -run "TestScorePost|TestFeedService_" -v` (ranking tests + all feed service tests still pass) then `make check` (or the non-race fallback).
Expected: PASS. (`DiscoverTimeline`'s existing test still passes — the two new mock-repo methods return zero values via the mocks' default funcs; if that test's mocks don't set `LikedPostIDsFn`/`SecondDegreeAuthorIDsFn`, add them returning empty maps.)

- [ ] **Step 6: Commit**

```bash
git add internal/service/feed_ranking.go internal/service/feed_service.go internal/service/feed_ranking_test.go
git commit -m "feat(feed): discover liked-downrank + 2nd-degree boost in scorePost"
```

---

## Notes for the implementer

- **Order:** Task 1 → Task 2 (Task 2 consumes `SecondDegreeAuthorIDs`).
- **Task 2 changes `scorePost`'s signature**, which breaks its only caller (`DiscoverTimeline`) until Step 4 updates it — do Steps 3 and 4 together before building.
- **The existing `TestFeedService_DiscoverTimeline_*` test** uses `MockFeedPostLikeRepository`/`MockFollowRepository`; ensure `LikedPostIDsFn` and `SecondDegreeAuthorIDsFn` are set on those mocks (returning empty maps) so the wired calls don't nil-panic. Update that test's mock setup if needed.
- **`make check` after each backend task** (non-race fallback on this Windows box — no gcc).
- **Do not `git add -A`** — leave the unrelated `web/src/components/learn/*.vue` changes uncommitted.

## E2E (finish step, not a task)

Seed and drive the real app to confirm ordering:
- A post the viewer already liked ranks below an equally-engaged un-liked post (liked-downrank).
- A low-engagement post by a 2nd-degree author (a followee's followee, not directly followed) ranks above an equally-low-engagement post by a stranger (2nd-degree boost).
```
