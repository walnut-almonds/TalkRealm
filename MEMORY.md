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
```

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
- `golangci-lint --fix` + `whole-files: true` 坑：修改 `mocks.go` 會曝露所有既有的 nilnil 問題。已用 `//nolint:nilnil` 全部標記。新增 mock 方法必須一同加標記。
- `golangci-lint --fix` 會重新格式化 oauth_handler.go，造成 `NewRequestWithContext` 行號改變；`wsl_v5` 需在 `if err != nil { c.JSON(); return }` 的 return 前加空行。
- `wsl_v5` 在 service 邏輯中也會要求 guard-return 後與下一個賦值語句之間保留空行（例如 `if channel.GuildID == nil { return ... }` 之後的 `member, err := ...`），否則會報 `missing whitespace above this line`。
- 前端拖曳檔案判斷不可只用 `e.dataTransfer.types.includes('Files')`：Safari/部分瀏覽器 `types` 是 `DOMStringList`，需改用 `types.contains('Files')` 或 `Array.from(types).includes('Files')`；另外要在 `window.dragover` `preventDefault()`，避免瀏覽器直接開啟拖入檔案。
- 若部署使用 `docker-compose.prod.yml`，必須包含 `livekit` service（`livekit:7880` 供 nginx upstream 轉發）。缺少該容器會導致 `wss://voice.../rtc/v1` 連線失敗，前端可能同時看到 `/rtc/v1/validate` CORS 錯誤（實際上常是 upstream 不可達）。
- LiveKit `--keys` 參數格式必須是 **`"key: secret"`**（冒號後必須有空白）。在 compose 建議整段 `command` 用單引號包住，避免 YAML 把 `:` 誤判為 mapping。

## Decisions
- MQ 選擇 NATS JetStream（輕量，適合小團隊），備選 Kafka
- 物件儲存選 Minio（self-hosted S3-compatible），生產可換 AWS S3
- 語音選 LiveKit（WebRTC SFU）
- 檔案上傳採 Pre-signed URL 模式，API Server 不處理 binary

## Last Updated
2026-05-30 — 未讀 & Mention 系統完整實裝（make check 通過）：
- **後端**：`model.ChannelReadState`/`model.MessageMention` + AutoMigrate；`ChannelReadStateRepository`/`MessageMentionRepository` 完整 CRUD；`UnreadService`（AckChannel/GetChannelUnread/GetAllUnread）；`UnreadHandler`（GET /me/unread、GET /channels/:id/unread、POST /channels/:id/ack）
- **`@here` 線上過濾**：`WebSocketManager` interface 新增 `IsUserOnline(userID uint) bool`；`parseMentions` 在 `hasHere` 時跳過離線成員
- **DM 未讀支援**：`GetAllUnread` 除 guild text channel 外，也 JOIN `channel_participants` 取得 DM channel；`DMSidebar.vue` 顯示 `badge-mention`/`badge-unread`；`openDMChannel` 載入完訊息後呼叫 `ackChannel` API
- **前端 DM store**：`pushIncomingDM` 當非當前頻道時增加 `channelUnreadMap` 計數
- **Bug fixes**：`handleNewMessage` 移除多餘的雙層 `if` 判斷；`MockMessageMentionRepository`、`MockChannelReadStateRepository`、`MockUnreadService` 加入 `internal/testutil/mocks.go`
- **陷阱**：mock 中返回 `nil, nil` 對於返回 slice/pointer 的函數會觸發 `nilnil` linter；改用 `[]T{}` 或 `&T{}` 即可；`prealloc` 建議為 slice 宣告時給 capacity

