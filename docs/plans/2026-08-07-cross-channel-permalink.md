# Cross-Channel Permalink — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let any chat message or guild wall post be shared as an internal `#/m/<id>` link that renders as a preview chip in another channel; clicking it precisely jumps to the source — loading the message even if it's old/unloaded, scrolling to it, and flashing a highlight.

**Architecture:** Chat messages and wall posts are both `model.Message`, and both render through `MessageItem`, so the share action, chip rendering, DOM anchor, and highlight all live in `MessageItem` and cover both surfaces. Backend adds a permalink-resolve endpoint and "load around a message" queries. The jump is coordinated through a store `jumpTarget` that the chat list (store.messages) and the wall (`FeedArea`) each consume.

**Tech Stack:** Go + Gin + GORM/PostgreSQL; Vue 3 (hash router, Pinia-style store); testify + `testutil` mocks; go-sqlmock.

## Global Constraints

- Spec: `docs/specs/2026-08-07-cross-channel-permalink-design.md` — authoritative.
- In scope: chat messages (text channels) + guild wall posts (feed channels) — both `model.Message`. Out of scope: cross-community `FeedPost`, DM messages.
- Token format: `{location.origin}/#/m/<messageId>` (hash deep-link). Access control is enforced at resolve time (non-member → 403).
- Precise jump (B): guarantee the target loads (around-query), scrolls into view, and flashes a highlight — even for old messages.
- Preview chip shows author · #channel · truncated content (~100 chars).
- Reuse existing idioms: message list enrichment (`GetByChannelIDCursor`/`GetPostsByChannelCursor` patterns), `MessageItem` renders content via `renderMarkdown`, store `selectGuild`/`selectChannel`/`loadMessages`.
- After each backend task run `make check` (fall back to `go build ./... && go vet ./... && go test ./...` if `-race`/mise shims break — no gcc on this Windows box).
- Frontend: no unit-test framework; verify with `npm --prefix web run build` + the E2E in the finish step.
- Commit after each task. Do NOT `git add -A`/`git add .` — there are unrelated uncommitted `web/src/components/learn/*.vue` changes; add only your files.

---

### Task 1: Repo — `GetMessagesAround` + `GetPostsAround`

**Files:**
- Modify: `internal/repository/message_repository.go` (interface + impl)
- Modify: `internal/testutil/mocks.go` (extend `MockMessageRepository`)
- Test: `internal/repository/message_repository_test.go`

**Interfaces:**
- Produces on `MessageRepository`:
```go
GetMessagesAround(channelID, aroundID uint, limit int) ([]*model.Message, error)   // window centered on aroundID, chronological
GetPostsAround(channelID, aroundID uint, limit int) ([]*model.Message, error)       // same, parent_id IS NULL
```
- Mock fields: `GetMessagesAroundFn`, `GetPostsAroundFn`.

- [ ] **Step 1: Write the failing test**

Add to `internal/repository/message_repository_test.go`:

