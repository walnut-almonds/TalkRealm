# TALKREALM

## 專案摘要

TalkRealm 是一個開源的即時通訊解決方案，提供以下核心功能：

- **伺服器/社群管理**：建立和管理多個社群空間
- **即時文字聊天**：WebSocket 即時訊息推送，無需輪詢
- **即時通知**：訊息推送、使用者狀態、正在輸入提示
- **語音聊天室**：高品質的即時語音通訊（開發中）
- **使用者系統**：註冊、登入、個人資料管理
- **權限管理**：角色與權限控制系統
- **頻道系統**：文字頻道和語音頻道分類

## Skills 使用準則

Skill 文件位於 `.agents/skills/*/SKILL.md`。

- 遇到特定領域任務時，優先讀取對應 skill（例如 `uv-package-manager`、`golang-pro`）。
- 進行該領域實作前，先吸收並遵循對應 skill 指引。

## 修改後檢查流程

請依序執行以下命令，確保品質與可執行性：
完成修改後請執行：

1. 全部檢查：`make check`

## Agent Memory（`MEMORY.md`）

`MEMORY.md` 是本儲存庫 AI agent 的共享長期記憶索引。

### 協議（必須遵守）

1. **任務開始前**：讀取 `MEMORY.md`（必要時連同 `memory/*.md`）再分析或修改。
2. **任務完成前**：判斷是否有可重用新知；若有，更新 `MEMORY.md` / `memory/*.md`。
3. **最終回覆前**：確認已完成上述兩步；若無需更新，明確標記「No memory update needed」。

### 原則

- 規則 → `AGENTS.md`；可重用知識 → `MEMORY.md`
- `MEMORY.md` 為精簡索引；長內容放 `memory/*.md` 並連結
- 只記錄跨任務仍有價值的內容（命令、架構、陷阱、決策）
- 精簡條列；新增時同步清理過時內容；禁止記錄機敏資訊

### `MEMORY.md` 結構

`## Quick Facts` · `## Commands` · `## Architecture Notes` · `## Pitfalls` · `## Decisions` · `## Last Updated`
