package repository

import (
	"errors"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
)

// GameStateRepository 猜測遊戲狀態資料庫操作介面
type GameStateRepository interface {
	Create(gs *model.GameState) error
	GetByMessageAndGuesser(messageID, guesserID uint, hiddenLang string) (*model.GameState, error)
	ListByMessage(messageID uint) ([]*model.GameState, error)
}

type gameStateRepository struct {
	db *gorm.DB
}

// NewGameStateRepository 建立 game state repository
func NewGameStateRepository(db *gorm.DB) GameStateRepository {
	return &gameStateRepository{db: db}
}

// Create 建立猜測記錄
func (r *gameStateRepository) Create(gs *model.GameState) error {
	return r.db.Create(gs).Error
}

// GetByMessageAndGuesser 取得特定使用者對特定訊息的猜測狀態
func (r *gameStateRepository) GetByMessageAndGuesser(
	messageID, guesserID uint,
	hiddenLang string,
) (*model.GameState, error) {
	var gs model.GameState

	err := r.db.Where("message_id = ? AND guesser_id = ? AND hidden_lang = ?",
		messageID, guesserID, hiddenLang).First(&gs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}

		return nil, err
	}

	return &gs, nil
}

// ListByMessage 取得某訊息的所有猜測記錄
func (r *gameStateRepository) ListByMessage(messageID uint) ([]*model.GameState, error) {
	var states []*model.GameState

	if err := r.db.Where("message_id = ?", messageID).Find(&states).Error; err != nil {
		return nil, err
	}

	return states, nil
}
