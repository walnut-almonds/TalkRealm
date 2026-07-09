# TalkRealm — Agent Memory

## Quick Facts
- Module: `github.com/walnut-almonds/talkrealm`
- Go version: 1.26.1
- Web framework: Gin v1.10.0
- ORM: GORM + PostgreSQL (`gorm.io/driver/postgres`)
- WebSocket: `gorilla/websocket`
- Auth: `golang-jwt/jwt/v5`
- OAuth: `golang.org/x/oauth2` (Google OAuth 已實作)
- Config: Viper
- Logger: `go.uber.org/zap`
- 目前是 **monolith**，架構目標是漸進拆分為微服務（見 `plan.md`）

## Commands
```bash
make check        # 全部檢查（lint + build + test）
cd web && npm run check:i18n  # 掃描 t/$t key 使用並檢查 locale key 完整性
```

- **免後端視覺驗證**：`npx vite --port 5199` 起 dev server 後，用 chrome-devtools 的 `navigate_page` + `initScript` 攔截 `window.fetch`（mock `/api/v1/*` 回應）並塞 `talkrealm_token`/`talkrealm_last_guild`/`talkrealm_last_channel` 進 localStorage，即可渲染 Galaxy 首頁與聊天主畫面截圖。API 形狀：`{user}`, `{guilds}`, `{channels}`, `{members}`, `{messages}`（見 `web/src/api/index.js` 的 `EP`）。注意 reload 會失去 initScript，需重新 navigate；Windows 下背景 vite 停止後 port 可能殘留，需 `taskkill //PID`。

## Architecture Notes
- `internal/server/server.go`：DI 組裝、路由設定的主入口
- `internal/websocket/manager.go`：channel 訂閱索引（`channelSubscriptions map[uint]map[*Client]bool`）+ guild 訂閱索引（`guildSubscriptions map[uint]map[*Client]bool`），O(1) 廣播；jwtManager 注入用於 identify op；identify 後自動呼叫 `SubscribeClientToUserGuilds` 訂閱所有 guild
- WS 協議：client→server op: `identify`, `heartbeat`, `subscribe`, `unsubscribe`, `typing_start`, `send_message`, `voice_state_update`；server→client op: `hello`, `ready`, `heartbeat_ack`, `message_create`, `message_update`, `message_delete`, `typing_start`, `presence_update`, `error`, `guild_update`, `guild_delete`, `guild_member_add`, `guild_member_remove`, `guild_member_update`, `channel_create`, `channel_update`, `channel_delete`, `voice_state_update`
- WS 端點：`GET /api/v1/ws`（無需 JWT 中間件，由 identify op 驗證）
- identify flow：client 連線 → server 送 `hello`（heartbeat_interval=30000ms）→ client 送 `identify`（token + channels[]）→ server 驗證 JWT，送 `ready` + 廣播 `presence_update online`
- `pkg/auth/jwt.go`：JWTManager，sign / verify token
- `pkg/database/database.go`：GORM DB singleton
- REST API 路由前綴：`/api/v1/`
- WebSocket 端點：`GET /api/v1/ws`（token 透過 identify op 傳遞，不再放 query string）
- 目前訊息分頁是 offset，計畫改為 cursor-based（before message_id）

