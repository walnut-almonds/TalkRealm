package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"gorm.io/gorm"
)

const (
	messageTypeText = "text"
	roleOwner       = "owner"
	roleAdmin       = "admin"

	// MaxMessageContentLen 單則訊息內容上限（rune 數）。
	// 沒有上限時任何使用者都能塞爆 DB 與所有訂閱者的 WS buffer。
	MaxMessageContentLen = 4000

	// maxMentionsPerMessage 單則訊息最多處理幾個相異 <@id>，
	// 用來限制成員驗證的查詢次數上界。
	maxMentionsPerMessage = 50
)

// AllowedReactions 可用的表情回應。刻意是固定小集合而非任意 unicode：
// 白名單比對一行就驗完，也省掉前端的 emoji picker 與後端的 grapheme cluster 判定。
var AllowedReactions = map[string]bool{
	"👍":  true,
	"🎉":  true,
	"😂":  true,
	"❤️": true,
	"😮":  true,
	"😢":  true,
}

var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrNotChannelMemberMsg = errors.New("not a member of this channel's guild")
	ErrNotMessageOwner     = errors.New("not the owner of this message")
	ErrEmptyMessageContent = errors.New("message content cannot be empty")
	ErrMessageTooLong      = fmt.Errorf(
		"message content exceeds %d characters",
		MaxMessageContentLen,
	)
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrDuplicateNonce     = errors.New("duplicate nonce: message already exists")
	ErrFileNotAttachable  = errors.New("file is not available for this message")
	ErrInvalidReaction    = errors.New("unsupported reaction emoji")
)

// WebSocketManager 定義 WebSocket 管理器的介面（避免循環依賴）
type WebSocketManager interface {
	BroadcastToChannel(channelID uint, msgType string, data any)
	BroadcastToUser(userID uint, msgType string, data any)
	// IsUserOnline 檢查使用者是否在線（用於 @here 展開過濾）
	IsUserOnline(userID uint) bool
}

