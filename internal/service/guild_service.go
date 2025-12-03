package service

import (
	"errors"
	"time"

	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
)

var (
	ErrGuildNotFound      = errors.New("guild not found")
	ErrNotGuildOwner      = errors.New("not guild owner")
	ErrAlreadyInGuild     = errors.New("already in guild")
	ErrNotGuildMember     = errors.New("not guild member")
	ErrCannotLeaveAsOwner = errors.New("owner cannot leave guild, transfer ownership first")
)

// CreateGuildRequest 建立社群請求
type CreateGuildRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
	Icon        string `json:"icon" binding:"max=256"`
}

// UpdateGuildRequest 更新社群請求
type UpdateGuildRequest struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
	Icon        string `json:"icon" binding:"max=256"`
}

// GuildService 社群服務介面
type GuildService interface {
	CreateGuild(ownerID uint, req *CreateGuildRequest) (*model.Guild, error)
	GetGuild(guildID uint) (*model.Guild, error)
	ListUserGuilds(userID uint) ([]*model.Guild, error)
	UpdateGuild(guildID, userID uint, req *UpdateGuildRequest) (*model.Guild, error)
	DeleteGuild(guildID, userID uint) error
	IsGuildOwner(guildID, userID uint) (bool, error)
	IsGuildMember(guildID, userID uint) (bool, error)
}

type guildService struct {
	guildRepo       repository.GuildRepository
	guildMemberRepo repository.GuildMemberRepository
}

// NewGuildService 建立社群服務
func NewGuildService(guildRepo repository.GuildRepository, guildMemberRepo repository.GuildMemberRepository) GuildService {
	return &guildService{
		guildRepo:       guildRepo,
		guildMemberRepo: guildMemberRepo,
	}
}

// CreateGuild 建立社群
func (s *guildService) CreateGuild(ownerID uint, req *CreateGuildRequest) (*model.Guild, error) {
	guild := &model.Guild{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.guildRepo.Create(guild); err != nil {
		return nil, err
	}

	// 自動將擁有者加入為成員
	member := &model.GuildMember{
		GuildID:   guild.ID,
		UserID:    ownerID,
		Role:      "owner",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.guildMemberRepo.Create(member); err != nil {
		// 如果加入成員失敗，刪除已建立的社群
		_ = s.guildRepo.Delete(guild.ID)
		return nil, err
	}

	return guild, nil
}

// GetGuild 取得社群詳情
func (s *guildService) GetGuild(guildID uint) (*model.Guild, error) {
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}
	return guild, nil
}

// ListUserGuilds 列出使用者所屬的所有社群
func (s *guildService) ListUserGuilds(userID uint) ([]*model.Guild, error) {
	return s.guildRepo.GetMemberGuilds(userID, 0, 100)
}

// UpdateGuild 更新社群資訊
func (s *guildService) UpdateGuild(guildID, userID uint, req *UpdateGuildRequest) (*model.Guild, error) {
	// 檢查是否為擁有者
	isOwner, err := s.IsGuildOwner(guildID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, ErrNotGuildOwner
	}

	// 取得社群
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	// 更新欄位
	if req.Name != "" {
		guild.Name = req.Name
	}
	if req.Description != "" {
		guild.Description = req.Description
	}
	if req.Icon != "" {
		guild.Icon = req.Icon
	}
	guild.UpdatedAt = time.Now()

	if err := s.guildRepo.Update(guild); err != nil {
		return nil, err
	}

	return guild, nil
}

// DeleteGuild 刪除社群
func (s *guildService) DeleteGuild(guildID, userID uint) error {
	// 檢查是否為擁有者
	isOwner, err := s.IsGuildOwner(guildID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotGuildOwner
	}

	// 刪除社群（會級聯刪除成員、頻道等）
	return s.guildRepo.Delete(guildID)
}

// IsGuildOwner 檢查是否為社群擁有者
func (s *guildService) IsGuildOwner(guildID, userID uint) (bool, error) {
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return false, ErrGuildNotFound
	}
	return guild.OwnerID == userID, nil
}

// IsGuildMember 檢查是否為社群成員
func (s *guildService) IsGuildMember(guildID, userID uint) (bool, error) {
	member, err := s.guildMemberRepo.GetMember(guildID, userID)
	if err != nil {
		return false, nil
	}
	return member != nil, nil
}

