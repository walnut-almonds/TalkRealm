# TalkRealm 翻譯猜字功能設計文件

## 功能概述

在 TalkRealm 聊天系統中整合一個帶有遊戲性質的翻譯功能。  
使用者在聊天時，對方的訊息會被翻譯成另外兩種語言，用戶可以選擇隱藏原文或譯文，透過「猜字遊戲」的方式學習語言。

支援語言：**中文 / 日文 / 英文（三語互譯）**

---

## 核心設計

### 遊戲機制
- 用戶可設定將對方說話的**原文**或**譯文**隱藏
- 猜測被隱藏的那一側內容
- 猜測正確與否由系統判斷
- 猜對可獲得獎勵（TODO）

### 判斷標準
- **單字模式**：完全匹配字典內的單字/詞彙即算正確
- **語意模式**：由 LLM 判斷語意相似度，達到 **70%** 以上算及格

---

## 分階段開發計畫

### Phase 1 — MVP（字典模式）

**目標**：驗證「猜字遊戲」這個互動模式有沒有人喜歡，成本控制在接近零。

**核心元件**
- Translation Service（呼叫翻譯 API，非同步處理）
- Dictionary Service（單字字典查詢，完全匹配）
- Game State DB（記錄猜測結果）

**流程**
```
用戶發訊息
  → Chat Server
  → [同步] 訊息存入 DB（content_zh / content_ja / content_en）
  → [非同步] 觸發 Translation Service 翻譯 × 2
  → 翻譯完成後 WS Push 通知接收方「可以猜了」
  → 接收方看到隱藏版訊息，輸入猜測
  → Dictionary Service 查字典
  → 猜中 → 顯示正確 + 揭示原文
  → 猜錯 → 顯示錯誤提示
  → 結果寫入 Game State DB
```

**技術選型（免費）**
| 用途 | 服務 | 說明 |
|---|---|---|
| 翻譯 | DeepL Free API | 中日英翻譯品質最穩，免費版每月 500,000 字元 |
| 語意判斷 | 尚未使用 | MVP 階段只用字典，不需要 LLM |

**MVP 驗證假設**
> 用戶真的願意在聊天中「停下來猜」嗎？

---

### Phase 2 — 完整版（字典 + LLM 雙模式）

**目標**：確認有用戶在玩後，加入 LLM 語意判斷，並讓兩套模式並存。

**新增元件**
- Guess Evaluation Service（呼叫 LLM 判斷語意相似度）
- Kafka MQ（topic: translate，讓翻譯觸發更穩定、可重播）
- Reward Service（TODO）

**雙模式設計**
| 模式 | 判斷方式 | 適合對象 |
|---|---|---|
| 單字模式 | Dictionary Service 完全匹配 | 初學者，門檻低 |
| 整句語意模式 | LLM 語意判斷 ≥ 70% | 進階用戶，更自然 |

**流程**
```
用戶發訊息
  → Chat Server
  → [同步] 訊息存入 DB（三語文字 + embeddings）
  → [非同步] 推送到 Kafka topic: translate
  → Translation Service consume → 翻譯 + 生成 embedding → 寫回 DB
  → WS Push 通知接收方
  → 接收方選擇模式（單字 or 整句）
    → 單字模式 → Dictionary Service → 完全匹配判斷
    → 整句模式 → Guess Evaluation Service (LLM) → 相似度 ≥ 70% 判斷
  → 猜中 → 顯示正確 + 揭示原文 + 觸發 Reward Service
  → 猜錯 → 顯示錯誤 + 給予提示
  → 寫入 Game State DB
```

**技術選型（免費測試）**
| 用途 | 服務 | 說明 |
|---|---|---|
| 翻譯 | DeepL Free API | 品質穩定，專為翻譯設計 |
| 語意判斷 | Gemini 1.5 Flash（免費 tier） | 免費額度大、速度快，支援三語 |
| 備選（低延遲） | Groq + Llama 3.1 | 推理速度極快，有免費額度 |
| 備選（本地） | Ollama + Llama 3.1 8B / Qwen2.5 | 完全免費，需要 GPU |

---

## DB Schema 變更

現有訊息表需要擴充，從單一 `content` 欄位改為三語欄位：

```sql
message {
  id
  room_id
  sender_id
  original_lang       -- 'zh' | 'ja' | 'en'
  content_zh          -- 中文內容
  content_ja          -- 日文內容
  content_en          -- 英文內容
  embedding_zh        -- Phase 2 新增，用於語意比對
  embedding_ja        -- Phase 2 新增
  embedding_en        -- Phase 2 新增
  created_at
}

game_state {
  id
  message_id
  guesser_id
  hidden_lang         -- 被隱藏的語言
  guess_content       -- 用戶輸入的猜測
  mode                -- 'dictionary' | 'semantic'
  similarity_score    -- LLM 回傳的相似度（0~1）
  is_correct          -- boolean
  guessed_at
}
```

---

## 系統架構影響

整合進現有 TalkRealm 架構後，新增的服務：

```
Translation Service     → 非同步翻譯，接 MQ 觸發
Dictionary Service      → 單字字典查詢
Guess Evaluation Service → LLM 語意判斷（Phase 2）
Reward Service          → 發放獎勵（TODO）
```

翻譯是**非同步**的，接收方可能在翻譯完成前就收到訊息，前端需要處理「翻譯載入中」的狀態。

---

## 風險與注意事項

| 風險 | 說明 | 對策 |
|---|---|---|
| 對話節奏被打斷 | 猜字會讓對話變慢 | 提供「學習模式」開關，讓用戶自行切換 |
| 翻譯品質影響遊戲體驗 | 翻譯太生硬會讓猜測變無聊 | 優先使用 DeepL，品質穩定 |
| LLM 語意判斷不一致 | 「差不多的意思」定義模糊 | 需要精心設計 prompt，並針對三語測試 |
| API 費用 | 用戶量成長後免費額度不夠 | 確認 PMF 後再評估付費方案（GPT-4o / Claude） |
| 翻譯延遲 | 非同步翻譯完成前用戶已等待 | 前端顯示 loading 狀態，翻譯完成後 WS Push |

---

## TODO

- [ ] Reward Service 獎勵機制設計（積分 / 徽章 / 排行榜）
- [ ] 前端「翻譯載入中」UI 狀態
- [ ] 單字字典資料來源與建置方式
- [ ] LLM Prompt 設計與三語測試
- [ ] 免費 API 額度監控機制
- [ ] 用戶可自訂猜字難度（只猜單字 / 猜整句）