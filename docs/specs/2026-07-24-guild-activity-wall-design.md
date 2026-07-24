# 社群內動態牆(Guild Activity Wall)設計

- 日期:2026-07-24
- 狀態:設計定案,待實作
- 範圍:單一 guild 內的動態牆,與現有聊天高度耦合

## 目標

在現有 Discord 式聊天平台上,加入 Twitter/FB 式的「動態牆」:一般成員可在牆上發貼文,其他成員可**按讚**、**單層留言**、貼文可**帶圖**、作者可**自行編輯/刪除**。牆以一種新的頻道類型呈現,與文字/語音頻道並排。

## Scope & Future Boundary(必讀)

本設計**只涵蓋 guild 內動態牆**,並刻意與聊天(`Message`/`Channel`)耦合、白嫖整套即時廣播與附件機制。

一開始討論到的兩件事是**兩個不同產品**:

| | 社群內動態牆(本設計) | 跨社群個人化 feed(未來,不在本設計) |
|---|---|---|
| 本質 | 聊天的延伸 | 獨立產品 |
| 資料 | 複用 `Message` / `Channel` | 自己的 model/repo/service/handler |
| 生命週期 | 綁 guild | 綁 user,跨 guild |
| 排序 | 時間新到舊 | 追蹤關係 + 演算法 + fan-out |
| 結構 | 高度耦合 | 仿 `learn` 模組,可整包抽離 |

**約束:** 未來的跨社群個人化 feed **不得**回頭修改本設計的資料表或耦合方式;它應在**模組邊界**上讀取資料(例如聚合各 guild 牆的內容),不共用 schema。兩者不共用 model 是刻意的特性,不是重複。API 命名空間 `/posts/` 保留給該未來模組;本設計所有 API 收在 `/channels/:id/` 底下。

## §1 資料模型

改動極小:改 1 個既有欄位語意 + 加 1 欄 + 加 1 表。

```
Channel.Type      新增合法值 'feed'（無 schema 變動,多一種字串值）

Message.ParentID  *uint  `gorm:"index"`   （新 nullable 欄位）
   · 貼文  → ParentID = nil
   · 留言  → ParentID = 該貼文的 message id

MessageLike（新表,貼文與留言共用）
   · ID         uint  primarykey
   · MessageID  uint  uniqueIndex(message_id, user_id)
   · UserID     uint  uniqueIndex(message_id, user_id)   ← 一人對一則只能讚一次
   · CreatedAt  time.Time
```

### 查詢對照

| 動作 | Query |
|---|---|
| 牆的貼文列表 | feed 頻道內 `parent_id IS NULL`、`id < before`、新到舊、limit N |
| 某貼文的留言 | `parent_id = 貼文id`、舊到新 |
| 讚數 | `SELECT COUNT(*) FROM message_likes WHERE message_id = ?` |
| 我讚了嗎 | `MessageLike` exists(message_id, user_id) |

### 白嫖(完全不用碰)

附件(`MessageAttachment`)、編輯旗標(`IsEdited`)、@mention、翻譯、WebSocket 廣播機制。

### 邊界決策

1. **讚數計數:** MVP 直接 `COUNT(*)`,不加 denormalized 計數欄。
   `ponytail: COUNT 即可,真的變熱門再加 like_count 快取欄`。
2. **刪貼文的 cascade:** 硬刪,貼文連同其留言與所有相關讚一起清,不留孤兒(見 §2)。
3. **`ParentID` 的附帶價值:** 之後聊天室要做「回覆某則訊息」可直接複用此欄,非為本功能專屬。
4. **開放 feed 頻道類型:** 建/改頻道的 type 守衛目前寫死只准 `text`/`voice`(`channel_service.go:96`、`:207`),兩處各加 `'feed'` 即開放建立動態牆頻道。這是唯一為「能建牆」所需的既有改動;頻道建立/排序/側邊欄/權限全複用。

### 既有潛在問題(不在本設計範圍,僅記錄)

現有 `messageRepo.Delete` 只刪 `Message` 本身,**未清 `MessageAttachment` / `MessageMention`**(structs 無 `OnDelete:CASCADE` tag),既有聊天刪訊息已在留孤兒關聯列。本設計不修此既有行為,但**貼文的 cascade 會做對的事**(見下),避免留下看不到的孤兒留言。

## §2 API + 即時事件

貼文與留言本體就是 `Message`,因此**編輯/刪除/發文全部複用既有 endpoint**,不新造。既有路由慣例是「頻道集合掛 `/channels/:id/...`、單則訊息操作掛 `/messages/:id/...`」(如 `/messages/:id/translation`),動態牆完全跟隨,不改既有 API,也不碰保留給未來跨社群模組的 `/posts/` 命名空間。

### 複用既有 endpoint(不新造)