```go
func TestMessageRepository_GetMessagesAround(t *testing.T) {
	db, mock, sqlDB := newTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	// older-or-equal half (id <= around, DESC)
	mock.ExpectQuery(`SELECT \* FROM "messages" WHERE channel_id = \$1 AND id <= \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel_id"}).AddRow(10, 1).AddRow(9, 1))
	// newer half (id > around, ASC)
	mock.ExpectQuery(`SELECT \* FROM "messages" WHERE channel_id = \$1 AND id > \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel_id"}).AddRow(11, 1))

	repo := repository.NewMessageRepository(db)
	msgs, err := repo.GetMessagesAround(1, 10, 6)
	require.NoError(t, err)
	// chronological ascending, target 10 present, window around it
	require.Len(t, msgs, 3)
	assert.Equal(t, uint(9), msgs[0].ID)
	assert.Equal(t, uint(10), msgs[1].ID)
	assert.Equal(t, uint(11), msgs[2].ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestMessageRepository_GetMessagesAround -v`
Expected: FAIL — `GetMessagesAround` undefined.

- [ ] **Step 3: Implement**

Add to the `MessageRepository` interface:
```go
	GetMessagesAround(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetPostsAround(channelID, aroundID uint, limit int) ([]*model.Message, error)
```
A shared private helper + the two methods (Preload User + Attachments.File like the cursor methods):
```go
// aroundWindow fetches a page centered on aroundID: ceil(limit/2) messages with
// id <= aroundID (older side) + floor(limit/2) with id > aroundID (newer side),
// merged into ascending chronological order. extraWhere is "" for chat, "AND parent_id IS NULL" for posts.
func (r *messageRepository) aroundWindow(channelID, aroundID uint, limit int, postsOnly bool) ([]*model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	older := (limit + 1) / 2
	newer := limit - older

	base := r.db.Preload("User").Preload("Attachments.File").Where("channel_id = ?", channelID)
	if postsOnly {
		base = base.Where("parent_id IS NULL")
	}

	var olderMsgs []*model.Message
	if err := base.Session(&gorm.Session{}).
		Where("id <= ?", aroundID).Order("id DESC").Limit(older).Find(&olderMsgs).Error; err != nil {
		return nil, err
	}
	var newerMsgs []*model.Message
	if err := base.Session(&gorm.Session{}).
		Where("id > ?", aroundID).Order("id ASC").Limit(newer).Find(&newerMsgs).Error; err != nil {
		return nil, err
	}
	// olderMsgs is DESC; reverse to ASC, then append newer (already ASC)
	for i, j := 0, len(olderMsgs)-1; i < j; i, j = i+1, j-1 {
		olderMsgs[i], olderMsgs[j] = olderMsgs[j], olderMsgs[i]
	}
	return append(olderMsgs, newerMsgs...), nil
}

func (r *messageRepository) GetMessagesAround(channelID, aroundID uint, limit int) ([]*model.Message, error) {
	return r.aroundWindow(channelID, aroundID, limit, false)
}

func (r *messageRepository) GetPostsAround(channelID, aroundID uint, limit int) ([]*model.Message, error) {
	return r.aroundWindow(channelID, aroundID, limit, true)
}
```
(Confirm `gorm` is imported in the file; the cursor methods already use it.)

- [ ] **Step 4: Extend the mock**

Add to `MockMessageRepository` in `internal/testutil/mocks.go`:
```go
	GetMessagesAroundFn func(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetPostsAroundFn    func(channelID, aroundID uint, limit int) ([]*model.Message, error)
```
```go
func (m *MockMessageRepository) GetMessagesAround(channelID, aroundID uint, limit int) ([]*model.Message, error) {
	return m.GetMessagesAroundFn(channelID, aroundID, limit)
}
func (m *MockMessageRepository) GetPostsAround(channelID, aroundID uint, limit int) ([]*model.Message, error) {
	return m.GetPostsAroundFn(channelID, aroundID, limit)
}
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/repository/ -run "TestMessageRepository_GetMessagesAround" -v && go build ./...`
Expected: PASS and build OK. If sqlmock preload sub-queries error (Attachments/User), stub them with empty rows as the existing cursor tests do; if the WHERE regex differs, relax it but keep the ascending-order assertion.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/message_repository.go internal/testutil/mocks.go internal/repository/message_repository_test.go
git commit -m "feat(permalink): GetMessagesAround / GetPostsAround repo queries"
```

---

### Task 2: Permalink resolve endpoint

**Files:**
- Modify: `internal/service/message_service.go` (DTO + `ResolvePermalink`)
- Modify: `internal/handler/message_handler.go` (handler)
- Modify: `internal/server/server.go` (route)
- Test: `internal/service/message_service_test.go`

**Interfaces:**
- Produces on `MessageService`:
```go
type PermalinkResponse struct {
	Message PermalinkMessage `json:"message"`
	Channel PermalinkChannel `json:"channel"`
}
type PermalinkMessage struct {
	ID      uint       `json:"id"`
	Content string     `json:"content"`
	Author  model.User `json:"author"`
}
type PermalinkChannel struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	GuildID *uint  `json:"guild_id"`
	Type    string `json:"type"`
}
ResolvePermalink(messageID, viewerID uint) (*PermalinkResponse, error)
```
- Produces HTTP: `GET /api/v1/messages/:id/permalink`.

- [ ] **Step 1: Write the failing test**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_ResolvePermalink_ChecksAccess(t *testing.T) {
	msg := &model.Message{ID: 5, ChannelID: 1, Content: "hello world", User: model.User{ID: 9, Username: "bob"}}
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "text", Name: "general"}
	mockMsg := &testutil.MockMessageRepository{GetByIDFn: func(id uint) (*model.Message, error) { return msg, nil }}
	mockCh := &testutil.MockChannelRepository{GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil }}

	// member → resolves
	member := &testutil.MockGuildMemberRepository{GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil }}
	svc := service.NewMessageService(mockMsg, mockCh, member, nil)
	resp, err := svc.ResolvePermalink(5, 7)
	require.NoError(t, err)
	assert.Equal(t, uint(1), resp.Channel.ID)
	assert.Equal(t, "general", resp.Channel.Name)
	assert.Equal(t, uint(9), resp.Message.Author.ID)

	// non-member → error
	nonMember := &testutil.MockGuildMemberRepository{GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return nil, errors.New("not found") }}
	svc2 := service.NewMessageService(mockMsg, mockCh, nonMember, nil)
	_, err = svc2.ResolvePermalink(5, 7)
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestMessageService_ResolvePermalink -v`
Expected: FAIL — `ResolvePermalink` undefined.

