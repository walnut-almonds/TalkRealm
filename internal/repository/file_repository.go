package repository

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// FileRepository 檔案資料庫操作介面
type FileRepository interface {
	Create(file *model.File) error
	GetByID(id uint) (*model.File, error)
	GetByStorageKey(key string) (*model.File, error)
	Update(file *model.File) error
	Delete(id uint) error

	// CountByUserToday 統計使用者當日上傳次數
	CountByUserToday(userID uint) (int64, error)
	// SumBytesByUserToday 統計使用者當日上傳總 bytes
	SumBytesByUserToday(userID uint) (int64, error)

	// FindExpired 找到已過期的 active 檔案（用於 TTL 清理）
	FindExpired(limit int) ([]*model.File, error)
	// FindLRUByUser 找到指定 user 最久未使用的 active 檔案（LRU 清理）
	FindLRUByUser(userID uint, limit int) ([]*model.File, error)

	// TouchLastAccessed 更新最後存取時間
	TouchLastAccessed(id uint) error

	// CreateAttachment 建立訊息附件關聯
	CreateAttachment(attachment *model.MessageAttachment) error
	// GetAttachmentsByMessageID 取得訊息的所有附件
	GetAttachmentsByMessageID(messageID uint) ([]*model.MessageAttachment, error)
}

type fileRepository struct {
	db *gorm.DB
}

// NewFileRepository 建立 File repository
func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *fileRepository) GetByID(id uint) (*model.File, error) {
	var file model.File

	err := r.db.Preload("User").First(&file, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	return &file, nil
}

func (r *fileRepository) GetByStorageKey(key string) (*model.File, error) {
	var file model.File

	err := r.db.Where("storage_key = ?", key).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	return &file, nil
}

func (r *fileRepository) Update(file *model.File) error {
	return r.db.Save(file).Error
}

func (r *fileRepository) Delete(id uint) error {
	return r.db.Delete(&model.File{}, id).Error
}

func (r *fileRepository) CountByUserToday(userID uint) (int64, error) {
	var count int64

	today := time.Now().UTC().Truncate(24 * time.Hour)

	err := r.db.Model(&model.File{}).
		Where("user_id = ? AND status = 'active' AND created_at >= ?", userID, today).
		Count(&count).Error

	return count, err
}

func (r *fileRepository) SumBytesByUserToday(userID uint) (int64, error) {
	var total int64

	today := time.Now().UTC().Truncate(24 * time.Hour)

	err := r.db.Model(&model.File{}).
		Where("user_id = ? AND status = 'active' AND created_at >= ?", userID, today).
		Select("COALESCE(SUM(size), 0)").
		Scan(&total).Error

	return total, err
}

func (r *fileRepository) FindExpired(limit int) ([]*model.File, error) {
	var files []*model.File

	err := r.db.Where("status = 'active' AND expires_at IS NOT NULL AND expires_at < ?", time.Now().UTC()).
		Limit(limit).
		Find(&files).
		Error

	return files, err
}

func (r *fileRepository) FindLRUByUser(userID uint, limit int) ([]*model.File, error) {
	var files []*model.File

	err := r.db.Where("user_id = ? AND status = 'active'", userID).
		Order("last_accessed_at ASC").
		Limit(limit).
		Find(&files).Error

	return files, err
}

func (r *fileRepository) TouchLastAccessed(id uint) error {
	return r.db.Model(&model.File{}).
		Where("id = ?", id).
		Update("last_accessed_at", time.Now().UTC()).Error
}

func (r *fileRepository) CreateAttachment(attachment *model.MessageAttachment) error {
	return r.db.Create(attachment).Error
}

func (r *fileRepository) GetAttachmentsByMessageID(
	messageID uint,
) ([]*model.MessageAttachment, error) {
	var attachments []*model.MessageAttachment

	err := r.db.Preload("File").
		Where("message_id = ?", messageID).
		Find(&attachments).Error

	return attachments, err
}
