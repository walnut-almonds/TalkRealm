package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// OAuthProviderRepository 管理 user_oauth_providers 資料表
type OAuthProviderRepository interface {
	// FindByProvider 以廠商名稱 + 廠商 ID 查詢連結（若不存在回傳 nil, nil）
	FindByProvider(provider, providerID string) (*model.UserOAuthProvider, error)
	// Create 建立新的 OAuth 連結
	Create(link *model.UserOAuthProvider) error
	// Update 更新連結（例如 ProviderID 異動時）
	Update(link *model.UserOAuthProvider) error
	// ListByUserID 列出某使用者所有已連結的 OAuth 廠商
	ListByUserID(userID uint) ([]*model.UserOAuthProvider, error)
	// DeleteByUserIDAndProvider 解除特定廠商的連結
	DeleteByUserIDAndProvider(userID uint, provider string) error
}

type oauthProviderRepository struct {
	db *gorm.DB
}

// NewOAuthProviderRepository 建立 OAuthProviderRepository
func NewOAuthProviderRepository(db *gorm.DB) OAuthProviderRepository {
	return &oauthProviderRepository{db: db}
}

func (r *oauthProviderRepository) FindByProvider(
	provider, providerID string,
) (*model.UserOAuthProvider, error) {
	var link model.UserOAuthProvider

	err := r.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	return &link, nil
}

func (r *oauthProviderRepository) Create(link *model.UserOAuthProvider) error {
	return r.db.Create(link).Error
}

func (r *oauthProviderRepository) Update(link *model.UserOAuthProvider) error {
	return r.db.Save(link).Error
}

func (r *oauthProviderRepository) ListByUserID(userID uint) ([]*model.UserOAuthProvider, error) {
	var links []*model.UserOAuthProvider

	err := r.db.Where("user_id = ?", userID).Find(&links).Error

	return links, err
}

func (r *oauthProviderRepository) DeleteByUserIDAndProvider(userID uint, provider string) error {
	return r.db.Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&model.UserOAuthProvider{}).
		Error
}