| 動作 | 複用 | 備註 |
|---|---|---|
| 發貼文 | `POST /channels/:id/messages` | `parent_id` 省略或 nil |
| 發留言 | `POST /channels/:id/messages` | 帶 `parent_id=貼文id`(建立請求新增可選欄位) |
| 編輯貼文/留言 | `PUT /messages/:id` | 原封複用 |
| 刪貼文/留言 | `DELETE /messages/:id` | **加 cascade 分支**(見下) |

### 新增 endpoint(僅三個,貼合既有慣例)

| Method | Path | 對照既有 | 說明 |
|---|---|---|---|
| `GET` | `/channels/:id/posts?before=&limit=` | 仿 `/channels/:id/messages` | 牆列表(`parent_id IS NULL`),新到舊,cursor 分頁;每筆帶讚數、我讚了嗎、留言數 |
| `GET` | `/messages/:id/comments?before=&limit=` | 仿 `/messages/:id/translation` | 某貼文留言,舊到新 |
| `PUT` / `DELETE` | `/messages/:id/like` | 仿 `/messages/:id/...` 子資源 | 按讚(idempotent)/ 收回讚 |

### 服務層

發文/留言/編輯就是帶(或不帶)`ParentID` 呼叫既有 `messageService`,在其上加薄層,**不另開平行 service**。權限(成員可發、僅作者或 guild admin/owner 可刪)複用既有 message 邏輯。

唯一需改到既有 handler 的點:**`DELETE /messages/:id` 加一個分支**——若目標是 feed 頻道的貼文(`parent_id=nil`)走 cascade(連留言+讚);若是留言或一般聊天訊息則現有行為 + 清自己的讚。

MVP 預設 feed 頻道所有成員皆可發文;之後可再細分(例如某 feed 頻道限 admin 發公告),不在本設計。

### Cascade(刪貼文)

單一 transaction 內:

```
DELETE FROM message_likes WHERE message_id IN (貼文id + 其所有留言id)
DELETE FROM messages      WHERE parent_id = 貼文id       -- 留言
DELETE FROM messages      WHERE id = 貼文id              -- 貼文本身
```

刪「留言」則僅:`DELETE likes WHERE message_id = 留言id` + 刪該留言。

### 即時事件

複用現有三個事件,前端靠 payload 的 `parent_id` 判斷是貼文還是留言。僅 `post_like` 為新事件。

| 事件 | 何時 | 廣播 | payload |
|---|---|---|---|
| `message_create` | 發貼文或留言 | `BroadcastToChannel(feedChannelID)` | 完整 message(含 `parent_id`) |
| `message_update` | 編輯 | 同上 | 更新後 message |
| `message_delete` | 刪(cascade 時貼文發一次,前端整塊移除) | 同上 | `{message_id, channel_id}` |
| `post_like` | 讚數變動(讚/收回) | 同上 | `{message_id, like_count}` |

## §3 前端呈現(Vue 3)

牆是 `Channel.Type='feed'`,在側邊欄即一個頻道項目(白嫖頻道列表 UI),點進後**內容區依 `channel.type` 分流**:

- `text` → 現有聊天元件
- `feed` → 新的 `FeedView`

### FeedView

- 頂部發文框:content + 貼圖(複用現有上傳流程)。
- 貼文卡片列表:新到舊,往下捲以 `before=` cursor 載入(複用聊天的無限捲 composable,方向相反)。
- 貼文卡片:作者/時間、內文、圖、`♥ 讚數`、`💬 留言數`。點留言數展開單層留言 + 留言框。

### 即時處理(feed 頻道的 WS)

| 收到 | 動作 |
|---|---|
| `message_create`,`parent_id=nil` | 插到列表頂端 |
| `message_create`,`parent_id` 有值 | 塞進對應貼文的留言區 |
| `message_update` | 就地更新對應貼文/留言 |
| `message_delete` | 移除對應貼文(整塊)或留言 |
| `post_like` | 更新該卡片讚數 |

### 視覺

依 `DESIGN.md` 的 Kinetic Noir 母本(低調沉穩、禁 Discord DNA)。複用既有訊息元件樣式為基礎,不另立設計語言。

## 測試

- **後端**(table-driven,照既有 `*_service_test.go` 風格):
  - 發貼文 / 發留言(`parent_id` 正確)
  - 牆列表只回 `parent_id IS NULL`、cursor 分頁方向正確
  - 按讚 idempotent、收回讚、`COUNT` 正確、一人一讚 unique 約束
  - 刪貼文 cascade:留言與讚一併清除;刪留言只清自己
  - 權限:非作者非 admin 不可刪他人貼文
- **前端**:`channel.type` 分流渲染;WS 事件依 `parent_id` 分派到列表/留言區。

## 不做(YAGNI)

巢狀留言、轉發/quote、跨頻道跳轉 permalink、跨社群個人化 feed、denormalized 讚數、like_count 快取、演算法排序、公開分享網址。