## Pitfalls
- **OG 預覽對非 HTML 連結**：`GET /api/v1/og` 遇到 `image/*` 或其他非 `text/html` 內容時，現在改為回 `200`（image 會帶最小預覽 `{image:url}`，其他類型回空 OG）而非 `422`。可避免前端在訊息含 CDN 圖片連結（例如 `googleusercontent` avatar URL）時持續出現 console `Unprocessable Content`。
- **Tenor key 相容性**：`LIVDSRZULELA` 在 Tenor v2 (`tenor.googleapis.com/v2`) 目前會回 `API_KEY_INVALID`（400），但在 v1 (`g.tenor.com/v1`) 仍可用。前端 `searchGIFs` 應採「有 `VITE_TENOR_API_KEY` 才走 v2，否則/失敗 fallback v1」策略，避免 GIF picker 直接壞掉。
- DM 與群組訊息共用 `MessageItem` 時，編輯/刪除/翻譯 API 不能固定呼叫 `/messages/:id/*`；DM 需要走 `/dm/messages/:id/*`。建議以 `isDM` prop 分流，否則 DM 會出現 404/權限錯誤。
- Vue SFC 大改版時要避免「新版內容 + 舊版內容同檔重複貼上」；會造成 `<script>/<template>/<style>` 區塊重複、前端編譯直接失敗。
- DM 與群組訊息整合後，後端 `message_create` payload 主要欄位是 `channel_id`（不再保證有 `dm_channel_id`）。前端 DM store 若仍只讀 `dm_channel_id`，會導致私訊新訊息不顯示、頻道排序不更新。
- 歷史資料庫若 `channels.guild_id` 仍是 `NOT NULL`，建立 DM 頻道（`guild_id=NULL`）會噴 `SQLSTATE 23502`。`AutoMigrate` 不一定會自動放寬 constraint，需顯式執行 `ALTER TABLE channels ALTER COLUMN guild_id DROP NOT NULL`（已在 `pkg/database/database.go` 的 migration patch 內處理）。
- 前端聊天室 `renderMessages()` 會在每次新訊息時重繪整個訊息區；若圖片附件每次都重新呼叫 `getFileDownloadUrl`（pre-signed URL），會導致「每發話一次就重新下載歷史圖片」。已在 `web/js/app.js` 加入 `attachmentImageURLCache` 與 in-flight 去重，優先重用既有 URL，並在圖片 URL 過期時僅重抓一次。
- **File routes 404**：`/api/v1/files/*` 路由只在 Minio 初始化成功時才掛載。Minio 未設定或連線失敗會導致 `fileHandler == nil`，所有 file API 回傳 404 而非 503。已改為無條件掛載路由，Minio 不可用時回傳 503。若遇 404，先確認 Minio 容器是否正常運行及環境變數（`MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY`、`MINIO_BUCKET`）是否設定正確。
- WS Manager 已有 channel subscription index（Phase 1 完成）；Presence 系統目前無 Redis（狀態不持久化）
- `message_service.go` 中 WS Manager 以 interface 注入（避免循環依賴），需 `SetWebSocketManager()` 設定；另有 `CreateMessageWS()` 供 WS `send_message` op 呼叫（`MessageSender` interface 注入到 Manager）
- handler.go 仍有 TODO stub functions（已被 user_handler.go 等各自的 handler 取代）
- 部署到 VPS 使用 `docker-compose.prod.yml` 時，`POSTGRES_PASSWORD` / `REDIS_PASSWORD` 需由同目錄 `.env` 或 `--env-file` 提供；否則 Compose 會以空字串替代，造成 postgres healthcheck 失敗。
- **Presence（在線狀態）架構**：Redis 為唯一「是否在線」的判斷依據；DB `User.Status` 保留為使用者自選狀態偏好（offline/busy/away）。
  - `user_service.Login` / `OAuthLoginOrRegister` **不再** 自動寫 `online` 到 DB。
  - WS `handleIdentify` 只做 Redis 寫入（`redisOnIdentify`）；`handleUnregister` 只做 Redis 清理（`redisOnDisconnect`）+ 廣播，不碰 DB。
  - 多標籤頁修正已實裝：`handleUnregister` 先掃描 `hasOtherConnections`，只在最後一個連線斷開時才執行 Redis 清理與廣播。
  - `handler.GuildHandler` 新增 `OnlineChecker` interface + `SetOnlineChecker` setter；`ListGuildMembers` 回傳前以 `IsUserOnline` 動態覆寫狀態為 `"online"`（若 Redis 確認在線）。
  - `server.go` 以 `guildHandler.SetOnlineChecker(wsManager)` 注入；不再有 `SetUserStatusUpdater`。
  - `UpdateStatus` 方法（repo/service）仍保留，供使用者透過 REST 設定 busy/away 偏好用途。
