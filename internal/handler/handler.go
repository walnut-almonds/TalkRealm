package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康檢查處理器
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "talkrealm",
	})
}

// Ping 簡單的 ping 處理器
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
