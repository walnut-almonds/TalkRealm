# Discover 排序精修設計(已讚降權 + 二度好友加權)

- 日期:2026-08-03
- 狀態:設計定案,待實作
- 範圍:在既有 discover(為你推薦)排序上加兩個訊號——已讚降權、二度好友候選加權。純排序層,零新表、零新端點

## 目標

現有 discover 分數 = `(engagement + W_aff×affinity) / decay × jitter`。本設計加兩項:
- **已讚降權**:你已按讚的貼文分數乘一個懲罰係數,不易再排到前面(但不完全排除)。
- **二度好友候選加權**:作者是「你追蹤的人所追蹤的人」(二度、且你尚未直接追蹤)時給固定 boost,提升「發現新的人」。

## 與既有設計的關係

延伸 `feed` 模組的 discover 功能(`docs/specs/2026-07-29-discover-feed-design.md`)。所有改動集中在排序層:`scorePost`(feed_ranking.go)、一支新 `FollowRepository` 查詢、`DiscoverTimeline` 接線。不動候選池規則、分頁、API、資料表。

## §1 `scorePost` 公式(internal/service/feed_ranking.go)

新增兩個常數(與既有參數並列,集中管理):
```
rankLikedPenalty  = 0.3   // 已讚貼文分數乘此係數(降權,不排除)
rankWSecondDegree = 3.0   // 二度好友作者的固定 boost(加在分子,約等於 1 單位 affinity)
```

`scorePost` 簽名加兩個布林參數:
```go
func scorePost(
	postID uint,
	likeCount, commentCount, affinity int64,
	likedByMe, secondDegree bool,
	createdAt, now time.Time,
) float64
```

公式:
```
engagement    = likeCount + rankWComment × commentCount
secondBoost   = secondDegree ? rankWSecondDegree : 0
decay         = (ageHours + 2) ^ rankGravity
base          = (engagement + rankWAffinity × min(affinity, cap) + secondBoost) / decay
score         = base × (1 + jitter)
if likedByMe: score ×= rankLikedPenalty
```

- 二度 boost 是**加在分子**(跟 affinity 同層,線性疊加)。
- 已讚降權是**最後乘上係數**(對整體分數打折,含 jitter 後)。

## §2 新 repo 查詢(internal/repository/feed_follow_repository.go)

`SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error)` —— 二度作者集合:你追蹤的人所追蹤的人,**排除你直接追蹤的人與你自己**。

```sql
SELECT DISTINCT f2.followee_id
FROM follows f1
JOIN follows f2 ON f2.follower_id = f1.followee_id
WHERE f1.follower_id = ?
  AND f2.followee_id <> ?
  AND f2.followee_id NOT IN (SELECT followee_id FROM follows WHERE follower_id = ?)
```
(三個 `?` 都是 viewerID。)回傳 `map[uint]bool`。加 `MockFollowRepository.SecondDegreeAuthorIDsFn`。

## §3 `DiscoverTimeline` 接線(internal/service/feed_service.go)

評分迴圈前多算兩個集合:
- `likedSet, _ := s.likeRepo.LikedPostIDs(viewerID, ids)`(既有方法,enrich 已用)。
- `secondSet, _ := s.followRepo.SecondDegreeAuthorIDs(viewerID)`。

評分時傳入:
```go
scorePost(
	p.ID,
	likeCounts[p.ID], commentCounts[p.ID], affinity[p.AuthorID],
	likedSet[p.ID], secondSet[p.AuthorID],
	p.CreatedAt, now,
)
```
其餘(排序、offset 分頁、enrich)不變。

## §4 測試

- **ranking(純函數,主要保證)**:
  - `likedByMe=true` → 分數為未降權版的 0.3 倍(同 postID/時間 → 同 jitter,可精確比對)。
  - `secondDegree=true` → 分數高於 `secondDegree=false`(其餘相同)。
  - 兩者疊加:二度 + 已讚 → 先加 boost 再打 0.3 折。
- **repo**:`SecondDegreeAuthorIDs` sqlmock —— 排除直接追蹤者與自己(regex 允許依 GORM 實際 SQL 放寬)。
- **service(選配)**:`DiscoverTimeline` 用 mock 驗證已讚貼文排序被壓低、二度作者貼文被拉高。
- **E2E**:
  - 造 viewer 已按讚的熱門貼文 A 與同條件未讚貼文 B → discover 中 B 排在 A 之前(A 被降權)。
  - 造二度作者(朋友的朋友、未直接追蹤)的低互動貼文 C 與陌生人同互動貼文 D → C 排在 D 之前(二度 boost)。

## 不做(YAGNI)
- 「已**看過**降權」(需曝光記錄,屬 Tier 2 基建,不在此)。
- 二度以上(三度…)、二度權重隨距離衰減。
- 把二度作者塞進候選池生成(目前候選是全站最近,已含二度作者的貼文;這裡只做**加權**,不改候選來源)。