// MentionNotifier 通知被 @ 提及的使用者（WS push）
type MentionCreatePayload struct {
	MessageID uint   `json:"message_id"`
	ChannelID uint   `json:"channel_id"`
	GuildID   *uint  `json:"guild_id"`
	AuthorID  uint   `json:"author_id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
}

// mentionRe 解析 <@123> 格式的 user mention；hereRe/everyoneRe 解析廣播提及。
// 每則訊息都會跑到，故一律在 package 層編譯一次。
var (
	mentionRe  = regexp.MustCompile(`<@(\d+)>`)
	hereRe     = regexp.MustCompile(`(?i)@here\b`)
	everyoneRe = regexp.MustCompile(`(?i)@everyone\b`)
)

// MessageService 訊息服務介面
type MessageService interface {
	CreateMessage(userID uint, req *CreateMessageRequest) (*model.Message, error)
	GetMessage(messageID, userID uint) (*model.Message, error)
	ListChannelMessages(
		channelID, userID uint,
		limit int,
		before uint,
	) (*MessageListResponse, error)
	UpdateMessage(messageID, userID uint, req *UpdateMessageRequest) (*model.Message, error)
	DeleteMessage(messageID, userID uint) error
	SetWebSocketManager(manager WebSocketManager)
	SetFileService(fs FileService)
	SetTranslationService(ts TranslationService)
	SetLikeRepo(r repository.MessageLikeRepository)
	SetReactionRepo(r repository.MessageReactionRepository)
	ToggleReaction(messageID, userID uint, emoji string) (bool, error)
	LikePost(messageID, userID uint) (int64, error)
	UnlikePost(messageID, userID uint) (int64, error)
	ListPosts(channelID, userID uint, limit int, before uint) (*PostListResponse, error)
	ListComments(postID, userID uint, limit int, before uint) (*WallCommentListResponse, error)
	ResolvePermalink(messageID, viewerID uint) (*PermalinkResponse, error)
	MessagesAround(channelID, userID, aroundID uint, limit int) (*MessageListResponse, error)
	PostsAround(channelID, userID, aroundID uint, limit int) (*PostListResponse, error)
	// CreateMessageWS 提供給 WebSocket send_message op 的薄包裝
	CreateMessageWS(
		userID, channelID uint,
		content, contentType, nonce string,
		fileIDs []uint,
	) (any, error)
}

type messageService struct {
	messageRepo        repository.MessageRepository
	channelRepo        repository.ChannelRepository
	guildMemberRepo    repository.GuildMemberRepository
	mentionRepo        repository.MessageMentionRepository
	likeRepo           repository.MessageLikeRepository
	reactionRepo       repository.MessageReactionRepository
	wsManager          WebSocketManager
	fileService        FileService        // 可選，用於建立附件關聯
	translationService TranslationService // 可選，用於非同步翻譯
}

// NewMessageService 建立訊息服務實例
func NewMessageService(
	messageRepo repository.MessageRepository,
	channelRepo repository.ChannelRepository,
	guildMemberRepo repository.GuildMemberRepository,
	mentionRepo repository.MessageMentionRepository,
) MessageService {
	return &messageService{
		messageRepo:     messageRepo,
		channelRepo:     channelRepo,
		guildMemberRepo: guildMemberRepo,
		mentionRepo:     mentionRepo,
		wsManager:       nil,
		fileService:     nil, // 稍後透過 SetFileService 設定
	}
}

// SetFileService 設定 FileService（用於建立附件關聯）
func (s *messageService) SetFileService(fs FileService) {
	s.fileService = fs
}

// SetWebSocketManager 設定 WebSocket 管理器
func (s *messageService) SetWebSocketManager(manager WebSocketManager) {
	s.wsManager = manager
}

// SetTranslationService 設定翻譯服務（用於非同步翻譯）
func (s *messageService) SetTranslationService(ts TranslationService) {
	s.translationService = ts
}

// SetLikeRepo 設定按讚 repository（動態牆功能所需）
func (s *messageService) SetLikeRepo(r repository.MessageLikeRepository) { s.likeRepo = r }

// SetReactionRepo 設定表情回應 repository
func (s *messageService) SetReactionRepo(r repository.MessageReactionRepository) {
	s.reactionRepo = r
}

// ToggleReaction 對訊息加上或收回一個表情回應，回傳 true 代表這次是新增。
// 授權與按讚共用 messageAccessCheck：非 guild 成員／非 DM 參與者一律擋下。
func (s *messageService) ToggleReaction(
	messageID, userID uint,
	emoji string,
) (bool, error) {
	if !AllowedReactions[emoji] {
		return false, ErrInvalidReaction
	}

	if s.reactionRepo == nil {
		return false, ErrInvalidReaction
	}

	msg, err := s.messageAccessCheck(messageID, userID)
	if err != nil {
		return false, err
	}

	added, err := s.reactionRepo.Toggle(messageID, userID, emoji)
	if err != nil {
		return false, err
	}

	if s.wsManager != nil {
		action := "remove"
		if added {
			action = "add"
		}

		s.wsManager.BroadcastToChannel(msg.ChannelID, "message_reaction", map[string]any{
			"message_id": messageID,
			"channel_id": msg.ChannelID,
			"user_id":    userID,
			"emoji":      emoji,
			"action":     action,
		})
	}

	return added, nil
}

// messageAccessCheck 確認訊息存在且使用者可存取其 guild 或 DM 頻道，回傳訊息。
// 按讚與表情回應共用這條授權路徑。
func (s *messageService) messageAccessCheck(messageID, userID uint) (*model.Message, error) {
	msg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	channel, err := s.channelRepo.GetByID(msg.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, msg.ChannelID, userID); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *messageService) broadcastLikeCount(msg *model.Message) (int64, error) {
	n, err := s.likeRepo.CountByMessageID(msg.ID)
	if err != nil {
		return 0, err
	}

	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(msg.ChannelID, "post_like", map[string]any{
			"message_id": msg.ID,
			"like_count": n,
		})
	}

	return n, nil
}

// LikePost 按讚（idempotent）
func (s *messageService) LikePost(messageID, userID uint) (int64, error) {
	msg, err := s.messageAccessCheck(messageID, userID)
	if err != nil {
		return 0, err
	}

	if err := s.likeRepo.Create(
		&model.MessageLike{MessageID: messageID, UserID: userID},
	); err != nil {
		return 0, err
	}

	return s.broadcastLikeCount(msg)
}

// UnlikePost 收回讚
func (s *messageService) UnlikePost(messageID, userID uint) (int64, error) {
	msg, err := s.messageAccessCheck(messageID, userID)
	if err != nil {
		return 0, err
	}

	if err := s.likeRepo.Delete(messageID, userID); err != nil {
		return 0, err
	}

	return s.broadcastLikeCount(msg)
}

// CreateMessageWS 給 WebSocket send_message op 使用的薄包裝
func (s *messageService) CreateMessageWS(
	userID, channelID uint,
	content, contentType, nonce string,
	fileIDs []uint,
) (any, error) {
	return s.CreateMessage(userID, &CreateMessageRequest{
		ChannelID: channelID,
		Content:   content,
		Type:      contentType,
		Nonce:     nonce,
		FileIDs:   fileIDs,
	})
}

// CreateMessageRequest 建立訊息請求
type CreateMessageRequest struct {
	ChannelID uint   `json:"channel_id"`
	Content   string `json:"content"`
	Type      string `json:"type"`      // text, image, file (預設: text)
	Nonce     string `json:"nonce"`     // client 產生的冪等 key（可選，建議 UUID v4）
	FileIDs   []uint `json:"file_ids"`  // 附加的已確認檔案 ID（可選）
	ParentID  *uint  `json:"parent_id"` // 有值 = 留言，指向貼文（單層）
}

// UpdateMessageRequest 更新訊息請求
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// MessageListResponse 訊息列表回應
type MessageListResponse struct {
	Messages []*model.Message `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

// PermalinkResponse 跨頻道永久連結解析結果
type PermalinkResponse struct {
	Message PermalinkMessage `json:"message"`
	Channel PermalinkChannel `json:"channel"`
}

type PermalinkMessage struct {
	ID      uint       `json:"id"`
	Content string     `json:"content"`
	Author  model.User `json:"author"`
}

type PermalinkChannel struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	GuildID *uint  `json:"guild_id"`
	Type    string `json:"type"`
}

