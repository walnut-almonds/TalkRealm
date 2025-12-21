package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生產環境中，應該檢查來源
		// TODO: 實現適當的 CORS 檢查
		return true
	},
}

// HandleWebSocket 處理 WebSocket 連接請求
func HandleWebSocket(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從上下文中獲取使用者資訊（由認證中介軟體設置）
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授權"})
			return
		}

		username, exists := c.Get("username")
		if !exists {
			username = "unknown"
		}

		// 升級 HTTP 連接到 WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}

		// 創建新客戶端
		client := NewClient(conn, manager, userID.(uint), username.(string))

		// 註冊客戶端
		manager.RegisterClient(client)

		log.Printf("WebSocket connection established for user %s (ID: %d)", username, userID)
	}
}

// ExtractUserFromContext 從 Gin 上下文中提取使用者資訊
// 這個函數輔助認證中介軟體使用
func ExtractUserFromContext(c *gin.Context) (*auth.Claims, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, false
	}

	username, _ := c.Get("username")
	email, _ := c.Get("email")

	claims := &auth.Claims{
		UserID:   userID.(uint),
		Username: username.(string),
		Email:    email.(string),
	}

	return claims, true
}
