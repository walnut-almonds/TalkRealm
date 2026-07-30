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