- `golangci-lint --fix` + `whole-files: true` 坑：修改 `mocks.go` 會曝露所有既有的 nilnil 問題。已用 `//nolint:nilnil` 全部標記。新增 mock 方法必須一同加標記。同理：改到舊測試檔會曝露整檔既有 noctx（`httptest.NewRequest`）— 修法是換成 `httptest.NewRequestWithContext(t.Context(), ...)`（guild_handler_test.go 已全數改完）。
- `golangci-lint --fix` 會重新格式化 oauth_handler.go，造成 `NewRequestWithContext` 行號改變；`wsl_v5` 需在 `if err != nil { c.JSON(); return }` 的 return 前加空行。
- `wsl_v5` 在 service 邏輯中也會要求 guard-return 後與下一個賦值語句之間保留空行（例如 `if channel.GuildID == nil { return ... }` 之後的 `member, err := ...`），否則會報 `missing whitespace above this line`。
- 前端拖曳檔案判斷不可只用 `e.dataTransfer.types.includes('Files')`：Safari/部分瀏覽器 `types` 是 `DOMStringList`，需改用 `types.contains('Files')` 或 `Array.from(types).includes('Files')`；另外要在 `window.dragover` `preventDefault()`，避免瀏覽器直接開啟拖入檔案。
- 若部署使用 `docker-compose.prod.yml`，必須包含 `livekit` service（`livekit:7880` 供 nginx upstream 轉發）。缺少該容器會導致 `wss://voice.../rtc/v1` 連線失敗，前端可能同時看到 `/rtc/v1/validate` CORS 錯誤（實際上常是 upstream 不可達）。
- LiveKit `--keys` 參數格式必須是 **`"key: secret"`**（冒號後必須有空白）。在 compose 建議整段 `command` 用單引號包住，避免 YAML 把 `:` 誤判為 mapping。
- 前端 i18n 新增大量 key 時，`web/src/i18n/locales/zh.js`、`web/src/i18n/locales/zh-tw.js`、`web/src/i18n/locales/ja.js` 已改為 `import en from './en.js'` 並用 `...en` + 分區覆寫，避免缺 key 時大規模漏翻造成 runtime 噪音。
- `check:i18n`（`web/scripts/check-i18n-keys.mjs`）用 regex 抓 `t('...')` 字面 key：模板字串動態 key 如 ``t(`learn.tier${tv}`)`` 會被當成字面 key 而 fail。動態 key 要放進變數/查表再傳給 `t()`（regex 只在 `t(` 後緊接引號時匹配）。另：未使用的 key 不會報錯，只有 used-but-missing-in-en 會 fail。
- **Windows 開發環境**：`.tool-versions` 的 swag 需用 `go:github.com/swaggo/swag/cmd/swag` backend（aqua backend 不支援 windows）；Makefile 的 setup scripts 需以 `bash ./scripts/...` 呼叫（直接執行 `.sh` 會被 Windows 丟給 WSL）；`go test -race` 需 cgo + gcc，Windows 無 gcc 時 Makefile 以 `ifeq ($(OS),Windows_NT)` 跳過 `-race`。mise reshim 在 claude 執行中會因 claude.exe shim 被鎖而整批失敗；缺 shim 時可直接複製任一既有 shim（全是同一顆通用 exe，靠檔名辨識）：`cp shims/go.exe shims/<tool>.exe`（已補 golangci-lint/swag/kubectl/k9s）。
- **Status 顯示規則（invisible/idle/dnd）**：`Status` 欄位是使用者自選偏好；對「其他人」顯示時 invisible 一律映射為 offline（`ListGuildMembers` 的 switch、`user_service.publicStatus()`）。WS identify 廣播 presence 時經 `Manager.userLookup`（`SetUserLookup(userRepo)` 注入）查偏好：invisible 不廣播、idle/dnd/busy/away 廣播自選值。前端 `handleUserStatus` 只有收到 `offline` 才從 `onlineUserIds` 移除。已知限制：透過 REST 改 status 不會即時廣播 presence，需等下次成員清單載入。注意：message/friendship 等 Preload("User") 的 JSON 仍會帶原始 status（含 invisible），尚未清洗。

