# TalkRealm TODO List

> 基於 2026-04-30 架構重設計，詳細規劃見 [`plan.md`](./plan.md)。

---

## 🌌 Social Galaxy — 首頁銀河可視化

> 詳細設計見 [`social_galaxy.md`](./social_galaxy.md)。  
> 目標：取代首頁空白畫面，以 SVG force-layout 呈現使用者的社群宇宙。  
> 實作策略：純 Vue + SVG + CSS 動畫，不引入新 npm 套件（MVP）。

### MVP（Phase 1）
- [x] **HomeView.vue**：SVG 星空背景 + 圓形軌道佈局，用戶居中、Guild 節點環繞，點擊節點進入社群
- [x] **onlineUserIds（AppStore）**：新增全域在線集合，`handleUserStatus` 同步更新，供 HomeView 顯示在線狀態光暈
- [x] **ChatView 整合**：`!currentGuild && !isDMMode` 時渲染 `<HomeView />`，取代空白畫面

### Phase 2（後續）
- [x] **互動強度 API**：後端新增 `GET /api/v1/users/me/interaction-stats`（最近 N 天訊息頻率），前端依強度調整節點距離與連線粗細
- [x] **成員節點**：載入首頁時批量取各 Guild 部分成員，作為 Guild cluster 的子節點顯示
- [x] **d3-force 物理模擬**：引入 `d3-force` 替換靜態圓形佈局，實現有機力導向布局
- [x] **縮放 / 拖曳**：scroll wheel zoom + drag-to-pan 導覽
- [x] **未讀衛星**：有未讀訊息的 Guild 節點顯示 orbiting 小衛星粒子（讀取 `appStore.unreadGuildIds`；目前為 **session-only**，頁面重整後清零 — 待「未讀 & Mention 系統」後端實作後，啟動時從 API 初始化即可自動補上跨 session 的未讀狀態）

---

## 🔴 Phase 1 — 強化現有 Monolith（高優先）

### WebSocket 改善
- [x] **WS Channel Index**：Manager 加入 `channelSubscriptions map[uint]map[*Client]bool`，只推訊息給訂閱該頻道的 client，O(1) 查找
- [x] **WS Identify Op**：連線後 server 發 `hello`（帶 heartbeat_interval），client 發 `identify`（帶 JWT），server 回 `ready`，並訂閱初始頻道列表
- [x] **Typing Indicator**：實作 `typing_start` op（WS Client → Server → 同頻道廣播，排除發送者）
- [x] **Presence 系統**：identify 後廣播 `presence_update` online，斷線後廣播 offline
- [x] **WS Heartbeat / Reconnect**：標準化 `heartbeat` / `heartbeat_ack`，前端自動重連邏輯
- [x] **Heartbeat 刷新 Redis TTL**：收到 `heartbeat` 時執行 `EXPIRE user:{userID}:server 86400`，以 key 存在與否作為 online/offline 唯一標準
- [x] **WS `send_message` op**：client 透過 WS 發訊息（`op: send_message`，帶 `channel_id / content / type / nonce`），server 驗證權限後寫 DB 並廣播 `message_create`（目前訊息發送走 REST POST）
- [x] **WS `message_create` / `message_update` / `message_delete` op**：統一 server→client 的訊息事件格式（目前部分廣播格式未對齊 plan.md Discord-like `op`/`d` 規格）
- [x] **`is_edited` DB migration**：`messages` 表加 `is_edited` 欄位（model 已有，migration 未執行）

### Redis 整合
- [x] **引入 Redis Client**：在 `pkg/redis/` 封裝連線，config 加入 Redis 設定
- [x] **User Server Mapping**：Chat Server 連線時 `SET user:{userID}:server {serverID} EX 86400`
- [x] **Guild Online Set**：`SADD guild:{guildID}:online {userID}` / `SREM` on disconnect
- [x] **Rate Limiting Middleware**：`INCR ratelimit:{userID}:msg` TTL 1s，超過 10 則回 429
- [x] **IsUserOnline**：`Manager.IsUserOnline(userID)` 以 Redis key 存在為準，無 Redis 時 fallback 本地 clients
- [ ] **Notification 分流**：Chat Server 發訊息前呼叫 `IsUserOnline(targetUserID)`；online → publish `topic:server.{serverID}`，offline → publish `topic:notification`（目前無此分流邏輯）

