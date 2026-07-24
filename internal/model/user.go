package model

import (
	"time"
)

// User 使用者模型
type User struct {
	ID            uint      `gorm:"primarykey"           json:"id"`
	Username      string    `gorm:"uniqueIndex;not null" json:"username"`
	Email         string    `gorm:"uniqueIndex;not null" json:"email"`
	Password      string    `gorm:"default:null"         json:"-"`
	Nickname      string    `                            json:"nickname"`
	Avatar        string    `                            json:"avatar"`
	Status        string    `gorm:"default:'offline'"    json:"status"`         // online, idle, dnd, invisible (also accepts offline, busy, away)
	PreferredLang string    `gorm:"default:'zh'"         json:"preferred_lang"` // zh, zh-tw, ja, en (translation target)
	UILocale      string    `gorm:"type:varchar(10)"     json:"ui_locale"`      // zh, zh-tw, ja, en (UI locale, empty = not set)
	CreatedAt     time.Time `                            json:"created_at"`
	UpdatedAt     time.Time `                            json:"updated_at"`
}

// Friendship 好友關係模型
// Status: "pending"（申請中）/ "accepted"（已接受）/ "blocked"（已封鎖）
type Friendship struct {
	ID          uint      `gorm:"primarykey"                               json:"id"`
	RequesterID uint      `gorm:"not null;uniqueIndex:idx_friendship_pair" json:"requester_id"`
	Requester   User      `gorm:"foreignKey:RequesterID"                   json:"requester,omitempty"`
	AddresseeID uint      `gorm:"not null;uniqueIndex:idx_friendship_pair" json:"addressee_id"`
	Addressee   User      `gorm:"foreignKey:AddresseeID"                   json:"addressee,omitempty"`
	Status      string    `gorm:"not null;default:'pending'"               json:"status"`
	CreatedAt   time.Time `                                                json:"created_at"`
	UpdatedAt   time.Time `                                                json:"updated_at"`
}

