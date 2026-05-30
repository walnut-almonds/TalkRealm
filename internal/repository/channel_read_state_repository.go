package repository

import (
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChannelUnreadCount 單一頻道的未讀統計
type ChannelUnreadCount struct {
	ChannelID    uint  `json:"channel_id"`
	GuildID      *uint `json:"guild_id"`
	UnreadCount  int64 `json:"unread_count"`
	MentionCount int64 `json:"mention_count"`
}

// ChannelReadStateRepository 已讀狀態資料庫操作介面
type ChannelReadStateRepository interface {
	// Upsert 更新（或建立）使用者在某頻道的已讀位置
	Upsert(userID, channelID, lastMessageID uint) error
	// GetLastRead 取得使用者在某頻道最後已讀的 message ID（不存在時回傳 0, nil）
	GetLastRead(userID, channelID uint) (uint, error)
	// GetAllUnread 批次取得使用者在所有頻道的未讀計數（與 mention 計數）
	GetAllUnread(userID uint) ([]*ChannelUnreadCount, error)
	// GetChannelUnread 取得使用者在某一頻道的未讀計數
	GetChannelUnread(userID, channelID uint) (*ChannelUnreadCount, error)
}

type channelReadStateRepository struct {
	db *gorm.DB
}

// NewChannelReadStateRepository 建立 ChannelReadState repository
func NewChannelReadStateRepository(db *gorm.DB) ChannelReadStateRepository {
	return &channelReadStateRepository{db: db}
}

// Upsert 建立或更新已讀狀態
func (r *channelReadStateRepository) Upsert(userID, channelID, lastMessageID uint) error {
	state := model.ChannelReadState{
		UserID:            userID,
		ChannelID:         channelID,
		LastReadMessageID: lastMessageID,
		UpdatedAt:         time.Now(),
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_read_message_id", "updated_at"}),
	}).Create(&state).Error
}

// GetLastRead 取得最後已讀的 message ID
func (r *channelReadStateRepository) GetLastRead(userID, channelID uint) (uint, error) {
	var state model.ChannelReadState

	err := r.db.
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		First(&state).Error
	if err != nil {
		if isNotFound(err) {
			return 0, nil
		}

		return 0, err
	}

	return state.LastReadMessageID, nil
}

// GetAllUnread 批次計算使用者所有頻道的未讀數與 mention 數
func (r *channelReadStateRepository) GetAllUnread(userID uint) ([]*ChannelUnreadCount, error) {
	// 取得使用者參與的所有頻道（guild 頻道：透過 guild_members；DM 頻道：透過 channel_participants）
	// 此查詢分兩步：1. 取候選頻道 IDs，2. 計算各頻道未讀
	type channelRow struct {
		ChannelID uint
		GuildID   *uint
	}

	var rows []channelRow

	// Guild text channels the user is a member of
	if err := r.db.Raw(`
		SELECT c.id AS channel_id, c.guild_id
		FROM channels c
		JOIN guild_members gm ON gm.guild_id = c.guild_id
		WHERE gm.user_id = ? AND c.type = 'text'
	`, userID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return []*ChannelUnreadCount{}, nil
	}

	channelIDs := make([]uint, len(rows))
	guildByChannel := make(map[uint]*uint, len(rows))

	for i, r := range rows {
		channelIDs[i] = r.ChannelID
		gid := r.GuildID
		guildByChannel[r.ChannelID] = gid
	}

	// 計算每個頻道中 last_read_message_id 之後的訊息數
	type unreadRow struct {
		ChannelID    uint
		UnreadCount  int64
		MentionCount int64
	}

	var unreadRows []unreadRow

	if err := r.db.Raw(`
		SELECT
			m.channel_id,
			COUNT(DISTINCT m.id)                        AS unread_count,
			COUNT(DISTINCT mm.id)                       AS mention_count
		FROM messages m
		LEFT JOIN channel_read_states crs
			ON crs.channel_id = m.channel_id AND crs.user_id = ?
		LEFT JOIN message_mentions mm
			ON mm.message_id = m.id AND mm.user_id = ?
		WHERE m.channel_id IN (?)
		  AND (crs.last_read_message_id IS NULL OR m.id > crs.last_read_message_id)
		  AND m.user_id != ?
		GROUP BY m.channel_id
	`, userID, userID, channelIDs, userID).Scan(&unreadRows).Error; err != nil {
		return nil, err
	}

	result := make([]*ChannelUnreadCount, 0, len(unreadRows))

	for _, u := range unreadRows {
		gid := guildByChannel[u.ChannelID]
		result = append(result, &ChannelUnreadCount{
			ChannelID:    u.ChannelID,
			GuildID:      gid,
			UnreadCount:  u.UnreadCount,
			MentionCount: u.MentionCount,
		})
	}

	return result, nil
}

// GetChannelUnread 取得使用者在某頻道的未讀計數
func (r *channelReadStateRepository) GetChannelUnread(
	userID, channelID uint,
) (*ChannelUnreadCount, error) {
	type unreadRow struct {
		UnreadCount  int64
		MentionCount int64
	}

	var row unreadRow

	if err := r.db.Raw(`
		SELECT
			COUNT(DISTINCT m.id)  AS unread_count,
			COUNT(DISTINCT mm.id) AS mention_count
		FROM messages m
		LEFT JOIN channel_read_states crs
			ON crs.channel_id = m.channel_id AND crs.user_id = ?
		LEFT JOIN message_mentions mm
			ON mm.message_id = m.id AND mm.user_id = ?
		WHERE m.channel_id = ?
		  AND (crs.last_read_message_id IS NULL OR m.id > crs.last_read_message_id)
		  AND m.user_id != ?
	`, userID, userID, channelID, userID).Scan(&row).Error; err != nil {
		return nil, err
	}

	// 取得頻道的 guild_id
	var ch model.Channel
	if err := r.db.Select("id, guild_id").First(&ch, channelID).Error; err != nil {
		return nil, err
	}

	return &ChannelUnreadCount{
		ChannelID:    channelID,
		GuildID:      ch.GuildID,
		UnreadCount:  row.UnreadCount,
		MentionCount: row.MentionCount,
	}, nil
}

// isNotFound 判斷是否為 not found 錯誤（避免直接 import gorm）
func isNotFound(err error) bool {
	return err != nil && err.Error() == "record not found"
}
