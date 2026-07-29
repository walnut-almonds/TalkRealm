# 「為你推薦」發現型 Feed 設計(Tier 0+1)

- 日期:2026-07-29
- 狀態:設計定案,待實作
- 範圍:在現有 `feed` 模組上加一個演算法排序的「為你推薦」時間軸(趨勢 + 輕量個人化),與現有「追蹤中」時間軸並列

## 目標

現有 `/feed` 是「追蹤中」(Following)時間軸:時間序、只有你追蹤的人。本設計新增「為你推薦」(For You):以**互動熱度 + 時間衰減**排序(Tier 0 趨勢),再依**你過去對各作者的互動**做輕量個人化加權(Tier 1),讓你看到全站你可能喜歡的貼文(含未追蹤者)。**不做 ML、不記曝光/停留、不做背景 worker/物化表。**

## 與現有 feed 模組的關係

這是 `feed` 模組的延伸,不是新模組。**複用** `FeedPost` / `FeedPostLike` / `FeedComment` 與現有的 enrichment DTO(`FeedPostResponse`)。**不新增任何資料表。** 新增一個 `/feed/discover` 端點與排序邏輯;「追蹤中」`/feed/timeline` 不動。

## §1 候選池與排除規則

每次請求即時計算,不預先物化:

- **時間窗**:只納入最近 **14 天**的貼文(`FEED_DISCOVER_WINDOW_DAYS`)。
- **排除**:排除**檢視者自己**的貼文(不該「發現」自己)。**不排除**已追蹤者的貼文(For You 本來就會把已追蹤者的爆文推上來,與「追蹤中」部分重疊屬正常)。
- **候選上限**:取窗內**最近 N=500 筆**(`FEED_DISCOVER_POOL_SIZE`)進入評分。用上限把每次請求的運算 bound 住。
  `ponytail: pool 上限 500 取最近;窗內貼文長期超過再改預算或物化,升級路徑明確`。

以本平台規模,每次請求重算 ≤500 筆的分數毫無壓力,無需背景 worker。

## §2 評分公式

對候選池每一筆計算分數,由大到小排序:

```
engagement = like_count + W_comment × comment_count
affinity   = min(viewer_affinity[author_id], AFFINITY_CAP)
decay      = pow(age_hours + 2, GRAVITY)
jitter     = 1 + jitterFrac(post_id, today)          # 探索抖動，當天穩定

score = (engagement + W_aff × affinity) / decay × jitter
```

### 參數(集中管理，config 或常數；預設值)

| 參數 | 常數名 | 預設 | 意義 |
|---|---|---|---|
| 留言權重 | `W_comment` | 2 | 留言比讚值錢 |
| 親近度權重 | `W_aff` | 3 | 個人化強度 |
| 親近度上限 | `AFFINITY_CAP` | 10 | 避免單一作者壟斷 |
| 時間衰減指數 | `GRAVITY` | 1.5 | 越舊掉越快 |
| 抖動幅度 | `JITTER_FRAC` | 0.10 | ±10% 探索 |
| 時間窗 | `FEED_DISCOVER_WINDOW_DAYS` | 14 | 候選新鮮度 |
| 候選上限 | `FEED_DISCOVER_POOL_SIZE` | 500 | 每請求運算預算 |

**所有參數集中在一處常數/設定**,不散落程式各處——此類公式須對真實行為反覆調校,寫死即失去調整能力。

### 親近度(Tier 1 個人化訊號)

`viewer_affinity[author_id]` = 檢視者過去對該作者的**按讚 + 留言**筆數之和,只用現有 `FeedPostLike` / `FeedComment` 紀錄算出。一支(或兩支)查詢在請求開始時算出 `map[authorID]int64`,對候選逐筆查表。不使用曝光/停留/ML。

### 抖動(探索)

`jitterFrac(post_id, today)` = 以 `hash(post_id + "YYYY-MM-DD")` 為種子的偽隨機,落在 `[-JITTER_FRAC, +JITTER_FRAC]`。**同一天同一篇抖動固定** → 一次瀏覽期間排序穩定(offset 分頁可用);**隔天種子變** → 每天刷有新鮮感,並給冷門貼文機會。

