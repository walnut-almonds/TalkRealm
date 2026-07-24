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
	CampaignLevelNos() ([]int, error)
	CreateCampaignLevel(l *model.LearnCampaignLevel) error
	CampaignLevelByNo(no int) (*model.LearnCampaignLevel, error)
	CampaignProgress(userID uint) ([]*model.LearnCampaignProgress, error)
	CreateCampaignProgress(p *model.LearnCampaignProgress) (bool, error)
	CampaignTotals(userIDs []uint, limit int) ([]*CampaignTotal, error)
	CampaignRank(userID uint, userIDs []uint) (*CampaignTotal, int, error)
	UpsertWeeklyXP(userID uint, week string, xp int) error
	TopWeeklyXP(week string, userIDs []uint, limit int) ([]*model.LearnWeeklyXP, error)
	WeeklyRank(userID uint, week string, userIDs []uint) (*model.LearnWeeklyXP, int, error)
	CountDueReviews(userID uint, now time.Time) (int64, error)
	CountNewSentenceWords(userID uint) (int64, error)
	DueReviewWordIDs(userID uint, now time.Time, limit int) ([]uint, error)
	NewSentenceWordIDs(userID uint, limit int) ([]uint, error)
	RandomSentenceByWord(wordID uint) (*model.LearnSentence, error)
	GetWordRecord(userID, wordID uint) (*model.LearnWordRecord, error)
	SaveSRSResult(userID, wordID uint, stage int, nextReviewAt time.Time, correct bool) error
}

// CampaignTotal 固定關卡榜單列聚合（總分 + 最遠關卡）
type CampaignTotal struct {
	UserID   uint `json:"user_id"`
	Total    int  `json:"total"`
	Furthest int  `json:"furthest"`
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
			// ON CONFLICT DO UPDATE 內未加表名的欄位引用有歧義（target vs excluded），必須帶表名
			col:            gorm.Expr("learn_word_records." + col + " + 1"),
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

// dueReviewScope 到期複習的共用條件：已進輪替（srs_stage>=1）、到期、且該字仍有例句可出題
func dueReviewScope(db *gorm.DB, userID uint, now time.Time) *gorm.DB {
	return db.Model(&model.LearnWordRecord{}).
		Where("user_id = ? AND srs_stage >= 1 AND next_review_at <= ?", userID, now).
		Where("EXISTS (SELECT 1 FROM learn_sentences s WHERE s.word_id = learn_word_records.word_id)")
}

// CountDueReviews 到期待複習字數
func (r *learnRepository) CountDueReviews(userID uint, now time.Time) (int64, error) {
	var n int64

	err := dueReviewScope(r.db, userID, now).Count(&n).Error

	return n, err
}

// CountNewSentenceWords 尚未進入 SRS 輪替、且有例句可出題的新字數
func (r *learnRepository) CountNewSentenceWords(userID uint) (int64, error) {
	var n int64

	err := r.db.Model(&model.LearnSentence{}).
		Joins("LEFT JOIN learn_word_records r ON r.word_id = learn_sentences.word_id AND r.user_id = ?", userID).
		Where("r.id IS NULL OR r.srs_stage = 0").
		Distinct("learn_sentences.word_id").
		Count(&n).Error

	return n, err
}

// DueReviewWordIDs 到期字 ID（到期早者先）
func (r *learnRepository) DueReviewWordIDs(userID uint, now time.Time, limit int) ([]uint, error) {
	var ids []uint

	err := dueReviewScope(r.db, userID, now).
		Order("next_review_at ASC").Limit(limit).
		Pluck("word_id", &ids).Error

	return ids, err
}

// NewSentenceWordIDs 新字 ID（有例句、未進輪替；隨機取樣避免每次同一批）。
// Postgres 不允許 SELECT DISTINCT 搭配不在 select list 的 ORDER BY random()，
// 故先取 distinct word_id 子查詢，外層再隨機排序。
func (r *learnRepository) NewSentenceWordIDs(userID uint, limit int) ([]uint, error) {
	var ids []uint

	sub := r.db.Model(&model.LearnSentence{}).
		Joins("LEFT JOIN learn_word_records r ON r.word_id = learn_sentences.word_id AND r.user_id = ?", userID).
		Where("r.id IS NULL OR r.srs_stage = 0").
		Distinct("learn_sentences.word_id")

	err := r.db.Table("(?) AS t", sub).
		Order("random()").Limit(limit).
		Pluck("word_id", &ids).Error

	return ids, err
}

// RandomSentenceByWord 取該字的一則例句（隨機）；無例句回傳 (nil, nil)
func (r *learnRepository) RandomSentenceByWord(wordID uint) (*model.LearnSentence, error) {
	var s model.LearnSentence

	err := r.db.Where("word_id = ?", wordID).Order("random()").First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil //nolint:nilnil // nil = 無例句，由 service 決定跳過
	}

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetWordRecord 取 user×word 記錄（含 SRS 狀態）；無記錄回傳 (nil, nil)
func (r *learnRepository) GetWordRecord(userID, wordID uint) (*model.LearnWordRecord, error) {
	var rec model.LearnWordRecord

	err := r.db.Where("user_id = ? AND word_id = ?", userID, wordID).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil //nolint:nilnil // nil = 尚無記錄（全新字）
	}

	if err != nil {
		return nil, err
	}

	return &rec, nil
}