// GuildMemberService 社群成員服務介面
type GuildMemberService interface {
	JoinGuild(guildID, userID uint) error
	LeaveGuild(guildID, userID uint) error
	KickMember(guildID, targetUserID, operatorUserID uint) error
	ListGuildMembers(guildID uint) ([]*model.GuildMember, error)
	GetMember(guildID, userID uint) (*model.GuildMember, error)
	UpdateMemberRole(guildID, targetUserID, operatorUserID uint, role string) error
}

type guildMemberService struct {
	guildRepo       repository.GuildRepository
	guildMemberRepo repository.GuildMemberRepository
}

// NewGuildMemberService 建立社群成員服務
func NewGuildMemberService(guildRepo repository.GuildRepository, guildMemberRepo repository.GuildMemberRepository) GuildMemberService {
	return &guildMemberService{
		guildRepo:       guildRepo,
		guildMemberRepo: guildMemberRepo,
	}
}

// JoinGuild 加入社群
func (s *guildMemberService) JoinGuild(guildID, userID uint) error {
	// 檢查社群是否存在
	_, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return ErrGuildNotFound
	}

	// 檢查是否已是成員
	existingMember, _ := s.guildMemberRepo.GetMember(guildID, userID)
	if existingMember != nil {
		return ErrAlreadyInGuild
	}

	// 加入社群
	member := &model.GuildMember{
		GuildID:   guildID,
		UserID:    userID,
		Role:      "member",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.guildMemberRepo.Create(member)
}

// LeaveGuild 離開社群
func (s *guildMemberService) LeaveGuild(guildID, userID uint) error {
	// 檢查社群是否存在
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return ErrGuildNotFound
	}

	// 擁有者不能離開，需要先轉移所有權
	if guild.OwnerID == userID {
		return ErrCannotLeaveAsOwner
	}

	// 檢查是否為成員
	member, err := s.guildMemberRepo.GetMember(guildID, userID)
	if err != nil || member == nil {
		return ErrNotGuildMember
	}

	return s.guildMemberRepo.Delete(member.ID)
}

// KickMember 踢出成員
func (s *guildMemberService) KickMember(guildID, targetUserID, operatorUserID uint) error {
	// 檢查操作者是否為擁有者
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return ErrGuildNotFound
	}

	if guild.OwnerID != operatorUserID {
		return ErrNotGuildOwner
	}

	// 不能踢出自己
	if targetUserID == operatorUserID {
		return errors.New("cannot kick yourself")
	}

	// 檢查目標是否為成員
	member, err := s.guildMemberRepo.GetMember(guildID, targetUserID)
	if err != nil || member == nil {
		return ErrNotGuildMember
	}

	return s.guildMemberRepo.Delete(member.ID)
}

// ListGuildMembers 列出社群成員
func (s *guildMemberService) ListGuildMembers(guildID uint) ([]*model.GuildMember, error) {
	// 檢查社群是否存在
	_, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return nil, ErrGuildNotFound
	}

	return s.guildMemberRepo.GetByGuildID(guildID)
}

// GetMember 取得特定成員資訊
func (s *guildMemberService) GetMember(guildID, userID uint) (*model.GuildMember, error) {
	member, err := s.guildMemberRepo.GetMember(guildID, userID)
	if err != nil || member == nil {
		return nil, ErrNotGuildMember
	}
	return member, nil
}

// UpdateMemberRole 更新成員角色
func (s *guildMemberService) UpdateMemberRole(guildID, targetUserID, operatorUserID uint, role string) error {
	// 檢查操作者是否為擁有者
	guild, err := s.guildRepo.GetByID(guildID)
	if err != nil {
		return ErrGuildNotFound
	}

	if guild.OwnerID != operatorUserID {
		return ErrNotGuildOwner
	}

	// 不能修改自己的角色
	if targetUserID == operatorUserID {
		return errors.New("cannot modify your own role")
	}

	// 取得目標成員
	member, err := s.guildMemberRepo.GetMember(guildID, targetUserID)
	if err != nil || member == nil {
		return ErrNotGuildMember
	}

	// 更新角色
	member.Role = role
	member.UpdatedAt = time.Now()

	return s.guildMemberRepo.Update(member)
}
