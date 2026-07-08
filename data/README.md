# data/

- `words.csv` — 單字學習遊戲字表（committed）。由 `go run ./scripts/buildwords` 產生。
- `raw/` — 原始字典（不 commit）：
  - `ecdict.csv`（https://github.com/skywind3000/ECDICT release）
  - `ejdict-hand-utf8.txt`（https://github.com/kujirahand/EJDict release）

欄位：`word,phonetic,tier,frequency,definition_en,definition_zh,definition_zh_tw,definition_ja`
tier 1–5 = 國中 / 高中 / CET4-6 / TOEFL-IELTS-考研 / GRE。
