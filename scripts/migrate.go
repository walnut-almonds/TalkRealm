package main

import (
	"flag"
	"log"

	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

func main() {
	// 解析命令列參數
	drop := flag.Bool("drop", false, "Drop all tables before migration")
	flag.Parse()

	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日誌
	if err := logger.Init(cfg.Log.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 初始化資料庫
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", "error", err)
	}

	db := database.GetDB()

	// 如果指定了 drop 參數，則先刪除所有表
	if *drop {
		logger.Info("Dropping all tables...")
		// 注意：這會刪除所有資料！
		if err := db.Migrator().DropTable(
			"guild_members",
			"messages",
			"channels",
			"guilds",
			"users",
		); err != nil {
			logger.Fatal("Failed to drop tables", "error", err)
		}
		logger.Info("All tables dropped successfully")
	}

	// 執行自動遷移
	if err := database.AutoMigrate(); err != nil {
		logger.Fatal("Migration failed", "error", err)
	}

	logger.Info("Database migration completed successfully!")
}