// SaveSRSResult upsert SRS 排程結果：累加 correct/wrong 並寫入新的 stage/next_review_at
func (r *learnRepository) SaveSRSResult(
	userID, wordID uint, stage int, nextReviewAt time.Time, correct bool,
) error {
	now := time.Now().UTC()
	col := "wrong_count"

	if correct {
		col = "correct_count"
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "word_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			// ON CONFLICT DO UPDATE 內欄位引用需帶表名，避免 42702 歧義（target vs excluded）
			col:              gorm.Expr("learn_word_records." + col + " + 1"),
			"srs_stage":      stage,
			"next_review_at": nextReviewAt,
			"last_seen_at":   now,
		}),
	}).Create(&model.LearnWordRecord{
		UserID: userID, WordID: wordID,
		CorrectCount: boolToInt(correct), WrongCount: boolToInt(!correct),
		SRSStage: stage, NextReviewAt: &nextReviewAt, LastSeenAt: now,
	}).Error
}

// CampaignLevelNos 取已存在的固定關卡編號（開機冪等生成用）
func (r *learnRepository) CampaignLevelNos() ([]int, error) {
	var nos []int

	err := r.db.Model(&model.LearnCampaignLevel{}).
		Order("level_no").Pluck("level_no", &nos).Error
	if err != nil {
		return nil, err
	}

	return nos, nil
}

// CreateCampaignLevel 寫入固定關卡
func (r *learnRepository) CreateCampaignLevel(l *model.LearnCampaignLevel) error {
	return r.db.Create(l).Error
}

// CampaignLevelByNo 取指定編號的固定關卡；不存在回傳 (nil, nil)
func (r *learnRepository) CampaignLevelByNo(no int) (*model.LearnCampaignLevel, error) {
	var l model.LearnCampaignLevel

	err := r.db.Where("level_no = ?", no).First(&l).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil //nolint:nilnil // nil = 關卡不存在，由 service 轉語意錯誤
	}

	if err != nil {
		return nil, err
	}

	return &l, nil
}

// CampaignProgress 取使用者全部首通紀錄（關卡數量級 ≤ 百，直接全取）
func (r *learnRepository) CampaignProgress(userID uint) ([]*model.LearnCampaignProgress, error) {
	var ps []*model.LearnCampaignProgress

	err := r.db.Where("user_id = ?", userID).Order("level_no").Find(&ps).Error
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// CreateCampaignProgress 寫入首通紀錄；該關已有記錄回傳 false（不覆寫，重玩不刷榜）
func (r *learnRepository) CreateCampaignProgress(p *model.LearnCampaignProgress) (bool, error) {
	res := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(p)
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected > 0, nil
}

// CampaignTotals 固定關卡榜（總分高者先，同分最遠關卡先）；userIDs 非空時限定範圍（好友榜）
func (r *learnRepository) CampaignTotals(userIDs []uint, limit int) ([]*CampaignTotal, error) {
	q := r.db.Model(&model.LearnCampaignProgress{}).
		Select("user_id, SUM(score) AS total, MAX(level_no) AS furthest").
		Group("user_id").
		Order("total DESC, furthest DESC").
		Limit(limit)

	if len(userIDs) > 0 {
		q = q.Where("user_id IN ?", userIDs)
	}

	var out []*CampaignTotal
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}

	return out, nil
}

