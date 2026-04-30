package repository

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// RefreshTokenRepository Refresh Token 資料庫操作介面
type RefreshTokenRepository interface {
	Create(token *model.RefreshToken) error
	GetByToken(token string) (*model.RefreshToken, error)
	RevokeByToken(token string) error
	RevokeAllByUserID(userID uint) error
	DeleteExpired() error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository 建立 refresh token repository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// Create 建立 refresh token
func (r *refreshTokenRepository) Create(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

// GetByToken 透過 token 字串取得 refresh token
func (r *refreshTokenRepository) GetByToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken

	err := r.db.Where("token = ?", token).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found")
		}

		return nil, err
	}

	return &rt, nil
}

// RevokeByToken 撤銷指定 token
func (r *refreshTokenRepository) RevokeByToken(token string) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}

// RevokeAllByUserID 撤銷使用者的所有 refresh token（登出所有裝置）
func (r *refreshTokenRepository) RevokeAllByUserID(userID uint) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error
}

// DeleteExpired 刪除已過期的 token
func (r *refreshTokenRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.RefreshToken{}).Error
}