2026-05-29 — Social Galaxy 增強功能實裝（HomeView.vue）：
- **`web/src/views/HomeView.vue`**：
  - 新增 `useVoiceStore` import；新增 `loadGuildChannels()`（guildId→Set\<channelId\> map）
  - `typingGuildIds` computed：透過 `store.typingUsers` + `guildChannelMap` 計算有人打字的 guild
  - `voiceGuildIds` computed：透過 `voiceStore.voiceParticipants` + `guildChannelMap` 計算有語音的 guild
  - `flashGuildIds` reactive Set：watcher on `store.unreadGuildIds`，新增 guild 時觸發 1.4 s flash
  - `harmonicNoise()` + 動畫迴圈 noise drift（收斂後對所有 non-fixed 節點加三諧波有機漂移）
  - `timeAtmosphere` ref + `updateAtmosphere()`：根據小時設定 night/dawn/dusk/day，`data-atmosphere` 屬性驅動 CSS
  - `focusedNodeId` + `smoothPan()` + `focusOnNode()` + `resetFocus()`：點擊節點 cubic ease-out 縮放聚焦，背景點擊復原
  - Template：guild 節點新增 typing ripple / voice pulse / message flash 圓圈；friend 節點新增 online breathing glow；opacity dimming for unfocused nodes
  - Tooltip：新增 `isTyping` / `hasVoice` badges
  - CSS：`sg-typing-ripple`、`sg-voice-pulse`、`sg-msg-flash`、`sg-friend-breathe`、atmosphere 大氣層 filters、opacity transition

2026-05-27 — DM/群組訊息整合收斂：補上 DM 翻譯讀取授權檢查、修正 WebSocket `msgSender` 欄位名、前端 DM store 移除重複定義並相容 `channel_id`。
2026-05-22 — Presence 系統改為 Redis-only 架構（移除 ghost-online bug）：
- **`web/src/api/index.js`**：新增 `guildLastChannel` helper（`get(guildId)`/`set(guildId, channelId)`），使用 `talkrealm_last_channel_{guildId}` 作為 localStorage key，實現 per-guild 頻道記憶
- **`web/src/stores/useAppStore.js`**：`selectGuild()` 在重置 `currentChannel`/`messages` 後，自動 `await selectChannel()` 至最後停留頻道（fallback 至第一個文字頻道）；`selectChannel()` 改為呼叫 `guildLastChannel.set()` 而非全域 `LAST_CHANNEL`；`loadUserData()` 移除重複的頻道恢復邏輯（由 `selectGuild` 統一處理）
- **`web/src/components/GuildSidebar.vue`**：首頁按鈕改呼叫 `goHome()` 函數，同時清空 `currentGuild`、`currentChannel`、`messages`
2026-05-11 — 語音視訊 UX 強化（畫質/FPS 控制 + 音量滑桿 + 加入前參與者預覽）：
- **`internal/websocket/manager.go`**：新增 `voiceParticipants map[uint]map[uint]string`（channelID→userID→username），初始化於 `NewManager`；新增 `UpsertVoiceParticipant`、`RemoveVoiceParticipant`、`GetVoiceParticipants` 方法（受 `m.mu` 保護）
- **`internal/websocket/client.go`**：`handleVoiceStateUpdate` join/leave 分別呼叫 `UpsertVoiceParticipant`/`RemoveVoiceParticipant`
- **`internal/handler/voice_handler.go`**：新增 `VoiceParticipantsGetter` interface；`VoiceHandler` 加入 `vpGetter`；`NewVoiceHandler(vm, vpg)` 新增第二參數；新增 `GetVoiceParticipants` handler 回傳 `{ participants: [{user_id, username}] }`
- **`internal/server/server.go`**：`NewVoiceHandler(voiceManager, wsManager)`；新增路由 `GET /channels/:id/voice/participants`
- **`web/src/api/index.js`**：新增 `VOICE_PARTICIPANTS` endpoint 與 `getVoiceParticipants(channelId)` 方法
- **`web/src/stores/useVoiceStore.js`**：新增 `participantVolumes`（identity→volume 0..1）、`identityToAudioKeys`（identity→Set）、`videoQuality`（預設 '720p'）、`screenShareFps`（預設 15）；`attachAudioTrack` 套用音量並追蹤 keys；`setParticipantVolume` 即時套用至所有 audio elements；`reset()` 清空兩 Map
- **`web/src/composables/useVoice.js`**：import `VideoPresets`；新增 `CAMERA_PRESETS` map；`toggleCamera`/`toggleScreenShare` 帶 resolution/fps preset；新增 `updateVideoQuality(quality)` 與 `updateScreenFps(fps)` 函數（重啟 track 套用新設定）
- **`web/src/stores/useAppStore.js`**：`selectGuild` 載入頻道後並行呼叫 `getVoiceParticipants` 預填 `voiceParticipants`/`voiceParticipantStates`（加入前預覽）
- **`web/src/components/VoiceVideoOverlay.vue`**：script 改用 `inject('voice')` 模式；新增 `showSettings`、質量/FPS 選項常數、`onQualityChange`/`onFpsChange`/`getVolume`/`setVolume` 函數；template 已含設定面板 `<Transition name="vvo-slide">`、音量滑桿 `.vvo-volume-control`
- **`web/src/styles/main.css`**：`.vvo-panel` 放大至 `min(96vw,1400px)/92vh`；新增 `.vvo-settings`、`.vvo-settings-row`、`.vvo-settings-label`、`.vvo-select`、`.vvo-volume-control`（hover 顯示）、`.vvo-volume-slider`、`.vvo-btn-icon.active`、`vvo-slide` transition 樣式

