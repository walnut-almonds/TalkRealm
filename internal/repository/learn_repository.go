package repository

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LearnRepository 單字學習資料庫操作介面
type LearnRepository interface {
	RandomWordsByTier(tier, n int) ([]*model.Word, error)
	WordsByIDs(ids []uint) ([]*model.Word, error)
	AllWordsForIndex() ([]*model.Word, error)
	GetOrCreateStats(userID uint) (*model.LearnStat, error)
	SaveStats(s *model.LearnStat) error
	UpsertWordRecord(userID, wordID uint, correct bool) error
	CreateDailyScore(s *model.LearnDailyScore) (bool, error)
	TopDailyScores(date string, limit int) ([]*model.LearnDailyScore, error)
	UserDailyRank(userID uint, date string) (*model.LearnDailyScore, int, error)
}

type learnRepository struct {
	db *gorm.DB
}

// NewLearnRepository 建立 learn repository
func NewLearnRepository(db *gorm.DB) LearnRepository {
	return &learnRepository{db: db}
}

// RandomWordsByTier 隨機抽 n 個指定難度的字
// ponytail: ORDER BY random() 全表掃，5 萬字以內無感；字表破百萬再換 TABLESAMPLE
func (r *learnRepository) RandomWordsByTier(tier, n int) ([]*model.Word, error) {
	var words []*model.Word

	err := r.db.Where("tier = ?", tier).
		Order("random()").Limit(n).Find(&words).Error
	if err != nil {
		return nil, err
	}

	return words, nil
}

// WordsByIDs 依 ID 取字
func (r *learnRepository) WordsByIDs(ids []uint) ([]*model.Word, error) {
	var words []*model.Word

	if err := r.db.Where("id IN ?", ids).Find(&words).Error; err != nil {
		return nil, err
	}

	return words, nil
}

// AllWordsForIndex 取全字表輕量欄位（anagram 索引用）
func (r *learnRepository) AllWordsForIndex() ([]*model.Word, error) {
	var words []*model.Word

	err := r.db.Select("id", "word", "tier", "frequency", "length").
		Find(&words).Error
	if err != nil {
		return nil, err
	}

	return words, nil
}

// GetOrCreateStats 取得（無則建立）使用者統計
func (r *learnRepository) GetOrCreateStats(userID uint) (*model.LearnStat, error) {
	var s model.LearnStat

	err := r.db.Where("user_id = ?", userID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s = model.LearnStat{UserID: userID}

		if err := r.db.Create(&s).Error; err != nil {
			return nil, err
		}

		return &s, nil
	}

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// SaveStats 儲存統計
func (r *learnRepository) SaveStats(s *model.LearnStat) error {
	return r.db.Save(s).Error
}

// UpsertWordRecord 累加 user×word 的對/錯次數
func (r *learnRepository) UpsertWordRecord(userID, wordID uint, correct bool) error {
	col := "wrong_count"
	if correct {
		col = "correct_count"
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "word_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			col:            gorm.Expr(col + " + 1"),
			"last_seen_at": time.Now().UTC(),
		}),
	}).Create(&model.LearnWordRecord{
		UserID: userID, WordID: wordID,
		CorrectCount: boolToInt(correct), WrongCount: boolToInt(!correct),
		LastSeenAt: time.Now().UTC(),
	}).Error
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// CreateDailyScore 寫入每日分數；當日已有記錄回傳 false（不覆寫）
func (r *learnRepository) CreateDailyScore(s *model.LearnDailyScore) (bool, error) {
	res := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(s)
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected > 0, nil
}

// TopDailyScores 取當日排行榜（分數高者先，同分先完成者先）
func (r *learnRepository) TopDailyScores(date string, limit int) ([]*model.LearnDailyScore, error) {
	var scores []*model.LearnDailyScore

	err := r.db.Where("date = ?", date).
		Order("score DESC, completed_in_ms ASC").Limit(limit).Find(&scores).Error
	if err != nil {
		return nil, err
	}

	return scores, nil
}

// UserDailyRank 取使用者當日分數與名次；未完成回傳 (nil, 0, nil)
func (r *learnRepository) UserDailyRank(
	userID uint,
	date string,
) (*model.LearnDailyScore, int, error) {
	var s model.LearnDailyScore

	err := r.db.Where("user_id = ? AND date = ?", userID, date).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, nil
	}

	if err != nil {
		return nil, 0, err
	}

	var better int64

	err = r.db.Model(&model.LearnDailyScore{}).
		Where("date = ? AND (score > ? OR (score = ? AND completed_in_ms < ?))",
			date, s.Score, s.Score, s.CompletedInMs).
		Count(&better).Error
	if err != nil {
		return nil, 0, err
	}

	return &s, int(better) + 1, nil
}