- **手機版左側抽屜（Discord-style）**：nav-rail 在聊天頁（DOM 有 `.channels-sidebar` 或 `.dm-sidebar` 時）透過 `main.css` mobile 區塊的 `.app-shell:has(...)` 規則變 fixed off-canvas，與 sidebar 一起滑入（sidebar `left:56px`、closed transform 是 `translateX(calc(-100% - 56px))`）；HomeView 無 sidebar 時 rail 留在 flow 內。注意 mobile 樣式分兩處：`channels-sidebar`/`members-sidebar` 在 `main.css`，`dm-sidebar` 在 `DMSidebar.vue` 的 scoped style，改抽屜行為要兩邊同步。

## Decisions
- **前端視覺系統：Kinetic Noir（TalkRealm Edition）**，規範見根目錄 `DESIGN.md`（改編自 walnut-almonds.github.io 的同名系統）。要點：近黑 surface 階梯（#0e0e0e→#2a2a2a）、唯一裝飾色 slate-blue `--accent: #b3c6f3`、直角（`--radius: 0px`；頭像/presence 圓點例外——「人=圓、地方=方」）、1px hairline 取代陰影、Geist Mono 做系統性文字（分類標題/時間戳/徽章）、按鈕 hover 即時反白。tokens 在 `web/src/styles/main.css` `:root`（`--accent`/`--accent-hover`/`--brand` 已定義，元件的 var() fallback 不再吃到 Discord 色）；字體在 `web/index.html` 載入（Hanken Grotesk + Noto Sans TC + Geist Mono）。`web/css/styles.css` 是 pre-Vue 舊版，未套用新主題。新樣式禁用 Discord 特徵：blurple、圓→方 morph、紫色漸層。Social Galaxy 首頁（`web/src/views/HomeView.vue`，SVG 實作）已同步換色：`GUILD_PALETTE` 8 色是去飽和「noir 星座」色系、星雲/時段氛圍（data-atmosphere day/night/dawn/dusk）漸層降飽和；新增 guild 色一律走 muted pastel，不可回填飽和色。
- MQ 選擇 NATS JetStream（輕量，適合小團隊），備選 Kafka
- 物件儲存選 Minio（self-hosted S3-compatible），生產可換 AWS S3
- 語音選 LiveKit（WebRTC SFU）
- 檔案上傳採 Pre-signed URL 模式，API Server 不處理 binary

## Last Updated
2026-07-08
 — 手機版左側抽屜改為 Discord-style（nav-rail 併入抽屜，見 Pitfalls）

2026-07-06
 — 前端視覺系統改版為 Kinetic Noir（見 Decisions 與 `DESIGN.md`）

2026-07-03
 — Windows 開發環境修正（swag go backend、Makefile bash 呼叫、-race 條件跳過）；invisible/idle/dnd 狀態顯示規則實裝（詳見 Pitfalls）

2026-06-15
 — 使用者語言偏好已拆分：`users.ui_locale`（介面語言）與 `users.preferred_lang`（訊息翻譯目標語言）分離；前端 `UserSettingsModal` 會同時送出兩者，`useAppStore.loadUserData()` 以 `ui_locale` 設定 i18n locale
 — i18n 規則：未設定 `ui_locale` 時，前端以 `navigator.languages` 順序決定初始語言（中文依繁簡/地區判斷：`hant|tw|hk|mo -> zh-tw`，`hans|cn|sg -> zh`，其餘 zh 預設簡體）；缺少翻譯 key 時 fallback 到英文

2026-06-09 — 整理 MEMORY.md 與 AGENTS.md 格式；移除逐次 changelog（技術要點已收入 Pitfalls / Architecture Notes）