2026-05-10 — 語音視訊（螢幕分享 + 攝影機）功能：
- **`web/src/stores/useVoiceStore.js`**：`voiceSelfState` 新增 `screenSharing`、`cameraEnabled`；新增 `remoteVideoTracks`（陣列，每項含 trackSid、participantIdentity、kind: `screen|camera`、element、userId、username）與 `videoOverlayOpen`；新增 `addRemoteVideoTrack`、`removeRemoteVideoTrack`、`cleanupVideoTracks` 方法；`reset()` 同步清空 video tracks。
- **`web/src/composables/useVoice.js`**：`TrackSubscribed/Unsubscribed` 同時處理 video tracks；新增 `toggleScreenShare()`（呼叫 `setScreenShareEnabled`）與 `toggleCamera()`（呼叫 `setCameraEnabled`）；`broadcastSelfState` 加入 `screen_sharing`、`camera_enabled`；`handleVoiceData` 反寫參與者狀態並 backfill `remoteVideoTracks` 的 userId/username；`leaveVoiceChannel` 先關閉 screen share & camera 再 disconnect。
- **`web/src/components/VoiceVideoOverlay.vue`**（新增）：Teleport to body 的 modal；自動掛載 remote video track element（`<div ref=...>` + `appendChild`）；自偵測 self camera / screen 並透過 watch + `getTrackPublication` attach；支援 pin（固定某畫面，佔滿整行）；無視訊時顯示 empty state；關閉時不影響 LiveKit 連線。
- **`web/src/components/VoiceBar.vue`**：新增攝影機、螢幕分享、展開視訊視窗按鈕；參與者狀態圖示新增 `fa-display`（螢幕分享中）與 `fa-video`（攝影機開啟中）。
- **`web/src/components/MainLayout.vue`**：引入並掛載 `VoiceVideoOverlay`（全域一個實例）。
- **`web/src/styles/main.css`**：新增 `.voice-bar-toggle.active`、`.voice-video-overlay`、`.vvo-*` 等視訊 overlay 相關樣式。

2026-05-09 — 前端語音 UX 強化（加入/離開提示音 + 狀態同步）：
- **`web/js/app.js`**：新增 `playVoiceNotificationSound()`，收到 WS `voice_state_update` 的 `join/leave`（非自己）時播放輕量提示音。
- **`web/js/app.js`**：語音狀態新增 `voiceSelfState`（`micEnabled` / `deafened`）與 `voiceParticipantStates`；加入 `toggleVoiceMicrophone()` / `toggleVoiceDeafen()`。
- **`web/js/app.js`**：透過 LiveKit `RoomEvent.DataReceived` + `localParticipant.publishData(...)` 同步 `voice_user_state`（麥克風/收音）給同語音房成員，並在語音頻道列表與 voice bar 顯示。
- **`web/index.html` / `web/css/styles.css`**：voice bar 新增麥克風/收音切換按鈕與「目前語音群成員」區塊，顯示每位成員的麥克風與收音圖示狀態。

