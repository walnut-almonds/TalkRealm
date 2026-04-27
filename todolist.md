# TalkRealm TODO List

## 🔥 核心功能

- [x] **使用者系統**
  - [x] 註冊與登入
  - [x] JWT 身份驗證 (`pkg/auth/jwt.go`)
- [x] **伺服器/社群系統 (Guilds)**
  - [x] 建立與管理社群 (`internal/handler/guild_handler.go`)
  - [x] 頻道系統 (文字與語音頻道分類)
- [x] **即時通訊 (WebSocket)**
  - [x] WebSocket 連線與狀態管理 (`internal/websocket/`)
  - [x] 即時文字聊天與推播
- [ ] **權限系統**
  - [ ] 角色與複雜權限控制系統 (RBAC)
- [ ] **語音聊天室**
  - [ ] 高品質的即時語音通訊 (WebRTC 整合 - 開發中)
- [ ] **SDK**
  - [ ] Go SDK (`sdk/go/`)
  - [ ] TypeScript/JS SDK (`sdk/ts/`)
  - [ ] 嘗試 https://github.com/microsoft/kiota 生成 SDK 文件

## 🏗️ 基礎架構 & 部署

- [x] **容器化**
  - [x] Dockerfile 與服務打包
  - [x] Docker Compose 本地開發環境 (Postgres + Redis)
- [x] **Kubernetes 部署配置**
  - [x] Kustomize Base & Overlays (`deploy/k8s/`)
- [ ] **CI/CD**
  - [ ] GitHub Actions 自動化建置與測試 pipelines

## 🧪 測試 & 品質

- [ ] **Unit Test** — 擴展與提升各模組 (Handler, Service, Repository) 的單元測試覆蓋率
- [ ] **Integration Test** — 使用 `docker-compose` 跑完整 Postgres + Redis local 測試
- [ ] **E2E Test** — API 端到端功能測試

## 📚 文件 & 前端整合

- [x] **API 文件**
  - [x] OpenAPI / Swagger 定義 (`docs/openapi/`)
- [x] **基礎前端**
  - [x] Web 客戶端實作與 WebSocket 介接 (`web/`)
- [ ] **進階開發者指南**
  - [ ] 完善架構圖與開發規範文件
