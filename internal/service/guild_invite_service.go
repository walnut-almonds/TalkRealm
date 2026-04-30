package service

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteExpired  = errors.New("invite has expired")
	ErrInviteMaxUses  = errors.New("invite has reached max uses")
)

// CreateInviteRequest 建立邀請碼請求
type CreateInviteRequest struct {
	MaxUses   int `json:"max_uses"`   // 0 = unlimited
	ExpiresIn int `json:"expires_in"` // 秒數，0 = 永不過期
}

// GuildInviteService 社群邀請碼服務介面
type GuildInviteService interface {
	CreateInvite(guildID, creatorID uint, req *CreateInviteRequest) (*model.GuildInvite, error)
	GetInviteByCode(code string) (*model.GuildInvite, error)
	JoinByInvite(code string, userID uint) error
}

type guildInviteService struct {
	inviteRepo      repository.GuildInviteRepository
	guildRepo       repository.GuildRepository
	guildMemberRepo repository.GuildMemberRepository
}

// NewGuildInviteService 建立社群邀請碼服務
func NewGuildInviteService(
	inviteRepo repository.GuildInviteRepository,
	guildRepo repository.GuildRepository,
	guildMemberRepo repository.GuildMemberRepository,
) GuildInviteService {
	return &guildInviteService{
		inviteRepo:      inviteRepo,
		guildRepo:       guildRepo,
		guildMemberRepo: guildMemberRepo,
	}
}

// generateInviteCode 生成 8 字元邀請碼（Base32 大寫）
func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return strings.ToUpper(base32.StdEncoding.EncodeToString(b))[:8], nil
}

// CreateInvite 建立邀請碼
func (s *guildInviteService) CreateInvite(
	guildID, creatorID uint,
	req *CreateInviteRequest,
) (*model.GuildInvite, error) {
	// 確認 guild 存在
	_, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	// 確認 creator 是社群成員
	member, err := s.guildMemberRepo.GetMember(guildID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotGuildMember
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invite := &model.GuildInvite{
		GuildID:   guildID,
		CreatorID: creatorID,
		Code:      code,
		MaxUses:   req.MaxUses,
		Uses:      0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		invite.ExpiresAt = &t
	}

	if err := s.inviteRepo.Create(invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// GetInviteByCode 透過邀請碼取得邀請資訊（同時驗證有效性）
func (s *guildInviteService) GetInviteByCode(code string) (*model.GuildInvite, error) {
	invite, err := s.inviteRepo.GetByCode(code)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	// 檢查是否過期
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	// 檢查是否達到使用上限
	if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
		return nil, ErrInviteMaxUses
	}

	return invite, nil
}

// JoinByInvite 透過邀請碼加入社群
func (s *guildInviteService) JoinByInvite(code string, userID uint) error {
	invite, err := s.GetInviteByCode(code)
	if err != nil {
		return err
	}

	// 檢查是否已是成員
	existingMember, _ := s.guildMemberRepo.GetMember(invite.GuildID, userID)
	if existingMember != nil {
		return ErrAlreadyInGuild
	}

	// 加入社群
	member := &model.GuildMember{
		GuildID:   invite.GuildID,
		UserID:    userID,
		Role:      "member",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.guildMemberRepo.Create(member); err != nil {
		return err
	}

	// 增加邀請碼使用次數
	return s.inviteRepo.IncrementUses(invite.ID)
}
