package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// TranslationRepository 訊息翻譯資料庫操作介面
type TranslationRepository interface {
	Upsert(t *model.MessageTranslation) error
	GetByMessageID(messageID uint) (*model.MessageTranslation, error)
}

type translationRepository struct {
	db *gorm.DB
}

// NewTranslationRepository 建立翻譯 repository
func NewTranslationRepository(db *gorm.DB) TranslationRepository {
	return &translationRepository{db: db}
}

// Upsert 建立或更新翻譯記錄
func (r *translationRepository) Upsert(t *model.MessageTranslation) error {
	return r.db.Save(t).Error
}

// GetByMessageID 透過訊息 ID 取得翻譯
func (r *translationRepository) GetByMessageID(messageID uint) (*model.MessageTranslation, error) {
	var t model.MessageTranslation

	err := r.db.Where("message_id = ?", messageID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	return &t, nil
}
