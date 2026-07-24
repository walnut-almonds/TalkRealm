package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/walnut-almonds/talkrealm/buildinfo"
	"github.com/walnut-almonds/talkrealm/internal/server"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

func main() {
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

	logger.Info("config", "config", cfg)

	logger.Info("Starting TalkRealm", "version", buildinfo.Version)

	// 初始化資料庫
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("Failed to initialize database", "error", err)
	}

	defer func() { _ = database.Close() }()

	// 執行資料庫遷移（可選，建議在生產環境使用專門的遷移腳本）
	if err := database.AutoMigrate(); err != nil {
		logger.Fatal("Failed to migrate database", "error", err)
	}

	// 補齊單字學習字表（冪等 upsert，字典檔缺失時僅警告不中止開機——Learn 非核心聊天功能）
	if n, err := database.SeedWords("data/words.csv"); err != nil {
		logger.Warn("Failed to seed learn words, Learn features may be degraded", "error", err)
	} else {
		logger.Info("Learn words seeded", "count", n)
	}

	// 補齊 SRS 例句表（冪等 upsert；字表沒有的字直接跳過，缺檔僅警告不中止開機）
	if imp, skip, err := database.SeedSentences("data/sentences.csv"); err != nil {
		logger.Warn("Failed to seed learn sentences, SRS review may be degraded", "error", err)
	} else {
		logger.Info("Learn sentences seeded", "imported", imp, "skipped", skip)
	}

	// 創建伺服器
	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal("Failed to create server", "error", err)
	}

	// 啟動 HTTP 伺服器
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      srv.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 在 goroutine 中啟動伺服器
	go func() {
		logger.Info("Starting TalkRealm server", "port", cfg.Server.Port)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// 優雅關閉
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