## §3 分頁

發現型 feed 的分數排序**非單調 id**,故不用 cursor,改 **offset 分頁**:`GET /feed/discover?offset=&limit=`。因抖動當天穩定,同一天的多次「載入更多」排序一致,offset 不會前後跳動。`has_more` 由「候選池排序後是否還有下一段」決定(受 pool 上限與 14 天窗界定)。

## §4 API

新增一個端點,回應形狀與 `/feed/timeline` 一致(複用 `FeedPostResponse` 的 enrichment:作者、附件、like_count、comment_count、liked_by_me):

```
GET /feed/discover?offset=0&limit=20   為你推薦，演算法排序，enrich
```
回應:
```
{ "posts": [FeedPostResponse...], "has_more": bool }
```
其餘 feed 端點(發文、追蹤、按讚、留言、profile)全部不變、直接複用。

## §5 服務 / 資料層

在 `feed` 模組內,不動既有檔案的核心邏輯:

- **`internal/service/feed_ranking.go`(新)**:純函數 `scorePost(post, affinity, now, params) float64` 與 `jitterFrac(postID, day) float64`。無 I/O,好做 table-driven 測試。
- **`feed_service.go`**:新增 `DiscoverTimeline(viewerID uint, offset, limit int) (*TimelineResponse, error)`——撈候選 → 算 affinity map → 逐筆 `scorePost` → 排序 → 切 offset 分頁 → 複用既有 `enrich()`。
- **`feed_post_repository.go`**:新增
  - `RecentCandidates(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error)`(preload Author/Attachments,`author_id != excludeAuthorID AND created_at >= since`,`ORDER BY id DESC LIMIT limit`)。
  - `AuthorAffinity(viewerID uint) (map[uint]int64, error)`:合併「檢視者的讚」「檢視者的留言」各自 join 到 `FeedPost.author_id` 後 GROUP BY author 的計數。(可拆兩支查詢在 Go 合併,或一支 UNION ALL 子查詢。)

## §6 前端

`FeedView.vue` 頂部加一排 **tab 切換**:「為你推薦 / 追蹤中」。

- 兩個 tab 共用同一內容區與 `FeedPostCard`;選中的 tab 決定打哪個 API:
  - 追蹤中 → `api.getTimeline()`(cursor,現況不變)
  - 為你推薦 → `api.getDiscover(offset, limit)`(offset 分頁)
- **預設 tab = 追蹤中**(早期發現內容稀疏,預設穩;之後可一行改為預設「為你推薦」)。
- 右側「推薦追蹤」欄照舊。
- 無限捲:追蹤中用既有 cursor 邏輯;為你推薦用 offset 累加。
- `api/index.js` 新增 `getDiscover(offset=0, limit=20)`;i18n 4 語系補「為你推薦 / 追蹤中」tab 標籤。

## 測試

- **後端**(table-driven,`feed_ranking.go` 純函數最好測):
  - `scorePost`:留言權重、affinity 加權與 cap、時間衰減單調性(越舊分越低)、jitter 在 `[-JITTER_FRAC, +JITTER_FRAC]` 且同 `(post_id, day)` 穩定、隔天不同。
  - `DiscoverTimeline`(service,用 mock repo):排除自己的貼文、affinity 提升特定作者排序、offset 分頁與 has_more、enrich 正確。
  - repo:`RecentCandidates` 時間窗 + 排除作者 + 上限;`AuthorAffinity` 計數正確。
- **前端**:無單元框架,build + 手動(tab 切換、為你推薦載入與無限捲、預設 tab)。

## 不做(YAGNI)

ML/學習式排序、曝光/停留/點擊記錄、fan-out/物化 timeline、背景重算 worker、「已看過/已讚降權」、跨模組的 hashtag/主題、推薦不認識的人的專門候選生成(二度好友)——目前 affinity + 全站熱門已足夠;二度候選待 Tier 1 觀察後再議。
