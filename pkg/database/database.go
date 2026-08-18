package database

import (
	"fmt"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var db *gorm.DB

// Init 初始化資料庫連線
func Init(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Taipei",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.Port,
		cfg.SSLMode,
	)

	// 設定 GORM logger
	gormLogLevel := gormlogger.Silent
	if cfg.LogMode {
		gormLogLevel = gormlogger.Info
	}

	var err error

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 取得底層的 SQL DB 進行連線池設定
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 設定連線池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	logger.Info("Database connected successfully",
		"host", cfg.Host,
		"database", cfg.DBName,
	)

	return nil
}

// GetDB 取得資料庫實例
func GetDB() *gorm.DB {
	return db
}

// Close 關閉資料庫連線
func Close() error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// AutoMigrate 自動遷移資料庫結構
func AutoMigrate() error {
	logger.Info("Running database migrations...")

	err := db.AutoMigrate(
		&model.User{},
		&model.Guild{},
		&model.Channel{},
		&model.ChannelParticipant{},
		&model.Message{},
		&model.GuildMember{},
		&model.GuildInvite{},
		&model.RefreshToken{},
		&model.UserOAuthProvider{},
		&model.File{},
		&model.MessageAttachment{},
		&model.MessageLike{},
		&model.MessageReaction{},
		&model.MessageTranslation{},
		&model.GameState{},
		&model.Friendship{},
		&model.ChannelReadState{},
		&model.MessageMention{},
		&model.Word{},
		&model.LearnStat{},
		&model.LearnWordRecord{},
		&model.LearnDailyScore{},
		&model.LearnCampaignLevel{},
		&model.LearnCampaignProgress{},
		&model.LearnWeeklyXP{},
		&model.LearnSentence{},
		&model.Follow{},
		&model.FeedPost{},
		&model.FeedComment{},
		&model.FeedPostLike{},
		&model.FeedCommentLike{},
		&model.FeedPostAttachment{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	if err := patchChannelGuildIDNullable(); err != nil {
		return err
	}

	if err := patchUserUILocaleDefault(); err != nil {
		return err
	}

	logger.Info("Database migrations completed successfully")

	return nil
}

// patchChannelGuildIDNullable ensures channels.guild_id allows NULL for DM channels.
// GORM AutoMigrate may not relax an existing NOT NULL constraint in place.
func patchChannelGuildIDNullable() error {
	err := db.Exec(`ALTER TABLE channels ALTER COLUMN guild_id DROP NOT NULL`).Error
	if err != nil {
		return fmt.Errorf("failed to patch channels.guild_id nullable: %w", err)
	}

	return nil
}

// patchUserUILocaleDefault removes server-side default for users.ui_locale.
// Empty value means "user has not set preferred UI locale yet".
func patchUserUILocaleDefault() error {
	err := db.Exec(`ALTER TABLE users ALTER COLUMN ui_locale DROP DEFAULT`).Error
	if err != nil {
		return fmt.Errorf("failed to patch users.ui_locale default: %w", err)
	}

	return nil
}

// HealthCheck 檢查資料庫連線狀態
func HealthCheck() error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}
