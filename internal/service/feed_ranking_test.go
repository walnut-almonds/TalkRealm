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
	moreLikes := scorePost(1, 4, 0, 0, false, false, created, now)
	moreComments := scorePost(1, 0, 4, 0, false, false, created, now)
	assert.Greater(t, moreComments, moreLikes)

	// affinity boosts score (same post id, same time → same jitter)
	noAff := scorePost(1, 1, 0, 0, false, false, created, now)
	withAff := scorePost(1, 1, 0, 5, false, false, created, now)
	assert.Greater(t, withAff, noAff)

	// affinity is capped
	capped := scorePost(1, 1, 0, rankAffinityCap, false, false, created, now)
	over := scorePost(1, 1, 0, rankAffinityCap+50, false, false, created, now)
	assert.Equal(t, capped, over)

	// older post (same id, same engagement) scores lower — decay
	old := scorePost(1, 5, 0, 0, false, false, now.Add(-48*time.Hour), now)
	fresh := scorePost(1, 5, 0, 0, false, false, now.Add(-1*time.Hour), now)
	assert.Greater(t, fresh, old)
}

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
