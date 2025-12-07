package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/middleware"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
)

// Server 代表應用程式伺服器
type Server struct {
	config         *config.Config
	router         *gin.Engine
	jwtManager     *auth.JWTManager
	userHandler    *handler.UserHandler
	guildHandler   *handler.GuildHandler
	channelHandler *handler.ChannelHandler
	messageHandler *handler.MessageHandler
}

// New 創建新的伺服器實例
func New(cfg *config.Config) (*Server, error) {
	// 設定 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 全局中介軟體
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// 初始化 JWT 管理器
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.ExpirationHours)*time.Hour,
	)

	// 獲取資料庫連接
	db := database.GetDB()

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	guildRepo := repository.NewGuildRepository(db)
	guildMemberRepo := repository.NewGuildMemberRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// 初始化 Service
	userService := service.NewUserService(userRepo, jwtManager)
	guildService := service.NewGuildService(guildRepo, guildMemberRepo)
	guildMemberService := service.NewGuildMemberService(guildRepo, guildMemberRepo)
	channelService := service.NewChannelService(channelRepo, guildRepo, guildMemberRepo)
	messageService := service.NewMessageService(messageRepo, channelRepo, guildMemberRepo)

	// 初始化 Handler
	userHandler := handler.NewUserHandler(userService)
	guildHandler := handler.NewGuildHandler(guildService, guildMemberService)
	channelHandler := handler.NewChannelHandler(channelService)
	messageHandler := handler.NewMessageHandler(messageService)

	s := &Server{
		config:         cfg,
		router:         router,
		jwtManager:     jwtManager,
		userHandler:    userHandler,
		guildHandler:   guildHandler,
		channelHandler: channelHandler,
		messageHandler: messageHandler,
	}

	// 設定路由
	s.setupRoutes()

	return s, nil
}

// setupRoutes 設定所有路由
func (s *Server) setupRoutes() {
	// 健康檢查
	s.router.GET("/health", handler.HealthCheck)
	s.router.GET("/ping", handler.Ping)

	// API v1 路由群組
	v1 := s.router.Group("/api/v1")
	{
		// 公開路由 - 認證相關
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.userHandler.Register)
			auth.POST("/login", s.userHandler.Login)
		}

		// 需要認證的路由
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(s.jwtManager))
		{
			// 使用者相關
			users := protected.Group("/users")
			{
				users.GET("/me", s.userHandler.GetCurrentUser)
				users.PATCH("/me", s.userHandler.UpdateCurrentUser)
			}

			// 伺服器/社群相關
			guilds := protected.Group("/guilds")
			{
				guilds.POST("", s.guildHandler.CreateGuild)
				guilds.GET("", s.guildHandler.ListUserGuilds)
				guilds.GET("/:id", s.guildHandler.GetGuild)
				guilds.PUT("/:id", s.guildHandler.UpdateGuild)
				guilds.DELETE("/:id", s.guildHandler.DeleteGuild)

				// 社群成員操作
				guilds.POST("/:id/join", s.guildHandler.JoinGuild)
				guilds.POST("/:id/leave", s.guildHandler.LeaveGuild)
				guilds.GET("/:id/members", s.guildHandler.ListGuildMembers)
				guilds.DELETE("/:id/members/:userId", s.guildHandler.KickMember)
				guilds.PUT("/:id/members/:userId/role", s.guildHandler.UpdateMemberRole)
			}

			// 頻道相關
			channels := protected.Group("/channels")
			{
				channels.POST("", s.channelHandler.CreateChannel)
				channels.GET("", s.channelHandler.ListGuildChannels)
				channels.GET("/:id", s.channelHandler.GetChannel)
				channels.PUT("/:id", s.channelHandler.UpdateChannel)
				channels.DELETE("/:id", s.channelHandler.DeleteChannel)
				channels.PUT("/:id/position", s.channelHandler.UpdateChannelPosition)
			}

			// 訊息相關
			messages := protected.Group("/messages")
			{
				messages.POST("", s.messageHandler.CreateMessage)
				messages.GET("", s.messageHandler.ListChannelMessages)
				messages.GET("/:id", s.messageHandler.GetMessage)
				messages.PUT("/:id", s.messageHandler.UpdateMessage)
				messages.DELETE("/:id", s.messageHandler.DeleteMessage)
			}
		}
	}
} // Router 返回 gin 路由器
func (s *Server) Router() *gin.Engine {
	return s.router
}
