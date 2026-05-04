package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/service"
	"github.com/walnut-almonds/talkrealm/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthStateCookieName = "oauth_state"
	googleUserInfoURL    = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// OAuthHandler 處理 OAuth 登入流程
type OAuthHandler struct {
	userService  service.UserService
	googleConfig *oauth2.Config
	frontendURL  string
}

// NewOAuthHandler 建立 OAuthHandler
func NewOAuthHandler(userService service.UserService, cfg *config.Config) *OAuthHandler {
	googleCfg := &oauth2.Config{
		ClientID:     cfg.OAuth.Google.ClientID,
		ClientSecret: cfg.OAuth.Google.ClientSecret,
		RedirectURL:  cfg.OAuth.Google.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return &OAuthHandler{
		userService:  userService,
		googleConfig: googleCfg,
		frontendURL:  "",
	}
}

// GoogleLogin 重導向到 Google OAuth 授權頁面
//
//	@Summary	Google OAuth 登入
//	@Tags		auth
//	@Success	302
//	@Router		/api/v1/auth/google [get]
func (h *OAuthHandler) GoogleLogin(c *gin.Context) {
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// 將 state 存入 cookie，供 callback 驗證（防 CSRF）
	c.SetCookie(oauthStateCookieName, state, 300, "/", "", false, true)

	url := h.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback 處理 Google OAuth 回調
//
//	@Summary	Google OAuth Callback
//	@Tags		auth
//	@Success	302
//	@Router		/api/v1/auth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	// 驗證 state 防 CSRF
	cookieState, err := c.Cookie(oauthStateCookieName)
	if err != nil || cookieState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	// 清除 state cookie
	c.SetCookie(oauthStateCookieName, "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	// 交換 token
	token, err := h.googleConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to exchange token: " + err.Error()},
		)

		return
	}

	// 取得 Google 使用者資訊
	googleUser, err := fetchGoogleUserInfo(c.Request.Context(), token.AccessToken)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get user info: " + err.Error()},
		)

		return
	}

	// 登入或自動建立帳號
	resp, err := h.userService.OAuthLoginOrRegister(&service.OAuthUserInfo{
		Provider:   "google",
		ProviderID: googleUser.ID,
		Email:      googleUser.Email,
		Name:       googleUser.Name,
		Avatar:     googleUser.Picture,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oauth login failed: " + err.Error()})
		return
	}

	// 重導向前端並帶上 token（前端可以從 query string 讀取後存入 localStorage）
	frontendURL := h.frontendURL
	if frontendURL == "" {
		frontendURL = getRequestOrigin(c.Request)
	}

	redirectURL := fmt.Sprintf("%s/?access_token=%s&refresh_token=%s",
		frontendURL,
		resp.AccessToken,
		resp.RefreshToken,
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func getRequestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// googleUserInfo Google userinfo 回應
type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := func() ([]byte, error) {
		defer resp.Body.Close() //nolint:errcheck
		return io.ReadAll(resp.Body)
	}()
	if err != nil {
		return nil, err
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
