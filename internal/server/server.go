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
	dmHandler          *handler.DMHandler
	friendHandler      *handler.FriendHandler
	interactionHandler *handler.InteractionHandler
	rdb                *goredis.Client
	guildMemberRepo    repository.GuildMemberRepository
	unreadHandler      *handler.UnreadHandler
	learnHandler       *handler.LearnHandler
	feedHandler        *handler.FeedHandler
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
	channelReadStateRepo := repository.NewChannelReadStateRepository(db)
	messageMentionRepo := repository.NewMessageMentionRepository(db)
	oauthProviderRepo := repository.NewOAuthProviderRepository(db)

	// 初始化 WebSocket 管理器（傳入 jwtManager，用於 identify op 驗證）
	wsManager := websocket.NewManager(jwtManager)
	// GuildLookup 讓 WS manager 在 identify 時訂閱使用者的所有 guild（Redis 可選）
	wsManager.SetGuildLookup(guildMemberRepo)
	// ChannelAccessLookup 驗證 identify、subscribe 與 typing 事件的頻道存取權限。
	wsManager.SetChannelAccessLookup(channelRepo)
	// UserLookup 讓 WS manager 在 identify 時判斷自選狀態（invisible 不廣播上線）
	wsManager.SetUserLookup(userRepo)

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
	messageService := service.NewMessageService(
		messageRepo,
		channelRepo,
		guildMemberRepo,
		messageMentionRepo,
	)

	// DM 服務
	dmService := service.NewDMService(channelRepo, userRepo)

	// 動態牆按讚 repository
	messageService.SetLikeRepo(repository.NewMessageLikeRepository(db))

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

	fileRepo := repository.NewFileRepository(db)

	minioClient, minioErr := storage.NewClient(&cfg.Minio)
	if minioErr != nil {
		logger.Warn("Minio init failed, file service disabled", "error", minioErr)
	} else {
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
	dmHandler := handler.NewDMHandler(dmService, messageService, translationSvc)

	// 初始化好友服務
	friendshipRepo := repository.NewFriendshipRepository(db)
	friendSvc := service.NewFriendService(friendshipRepo, userRepo)
	friendSvc.SetNotifier(wsManager)
	friendHandler := handler.NewFriendHandler(friendSvc)

	// 未讀服務
	unreadService := service.NewUnreadService(
		channelReadStateRepo,
		channelRepo,
		guildMemberRepo,
		messageRepo,
	)
	unreadHandler := handler.NewUnreadHandler(unreadService)

	// 互動統計 Handler
	interactionHandler := handler.NewInteractionHandler(messageRepo, guildMemberRepo)

	// 單字學習服務（模組邊界：learn 只依賴 userRepo 的 GetByID，經 interface 注入）
	learnRepo := repository.NewLearnRepository(db)

	var levelStore service.LevelStore
	if rdb != nil {
		levelStore = service.NewRedisLevelStore(rdb)
	} else {
		// ponytail: 單機降級；多實例部署必須有 Redis
		levelStore = service.NewMemoryLevelStore()
	}

	learnService := service.NewLearnService(learnRepo, userRepo, friendshipRepo, levelStore)
	learnHandler := handler.NewLearnHandler(learnService)

	// 個人動態牆（feed）服務：重用既有 friendshipRepo 提供追蹤建議
	feedService := service.NewFeedService(
		repository.NewFollowRepository(db),
		repository.NewFeedPostRepository(db),
		repository.NewFeedPostLikeRepository(db),
		repository.NewFeedCommentRepository(db),
		friendshipRepo,
	)
	feedService.SetCommentLikeRepo(repository.NewFeedCommentLikeRepository(db))
	feedService.SetFileLookup(fileRepo)
	feedService.SetWebSocketManager(wsManager)
	feedHandler := handler.NewFeedHandler(feedService)

	// 固定關卡開機冪等生成（已存在的關卡不重生）；失敗僅降級警告，不擋開機
	if n, err := learnService.EnsureCampaignLevels(); err != nil {
		logger.Warn("Failed to ensure learn campaign levels", "error", err, "created", n)
	} else if n > 0 {
		logger.Info("Learn campaign levels generated", "created", n)
	}

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
		dmHandler:          dmHandler,
		friendHandler:      friendHandler,
		interactionHandler: interactionHandler,
		unreadHandler:      unreadHandler,
		learnHandler:       learnHandler,
		feedHandler:        feedHandler,
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
				users.GET("/search", s.userHandler.SearchUsers)
				users.GET("/me/interaction-stats", s.interactionHandler.GetInteractionStats)
				users.GET("/me/unread", s.unreadHandler.GetAllUnread)
			}

			// 好友相關
			friends := protected.Group("/friends")
			{
				friends.GET("", s.friendHandler.ListFriends)
				friends.GET("/requests/incoming", s.friendHandler.ListIncomingRequests)
				friends.GET("/requests/outgoing", s.friendHandler.ListOutgoingRequests)
				friends.POST("", s.friendHandler.SendRequest)
				friends.PUT("/:userId/accept", s.friendHandler.AcceptRequest)
				friends.DELETE("/:userId", s.friendHandler.RemoveFriend)
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
				// 動態牆貼文列表
				channels.GET("/:id/posts", s.messageHandler.ListPosts)
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
				channels.GET("/:id/unread", s.unreadHandler.GetChannelUnread)
				channels.POST("/:id/ack", s.unreadHandler.AckChannel)
			}

			// 訊息相關
			messages := protected.Group("/messages")
			{
				messages.GET("/:id", s.messageHandler.GetMessage)
				messages.GET("/:id/permalink", s.messageHandler.ResolvePermalink)
				messages.PUT("/:id", s.messageHandler.UpdateMessage)
				messages.PATCH("/:id", s.messageHandler.UpdateMessage)
				messages.DELETE("/:id", s.messageHandler.DeleteMessage)

				// 動態牆留言與按讚
				messages.GET("/:id/comments", s.messageHandler.ListComments)
				messages.PUT("/:id/like", s.messageHandler.LikePost)
				messages.DELETE("/:id/like", s.messageHandler.UnlikePost)

				// 翻譯 & 猜測遊戲
				messages.GET("/:id/translation", s.translationHandler.GetTranslation)
				messages.GET("/:id/translation/ensure", s.translationHandler.EnsureTranslation)
				messages.POST("/:id/translation", s.translationHandler.RequestTranslation)
				messages.POST("/:id/guess", s.translationHandler.SubmitGuess)
				messages.GET("/:id/game", s.translationHandler.GetGameStatus)
			}

			// 私訊（DM）相關
			dm := protected.Group("/dm")
			{
				dm.POST("/channels", s.dmHandler.GetOrCreateDMChannel)
				dm.GET("/channels", s.dmHandler.ListDMChannels)
				dm.GET("/channels/:id/messages", s.dmHandler.ListDMMessages)
				dm.POST("/channels/:id/messages", s.dmHandler.SendDMMessage)
				dm.PATCH("/messages/:id", s.dmHandler.UpdateDMMessage)
				dm.DELETE("/messages/:id", s.dmHandler.DeleteDMMessage)
				dm.GET("/messages/:id/translation", s.dmHandler.GetDMTranslation)
				dm.GET("/messages/:id/translation/ensure", s.dmHandler.EnsureDMTranslation)
			}

			// 單字學習
			learn := protected.Group("/learn")
			{
				learn.POST("/levels", s.learnHandler.CreateLevel)
				learn.POST("/levels/crossword", s.learnHandler.CreateCrossword)
				learn.POST("/levels/:id/guess", s.learnHandler.Guess)
				learn.POST("/levels/:id/hint", s.learnHandler.Hint)
				learn.POST("/levels/:id/reveal", s.learnHandler.Reveal)
				learn.GET("/stats", s.learnHandler.GetStats)
				learn.GET("/daily", s.learnHandler.GetDaily)
				learn.GET("/daily/leaderboard", s.learnHandler.GetDailyLeaderboard)
				learn.GET("/campaign", s.learnHandler.GetCampaign)
				learn.GET("/campaign/leaderboard", s.learnHandler.GetCampaignLeaderboard)
				learn.POST("/campaign/:no", s.learnHandler.StartCampaign)
				learn.GET("/leaderboard/weekly", s.learnHandler.GetWeeklyLeaderboard)
				learn.GET("/srs/overview", s.learnHandler.GetSRSOverview)
				learn.POST("/srs", s.learnHandler.StartSRS)
				learn.POST("/srs/:id/answer", s.learnHandler.AnswerSRS)
			}

			// 個人動態牆（跨社群 feed）
			feed := protected.Group("/feed")
			{
				feed.GET("/suggestions", s.feedHandler.Suggestions)
				feed.POST("/follows/:userId", s.feedHandler.Follow)
				feed.DELETE("/follows/:userId", s.feedHandler.Unfollow)
				feed.GET("/users/:userId/following", s.feedHandler.ListFollowing)
				feed.GET("/users/:userId/followers", s.feedHandler.ListFollowers)
				feed.GET("/users/:userId/posts", s.feedHandler.ProfilePosts)
				feed.GET("/timeline", s.feedHandler.Timeline)
				feed.GET("/discover", s.feedHandler.Discover)
				feed.POST("/posts", s.feedHandler.CreatePost)
				feed.PUT("/posts/:id", s.feedHandler.UpdatePost)
				feed.DELETE("/posts/:id", s.feedHandler.DeletePost)
				feed.PUT("/posts/:id/like", s.feedHandler.LikePost)
				feed.DELETE("/posts/:id/like", s.feedHandler.UnlikePost)
				feed.GET("/posts/:id/comments", s.feedHandler.ListComments)
				feed.POST("/posts/:id/comments", s.feedHandler.AddComment)
				feed.PUT("/comments/:id", s.feedHandler.UpdateComment)
				feed.DELETE("/comments/:id", s.feedHandler.DeleteComment)
				feed.PUT("/comments/:id/like", s.feedHandler.LikeComment)
				feed.DELETE("/comments/:id/like", s.feedHandler.UnlikeComment)
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

		// Open Graph 預覽（需要認證，防止未授權爬取）
		protected.GET("/og", handler.GetOGPreview)
	}
} // Router 返回 gin 路由器
func (s *Server) Router() *gin.Engine {
	return s.router
}