### API 補全
- [x] **Cursor-based Pagination**：`GET /api/v1/channels/{id}/messages` 改為 `?before={id}&limit=50`
- [x] **Guild 邀請碼系統**（`guild_invites` 表、`POST/GET/JOIN` API）
- [x] **Refresh Token**（`refresh_tokens` 表、`/auth/refresh`、`/auth/logout`）
- [x] **User 公開資料 API**：`GET /api/v1/users/{id}`（不含 email）
- [x] **訊息 Attachment 欄位**：Message 回應加入 `attachments []` 陣列
- [x] **`PATCH /api/v1/users/me`**：更新暱稱 / avatar / status（plan.md 4.2 已定義，目前未實作）
- [x] **`PUT /api/v1/channels/{id}/position`**：更新頻道排序（plan.md 4.4 已定義，未實作）
- [x] **`GET /api/v1/messages/{id}`**：取得單一訊息詳情（plan.md 4.5 已定義，未實作
- [ ] **OAuth 登入整合**：支援第三方登入 provider、callback 處理與 JWT 發放

### 權限系統（RBAC）
- [x] **Permission 中介層**：封裝 `RequireGuildRole(minRole)` middleware
- [x] **頻道操作權限**：create / update / delete channel 需 `admin` 以上
- [x] **成員管理權限**：kick / update role 需 `admin` 以上
- [x] **訊息刪除權限**：允許 channel admin 刪除他人訊息

### 未讀 & Mention 系統

> **與 Social Galaxy 的關係**：`appStore` 已有 `unreadGuildIds`（`reactive Set<guildId>`）與 `channelGuildMap`（`Map<channelId,guildId>`），`handleNewMessage()` 靠這兩個結構在 session 中動態更新 guild 級別的未讀，`selectGuild()` 已呼叫 `unreadGuildIds.delete(guildId)` 清除。  
> 以下後端任務完成後，前端只需「啟動時從 API 初始化這兩個 store 結構」，Social Galaxy 衛星動畫即自動獲得跨 session 的持久化未讀狀態，無需改動 HomeView.vue。  
> 新增 `channelUnreadMap`（`Map<channelId, {unread, mention}>`）供 sidebar 顯示 per-channel 計數。

#### DB / 後端
- [ ] **`channel_read_states` 表**：`user_id`、`channel_id`、`last_read_message_id`，記錄每個用戶在每個頻道的已讀位置（獨立表，比在 `guild_members` 加欄更彈性）
- [ ] **`message_mentions` 表**：`message_id`、`user_id`、`mention_type`（`user` / `here` / `everyone`），記錄每條訊息的被提及者
- [ ] **Message 解析 Mention**：`MessageService.CreateMessage()` 解析訊息內容，提取 `<@{userID}>` / `@here` / `@everyone`，批次寫入 `message_mentions`；`@here` 展開為頻道內 online 成員（呼叫 `IsUserOnline`），`@everyone` 展開為 guild 全員
- [ ] **`GET /api/v1/channels/{id}/unread`**：回傳 `{ unread_count, mention_count }`（依 `channel_read_states` 與 `message_mentions` 計算）
- [ ] **`POST /api/v1/channels/{id}/ack`**：body `{ last_message_id }`，更新 `channel_read_states.last_read_message_id`，清零前端未讀計數
- [ ] **`GET /api/v1/users/me/unread`**：批次回傳所有頻道的 `{ channel_id, guild_id, unread_count, mention_count }`（前端啟動時一次呼叫；回應需帶 `guild_id` 以供前端填充 `channelGuildMap`）
- [ ] **WS `mention_create` op**：訊息含有提及當前用戶時，server 推送獨立事件 `{ message_id, channel_id, guild_id, author }`，觸發前端高優先 badge 更新與通知

