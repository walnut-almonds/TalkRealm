# 跨社群 Feed 即時化設計(a + b,B1)

- 日期:2026-07-31
- 狀態:設計定案,待實作
- 範圍:讓跨社群個人 feed 有「即時感」——(a)「N 則新貼文」pill、(b) 即時讚/留言數;**不做 auto-inject**、不做展開留言串的即時插入(B1)

## 目標

現有跨社群 feed 是純 pull。本設計加入即時感,遵循 Twitter 模式:
- **(a)** 你追蹤的人發新貼文 → timeline 頂端出現「N 則新貼文 ↑」pill,**點了才載入**,不動你的捲動位置(不自動插入)。
- **(b)** 你正在看的貼文,別人按讚/留言時**數字即時跳**(B1:僅更新計數,展開的留言串不即時長出新留言)。

## 與既有設計的關係

延伸 `feed` 模組,不新增資料表。當初 MVP 刻意砍掉即時(純 pull);本設計以**輕量 fan-out ping** 補上,而非當初砍掉的「fan-out-on-write 物化 timeline」——只推訊號、不複製貼文內容進任何 per-user 表。以本平台規模,發文時對作者的數百名粉絲各推一則 `BroadcastToUser` 毫無壓力。

群組動態牆(頻道制)已有即時(複用聊天 WS),不在本設計範圍。

## §1 後端

### 新增 repo 查詢

`FollowRepository.FollowerIDs(followeeID uint) ([]uint, error)` — 回傳某作者的粉絲 id 清單,鏡像既有 `FolloweeIDs`(`SELECT follower_id FROM follows WHERE followee_id = ?`)。用於 fan-out 對象。加對應 `MockFollowRepository.FollowerIDsFn`。

### feedService 接 WebSocket

鏡像 `messageService` 的做法:
- `feedService` 加欄位 `wsManager service.WebSocketManager`(既有介面,`internal/service/message_service.go:31`,含 `BroadcastToUser(userID uint, msgType string, data any)`)。
- 加 `SetWebSocketManager(m WebSocketManager)` 到 `FeedService` 介面 + impl。
- `internal/server/server.go` 在 feedService 建構後加一行 `feedService.SetWebSocketManager(wsManager)`(`wsManager` 已在該 scope)。

### 三個事件(寫入成功後,非阻塞 goroutine fan-out)

所有 fan-out 皆 `if s.wsManager != nil` 保護;粉絲查詢失敗只記 log,不影響主流程。

| 觸發點(service) | 事件名 | 推送對象 | payload |
|---|---|---|---|
| `CreatePost` 成功後 | `feed_new_post` | `FollowerIDs(authorID)`(**不含作者自己**) | `{"author_id": <uint>}` |
| `LikePost` / `UnlikePost` 取得新 count 後 | `feed_post_like` | `FollowerIDs(authorID)` + `authorID` | `{"post_id": <uint>, "like_count": <int64>}` |
| `AddComment` 成功後 | `feed_comment_count` | `FollowerIDs(post.AuthorID)` + `post.AuthorID` | `{"post_id": <uint>, "comment_count": <int64>}` |

- `LikePost`/`UnlikePost` 已 `GetByID(postID)` 拿得到 `post.AuthorID` 與新 count。
- `AddComment` 已有 `post`(GetByID)與可用 `commentRepo.CountByPostIDs([]uint{postID})` 取得 `comment_count`。
- fan-out 以一個私有 helper 收斂,例如 `s.broadcastToFollowers(authorID uint, includeAuthor bool, event string, data any)`:查 `FollowerIDs`,對每人 `BroadcastToUser`,`includeAuthor` 時也推給作者。
- `feed_new_post` **不推給作者**(作者發文後前端已 optimistic 置頂,再收 pill 會怪)。

## §2 前端

### WS 分派(`web/src/composables/useWebSocket.js`)

在既有 message case 群旁加三個:
```js
case 'feed_new_post':      notify('feed_new_post', msg.d); break
case 'feed_post_like':     notify('feed_post_like', msg.d); break
case 'feed_comment_count': notify('feed_comment_count', msg.d); break
```

### FeedView(`web/src/views/FeedView.vue`)

訂閱既有 `useWebSocket` 的 `onMessage/offMessage` bus(`onMounted` 訂閱、`onUnmounted` 取消,鏡像 FeedArea/DMChatArea):

- **`feed_new_post`** → `newCount.value++`。頂端顯示 pill「`{{ newCount }} 則新貼文 ↑`」(僅 `newCount > 0` 時)。**點 pill** → `newCount = 0` + `load()`(重載當前 tab 頂端)。**不 auto-inject**。
  - 切換 tab(`switchTab`)或手動 `load()` 時一併 `newCount = 0`。
- **`feed_post_like`** → 在 `posts.value` 找 `p.id === data.post_id`,設 `p.like_count = data.like_count`。
- **`feed_comment_count`** → 找 `p.id === data.post_id`,設 `p.comment_count = data.comment_count`。
- 即時計數作用於 timeline(追蹤中 + 為你推薦皆吃,靠 post id 比對)。**FeedProfile 的即時先不做**(情境少;同一套事件日後可加)。

i18n:4 語系加 `feed.newPosts`(帶數量,如 `'{count} 則新貼文'` / `'{count} new posts'`)。

pill 樣式沿用 Kinetic Noir(accent 色小膠囊,置中固定於 timeline 頂),不引入新設計語言。

## §3 測試

### 後端(table-driven + testutil mock)
- 需要 `testutil.MockWebSocketManager`(已存在,記錄 `BroadcastToUser` 呼叫;若欄位不足則擴充記錄 user 廣播)。
- `CreatePost` → 對每個粉絲推 `feed_new_post`,**不含作者**。
- `LikePost`/`UnlikePost` → 推 `feed_post_like`,payload `{post_id, like_count}` 正確,對象含作者。
- `AddComment` → 推 `feed_comment_count`,`comment_count` 正確。
- `FollowRepository.FollowerIDs` repo 測試(sqlmock)。

### E2E(雙瀏覽器 session)
- A 追蹤 B。
- B 發文 → A 頂端出現「1 則新貼文」pill;A 點 pill → 載入該貼文。
- B(或他人)對某貼文按讚 → A timeline 該貼文讚數即時跳。
- B 留言 → A 該貼文留言數即時跳。
- 驗證 A 的捲動位置**不被 auto-inject 打斷**(pill 出現但貼文不自己冒出來)。

## 不做(YAGNI)
- Auto-inject 新貼文進 timeline(Twitter 也不做)。
- 展開留言串的即時插入(B2)。
- FeedProfile 頁的即時更新。
- fan-out-on-write 物化 timeline、離線推播/通知中心、pill 顯示粉絲頭像。