// CampaignRank 取使用者關卡榜總分與名次；無任何首通回傳 (nil, 0, nil)。
// userIDs 非空時名次僅在該範圍內計算（好友榜）。
func (r *learnRepository) CampaignRank(
	userID uint, userIDs []uint,
) (*CampaignTotal, int, error) {
	var me CampaignTotal

	err := r.db.Model(&model.LearnCampaignProgress{}).
		Select("user_id, SUM(score) AS total, MAX(level_no) AS furthest").
		Where("user_id = ?", userID).
		Group("user_id").
		Scan(&me).Error
	if err != nil {
		return nil, 0, err
	}

	if me.UserID == 0 {
		return nil, 0, nil
	}

	sub := r.db.Model(&model.LearnCampaignProgress{}).
		Select("user_id, SUM(score) AS total, MAX(level_no) AS furthest").
		Group("user_id")

	if len(userIDs) > 0 {
		sub = sub.Where("user_id IN ?", userIDs)
	}

	var better int64

	err = r.db.Table("(?) AS t", sub).
		Where("t.total > ? OR (t.total = ? AND t.furthest > ?)", me.Total, me.Total, me.Furthest).
		Count(&better).Error
	if err != nil {
		return nil, 0, err
	}

	return &me, int(better) + 1, nil
}

// UpsertWeeklyXP 累加使用者當週 XP
func (r *learnRepository) UpsertWeeklyXP(userID uint, week string, xp int) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "week"}},
		DoUpdates: clause.Assignments(map[string]any{
			// ON CONFLICT DO UPDATE 內未加表名的欄位引用有歧義（target vs excluded），必須帶表名
			"xp":         gorm.Expr("learn_weekly_xps.xp + ?", xp),
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&model.LearnWeeklyXP{
		UserID: userID, Week: week, XP: xp, UpdatedAt: time.Now().UTC(),
	}).Error
}

// TopWeeklyXP 週榜（XP 高者先，同分先達成者先）；userIDs 非空時限定範圍（好友榜）
func (r *learnRepository) TopWeeklyXP(
	week string, userIDs []uint, limit int,
) ([]*model.LearnWeeklyXP, error) {
	q := r.db.Where("week = ?", week).
		Order("xp DESC, updated_at ASC").Limit(limit)

	if len(userIDs) > 0 {
		q = q.Where("user_id IN ?", userIDs)
	}

	var rows []*model.LearnWeeklyXP
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

// WeeklyRank 取使用者當週 XP 與名次；本週未得分回傳 (nil, 0, nil)。
// userIDs 非空時名次僅在該範圍內計算（好友榜）。
func (r *learnRepository) WeeklyRank(
	userID uint, week string, userIDs []uint,
) (*model.LearnWeeklyXP, int, error) {
	var me model.LearnWeeklyXP

	err := r.db.Where("user_id = ? AND week = ?", userID, week).First(&me).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, nil
	}

	if err != nil {
		return nil, 0, err
	}

	q := r.db.Model(&model.LearnWeeklyXP{}).
		Where("week = ? AND (xp > ? OR (xp = ? AND updated_at < ?))",
			week, me.XP, me.XP, me.UpdatedAt)

	if len(userIDs) > 0 {
		q = q.Where("user_id IN ?", userIDs)
	}

	var better int64
	if err := q.Count(&better).Error; err != nil {
		return nil, 0, err
	}

	return &me, int(better) + 1, nil
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
