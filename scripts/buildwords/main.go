// scripts/buildwords/main.go
// 離線工具：讀 data/raw/ecdict.csv + data/raw/ejdict-hand-utf8.txt，
// 產出 data/words.csv（commit 進 repo）。只需在字表更新時重跑。
//
// 用法：go run ./scripts/buildwords
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/longbridgeapp/opencc"
)

const (
	ecdictPath = "data/raw/ecdict.csv"
	ejdictPath = "data/raw/ejdict-hand-utf8.txt"
	outPath    = "data/words.csv"
)

func main() {
	ja, err := loadEJDict(ejdictPath)
	if err != nil {
		log.Fatalf("load ejdict: %v", err)
	}

	s2tw, err := opencc.New("s2twp") // 簡→繁（台灣用語）
	if err != nil {
		log.Fatalf("init opencc: %v", err)
	}

	in, err := os.Open(ecdictPath)
	if err != nil {
		log.Fatalf("open ecdict: %v", err)
	}

	defer func() { _ = in.Close() }()

	out, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create out: %v", err)
	}

	defer func() { _ = out.Close() }()

	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	w := csv.NewWriter(out)
	defer w.Flush()

	header, err := r.Read() // ECDICT 首列是 header
	if err != nil {
		log.Fatalf("read header: %v", err)
	}

	col := map[string]int{}
	for i, name := range header {
		col[name] = i
	}

	_ = w.Write([]string{
		"word", "phonetic", "tier", "frequency",
		"definition_en", "definition_zh", "definition_zh_tw", "definition_ja",
	})

	count := 0

	for {
		rec, err := r.Read()
		if err != nil {
			break // io.EOF 或壞列，一律停止/略過
		}

		word := rec[col["word"]]
		translation := rec[col["translation"]]

		if !keepWord(word, translation) {
			continue
		}

		frq, _ := strconv.Atoi(rec[col["frq"]])

		tier := tierOf(rec[col["tag"]], frq)
		if tier == 0 {
			continue
		}

		zhTW, err := s2tw.Convert(translation)
		if err != nil {
			zhTW = translation // 轉換失敗退回簡體
		}

		// ECDICT 的換行以 \n 存在欄位內，釋義只取到合理長度即可
		_ = w.Write([]string{
			word,
			rec[col["phonetic"]],
			strconv.Itoa(tier),
			strconv.Itoa(frq),
			strings.ReplaceAll(rec[col["definition"]], `\n`, "; "),
			strings.ReplaceAll(translation, `\n`, "; "),
			strings.ReplaceAll(zhTW, `\n`, "; "),
			ja[word], // 未命中為空字串，執行期 fallback en
		})
		count++
	}

	fmt.Printf("wrote %d words to %s\n", count, outPath)
}

// loadEJDict 讀 EJDict-hand（word<TAB>釋義；word 欄可能是逗號分隔的多個詞形）
func loadEJDict(path string) (map[string]string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // 離線工具，path 為程式內常數
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)

	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		for _, w := range strings.Split(parts[0], ",") {
			w = strings.ToLower(strings.TrimSpace(w))
			if _, ok := m[w]; !ok {
				m[w] = strings.TrimSpace(parts[1])
			}
		}
	}

	return m, nil
}
