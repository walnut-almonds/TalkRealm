package repository

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// MessageRepository 訊息資料庫操作介面
type MessageRepository interface {
	Create(message *model.Message) error
	GetByID(id uint) (*model.Message, error)
	GetByNonce(userID uint, nonce string) (*model.Message, error)
	Update(message *model.Message) error
	Delete(id uint) error
	GetByChannelID(channelID uint, offset, limit int) ([]*model.Message, error)
	GetByChannelIDCursor(channelID, before uint, limit int) ([]*model.Message, error)
	GetByUserID(userID uint, offset, limit int) ([]*model.Message, error)
	// CountByUserInGuildsSince 回傳 userID 在指定 guildIDs 內各 guild 的訊息數量（since 之後）
	CountByUserInGuildsSince(userID uint, guildIDs []uint, since time.Time) (map[uint]int64, error)
	GetMessagesAround(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetPostsAround(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetPostsByChannelCursor(channelID, before uint, limit int) ([]*model.Message, error)
	GetCommentsByPostCursor(postID, before uint, limit int) ([]*model.Message, error)
	CountCommentsByPostIDs(ids []uint) (map[uint]int64, error)
	DeletePostCascade(postID uint) error
}

type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 建立訊息 repository
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Create 建立新訊息
func (r *messageRepository) Create(message *model.Message) error {
	return r.db.Create(message).Error
}

// GetByNonce 透過 userID + nonce 查詢訊息（用於冪等去重）
func (r *messageRepository) GetByNonce(userID uint, nonce string) (*model.Message, error) {
	var message model.Message

	err := r.db.Preload("User").Preload("Attachments.File").Preload("Reactions").
		Where("user_id = ? AND nonce = ?", userID, nonce).
		First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	return &message, nil
}

// GetByID 透過 ID 取得訊息
func (r *messageRepository) GetByID(id uint) (*model.Message, error) {
	var message model.Message

	err := r.db.Preload("User").
		Preload("Attachments.File").
		Preload("Reactions").
		First(&message, id).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("message not found")
		}

		return nil, err
	}

	return &message, nil
}

// Update 更新訊息
func (r *messageRepository) Update(message *model.Message) error {
	return r.db.Save(message).Error
}

// Delete 刪除訊息
func (r *messageRepository) Delete(id uint) error {
	return r.db.Delete(&model.Message{}, id).Error
}

// GetByChannelID 取得頻道的訊息（分頁）
func (r *messageRepository) GetByChannelID(
	channelID uint,
	offset, limit int,
) ([]*model.Message, error) {
	var messages []*model.Message

	err := r.db.
		Preload("User").Preload("Attachments.File").Preload("Reactions").
		Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error

	return messages, err
}

