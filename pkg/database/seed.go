package database

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"gorm.io/gorm/clause"
)

// SeedWords 從 CSV upsert 單字表（依 word 欄位衝突更新，保留既有 id，可重複執行；
// 呼叫端可安全地每次開機都跑一次，不需要判斷 CSV 是否變更）。
func SeedWords(csvPath string) (int, error) {
	f, err := os.Open(csvPath) //nolint:gosec // 呼叫端一律傳程式內常數路徑，非使用者輸入
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", csvPath, err)
	}

	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)

	if _, err := r.Read(); err != nil { // skip header
		return 0, fmt.Errorf("read header: %w", err)
	}

	batch := make([]model.Word, 0, 1000)
	total := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
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
			return fmt.Errorf("upsert batch: %w", err)
		}

		total += len(batch)
		batch = batch[:0]

		return nil
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return total, fmt.Errorf("read csv: %w", err)
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
			if err := flush(); err != nil {
				return total, err
			}
		}
	}

	if err := flush(); err != nil {
		return total, err
	}

	return total, nil
}