#### 前端
- [ ] **`channelUnreadMap` store state**：`appStore` 新增 `reactive Map<channelId, { unread: number, mention: number }>`，供 sidebar per-channel badge 使用
- [ ] **啟動時載入未讀狀態**：登入後呼叫 `GET /api/v1/users/me/unread`，同時初始化 `channelUnreadMap`、`unreadGuildIds`（guild 有任一 channel unread > 0）、及補齊 `channelGuildMap`（解決未訪問 guild 的頻道訊息被漏追蹤的問題）
- [ ] **切換頻道時 ACK**：`selectChannel()` 結尾呼叫 `POST /api/v1/channels/{id}/ack`，清零該頻道的 `channelUnreadMap` 計數；`selectGuild()` 已清 `unreadGuildIds`，需確認 guild 下所有頻道都 ACK（或清 map）
- [ ] **`handleNewMessage()` 補 channel-level 計數**：收到非當前頻道的訊息時，額外更新 `channelUnreadMap[channel_id].unread++`（mention 由 `mention_create` WS 事件更新）
- [ ] **頻道側欄未讀 badge**：頻道名旁顯示 `channelUnreadMap` 的未讀數字 badge；mention > 0 時改為紅底標示，普通未讀用灰色圓點
- [ ] **Guild 未讀提示**：Guild 頭像旁顯示小圓點（有普通未讀）或紅色數字（有 mention）；Social Galaxy 衛星動畫已自動反映（`unreadGuildIds` 共用）
- [ ] **WS `mention_create` 處理**：接收後更新 `channelUnreadMap[channel_id].mention++` 與 `unreadGuildIds`，可選觸發 browser notification（`Notification API`）或提示音
- [ ] **`@` 輸入提示 UI**：輸入框鍵入 `@` 後彈出成員選擇 dropdown（模糊搜尋成員名稱）；`@h` 顯示 `@here`、`@e` 顯示 `@everyone` 特殊選項；選擇後插入 `<@{userID}>` / `@here` / `@everyone`
- [ ] **訊息 Mention 渲染**：訊息中的 `<@{userID}>` 轉換為 `@username` 高亮 chip 樣式；`@here` / `@everyone` 顯示特殊配色 badge；提及到當前登入用戶時整行訊息加背景色標示
- [ ] **`@mention` 收件匣（延後）**：個人 Inbox 頁面列出所有含自己 mention 的歷史訊息（依時間排序）

---

## 🟣 Translation Plan — 全句翻譯 + LLM 語意猜測

> 詳細設計見 [`translation-plan.md`](./translation-plan.md)（已更新）

> **架構決策**：字典 MVP 移除，改為 DeepL 整句翻譯（三語全存）；猜測評估直接使用 LLM 語意相似度（Gemini 1.5 Flash / Groq），跳過字典模式；儲存層改用 PostgreSQL `message_translations` 表（schema 設計對未來 Cassandra 遷移友好：每筆訊息一列，無 JOIN）；字典單字 hover 功能延後，屆時串接公開 API（Jisho / CC-CEDICT / dictionaryapi.dev）。

### DB / 儲存層
- [x] **`message_translations` 表（PostgreSQL）**：`message_id PK`、`original_lang`、`content_zh`、`content_ja`、`content_en`、`translation_status`、`translated_at`
- [x] **`game_states` 表（PostgreSQL）**：`message_id`、`guesser_id`、`hidden_lang`、`guess_content`、`mode`、`is_correct`、`similarity_score`、`guessed_at`
- [x] **`users.preferred_lang`**：新增 `preferred_lang VARCHAR(5) DEFAULT 'zh'`，`PATCH /api/v1/users/me` 支援更新

### Translation Service（`internal/service/translation_service.go`）
- [x] **DeepL API 整合**：`TranslateAll()` 並行翻譯三語（ZH / JA / EN-US），DeepL 自動偵測來源語言
- [x] **非同步翻譯流程**：`MessageService.CreateMessage()` 寫 DB 後 `go translationService.TranslateAndPush()`，翻譯結果寫入 `message_translations`
- [x] **WS Push `translation_ready`**：翻譯完成後 `BroadcastToChannel`，payload: `{ message_id, original_lang, translations: {zh, ja, en} }`
- [x] **Config**：`deepl.api_key`、`deepl.enabled`；`llm.provider`、`llm.api_key`、`llm.model`、`llm.similarity_threshold`（預設 0.70）、`llm.enabled`

### Guess Service（`internal/service/guess_service.go`）
- [x] **LLM 語意評估**：呼叫 Gemini 1.5 Flash 或 Groq + Llama，score ≥ threshold = 正確
- [x] **`POST /api/v1/messages/{id}/guess`**：body `{ guess_content, hidden_lang }`，回傳 `{ is_correct, similarity_score, correct_content }`
- [x] **`GET /api/v1/messages/{id}/game`**：query `?hidden_lang=zh`，回傳 `{ has_guessed, is_correct, hidden_lang, similarity_score }`
- [x] **`GET /api/v1/messages/{id}/translation`**：取得翻譯結果（三語 + status）

