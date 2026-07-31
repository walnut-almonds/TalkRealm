package model

import (
	"time"

	"gorm.io/gorm"
)

// Follow 單向追蹤關係（feed 模組自有，不共用 Friendship）
type Follow struct {
	ID         uint      `gorm:"primarykey"                                 json:"id"`
	FollowerID uint      `gorm:"not null;uniqueIndex:idx_follow_pair;index" json:"follower_id"`
	FolloweeID uint      `gorm:"not null;uniqueIndex:idx_follow_pair;index" json:"followee_id"`
	Follower   User      `gorm:"foreignKey:FollowerID"                      json:"follower,omitempty"`
	Followee   User      `gorm:"foreignKey:FolloweeID"                      json:"followee,omitempty"`
	CreatedAt  time.Time `                                                  json:"created_at"`
}

// FeedPost 個人貼文（跨社群，不屬於任何 guild）
type FeedPost struct {
	ID          uint                 `gorm:"primarykey"          json:"id"`
	AuthorID    uint                 `gorm:"not null;index"      json:"author_id"`
	Author      User                 `gorm:"foreignKey:AuthorID" json:"author"`
	Content     string               `gorm:"not null"            json:"content"`
	IsEdited    bool                 `gorm:"default:false"       json:"is_edited"`
	Attachments []FeedPostAttachment `gorm:"foreignKey:PostID"   json:"attachments"`
	CreatedAt   time.Time            `                           json:"created_at"`
	UpdatedAt   time.Time            `                           json:"updated_at"`
}

// AfterFind 確保 Attachments 序列化為 [] 而非 null
func (p *FeedPost) AfterFind(_ *gorm.DB) error {
	if p.Attachments == nil {
		p.Attachments = []FeedPostAttachment{}
	}

	return nil
}

// FeedComment 貼文的單層留言
type FeedComment struct {
	ID        uint      `gorm:"primarykey"          json:"id"`
	PostID    uint      `gorm:"not null;index"      json:"post_id"`
	AuthorID  uint      `gorm:"not null"            json:"author_id"`
	Author    User      `gorm:"foreignKey:AuthorID" json:"author"`
	Content   string    `gorm:"not null"            json:"content"`
	IsEdited  bool      `gorm:"default:false"       json:"is_edited"`
	CreatedAt time.Time `                           json:"created_at"`
	UpdatedAt time.Time `                           json:"updated_at"`
}

// FeedPostLike 貼文按讚（一人對一貼文只能讚一次）
type FeedPostLike struct {
	ID        uint      `gorm:"primarykey"                             json:"id"`
	PostID    uint      `gorm:"not null;uniqueIndex:idx_feedlike_pair" json:"post_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_feedlike_pair" json:"user_id"`
	CreatedAt time.Time `                                              json:"created_at"`
}

// FeedCommentLike 留言按讚（一人對一留言只能讚一次）
type FeedCommentLike struct {
	ID        uint      `gorm:"primarykey"                                    json:"id"`
	CommentID uint      `gorm:"not null;uniqueIndex:idx_feedcommentlike_pair" json:"comment_id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_feedcommentlike_pair" json:"user_id"`
	CreatedAt time.Time `                                                     json:"created_at"`
}

// FeedPostAttachment 貼文附件（連到通用 File）
type FeedPostAttachment struct {
	ID        uint      `gorm:"primarykey"        json:"id"`
	PostID    uint      `gorm:"not null;index"    json:"post_id"`
	FileID    uint      `gorm:"not null"          json:"file_id"`
	File      File      `gorm:"foreignKey:FileID" json:"file"`
	CreatedAt time.Time `                         json:"created_at"`
}
