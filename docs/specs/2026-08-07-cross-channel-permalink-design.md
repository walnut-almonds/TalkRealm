# 跨頻道 Permalink 設計(精準跳轉)

- 日期:2026-08-07
- 狀態:設計定案,待實作
- 範圍:任一聊天訊息或動態牆貼文都可產生內部連結,貼到其他頻道 → 渲染成引用預覽卡 → 點擊精準跳轉(載入該訊息附近、捲到、高亮),即使目標是很舊、未載入的訊息

## 目標

讓使用者把「某頻道的某則訊息 / 某篇貼文」分享到另一個頻道,收到的人點一下就跳到原處——對齊主流聊天軟體的訊息連結體驗。聊天訊息與牆貼文一致支援(兩者底層都是 `Message`)。

## 名詞與現況

- 聊天訊息與牆貼文都是 `model.Message`(貼文 = feed 頻道、`ParentID IS NULL` 的 Message)。
- 現有 `GET /channels/:id/messages?before=` 只能往舊翻頁;`GET /channels/:id/posts?before=` 同理。**沒有「載入某訊息附近」的能力**,前端也**沒有捲到指定訊息 / 高亮**的基建——這是本功能的主要工程量。
- 前端導航:store 的 `selectGuild(id)` / `selectChannel(id)` 已存在;router 為 hash 模式。

## §1 連結產生 + 引用卡片

### Token 格式(URL 式內部深連結)

`{origin}/#/m/<messageId>`
- 訊息與貼文共用(都是 message id)。
- 貼進聊天輸入框時就是一段文字;送出後前端 renderer 以正則認得**自家** `#/m/<id>` 樣式 → 渲染成引用卡(而非裸連結)。
- 直接於瀏覽器開啟此 URL 也能 deep-link 跳轉(router 加 `/m/:id` 路由,載入時觸發跳轉流程)。
- 存取控制在**解析時**做(非該 guild 成員 → 擋),故 URL 外流無妨。

### 分享動作

- `MessageItem`(聊天)與 `FeedPostCard`(牆)的動作列各加「複製連結」→ 將 `{origin}/#/m/<id>` 寫入剪貼簿(`navigator.clipboard.writeText`),toast 提示已複製。

### 解析端點(供預覽卡 + 導航)

`GET /api/v1/messages/:id/permalink` → 
```json
{
  "message": { "id": 123, "content": "<摘要，截斷>", "author": { "id", "nickname"/"username", "avatar" } },
  "channel": { "id": 5, "name": "general", "guild_id": 2, "type": "text" }
}
```
- 複用既有 `messageService.GetMessage` 取訊息 + `channelRepo.GetByID` 取頻道;存取檢查:檢視者須為 `channel.guild_id` 的成員(複用既有 `ensureChannelAccess` / guildMember 檢查)。非成員 → 403。
- `content` 摘要:截斷至約 100 字元(去換行)。

### 引用卡片(預覽)

- 新元件 `MessagePermalinkCard`:props `messageId`。掛載時 `api.resolvePermalink(messageId)` → 顯示 `作者 · #頻道名 · 摘要`。載入中顯示 skeleton;解析失敗(403/404)顯示「無法存取的引用」停用狀態。
- 點卡片 → `store.jumpToPermalink(messageId)`(見 §2)。
- 在聊天訊息內容渲染時,偵測 `#/m/<id>` → 以 `MessagePermalinkCard` 取代該段文字。

## §2 精準跳轉

### 後端「載入某訊息附近」查詢

新增(回應形狀與既有 list 一致,含 enrichment):
- `GET /api/v1/channels/:id/messages?around=<msgId>&limit=` → 以該訊息為中心一頁:前 `limit/2` 則(`id <= around`)+ 後 `limit/2` 則(`id > around`),時間序回傳。
- `GET /api/v1/channels/:id/posts?around=<postId>&limit=` → 同理,牆貼文(`parent_id IS NULL`)。
- repo:`GetMessagesAround(channelID, aroundID, limit)` / `GetPostsAround(channelID, aroundID, limit)` —— 兩段查詢(`id <= around` DESC limit ⌈n/2⌉、`id > around` ASC limit ⌊n/2⌋)在 Go 合併為時間序。`around` 不存在時回空。
- handler:既有 list handler 加 `around` query 分支(有 `around` 走 around 查詢,否則維持 `before` cursor)。

### 前端跳轉流程

store 新增 `jumpToPermalink(messageId)`:
1. `info = await api.resolvePermalink(messageId)`;失敗 → toast「無法跳轉/無權限」並中止。
2. 若 `info.channel.guild_id !== currentGuild?.id` → `await selectGuild(info.channel.guild_id)`。
3. 依 `info.channel.type` 載入**該訊息附近**:
   - `text` → 進該頻道並以 `around` 載入 MessageList(新增 `selectChannel(channelId, { around })` 或 `loadMessagesAround`)。
   - `feed` → 進該 feed 頻道並以 `around` 載入 FeedArea 的貼文。
4. 設 `store.highlightMessageId = messageId`。
5. `nextTick` 後 `scrollToMessage(messageId)`。

### 捲動 + 高亮基建(聊天 + 牆各接一次)

- `MessageList`(聊天)與 `FeedArea`(牆)給每則訊息/貼文容器加 `:id="'msg-' + m.id"`。
- `scrollToMessage(id)`:`document.getElementById('msg-'+id)?.scrollIntoView({ block: 'center', behavior: 'smooth' })`。放 store 或共用 util。
- 高亮:元件對 `store.highlightMessageId === m.id` 套一個 `.msg-highlight` class(短暫背景閃爍);跳轉後約 2s 由 store 清除 `highlightMessageId`。

## §3 測試

- **後端(table-driven + sqlmock)**:
  - `GetMessagesAround` / `GetPostsAround`:回傳以 around 為中心的窗格、時間序、around 不存在回空。
  - `permalink` 解析:回傳訊息摘要 + 頻道(guild_id/type);非成員 → 403(存取檢查)。
- **前端**:build + 手動(見 E2E)。
- **E2E(雙情境,精準跳轉是重點)**:
  1. **聊天訊息**:頻道 X 發足夠多訊息使目標訊息捲出畫面 → 複製其連結 → 到頻道 Y 貼上送出 → 出現引用卡(作者/頻道/摘要正確)→ 點卡片 → 跳到頻道 X、**載入該舊訊息**、捲到並高亮。
  2. **牆貼文**:feed 頻道發多篇 → 分享一篇舊貼文 → 貼到某 text 頻道 → 點卡片 → 跳到 feed 頻道、載入該貼文、捲到並高亮。
  3. 非成員點卡片 → 被擋(toast)。

## 不做(YAGNI)
- 公開(對外)分享網址、Open Graph 預覽。
- 引用卡片顯示附件縮圖(先只作者/頻道/文字摘要)。
- 跨 guild 未加入時的「預覽但不可跳」;非成員一律擋。
- 連結到 DM 訊息(僅 guild 頻道與 feed 頻道)。
- 編輯後連結預覽即時更新(卡片載入當下解析一次即可)。