- [ ] **Step 3: Implement service**

Add the DTOs + method to `internal/service/message_service.go` (and the `MessageService` interface). Mirror the access check used by `GetMessage`/`ListComments` (fetch message → fetch channel → require guild membership):
```go
func (s *messageService) ResolvePermalink(messageID, viewerID uint) (*PermalinkResponse, error) {
	msg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	channel, err := s.channelRepo.GetByID(msg.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}
	if err := s.ensureChannelAccess(channel, msg.ChannelID, viewerID); err != nil {
		return nil, err
	}
	content := msg.Content
	if len(content) > 100 {
		content = content[:100]
	}
	return &PermalinkResponse{
		Message: PermalinkMessage{ID: msg.ID, Content: content, Author: msg.User},
		Channel: PermalinkChannel{ID: channel.ID, Name: channel.Name, GuildID: channel.GuildID, Type: channel.Type},
	}, nil
}
```
Note: truncating with `content[:100]` cuts bytes — for MVP that's acceptable; if the last byte splits a multibyte rune, trim with `strings.ToValidUTF8(content[:100], "")` to avoid broken runes. Use that safe form.

- [ ] **Step 4: Handler + route**

In `internal/handler/message_handler.go` add (mirror `GetMessage`):
```go
// ResolvePermalink GET /messages/:id/permalink
func (h *MessageHandler) ResolvePermalink(c *gin.Context) {
	messageID := /* parse :id like GetMessage does */
	userID := /* userID from context like GetMessage does */
	resp, err := h.messageService.ResolvePermalink(messageID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
```
(Use the exact id/userID parsing idiom already in `GetMessage` in that file.)
In `internal/server/server.go`, in the `messages` group next to `messages.GET("/:id", ...)`:
```go
				messages.GET("/:id/permalink", s.messageHandler.ResolvePermalink)
```

- [ ] **Step 5: Run test + full check**

