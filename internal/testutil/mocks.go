package testutil

import (
	"github.com/walnut-almonds/talkrealm/internal/model"
	"github.com/walnut-almonds/talkrealm/internal/repository"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// ---------------------------------------------------------------------------
// MockUserRepository
// ---------------------------------------------------------------------------

// MockUserRepository is a test double for repository.UserRepository.
type MockUserRepository struct {
	CreateFn        func(user *model.User) error
	GetByIDFn       func(id uint) (*model.User, error)
	GetByEmailFn    func(email string) (*model.User, error)
	GetByUsernameFn func(username string) (*model.User, error)
	UpdateFn        func(user *model.User) error
	DeleteFn        func(id uint) error
	ListFn          func(offset, limit int) ([]*model.User, error)
	UpdateStatusFn  func(id uint, status string) error
}

var _ repository.UserRepository = (*MockUserRepository)(nil)

func (m *MockUserRepository) Create(user *model.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(user)
	}

	return nil
}

func (m *MockUserRepository) GetByID(id uint) (*model.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockUserRepository) GetByEmail(email string) (*model.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(email)
	}

	return nil, nil
}

func (m *MockUserRepository) GetByUsername(username string) (*model.User, error) {
	if m.GetByUsernameFn != nil {
		return m.GetByUsernameFn(username)
	}

	return nil, nil
}

func (m *MockUserRepository) Update(user *model.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(user)
	}

	return nil
}

func (m *MockUserRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockUserRepository) List(offset, limit int) ([]*model.User, error) {
	if m.ListFn != nil {
		return m.ListFn(offset, limit)
	}

	return nil, nil
}

func (m *MockUserRepository) UpdateStatus(id uint, status string) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(id, status)
	}

	return nil
}

// ---------------------------------------------------------------------------
// MockRefreshTokenRepository
// ---------------------------------------------------------------------------

// MockRefreshTokenRepository is a test double for repository.RefreshTokenRepository.
type MockRefreshTokenRepository struct {
	CreateFn            func(token *model.RefreshToken) error
	GetByTokenFn        func(token string) (*model.RefreshToken, error)
	RevokeByTokenFn     func(token string) error
	RevokeAllByUserIDFn func(userID uint) error
	DeleteExpiredFn     func() error
}

var _ repository.RefreshTokenRepository = (*MockRefreshTokenRepository)(nil)

func (m *MockRefreshTokenRepository) Create(token *model.RefreshToken) error {
	if m.CreateFn != nil {
		return m.CreateFn(token)
	}

	return nil
}

func (m *MockRefreshTokenRepository) GetByToken(token string) (*model.RefreshToken, error) {
	if m.GetByTokenFn != nil {
		return m.GetByTokenFn(token)
	}

	return nil, nil
}

func (m *MockRefreshTokenRepository) RevokeByToken(token string) error {
	if m.RevokeByTokenFn != nil {
		return m.RevokeByTokenFn(token)
	}

	return nil
}

func (m *MockRefreshTokenRepository) RevokeAllByUserID(userID uint) error {
	if m.RevokeAllByUserIDFn != nil {
		return m.RevokeAllByUserIDFn(userID)
	}

	return nil
}

func (m *MockRefreshTokenRepository) DeleteExpired() error {
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn()
	}

	return nil
}

// ---------------------------------------------------------------------------
// MockGuildRepository
// ---------------------------------------------------------------------------

// MockGuildRepository is a test double for repository.GuildRepository.
type MockGuildRepository struct {
	CreateFn          func(guild *model.Guild) error
	GetByIDFn         func(id uint) (*model.Guild, error)
	UpdateFn          func(guild *model.Guild) error
	DeleteFn          func(id uint) error
	ListFn            func(offset, limit int) ([]*model.Guild, error)
	GetByOwnerIDFn    func(ownerID uint) ([]*model.Guild, error)
	GetMemberGuildsFn func(userID uint, offset, limit int) ([]*model.Guild, error)
}