// PostResponse 貼文（含讚數、留言數、我是否讚過）
type PostResponse struct {
	*model.Message

	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	LikedByMe    bool  `json:"liked_by_me"`
}

// PostListResponse 貼文列表回應
type PostListResponse struct {
	Posts   []*PostResponse `json:"posts"`
	HasMore bool            `json:"has_more"`
}

// WallCommentResponse 動態牆留言（含讚數、我是否讚過）
// ponytail: Wall 前綴避開與 feed_service.go 同名 DTO 衝突（同 package）
type WallCommentResponse struct {
	*model.Message

	LikeCount int64 `json:"like_count"`
	LikedByMe bool  `json:"liked_by_me"`
}

// WallCommentListResponse 動態牆留言列表回應
type WallCommentListResponse struct {
	Comments []*WallCommentResponse `json:"comments"`
	HasMore  bool                   `json:"has_more"`
}

// ListPosts 回傳 feed 頻道貼文（新到舊），每筆帶讚數/留言數/我是否讚過
func (s *messageService) ListPosts(
	channelID, userID uint,
	limit int,
	before uint,
) (*PostListResponse, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, channelID, userID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	posts, err := s.messageRepo.GetPostsByChannelCursor(channelID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(posts) > limit {
		hasMore = true
		posts = posts[:limit]
	}

	return &PostListResponse{Posts: s.enrichPosts(posts, userID), HasMore: hasMore}, nil
}

// enrichPosts 為貼文批次補上讚數/留言數/我是否讚過，供 ListPosts 與 PostsAround 共用。
func (s *messageService) enrichPosts(posts []*model.Message, userID uint) []*PostResponse {
	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	var likeCounts map[uint]int64

	var liked map[uint]bool

	if s.likeRepo != nil {
		likeCounts, _ = s.likeRepo.CountByMessageIDs(ids)
		liked, _ = s.likeRepo.LikedMessageIDs(userID, ids)
	}

	commentCounts, _ := s.messageRepo.CountCommentsByPostIDs(ids)

	out := make([]*PostResponse, len(posts))
	for i, p := range posts {
		out[i] = &PostResponse{
			Message:      p,
			LikeCount:    likeCounts[p.ID],
			CommentCount: commentCounts[p.ID],
			LikedByMe:    liked[p.ID],
		}
	}

	return out
}

// ListComments 回傳某貼文的留言（時間順序）
func (s *messageService) ListComments(
	postID, userID uint,
	limit int,
	before uint,
) (*WallCommentListResponse, error) {
	post, err := s.messageRepo.GetByID(postID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	channel, err := s.channelRepo.GetByID(post.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, post.ChannelID, userID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	comments, err := s.messageRepo.GetCommentsByPostCursor(postID, before, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(comments) > limit {
		hasMore = true
		comments = comments[len(comments)-limit:]
	}

	ids := make([]uint, len(comments))
	for i, c := range comments {
		ids[i] = c.ID
	}

	var likeCounts map[uint]int64

	var liked map[uint]bool

	if s.likeRepo != nil {
		likeCounts, _ = s.likeRepo.CountByMessageIDs(ids)
		liked, _ = s.likeRepo.LikedMessageIDs(userID, ids)
	}

	out := make([]*WallCommentResponse, len(comments))
	for i, c := range comments {
		out[i] = &WallCommentResponse{
			Message:   c,
			LikeCount: likeCounts[c.ID],
			LikedByMe: liked[c.ID],
		}
	}

	return &WallCommentListResponse{Comments: out, HasMore: hasMore}, nil
}

// ResolvePermalink 解析訊息永久連結：驗證存取權後回傳預覽用的訊息+頻道摘要
func (s *messageService) ResolvePermalink(messageID, viewerID uint) (*PermalinkResponse, error) {
	msg, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	channel, err := s.channelRepo.GetByID(msg.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, msg.ChannelID, viewerID); err != nil {
		return nil, err
	}

	content := msg.Content
	if len(content) > 100 {
		content = strings.ToValidUTF8(content[:100], "")
	}

	return &PermalinkResponse{
		Message: PermalinkMessage{ID: msg.ID, Content: content, Author: msg.User},
		Channel: PermalinkChannel{
			ID:      channel.ID,
			Name:    channel.Name,
			GuildID: channel.GuildID,
			Type:    channel.Type,
		},
	}, nil
}

// MessagesAround 回傳以 aroundID 為中心的訊息視窗（時間順序），供永久連結精準跳轉使用
func (s *messageService) MessagesAround(
	channelID, userID, aroundID uint,
	limit int,
) (*MessageListResponse, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, channelID, userID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	msgs, err := s.messageRepo.GetMessagesAround(channelID, aroundID, limit)
	if err != nil {
		return nil, err
	}

	return &MessageListResponse{Messages: msgs, HasMore: false}, nil
}

// PostsAround 回傳以 aroundID 為中心的貼文視窗（時間順序，含讚數/留言數/我是否讚過）
func (s *messageService) PostsAround(
	channelID, userID, aroundID uint,
	limit int,
) (*PostListResponse, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, channelID, userID); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	posts, err := s.messageRepo.GetPostsAround(channelID, aroundID, limit)
	if err != nil {
		return nil, err
	}

	return &PostListResponse{Posts: s.enrichPosts(posts, userID), HasMore: false}, nil
}

// CreateMessage 建立訊息
func (s *messageService) CreateMessage(
	userID uint,
	req *CreateMessageRequest,
) (*model.Message, error) {
	msgType, err := validateCreateRequest(req)
	if err != nil {
		return nil, err
	}

	existing, found, err := s.findMessageByNonce(userID, req.Nonce)
	if err != nil {
		return nil, err
	}

	if found {
		return existing, nil
	}

	// 檢查頻道是否存在
	channel, err := s.channelRepo.GetByID(req.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.ensureChannelAccess(channel, req.ChannelID, userID); err != nil {
		return nil, err
	}

	// 留言：驗證父貼文存在、同頻道、且父本身是貼文（單層留言）
	if req.ParentID != nil {
		parent, perr := s.messageRepo.GetByID(*req.ParentID)
		if perr != nil {
			return nil, errors.New("parent post not found")
		}

		if parent.ChannelID != req.ChannelID {
			return nil, errors.New("parent post is in a different channel")
		}

		if parent.ParentID != nil {
			return nil, errors.New("cannot comment on a comment")
		}
	}

	if err := s.validateFileAttachments(userID, req.FileIDs); err != nil {
		return nil, err
	}

	// 建立訊息
	message := &model.Message{
		ChannelID: req.ChannelID,
		UserID:    userID,
		Content:   req.Content,
		Type:      msgType,
		Nonce:     req.Nonce,
		ParentID:  req.ParentID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// 建立附件關聯。附件已在建立訊息前驗過，走到這裡代表期間檔案被刪或改狀態，
	// 此時整筆訊息回滾，避免留下沒有附件卻也沒廣播出去的孤兒訊息。
	if err := s.attachFiles(userID, message.ID, req.FileIDs); err != nil {
		_ = s.messageRepo.Delete(message.ID)

		return nil, err
	}

	// 重新取得訊息（包含關聯資料）
	fullMessage, err := s.messageRepo.GetByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 如果有 WebSocket 管理器，即時推送新訊息
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(req.ChannelID, "message_create", fullMessage)
	}

	// 非同步解析 mention 並推送通知（不阻塞訊息回傳）
	if s.mentionRepo != nil {
		go s.processMentions(fullMessage, channel)
	}

	// 非同步翻譯（僅 text 類型）
	if s.translationService != nil && msgType == messageTypeText {
		s.translationService.TranslateAndPush(fullMessage.ID, req.Content, req.ChannelID)
	}

	return fullMessage, nil
}

func validateCreateRequest(req *CreateMessageRequest) (string, error) {
	if req.Content == "" && len(req.FileIDs) == 0 {
		return "", ErrEmptyMessageContent
	}

	if utf8.RuneCountInString(req.Content) > MaxMessageContentLen {
		return "", ErrMessageTooLong
	}

	msgType := req.Type
	if msgType == "" {
		msgType = messageTypeText
	}

	if msgType != messageTypeText && msgType != "image" && msgType != "file" {
		return "", ErrInvalidMessageType
	}

	return msgType, nil
}

func (s *messageService) findMessageByNonce(
	userID uint,
	nonce string,
) (*model.Message, bool, error) {
	if nonce == "" {
		return nil, false, nil
	}

	existing, err := s.messageRepo.GetByNonce(userID, nonce)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if existing == nil {
		return nil, false, nil
	}

	return existing, true, nil
}

func (s *messageService) ensureChannelAccess(channel *model.Channel, channelID, userID uint) error {
	if channel.Type == "dm" {
		ok, err := s.channelRepo.IsDMParticipant(channelID, userID)
		if err != nil || !ok {
			return ErrNotChannelMemberMsg
		}

		return nil
	}

	if channel.GuildID == nil {
		return errors.New("invalid channel: missing guild_id")
	}

	member, err := s.guildMemberRepo.GetMember(*channel.GuildID, userID)
	if err != nil || member == nil {
		return ErrNotChannelMemberMsg
	}

	return nil
}

func (s *messageService) validateFileAttachments(userID uint, fileIDs []uint) error {
	if s.fileService == nil && len(fileIDs) > 0 {
		return ErrFileNotAttachable
	}

	for _, fileID := range fileIDs {
		file, err := s.fileService.GetFile(userID, fileID)
		if err != nil || file.Status != fileStatusActive {
			return ErrFileNotAttachable
		}
	}

	return nil
}

func (s *messageService) attachFiles(userID, messageID uint, fileIDs []uint) error {
	if s.fileService == nil || len(fileIDs) == 0 {
		return nil
	}

	for _, fileID := range fileIDs {
		if _, err := s.fileService.AttachToMessage(userID, messageID, fileID); err != nil {
			// 包成 ErrFileNotAttachable，handler 才會回 403 而不是把
			// ErrFileForbidden／ErrUploadNotConfirmed 落到 default 變成 500。
			return fmt.Errorf("%w: %w", ErrFileNotAttachable, err)
		}
	}

	return nil
}

// GetMessage 取得訊息
func (s *messageService) GetMessage(messageID, userID uint) (*model.Message, error) {
	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// 檢查使用者是否為該社群成員
	channel, err := s.channelRepo.GetByID(message.ChannelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if channel.Type == "dm" {
		ok, err := s.channelRepo.IsDMParticipant(message.ChannelID, userID)
		if err != nil || !ok {
			return nil, ErrNotChannelMemberMsg
		}
	} else {
		if channel.GuildID == nil {
			return nil, errors.New("invalid channel: missing guild_id")
		}

		member, err := s.guildMemberRepo.GetMember(*channel.GuildID, userID)
		if err != nil || member == nil {
			return nil, ErrNotChannelMemberMsg
		}
	}

	return message, nil
}

// ListChannelMessages 列出頻道的訊息（cursor-based 分頁）
func (s *messageService) ListChannelMessages(
	channelID, userID uint,
	limit int,
	before uint,
) (*MessageListResponse, error) {
	// 檢查頻道是否存在
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	// 檢查使用者是否有權限讀取
	if channel.Type == "dm" {
		ok, err := s.channelRepo.IsDMParticipant(channelID, userID)
		if err != nil || !ok {
			return nil, ErrNotChannelMemberMsg
		}
	} else {
		if channel.GuildID == nil {
			return nil, errors.New("invalid channel: missing guild_id")
		}

		member, err := s.guildMemberRepo.GetMember(*channel.GuildID, userID)
		if err != nil || member == nil {
			return nil, ErrNotChannelMemberMsg
		}
	}

	if limit < 1 || limit > 100 {
		limit = 50
	}

	// 多取一筆用於判斷 has_more
	messages, err := s.messageRepo.GetByChannelIDCursor(channelID, before, limit+1)
	if err != nil {
		return nil, err
	}

	// messages 已由 repo 反轉為 ASC（由舊到新）
	// 多取的那筆在 index 0（最舊的那筆），用來判斷是否還有更舊的資料
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[1:] // 捨棄最舊的 sentinel，保留最新的 limit 筆
	}

	return &MessageListResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

// UpdateMessage 更新訊息
func (s *messageService) UpdateMessage(
	messageID, userID uint,
	req *UpdateMessageRequest,
) (*model.Message, error) {
	// 驗證訊息內容
	if req.Content == "" {
		return nil, ErrEmptyMessageContent
	}

	if utf8.RuneCountInString(req.Content) > MaxMessageContentLen {
		return nil, ErrMessageTooLong
	}

	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// 檢查是否為訊息擁有者
	if message.UserID != userID {
		return nil, ErrNotMessageOwner
	}

	// 更新訊息
	message.Content = req.Content
	message.IsEdited = true
	message.UpdatedAt = time.Now()

	if err := s.messageRepo.Update(message); err != nil {
		return nil, err
	}

	// 重新取得訊息（包含關聯資料）
	updated, err := s.messageRepo.GetByID(message.ID)
	if err != nil {
		return nil, err
	}

	// 廣播訊息更新事件
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(message.ChannelID, "message_update", updated)
	}

	return updated, nil
}

// DeleteMessage 刪除訊息
func (s *messageService) DeleteMessage(messageID, userID uint) error {
	// 取得訊息
	message, err := s.messageRepo.GetByID(messageID)
	if err != nil {
		return ErrMessageNotFound
	}

	// 預先載入頻道，供權限檢查與 cascade 判斷共用（只 fetch 一次）
	channel, _ := s.channelRepo.GetByID(message.ChannelID)

	// 檢查是否為訊息擁有者或社群管理員
	if message.UserID != userID {
		// 檢查是否為社群管理員
		if channel == nil {
			return errors.New("channel not found")
		}

		if channel.Type == "dm" {
			// DM 頻道：只有訊息發送者可刪除
			return ErrNotMessageOwner
		}

		if channel.GuildID == nil {
			return errors.New("invalid channel: missing guild_id")
		}

		member, err := s.guildMemberRepo.GetMember(*channel.GuildID, userID)
		if err != nil || member == nil {
			return ErrNotChannelMemberMsg
		}

		// 只有擁有者或管理員可以刪除他人訊息
		if member.Role != roleOwner && member.Role != roleAdmin {
			return ErrNotMessageOwner
		}
	}

	// feed 頻道的貼文（parent_id = nil）→ cascade 刪留言與所有讚
	if channel != nil && channel.Type == channelTypeFeed && message.ParentID == nil {
		if err := s.messageRepo.DeletePostCascade(messageID); err != nil {
			return err
		}
	} else {
		if s.likeRepo != nil {
			_ = s.likeRepo.DeleteByMessageIDs([]uint{messageID})
		}

		if s.reactionRepo != nil {
			_ = s.reactionRepo.DeleteByMessageIDs([]uint{messageID})
		}

		if err := s.messageRepo.Delete(messageID); err != nil {
			return err
		}
	}

	// 廣播訊息刪除事件
	if s.wsManager != nil {
		s.wsManager.BroadcastToChannel(message.ChannelID, "message_delete", map[string]any{
			"message_id": messageID,
			"channel_id": message.ChannelID,
		})
	}

	return nil
}

// processMentions 解析訊息內容中的 @提及，儲存到 DB 並透過 WS 推送 mention_create 事件
// 此函數在 goroutine 中執行，錯誤僅記錄 log，不影響訊息主流程
func (s *messageService) processMentions(msg *model.Message, channel *model.Channel) {
	mentions := s.parseMentions(msg, channel)
	if len(mentions) == 0 {
		return
	}

	if err := s.mentionRepo.BulkCreate(mentions); err != nil {
		return
	}

	if s.wsManager == nil {
		return
	}

	// 對每個被提及的使用者推送 mention_create 事件（去重）
	seen := make(map[uint]bool)

	authorName := msg.User.Nickname
	if authorName == "" {
		authorName = msg.User.Username
	}

	payload := MentionCreatePayload{
		MessageID: msg.ID,
		ChannelID: msg.ChannelID,
		GuildID:   channel.GuildID,
		AuthorID:  msg.UserID,
		Author:    authorName,
		Content:   msg.Content,
	}

	for _, m := range mentions {
		if m.UserID == msg.UserID || seen[m.UserID] {
			continue // 不通知自己，也不重複推送
		}

		seen[m.UserID] = true
		s.wsManager.BroadcastToUser(m.UserID, "mention_create", payload)
	}
}

// parseMentions 從訊息內容中解析出所有被提及的使用者（含 @here / @everyone 展開）
func (s *messageService) parseMentions(
	msg *model.Message,
	channel *model.Channel,
) []*model.MessageMention {
	content := msg.Content

	matches := mentionRe.FindAllStringSubmatch(content, -1)
	hasHere := hereRe.MatchString(content)
	hasEveryone := everyoneRe.MatchString(content)

	if len(matches) == 0 && !hasHere && !hasEveryone {
		return nil
	}

	mentions := make([]*model.MessageMention, 0, 8)
	mentionIndexByUser := make(map[uint]int, 8)

	upsertMention := func(userID uint, mentionType string) {
		if idx, ok := mentionIndexByUser[userID]; ok {
			if mentionTypePriority(mentionType) > mentionTypePriority(mentions[idx].MentionType) {
				mentions[idx].MentionType = mentionType
			}

			return
		}

		mentionIndexByUser[userID] = len(mentions)
		mentions = append(mentions, &model.MessageMention{
			MessageID:   msg.ID,
			UserID:      userID,
			MentionType: mentionType,
		})
	}

	// 解析 <@123> 格式。<@id> 的數字是使用者自由輸入的，沒有這道過濾等於任何人
	// 都能對站上任意 user 推播含任意內容的 mention_create。
	for _, uid := range s.visibleMentionTargets(msg.ChannelID, channel, matches) {
		upsertMention(uid, "user")
	}

	//	@here	/ @everyone 僅限 guild 頻道
	if channel.GuildID == nil || (!hasHere && !hasEveryone) {
		return mentions
	}

	mentionType := "everyone"
	if hasHere {
		mentionType = "here"
	}

	for _, userID := range s.guildRoster(*channel.GuildID) {
		if userID == msg.UserID {
			continue // 跳過發送者自己
		}

		//	@here	只通知在線成員；@everyone 通知所有成員
		if hasHere && s.wsManager != nil && !s.wsManager.IsUserOnline(userID) {
			continue
		}

		upsertMention(userID, mentionType)
	}

	return mentions
}

// visibleMentionTargets 從 <@id> 比對結果篩出「真的看得到這個頻道」的使用者。
// 刻意不整份載入 guild 成員來做比對：GetByGuildID 會 Preload("User") 撈出整張
// 成員表，掛在每一則含 mention 的訊息上等於把 mention spam 變成 DB 壓力測試。
// 內容本身已有 MaxMessageContentLen 上限，這裡再限 maxMentionsPerMessage 個
// 相異 ID，查詢次數就有上界。
func (s *messageService) visibleMentionTargets(
	channelID uint,
	channel *model.Channel,
	matches [][]string,
) []uint {
	ids := make([]uint, 0, len(matches))
	seen := make(map[uint]bool, len(matches))

	for _, m := range matches {
		uid, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil || seen[uint(uid)] {
			continue
		}

		seen[uint(uid)] = true

		ids = append(ids, uint(uid))
		if len(ids) >= maxMentionsPerMessage {
			break
		}
	}

	if len(ids) == 0 {
		return nil
	}

	if channel.Type == "dm" {
		participants, err := s.channelRepo.GetDMParticipants(channelID)
		if err != nil {
			return nil
		}

		inDM := make(map[uint]bool, len(participants))
		for _, p := range participants {
			inDM[p.UserID] = true
		}

		return filterUint(ids, func(uid uint) bool { return inDM[uid] })
	}

	if channel.GuildID == nil {
		return nil
	}

	return filterUint(ids, func(uid uint) bool {
		member, err := s.guildMemberRepo.GetMember(*channel.GuildID, uid)

		return err == nil && member != nil
	})
}

// guildRoster 回傳 guild 全體成員 ID（僅 @here / @everyone 展開時才需要）。
func (s *messageService) guildRoster(guildID uint) []uint {
	members, err := s.guildMemberRepo.GetByGuildID(guildID)
	if err != nil {
		return nil
	}

	ids := make([]uint, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}

	return ids
}

func filterUint(ids []uint, keep func(uint) bool) []uint {
	out := ids[:0]

	for _, id := range ids {
		if keep(id) {
			out = append(out, id)
		}
	}

	return out
}

func mentionTypePriority(mentionType string) int {
	switch mentionType {
	case "user":
		return 3
	case "here":
		return 2
	default:
		return 1
	}
}
