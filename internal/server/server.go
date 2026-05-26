package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/middleware"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/internal/websocket"
	"github.com/walnut-almonds/talkrealm/pkg/auth"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"github.com/walnut-almonds/talkrealm/pkg/database"
	"github.com/walnut-almonds/talkrealm/pkg/logger"
	pkgredis "github.com/walnut-almonds/talkrealm/pkg/redis"
	"github.com/walnut-almonds/talkrealm/pkg/storage"
	"github.com/walnut-almonds/talkrealm/pkg/voice"
)

// Server 代表應用程式伺服器
type Server struct {
	config             *config.Config
	router             *gin.Engine
	jwtManager         *auth.JWTManager
	wsManager          *websocket.Manager
	userHandler        *handler.UserHandler
	guildHandler       *handler.GuildHandler
	channelHandler     *handler.ChannelHandler
	messageHandler     *handler.MessageHandler
	oauthHandler       *handler.OAuthHandler
	fileHandler        *handler.FileHandler
	voiceHandler       *handler.VoiceHandler
	translationHandler *handler.TranslationHandler
	rdb                *goredis.Client
	guildMemberRepo    repository.GuildMemberRepository
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

	// 初始化 Redis client（失敗時記錄 warning 但繼續啟動）
	rdb, redisErr := pkgredis.NewClient(&cfg.Redis)
	if redisErr != nil {
		// 非致命錯誤：Redis 不可用時降級運行
		_ = redisErr
	}

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	guildRepo := repository.NewGuildRepository(db)
	guildMemberRepo := repository.NewGuildMemberRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	guildInviteRepo := repository.NewGuildInviteRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	oauthProviderRepo := repository.NewOAuthProviderRepository(db)

	// 初始化 WebSocket 管理器（傳入 jwtManager，用於 identify op 驗證）
	wsManager := websocket.NewManager(jwtManager)
	// GuildLookup 讓 WS manager 在 identify 時訂閱使用者的所有 guild（Redis 可選）
	wsManager.SetGuildLookup(guildMemberRepo)

	if rdb != nil {
		wsManager.SetRedis(rdb)
	}

	go wsManager.Run() // 啟動 WebSocket 管理器

	// 初始化 Service
	userService := service.NewUserService(userRepo, refreshTokenRepo, oauthProviderRepo, jwtManager)
	guildService := service.NewGuildService(guildRepo, guildMemberRepo)
	guildMemberService := service.NewGuildMemberService(guildRepo, guildMemberRepo)
	guildInviteService := service.NewGuildInviteService(guildInviteRepo, guildRepo, guildMemberRepo)
	channelService := service.NewChannelService(channelRepo, guildRepo, guildMemberRepo)
	messageService := service.NewMessageService(messageRepo, channelRepo, guildMemberRepo)

	// 設定 WebSocket 管理器到各 Service
	messageService.SetWebSocketManager(wsManager)
	guildService.SetWebSocketManager(wsManager)
	guildMemberService.SetWebSocketManager(wsManager)
	guildInviteService.SetWebSocketManager(wsManager)
	channelService.SetWebSocketManager(wsManager)

	// 設定 MessageSender：讓 WS Manager 能處理 send_message op
	wsManager.SetMessageSender(messageService)

	// 初始化 Handler
	userHandler := handler.NewUserHandler(userService)
	guildHandler := handler.NewGuildHandler(guildService, guildMemberService, guildInviteService)
	guildHandler.SetOnlineChecker(wsManager)
	channelHandler := handler.NewChannelHandler(channelService)
	messageHandler := handler.NewMessageHandler(messageService)
	oauthHandler := handler.NewOAuthHandler(userService, cfg)

	// 初始化 File Service（Minio 可選，失敗時記錄 warning）
	var fileHandler *handler.FileHandler

	minioClient, minioErr := storage.NewClient(&cfg.Minio)
	if minioErr != nil {
		logger.Warn("Minio init failed, file service disabled", "error", minioErr)
	} else {
		fileRepo := repository.NewFileRepository(db)
		fileService := service.NewFileService(fileRepo, minioClient, rdb, &cfg.Minio)
		fileHandler = handler.NewFileHandler(fileService)
		// 設定 FileService 到 MessageService（用於附件關聯）
		messageService.SetFileService(fileService)
	}

	// 初始化 Voice Handler（LiveKit Token 生成）
	voiceManager := voice.NewManager(&cfg.LiveKit)
	voiceHandler := handler.NewVoiceHandler(voiceManager, wsManager)

	// 初始化 Translation Service（DeepL 可選，未設定 api_key 時 enabled=false）
	translationRepo := repository.NewTranslationRepository(db)
	gameStateRepo := repository.NewGameStateRepository(db)
	translationSvc := service.NewTranslationService(translationRepo, &cfg.DeepL)
	translationSvc.SetWebSocketManager(wsManager)
	messageService.SetTranslationService(translationSvc)

	guessSvc := service.NewGuessService(gameStateRepo, translationRepo, &cfg.LLM)
	translationHandler := handler.NewTranslationHandler(translationSvc, guessSvc, messageService)

	s := &Server{
		config:             cfg,
		router:             router,
		jwtManager:         jwtManager,
		wsManager:          wsManager,
		userHandler:        userHandler,
		guildHandler:       guildHandler,
		channelHandler:     channelHandler,
		messageHandler:     messageHandler,
		oauthHandler:       oauthHandler,
		fileHandler:        fileHandler,
		voiceHandler:       voiceHandler,
		rdb:                rdb,
		guildMemberRepo:    guildMemberRepo,
		translationHandler: translationHandler,
	}

	// 設定路由
	s.setupRoutes()

	return s, nil
}

