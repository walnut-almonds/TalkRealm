# TalkRealm 前端

這是 TalkRealm 的完整前端實現，提供了一個現代化的即時通訊介面。

## 功能特性

### ✅ 已實現功能

#### 使用者認證
- 使用者註冊
- 使用者登入
- 自動登入狀態保持
- JWT Token 管理

#### 社群管理
- 瀏覽社群列表
- 建立新社群
- 選擇和切換社群
- 社群圖示顯示

#### 頻道管理
- 文字頻道列表
- 語音頻道列表
- 建立新頻道
- 頻道描述顯示

#### 即時訊息
- 發送文字訊息
- 接收即時訊息（WebSocket）
- 訊息歷史載入
- 訊息分組顯示
- 時間戳記格式化

#### 成員管理
- 成員列表顯示
- 使用者狀態指示器（線上/離線/忙碌/離開）
- 成員頭像顯示

#### WebSocket 即時功能
- 即時訊息推送
- 頻道訂閱/取消訂閱
- 心跳機制（Ping/Pong）
- 自動重連
- 使用者狀態同步

#### 使用者介面
- 響應式設計
- 深色主題（類 Discord 風格）
- 平滑動畫效果
- 通知提示
- 載入指示器
- 模態視窗

## 檔案結構

```
web/
├── index.html          # 主 HTML 檔案
├── css/
│   └── styles.css      # 樣式表
├── js/
│   ├── config.js       # 配置檔案
│   ├── api.js          # API 請求處理
│   ├── websocket.js    # WebSocket 管理
│   └── app.js          # 主應用邏輯
└── README.md           # 本檔案
```

## 使用方式

### 1. 啟動後端伺服器

確保 TalkRealm 後端伺服器正在運行：

```bash
# 從專案根目錄
cd /workspaces/TalkRealm
go run cmd/server/main.go
```

### 2. 提供前端檔案

您可以使用任何靜態檔案伺服器來提供前端檔案。以下是幾種方式：

#### 方式 1: 使用 Python 簡易伺服器

```bash
cd /workspaces/TalkRealm/web
python3 -m http.server 3000
```

然後在瀏覽器中開啟 `http://localhost:3000`

#### 方式 2: 使用 Node.js http-server

```bash
# 安裝 http-server（如果還沒安裝）
npm install -g http-server

# 啟動伺服器
cd /workspaces/TalkRealm/web
http-server -p 3000
```

#### 方式 3: 使用 VS Code Live Server 擴充功能

1. 在 VS Code 中安裝 "Live Server" 擴充功能
2. 右鍵點擊 `index.html`
3. 選擇 "Open with Live Server"

#### 方式 4: 直接用瀏覽器開啟

```bash
# 在瀏覽器中開啟
file:///workspaces/TalkRealm/web/index.html
```

> ⚠️ 注意：直接用瀏覽器開啟可能會遇到 CORS 問題，建議使用上述的伺服器方式。

### 3. 開始使用

1. **註冊帳號**
   - 輸入電子郵件、使用者名稱和密碼
   - 點擊「註冊」按鈕

2. **登入**
   - 使用註冊的電子郵件和密碼登入
   - 系統會自動保存登入狀態

3. **建立社群**
   - 點擊左側邊欄底部的 "+" 按鈕
   - 輸入社群名稱和描述
   - 點擊「建立」

4. **建立頻道**
   - 選擇一個社群
   - 點擊頻道分類旁的 "+" 按鈕
   - 選擇頻道類型（文字或語音）
   - 輸入頻道名稱和描述
   - 點擊「建立」

5. **發送訊息**
   - 選擇一個頻道
   - 在底部輸入框中輸入訊息
   - 按 Enter 鍵或點擊發送按鈕

## 配置

### API 端點配置

在 `js/config.js` 中修改 API 配置：

```javascript
const API_CONFIG = {
    BASE_URL: 'http://localhost:8080',  // 後端 API URL
    WS_URL: 'ws://localhost:8080',       // WebSocket URL
    // ...
};
```

### 本地儲存

