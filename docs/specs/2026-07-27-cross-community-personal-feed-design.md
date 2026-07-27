# 跨社群個人化 Feed 模組設計

- 日期:2026-07-27
- 狀態:設計定案,待實作
- 範圍:獨立的跨社群個人動態模組(Twitter 式:發個人貼文 + 單向追蹤 + 時間序 timeline)

## 目標

在 TalkRealm 加入一個**獨立於聊天/動態牆**的個人化 feed:使用者發「個人貼文」(不屬於任何 guild),單向追蹤其他使用者,首頁 timeline 顯示「我追蹤的人 + 我自己」的貼文,新到舊。貼文支援按讚與單層留言。此模組仿 `learn` 的組織方式,可整包抽離。

## 與動態牆的關係(讀本設計前必讀)

這是先前 `docs/specs/2026-07-24-guild-activity-wall-design.md` 中明確劃為「未來獨立模組」的那一塊。兩者是**兩個不同產品**:

| | 社群內動態牆(已完成) | 跨社群個人化 feed(本設計) |
|---|---|---|
| 本質 | 聊天的延伸 | 獨立產品 |
| 內容 | 就是 `Message`(feed 頻道) | 全新型別 `FeedPost` |
| 關係軸 | guild 成員 | 單向 `Follow` |
| 資料 | 複用 Message/Channel | 自己的表,不共用 |
| 即時 | WebSocket(頻道廣播) | **MVP 不做即時,純 pull** |

**硬約束:本模組不得修改或共用動態牆/聊天的資料表。** 邊界上只**唯讀**既有資料(見下),不改對方 schema。兩者不共用 model 是刻意的獨立性,不是重複。

## 模組結構(仿 learn)

同 package、以 `feed_` 前綴分檔:
- `internal/model/feed.go`
- `internal/repository/feed_repository.go`(可含 `feed_follow_repository.go` 視大小拆分)
- `internal/service/feed_service.go`
- `internal/handler/feed_handler.go`
- 路由群組:`/api/v1/feed`
- 前端:`web/src/views/FeedView.vue`(現為 placeholder,改寫)+ `web/src/components/feed/*`

### 本模組「擁有」的表

`Follow`、`FeedPost`、`FeedComment`、`FeedPostLike`、`FeedPostAttachment`。

### 只在邊界「唯讀」的既有資源(不共用 schema、不改對方)

| 讀什麼 | 用途 |
|---|---|
| `User` | 貼文作者的頭像/暱稱 |
| `Friendship` | 追蹤建議:列出 chat 好友中尚未追蹤者 |
| `File` + presign/confirm 上傳管線 | 貼文帶圖(通用儲存基建,非牆專屬) |

### 明確不做即時(MVP)

個人 feed 是跨 guild、以追蹤為軸,沒有頻道可廣播;即時化需要「發文時 fan-out 推給每個粉絲」,正是本設計不採用的 fan-out 模式。且真實產品(Twitter)也不即時推 timeline,而是進頁/下拉載入。**MVP 完全不接 WebSocket,純 pull**;即時 like/comment 數亦不做,查詢時算好即可。

## §1 資料模型(5 張自己的表)

```
Follow                          單向追蹤
  ID          uint  primarykey
  FollowerID  uint  uniqueIndex(idx_follow_pair) index   // 追蹤者(我)
  FolloweeID  uint  uniqueIndex(idx_follow_pair) index   // 被追蹤者
  CreatedAt   time.Time
  // FollowerID index → 算 timeline 來源；FolloweeID index → 算粉絲

FeedPost                        個人貼文(只有頂層)
  ID         uint  primarykey
  AuthorID   uint  index                // → User
  Content    string  not null
  IsEdited   bool  default:false
  CreatedAt / UpdatedAt

FeedComment                     單層留言
  ID         uint  primarykey
  PostID     uint  index                // → FeedPost
  AuthorID   uint                       // → User
  Content    string  not null
  IsEdited   bool  default:false
  CreatedAt / UpdatedAt

FeedPostLike                    讚(貼文層)
  ID         uint  primarykey
  PostID     uint  uniqueIndex(idx_feedlike_pair)   // 一人對一貼文只能讚一次
  UserID     uint  uniqueIndex(idx_feedlike_pair)
  CreatedAt  time.Time

FeedPostAttachment              貼文帶圖(連到通用 File)
  ID         uint  primarykey
  PostID     uint  index
  FileID     uint                       // → File(邊界複用)
  CreatedAt  time.Time
```

### 邊界決策

1. **timeline 含自己的貼文**:來源 = `我追蹤的人 + 我自己`。
2. **留言不可按讚**(MVP):`FeedPostLike` 只作用於貼文。留言按讚 YAGNI,日後加一張表即可。
3. **刪貼文 cascade**:單一 transaction 內清 `FeedComment` + `FeedPostLike` + `FeedPostAttachment` + 貼文本身。刪留言只清該留言。
4. **個人主頁**:某人貼文 = `author_id = :userId` 一支查詢,兼作 profile timeline。
5. **計數**:`like_count`/`comment_count`/`liked_by_me` 以批次查詢 enrich,不做 denormalized 計數欄。
   `ponytail: COUNT 即可,真的變熱門再加快取欄`。