2026-05-09 — Minio public_read 模式：
- **`pkg/config/config.go`**：`MinioConfig` 新增 `PublicRead bool`（mapstructure: `public_read`，預設 false）
- **`pkg/storage/minio.go`**：`NewClient` 若 `cfg.PublicRead=true` 自動呼叫 `applyPublicReadPolicy` 套用 S3 public-read policy（Allow s3:GetObject *）；新增 `PublicFileURL(key)` 回傳 `{publicEndpoint}/{bucket}/{key}` 永久 URL
- **`internal/service/file_service.go`**：`GetDownloadURL` 若 `public_read=true` 直接回傳永久 URL（expiresIn=-1），跳過 Redis 快取與 presign 流程
- **`internal/handler/file_handler.go`**：`expiresIn<0` 時回應 `Cache-Control: public, max-age=31536000, immutable`
- **`configs/config.example.yaml`**：新增 `public_read: false` 欄位

2026-05-09 — 圖片快取修正（server-side Redis + HTTP Cache-Control）：
- **`internal/service/file_service.go`**：`GetDownloadURL` 回傳 `(string, int, error)`；先查 Redis key `file:dl_url:{fileID}`，命中則回傳同一 URL + 剩餘 TTL 秒數；未命中才呼叫 Minio presign，並以 `(PresignExpiry-2) min` TTL 寫入 Redis。`DeleteFile` 同步 `DEL` 該 key。
- **`internal/handler/file_handler.go`**：`GetDownloadURL` handler 新增 `Cache-Control: private, max-age={expiresIn}` 回應標頭，並在 JSON 加入 `expires_in` 欄位，讓瀏覽器可快取 API 回應。
- 效果：同一 fileID 在 presign 週期內永遠取得相同 URL，瀏覽器圖片快取得以生效；切換頻道返回不再重新下載已載入圖片。

2026-05-08 — 前端圖片重複下載修正：
- **`web/js/app.js`**：圖片附件載入流程加入 URL 快取（`attachmentImageURLCache`）與請求去重（`attachmentImageFetchInFlight`），`renderMessages()` 先用快取 URL，不再每次重繪都重新打 `GET /files/:id/url`；若 URL 過期，`img.onerror` 只觸發一次強制更新，避免無限重試。

2026-05-08 — 前端語音播放修正（LiveKit）：
- **`web/js/app.js`**：加入 `RoomEvent.TrackSubscribed/TrackUnsubscribed/TrackSubscriptionFailed` 監聽，將遠端 `audio` track `attach()` 到隱藏 `<audio>` 元素，離房/斷線時 cleanup；解決「可進同房但聽不到聲音」的主要前端缺口。

2026-05-08 — Voice 連線修正（prod 部署）：
- **`docker-compose.prod.yml`**：新增 `livekit` service（keys 讀取 `${LIVEKIT_API_KEY}:${LIVEKIT_API_SECRET}`、expose `7880`、開放 `7881` 與 `50100-50200/udp`）
- **`docker-compose.prod.yml`**：`nginx` 增加 `depends_on: livekit`
- **`nginx/nginx.conf`**：`location /rtc` CORS header 改為穩定 always 模式（含 `Vary: Origin`、`X-Livekit-*` headers），並保留 `OPTIONS -> 204`

2026-05-08 — LiveKit 語音整合（Phase 3）實作完成：
- **`pkg/voice/token.go`**：`Manager` 封裝 LiveKit token 生成；`GenerateRoomToken(channelID, userID, username)` 以 `channel:{id}` 作為 room name，回傳 `RoomTokenResponse{Token, URL, RoomName, Identity}`
- **`pkg/config/config.go`**：新增 `LiveKitConfig{APIKey, APISecret, URL, PublicURL, TokenTTL}`；`--dev` 模式預設 key=devkey / secret=secret
- **`internal/handler/voice_handler.go`**：`GET /api/v1/channels/:id/voice/token`（需認證）；LiveKit 未設定時回 503
- **WS `voice_state_update` op**（client→server / server→channel）：payload `{channel_id, action: "join"|"leave"}`；廣播格式 `{channel_id, user_id, username, action}`；實作在 `handleVoiceStateUpdate()` 方法
- **`internal/websocket/client.go`**：refactored `handleMessage` — `send_message` 邏輯提取為 `handleSendMessage()`，降低 cognitive complexity；新增 `handleVoiceStateUpdate()`
- **`docker-compose.yml`**：加入 `livekit/livekit-server:latest`（`--dev` 模式）；port 7880/7881/50100-50200(udp)
- **`configs/config.example.yaml`**：新增 livekit 區段（api_key=devkey, api_secret=secret, url=ws://livekit:7880, public_url=ws://localhost:7880, token_ttl=3600）
- **依賴新增**：`github.com/livekit/server-sdk-go/v2`（token 生成用 `github.com/livekit/protocol/auth`）
- **nolint**：`client.go` readPump/writePump 的 `conn.Close/SetReadDeadline/SetWriteDeadline/WriteMessage` 加 `//nolint:errcheck,gosec`（pre-existing pattern）

