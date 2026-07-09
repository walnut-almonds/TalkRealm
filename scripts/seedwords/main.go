// 匯入 data/words.csv 到 words 表（upsert，可重複執行）。
//
// 用法：go run ./scripts/seedwords
package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
	"gorm.io/gorm/clause"
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

	f, err := os.Open("data/words.csv")
	if err != nil {
		log.Fatalf("open words.csv: %v", err)
	}

	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)

	if _, err := r.Read(); err != nil { // skip header
		log.Fatalf("read header: %v", err)
	}

	db := database.GetDB()
	batch := make([]model.Word, 0, 1000)
	total := 0

	flush := func() {
		if len(batch) == 0 {
			return
		}

		err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "word"}},
			DoUpdates: clause.AssignmentColumns([]string{
				// 注意：GORM naming 把 DefinitionZHTW 轉成 definition_zhtw（無底線分隔）
				"phonetic", "tier", "frequency", "length",
				"definition_en", "definition_zh", "definition_zhtw", "definition_ja",
			}),
		}).Create(&batch).Error
		if err != nil {
			log.Fatalf("upsert batch: %v", err)
		}

		total += len(batch)
		batch = batch[:0]
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("read csv: %v", err)
		}

		tier, _ := strconv.Atoi(rec[2])
		frq, _ := strconv.Atoi(rec[3])

		batch = append(batch, model.Word{
			Word: rec[0], Phonetic: rec[1], Tier: tier, Frequency: frq,
			Length:       len(rec[0]),
			DefinitionEN: rec[4], DefinitionZH: rec[5],
			DefinitionZHTW: rec[6], DefinitionJA: rec[7],
		})

		if len(batch) == 1000 {
			flush()
		}
	}

	flush()
	log.Printf("seeded %d words", total)
}
