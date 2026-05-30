package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"gorm.io/gorm"
)

const (
	messageTypeText = "text"
	roleOwner       = "owner"
	roleAdmin       = "admin"
)

var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrNotChannelMemberMsg = errors.New("not a member of this channel's guild")
	ErrNotMessageOwner     = errors.New("not the owner of this message")
	ErrEmptyMessageContent = errors.New("message content cannot be empty")
	ErrInvalidMessageType  = errors.New("invalid message type")
	ErrDuplicateNonce      = errors.New("duplicate nonce: message already exists")
)

// WebSocketManager 定義 WebSocket 管理器的介面（避免循環依賴）
type WebSocketManager interface {
	BroadcastToChannel(channelID uint, msgType string, data any)
	BroadcastToUser(userID uint, msgType string, data any)
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

// mentionRe 解析 <@123> 格式的 user mention
var mentionRe = regexp.MustCompile(`<@(\d+)>`)

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
	Type      string `json:"type"`     // text, image, file (預設: text)
	Nonce     string `json:"nonce"`    // client 產生的冪等 key（可選，建議 UUID v4）
	FileIDs   []uint `json:"file_ids"` // 附加的已確認檔案 ID（可選）
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

	// 建立訊息
	message := &model.Message{
		ChannelID: req.ChannelID,
		UserID:    userID,
		Content:   req.Content,
		Type:      msgType,
		Nonce:     req.Nonce,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}

	// 建立附件關聯
	s.attachFiles(message.ID, req.FileIDs)

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

func (s *messageService) attachFiles(messageID uint, fileIDs []uint) {
	if s.fileService == nil || len(fileIDs) == 0 {
		return
	}

	for _, fid := range fileIDs {
		if _, err := s.fileService.AttachToMessage(messageID, fid); err != nil {
			_ = err
		}
	}
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

	// 檢查是否為訊息擁有者或社群管理員
	if message.UserID != userID {
		// 檢查是否為社群管理員
		channel, err := s.channelRepo.GetByID(message.ChannelID)
		if err != nil {
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

	// 刪除訊息
	if err := s.messageRepo.Delete(messageID); err != nil {
		return err
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

	var mentions []*model.MessageMention

	// 解析 <@123> 格式
	matches := mentionRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		uid, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			continue
		}

		mentions = append(mentions, &model.MessageMention{
			MessageID:   msg.ID,
			UserID:      uint(uid),
			MentionType: "user",
		})
	}

	// 解析 @here 和 @everyone（僅 guild 頻道）
	if channel.GuildID == nil {
		return mentions
	}

	hasHere := containsMentionKeyword(content, "here")
	hasEveryone := containsMentionKeyword(content, "everyone")

	if !hasHere && !hasEveryone {
		return mentions
	}

	// 取得 guild 所有成員
	members, err := s.guildMemberRepo.GetByGuildID(*channel.GuildID)
	if err != nil {
		return mentions
	}

	mentionType := "everyone"
	if hasHere {
		mentionType = "here"
	}

	for _, member := range members {
		if member.UserID == msg.UserID {
			continue // 跳過發送者自己
		}

		mentions = append(mentions, &model.MessageMention{
			MessageID:   msg.ID,
			UserID:      member.UserID,
			MentionType: mentionType,
		})
	}

	return mentions
}

// containsMentionKeyword 檢查內容中是否包含 @keyword（不區分大小寫，詞邊界匹配）
func containsMentionKeyword(content, keyword string) bool {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)@%s\b`, keyword))
	return re.MatchString(content)
}