### 遷移

於 `pkg/database/database.go` 的 `AutoMigrate(...)` 註冊 5 個新 model。

## §2 API(全在 `/api/v1/feed` 底下,完全獨立)

因為貼文是新型別,編輯/刪除/讚皆為模組自有 endpoint,不複用 `/messages/:id`(這是獨立性的必然)。

### 追蹤
```
GET    /feed/suggestions                 chat 好友中尚未追蹤者(邊界讀 Friendship)
POST   /feed/follows/:userId             追蹤
DELETE /feed/follows/:userId             取消追蹤
GET    /feed/users/:userId/following     某人追蹤中(含 count)
GET    /feed/users/:userId/followers     某人粉絲(含 count)
```

### 貼文 / timeline
```
GET    /feed/timeline?before=&limit=              首頁 timeline(追蹤的人 + 自己),enrich
POST   /feed/posts                                發文(content + file_ids)
GET    /feed/users/:userId/posts?before=&limit=   某人 profile timeline,enrich
PUT    /feed/posts/:id                            編輯自己的貼文
DELETE /feed/posts/:id                            刪自己的貼文(cascade)
PUT    /feed/posts/:id/like                       按讚(idempotent)
DELETE /feed/posts/:id/like                       收回讚
GET    /feed/posts/:id/comments?before=&limit=    留言列表
POST   /feed/posts/:id/comments                   留言
PUT    /feed/comments/:id                         編輯自己的留言
DELETE /feed/comments/:id                         刪自己的留言
```

### 權限
- 發文/留言:任何登入使用者。
- 編輯/刪除:僅作者本人(貼文與留言皆是)。
- 追蹤:不能追蹤自己;重複追蹤 idempotent。

### 回應 DTO
```
FeedPostResponse {
  ...FeedPost 欄位,
  author: User(頭像/暱稱),
  attachments: []File,
  like_count, comment_count int64,
  liked_by_me bool,
}
TimelineResponse { posts: []FeedPostResponse, has_more bool }
```

## §3 timeline 查詢

```sql
SELECT * FROM feed_posts
WHERE author_id IN (SELECT followee_id FROM follows WHERE follower_id = :me
                    UNION SELECT :me)
  [AND id < :before]              -- cursor
ORDER BY id DESC
LIMIT :limit + 1;                 -- +1 判斷 has_more
```
- profile timeline:把 `IN(...)` 換成 `author_id = :userId`。
- enrich:批次 `CountLikesByPostIDs`、`CountCommentsByPostIDs`、`LikedPostIDs(userID, ids)`、Preload 作者與附件的 File。
- cursor:`id DESC`,`before` = 前一頁最後一筆 id。

## §4 前端(填 `/feed` 現成路由)

路由 `/feed`、`FeedView.vue`、NavRail「動態」分頁**皆已接好**;`FeedView.vue` 現為 placeholder,改寫為真 feed 頁。純 pull、不接 WS。

### 版面
單欄為主,寬螢幕右側補「建議追蹤」:
- 頂部發文框(文字 + 圖,複用既有上傳流程)。
- 貼文卡片列表,新到舊,下拉無限捲(`before=` cursor)。
- 貼文卡:作者/時間、內文、圖、`♥ 讚數`、`💬 留言數`;展開留言(單層)+ 留言框。
- 寬螢幕右側:`FeedFollowSuggestions`(chat 好友,一鍵追蹤)。
- 點作者頭像 → `FeedProfile`(該人 timeline + 追蹤鈕 + 追蹤/粉絲數)。

### 新元件(複用既有頭像/時間/附件/Lightbox 與 Kinetic Noir 樣式)
`web/src/views/FeedView.vue`(改寫)、`web/src/components/feed/FeedComposer.vue`、`FeedPostCard.vue`、`FeedProfile.vue`、`FeedFollowSuggestions.vue`。

### API client
`web/src/api/index.js` 新增 `feed` 方法群:`getTimeline`、`createFeedPost`、`getUserPosts`、`likeFeedPost`、`unlikeFeedPost`、`getFeedComments`、`addFeedComment`、`follow`、`unfollow`、`getSuggestions`、`getFollowing`、`getFollowers`。i18n 4 語系補鍵。

## 測試

- **後端**(table-driven,仿既有 `*_service_test.go` + `testutil` mock):
  - 追蹤/取消追蹤 idempotent、不能追蹤自己
  - timeline 只回「追蹤者 + 自己」的貼文、cursor 方向正確、has_more
  - profile timeline 只回該作者
  - 發文/留言(author 綁定)、編輯/刪除僅作者
  - 讚 idempotent、收回、一人一讚 unique、count 正確、liked_by_me
  - 刪貼文 cascade:留言/讚/附件一併清;刪留言只清自己
  - 建議名單:排除已追蹤者與自己
- **前端**:無單元測試框架,build + 手動驗證(發文、追蹤、timeline 顯示、profile、無限捲)。

## 不做(YAGNI)

演算法排序、fan-out-on-write、即時 WebSocket、留言按讚、轉發/quote、推薦不認識的人、denormalized 計數、通知系統。