// GetByChannelIDCursor 取得頻道的訊息（cursor-based 分頁，從 before_id 往前取）
func (r *messageRepository) GetByChannelIDCursor(
	channelID uint,
	before uint,
	limit int,
) ([]*model.Message, error) {
	var messages []*model.Message

	q := r.db.
		Preload("User").Preload("Attachments.File").Preload("Reactions").
		Where("channel_id = ?", channelID).
		Order("id DESC"). // fetch DESC to apply cursor filter, then reverse
		Limit(limit)

	if before > 0 {
		q = q.Where("id < ?", before)
	}

	err := q.Find(&messages).Error
	if err != nil {
		return nil, err
	}
	// Reverse to chronological (ascending) order for the caller
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// aroundWindow fetches a page centered on aroundID: ceil(limit/2) messages with
// id <= aroundID (older side) + floor(limit/2) with id > aroundID (newer side),
// merged into ascending chronological order. postsOnly adds "parent_id IS NULL" for wall posts.
func (r *messageRepository) aroundWindow(
	channelID, aroundID uint,
	limit int,
	postsOnly bool,
) ([]*model.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	older := (limit + 1) / 2
	newer := limit - older

	base := r.db.Preload("User").
		Preload("Attachments.File").
		Preload("Reactions").
		Where("channel_id = ?", channelID)
	if postsOnly {
		base = base.Where("parent_id IS NULL")
	}

	var olderMsgs []*model.Message //nolint:prealloc // GORM Find 會整個覆寫這個 slice
	if err := base.Session(&gorm.Session{}).
		Where("id <= ?", aroundID).
		Order("id DESC").Limit(older).Find(&olderMsgs).Error; err != nil {
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

// GetMessagesAround 取得以 aroundID 為中心的訊息視窗（時間順序），用於 permalink 精準跳轉
func (r *messageRepository) GetMessagesAround(
	channelID, aroundID uint,
	limit int,
) ([]*model.Message, error) {
	return r.aroundWindow(channelID, aroundID, limit, false)
}

// GetPostsAround 取得以 aroundID 為中心的貼文視窗（parent_id IS NULL，時間順序）
func (r *messageRepository) GetPostsAround(
	channelID, aroundID uint,
	limit int,
) ([]*model.Message, error) {
	return r.aroundWindow(channelID, aroundID, limit, true)
}

// GetByUserID 取得使用者的訊息（分頁）
func (r *messageRepository) GetByUserID(userID uint, offset, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	err := r.db.
		Preload("Channel").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&messages).Error

	return messages, err
}

// CountByUserInGuildsSince 計算 userID 在各 guild 頻道內 since 之後的訊息數量
func (r *messageRepository) CountByUserInGuildsSince(
	userID uint,
	guildIDs []uint,
	since time.Time,
) (map[uint]int64, error) {
	if len(guildIDs) == 0 {
		return map[uint]int64{}, nil
	}

	type row struct {
		GuildID uint
		Count   int64
	}

	var rows []row

	err := r.db.
		Model(&model.Message{}).
		Select("channels.guild_id AS guild_id, COUNT(*) AS count").
		Joins("JOIN channels ON channels.id = messages.channel_id").
		Where("messages.user_id = ? AND channels.guild_id IN ? AND messages.created_at >= ?",
			userID, guildIDs, since).
		Group("channels.guild_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint]int64, len(guildIDs))
	for _, row := range rows {
		result[row.GuildID] = row.Count
	}

	return result, nil
}

// GetPostsByChannelCursor 取得 feed 頻道的貼文（parent_id IS NULL，新到舊 cursor 分頁）
func (r *messageRepository) GetPostsByChannelCursor(
	channelID, before uint,
	limit int,
) ([]*model.Message, error) {
	var posts []*model.Message

	q := r.db.
		Preload("User").Preload("Attachments.File").Preload("Reactions").
		Where("channel_id = ? AND parent_id IS NULL", channelID).
		Order("id DESC").
		Limit(limit)

	if before > 0 {
		q = q.Where("id < ?", before)
	}

	// 貼文列表保持新到舊，不反轉
	return posts, q.Find(&posts).Error
}

// GetCommentsByPostCursor 取得某貼文的留言（parent_id = postID，回傳時間順序）
func (r *messageRepository) GetCommentsByPostCursor(
	postID, before uint,
	limit int,
) ([]*model.Message, error) {
	var comments []*model.Message

	q := r.db.
		Preload("User").Preload("Attachments.File").Preload("Reactions").
		Where("parent_id = ?", postID).
		Order("id DESC").
		Limit(limit)

	if before > 0 {
		q = q.Where("id < ?", before)
	}

	if err := q.Find(&comments).Error; err != nil {
		return nil, err
	}
	// Reverse to chronological (ascending) order for the caller
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}

	return comments, nil
}

// CountCommentsByPostIDs 批次計算每則貼文的留言數（避免 N+1）
func (r *messageRepository) CountCommentsByPostIDs(ids []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	type row struct {
		ParentID uint
		Cnt      int64
	}

	var rows []row

	err := r.db.Model(&model.Message{}).
		Select("parent_id, COUNT(*) AS cnt").
		Where("parent_id IN ?", ids).
		Group("parent_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, x := range rows {
		out[x.ParentID] = x.Cnt
	}

	return out, nil
}

// DeletePostCascade 於單一 transaction 內刪除貼文、其留言、及全部相關讚與表情回應。
// message_reactions 有指向 messages 的外鍵，漏刪會讓整個刪除以 23503 失敗。
func (r *messageRepository) DeletePostCascade(postID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var commentIDs []uint //nolint:prealloc // Pluck 會整個覆寫這個 slice
		if err := tx.Model(&model.Message{}).
			Where("parent_id = ?", postID).
			Pluck("id", &commentIDs).Error; err != nil {
			return err
		}

		ids := append(commentIDs, postID) //nolint:gocritic

		if err := tx.Where("message_id IN ?", ids).
			Delete(&model.MessageLike{}).Error; err != nil {
			return err
		}

		if err := tx.Where("message_id IN ?", ids).
			Delete(&model.MessageReaction{}).Error; err != nil {
			return err
		}

		if err := tx.Where("parent_id = ?", postID).
			Delete(&model.Message{}).Error; err != nil {
			return err
		}

		return tx.Delete(&model.Message{}, postID).Error
	})
}