var _ repository.GuildRepository = (*MockGuildRepository)(nil)

func (m *MockGuildRepository) Create(guild *model.Guild) error {
	if m.CreateFn != nil {
		return m.CreateFn(guild)
	}

	return nil
}

func (m *MockGuildRepository) GetByID(id uint) (*model.Guild, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockGuildRepository) Update(guild *model.Guild) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(guild)
	}

	return nil
}

func (m *MockGuildRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockGuildRepository) List(offset, limit int) ([]*model.Guild, error) {
	if m.ListFn != nil {
		return m.ListFn(offset, limit)
	}

	return nil, nil
}

func (m *MockGuildRepository) GetByOwnerID(ownerID uint) ([]*model.Guild, error) {
	if m.GetByOwnerIDFn != nil {
		return m.GetByOwnerIDFn(ownerID)
	}

	return nil, nil
}

func (m *MockGuildRepository) GetMemberGuilds(userID uint, offset, limit int) ([]*model.Guild, error) {
	if m.GetMemberGuildsFn != nil {
		return m.GetMemberGuildsFn(userID, offset, limit)
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// MockGuildMemberRepository
// ---------------------------------------------------------------------------

// MockGuildMemberRepository is a test double for repository.GuildMemberRepository.
type MockGuildMemberRepository struct {
	CreateFn          func(member *model.GuildMember) error
	GetByIDFn         func(id uint) (*model.GuildMember, error)
	UpdateFn          func(member *model.GuildMember) error
	DeleteFn          func(id uint) error
	GetByGuildIDFn    func(guildID uint) ([]*model.GuildMember, error)
	GetByUserIDFn     func(userID uint) ([]*model.GuildMember, error)
	GetMemberFn       func(guildID, userID uint) (*model.GuildMember, error)
	IsMemberFn        func(guildID, userID uint) (bool, error)
	GetUserGuildIDsFn func(userID uint) ([]uint, error)
}

var _ repository.GuildMemberRepository = (*MockGuildMemberRepository)(nil)

func (m *MockGuildMemberRepository) Create(member *model.GuildMember) error {
	if m.CreateFn != nil {
		return m.CreateFn(member)
	}

	return nil
}

func (m *MockGuildMemberRepository) GetByID(id uint) (*model.GuildMember, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockGuildMemberRepository) Update(member *model.GuildMember) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(member)
	}

	return nil
}

func (m *MockGuildMemberRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockGuildMemberRepository) GetByGuildID(guildID uint) ([]*model.GuildMember, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(guildID)
	}

	return nil, nil
}

func (m *MockGuildMemberRepository) GetByUserID(userID uint) ([]*model.GuildMember, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(userID)
	}

	return nil, nil
}

func (m *MockGuildMemberRepository) GetMember(guildID, userID uint) (*model.GuildMember, error) {
	if m.GetMemberFn != nil {
		return m.GetMemberFn(guildID, userID)
	}

	return nil, nil
}

func (m *MockGuildMemberRepository) IsMember(guildID, userID uint) (bool, error) {
	if m.IsMemberFn != nil {
		return m.IsMemberFn(guildID, userID)
	}

	return false, nil
}

func (m *MockGuildMemberRepository) GetUserGuildIDs(userID uint) ([]uint, error) {
	if m.GetUserGuildIDsFn != nil {
		return m.GetUserGuildIDsFn(userID)
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// MockMessageRepository
// ---------------------------------------------------------------------------

// MockMessageRepository is a test double for repository.MessageRepository.
type MockMessageRepository struct {
	CreateFn               func(message *model.Message) error
	GetByIDFn              func(id uint) (*model.Message, error)
	UpdateFn               func(message *model.Message) error
	DeleteFn               func(id uint) error
	GetByChannelIDFn       func(channelID uint, offset, limit int) ([]*model.Message, error)
	GetByChannelIDCursorFn func(channelID uint, before uint, limit int) ([]*model.Message, error)
	GetByUserIDFn          func(userID uint, offset, limit int) ([]*model.Message, error)
}

var _ repository.MessageRepository = (*MockMessageRepository)(nil)

func (m *MockMessageRepository) Create(message *model.Message) error {
	if m.CreateFn != nil {
		return m.CreateFn(message)
	}

	return nil
}

func (m *MockMessageRepository) GetByID(id uint) (*model.Message, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockMessageRepository) Update(message *model.Message) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(message)
	}

	return nil
}

