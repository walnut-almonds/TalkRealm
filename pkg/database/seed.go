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

// SeedSentences 從 CSV upsert 例句表（依 text_en 衝突更新）。CSV 欄位：
// word,answer,text_en,text_zh,text_zh_tw,text_ja。以 word 欄位查 words.id 解出 word_id；
// 字表沒有的 word 直接跳過（回傳 skipped 數）。可安全每次開機重跑。
func SeedSentences(csvPath string) (imported, skipped int, err error) {
	f, err := os.Open(csvPath) //nolint:gosec // 呼叫端一律傳程式內常數路徑，非使用者輸入
	if err != nil {
		return 0, 0, fmt.Errorf("open %s: %w", csvPath, err)
	}

	defer func() { _ = f.Close() }()

	// 先把 words 全表的 word→id 讀進記憶體，避免每列一次 query
	wordID := map[string]uint{}

	var words []model.Word
	if err := db.Select("id", "word").Find(&words).Error; err != nil {
		return 0, 0, fmt.Errorf("load words: %w", err)
	}

	for _, w := range words {
		wordID[w.Word] = w.ID
	}

	r := csv.NewReader(f)

	if _, err := r.Read(); err != nil { // skip header
		return 0, 0, fmt.Errorf("read header: %w", err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return imported, skipped, fmt.Errorf("read csv: %w", err)
		}

		id, ok := wordID[rec[0]]
		if !ok {
			skipped++

			continue
		}

		row := model.LearnSentence{
			WordID: id, Answer: rec[1], TextEN: rec[2],
			TextZH: rec[3], TextZHTW: rec[4], TextJA: rec[5],
		}

		err = db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "text_en"}},
			// GORM naming 把 TextZHTW 轉成 text_zhtw（無底線分隔，同 DefinitionZHTW 慣例）
			DoUpdates: clause.AssignmentColumns([]string{
				"word_id", "answer", "text_zh", "text_zhtw", "text_ja",
			}),
		}).Create(&row).Error
		if err != nil {
			return imported, skipped, fmt.Errorf("upsert sentence: %w", err)
		}

		imported++
	}

	return imported, skipped, nil
}