### Repository 層
- [x] **`internal/repository/translation_repository.go`**：`Upsert`、`GetByMessageID`
- [x] **`internal/repository/game_state_repository.go`**：`Create`、`GetByMessageAndGuesser`、`ListByMessage`

### 前端
- [x] **多行輸入 (Shift+Enter 換行)**：input 改為 textarea，自動伸縮高度，Enter 送出 / Shift+Enter 換行
- [x] **基礎 Markdown 渲染**：訊息顯示支援 ` ``` ` 程式碼區塊、`` ` `` 行內程式碼、`-` 列舉，以及換行顯示
- [x] **完整 Markdown 支援**：`**bold**`、`*italic*`、`>` blockquote、`### heading`、連結（可點擊 + OGP 預覽卡片）
- [x] **使用者語言偏好 UI**：Profile 設定頁加 `preferred_lang` 選擇器（zh/ja/en），呼叫 `PATCH /api/v1/users/me`
- [x] **「翻譯載入中」UI 狀態**：收到訊息後顯示 loading，等待 `translation_ready` WS 事件
- [x] **依偏好語言顯示譯文**：`translation_ready` 後依 user `preferred_lang` 自動顯示對應譯文
- [x] **隱藏原文 / 譯文 toggle UI**：讓用戶選擇隱藏哪一側（觸發猜測流程）
- [x] **猜字輸入 UI**：顯示猜測輸入框，送出後呼叫 `POST /api/v1/messages/{id}/guess`，回饋正確/錯誤 + similarity score

### 延後事項
- [ ] **字典單字 hover**：前端 hover 單字 → `GET /api/v1/dictionary/{lang}/{word}`（後端 proxy 公開 API：Jisho / CC-CEDICT / dictionaryapi.dev）

### i18n 介面在地化（新增）
- [x] **i18n 基礎骨架**：前端加入 `vue-i18n`、建立 `web/src/i18n/index.js`，在 `main.js` 掛載 plugin
- [x] **語系檔初始化**：建立 `zh` / `zh-tw` / `ja` / `en` locale 檔（先覆蓋 Settings 模組）
- [x] **設定頁接入 i18n**：`UserSettingsModal.vue` 文案改為 i18n key，新增「介面語言」切換（localStorage 持久化）
- [x] **聊天室文案 i18n 化**：`MessageItem.vue`、通知文案與常用按鈕改用 i18n key
- [x] **語言偏好拆分**：新增 `ui_locale`（介面語言）與 `preferred_lang`（翻譯目標語言）分離，補 API + migration
- [ ] **i18n 測試與檢查**：新增 key 覆蓋檢查與語言切換 smoke test，納入 `make check`

---


## 🟡 Phase 2 — MQ 整合（中期）

### NATS JetStream（對應架構圖 Kafka，等效實作）
- [ ] **引入 NATS client**：`pkg/mq/` 封裝 Publish / Subscribe
- [ ] **`topic:record` Stream**：Chat Server 發送訊息時永遠 publish，供 Message Persistence Consumer 消費寫 DB
- [ ] **`topic:server.{id}` Topic**：Chat Server 收到目標 user 在另一 server（online）時，publish 到對應 topic
- [ ] **`topic:notification` Topic**：目標 user offline（`IsUserOnline` 為 false）時 publish，供 Notification Service 消費
- [ ] **Message Persistence Consumer**：獨立 goroutine（或獨立 binary），consume `topic:record` → 批次寫入 DB，同時非同步派發翻譯任務給 Translation Service
- [ ] **跨 Server 訊息路由**：Chat Server subscribe 自身 topic，收到後透過 WS Manager 推給 client
- [ ] **Notification Service 骨架**：consume `topic:notification` → 呼叫 Push Gateway（FCM/APNs）→ 寫通知記錄至 DB
- [ ] **`last_read_message_id` schema**：已由 Phase 1「未讀 & Mention 系統」的 `channel_read_states` 表涵蓋，MQ 完成後直接沿用同一表

### 服務拆分（Binary 層）
- [ ] **ws-gateway binary**：僅處理 WebSocket 連線管理，與 api-server 分開
- [ ] **api-server binary**：僅處理 REST API，不含 WS 邏輯
- [ ] Docker Compose 分別啟動兩個服務，設定不同 port

