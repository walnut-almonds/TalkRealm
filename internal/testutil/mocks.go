package testutil

import (
	"time"

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

	return nil, nil //nolint:nilnil
}

func (m *MockUserRepository) GetByEmail(email string) (*model.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(email)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserRepository) GetByUsername(username string) (*model.User, error) {
	if m.GetByUsernameFn != nil {
		return m.GetByUsernameFn(username)
	}

	return nil, nil //nolint:nilnil
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

func (m *MockUserRepository) SearchUsers(
	query string,
	excludeID uint,
	limit int,
) ([]*model.User, error) {
	return nil, nil
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

	return nil, nil //nolint:nilnil
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
// MockOAuthProviderRepository
// ---------------------------------------------------------------------------

// MockOAuthProviderRepository is a test double for repository.OAuthProviderRepository.
type MockOAuthProviderRepository struct {
	FindByProviderFn            func(provider, providerID string) (*model.UserOAuthProvider, error)
	CreateFn                    func(link *model.UserOAuthProvider) error
	UpdateFn                    func(link *model.UserOAuthProvider) error
	ListByUserIDFn              func(userID uint) ([]*model.UserOAuthProvider, error)
	DeleteByUserIDAndProviderFn func(userID uint, provider string) error
}

var _ repository.OAuthProviderRepository = (*MockOAuthProviderRepository)(nil)

func (m *MockOAuthProviderRepository) FindByProvider(
	provider, providerID string,
) (*model.UserOAuthProvider, error) {
	if m.FindByProviderFn != nil {
		return m.FindByProviderFn(provider, providerID)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockOAuthProviderRepository) Create(link *model.UserOAuthProvider) error {
	if m.CreateFn != nil {
		return m.CreateFn(link)
	}

	return nil
}

func (m *MockOAuthProviderRepository) Update(link *model.UserOAuthProvider) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(link)
	}

	return nil
}

func (m *MockOAuthProviderRepository) ListByUserID(
	userID uint,
) ([]*model.UserOAuthProvider, error) {
	if m.ListByUserIDFn != nil {
		return m.ListByUserIDFn(userID)
	}

	return nil, nil
}

func (m *MockOAuthProviderRepository) DeleteByUserIDAndProvider(
	userID uint,
	provider string,
) error {
	if m.DeleteByUserIDAndProviderFn != nil {
		return m.DeleteByUserIDAndProviderFn(userID, provider)
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

	return nil, nil //nolint:nilnil
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

func (m *MockGuildRepository) GetMemberGuilds(
	userID uint,
	offset, limit int,
) ([]*model.Guild, error) {
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

	return nil, nil //nolint:nilnil
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

	return nil, nil //nolint:nilnil
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
	CreateFn                   func(message *model.Message) error
	GetByIDFn                  func(id uint) (*model.Message, error)
	GetByNonceFn               func(userID uint, nonce string) (*model.Message, error)
	UpdateFn                   func(message *model.Message) error
	DeleteFn                   func(id uint) error
	GetByChannelIDFn           func(channelID uint, offset, limit int) ([]*model.Message, error)
	GetByChannelIDCursorFn     func(channelID, before uint, limit int) ([]*model.Message, error)
	GetMessagesAroundFn        func(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetPostsAroundFn           func(channelID, aroundID uint, limit int) ([]*model.Message, error)
	GetByUserIDFn              func(userID uint, offset, limit int) ([]*model.Message, error)
	CountByUserInGuildsSinceFn func(userID uint, guildIDs []uint, since time.Time) (map[uint]int64, error)
	GetPostsByChannelCursorFn  func(channelID, before uint, limit int) ([]*model.Message, error)
	GetCommentsByPostCursorFn  func(postID, before uint, limit int) ([]*model.Message, error)
	CountCommentsByPostIDsFn   func(ids []uint) (map[uint]int64, error)
	DeletePostCascadeFn        func(postID uint) error
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

	return nil, nil //nolint:nilnil
}

func (m *MockMessageRepository) GetByNonce(userID uint, nonce string) (*model.Message, error) {
	if m.GetByNonceFn != nil {
		return m.GetByNonceFn(userID, nonce)
	}

	return nil, nil //nolint:nilnil
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

func (m *MockMessageRepository) GetByChannelID(
	channelID uint,
	offset, limit int,
) ([]*model.Message, error) {
	if m.GetByChannelIDFn != nil {
		return m.GetByChannelIDFn(channelID, offset, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetByChannelIDCursor(
	channelID, before uint,
	limit int,
) ([]*model.Message, error) {
	if m.GetByChannelIDCursorFn != nil {
		return m.GetByChannelIDCursorFn(channelID, before, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetMessagesAround(
	channelID, aroundID uint,
	limit int,
) ([]*model.Message, error) {
	if m.GetMessagesAroundFn != nil {
		return m.GetMessagesAroundFn(channelID, aroundID, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetPostsAround(
	channelID, aroundID uint,
	limit int,
) ([]*model.Message, error) {
	if m.GetPostsAroundFn != nil {
		return m.GetPostsAroundFn(channelID, aroundID, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetByUserID(
	userID uint,
	offset, limit int,
) ([]*model.Message, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(userID, offset, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) CountByUserInGuildsSince(
	userID uint,
	guildIDs []uint,
	since time.Time,
) (map[uint]int64, error) {
	if m.CountByUserInGuildsSinceFn != nil {
		return m.CountByUserInGuildsSinceFn(userID, guildIDs, since)
	}

	return map[uint]int64{}, nil
}

func (m *MockMessageRepository) GetPostsByChannelCursor(
	channelID, before uint,
	limit int,
) ([]*model.Message, error) {
	if m.GetPostsByChannelCursorFn != nil {
		return m.GetPostsByChannelCursorFn(channelID, before, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) GetCommentsByPostCursor(
	postID, before uint,
	limit int,
) ([]*model.Message, error) {
	if m.GetCommentsByPostCursorFn != nil {
		return m.GetCommentsByPostCursorFn(postID, before, limit)
	}

	return nil, nil
}

func (m *MockMessageRepository) CountCommentsByPostIDs(ids []uint) (map[uint]int64, error) {
	if m.CountCommentsByPostIDsFn != nil {
		return m.CountCommentsByPostIDsFn(ids)
	}

	return map[uint]int64{}, nil
}

func (m *MockMessageRepository) DeletePostCascade(postID uint) error {
	if m.DeletePostCascadeFn != nil {
		return m.DeletePostCascadeFn(postID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// MockMessageLikeRepository
// ---------------------------------------------------------------------------

// MockMessageLikeRepository is a test double for repository.MessageLikeRepository.
type MockMessageLikeRepository struct {
	CreateFn             func(like *model.MessageLike) error
	DeleteFn             func(messageID, userID uint) error
	CountByMessageIDFn   func(messageID uint) (int64, error)
	CountByMessageIDsFn  func(ids []uint) (map[uint]int64, error)
	LikedMessageIDsFn    func(userID uint, ids []uint) (map[uint]bool, error)
	DeleteByMessageIDsFn func(ids []uint) error
}

var _ repository.MessageLikeRepository = (*MockMessageLikeRepository)(nil)

func (m *MockMessageLikeRepository) Create(like *model.MessageLike) error {
	if m.CreateFn != nil {
		return m.CreateFn(like)
	}

	return nil
}

func (m *MockMessageLikeRepository) Delete(messageID, userID uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(messageID, userID)
	}

	return nil
}

func (m *MockMessageLikeRepository) CountByMessageID(messageID uint) (int64, error) {
	if m.CountByMessageIDFn != nil {
		return m.CountByMessageIDFn(messageID)
	}

	return 0, nil
}

func (m *MockMessageLikeRepository) CountByMessageIDs(ids []uint) (map[uint]int64, error) {
	if m.CountByMessageIDsFn != nil {
		return m.CountByMessageIDsFn(ids)
	}

	return map[uint]int64{}, nil
}

func (m *MockMessageLikeRepository) LikedMessageIDs(
	userID uint,
	ids []uint,
) (map[uint]bool, error) {
	if m.LikedMessageIDsFn != nil {
		return m.LikedMessageIDsFn(userID, ids)
	}

	return map[uint]bool{}, nil
}

func (m *MockMessageLikeRepository) DeleteByMessageIDs(ids []uint) error {
	if m.DeleteByMessageIDsFn != nil {
		return m.DeleteByMessageIDsFn(ids)
	}

	return nil
}

// ---------------------------------------------------------------------------
// MockChannelRepository
// ---------------------------------------------------------------------------

// MockChannelRepository is a test double for repository.ChannelRepository.
type MockChannelRepository struct {
	CreateFn          func(channel *model.Channel) error
	GetByIDFn         func(id uint) (*model.Channel, error)
	UpdateFn          func(channel *model.Channel) error
	DeleteFn          func(id uint) error
	GetByGuildIDFn    func(guildID uint) ([]*model.Channel, error)
	GetByTypeFn       func(guildID uint, channelType string) ([]*model.Channel, error)
	IsDMParticipantFn func(channelID, userID uint) (bool, error)
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

	return nil, nil //nolint:nilnil
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

func (m *MockChannelRepository) GetByType(
	guildID uint,
	channelType string,
) ([]*model.Channel, error) {
	if m.GetByTypeFn != nil {
		return m.GetByTypeFn(guildID, channelType)
	}

	return nil, nil
}

func (m *MockChannelRepository) GetOrCreateDMChannel(
	user1ID, user2ID uint,
) (*model.Channel, error) {
	return nil, nil //nolint:nilnil
}

func (m *MockChannelRepository) ListDMChannels(userID uint) ([]*model.Channel, error) {
	return nil, nil
}

func (m *MockChannelRepository) IsDMParticipant(channelID, userID uint) (bool, error) {
	if m.IsDMParticipantFn != nil {
		return m.IsDMParticipantFn(channelID, userID)
	}

	return true, nil
}

func (m *MockChannelRepository) GetDMParticipants(
	channelID uint,
) ([]*model.ChannelParticipant, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// MockGuildInviteRepository
// ---------------------------------------------------------------------------

// MockGuildInviteRepository is a test double for repository.GuildInviteRepository.
type MockGuildInviteRepository struct {
	CreateFn        func(invite *model.GuildInvite) error
	GetByCodeFn     func(code string) (*model.GuildInvite, error)
	ListByGuildIDFn func(guildID uint) ([]*model.GuildInvite, error)
	IncrementUsesFn func(id uint) error
	DeleteFn        func(id uint) error
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

	return nil, nil //nolint:nilnil
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
// MockFollowRepository
// ---------------------------------------------------------------------------

// MockFollowRepository is a test double for repository.FollowRepository.
type MockFollowRepository struct {
	FollowFn                func(followerID, followeeID uint) error
	UnfollowFn              func(followerID, followeeID uint) error
	IsFollowingFn           func(followerID, followeeID uint) (bool, error)
	FolloweeIDsFn           func(followerID uint) ([]uint, error)
	FollowerIDsFn           func(followeeID uint) ([]uint, error)
	SecondDegreeAuthorIDsFn func(viewerID uint) (map[uint]bool, error)
	ListFollowingFn         func(userID uint) ([]*model.Follow, error)
	ListFollowersFn         func(userID uint) ([]*model.Follow, error)
	CountFollowingFn        func(userID uint) (int64, error)
	CountFollowersFn        func(userID uint) (int64, error)
}

var _ repository.FollowRepository = (*MockFollowRepository)(nil)

func (m *MockFollowRepository) Follow(followerID, followeeID uint) error {
	if m.FollowFn != nil {
		return m.FollowFn(followerID, followeeID)
	}

	return nil
}

func (m *MockFollowRepository) Unfollow(followerID, followeeID uint) error {
	if m.UnfollowFn != nil {
		return m.UnfollowFn(followerID, followeeID)
	}

	return nil
}

func (m *MockFollowRepository) IsFollowing(followerID, followeeID uint) (bool, error) {
	if m.IsFollowingFn != nil {
		return m.IsFollowingFn(followerID, followeeID)
	}

	return false, nil
}

func (m *MockFollowRepository) FolloweeIDs(followerID uint) ([]uint, error) {
	if m.FolloweeIDsFn != nil {
		return m.FolloweeIDsFn(followerID)
	}

	return nil, nil
}

func (m *MockFollowRepository) FollowerIDs(followeeID uint) ([]uint, error) {
	if m.FollowerIDsFn != nil {
		return m.FollowerIDsFn(followeeID)
	}

	return nil, nil
}

func (m *MockFollowRepository) SecondDegreeAuthorIDs(viewerID uint) (map[uint]bool, error) {
	if m.SecondDegreeAuthorIDsFn != nil {
		return m.SecondDegreeAuthorIDsFn(viewerID)
	}

	return map[uint]bool{}, nil
}

func (m *MockFollowRepository) ListFollowing(userID uint) ([]*model.Follow, error) {
	if m.ListFollowingFn != nil {
		return m.ListFollowingFn(userID)
	}

	return nil, nil
}

func (m *MockFollowRepository) ListFollowers(userID uint) ([]*model.Follow, error) {
	if m.ListFollowersFn != nil {
		return m.ListFollowersFn(userID)
	}

	return nil, nil
}

func (m *MockFollowRepository) CountFollowing(userID uint) (int64, error) {
	if m.CountFollowingFn != nil {
		return m.CountFollowingFn(userID)
	}

	return 0, nil
}

func (m *MockFollowRepository) CountFollowers(userID uint) (int64, error) {
	if m.CountFollowersFn != nil {
		return m.CountFollowersFn(userID)
	}

	return 0, nil
}

// ---------------------------------------------------------------------------
// MockFeedPostRepository
// ---------------------------------------------------------------------------

// MockFeedPostRepository is a test double for repository.FeedPostRepository.
type MockFeedPostRepository struct {
	CreateFn           func(p *model.FeedPost) error
	GetByIDFn          func(id uint) (*model.FeedPost, error)
	UpdateFn           func(p *model.FeedPost) error
	AttachFilesFn      func(postID uint, fileIDs []uint) error
	TimelineCursorFn   func(authorIDs []uint, before uint, limit int) ([]*model.FeedPost, error)
	ByAuthorCursorFn   func(authorID, before uint, limit int) ([]*model.FeedPost, error)
	DeleteCascadeFn    func(postID uint) error
	RecentCandidatesFn func(excludeAuthorID uint, since time.Time, limit int) ([]*model.FeedPost, error)
	AuthorAffinityFn   func(viewerID uint) (map[uint]int64, error)
}

var _ repository.FeedPostRepository = (*MockFeedPostRepository)(nil)

func (m *MockFeedPostRepository) Create(p *model.FeedPost) error {
	if m.CreateFn != nil {
		return m.CreateFn(p)
	}

	return nil
}

func (m *MockFeedPostRepository) GetByID(id uint) (*model.FeedPost, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil
}

func (m *MockFeedPostRepository) Update(p *model.FeedPost) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(p)
	}

	return nil
}

func (m *MockFeedPostRepository) AttachFiles(postID uint, fileIDs []uint) error {
	if m.AttachFilesFn != nil {
		return m.AttachFilesFn(postID, fileIDs)
	}

	return nil
}

func (m *MockFeedPostRepository) TimelineCursor(
	authorIDs []uint,
	before uint,
	limit int,
) ([]*model.FeedPost, error) {
	if m.TimelineCursorFn != nil {
		return m.TimelineCursorFn(authorIDs, before, limit)
	}

	return nil, nil
}

func (m *MockFeedPostRepository) ByAuthorCursor(
	authorID, before uint,
	limit int,
) ([]*model.FeedPost, error) {
	if m.ByAuthorCursorFn != nil {
		return m.ByAuthorCursorFn(authorID, before, limit)
	}

	return nil, nil
}

func (m *MockFeedPostRepository) DeleteCascade(postID uint) error {
	if m.DeleteCascadeFn != nil {
		return m.DeleteCascadeFn(postID)
	}

	return nil
}

func (m *MockFeedPostRepository) RecentCandidates(
	excludeAuthorID uint,
	since time.Time,
	limit int,
) ([]*model.FeedPost, error) {
	if m.RecentCandidatesFn != nil {
		return m.RecentCandidatesFn(excludeAuthorID, since, limit)
	}

	return nil, nil
}

func (m *MockFeedPostRepository) AuthorAffinity(viewerID uint) (map[uint]int64, error) {
	if m.AuthorAffinityFn != nil {
		return m.AuthorAffinityFn(viewerID)
	}

	return map[uint]int64{}, nil
}

// ---------------------------------------------------------------------------
// MockFeedPostLikeRepository
// ---------------------------------------------------------------------------

// MockFeedPostLikeRepository is a test double for repository.FeedPostLikeRepository.
type MockFeedPostLikeRepository struct {
	CreateFn         func(like *model.FeedPostLike) error
	DeleteFn         func(postID, userID uint) error
	CountByPostIDFn  func(postID uint) (int64, error)
	CountByPostIDsFn func(ids []uint) (map[uint]int64, error)
	LikedPostIDsFn   func(userID uint, ids []uint) (map[uint]bool, error)
}

var _ repository.FeedPostLikeRepository = (*MockFeedPostLikeRepository)(nil)

func (m *MockFeedPostLikeRepository) Create(like *model.FeedPostLike) error {
	if m.CreateFn != nil {
		return m.CreateFn(like)
	}

	return nil
}

func (m *MockFeedPostLikeRepository) Delete(postID, userID uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(postID, userID)
	}

	return nil
}

func (m *MockFeedPostLikeRepository) CountByPostID(postID uint) (int64, error) {
	if m.CountByPostIDFn != nil {
		return m.CountByPostIDFn(postID)
	}

	return 0, nil
}

func (m *MockFeedPostLikeRepository) CountByPostIDs(ids []uint) (map[uint]int64, error) {
	if m.CountByPostIDsFn != nil {
		return m.CountByPostIDsFn(ids)
	}

	return map[uint]int64{}, nil
}

func (m *MockFeedPostLikeRepository) LikedPostIDs(
	userID uint,
	ids []uint,
) (map[uint]bool, error) {
	if m.LikedPostIDsFn != nil {
		return m.LikedPostIDsFn(userID, ids)
	}

	return map[uint]bool{}, nil
}

// ---------------------------------------------------------------------------
// MockFeedCommentLikeRepository
// ---------------------------------------------------------------------------

// MockFeedCommentLikeRepository is a test double for repository.FeedCommentLikeRepository.
type MockFeedCommentLikeRepository struct {
	CreateFn            func(like *model.FeedCommentLike) error
	DeleteFn            func(commentID, userID uint) error
	CountByCommentIDFn  func(commentID uint) (int64, error)
	CountByCommentIDsFn func(ids []uint) (map[uint]int64, error)
	LikedCommentIDsFn   func(userID uint, ids []uint) (map[uint]bool, error)
}

var _ repository.FeedCommentLikeRepository = (*MockFeedCommentLikeRepository)(nil)

func (m *MockFeedCommentLikeRepository) Create(like *model.FeedCommentLike) error {
	if m.CreateFn != nil {
		return m.CreateFn(like)
	}

	return nil
}

func (m *MockFeedCommentLikeRepository) Delete(commentID, userID uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(commentID, userID)
	}

	return nil
}

func (m *MockFeedCommentLikeRepository) CountByCommentID(commentID uint) (int64, error) {
	if m.CountByCommentIDFn != nil {
		return m.CountByCommentIDFn(commentID)
	}

	return 0, nil
}

func (m *MockFeedCommentLikeRepository) CountByCommentIDs(ids []uint) (map[uint]int64, error) {
	if m.CountByCommentIDsFn != nil {
		return m.CountByCommentIDsFn(ids)
	}

	return map[uint]int64{}, nil
}

func (m *MockFeedCommentLikeRepository) LikedCommentIDs(
	userID uint,
	ids []uint,
) (map[uint]bool, error) {
	if m.LikedCommentIDsFn != nil {
		return m.LikedCommentIDsFn(userID, ids)
	}

	return map[uint]bool{}, nil
}

// ---------------------------------------------------------------------------
// MockFeedCommentRepository
// ---------------------------------------------------------------------------

// MockFeedCommentRepository is a test double for repository.FeedCommentRepository.
type MockFeedCommentRepository struct {
	CreateFn         func(c *model.FeedComment) error
	GetByIDFn        func(id uint) (*model.FeedComment, error)
	UpdateFn         func(c *model.FeedComment) error
	DeleteFn         func(id uint) error
	ByPostCursorFn   func(postID, before uint, limit int) ([]*model.FeedComment, error)
	CountByPostIDsFn func(ids []uint) (map[uint]int64, error)
}

var _ repository.FeedCommentRepository = (*MockFeedCommentRepository)(nil)

func (m *MockFeedCommentRepository) Create(c *model.FeedComment) error {
	if m.CreateFn != nil {
		return m.CreateFn(c)
	}

	return nil
}

func (m *MockFeedCommentRepository) GetByID(id uint) (*model.FeedComment, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockFeedCommentRepository) Update(c *model.FeedComment) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(c)
	}

	return nil
}

func (m *MockFeedCommentRepository) Delete(id uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}

	return nil
}

func (m *MockFeedCommentRepository) ByPostCursor(
	postID, before uint,
	limit int,
) ([]*model.FeedComment, error) {
	if m.ByPostCursorFn != nil {
		return m.ByPostCursorFn(postID, before, limit)
	}

	return nil, nil
}

func (m *MockFeedCommentRepository) CountByPostIDs(ids []uint) (map[uint]int64, error) {
	if m.CountByPostIDsFn != nil {
		return m.CountByPostIDsFn(ids)
	}

	return map[uint]int64{}, nil
}

// ---------------------------------------------------------------------------
// MockFriendshipRepository
// ---------------------------------------------------------------------------

// MockFriendshipRepository is a test double for repository.FriendshipRepository.
type MockFriendshipRepository struct {
	CreateFn               func(requesterID, addresseeID uint) (*model.Friendship, error)
	GetBetweenFn           func(userA, userB uint) (*model.Friendship, error)
	UpdateStatusFn         func(requesterID, addresseeID uint, status string) error
	DeleteFn               func(userA, userB uint) error
	ListFriendsFn          func(userID uint) ([]*model.Friendship, error)
	ListIncomingRequestsFn func(userID uint) ([]*model.Friendship, error)
	ListOutgoingRequestsFn func(userID uint) ([]*model.Friendship, error)
	FriendIDsFn            func(userID uint) ([]uint, error)
}

var _ repository.FriendshipRepository = (*MockFriendshipRepository)(nil)

func (m *MockFriendshipRepository) Create(
	requesterID, addresseeID uint,
) (*model.Friendship, error) {
	if m.CreateFn != nil {
		return m.CreateFn(requesterID, addresseeID)
	}

	return nil, nil
}

func (m *MockFriendshipRepository) GetBetween(userA, userB uint) (*model.Friendship, error) {
	if m.GetBetweenFn != nil {
		return m.GetBetweenFn(userA, userB)
	}

	return nil, nil
}

func (m *MockFriendshipRepository) UpdateStatus(
	requesterID, addresseeID uint,
	status string,
) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(requesterID, addresseeID, status)
	}

	return nil
}

func (m *MockFriendshipRepository) Delete(userA, userB uint) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(userA, userB)
	}

	return nil
}

func (m *MockFriendshipRepository) ListFriends(userID uint) ([]*model.Friendship, error) {
	if m.ListFriendsFn != nil {
		return m.ListFriendsFn(userID)
	}

	return nil, nil
}

func (m *MockFriendshipRepository) ListIncomingRequests(userID uint) ([]*model.Friendship, error) {
	if m.ListIncomingRequestsFn != nil {
		return m.ListIncomingRequestsFn(userID)
	}

	return nil, nil
}

func (m *MockFriendshipRepository) ListOutgoingRequests(userID uint) ([]*model.Friendship, error) {
	if m.ListOutgoingRequestsFn != nil {
		return m.ListOutgoingRequestsFn(userID)
	}

	return nil, nil
}

func (m *MockFriendshipRepository) FriendIDs(userID uint) ([]uint, error) {
	if m.FriendIDsFn != nil {
		return m.FriendIDsFn(userID)
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock Service interfaces (for handler tests)
// ---------------------------------------------------------------------------

// MockUserService is a test double for service.UserService.
type MockUserService struct {
	RegisterFn             func(req *service.RegisterRequest) (*model.User, error)
	LoginFn                func(req *service.LoginRequest) (*service.LoginResponse, error)
	GetByIDFn              func(id uint) (*model.User, error)
	GetPublicByIDFn        func(id uint) (*service.PublicUser, error)
	UpdateFn               func(id uint, req *service.UpdateUserRequest) (*model.User, error)
	UpdateStatusFn         func(id uint, status string) error
	RefreshAccessTokenFn   func(refreshToken string) (*service.LoginResponse, error)
	RevokeRefreshTokenFn   func(refreshToken string) error
	OAuthLoginOrRegisterFn func(req *service.OAuthUserInfo) (*service.LoginResponse, error)
}

var _ service.UserService = (*MockUserService)(nil)

func (m *MockUserService) Register(req *service.RegisterRequest) (*model.User, error) {
	if m.RegisterFn != nil {
		return m.RegisterFn(req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) Login(req *service.LoginRequest) (*service.LoginResponse, error) {
	if m.LoginFn != nil {
		return m.LoginFn(req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) GetByID(id uint) (*model.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) GetPublicByID(id uint) (*service.PublicUser, error) {
	if m.GetPublicByIDFn != nil {
		return m.GetPublicByIDFn(id)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) Update(id uint, req *service.UpdateUserRequest) (*model.User, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(id, req)
	}

	return nil, nil //nolint:nilnil
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

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) RevokeRefreshToken(refreshToken string) error {
	if m.RevokeRefreshTokenFn != nil {
		return m.RevokeRefreshTokenFn(refreshToken)
	}

	return nil
}

func (m *MockUserService) OAuthLoginOrRegister(
	req *service.OAuthUserInfo,
) (*service.LoginResponse, error) {
	if m.OAuthLoginOrRegisterFn != nil {
		return m.OAuthLoginOrRegisterFn(req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockUserService) SearchUsers(query string, excludeID uint) ([]*service.PublicUser, error) {
	return nil, nil
}

// MockGuildService is a test double for service.GuildService.
type MockGuildService struct {
	CreateGuildFn         func(ownerID uint, req *service.CreateGuildRequest) (*model.Guild, error)
	GetGuildFn            func(guildID uint) (*model.Guild, error)
	ListUserGuildsFn      func(userID uint) ([]*model.Guild, error)
	UpdateGuildFn         func(guildID, userID uint, req *service.UpdateGuildRequest) (*model.Guild, error)
	DeleteGuildFn         func(guildID, userID uint) error
	IsGuildOwnerFn        func(guildID, userID uint) (bool, error)
	IsGuildMemberFn       func(guildID, userID uint) (bool, error)
	SetWebSocketManagerFn func(m service.GuildEventBroadcaster)
}

var _ service.GuildService = (*MockGuildService)(nil)

func (m *MockGuildService) CreateGuild(
	ownerID uint,
	req *service.CreateGuildRequest,
) (*model.Guild, error) {
	if m.CreateGuildFn != nil {
		return m.CreateGuildFn(ownerID, req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockGuildService) GetGuild(guildID uint) (*model.Guild, error) {
	if m.GetGuildFn != nil {
		return m.GetGuildFn(guildID)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockGuildService) ListUserGuilds(userID uint) ([]*model.Guild, error) {
	if m.ListUserGuildsFn != nil {
		return m.ListUserGuildsFn(userID)
	}

	return nil, nil
}

func (m *MockGuildService) UpdateGuild(
	guildID, userID uint,
	req *service.UpdateGuildRequest,
) (*model.Guild, error) {
	if m.UpdateGuildFn != nil {
		return m.UpdateGuildFn(guildID, userID, req)
	}

	return nil, nil //nolint:nilnil
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

func (m *MockGuildService) SetWebSocketManager(mgr service.GuildEventBroadcaster) {
	if m.SetWebSocketManagerFn != nil {
		m.SetWebSocketManagerFn(mgr)
	}
}

// MockGuildMemberService is a test double for service.GuildMemberService.
type MockGuildMemberService struct {
	JoinGuildFn           func(guildID, userID uint) error
	LeaveGuildFn          func(guildID, userID uint) error
	KickMemberFn          func(guildID, targetUserID, operatorUserID uint) error
	ListGuildMembersFn    func(guildID uint) ([]*model.GuildMember, error)
	GetMemberFn           func(guildID, userID uint) (*model.GuildMember, error)
	UpdateMemberRoleFn    func(guildID, targetUserID, operatorUserID uint, role string) error
	SetWebSocketManagerFn func(m service.GuildEventBroadcaster)
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

	return nil, nil //nolint:nilnil
}

func (m *MockGuildMemberService) UpdateMemberRole(
	guildID, targetUserID, operatorUserID uint,
	role string,
) error {
	if m.UpdateMemberRoleFn != nil {
		return m.UpdateMemberRoleFn(guildID, targetUserID, operatorUserID, role)
	}

	return nil
}

func (m *MockGuildMemberService) SetWebSocketManager(mgr service.GuildEventBroadcaster) {
	if m.SetWebSocketManagerFn != nil {
		m.SetWebSocketManagerFn(mgr)
	}
}

// MockGuildInviteService is a test double for service.GuildInviteService.
type MockGuildInviteService struct {
	CreateInviteFn        func(guildID, creatorID uint, req *service.CreateInviteRequest) (*model.GuildInvite, error)
	GetInviteByCodeFn     func(code string) (*model.GuildInvite, error)
	JoinByInviteFn        func(code string, userID uint) error
	SetWebSocketManagerFn func(m service.GuildEventBroadcaster)
}

var _ service.GuildInviteService = (*MockGuildInviteService)(nil)

func (m *MockGuildInviteService) CreateInvite(
	guildID, creatorID uint,
	req *service.CreateInviteRequest,
) (*model.GuildInvite, error) {
	if m.CreateInviteFn != nil {
		return m.CreateInviteFn(guildID, creatorID, req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockGuildInviteService) GetInviteByCode(code string) (*model.GuildInvite, error) {
	if m.GetInviteByCodeFn != nil {
		return m.GetInviteByCodeFn(code)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockGuildInviteService) JoinByInvite(code string, userID uint) error {
	if m.JoinByInviteFn != nil {
		return m.JoinByInviteFn(code, userID)
	}

	return nil
}

func (m *MockGuildInviteService) SetWebSocketManager(mgr service.GuildEventBroadcaster) {
	if m.SetWebSocketManagerFn != nil {
		m.SetWebSocketManagerFn(mgr)
	}
}

// MockMessageService is a test double for service.MessageService.
type MockMessageService struct {
	CreateMessageFn       func(userID uint, req *service.CreateMessageRequest) (*model.Message, error)
	GetMessageFn          func(messageID, userID uint) (*model.Message, error)
	ListChannelMessagesFn func(channelID, userID uint, limit int, before uint) (*service.MessageListResponse, error)
	UpdateMessageFn       func(messageID, userID uint, req *service.UpdateMessageRequest) (*model.Message, error)
	DeleteMessageFn       func(messageID, userID uint) error
	SetWebSocketManagerFn func(manager service.WebSocketManager)
	CreateMessageWSFn     func(userID, channelID uint, content, contentType, nonce string, fileIDs []uint) (any, error)
	LikePostFn            func(messageID, userID uint) (int64, error)
	UnlikePostFn          func(messageID, userID uint) (int64, error)
	ListPostsFn           func(channelID, userID uint, limit int, before uint) (*service.PostListResponse, error)
	ListCommentsFn        func(postID, userID uint, limit int, before uint) (*service.WallCommentListResponse, error)
	ResolvePermalinkFn    func(messageID, viewerID uint) (*service.PermalinkResponse, error)
	MessagesAroundFn      func(channelID, userID, aroundID uint, limit int) (*service.MessageListResponse, error)
	PostsAroundFn         func(channelID, userID, aroundID uint, limit int) (*service.PostListResponse, error)
}

var _ service.MessageService = (*MockMessageService)(nil)

func (m *MockMessageService) CreateMessage(
	userID uint,
	req *service.CreateMessageRequest,
) (*model.Message, error) {
	if m.CreateMessageFn != nil {
		return m.CreateMessageFn(userID, req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) GetMessage(messageID, userID uint) (*model.Message, error) {
	if m.GetMessageFn != nil {
		return m.GetMessageFn(messageID, userID)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) ResolvePermalink(
	messageID, viewerID uint,
) (*service.PermalinkResponse, error) {
	if m.ResolvePermalinkFn != nil {
		return m.ResolvePermalinkFn(messageID, viewerID)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) MessagesAround(
	channelID, userID, aroundID uint,
	limit int,
) (*service.MessageListResponse, error) {
	if m.MessagesAroundFn != nil {
		return m.MessagesAroundFn(channelID, userID, aroundID, limit)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) PostsAround(
	channelID, userID, aroundID uint,
	limit int,
) (*service.PostListResponse, error) {
	if m.PostsAroundFn != nil {
		return m.PostsAroundFn(channelID, userID, aroundID, limit)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) ListChannelMessages(
	channelID, userID uint,
	limit int,
	before uint,
) (*service.MessageListResponse, error) {
	if m.ListChannelMessagesFn != nil {
		return m.ListChannelMessagesFn(channelID, userID, limit, before)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) UpdateMessage(
	messageID, userID uint,
	req *service.UpdateMessageRequest,
) (*model.Message, error) {
	if m.UpdateMessageFn != nil {
		return m.UpdateMessageFn(messageID, userID, req)
	}

	return nil, nil //nolint:nilnil
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

func (m *MockMessageService) SetFileService(_ service.FileService) {}

func (m *MockMessageService) SetTranslationService(_ service.TranslationService) {}

func (m *MockMessageService) SetLikeRepo(_ repository.MessageLikeRepository) {}

func (m *MockMessageService) LikePost(messageID, userID uint) (int64, error) {
	if m.LikePostFn != nil {
		return m.LikePostFn(messageID, userID)
	}

	return 0, nil
}

func (m *MockMessageService) UnlikePost(messageID, userID uint) (int64, error) {
	if m.UnlikePostFn != nil {
		return m.UnlikePostFn(messageID, userID)
	}

	return 0, nil
}

func (m *MockMessageService) ListPosts(
	channelID, userID uint,
	limit int,
	before uint,
) (*service.PostListResponse, error) {
	if m.ListPostsFn != nil {
		return m.ListPostsFn(channelID, userID, limit, before)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) ListComments(
	postID, userID uint,
	limit int,
	before uint,
) (*service.WallCommentListResponse, error) {
	if m.ListCommentsFn != nil {
		return m.ListCommentsFn(postID, userID, limit, before)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockMessageService) CreateMessageWS(
	userID, channelID uint,
	content, contentType, nonce string,
	fileIDs []uint,
) (any, error) {
	if m.CreateMessageWSFn != nil {
		return m.CreateMessageWSFn(userID, channelID, content, contentType, nonce, fileIDs)
	}

	return nil, nil //nolint:nilnil
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
	SetWebSocketManagerFn   func(m service.GuildEventBroadcaster)
}

var _ service.ChannelService = (*MockChannelService)(nil)

func (m *MockChannelService) CreateChannel(
	userID uint,
	req *service.CreateChannelRequest,
) (*model.Channel, error) {
	if m.CreateChannelFn != nil {
		return m.CreateChannelFn(userID, req)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockChannelService) GetChannel(channelID, userID uint) (*model.Channel, error) {
	if m.GetChannelFn != nil {
		return m.GetChannelFn(channelID, userID)
	}

	return nil, nil //nolint:nilnil
}

func (m *MockChannelService) ListGuildChannels(guildID, userID uint) ([]*model.Channel, error) {
	if m.ListGuildChannelsFn != nil {
		return m.ListGuildChannelsFn(guildID, userID)
	}

	return nil, nil
}

func (m *MockChannelService) UpdateChannel(
	channelID, userID uint,
	req *service.UpdateChannelRequest,
) (*model.Channel, error) {
	if m.UpdateChannelFn != nil {
		return m.UpdateChannelFn(channelID, userID, req)
	}

	return nil, nil //nolint:nilnil
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

func (m *MockChannelService) SetWebSocketManager(mgr service.GuildEventBroadcaster) {
	if m.SetWebSocketManagerFn != nil {
		m.SetWebSocketManagerFn(mgr)
	}
}

// PtrUint returns a pointer to the given uint value.
func PtrUint(v uint) *uint { return &v }

// ---------------------------------------------------------------------------
// MockWebSocketManager
// ---------------------------------------------------------------------------

// WSBroadcast records a single BroadcastToChannel/BroadcastToUser call.
type WSBroadcast struct {
	ID      uint // channelID or userID
	MsgType string
	Data    any
}

// MockWebSocketManager is a test double for service.WebSocketManager that
// records broadcasts so tests can assert on them.
type MockWebSocketManager struct {
	ChannelBroadcasts []WSBroadcast
	UserBroadcasts    []WSBroadcast
	OnlineFn          func(userID uint) bool
}

var _ service.WebSocketManager = (*MockWebSocketManager)(nil)

func (m *MockWebSocketManager) BroadcastToChannel(channelID uint, msgType string, data any) {
	m.ChannelBroadcasts = append(
		m.ChannelBroadcasts,
		WSBroadcast{ID: channelID, MsgType: msgType, Data: data},
	)
}

func (m *MockWebSocketManager) BroadcastToUser(userID uint, msgType string, data any) {
	m.UserBroadcasts = append(
		m.UserBroadcasts,
		WSBroadcast{ID: userID, MsgType: msgType, Data: data},
	)
}

func (m *MockWebSocketManager) IsUserOnline(userID uint) bool {
	if m.OnlineFn != nil {
		return m.OnlineFn(userID)
	}

	return false
}

// ── MockMessageMentionRepository ─────────────────────────────────────────────

// MockMessageMentionRepository 是 repository.MessageMentionRepository 的 mock。
type MockMessageMentionRepository struct {
	BulkCreateFn          func(mentions []*model.MessageMention) error
	GetByMessageIDFn      func(messageID uint) ([]*model.MessageMention, error)
	GetMentionedUserIDsFn func(messageID uint) ([]uint, error)
}

func (m *MockMessageMentionRepository) BulkCreate(mentions []*model.MessageMention) error {
	if m.BulkCreateFn != nil {
		return m.BulkCreateFn(mentions)
	}

	return nil
}

func (m *MockMessageMentionRepository) GetByMessageID(
	messageID uint,
) ([]*model.MessageMention, error) {
	if m.GetByMessageIDFn != nil {
		return m.GetByMessageIDFn(messageID)
	}

	return []*model.MessageMention{}, nil
}

func (m *MockMessageMentionRepository) GetMentionedUserIDs(messageID uint) ([]uint, error) {
	if m.GetMentionedUserIDsFn != nil {
		return m.GetMentionedUserIDsFn(messageID)
	}

	return []uint{}, nil
}

// ── MockChannelReadStateRepository ───────────────────────────────────────────

// MockChannelReadStateRepository 是 repository.ChannelReadStateRepository 的 mock。
type MockChannelReadStateRepository struct {
	UpsertFn           func(userID, channelID, lastMessageID uint) error
	GetLastReadFn      func(userID, channelID uint) (uint, error)
	GetAllUnreadFn     func(userID uint) ([]*repository.ChannelUnreadCount, error)
	GetChannelUnreadFn func(userID, channelID uint) (*repository.ChannelUnreadCount, error)
}

func (m *MockChannelReadStateRepository) Upsert(userID, channelID, lastMessageID uint) error {
	if m.UpsertFn != nil {
		return m.UpsertFn(userID, channelID, lastMessageID)
	}

	return nil
}

func (m *MockChannelReadStateRepository) GetLastRead(userID, channelID uint) (uint, error) {
	if m.GetLastReadFn != nil {
		return m.GetLastReadFn(userID, channelID)
	}

	return 0, nil
}

func (m *MockChannelReadStateRepository) GetAllUnread(
	userID uint,
) ([]*repository.ChannelUnreadCount, error) {
	if m.GetAllUnreadFn != nil {
		return m.GetAllUnreadFn(userID)
	}

	return []*repository.ChannelUnreadCount{}, nil
}

func (m *MockChannelReadStateRepository) GetChannelUnread(
	userID, channelID uint,
) (*repository.ChannelUnreadCount, error) {
	if m.GetChannelUnreadFn != nil {
		return m.GetChannelUnreadFn(userID, channelID)
	}

	return &repository.ChannelUnreadCount{}, nil
}

// ── MockUnreadService ─────────────────────────────────────────────────────────

// MockUnreadService 是 service.UnreadService 的 mock。
type MockUnreadService struct {
	AckChannelFn       func(userID, channelID, lastMessageID uint) error
	GetChannelUnreadFn func(userID, channelID uint) (*repository.ChannelUnreadCount, error)
	GetAllUnreadFn     func(userID uint) ([]*repository.ChannelUnreadCount, error)
}

func (m *MockUnreadService) AckChannel(userID, channelID, lastMessageID uint) error {
	if m.AckChannelFn != nil {
		return m.AckChannelFn(userID, channelID, lastMessageID)
	}

	return nil
}

func (m *MockUnreadService) GetChannelUnread(
	userID, channelID uint,
) (*repository.ChannelUnreadCount, error) {
	if m.GetChannelUnreadFn != nil {
		return m.GetChannelUnreadFn(userID, channelID)
	}

	return &repository.ChannelUnreadCount{}, nil
}

func (m *MockUnreadService) GetAllUnread(userID uint) ([]*repository.ChannelUnreadCount, error) {
	if m.GetAllUnreadFn != nil {
		return m.GetAllUnreadFn(userID)
	}

	return []*repository.ChannelUnreadCount{}, nil
}
