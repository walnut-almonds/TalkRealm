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
go run ./scripts/seedwords    # 匯入 data/words.csv 到 words 表（冪等 upsert）
go run ./scripts/buildwords   # 重建 data/words.csv（需 data/raw/ 原始字典，見 data/README.md）
```

- **免後端視覺驗證**：`npx vite --port 5199` 起 dev server 後，用 chrome-devtools 的 `navigate_page` + `initScript` 攔截 `window.fetch`（mock `/api/v1/*` 回應）並塞 `talkrealm_token`/`talkrealm_last_guild`/`talkrealm_last_channel` 進 localStorage，即可渲染 Galaxy 首頁與聊天主畫面截圖。API 形狀：`{user}`, `{guilds}`, `{channels}`, `{members}`, `{messages}`（見 `web/src/api/index.js` 的 `EP`）。注意 reload 會失去 initScript，需重新 navigate；Windows 下背景 vite 停止後 port 可能殘留，需 `taskkill //PID`。
  - chrome-devtools CLI 的 `navigate_page` 導航到**與當前相同的 URL** 會變 same-document navigation，`--initScript` 不會重跑（token/mock 全失效、看起來像登出）——重導時加個 `?fresh=<random>` query 破除。
  - chrome-devtools MCP 未連線時可用 CLI：`npm i -g chrome-devtools-mcp` 後 mise 缺 shim（複製任一 shim 補上），但 batch shim 遇多行參數（如 `--initScript "$(cat mock.js)"`）會噴 `batch file arguments are invalid`——改直接呼叫 `~/AppData/Local/mise/installs/node/<ver>/chrome-devtools`（sh script，bash 可跑）。前端是 hash router，要開 `http://localhost:5199/#/learn` 而非 `/learn`。合成 PointerEvent 測拖曳時 `setPointerCapture` 會因無 active pointer 而 throw（LetterTray 已 try/catch），互動狀態的 DOM 斷言要等 ~100ms（Vue nextTick）再讀。

## Architecture Notes
- `internal/server/server.go`：DI 組裝、路由設定的主入口
- **Learn 模組（單字學習遊戲）**：`/api/v1/learn/*`；spec 在 `docs/superpowers/specs/2026-07-08-learn-vocab-game-design.md`（gitignored，本地）。模組邊界：learn 表（`words`/`learn_*`）只存 plain `user_id` 不建 GORM 關聯、排行榜顯示走 `LearnUserLookup` interface、Redis key 全帶 `learn:` 前綴——為未來拆獨立 service 預留。關卡含答案存 LevelStore（Redis，無 Redis 退記憶體）TTL 2h；每日挑戰用 SetNX 快取當日模板達成全站同題；anagram 索引（`learn_anagram.go`）lazy 建於首個 wheel 請求。
- **Learn crossword 模式**（`internal/service/learn_crossword.go`，spec 見 `docs/superpowers/specs/2026-07-13-crossword-grid-mode-design.md`）：答案字互相交叉排成 2D 網格（Wordscapes 式自由形狀），獨立於 fill/wheel——`crosswordLevel` 是全新 struct，不與 `learnLevel` 共用；Redis 存值多包一層 `levelEnvelope{Mode, Data}` 信封辨識模式（`saveEnvelope`/`loadEnvelope`），`Guess` 依 `env.Mode` 分流到 `guessFillWheel`/`guessCrossword`。排版演算法是回溯搜尋 + branch-and-bound 剪枝 + 步數上限（20000）保底，找不出交叉的字會落到 bonus 列表；`validPlacement` 強制 Wordscapes 式鄰接規則（頭尾留白 + 非交叉格側邊留白，字與字只在交叉點相碰），版面離散感來自這條硬規則而非評分。前端交叉格「提前顯示字母」完全是前端純渲染衍生（`crosswordGrid.js` 的 `buildCells`，依 `masked` 欄位逐格取非底線字元），後端不用額外算。前端 `LetterTray.vue` 是從 `LetterWheel.vue` 抽出的共用字母盤點選元件，`Crossword.vue` 與 `LetterWheel.vue` 都用它。
- **Learn 提示系統（hint/reveal）**：spec 見 `docs/superpowers/specs/2026-07-14-learn-hint-system-design.md`。三階提示梯（`maskWithHint`/`hintDiscount`/`advanceHint`，`internal/service/learn_service.go`）：`base=len(word)*tier`；tier0 原價；tier1（揭 1 字母）扣 1/4；tier2（顯示釋義）只剩 1/4；「揭曉答案」任何階段都可跳（0 XP，且不寫入 `learn_word_records`，不算學習信號）。`learnLevel`/`crosswordLevel` 各字都帶平行陣列 `HintTier[]`/`HintPos[]`，`Hint`/`Reveal` 依 envelope `Mode` 分流到 `hintWheel`/`hintCrossword`、`revealWheel`/`revealCrossword`，`Guess` 的 XP 計算統一套 `hintDiscount`。前端統一提示面板 `HintList.vue`（依 `hintTier` 顯示「揭字母/顯示釋義/揭曉答案」按鈕文案），`LetterWheel.vue`/`Crossword.vue` 都嵌入同一元件，各自把 `slots`/`words` 轉成 `hintItems` 餵給它。
- **Learn 固定關卡（campaign）+ 排行榜（2026-07-17）**：`internal/service/learn_campaign.go`。50 關 easy（tier 1）固定關卡，`EnsureCampaignLevels()` 開機冪等生成（server.go 呼叫，失敗僅警告）存 `learn_campaign_levels`（puzzle JSON 含答案；**發布後不可變**，重生成會回溯改題破壞進度/榜——只允許追加新關）。難度曲線 `campaignWordCount`：每 10 關答案數 +1（3→7）。遊玩完全沿用 crossword envelope 流程：`crosswordLevel.Campaign>0` 標記關卡編號，完關時 `onCrosswordCompleted` 記 `learn_campaign_progresses` 首通（(user,level) unique DO NOTHING，重玩不刷榜）；解鎖規則=前一關已首通。排行榜三種共用 `LeaderboardView`/`boardEntry`：每日（既有）、關卡榜（SUM(score)+MAX(level_no) tiebreak）、週榜（`learn_weekly_xps`，ISO week，`onLevelCompleted` 對所有模式 upsert 累加——量投入非首通）；`?scope=friends` 走 `LearnFriendLookup` interface（friendshipRepo.FriendIDs 注入，維持模組邊界）。前端：hub 關卡格狀選單（done/next/locked）、完關「下一關」按鈕（`Crossword.vue` 讀 `crossword.campaign`）、排行榜卡 tab（關卡/本週）×scope（全球/好友）。XP 總分刻意不做榜（肝榜+新人追不上+公式會改），只留 hub 個人數字。
- **Learn XP 加成與答案數自選（2026-07-17）**：per-word XP = `wordXP(字, effectiveTier, mode)` × `scaleXPBySize`。`effectiveTier` = max(字自身 tier, 關卡 tier)——wheel/crossword anagram 子字不分難度，抽到難字有加成、簡單字不受罰；tier 本身已是 ×1~×5 乘數，**不要再疊第二層難度乘數**（會重複計算）。`scaleXPBySize`：3 字 ×1.0 每多 1 字 +10% 封頂 ×1.6（整數運算 `xp*(10+b)/10`），campaign 後段關卡自動受惠。各 level struct 存 `Tiers []int`（Redis 舊關卡缺此欄位 → `effectiveTier` fallback 關卡 tier，部署時 in-flight 關卡不會 panic）。隨機三模式支援 `count` 參數（0=預設，clamp 3..9；fill 精確、wheel/crossword 為上限），前端 hub「單字數」少(3)/中(6)/多(9) 按鈕存 localStorage `talkrealm_learn_count`（預設 6）。campaign 生成接受條件是 `placed == want`（湊滿目標字數才立即收，試滿 30 次才降級取最多），難度曲線是盡力保證非上限。底字長度另有曲線 `campaignBaseLen`（1-5 關 3 字母、6-10 關 4、11-20 關 5、21-35 關 6、36-50 關 7）——生成是**兩階段**（`generateCampaignPuzzle`/`tryCampaignDraw`）：第一階段堅持指定長度、第一階段有任何成品（即使字數不足）就採用，**長度優先於字數**；全空手才進放寬階段（允許重複 anagram 組 → findWheelBase 保底）。教訓：保底不能放在同一個 try 內立刻觸發，也不能讓放寬階段的「字數湊滿」蓋過堅持階段的候選——兩個 bug 都會讓第 1 關長出 5-7 字母輪盤（用真實 words.csv 的 in-process 生成 demo 才抓到，純 fixture 測不出）。usedBase 以字母簽名（sortLetters）去重，避免同 anagram 組連續出題。**改生成演算法後既有 DB 關卡不會重生**（發布後不可變），開發環境要 `DELETE FROM learn_campaign_levels; DELETE FROM learn_campaign_progresses;` 再重啟才吃到新曲線。
- **Learn 例句填空 SRS 複習（2026-07-23）**：間隔重複複習模式，`internal/service/learn_srs.go`。新表 `LearnSentence`（`data/sentences.csv` 由 LLM 生成 **318 句/318 個 distinct 簡單字**（多 tier-1，2026-07-23 從 35 擴充）、`SeedSentences` 冪等匯入、`text_en` 唯一、內含 `{{}}` 挖空標記、答案另存 `Answer`、三語翻譯）；新增例句流程：候選字先 `awk` 比對 words.csv 確保存在→產 CSV→awk 驗欄位數/唯一/挖空→temp Postgres 跑 `SeedSentences` 確認 imported/skipped（唯一能證 word 全解析+Go csv 解析的方式，翻譯錯誤無測試可擋故需人工把關）；`LearnWordRecord` 擴充 `srs_stage`/`next_review_at`（**只此模式維護**，其他模式 UpsertWordRecord 明列欄位不碰，故 crossword 玩過的字 srs_stage 仍 0＝未進輪替）。間隔曲線 `srsIntervalDays`：stage 1/2/3/4/5+ → 1/2/4/7/15 天，答對進位、答錯歸 0 重置。session 組成 `planSRSSession`：新字配額 `count/4`（**floor 非 ceil**——複習優先，小場次全給到期字，沒到期才用新字回填）；due 與 new 互斥（stage>=1 vs stage=0）。session 用 envelope（ModeSRS）存 Redis，完成 session 觸發 `onLevelCompleted`（streak+週榜）。**當場重新學習佇列**（Anki-style）：`AnswerSRS` 答錯不退場只記 `EverWrong`（不寫 DB），答對才「退場」`Retired` 並寫長期排程——整場乾淨過關才進位、中途錯過即 lapse 重置 stage0，XP 乾淨全額/lapse 折半。長期排程 DB 只在退場時寫一次（棄局的錯卡維持原狀不重置，語意合理）。5 分鐘重考與「前面沒別的卡就直接再出現」的節奏在**前端佇列**（`SentenceReview.vue`）：`queue=[{index,dueAt}]`，fresh 卡 dueAt=0 永遠優先，答錯 push `dueAt=now+5min`，`pickNext` 挑最早 dueAt（都沒到期就挑最早那張＝fallback 直接再出現），queue 清空＝全退場＝完成。前端：hub「複習」分頁（預設首頁）+ 例句 `{{}}` 拆前後段夾輸入框、打字或 `useSpeechInput` 語音、作答後揭答上色+音標+SpeakButton。**踩雷**：`NewSentenceWordIDs` 原本 `SELECT DISTINCT ... ORDER BY random()` 在 Postgres 報 42P10（DISTINCT 的 ORDER BY 必須在 select list），改用 distinct 子查詢外層再 random——mock 測不出，靠臨時 Postgres（docker `postgres:16` on 5433）跑 `cmd/srsdbcheck` 一次性 smoke（migration+seed+session 全路徑）才抓到，驗完即刪。
- **Learn 單字發音（2026-07-21）**：`composables/useSpeak.js`（瀏覽器原生 `speechSynthesis`，免後端免依賴）+ `components/learn/SpeakButton.vue` 共用喇叭鈕，四處掛載：`HintList.vue` 已解列、`Crossword.vue`/`LetterWheel.vue` 完關回顧列表、`WordFill.vue` 已解格。**硬規則：只對 `solved` 之後的答案顯示發音鈕**——念未解字等於繞過提示系統的三階梯直接洩答案，跟「後端絕不下發未解答案」是同一條防線，前端不可自己開一個後門。`WordFill.vue` 的 `wf-slot` 原本整格是 `<button>`，已解格塞入 `SpeakButton`（也是 button）會變成無效 HTML 巢狀按鈕，改用 `<component :is="slot.solved ? 'div' : 'button'">` 動態切換標籤解決；未解格仍是真按鈕保留鍵盤可達性，已解格本來就不可互動改用 div 更貼近語意。
- **Learn hub 分頁化（2026-07-20）**：`LearnView.vue` 的 `learn-hub` 從單頁一路往下疊卡片（每日/關卡/難度模式/排行榜）改成 `hub-tabs`（比照 chat 頻道切換）：`activeTab` state 切 4 個分頁（daily/campaign/random/board），每次只渲染一個 `<section>`，內容天然不會越長越要捲動。`learn.error` 提示移到 tabs 外層、所有分頁共用一處顯示（不再各分頁各留一份）。分頁清單 `tabs` 陣列存字面 i18n key（同 `tierKeys` 慣例，check-i18n-keys 認不到動態 key 但不影響——只是不驗證用量，key 本身要自己顧齊四語）。
- **Learn 前端互動（2026-07-17）**：`LetterTray.vue` 是圓形字母輪，支援拖曳連線（pointerdown 起手/滑入加選/滑回上一顆撤銷/放開即送出，Wordscapes 慣例）與點選並存——命中判定用字母中心距離（setPointerCapture 後 pointerenter 不會落在字母上），字母按鈕 `pointer-events:none` 只留鍵盤 click。crossword 網格與 `HintList` 雙向 hover 高亮：`crosswordGrid.js` 的 `buildCells` 每格帶 `words[]`（交叉格兩個字都亮），`Crossword.vue` 持 `activeIndexes` 綁兩邊（`HintList` 的 `activeIndexes` prop / `activate`/`deactivate` emit 是選配，wheel 不綁）。
- `internal/websocket/manager.go`：channel 訂閱索引（`channelSubscriptions map[uint]map[*Client]bool`）+ guild 訂閱索引（`guildSubscriptions map[uint]map[*Client]bool`），O(1) 廣播；jwtManager 注入用於 identify op；identify 後自動呼叫 `SubscribeClientToUserGuilds` 訂閱所有 guild
- **WS 頻道存取與撤銷**：`server.go` 必須以 `SetChannelAccessLookup(channelRepo)` 注入存取查詢；identify 初始 channels、`subscribe`、`typing_start` 與 `voice_state_update` 全部先驗證 guild member 或 DM participant，voice event 的 guild ID 必須由 server-side channel metadata 推導、不可信任 client payload。使用者 leave/kick 後，`GuildSubscriptionRevoker` 會立刻移除其所有連線對該 guild 的 channel/guild subscription，否則惡意 client 可保持舊訂閱接收後續廣播。
- WS 協議：client→server op: `identify`, `heartbeat`, `subscribe`, `unsubscribe`, `typing_start`, `send_message`, `voice_state_update`；server→client op: `hello`, `ready`, `heartbeat_ack`, `message_create`, `message_update`, `message_delete`, `typing_start`, `presence_update`, `error`, `guild_update`, `guild_delete`, `guild_member_add`, `guild_member_remove`, `guild_member_update`, `channel_create`, `channel_update`, `channel_delete`, `voice_state_update`
- WS 端點：`GET /api/v1/ws`（無需 JWT 中間件，由 identify op 驗證）
- identify flow：client 連線 → server 送 `hello`（heartbeat_interval=30000ms）→ client 送 `identify`（token + channels[]）→ server 驗證 JWT，送 `ready` + 廣播 `presence_update online`
- `pkg/auth/jwt.go`：JWTManager，sign / verify token
- `pkg/database/database.go`：GORM DB singleton
- REST API 路由前綴：`/api/v1/`
- WebSocket 端點：`GET /api/v1/ws`（token 透過 identify op 傳遞，不再放 query string）
- 目前訊息分頁是 offset，計畫改為 cursor-based（before message_id）
- **Learn words 表冪等 seeding**：`pkg/database/seed.go` 的 `SeedWords(csvPath)` 用 `ON CONFLICT(word) DO UPDATE` upsert（保留既有 `id`，不動 `learn_word_records.word_id`/Redis in-flight 關卡狀態），`cmd/server/main.go` 在 `AutoMigrate()` 後每次開機都無條件呼叫（讀不到 CSV 只 `logger.Warn`，非 fatal）；`Dockerfile` 對應 `COPY data/words.csv`。故意不用「CI 偵測 words.csv 變更才 TRUNCATE+INSERT」：TRUNCATE 會重置 auto-increment id，破壞既有 `word_id` 參照與進行中的關卡。`scripts/seedwords/main.go`（本機用）與開機路徑共用同一 `SeedWords`。

## Pitfalls
- **OAuth callback 與 hash router**：後端把 token 放在 fragment（`{frontend}/#/oauth/callback?access_token=...`）避免落進 server log／Referer，但前端是 `createWebHashHistory()`，那段 fragment 同時就是 router 的 path+query。清 token 時**不能**用 `history.replaceState`——它只改網址列，router 的 `currentRoute` 會停在 `oauth-callback` 與 URL 脫節。要用 `router.replace({ name: 'chat' })`。
- **OG 預覽對非 HTML 連結**：`GET /api/v1/og` 遇到 `image/*` 或其他非 `text/html` 內容時，現在改為回 `200`（image 會帶最小預覽 `{image:url}`，其他類型回空 OG）而非 `422`。可避免前端在訊息含 CDN 圖片連結（例如 `googleusercontent` avatar URL）時持續出現 console `Unprocessable Content`。
- **Tenor v1 已全面停用（2026-07 確認）**：`g.tenor.com/v1`（含舊 demo key `LIVDSRZULELA`）現在對任何 key 都回 `403 {"code":7,"error":"Tenor API is discontinued"}`，不再只是特定 key 失效。`web/src/api/index.js` 的 `searchGIFs` 已移除 v1 fallback，改為「沒有使用者自帶的 Tenor v2 API Key（設定於 UserSettingsModal，存 localStorage）就直接丟出『請填入 API Key』錯誤」，不再打死掉的 v1 端點。GIF picker 開新視窗發生錯誤時，先確認是否為此情況，而非程式碼回歸。
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

- **手機版左側抽屜（Discord-style）**：nav-rail 在聊天頁（DOM 有 `.channels-sidebar` 或 `.dm-sidebar`）與 Learn 頁（`.learn-view`）透過 `main.css` mobile 區塊的 `.app-shell:has(...)` 規則變 fixed off-canvas；聊天頁與 sidebar 一起滑入（sidebar `left:56px`、closed transform 是 `translateX(calc(-100% - 56px))`），Learn 頁由 `LearnView.vue` 的 `mobileNavOpen` state（root class `mobile-nav-open`）+ 復用 `.mobile-hamburger`/`.mobile-sidebar-backdrop` 控制。HomeView 無 sidebar 時 rail 留在 flow 內。注意 mobile 樣式分兩處：`channels-sidebar`/`members-sidebar` 在 `main.css`，`dm-sidebar` 在 `DMSidebar.vue` 的 scoped style，改抽屜行為要兩邊同步。

- **GORM 欄位命名陷阱（連續大寫縮寫）**：`DefinitionZHTW` 會被 GORM naming 轉成 `definition_zhtw`（不是 `definition_zh_tw`；json tag 可以自訂但 DB 欄位名跟著 GORM）。手寫 SQL/`clause.AssignmentColumns` 的欄位字串必須用 GORM 實際命名。同理 `ContentZHTW` → `content_zhtw`。
- **Postgres ON CONFLICT DO UPDATE 歧義**：`gorm.Expr("col + 1")` 在 DO UPDATE 內會報 42702（target 表與 excluded 都有該欄），必須帶表名：`gorm.Expr("learn_word_records.col + 1")`。mock repo 的單元測試測不出這類 SQL 錯誤，改 upsert 語句後要對真 DB 打一次。
- **本機驗證埠衝突**：5432/8080 可能被同機其他專案容器（infra-postgres/lobby）占用；smoke test 可用臨時容器（如 5433）+ `configs/config.yaml`（gitignored）改埠。
- **`AllWordsForIndex()` 輕量 SELECT 誤用陷阱**：`internal/repository/learn_repository.go` 的 `AllWordsForIndex()` 為了 anagram 索引效能故意只 `Select("id","word","tier","frequency","length")`，回傳的 `model.Word` 沒有 `Phonetic`/`Definition*`。`buildWheelLevel`/`buildCrosswordLevel` 曾直接拿這份輕量物件當最終答案資料，導致 wheel/crossword 所有非底字答案的音標/釋義自上線起就一直是空字串（fill 模式與底字沒事，因為底字另外用 `RandomWordsByTier` 全欄位查）。修法：`picked` 選定後再用 `s.repo.WordsByIDs(pickedIDs)` 換回全欄位（`internal/service/learn_service.go`/`learn_crossword.go`）。`fakeLearnRepo.AllWordsForIndex()`（`learn_service_test.go`）原本直接回傳完整 fixture，等於白盒測試也測不出來——已改成如實剝掉 phonetic/definition 欄位以誠實模擬正式環境；日後改 `AllWordsForIndex` 相關邏輯前，先確認 fake 是否仍誠實反映真實 SELECT 範圍。
- **`crosswordGrid.js` 的 `buildCells` 曾只認 `solved`**：交叉字謎前端只在單字整個解開（`word.solved && word.word`）時才把字母畫進網格格子，導致對網格字下「揭字母」提示時，提示清單顯示揭露字母、但網格格子仍是空的——兩處呈現不同步。後端 `masked` 欄位其實已經把提示揭露的字母也算進去（`maskWithHint`，底線代表未揭露）；修法是網格改吃 `masked` 逐格取非底線字元（不論是否 `solved`），而非只看 `solved`。

## Decisions
- **設定架構：齒輪只有一顆**。全站唯一設定入口在 NavRail 底部（`nav-foot`，呼叫 MainLayout `provide('openModal')` 開 `UserSettingsModal`）；modal 內用 section 分區（帳號/GIF/學習/密碼），新功能設定加 section、不新增齒輪。「改了立刻想看效果」的偏好做成功能內 inline 控制項（如 Learn 困難模式 toggle，hub 與遊戲中各一顆，同一份狀態）。純本機顯示偏好存 localStorage（`talkrealm_learn_hard`，state 在 `useLearnStore.hardMode`），需跨裝置/影響計分時才升級後端。困難模式渲染：`components/learn/mask.js` 的 `maskSegments()` 把連續 `_` 收成單一 gap 格。
- **前端視覺系統：Kinetic Noir（TalkRealm Edition）**，規範見根目錄 `DESIGN.md`（改編自 walnut-almonds.github.io 的同名系統）。要點：近黑 surface 階梯（#0e0e0e→#2a2a2a）、唯一裝飾色 slate-blue `--accent: #b3c6f3`、直角（`--radius: 0px`；頭像/presence 圓點例外——「人=圓、地方=方」）、1px hairline 取代陰影、Geist Mono 做系統性文字（分類標題/時間戳/徽章）、按鈕 hover 即時反白。tokens 在 `web/src/styles/main.css` `:root`（`--accent`/`--accent-hover`/`--brand` 已定義，元件的 var() fallback 不再吃到 Discord 色）；字體在 `web/index.html` 載入（Hanken Grotesk + Noto Sans TC + Geist Mono）。`web/css/styles.css` 是 pre-Vue 舊版，未套用新主題。新樣式禁用 Discord 特徵：blurple、圓→方 morph、紫色漸層。Social Galaxy 首頁（`web/src/views/HomeView.vue`，SVG 實作）已同步換色：`GUILD_PALETTE` 8 色是去飽和「noir 星座」色系、星雲/時段氛圍（data-atmosphere day/night/dawn/dusk）漸層降飽和；新增 guild 色一律走 muted pastel，不可回填飽和色。
- MQ 選擇 NATS JetStream（輕量，適合小團隊），備選 Kafka
- 物件儲存選 Minio（self-hosted S3-compatible），生產可換 AWS S3
- 語音選 LiveKit（WebRTC SFU）
- 檔案上傳採 Pre-signed URL 模式，API Server 不處理 binary

## Last Updated
2026-07-27
 — 修 GIF picker 開新視窗發生錯誤：根因是 Tenor v1 API 已被 Google 全面停用（見 Pitfalls），前端不再 fallback v1，改為缺 API Key 時直接提示使用者去設定填入
2026-07-23
 — SRS 加當場重新學習佇列：答錯的卡（新舊皆然）5 分鐘後或「前面沒別的卡就直接」再出現，反覆到答對一次才退場；長期排程改退場時才寫（乾淨進位/lapse 重置），節奏邏輯放前端佇列（見 Architecture Notes）；新增 `TestSRSWrongRequeuesUntilCorrect`，瀏覽器實測「錯→循環→重現→答對→完成」+38 XP
 — Learn 例句填空 SRS 間隔重複複習上線（新表 LearnSentence + LearnWordRecord SRS 欄位 + `data/sentences.csv` LLM 範例 35 句 + hub「複習」分頁 + 打字/語音作答，見 Architecture Notes「Learn 例句填空 SRS 複習」）；含 7 個新單元測試 + 真 Postgres smoke（抓到並修掉 DISTINCT+ORDER BY random() 的 42P10）+ 瀏覽器完整流程驗證。**部署提醒**：sentences.csv 已加進 Dockerfile COPY 與開機 SeedSentences

2026-07-21
 — Learn 單字發音（瀏覽器 speechSynthesis，見 Architecture Notes「Learn 單字發音」）：已解字才顯示喇叭鈕，避免繞過提示系統洩答案；WordFill 已解格改 `<component :is>` 動態標籤解決巢狀按鈕問題。純前端，無 i18n/後端變動，瀏覽器實測四處掛點皆正常、speechSynthesis 收到正確單字與 en-US

2026-07-20
 — Learn hub 改分頁式（每日/關卡/隨機/排行榜，比照 chat 頻道切換），解決卡片一路往下疊要一直捲動的問題，見 Architecture Notes「Learn hub 分頁化」；桌面+375px 手機皆截圖驗證
 — campaign 底字長度曲線修正（兩階段生成、長度優先於字數，見 Architecture Notes）；手機版三修：crossword 網格 `minmax(0,32px)`+`aspect-ratio` 自適應收縮、wheel/crossword 完關畫面加單字回顧列表（修「最後一字釋義看不到」）、Learn 頁 nav-rail 比照 chat 收成抽屜（見「手機版左側抽屜」條目）

2026-07-17
 — Learn XP 難字/字數加成 + 隨機模式答案數自選（少/中/多）+ campaign 生成改盡力保證目標字數（見 Architecture Notes「Learn XP 加成與答案數自選」）；含 4 個新單元測試，真瀏覽器驗證按鈕列與 request body
 — Learn 固定關卡 1~50（easy）+ 進度解鎖 + 關卡榜/週榜（好友/全球）上線（見 Architecture Notes「Learn 固定關卡」）；含開機冪等生成、首通不覆寫、四個新單元測試，真瀏覽器驗證關卡格/完關接關/榜切換
 — crossword 排版加入鄰接硬規則（`validPlacement` 頭尾留白 + 側邊留白），消除平行字貼齊的「表格感」，同時修掉相鄰字母偶然拼出偽答案字串的潛在困惑；`TestLayoutCrosswordAdjacencyRules` 驗證
 — Learn 前端互動改版：LetterTray 改圓形字母輪 + 拖曳連線（保留點選）、crossword 網格格子加底色/強邊框對比、網格↔提示列雙向 hover 高亮（見 Architecture Notes「Learn 前端互動」）；已用 chrome-devtools CLI + mock API 真瀏覽器驗證（拖曳送出/滑回撤銷/tap toggle/錯誤回饋/完關畫面）

2026-07-14
 — Learn wheel/crossword 三階提示系統（hint/reveal，見 Architecture Notes）上線，含前端統一 `HintList.vue`；真實瀏覽器驗證時額外發現並修復兩個既有 bug：(1) wheel/crossword 非底字答案音標/釋義自上線起一直是空字串（`AllWordsForIndex` 輕量 SELECT 誤用，見 Pitfalls），(2) crossword 網格對已提示但未解出的字不會 cross-reveal（`crosswordGrid.js` 只認 `solved`，見 Pitfalls）
 — 生產環境 `words` 表冪等 seeding 上線（開機自動 upsert，見 Architecture Notes 新增條目）；修復 crossword 誤顯示前一次 wheel 介面的殘留 state race，並補上遊戲中退出按鈕
 — 新增 Learn crossword 交叉字謎網格模式（獨立於 fill/wheel，見 Architecture Notes）；已用真實瀏覽器完整驗證（2D 網格渲染、cross-reveal 提示、bonus 字列表、完關計分皆正常）

2026-07-13
 — 設定入口收斂到 NavRail 底部齒輪；Learn 困難模式（隱藏底線數）上線（見 Decisions 設定架構條目）

2026-07-09
 — 單字學習遊戲 v1 完成（Learn 分頁：釋義填字/字母盤/每日挑戰+排行榜；見 Architecture Notes 的 Learn 模組）；新增 GORM 縮寫欄位命名與 ON CONFLICT 歧義兩條 Pitfalls

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