func (m *MockMessageRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockMessageRepository) GetByChannelID(channelID uint, offset, limit int) ([]*model.Message, error) {
	if m.GetByChannelIDFn != nil {
		return m.GetByChannelIDFn(channelID, offset, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetByChannelIDCursor(channelID uint, before uint, limit int) ([]*model.Message, error) {
	if m.GetByChannelIDCursorFn != nil {
		return m.GetByChannelIDCursorFn(channelID, before, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetByUserID(userID uint, offset, limit int) ([]*model.Message, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(userID, offset, limit)
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// MockChannelRepository
// ---------------------------------------------------------------------------

// MockChannelRepository is a test double for repository.ChannelRepository.
type MockChannelRepository struct {
	CreateFn       func(channel *model.Channel) error
	GetByIDFn      func(id uint) (*model.Channel, error)
	UpdateFn       func(channel *model.Channel) error
	DeleteFn       func(id uint) error
	GetByGuildIDFn func(guildID uint) ([]*model.Channel, error)
	GetByTypeFn    func(guildID uint, channelType string) ([]*model.Channel, error)
}

var _ repository.ChannelRepository = (*MockChannelRepository)(nil)

func (m *MockChannelRepository) Create(channel *model.Channel) error {
	if m.CreateFn != nil {
		return m.CreateFn(channel)
	}

	return nil
}

func (m *MockChannelRepository) GetByID(id uint) (*model.Channel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockChannelRepository) Update(channel *model.Channel) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(channel)
	}

	return nil
}

func (m *MockChannelRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockChannelRepository) GetByGuildID(guildID uint) ([]*model.Channel, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(guildID)
	}

	return nil, nil
}

func (m *MockChannelRepository) GetByType(guildID uint, channelType string) ([]*model.Channel, error) {
	if m.GetByTypeFn != nil {
		return m.GetByTypeFn(guildID, channelType)
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// MockGuildInviteRepository
// ---------------------------------------------------------------------------

// MockGuildInviteRepository is a test double for repository.GuildInviteRepository.
type MockGuildInviteRepository struct {
	CreateFn         func(invite *model.GuildInvite) error
	GetByCodeFn      func(code string) (*model.GuildInvite, error)
	ListByGuildIDFn  func(guildID uint) ([]*model.GuildInvite, error)
	IncrementUsesFn  func(id uint) error
	DeleteFn         func(id uint) error
}

var _ repository.GuildInviteRepository = (*MockGuildInviteRepository)(nil)

func (m *MockGuildInviteRepository) Create(invite *model.GuildInvite) error {
	if m.CreateFn != nil {
		return m.CreateFn(invite)
	}

	return nil
}

func (m *MockGuildInviteRepository) GetByCode(code string) (*model.GuildInvite, error) {
	if m.GetByCodeFn != nil {
		return m.GetByCodeFn(code)
	}

	return nil, nil
}

func (m *MockGuildInviteRepository) ListByGuildID(guildID uint) ([]*model.GuildInvite, error) {
	if m.ListByGuildIDFn != nil {
		return m.ListByGuildIDFn(guildID)
	}

	return nil, nil
}

func (m *MockGuildInviteRepository) IncrementUses(id uint) error {
	if m.IncrementUsesFn != nil {
		return m.IncrementUsesFn(id)
	}

	return nil
}

func (m *MockGuildInviteRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Mock Service interfaces (for handler tests)
// ---------------------------------------------------------------------------

// MockUserService is a test double for service.UserService.
type MockUserService struct {
	RegisterFn           func(req *service.RegisterRequest) (*model.User, error)
	LoginFn              func(req *service.LoginRequest) (*service.LoginResponse, error)
	GetByIDFn            func(id uint) (*model.User, error)
	GetPublicByIDFn      func(id uint) (*service.PublicUser, error)
	UpdateFn             func(id uint, req *service.UpdateUserRequest) (*model.User, error)
	UpdateStatusFn       func(id uint, status string) error
	RefreshAccessTokenFn func(refreshToken string) (*service.LoginResponse, error)
	RevokeRefreshTokenFn func(refreshToken string) error
}

var _ service.UserService = (*MockUserService)(nil)

func (m *MockUserService) Register(req *service.RegisterRequest) (*model.User, error) {
	if m.RegisterFn != nil {
		return m.RegisterFn(req)
	}

	return nil, nil
}

func (m *MockUserService) Login(req *service.LoginRequest) (*service.LoginResponse, error) {
	if m.LoginFn != nil {
		return m.LoginFn(req)
	}

	return nil, nil
}

func (m *MockUserService) GetByID(id uint) (*model.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockUserService) GetPublicByID(id uint) (*service.PublicUser, error) {
	if m.GetPublicByIDFn != nil {
		return m.GetPublicByIDFn(id)
	}

	return nil, nil
}

func (m *MockUserService) Update(id uint, req *service.UpdateUserRequest) (*model.User, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(id, req)
	}

	return nil, nil
}

func (m *MockUserService) UpdateStatus(id uint, status string) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(id, status)
	}

	return nil
}

func (m *MockUserService) RefreshAccessToken(refreshToken string) (*service.LoginResponse, error) {
	if m.RefreshAccessTokenFn != nil {
		return m.RefreshAccessTokenFn(refreshToken)
	}

	return nil, nil
}

func (m *MockUserService) RevokeRefreshToken(refreshToken string) error {
	if m.RevokeRefreshTokenFn != nil {
		return m.RevokeRefreshTokenFn(refreshToken)
	}

	return nil
}

// MockGuildService is a test double for service.GuildService.
type MockGuildService struct {
	CreateGuildFn    func(ownerID uint, req *service.CreateGuildRequest) (*model.Guild, error)
	GetGuildFn       func(guildID uint) (*model.Guild, error)
	ListUserGuildsFn func(userID uint) ([]*model.Guild, error)
	UpdateGuildFn    func(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error)
	DeleteGuildFn    func(guildID, userID uint) error
	IsGuildOwnerFn   func(guildID, userID uint) (bool, error)
	IsGuildMemberFn  func(guildID, userID uint) (bool, error)
}

var _ service.GuildService = (*MockGuildService)(nil)

func (m *MockGuildService) CreateGuild(ownerID uint, req *service.CreateGuildRequest) (*model.Guild, error) {
	if m.CreateGuildFn != nil {
		return m.CreateGuildFn(ownerID, req)
	}

	return nil, nil
}

func (m *MockGuildService) GetGuild(guildID uint) (*model.Guild, error) {
	if m.GetGuildFn != nil {
		return m.GetGuildFn(guildID)
	}

	return nil, nil
}

func (m *MockGuildService) ListUserGuilds(userID uint) ([]*model.Guild, error) {
	if m.ListUserGuildsFn != nil {
		return m.ListUserGuildsFn(userID)
	}

	return nil, nil
}

func (m *MockGuildService) UpdateGuild(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error) {
	if m.UpdateGuildFn != nil {
		return m.UpdateGuildFn(guildID, userID, req)
	}

	return nil, nil
}

func (m *MockGuildService) DeleteGuild(guildID, userID uint) error {
	if m.DeleteGuildFn != nil {
		return m.DeleteGuildFn(guildID, userID)
	}

	return nil
}

func (m *MockGuildService) IsGuildOwner(guildID, userID uint) (bool, error) {
	if m.IsGuildOwnerFn != nil {
		return m.IsGuildOwnerFn(guildID, userID)
	}

	return false, nil
}

func (m *MockGuildService) IsGuildMember(guildID, userID uint) (bool, error) {
	if m.IsGuildMemberFn != nil {
		return m.IsGuildMemberFn(guildID, userID)
	}

	return false, nil
}

// MockGuildMemberService is a test double for service.GuildMemberService.
type MockGuildMemberService struct {
	JoinGuildFn        func(guildID, userID uint) error
	LeaveGuildFn       func(guildID, userID uint) error
	KickMemberFn       func(guildID, targetUserID, operatorUserID uint) error
	ListGuildMembersFn func(guildID uint) ([]*model.GuildMember, error)
	GetMemberFn        func(guildID, userID uint) (*model.GuildMember, error)
	UpdateMemberRoleFn func(guildID, targetUserID, operatorUserID uint, role string) error
}

var _ service.GuildMemberService = (*MockGuildMemberService)(nil)

func (m *MockGuildMemberService) JoinGuild(guildID, userID uint) error {
	if m.JoinGuildFn != nil {
		return m.JoinGuildFn(guildID, userID)
	}

	return nil
}

func (m *MockGuildMemberService) LeaveGuild(guildID, userID uint) error {
	if m.LeaveGuildFn != nil {
		return m.LeaveGuildFn(guildID, userID)
	}

	return nil
}

func (m *MockGuildMemberService) KickMember(guildID, targetUserID, operatorUserID uint) error {
	if m.KickMemberFn != nil {
		return m.KickMemberFn(guildID, targetUserID, operatorUserID)
	}

	return nil
}

func (m *MockGuildMemberService) ListGuildMembers(guildID uint) ([]*model.GuildMember, error) {
	if m.ListGuildMembersFn != nil {
		return m.ListGuildMembersFn(guildID)
	}

	return nil, nil
}

func (m *MockGuildMemberService) GetMember(guildID, userID uint) (*model.GuildMember, error) {
	if m.GetMemberFn != nil {
		return m.GetMemberFn(guildID, userID)
	}

	return nil, nil
}

func (m *MockGuildMemberService) UpdateMemberRole(guildID, targetUserID, operatorUserID uint, role string) error {
	if m.UpdateMemberRoleFn != nil {
		return m.UpdateMemberRoleFn(guildID, targetUserID, operatorUserID, role)
	}

	return nil
}

// MockGuildInviteService is a test double for service.GuildInviteService.
type MockGuildInviteService struct {
	CreateInviteFn    func(guildID, creatorID uint, req *service.CreateInviteRequest) (*model.GuildInvite, error)
	GetInviteByCodeFn func(code string) (*model.GuildInvite, error)
	JoinByInviteFn    func(code string, userID uint) error
}

var _ service.GuildInviteService = (*MockGuildInviteService)(nil)

func (m *MockGuildInviteService) CreateInvite(guildID, creatorID uint, req *service.CreateInviteRequest) (*model.GuildInvite, error) {
	if m.CreateInviteFn != nil {
		return m.CreateInviteFn(guildID, creatorID, req)
	}

	return nil, nil
}

func (m *MockGuildInviteService) GetInviteByCode(code string) (*model.GuildInvite, error) {
	if m.GetInviteByCodeFn != nil {
		return m.GetInviteByCodeFn(code)
	}

	return nil, nil
}

func (m *MockGuildInviteService) JoinByInvite(code string, userID uint) error {
	if m.JoinByInviteFn != nil {
		return m.JoinByInviteFn(code, userID)
	}

	return nil
}

// MockMessageService is a test double for service.MessageService.
type MockMessageService struct {
	CreateMessageFn       func(userID uint, req *service.CreateMessageRequest) (*model.Message, error)
	GetMessageFn          func(messageID, userID uint) (*model.Message, error)
	ListChannelMessagesFn func(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error)
	UpdateMessageFn       func(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error)
	DeleteMessageFn       func(messageID, userID uint) error
	SetWebSocketManagerFn func(manager service.WebSocketManager)
}

var _ service.MessageService = (*MockMessageService)(nil)

func (m *MockMessageService) CreateMessage(userID uint, req *service.CreateMessageRequest) (*model.Message, error) {
	if m.CreateMessageFn != nil {
		return m.CreateMessageFn(userID, req)
	}

	return nil, nil
}

func (m *MockMessageService) GetMessage(messageID, userID uint) (*model.Message, error) {
	if m.GetMessageFn != nil {
		return m.GetMessageFn(messageID, userID)
	}

	return nil, nil
}

func (m *MockMessageService) ListChannelMessages(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error) {
	if m.ListChannelMessagesFn != nil {
		return m.ListChannelMessagesFn(channelID, userID, limit, before)
	}

	return nil, nil
}

func (m *MockMessageService) UpdateMessage(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error) {
	if m.UpdateMessageFn != nil {
		return m.UpdateMessageFn(messageID, userID, req)
	}

	return nil, nil
}

func (m *MockMessageService) DeleteMessage(messageID, userID uint) error {
	if m.DeleteMessageFn != nil {
		return m.DeleteMessageFn(messageID, userID)
	}

	return nil
}

func (m *MockMessageService) SetWebSocketManager(manager service.WebSocketManager) {
	if m.SetWebSocketManagerFn != nil {
		m.SetWebSocketManagerFn(manager)
	}
}

// ---------------------------------------------------------------------------
// MockChannelService
// ---------------------------------------------------------------------------

// MockChannelService is a test double for service.ChannelService.
type MockChannelService struct {
	CreateChannelFn         func(userID uint, req *service.CreateChannelRequest) (*model.Channel, error)
	GetChannelFn            func(channelID, userID uint) (*model.Channel, error)
	ListGuildChannelsFn     func(guildID, userID uint) ([]*model.Channel, error)
	UpdateChannelFn         func(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error)
	DeleteChannelFn         func(channelID, userID uint) error
	UpdateChannelPositionFn func(channelID, userID uint, position int) error
}

var _ service.ChannelService = (*MockChannelService)(nil)

func (m *MockChannelService) CreateChannel(userID uint, req *service.CreateChannelRequest) (*model.Channel, error) {
	if m.CreateChannelFn != nil {
		return m.CreateChannelFn(userID, req)
	}
	return nil, nil
}

func (m *MockChannelService) GetChannel(channelID, userID uint) (*model.Channel, error) {
	if m.GetChannelFn != nil {
		return m.GetChannelFn(channelID, userID)
	}
	return nil, nil
}

func (m *MockChannelService) ListGuildChannels(guildID, userID uint) ([]*model.Channel, error) {
	if m.ListGuildChannelsFn != nil {
		return m.ListGuildChannelsFn(guildID, userID)
	}
	return nil, nil
}

func (m *MockChannelService) UpdateChannel(channelID, userID uint, req *service.UpdateChannelRequest) (*model.Channel, error) {
	if m.UpdateChannelFn != nil {
		return m.UpdateChannelFn(channelID, userID, req)
	}
	return nil, nil
}

func (m *MockChannelService) DeleteChannel(channelID, userID uint) error {
	if m.DeleteChannelFn != nil {
		return m.DeleteChannelFn(channelID, userID)
	}
	return nil
}

func (m *MockChannelService) UpdateChannelPosition(channelID, userID uint, position int) error {
	if m.UpdateChannelPositionFn != nil {
		return m.UpdateChannelPositionFn(channelID, userID, position)
	}
	return nil
}