應用程式使用瀏覽器的 localStorage 來保存：
- JWT Token
- 使用者資訊
- 最後訪問的社群
- 最後訪問的頻道

## 技術細節

### 前端技術棧

- **HTML5** - 結構化語意標記
- **CSS3** - 現代化樣式和動畫
- **Vanilla JavaScript** - 無框架依賴
- **WebSocket API** - 即時通訊
- **Fetch API** - HTTP 請求
- **Font Awesome** - 圖示庫

### 設計模式

- **單一頁面應用（SPA）** - 無需重新載入頁面
- **狀態管理** - 集中式應用狀態管理
- **事件驅動** - WebSocket 事件處理
- **響應式設計** - 適配不同螢幕尺寸

### API 通訊

```javascript
// 所有 API 請求都通過 api 實例
await api.login(email, password);
await api.getMyGuilds();
await api.sendMessage(channelId, content);
```

### WebSocket 通訊

```javascript
// WebSocket 訊息處理
wsManager.onMessage((type, data) => {
    switch (type) {
        case 'message':
            // 處理新訊息
            break;
        case 'typing':
            // 處理正在輸入
            break;
        // ...
    }
});
```

## 功能擴充

### 計劃中的功能

- [ ] 訊息編輯和刪除
- [ ] 檔案上傳和圖片分享
- [ ] 表情符號選擇器
- [ ] @提及使用者
- [ ] 訊息搜尋
- [ ] 使用者個人資料頁面
- [ ] 社群設定頁面
- [ ] 頻道權限管理
- [ ] 私人訊息（DM）
- [ ] 語音通話支援
- [ ] 深色/淺色主題切換
- [ ] 多語言支援
- [ ] 通知設定
- [ ] 鍵盤快捷鍵

### 如何添加新功能

1. **添加 API 端點**（在 `js/api.js`）：
```javascript
async newFeature() {
    return this.post('/api/v1/new-feature', {});
}
```

2. **添加 WebSocket 事件處理**（在 `js/app.js`）：
```javascript
case 'new_event':
    handleNewEvent(data);
    break;
```

3. **添加 UI 處理函數**（在 `js/app.js`）：
```javascript
function handleNewEvent(data) {
    // 更新 UI
}
```

4. **添加樣式**（在 `css/styles.css`）：
```css
.new-feature {
    /* 樣式 */
}
```

## 故障排除

### WebSocket 連接失敗

1. 確認後端伺服器正在運行
2. 檢查 WebSocket URL 配置
3. 查看瀏覽器控制台的錯誤訊息
4. 確認防火牆沒有阻擋 WebSocket 連接

### CORS 錯誤

後端已配置 CORS 中介軟體，如果仍遇到 CORS 問題：

1. 確認後端 CORS 設定正確
2. 使用靜態檔案伺服器而非直接開啟 HTML 檔案
3. 檢查 API URL 配置是否正確

### 登入失敗

1. 確認電子郵件和密碼正確
2. 檢查後端伺服器日誌
3. 清除瀏覽器快取和 localStorage
4. 確認資料庫連接正常

### 訊息無法發送

1. 確認已選擇頻道
2. 檢查 WebSocket 連接狀態
3. 查看網路請求是否成功
4. 確認使用者有權限發送訊息

## 效能優化

### 已實現的優化

- 訊息分組顯示（減少 DOM 元素）
- 延遲載入訊息歷史
- WebSocket 心跳機制
- 自動重連機制
- 防抖輸入處理

### 建議的優化

- 虛擬滾動（大量訊息）
- 圖片延遲載入
- Service Worker（離線支援）
- 訊息快取
- 壓縮 JavaScript 和 CSS

## 瀏覽器支援

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

需要支援：
- ES6+ JavaScript
- CSS Grid and Flexbox
- WebSocket API
- Fetch API
- LocalStorage

## 授權

本專案使用與 TalkRealm 主專案相同的授權。

## 貢獻

歡迎提交 Pull Request 或回報問題！

## 聯絡方式

如有問題或建議，請在 GitHub 上開啟 Issue。
