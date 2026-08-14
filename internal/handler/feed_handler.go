package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

// FeedHandler 個人動態牆處理器（跨社群 feed，routes under /api/v1/feed）
type FeedHandler struct {
	feedService service.FeedService
}

// NewFeedHandler 建立 feed 處理器實例
func NewFeedHandler(feedService service.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

// createFeedPostBody 建立貼文請求 body
type createFeedPostBody struct {
	Content string `json:"content"`
	FileIDs []uint `json:"file_ids"`
}

// feedContentBody 更新貼文/新增/更新留言的共用 body
type feedContentBody struct {
	Content string `json:"content" binding:"required"`
}

// feedUserID 從 context 取得目前使用者 ID，缺失時回 401 並回傳 false。
func feedUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return 0, false
	}

	return userID.(uint), true
}

// feedError 將 feed 服務錯誤映射為 HTTP 狀態碼。
func feedError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrFeedPostNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotFeedOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrFeedFileForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrCannotFollowSelf),
		errors.Is(err, service.ErrFeedContentTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Suggestions 追蹤建議（好友中尚未追蹤者）
//
//	@Summary		追蹤建議
//	@Description	列出好友中尚未追蹤的使用者
//	@Tags			feed
//	@Produce		json
//	@Success		200	{array}		model.User
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/feed/suggestions [get]
func (h *FeedHandler) Suggestions(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	users, err := h.feedService.Suggestions(userID)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// Follow 追蹤使用者
//
//	@Summary		追蹤使用者
//	@Description	單向追蹤指定使用者（idempotent）
//	@Tags			feed
//	@Produce		json
//	@Param			userId	path		int	true	"被追蹤使用者 ID"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/follows/{userId} [post]
func (h *FeedHandler) Follow(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.feedService.Follow(userID, uint(targetID)); err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

// Unfollow 取消追蹤使用者
//
//	@Summary		取消追蹤
//	@Description	取消追蹤指定使用者
//	@Tags			feed
//	@Produce		json
//	@Param			userId	path		int	true	"被追蹤使用者 ID"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/follows/{userId} [delete]
func (h *FeedHandler) Unfollow(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.feedService.Unfollow(userID, uint(targetID)); err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

// ListFollowing 列出使用者追蹤中的對象
//
//	@Summary		追蹤中列表
//	@Description	列出指定使用者追蹤中的對象
//	@Tags			feed
//	@Produce		json
//	@Param			userId	path		int	true	"使用者 ID"
//	@Success		200		{object}	service.FollowListResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/users/{userId}/following [get]
func (h *FeedHandler) ListFollowing(c *gin.Context) {
	if _, ok := feedUserID(c); !ok {
		return
	}

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	resp, err := h.feedService.ListFollowing(uint(targetID))
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListFollowers 列出使用者的追蹤者
//
//	@Summary		追蹤者列表
//	@Description	列出指定使用者的追蹤者
//	@Tags			feed
//	@Produce		json
//	@Param			userId	path		int	true	"使用者 ID"
//	@Success		200		{object}	service.FollowListResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/users/{userId}/followers [get]
func (h *FeedHandler) ListFollowers(c *gin.Context) {
	if _, ok := feedUserID(c); !ok {
		return
	}

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	resp, err := h.feedService.ListFollowers(uint(targetID))
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ProfilePosts 列出指定使用者的貼文
//
//	@Summary		個人頁貼文
//	@Description	列出指定使用者的貼文（cursor-based 分頁）
//	@Tags			feed
//	@Produce		json
//	@Param			userId	path		int	true	"使用者 ID"
//	@Param			before	query		int	false	"貼文 ID，取得此 ID 之前的貼文"
//	@Param			limit	query		int	false	"每次取得數量 (1-100)"	default(20)
//	@Success		200		{object}	service.TimelineResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/users/{userId}/posts [get]
func (h *FeedHandler) ProfilePosts(c *gin.Context) {
	viewerID, ok := feedUserID(c)
	if !ok {
		return
	}

	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	limit, before := parseListQuery(c, 20)

	resp, err := h.feedService.ProfilePosts(uint(targetID), viewerID, before, limit)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Timeline 個人時間軸
//
//	@Summary		時間軸
//	@Description	取得「追蹤對象 + 自己」的時間軸（cursor-based 分頁）
//	@Tags			feed
//	@Produce		json
//	@Param			before	query		int	false	"貼文 ID，取得此 ID 之前的貼文"
//	@Param			limit	query		int	false	"每次取得數量 (1-100)"	default(20)
//	@Success		200		{object}	service.TimelineResponse
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/feed/timeline [get]
func (h *FeedHandler) Timeline(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	limit, before := parseListQuery(c, 20)

	resp, err := h.feedService.Timeline(userID, before, limit)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Discover 探索時間軸（為你推薦）
//
//	@Summary		探索時間軸
//	@Description	取得演算法排序的「為你推薦」時間軸（offset-based 分頁）
//	@Tags			feed
//	@Produce		json
//	@Param			offset	query		int	false	"略過的貼文數"
//	@Param			limit	query		int	false	"每次取得數量 (1-100)"	default(20)
//	@Success		200		{object}	service.TimelineResponse
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/feed/discover [get]
func (h *FeedHandler) Discover(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	resp, err := h.feedService.DiscoverTimeline(userID, offset, limit)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreatePost 建立貼文
//
//	@Summary		建立貼文
//	@Description	建立個人貼文（可附帶檔案）
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			request	body		createFeedPostBody	true	"建立貼文請求"
//	@Success		201		{object}	service.FeedPostResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/posts [post]
func (h *FeedHandler) CreatePost(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	var body createFeedPostBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.feedService.CreatePost(userID, body.Content, body.FileIDs)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusCreated, post)
}

// UpdatePost 更新貼文
//
//	@Summary		更新貼文
//	@Description	更新自己的貼文內容
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"貼文 ID"
//	@Param			request	body		feedContentBody	true	"更新貼文請求"
//	@Success		200		{object}	service.FeedPostResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id} [put]
func (h *FeedHandler) UpdatePost(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var body feedContentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.feedService.UpdatePost(uint(postID), userID, body.Content)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost 刪除貼文
//
//	@Summary		刪除貼文
//	@Description	刪除自己的貼文（連帶留言/按讚/附件）
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"貼文 ID"
//	@Success		200	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id} [delete]
func (h *FeedHandler) DeletePost(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	if err := h.feedService.DeletePost(uint(postID), userID); err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted"})
}

// LikePost 對貼文按讚
//
//	@Summary		按讚
//	@Description	對貼文按讚（idempotent），回傳最新讚數
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"貼文 ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id}/like [put]
func (h *FeedHandler) LikePost(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	n, err := h.feedService.LikePost(uint(postID), userID)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"post_id": uint(postID), "like_count": n})
}

// UnlikePost 收回貼文的讚
//
//	@Summary		收回讚
//	@Description	收回對貼文的讚，回傳最新讚數
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"貼文 ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id}/like [delete]
func (h *FeedHandler) UnlikePost(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	n, err := h.feedService.UnlikePost(uint(postID), userID)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"post_id": uint(postID), "like_count": n})
}

// LikeComment 對留言按讚
//
//	@Summary		留言按讚
//	@Description	對留言按讚（idempotent），回傳最新讚數
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"留言 ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/feed/comments/{id}/like [put]
func (h *FeedHandler) LikeComment(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	commentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	n, err := h.feedService.LikeComment(uint(commentID), userID)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment_id": uint(commentID), "like_count": n})
}

// UnlikeComment 收回留言的讚
//
//	@Summary		收回留言讚
//	@Description	收回對留言的讚，回傳最新讚數
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"留言 ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Router			/api/v1/feed/comments/{id}/like [delete]
func (h *FeedHandler) UnlikeComment(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	commentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	n, err := h.feedService.UnlikeComment(uint(commentID), userID)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment_id": uint(commentID), "like_count": n})
}

// ListComments 列出貼文留言
//
//	@Summary		列出留言
//	@Description	列出指定貼文的單層留言（cursor-based 分頁，時間順序）
//	@Tags			feed
//	@Produce		json
//	@Param			id		path		int	true	"貼文 ID"
//	@Param			before	query		int	false	"留言 ID，取得此 ID 之前的留言"
//	@Param			limit	query		int	false	"每次取得數量 (1-100)"	default(50)
//	@Success		200		{object}	service.CommentListResponse
//	@Failure		401		{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id}/comments [get]
func (h *FeedHandler) ListComments(c *gin.Context) {
	viewerID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	limit, before := parseListQuery(c, 50)

	resp, err := h.feedService.ListComments(uint(postID), viewerID, before, limit)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// AddComment 新增留言
//
//	@Summary		新增留言
//	@Description	對貼文新增單層留言
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"貼文 ID"
//	@Param			request	body		feedContentBody	true	"留言內容"
//	@Success		201		{object}	model.FeedComment
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/api/v1/feed/posts/{id}/comments [post]
func (h *FeedHandler) AddComment(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var body feedContentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.feedService.AddComment(uint(postID), userID, body.Content)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// UpdateComment 更新留言
//
//	@Summary		更新留言
//	@Description	更新自己的留言內容
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"留言 ID"
//	@Param			request	body		feedContentBody	true	"留言內容"
//	@Success		200		{object}	model.FeedComment
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/api/v1/feed/comments/{id} [put]
func (h *FeedHandler) UpdateComment(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	commentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var body feedContentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.feedService.UpdateComment(uint(commentID), userID, body.Content)
	if err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, comment)
}

// DeleteComment 刪除留言
//
//	@Summary		刪除留言
//	@Description	刪除自己的留言
//	@Tags			feed
//	@Produce		json
//	@Param			id	path		int	true	"留言 ID"
//	@Success		200	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Router			/api/v1/feed/comments/{id} [delete]
func (h *FeedHandler) DeleteComment(c *gin.Context) {
	userID, ok := feedUserID(c)
	if !ok {
		return
	}

	commentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	if err := h.feedService.DeleteComment(uint(commentID), userID); err != nil {
		feedError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}