---

## 🟠 Phase 3 — File Service & Voice（中期）

### File Access Service（`/api/v1/files`）
- [x] **Minio 整合**：`pkg/storage/` 封裝 Minio SDK，config 加入 Minio 設定
- [x] **DB migration**：`files` 表、`message_attachments` 表
- [x] **檔案類型限制**：允許圖片與檔案上傳，並以副檔名限制可接受類型（`.jpg/.png/.gif/.webp/.svg` 等圖片；`.pdf/.zip/.mp4` 等檔案）
- [x] **上傳次數/大小限制**：per-user quota（Redis 優先 + DB fallback）：單檔最大 MB、每日上傳次數、每日總量 MB 上限
- [x] `POST /api/v1/files/presign`：生成 Upload Pre-signed URL（15 min 有效）
- [x] `POST /api/v1/files/{id}/confirm`：客戶端上傳完成後呼叫，驗證物件存在於 Minio 後更新 status → active
- [x] `GET /api/v1/files/{id}`：取得檔案 metadata
- [x] `GET /api/v1/files/{id}/url`：生成 Download Pre-signed URL（私有檔案，更新 last_accessed_at）
- [x] `DELETE /api/v1/files/{id}`：刪除檔案（Minio + DB）
- [x] **過期與清理策略**：`expires_at` TTL 欄位 + `CleanupExpired()` 定期清理；`last_accessed_at` LRU 欄位 + `FindLRUByUser()` 供清理呼叫
- [ ] **WS file 訊息類型**：`send_message` op 支援 `type: "file"`，帶 `file_id`
- [ ] **LRU background job**：定時呼叫 `CleanupExpired()` + 超量用戶 LRU 清理 goroutine

### LiveKit 語音整合
- [x] **引入 LiveKit SDK**：`pkg/voice/` 封裝 token 生成
- [x] `GET /api/v1/channels/{id}/voice/token`：回傳 LiveKit Room Token + URL
- [x] **WS `voice_state_update` op**：廣播使用者加入/離開語音頻道事件
- [x] Docker Compose 加入 LiveKit Server

---

## ⚪ Phase 4 — 微服務拆分（長期）

- [ ] **Auth Service 獨立**：獨立 repo / binary，其他服務透過 gRPC 或 JWT 驗證
- [ ] **Message Persistence Service 獨立**：獨立部署，只消費 MQ
- [ ] **Translation Service 獨立**：獨立部署，接收翻譯任務，連接 Cassandra
- [ ] **Notification Service 獨立**：獨立部署，消費 `topic:notification`，連接 Push Gateway
- [ ] **File Access Service 獨立**：獨立 Deployment
- [ ] **API Gateway**：Nginx / Traefik 路由 `ws.talkrealm.com` 和 `api.talkrealm.com`
- [ ] **K8s HPA**：Chat Server 水平擴展，依連線數 autoscale
- [ ] **Observability**：Prometheus metrics + Grafana dashboard + OpenTelemetry tracing

---

## 🧪 測試 & 品質

- [x] **Unit Tests**：Handler / Service / Repository 各層覆蓋率 > 70%
- [ ] **WS Integration Test**：`gorilla/websocket` 模擬 client，測試 identify → send_message → message_create 流程
- [ ] **Load Test**：k6 模擬 1000 concurrent WS 連線
- [ ] **E2E Test**：docker-compose 啟動全服務，API 端到端測試

---

## 🛡️ Config 安全防護（源自 2026-05-09 production fatal panic — 重複 config key）

> 根本原因：YAML 重複 key 導致 viper 解析出非預期值，服務啟動後存取 nil 欄位引發 panic。