Run: `go test ./internal/service/ -run "TestMessageService_ResolvePermalink|TestMessageService_" -v` then `make check` (or non-race fallback). Swagger may regenerate — include `docs/openapi/*` if changed.
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/message_service.go internal/handler/message_handler.go internal/server/server.go internal/service/message_service_test.go docs/openapi/
git commit -m "feat(permalink): GET /messages/:id/permalink resolve endpoint"
```

---

### Task 3: `around` branch in message + post list endpoints

**Files:**
- Modify: `internal/service/message_service.go` (`MessagesAround` / `PostsAround`)
- Modify: `internal/handler/message_handler.go` (`ListChannelMessages` / `ListPosts` — around branch)
- Test: `internal/service/message_service_test.go`

**Interfaces:**
- Consumes: `GetMessagesAround` / `GetPostsAround` (Task 1).
- Produces on `MessageService`:
```go
MessagesAround(channelID, userID, aroundID uint, limit int) (*MessageListResponse, error)
PostsAround(channelID, userID, aroundID uint, limit int) (*PostListResponse, error)
```

- [ ] **Step 1: Write the failing test**

Add to `internal/service/message_service_test.go`:

```go
func TestMessageService_MessagesAround(t *testing.T) {
	channel := &model.Channel{ID: 1, GuildID: testutil.PtrUint(10), Type: "text"}
	win := []*model.Message{{ID: 9, ChannelID: 1}, {ID: 10, ChannelID: 1}, {ID: 11, ChannelID: 1}}
	mockMsg := &testutil.MockMessageRepository{
		GetMessagesAroundFn: func(cid, around uint, limit int) ([]*model.Message, error) { return win, nil },
	}
	mockCh := &testutil.MockChannelRepository{GetByIDFn: func(id uint) (*model.Channel, error) { return channel, nil }}
	member := &testutil.MockGuildMemberRepository{GetMemberFn: func(g, u uint) (*model.GuildMember, error) { return &model.GuildMember{UserID: u}, nil }}
	svc := service.NewMessageService(mockMsg, mockCh, member, nil)

	resp, err := svc.MessagesAround(1, 7, 10, 6)
	require.NoError(t, err)
	require.Len(t, resp.Messages, 3)
	assert.Equal(t, uint(10), resp.Messages[1].ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -run TestMessageService_MessagesAround -v`
Expected: FAIL — `MessagesAround` undefined.

- [ ] **Step 3: Implement service methods**

Add `MessagesAround` and `PostsAround` to `internal/service/message_service.go` (+ interface). Mirror the access check + limit clamp used by `ListChannelMessages`/`ListPosts`; `PostsAround` reuses the same batch enrichment `ListPosts` uses (like_count/comment_count/liked_by_me):
```go
func (s *messageService) MessagesAround(channelID, userID, aroundID uint, limit int) (*MessageListResponse, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}
	if err := s.ensureChannelAccess(channel, channelID, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	msgs, err := s.messageRepo.GetMessagesAround(channelID, aroundID, limit)
	if err != nil {
		return nil, err
	}
	return &MessageListResponse{Messages: msgs, HasMore: false}, nil
}
```
For `PostsAround`, fetch via `GetPostsAround` then run the SAME enrichment loop `ListPosts` uses to build `[]*PostResponse` (extract that enrichment into a private helper `enrichPosts(posts, userID) []*PostResponse` if not already, and call it from both `ListPosts` and `PostsAround`). Return `*PostListResponse{Posts: ..., HasMore: false}`.

- [ ] **Step 4: Handler around-branch**

In `internal/handler/message_handler.go`:
- `ListChannelMessages`: parse `around` query (`strconv` like the others); if `around > 0`, call `h.messageService.MessagesAround(channelID, userID, around, limit)` instead of the before-cursor path.
- `ListPosts`: same, calling `PostsAround`.

- [ ] **Step 5: Run tests + full check**

Run: `go test ./internal/service/ -run "TestMessageService_" -v` then `make check` (or fallback).
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/message_service.go internal/handler/message_handler.go internal/service/message_service_test.go docs/openapi/
git commit -m "feat(permalink): around-load branch on message + post list endpoints"
```

---

### Task 4: Frontend API + copy-link action + i18n

**Files:**
- Modify: `web/src/api/index.js`
- Modify: `web/src/components/MessageItem.vue` (copy-link action)
- Modify: `web/src/i18n/locales/{en,zh,zh-tw,ja}.js`
- Verify: `npm --prefix web run build`

**Interfaces:**
- Consumes: Tasks 2-3 endpoints.
- Produces: `api.resolvePermalink(id)`; `around` params on `getChannelMessages`/`getPosts`; a copy-link button on every message/post.

- [ ] **Step 1: API methods**

In `web/src/api/index.js` `EP` map: `MESSAGE_PERMALINK: (id) => \`/api/v1/messages/${id}/permalink\``. Method:
```js
    resolvePermalink(id) { return this.get(EP.MESSAGE_PERMALINK(id)) },
```
Add an optional `around` arg to the existing `getChannelMessages` and `getPosts` (append `&around=` when set), or add `getMessagesAround(channelId, aroundId, limit=50)` / `getPostsAround(...)` helpers. Match the existing method style.

- [ ] **Step 2: Copy-link action in MessageItem**

In `web/src/components/MessageItem.vue`, add to the `.message-actions-bar` (near the existing copy/edit buttons) a button:
```html
<button class="msg-action-btn" :title="t('chat.copyLink')" @click="copyPermalink">
  <i class="fas fa-link"></i>
</button>
```
```js
async function copyPermalink() {
  const url = `${location.origin}${location.pathname}#/m/${props.message.id}`
  try { await navigator.clipboard.writeText(url); store.showNotification(t('chat.linkCopied'), 'success') }
  catch { store.showNotification(t('chat.linkCopyFailed'), 'error') }
}
```

- [ ] **Step 3: i18n**

Add `chat.copyLink`, `chat.linkCopied`, `chat.linkCopyFailed`, and (for later tasks) `chat.permalinkUnavailable`, `chat.jumpFailed` to ALL FOUR locales.

- [ ] **Step 4: Verify build + commit**

Run: `npm --prefix web run build` (exit 0).
```bash
git add web/src/api/index.js web/src/components/MessageItem.vue web/src/i18n/locales/
git commit -m "feat(permalink): copy-link action + resolve API + i18n"
```

---

### Task 5: `MessagePermalinkCard` + chip rendering in MessageItem

**Files:**
- Create: `web/src/components/MessagePermalinkCard.vue`
- Modify: `web/src/components/MessageItem.vue` (detect permalinks, render cards, strip from text)
- Verify: `npm --prefix web run build`

**Interfaces:**
- Consumes: `api.resolvePermalink` (Task 4); `store.jumpToPermalink` (Task 6 — bind the click; until Task 6 lands, the click can call a no-op/`store.jumpToPermalink?.(id)`).

- [ ] **Step 1: Build `MessagePermalinkCard.vue`**

Props `messageId`. On mount, `api.resolvePermalink(messageId)` → show `作者 · #channel.name · content摘要`. States: loading (skeleton), error (disabled "無法存取的引用" using `t('chat.permalinkUnavailable')`). Click (when resolved) → `store.jumpToPermalink(messageId)`. Reuse avatar + Kinetic Noir styling (mirror `FeedPostCard`'s author/snippet look). Small card with a link icon.

- [ ] **Step 2: Detect + render in MessageItem**

In `web/src/components/MessageItem.vue`:
- Add a regex + computed for referenced ids:
```js
const PERMALINK_RE = /#\/m\/(\d+)/g
const permalinkIds = computed(() => {
  const ids = []
  const s = props.message.content || ''
  let m
  const re = new RegExp(PERMALINK_RE)
  while ((m = re.exec(s)) !== null) ids.push(Number(m[1]))
  return [...new Set(ids)]
})
```
- Strip the permalink URLs from the text that goes to `renderMarkdown` so the chip replaces them:
```js
const _rendered = computed(() => renderMarkdown((props.message.content || '').replace(/\S*#\/m\/\d+/g, '').trim()))
```
- In the template, after `.message-text`, render a card per id:
```html
<MessagePermalinkCard v-for="pid in permalinkIds" :key="pid" :messageId="pid" />
```
(Import `MessagePermalinkCard`.)

- [ ] **Step 3: Verify build + commit**

Run: `npm --prefix web run build` (exit 0).
```bash
git add web/src/components/MessagePermalinkCard.vue web/src/components/MessageItem.vue
git commit -m "feat(permalink): preview chip rendering in MessageItem"
```

---

### Task 6: Jump infra — store flow + anchors/highlight + around-load + router

**Files:**
- Modify: `web/src/stores/useAppStore.js` (`jumpToPermalink`, `loadMessagesAround`, `jumpTarget`, `highlightMessageId`, `scrollToMessage`)
- Modify: `web/src/components/MessageItem.vue` (DOM anchor id + highlight class)
- Modify: `web/src/components/FeedArea.vue` (consume `jumpTarget` → around-load posts)
- Modify: `web/src/router/index.js` (`/m/:id` deep-link)
- Verify: `npm --prefix web run build` + manual E2E

**Interfaces:**
- Consumes: Tasks 3-5.

- [ ] **Step 1: Store jump flow**

In `web/src/stores/useAppStore.js` add refs `highlightMessageId = ref(null)`, `jumpTarget = ref(null)`, and:
```js
function scrollToMessage(id) {
  requestAnimationFrame(() => {
    const el = document.getElementById('msg-' + id)
    if (el) el.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })
}
async function loadMessagesAround(channelId, aroundId) {
  const res = await api.getChannelMessages(channelId, 50, null, aroundId) // around variant
  messages.value = res.messages || []
}
async function jumpToPermalink(messageId) {
  let info
  try { info = await api.resolvePermalink(messageId) } catch { showNotification(translate('chat.jumpFailed'), 'error'); return }
  const ch = info.channel
  if (currentGuild.value?.id !== ch.guild_id) await selectGuild(ch.guild_id)
  highlightMessageId.value = messageId
  if (ch.type === 'feed') {
    // hand off to FeedArea via jumpTarget; it loads posts around + scrolls
    jumpTarget.value = { channelId: ch.id, messageId }
    // ensure the feed channel is the current one so FeedArea reacts
    if (currentChannel.value?.id !== ch.id) await selectChannel(ch.id)
  } else {
    if (currentChannel.value?.id !== ch.id) { currentChannel.value = await api.getChannel(ch.id) }
    await loadMessagesAround(ch.id, messageId)
    scrollToMessage(messageId)
  }
  setTimeout(() => { highlightMessageId.value = null }, 2000)
}
```
Export `jumpToPermalink`, `highlightMessageId`, `jumpTarget`, `scrollToMessage` from the store. (Adapt to the store's actual `selectGuild`/`selectChannel`/`currentGuild`/`currentChannel`/`messages`/`showNotification`/`translate` names — they exist.)

- [ ] **Step 2: Anchor + highlight in MessageItem**

On the message root element in `web/src/components/MessageItem.vue`:
```html
<div class="message ..." :id="'msg-' + message.id" :class="{ 'msg-highlight': store.highlightMessageId === message.id }">
```
Add scoped CSS:
```css
.msg-highlight { animation: msgFlash 2s ease-out; }
@keyframes msgFlash { 0% { background: var(--accent, #5865f2); } 30% { background: color-mix(in srgb, var(--accent,#5865f2) 25%, transparent); } 100% { background: transparent; } }
```
(Since MessageItem renders both chat messages and wall posts, this anchors/highlights both surfaces.)

- [ ] **Step 3: FeedArea consumes jumpTarget**

In `web/src/components/FeedArea.vue`, watch `store.jumpTarget`; when it targets the current channel, load posts around it and scroll:
```js
watch(() => store.jumpTarget, async (jt) => {
  if (!jt || jt.channelId !== store.currentChannel?.id) return
  const res = await api.getPosts(jt.channelId, 20, null, jt.messageId) // around variant
  posts.value = res.posts || []
  store.jumpTarget = null
  store.scrollToMessage(jt.messageId)
}, { immediate: false })
```

- [ ] **Step 4: Router deep-link**

In `web/src/router/index.js` add a route that resolves the permalink then lands on the right section:
```js
{ path: '/m/:id', name: 'permalink', component: ChatView, meta: { section: 'chat', permalink: true } },
```
Add a `router.afterEach`/`beforeEach` (or handle in `App.vue`) that, when `to.name === 'permalink'`, calls `store.jumpToPermalink(Number(to.params.id))` and `router.replace('/')` (the jump itself navigates to the right guild/channel). Keep it minimal — the chip-click path is primary; this covers opening the URL directly.

- [ ] **Step 5: Verify build**

Run: `npm --prefix web run build` (exit 0).

- [ ] **Step 6: Manual E2E** (see finish step) then commit

```bash
git add web/src/stores/useAppStore.js web/src/components/MessageItem.vue web/src/components/FeedArea.vue web/src/router/index.js
git commit -m "feat(permalink): precise jump (around-load + scroll + highlight) + deep-link route"
```

---

## Notes for the implementer

- **Order:** Tasks 1→3 backend (strict-ish; 2 and 3 both touch message_service), Tasks 4→6 frontend (6 needs 3-5). Task 5's chip click depends on Task 6's `store.jumpToPermalink` — guard with optional-call until 6 lands.
- **`make check` after each backend task** (non-race fallback on this Windows box — no gcc).
- **MessageItem is shared** by chat (MessageList) and wall (FeedArea) — the anchor id, highlight, copy-link, and chip rendering there cover both surfaces at once. Only the around-LOAD differs (store.messages for chat vs FeedArea.posts for wall), coordinated via `jumpTarget`.
- **Do not `git add -A`** — leave the unrelated `web/src/components/learn/*.vue` changes uncommitted.

## E2E (finish step)

1. **Chat message:** in channel X post enough messages to push an old one off-screen; copy its link; go to channel Y, paste + send → preview chip (author/#X/snippet) appears; click → jumps to X, loads the old message, scrolls to it, highlight flashes.
2. **Wall post:** in a feed channel post several; share an old post; paste into a text channel; click chip → jumps to the feed channel, loads the post, scrolls + highlights.
3. **Non-member** clicks a chip for a channel they can't access → blocked with a toast.
```