type UserOAuthProvider struct {
	ID         uint      `gorm:"primarykey"        json:"id"`
	UserID     uint      `gorm:"not null;index"    json:"user_id"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
	Provider   string    `gorm:"not null"          json:"provider"`    // google, github, discord ...
	ProviderID string    `gorm:"not null"          json:"provider_id"` // 各家給的 subject ID
	CreatedAt  time.Time `                         json:"created_at"`
	UpdatedAt  time.Time `                         json:"updated_at"`
}

// Guild 社群/伺服器模型
type Guild struct {
	ID          uint      `gorm:"primarykey"         json:"id"`
	Name        string    `gorm:"not null"           json:"name"`
	Description string    `                          json:"description"`
	Icon        string    `                          json:"icon"`
	OwnerID     uint      `gorm:"not null"           json:"owner_id"`
	Owner       User      `gorm:"foreignKey:OwnerID" json:"owner"`
	CreatedAt   time.Time `                          json:"created_at"`
	UpdatedAt   time.Time `                          json:"updated_at"`
}

// Channel 頻道模型（支援 guild 頻道與 DM 頻道，GuildID 為 nil 時代表 DM 頻道）
type Channel struct {
	ID           uint                  `gorm:"primarykey"           json:"id"`
	GuildID      *uint                 `gorm:"index"                json:"guild_id"` // nil for DM channels
	Guild        *Guild                `gorm:"foreignKey:GuildID"   json:"-"`
	Name         string                `                            json:"name"` // empty for DM channels
	Type         string                `gorm:"not null"             json:"type"` // text, voice, dm
	Topic        string                `                            json:"topic"`
	Position     int                   `gorm:"default:0"            json:"position"`
	Participants []*ChannelParticipant `gorm:"foreignKey:ChannelID" json:"participants,omitempty"`
	CreatedAt    time.Time             `                            json:"created_at"`
	UpdatedAt    time.Time             `                            json:"updated_at"`
}

// ChannelParticipant DM 頻道的參與者（僅 type='dm' 的 Channel 使用）
type ChannelParticipant struct {
	ID        uint      `gorm:"primarykey"                               json:"id"`
	ChannelID uint      `gorm:"not null;uniqueIndex:idx_cp_channel_user" json:"channel_id"`
	Channel   Channel   `gorm:"foreignKey:ChannelID"                     json:"-"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_cp_channel_user" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID"                        json:"user"`
	CreatedAt time.Time `                                                json:"created_at"`
}

// Message 訊息模型
type Message struct {
	ID          uint                `gorm:"primarykey"                              json:"id"`
	ChannelID   uint                `gorm:"not null"                                json:"channel_id"`
	Channel     Channel             `gorm:"foreignKey:ChannelID"                    json:"-"`
	UserID      uint                `gorm:"not null"                                json:"user_id"`
	User        User                `gorm:"foreignKey:UserID"                       json:"user"`
	Content     string              `gorm:"not null"                                json:"content"`
	Type        string              `gorm:"default:'text'"                          json:"type"`        // text, image, file
	IsEdited    bool                `gorm:"default:false"                           json:"is_edited"`   // 是否被編輯過
	Nonce       string              `gorm:"uniqueIndex:idx_user_nonce;default:null" json:"nonce"`       // client 產生的冪等 key（可選）
	ParentID    *uint               `gorm:"index"                                   json:"parent_id"`   // nil = 貼文/一般訊息；有值 = 留言，指向貼文
	Attachments []MessageAttachment `gorm:"foreignKey:MessageID"                    json:"attachments"` // 附件
	CreatedAt   time.Time           `                                               json:"created_at"`
	UpdatedAt   time.Time           `                                               json:"updated_at"`
}

// File 檔案儲存記錄
type File struct {
	ID             uint       `gorm:"primarykey"           json:"id"`
	UserID         uint       `gorm:"not null;index"       json:"user_id"`
	User           User       `gorm:"foreignKey:UserID"    json:"-"`
	Filename       string     `gorm:"not null"             json:"filename"`    // 原始檔名
	StorageKey     string     `gorm:"not null;uniqueIndex" json:"storage_key"` // Minio object key
	ContentType    string     `gorm:"not null"             json:"content_type"`
	Size           int64      `gorm:"not null"             json:"size"`             // bytes
	Status         string     `gorm:"default:'pending'"    json:"status"`           // pending, active, deleted
	ExpiresAt      *time.Time `                            json:"expires_at"`       // nil = 永久
	LastAccessedAt time.Time  `gorm:"default:now()"        json:"last_accessed_at"` // LRU 依據
	CreatedAt      time.Time  `                            json:"created_at"`
	UpdatedAt      time.Time  `                            json:"updated_at"`
}

// MessageAttachment 訊息附件關聯表
type MessageAttachment struct {
	ID        uint      `gorm:"primarykey"           json:"id"`
	MessageID uint      `gorm:"not null;index"       json:"message_id"`
	Message   Message   `gorm:"foreignKey:MessageID" json:"-"`
	FileID    uint      `gorm:"not null"             json:"file_id"`
	File      File      `gorm:"foreignKey:FileID"    json:"file"`
	CreatedAt time.Time `                            json:"created_at"`
}

// MessageLike 貼文/留言的按讚（一人對一則只能讚一次）
type MessageLike struct {
	ID        uint      `gorm:"primarykey"                                 json:"id"`
	MessageID uint      `gorm:"not null;uniqueIndex:idx_like_message_user" json:"message_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_like_message_user" json:"user_id"`
	CreatedAt time.Time `                                                  json:"created_at"`
}

// GuildMember 社群成員模型
type GuildMember struct {
	ID        uint      `gorm:"primarykey"         json:"id"`
	GuildID   uint      `gorm:"not null"           json:"guild_id"`
	Guild     Guild     `gorm:"foreignKey:GuildID" json:"-"`
	UserID    uint      `gorm:"not null"           json:"user_id"`
	User      User      `gorm:"foreignKey:UserID"  json:"user"`
	Nickname  string    `                          json:"nickname"`
	Role      string    `gorm:"default:'member'"   json:"role"` // owner, admin, member
	JoinedAt  time.Time `                          json:"joined_at"`
	CreatedAt time.Time `                          json:"created_at"`
	UpdatedAt time.Time `                          json:"updated_at"`
}

// GuildInvite 社群邀請碼模型
type GuildInvite struct {
	ID        uint       `gorm:"primarykey"           json:"id"`
	GuildID   uint       `gorm:"not null;index"       json:"guild_id"`
	Guild     Guild      `gorm:"foreignKey:GuildID"   json:"guild"`
	CreatorID uint       `gorm:"not null"             json:"creator_id"`
	Creator   User       `gorm:"foreignKey:CreatorID" json:"creator"`
	Code      string     `gorm:"uniqueIndex;not null" json:"code"`
	MaxUses   int        `gorm:"default:0"            json:"max_uses"` // 0 = unlimited
	Uses      int        `gorm:"default:0"            json:"uses"`
	ExpiresAt *time.Time `                            json:"expires_at"` // nil = no expiry
	CreatedAt time.Time  `                            json:"created_at"`
	UpdatedAt time.Time  `                            json:"updated_at"`
}

// RefreshToken Refresh Token 模型
type RefreshToken struct {
	ID        uint      `gorm:"primarykey"           json:"id"`
	UserID    uint      `gorm:"not null;index"       json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"not null"             json:"expires_at"`
	Revoked   bool      `gorm:"default:false"        json:"revoked"`
	CreatedAt time.Time `                            json:"created_at"`
}

// ChannelReadState 記錄每個使用者在每個頻道的已讀位置
type ChannelReadState struct {
	ID                uint      `gorm:"primarykey"                                       json:"id"`
	UserID            uint      `gorm:"not null;uniqueIndex:idx_read_state_user_channel" json:"user_id"`
	User              User      `gorm:"foreignKey:UserID"                                json:"-"`
	ChannelID         uint      `gorm:"not null;uniqueIndex:idx_read_state_user_channel" json:"channel_id"`
	Channel           Channel   `gorm:"foreignKey:ChannelID"                             json:"-"`
	LastReadMessageID uint      `gorm:"default:0"                                        json:"last_read_message_id"`
	UpdatedAt         time.Time `                                                        json:"updated_at"`
}

// MessageMention 記錄訊息中的 @提及
type MessageMention struct {
	ID          uint      `gorm:"primarykey"                         json:"id"`
	MessageID   uint      `gorm:"not null;index:idx_mention_message" json:"message_id"`
	Message     Message   `gorm:"foreignKey:MessageID"               json:"-"`
	UserID      uint      `gorm:"not null;index:idx_mention_user"    json:"user_id"`
	User        User      `gorm:"foreignKey:UserID"                  json:"user,omitempty"`
	MentionType string    `gorm:"not null;default:'user'"            json:"mention_type"` // user, here, everyone
	CreatedAt   time.Time `                                          json:"created_at"`
}

// MessageTranslation 訊息翻譯結果（多語全存，migration-friendly：每筆訊息一列，未來可直接遷至 Cassandra）
// 新增語言步驟：加欄位 → 更新 service.targetLangs → 執行 DB migration
type MessageTranslation struct {
	MessageID         uint      `gorm:"primarykey"           json:"message_id"`
	Message           Message   `gorm:"foreignKey:MessageID" json:"-"`
	OriginalLang      string    `gorm:"not null"             json:"original_lang"` // zh, zh-tw, ja, en
	ContentZH         string    `gorm:"type:text"            json:"content_zh"`
	ContentZHTW       string    `gorm:"type:text"            json:"content_zh_tw"` // 繁體中文
	ContentJA         string    `gorm:"type:text"            json:"content_ja"`
	ContentEN         string    `gorm:"type:text"            json:"content_en"`
	TranslationStatus string    `gorm:"default:'pending'"    json:"translation_status"` // pending, completed, failed
	TranslatedAt      time.Time `gorm:"default:now()"        json:"translated_at"`
}

// GameState 猜測遊戲狀態記錄
type GameState struct {
	ID              uint      `gorm:"primarykey"           json:"id"`
	MessageID       uint      `gorm:"not null;index"       json:"message_id"`
	Message         Message   `gorm:"foreignKey:MessageID" json:"-"`
	GuesserID       uint      `gorm:"not null;index"       json:"guesser_id"`
	Guesser         User      `gorm:"foreignKey:GuesserID" json:"-"`
	HiddenLang      string    `gorm:"not null"             json:"hidden_lang"` // which lang was hidden (zh, ja, en)
	GuessContent    string    `gorm:"type:text;not null"   json:"guess_content"`
	Mode            string    `gorm:"default:'semantic'"   json:"mode"` // semantic (only mode for now)
	IsCorrect       bool      `gorm:"default:false"        json:"is_correct"`
	SimilarityScore float64   `gorm:"default:0"            json:"similarity_score"` // LLM similarity 0..1
	GuessedAt       time.Time `gorm:"default:now()"        json:"guessed_at"`
}