2026-05-04 — File Access Service（Phase 3）實作完成：
- **前端檔案上傳**：`+` 按鈕 → `<input type="file">` → `handleFileSelected()` → `uploadFile()` (presign→PUT→confirm) → chip 預覽 → `sendMessage()` 帶 `file_ids`
- **`api.js`**：新增 `presignUpload`, `uploadToMinio` (XHR PUT), `confirmUpload`, `getFileDownloadUrl`, `deleteFile`；`sendMessage` 新增 `fileIds` 參數
- **`app.js`**：`appState.pendingFileIds`、`handleFileSelected`、`uploadFile`、chip CRUD、`renderAttachments`（圖片縮圖/檔案下載）、`openAttachment`、`loadAttachmentImage`
- **`CreateMessageRequest`**：新增 `FileIDs []uint`；允許空 Content（有附件時）
- **`MessageService`**：新增 `SetFileService(FileService)`；`CreateMessage` 後建立 `MessageAttachment` 記錄
- **`minio.go` 陷阱**：曾出現重複 `package storage` 宣告（garbled）；用 `/tmp/write_xxx.go + go run` 修復（heredoc 會觸發 tab-completion）
- **`PresignPutURL`**：簽名改為 `(key, contentType string, expiry int)`
- **`internal/model/user.go`**：新增 `File`（含 `status`, `expires_at`, `last_accessed_at`）、`MessageAttachment` 模型
- **`pkg/config/config.go`**：新增 `MinioConfig`（endpoint/bucket/ssl/presign_expiry/max_file_size_mb/daily_upload_max/daily_bytes_max_mb/file_ttl_days/lru_evict_enabled）
- **`internal/repository/file_repository.go`**：CRUD + `CountByUserToday`, `SumBytesByUserToday`, `FindExpired`, `FindLRUByUser`, `TouchLastAccessed`, attachment CRUD
- **`internal/service/file_service.go`**：副檔名白名單驗證、單檔大小限制、每日 quota（Redis pipeline 優先 + DB fallback）、Pre-signed upload 流程（presign→pending→confirm→active）、TTL CleanupExpired
- **`internal/handler/file_handler.go`**：`PresignUpload`, `ConfirmUpload`, `GetFile`, `GetDownloadURL`, `DeleteFile`
- **Routes**：`POST /presign`, `POST /{id}/confirm`, `GET /{id}`, `GET /{id}/url`, `DELETE /{id}` 掛於 `/api/v1/files`（need Auth middleware）
- **DB AutoMigrate**：`File`, `MessageAttachment` 加入 `pkg/database/database.go`
- **`configs/config.example.yaml`**：新增完整 minio 區段
- **依賴新增**：`github.com/minio/minio-go/v7`, `github.com/google/uuid`
- **未完成**：WS `type: "file"` send_message、background LRU cleanup goroutine
- **新增 Notification Service**：獨立服務，消費 `topic:notification`，呼叫 Push Gateway (FCM/APNs)，寫 DB
- **新增 Translation Service**：獨立服務，由 Message Persistence Service 非同步派發，DeepL/GPT-4o，結果存 **Cassandra**（非 PostgreSQL 欄位）
- **MQ Topic 命名統一**：`topic:record`、`topic:server.{id}`、`topic:notification`（對齊圖示；實作選 NATS JetStream，圖示標 Kafka，等效替代）
- **Message Persistence Service** 寫 DB 後需非同步派發翻譯任務給 Translation Service
- plan.md 第七節新增 Cassandra `translation_messages` schema 與 PostgreSQL `notifications` schema
- WS 連線 URL 修正：token 透過 identify op 傳遞，不放 query string
- `index.html`：新增 **Guild Settings modal**（編輯社群 + 邀請碼 + 刪除）、**Join by Invite modal**、guilds sidebar 加入「透過邀請碼加入」按鈕
- `app.js`：實作 `showGuildSettings()`、`handleUpdateGuild()`、`handleDeleteGuild()`、`handleCreateInvite()`、`copyInviteCode()`、`showJoinByInviteModal()`、`handleJoinByInvite()`、`handleKickMember()`、`handleUpdateMemberRole()`
- `renderMembers()` 現在顯示角色 badge（owner/admin/moderator）及管理員操作按鈕（踢人/改角色），透過 `ROLE_LEVEL` 比較決定可見性
- `styles.css`：新增 `.role-badge`、`.btn-danger`、`.btn-icon-sm`、`.invite-code-box`、`.settings-section`、`.member-actions`

