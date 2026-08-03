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
	rankLikedPenalty   = 0.3  // already-liked posts are downranked, not excluded
	rankWSecondDegree  = 3.0  // boost for 2nd-degree authors (friends-of-followees)
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
