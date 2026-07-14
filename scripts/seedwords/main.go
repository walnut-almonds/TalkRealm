// 匯入 data/words.csv 到 words 表（upsert，可重複執行）。
//
// 用法：go run ./scripts/seedwords
package main

import (
	"log"

	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := logger.Init(cfg.Log.Level); err != nil {
		log.Fatalf("init logger: %v", err)
	}

	defer logger.Sync()

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("init db: %v", err)
	}

	total, err := database.SeedWords("data/words.csv")
	if err != nil {
		log.Fatalf("seed words: %v", err)
	}

	log.Printf("seeded %d words", total)
}
