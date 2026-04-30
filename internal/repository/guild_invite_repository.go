package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// GuildInviteRepository 社群邀請碼資料庫操作介面
type GuildInviteRepository interface {
	Create(invite *model.GuildInvite) error
	GetByCode(code string) (*model.GuildInvite, error)
	ListByGuildID(guildID uint) ([]*model.GuildInvite, error)
	IncrementUses(id uint) error
	Delete(id uint) error
}

type guildInviteRepository struct {
	db *gorm.DB
}

// NewGuildInviteRepository 建立社群邀請碼 repository
func NewGuildInviteRepository(db *gorm.DB) GuildInviteRepository {
	return &guildInviteRepository{db: db}
}

// Create 建立邀請碼
func (r *guildInviteRepository) Create(invite *model.GuildInvite) error {
	return r.db.Create(invite).Error
}

// GetByCode 透過邀請碼取得邀請資訊
func (r *guildInviteRepository) GetByCode(code string) (*model.GuildInvite, error) {
	var invite model.GuildInvite

	err := r.db.
		Preload("Guild").
		Preload("Creator").
		Where("code = ?", code).
		First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invite not found")
		}

		return nil, err
	}

	return &invite, nil
}

// ListByGuildID 取得社群的所有邀請碼
func (r *guildInviteRepository) ListByGuildID(guildID uint) ([]*model.GuildInvite, error) {
	var invites []*model.GuildInvite

	err := r.db.
		Preload("Creator").
		Where("guild_id = ?", guildID).
		Order("created_at DESC").
		Find(&invites).Error

	return invites, err
}

// IncrementUses 增加邀請碼使用次數
func (r *guildInviteRepository) IncrementUses(id uint) error {
	return r.db.Model(&model.GuildInvite{}).
		Where("id = ?", id).
		UpdateColumn("uses", gorm.Expr("uses + 1")).Error
}

// Delete 刪除邀請碼
func (r *guildInviteRepository) Delete(id uint) error {
	return r.db.Delete(&model.GuildInvite{}, id).Error
}