2026-04-30 — 前後端對接修正（Phase 1）：
- `config.js`：補齊所有 Phase 1 端點（REFRESH, LOGOUT, PUBLIC_USER, KICK_MEMBER, UPDATE_MEMBER_ROLE, CREATE_INVITE, GET_INVITE, JOIN_BY_INVITE）及 `REFRESH_TOKEN` storage key
- `api.js`：新增 `setRefreshToken/getRefreshToken`；login 儲存 refresh_token；`request()` 自動 401 → refresh → retry；新增 `refreshToken/logout/getPublicUser/kickMember/updateMemberRole/createInvite/getInvite/joinByInvite`
- `api.js` `createChannel` 改送 `topic`（後端 `CreateChannelRequest.Topic`），不再送 `description`
- `app.js`：`handleLogout()` 改為 async，先呼叫 `api.logout()` 再清除狀態，並清除 `REFRESH_TOKEN`；`updateChannelHeader()` 改用 `channel.topic`
- Channel 模型 JSON 欄位為 `topic`（非 `description`）；Guild 模型用 `description`

2026-04-30 — RBAC 權限系統（Phase 1 完成）：
- `middleware.RequireGuildRole(minRole, guildMemberRepo)` 封裝角色驗證，從 URL param `:id` 取得 guild ID，角色階層 member<moderator<admin<owner，並將 `guild_member` 存入 context
- `middleware.HasMinRole(userRole, minRole)` 公開函式，供 service/handler 重用
- `guildRoleLevel` map 與 `hasMinRole()` 在 `guild_service.go` 內部維護
- `ErrInsufficientPermission` 新增至 guild_service errors
- `KickMember`：admin 可踢低階成員，不能踢同級或高階
- `UpdateMemberRole`：admin 可設定 moderator/member；只有 owner 可設定 admin
- 路由層 middleware 已套用：`DELETE /guilds/:id/members/:userId`、`PUT /guilds/:id/members/:userId/role`、`POST /guilds/:id/channels` 均需 `RequireGuildRole("admin")`
- `Server` struct 新增 `guildMemberRepo repository.GuildMemberRepository` 欄位
- 訊息刪除（`DeleteMessage`）已支援 admin 刪除他人訊息（service 層已處理）

2026-04-30 — API 補全（Phase 1）：
- Cursor-based pagination 已實作（`GetByChannelIDCursor` + `before`/`limit` query params）
- Guild 邀請碼系統：`GuildInvite` model、`guild_invite_repository.go`、`guild_invite_service.go`；routes: `POST /guilds/:id/invites`、`GET /invites/:code`、`POST /guilds/join-by-invite`
- Refresh Token：`RefreshToken` model、`refresh_token_repository.go`；`UserService` 新增 `RefreshAccessToken`/`RevokeRefreshToken`；token rotation 機制（舊 token 撤銷，發新 token）；routes: `POST /auth/refresh`、`POST /auth/logout`
- User 公開資料 API：`GET /api/v1/users/:id`（回傳 `PublicUser`，不含 email）
- Message Attachments：`Message.Attachments []string`（`gorm:"-"`），AfterFind hook 確保序列化為 `[]` 而非 `null`
- `NewUserService` 需傳入 `RefreshTokenRepository`；`NewGuildHandler` 需傳入 `GuildInviteService`
