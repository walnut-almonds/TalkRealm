package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// DMRepository 私訊資料庫操作介面
type DMRepository interface {
	// 頻道操作
	GetOrCreateChannel(user1ID, user2ID uint) (*model.DirectMessageChannel, error)
	GetChannelByID(id uint) (*model.DirectMessageChannel, error)
	ListChannelsForUser(userID uint) ([]*model.DirectMessageChannel, error)
	IsParticipant(channelID, userID uint) (bool, error)

	// 訊息操作
	CreateMessage(msg *model.DirectMessage) error
	GetMessageByNonce(senderID uint, nonce string) (*model.DirectMessage, error)
	ListMessages(dmChannelID, before uint, limit int) ([]*model.DirectMessage, error)
}

type dmRepository struct {
	db *gorm.DB
}

// NewDMRepository 建立 DM repository
func NewDMRepository(db *gorm.DB) DMRepository {
	return &dmRepository{db: db}
}

// GetOrCreateChannel 取得或建立兩位使用者之間的頻道
// 強制 user1ID < user2ID，確保每對使用者只有一個紀錄
func (r *dmRepository) GetOrCreateChannel(user1ID, user2ID uint) (*model.DirectMessageChannel, error) {
	// 正規化順序
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var ch model.DirectMessageChannel

	err := r.db.Preload("User1").Preload("User2").
		Where("user1_id = ? AND user2_id = ?", user1ID, user2ID).
		First(&ch).Error

	if err == nil {
		return &ch, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 不存在則建立
	ch = model.DirectMessageChannel{User1ID: user1ID, User2ID: user2ID}

	if err := r.db.Create(&ch).Error; err != nil {
		return nil, err
	}

	// 重新查詢以載入關聯
	if err := r.db.Preload("User1").Preload("User2").First(&ch, ch.ID).Error; err != nil {
		return nil, err
	}

	return &ch, nil
}

// GetChannelByID 透過 ID 取得 DM 頻道
func (r *dmRepository) GetChannelByID(id uint) (*model.DirectMessageChannel, error) {
	var ch model.DirectMessageChannel

	err := r.db.Preload("User1").Preload("User2").First(&ch, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dm channel not found")
		}

		return nil, err
	}

	return &ch, nil
}

// ListChannelsForUser 列出使用者參與的所有 DM 頻道
func (r *dmRepository) ListChannelsForUser(userID uint) ([]*model.DirectMessageChannel, error) {
	var channels []*model.DirectMessageChannel

	err := r.db.Preload("User1").Preload("User2").
		Where("user1_id = ? OR user2_id = ?", userID, userID).
		Order("updated_at DESC").
		Find(&channels).Error

	return channels, err
}

// IsParticipant 確認使用者是否為 DM 頻道的參與者
func (r *dmRepository) IsParticipant(channelID, userID uint) (bool, error) {
	var count int64

	err := r.db.Model(&model.DirectMessageChannel{}).
		Where("id = ? AND (user1_id = ? OR user2_id = ?)", channelID, userID, userID).
		Count(&count).Error

	return count > 0, err
}

// CreateMessage 建立私訊訊息
func (r *dmRepository) CreateMessage(msg *model.DirectMessage) error {
	if err := r.db.Create(msg).Error; err != nil {
		return err
	}

	return r.db.Preload("Sender").First(msg, msg.ID).Error
}

// GetMessageByNonce 透過 senderID + nonce 查詢訊息（冪等去重）
func (r *dmRepository) GetMessageByNonce(senderID uint, nonce string) (*model.DirectMessage, error) {
	var msg model.DirectMessage

	err := r.db.Preload("Sender").
		Where("sender_id = ? AND nonce = ?", senderID, nonce).
		First(&msg).Error
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// ListMessages 取得 DM 頻道的訊息（cursor-based 分頁，before=0 表示最新）
func (r *dmRepository) ListMessages(dmChannelID, before uint, limit int) ([]*model.DirectMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	q := r.db.Preload("Sender").
		Where("dm_channel_id = ?", dmChannelID).
		Order("created_at DESC").
		Limit(limit)

	if before > 0 {
		q = q.Where("id < ?", before)
	}

	var msgs []*model.DirectMessage
	if err := q.Find(&msgs).Error; err != nil {
		return nil, err
	}

	// 反轉使訊息由舊到新排列
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}
