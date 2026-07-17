package model

import "time"

// 模組邊界：learn 相關表只存 user_id（plain uint），不建 User 關聯、不 Preload，
// 未來可整組拆為獨立 service（見 spec 2026-07-08-learn-vocab-game-design.md §2）。

// Word 字表（ECDICT 子集，由 scripts/seedwords 匯入）
type Word struct {
	ID             uint   `gorm:"primarykey"           json:"id"`
	Word           string `gorm:"uniqueIndex;not null" json:"word"`
	Phonetic       string `                            json:"phonetic"`
	Tier           int    `gorm:"not null;index"       json:"tier"` // 1..5
	Frequency      int    `                            json:"frequency"`
	Length         int    `gorm:"not null;index"       json:"length"`
	DefinitionEN   string `gorm:"type:text"            json:"definition_en"`
	DefinitionZH   string `gorm:"type:text"            json:"definition_zh"`
	DefinitionZHTW string `gorm:"type:text"            json:"definition_zh_tw"`
	DefinitionJA   string `gorm:"type:text"            json:"definition_ja"`
}

// LearnStat 每位使用者一列的學習統計
type LearnStat struct {
	UserID         uint      `gorm:"primarykey"       json:"user_id"`
	XP             int       `gorm:"default:0"        json:"xp"`
	Streak         int       `gorm:"default:0"        json:"streak"`
	LastPlayedDate string    `gorm:"type:varchar(10)" json:"last_played_date"` // "2006-01-02"（UTC）
	UpdatedAt      time.Time `                        json:"updated_at"`
}

// LearnWordRecord user×word 的作答記錄（未來 SRS 地基）
type LearnWordRecord struct {
	ID           uint      `gorm:"primarykey"                             json:"id"`
	UserID       uint      `gorm:"not null;uniqueIndex:idx_lwr_user_word" json:"user_id"`
	WordID       uint      `gorm:"not null;uniqueIndex:idx_lwr_user_word" json:"word_id"`
	CorrectCount int       `gorm:"default:0"                              json:"correct_count"`
	WrongCount   int       `gorm:"default:0"                              json:"wrong_count"`
	LastSeenAt   time.Time `                                              json:"last_seen_at"`
}

// LearnDailyScore 每日挑戰分數（(user_id, date) unique，只記首次完成）
type LearnDailyScore struct {
	ID            uint      `gorm:"primarykey"                                                    json:"id"`
	UserID        uint      `gorm:"not null;uniqueIndex:idx_lds_user_date"                        json:"user_id"`
	Date          string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_lds_user_date;index" json:"date"`
	Score         int       `gorm:"not null"                                                      json:"score"`
	CompletedInMs int64     `gorm:"not null"                                                      json:"completed_in_ms"`
	CreatedAt     time.Time `                                                                     json:"created_at"`
}

// LearnCampaignLevel 固定關卡（開機冪等生成，發布後不可變——
// 重生成會回溯改題，破壞既有進度與排行榜的可比性，只允許往後追加新關）
type LearnCampaignLevel struct {
	ID        uint      `gorm:"primarykey"           json:"id"`
	LevelNo   int       `gorm:"not null;uniqueIndex" json:"level_no"`
	Tier      int       `gorm:"not null"             json:"tier"`
	Puzzle    string    `gorm:"type:text;not null"   json:"puzzle"` // campaignPuzzle JSON（含答案，不下發 client）
	CreatedAt time.Time `                            json:"created_at"`
}

// LearnCampaignProgress 固定關卡首通紀錄（(user_id, level_no) unique，只記首次；重玩不刷榜）
type LearnCampaignProgress struct {
	ID        uint      `gorm:"primarykey"                              json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_lcp_user_level" json:"user_id"`
	LevelNo   int       `gorm:"not null;uniqueIndex:idx_lcp_user_level" json:"level_no"`
	Score     int       `gorm:"not null"                                json:"score"`
	CreatedAt time.Time `                                               json:"created_at"`
}

// LearnWeeklyXP 週 XP 累計（week 為 ISO 週如 "2026-W29"；每次完關累加，週榜用）
type LearnWeeklyXP struct {
	ID        uint      `gorm:"primarykey"                                                   json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_lwx_user_week"                       json:"user_id"`
	Week      string    `gorm:"type:varchar(8);not null;uniqueIndex:idx_lwx_user_week;index" json:"week"`
	XP        int       `gorm:"not null"                                                     json:"xp"`
	UpdatedAt time.Time `                                                                    json:"updated_at"`
}
