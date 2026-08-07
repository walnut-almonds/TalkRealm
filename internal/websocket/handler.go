package websocket

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Non-browser clients do not send an Origin header.
		}

		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" {
			return false
		}

		return strings.EqualFold(parsed.Host, r.Host)
	},
}

// HandleWebSocket 處理 WebSocket 連接請求
// 注意：此端點不需要 JWT 中間件，認證由 WS identify op 處理
func HandleWebSocket(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 升級 HTTP 連接到 WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}

		// 創建新客戶端（未認證，等待 identify op）
		client := NewClient(conn, manager)

		// 註冊客戶端（會觸發發送 hello）
		manager.RegisterClient(client)

		log.Printf("WebSocket connection established (pending identify)")
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
