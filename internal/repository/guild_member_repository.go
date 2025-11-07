package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// GuildMemberRepository 社群成員資料庫操作介面
type GuildMemberRepository interface {
	Create(member *model.GuildMember) error
	GetByID(id uint) (*model.GuildMember, error)
	Update(member *model.GuildMember) error
	Delete(id uint) error
	GetByGuildID(guildID uint) ([]*model.GuildMember, error)
	GetByUserID(userID uint) ([]*model.GuildMember, error)
	GetMember(guildID, userID uint) (*model.GuildMember, error)
	IsMember(guildID, userID uint) (bool, error)
}

type guildMemberRepository struct {
	db *gorm.DB
}

// NewGuildMemberRepository 建立社群成員 repository
func NewGuildMemberRepository(db *gorm.DB) GuildMemberRepository {
	return &guildMemberRepository{db: db}
}

// Create 建立新成員
func (r *guildMemberRepository) Create(member *model.GuildMember) error {
	return r.db.Create(member).Error
}

// GetByID 透過 ID 取得成員
func (r *guildMemberRepository) GetByID(id uint) (*model.GuildMember, error) {
	var member model.GuildMember
	err := r.db.Preload("User").Preload("Guild").First(&member, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("guild member not found")
		}
		return nil, err
	}
	return &member, nil
}

// Update 更新成員資訊
func (r *guildMemberRepository) Update(member *model.GuildMember) error {
	return r.db.Save(member).Error
}

// Delete 刪除成員
func (r *guildMemberRepository) Delete(id uint) error {
	return r.db.Delete(&model.GuildMember{}, id).Error
}

// GetByGuildID 取得社群的所有成員
func (r *guildMemberRepository) GetByGuildID(guildID uint) ([]*model.GuildMember, error) {
	var members []*model.GuildMember
	err := r.db.Preload("User").Where("guild_id = ?", guildID).Find(&members).Error
	return members, err
}

// GetByUserID 取得使用者加入的所有社群成員資料
func (r *guildMemberRepository) GetByUserID(userID uint) ([]*model.GuildMember, error) {
	var members []*model.GuildMember
	err := r.db.Preload("Guild").Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

// GetMember 取得特定社群的特定成員
func (r *guildMemberRepository) GetMember(guildID, userID uint) (*model.GuildMember, error) {
	var member model.GuildMember
	err := r.db.
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("guild member not found")
		}
		return nil, err
	}
	return &member, nil
}

// IsMember 檢查使用者是否為社群成員
func (r *guildMemberRepository) IsMember(guildID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Count(&count).Error
	return count > 0, err
}