// setupRoutes 設定所有路由
//
//	@title			TalkRealm API
//	@version		1.0
//	@description	TalkRealm 是一個即時通訊平台，提供文字和語音聊天功能。
//	@termsOfService	http://talkrealm.example.com/terms/
//
//	@contact.name	API Support
//	@contact.url	http://www.talkrealm.example.com/support
//	@contact.email	support@talkrealm.example.com
func (s *Server) setupRoutes() {
	// 提供靜態檔案 (前端 Vue 構建輸出)
	s.router.Static("/assets", "./web/dist/assets")
	s.router.StaticFile("/", "./web/dist/index.html")
	// SPA fallback: 所有非 API 路由都返回 index.html
	s.router.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}

		c.File("./web/dist/index.html")
	})

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
			auth.POST("/refresh", s.userHandler.RefreshToken)
			auth.POST("/logout", s.userHandler.Logout)

			// Google OAuth
			auth.GET("/google", s.oauthHandler.GoogleLogin)
			auth.GET("/google/callback", s.oauthHandler.GoogleCallback)
		}

		// 公開路由 - 使用者公開資料
		v1.GET("/users/:id", s.userHandler.GetPublicUser)

		// 公開路由 - 邀請碼資訊（可無需登入查詢）
		v1.GET("/invites/:code", s.guildHandler.GetInvite)

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
				guilds.GET("/me", s.guildHandler.ListUserGuilds)
				guilds.GET("/:id", s.guildHandler.GetGuild)
				guilds.PUT("/:id", s.guildHandler.UpdateGuild)
				guilds.PATCH("/:id", s.guildHandler.UpdateGuild)
				guilds.DELETE("/:id", s.guildHandler.DeleteGuild)

				// 社群成員操作
				guilds.POST("/:id/join", s.guildHandler.JoinGuild)
				guilds.POST("/:id/leave", s.guildHandler.LeaveGuild)
				guilds.GET("/:id/members", s.guildHandler.ListGuildMembers)
				// 踢成員 & 更新角色需要至少 admin 角色
				guilds.DELETE("/:id/members/:userId",
					middleware.RequireGuildRole("admin", s.guildMemberRepo),
					s.guildHandler.KickMember,
				)
				guilds.PUT("/:id/members/:userId/role",
					middleware.RequireGuildRole("admin", s.guildMemberRepo),
					s.guildHandler.UpdateMemberRole,
				)

				// 社群邀請碼
				guilds.POST("/:id/invites", s.guildHandler.CreateInvite)
				guilds.POST("/join-by-invite", s.guildHandler.JoinByInvite)

				// 社群頻道
				guilds.GET("/:id/channels", s.channelHandler.ListGuildChannels)
				// 建立頻道需要至少 admin 角色
				guilds.POST("/:id/channels",
					middleware.RequireGuildRole("admin", s.guildMemberRepo),
					s.channelHandler.CreateChannel,
				)
			}

			// 頻道相關
			channels := protected.Group("/channels")
			{
				channels.GET("/:id", s.channelHandler.GetChannel)
				channels.PUT("/:id", s.channelHandler.UpdateChannel)
				channels.PATCH("/:id", s.channelHandler.UpdateChannel)
				channels.DELETE("/:id", s.channelHandler.DeleteChannel)
				channels.PUT("/:id/position", s.channelHandler.UpdateChannelPosition)

				// 頻道訊息
				channels.GET("/:id/messages", s.messageHandler.ListChannelMessages)
				// 訊息發送套用 rate limit：每秒最多 10 則
				if s.rdb != nil {
					channels.POST("/:id/messages",
						middleware.MessageRateLimit(s.rdb, 10),
						s.messageHandler.CreateMessage,
					)
				} else {
					channels.POST("/:id/messages", s.messageHandler.CreateMessage)
				}

				// 語音 Token（LiveKit）
				channels.GET("/:id/voice/token", s.voiceHandler.GetVoiceToken)
				channels.GET("/:id/voice/participants", s.voiceHandler.GetVoiceParticipants)
			}

			// 訊息相關
			messages := protected.Group("/messages")
			{
				messages.GET("/:id", s.messageHandler.GetMessage)
				messages.PUT("/:id", s.messageHandler.UpdateMessage)
				messages.PATCH("/:id", s.messageHandler.UpdateMessage)
				messages.DELETE("/:id", s.messageHandler.DeleteMessage)

				// 翻譯 & 猜測遊戲
				messages.GET("/:id/translation", s.translationHandler.GetTranslation)
				messages.GET("/:id/translation/ensure", s.translationHandler.EnsureTranslation)
				messages.POST("/:id/translation", s.translationHandler.RequestTranslation)
				messages.POST("/:id/guess", s.translationHandler.SubmitGuess)
				messages.GET("/:id/game", s.translationHandler.GetGameStatus)
			}
		}

		// 檔案服務（需要 Minio 啟動；若 Minio 未設定則回傳 503 而非 404）
		files := protected.Group("/files")
		{
			if s.fileHandler != nil {
				files.POST("/presign", s.fileHandler.PresignUpload)
				files.POST("/:id/confirm", s.fileHandler.ConfirmUpload)
				files.GET("/:id", s.fileHandler.GetFile)
				files.GET("/:id/url", s.fileHandler.GetDownloadURL)
				files.DELETE("/:id", s.fileHandler.DeleteFile)
			} else {
				fileUnavailable := func(c *gin.Context) {
					c.JSON(
						http.StatusServiceUnavailable,
						gin.H{"error": "file service unavailable: storage not configured"},
					)
				}
				files.POST("/presign", fileUnavailable)
				files.POST("/:id/confirm", fileUnavailable)
				files.GET("/:id", fileUnavailable)
				files.GET("/:id/url", fileUnavailable)
				files.DELETE("/:id", fileUnavailable)
			}
		}

		// WebSocket 連線（不需要 JWT 中間件，認證由 identify op 處理）
		v1.GET("/ws", websocket.HandleWebSocket(s.wsManager))
	}
} // Router 返回 gin 路由器
func (s *Server) Router() *gin.Engine {
	return s.router
}
