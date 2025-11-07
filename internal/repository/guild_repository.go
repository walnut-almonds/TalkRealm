package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// GuildRepository 社群資料庫操作介面
type GuildRepository interface {
	Create(guild *model.Guild) error
	GetByID(id uint) (*model.Guild, error)
	Update(guild *model.Guild) error
	Delete(id uint) error
	List(offset, limit int) ([]*model.Guild, error)
	GetByOwnerID(ownerID uint) ([]*model.Guild, error)
	GetMemberGuilds(userID uint, offset, limit int) ([]*model.Guild, error)
}

type guildRepository struct {
	db *gorm.DB
}

// NewGuildRepository 建立社群 repository
func NewGuildRepository(db *gorm.DB) GuildRepository {
	return &guildRepository{db: db}
}

// Create 建立新社群
func (r *guildRepository) Create(guild *model.Guild) error {
	return r.db.Create(guild).Error
}

// GetByID 透過 ID 取得社群
func (r *guildRepository) GetByID(id uint) (*model.Guild, error) {
	var guild model.Guild
	err := r.db.Preload("Owner").First(&guild, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("guild not found")
		}
		return nil, err
	}
	return &guild, nil
}

// Update 更新社群資訊
func (r *guildRepository) Update(guild *model.Guild) error {
	return r.db.Save(guild).Error
}

// Delete 刪除社群
func (r *guildRepository) Delete(id uint) error {
	return r.db.Delete(&model.Guild{}, id).Error
}

// List 列出所有社群（分頁）
func (r *guildRepository) List(offset, limit int) ([]*model.Guild, error) {
	var guilds []*model.Guild
	err := r.db.Preload("Owner").Offset(offset).Limit(limit).Find(&guilds).Error
	return guilds, err
}

// GetByOwnerID 取得使用者擁有的所有社群
func (r *guildRepository) GetByOwnerID(ownerID uint) ([]*model.Guild, error) {
	var guilds []*model.Guild
	err := r.db.Where("owner_id = ?", ownerID).Find(&guilds).Error
	return guilds, err
}

// GetMemberGuilds 取得使用者加入的所有社群
func (r *guildRepository) GetMemberGuilds(userID uint, offset, limit int) ([]*model.Guild, error) {
	var guilds []*model.Guild
	err := r.db.
		Joins("JOIN guild_members ON guild_members.guild_id = guilds.id").
		Where("guild_members.user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Find(&guilds).Error
	return guilds, err
}
