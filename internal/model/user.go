package model

import (
	"time"
)

// User 使用者模型
type User struct {
	ID        uint      `gorm:"primarykey"           json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null"             json:"-"`
	Nickname  string    `                            json:"nickname"`
	Avatar    string    `                            json:"avatar"`
	Status    string    `gorm:"default:'offline'"    json:"status"` // online, offline, busy, away
	CreatedAt time.Time `                            json:"created_at"`
	UpdatedAt time.Time `                            json:"updated_at"`
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

// Channel 頻道模型
type Channel struct {
	ID        uint      `gorm:"primarykey"         json:"id"`
	GuildID   uint      `gorm:"not null"           json:"guild_id"`
	Guild     Guild     `gorm:"foreignKey:GuildID" json:"guild"`
	Name      string    `gorm:"not null"           json:"name"`
	Type      string    `gorm:"not null"           json:"type"` // text, voice
	Topic     string    `                          json:"topic"`
	Position  int       `gorm:"default:0"          json:"position"`
	CreatedAt time.Time `                          json:"created_at"`
	UpdatedAt time.Time `                          json:"updated_at"`
}

// Message 訊息模型
type Message struct {
	ID        uint      `gorm:"primarykey"           json:"id"`
	ChannelID uint      `gorm:"not null"             json:"channel_id"`
	Channel   Channel   `gorm:"foreignKey:ChannelID" json:"channel"`
	UserID    uint      `gorm:"not null"             json:"user_id"`
	User      User      `gorm:"foreignKey:UserID"    json:"user"`
	Content   string    `gorm:"not null"             json:"content"`
	Type      string    `gorm:"default:'text'"       json:"type"` // text, image, file
	CreatedAt time.Time `                            json:"created_at"`
	UpdatedAt time.Time `                            json:"updated_at"`
}

// GuildMember 社群成員模型
type GuildMember struct {
	ID        uint      `gorm:"primarykey"         json:"id"`
	GuildID   uint      `gorm:"not null"           json:"guild_id"`
	Guild     Guild     `gorm:"foreignKey:GuildID" json:"guild"`
	UserID    uint      `gorm:"not null"           json:"user_id"`
	User      User      `gorm:"foreignKey:UserID"  json:"user"`
	Nickname  string    `                          json:"nickname"`
	Role      string    `gorm:"default:'member'"   json:"role"` // owner, admin, member
	JoinedAt  time.Time `                          json:"joined_at"`
	CreatedAt time.Time `                          json:"created_at"`
	UpdatedAt time.Time `                          json:"updated_at"`
}
