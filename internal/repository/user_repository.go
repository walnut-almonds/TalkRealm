package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// UserRepository 使用者資料庫操作介面
type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uint) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
	List(offset, limit int) ([]*model.User, error)
	UpdateStatus(id uint, status string) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 建立使用者 repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 建立新使用者
func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// GetByID 透過 ID 取得使用者
func (r *userRepository) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 透過 Email 取得使用者
func (r *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 透過 Username 取得使用者
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新使用者資訊
func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// Delete 刪除使用者
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

// List 列出使用者（分頁）
func (r *userRepository) List(offset, limit int) ([]*model.User, error) {
	var users []*model.User
	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// UpdateStatus 更新使用者狀態
func (r *userRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}
