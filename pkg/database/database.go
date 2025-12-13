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
		&model.Message{},
		&model.GuildMember{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Info("Database migrations completed successfully")

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