- [ ] **Config Struct Validation**：`Load()` 完成 `viper.Unmarshal` 後，加入 `go-playground/validator` 驗證（`validate:"required"` 標記必填欄位如 `jwt.secret`、`database.host`）；驗證失敗改為 `log.Fatal` 印出清楚錯誤並 `os.Exit(1)`，而非讓 nil pointer 在深層 panic
- [ ] **Strict YAML Parser（啟動期 fail-fast）**：`config.Load()` 改用 `yaml.v3` + `KnownFields(true)`（或等效 strict 模式）先 decode 到中繼 struct，再交由 viper；若出現 duplicate key / unknown field 立即回錯並中止啟動
- [ ] **YAML Duplicate Key 偵測**：CI pipeline（GitHub Actions）加入 `yamllint -d "{rules: {key-duplicates: enable}}"` 步驟，覆蓋 `configs/*.yaml`；重複 key 時 PR 無法合併
- [ ] **Config Schema 驗證（CI）**：新增 JSON Schema（或 CUE）描述 `configs/*.yaml` 結構與約束（必填欄位、數值範圍、enum）；CI 必跑 schema validation
- [ ] **`make config-check` dry-run 指令**：新增 Makefile target，執行 `go run ./scripts/config_check.go`，載入當前 config 並 pretty-print parsed struct，部署前人工確認
- [ ] **Binary `--validate-config` 模式**：`cmd/server/main.go` 支援 `--validate-config`，僅載入/解析/驗證 config，不連 DB、不啟 HTTP；失敗回傳 exit code 1，供本機與 CI 使用
- [ ] **CI 執行容器化驗證**：在 build 後增加 `docker run --rm <image> --validate-config`，確保「實際 binary + config」在部署前可啟動通過驗證
- [ ] **Panic 最後防線**：在 `main()` 最外層加 `defer recover()`，統一輸出含 stack trace 與 config 檔路徑/版本資訊的 structured log，避免只留下難讀 panic
- [ ] **Staging 環境 Overlay**：在 `deploy/k8s/overlays/` 加入 `staging` overlay，prod 部署前必須先在 staging 驗證 config 與服務啟動正常，再手動 promote
- [ ] **Progressive Delivery**：prod 改採 rolling/canary（先 1 pod 或 5% 流量），新版本健康檢查連續失敗時自動停止 rollout
- [ ] **Health Check + Auto-Rollback**：K8s readiness/liveness 與部署策略綁定，啟動 panic 導致 unhealthy 時自動回退上一版，縮小爆炸半徑
- [ ] **Config 變更四眼原則**：`configs/*` 強制 PR + 至少 1 名 reviewer approve（CODEOWNERS + branch protection），禁止直接 push 主分支

---

## 🏗️ 基礎架構 & 部署

- [x] Dockerfile 與服務打包
- [x] Docker Compose 本地環境（Postgres + Redis）
- [x] Kubernetes Kustomize Base & Overlays
- [ ] **Docker Compose 加入 NATS**：`nats:latest` with JetStream enabled
- [x] **Docker Compose 加入 Minio**：`minio/minio` latest（dev + prod compose；K8s base + local/dev overlays）
- [x] **Docker Compose 加入 LiveKit**：`livekit/livekit-server`
- [ ] **CI/CD**：GitHub Actions — lint + test + build + push Docker image
- [ ] **K8s Secrets 管理**：使用 Sealed Secrets 或 External Secrets Operator

---

## 📚 文件

- [x] OpenAPI / Swagger 定義（`docs/openapi/`）
- [ ] **更新 Swagger**：補充所有新 API（files、invites、voice token、cursor pagination）
- [ ] **WebSocket Protocol 文件**：所有 op codes 的 request/response schema
- [ ] **架構圖更新**：用 draw.io 或 Mermaid 繪製新架構圖並放入 `docs/architecture.md`
- [ ] **SDK 生成**：嘗試 [microsoft/kiota](https://github.com/microsoft/kiota) 從 OpenAPI 生成 Go / TS SDK

---

## ✅ 已完成

- [x] 使用者系統（註冊、登入、JWT）
- [x] Guild CRUD + 成員管理
- [x] Channel CRUD
- [x] 基礎訊息 CRUD（REST）
- [x] WebSocket 連線與基礎廣播
- [x] WS identify / hello / ready / heartbeat / presence_update
- [x] WS typing_start
- [x] WS subscribe / unsubscribe channel
- [x] Redis user server mapping + guild online set
- [x] Redis Heartbeat TTL 刷新 + `IsUserOnline`
- [x] Rate Limiting middleware（Redis counter）
- [x] Cursor-based message pagination
- [x] Guild 邀請碼系統
- [x] Refresh Token + logout
- [x] RBAC `RequireGuildRole` middleware
- [x] Docker Compose 本地開發環境
- [x] Kubernetes 部署配置
- [x] OpenAPI 基礎定義
