package config

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 應用程式配置結構
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
	Minio    MinioConfig    `mapstructure:"minio"`
	LiveKit  LiveKitConfig  `mapstructure:"livekit"`
	DeepL    DeepLConfig    `mapstructure:"deepl"`
	LLM      LLMConfig      `mapstructure:"llm"`
}

// DeepLConfig DeepL 翻譯 API 配置
type DeepLConfig struct {
	APIKey        string `mapstructure:"api_key"`
	APIURL        string `mapstructure:"api_url"` // 預設 Free: https://api-free.deepl.com/v2
	Enabled       bool   `mapstructure:"enabled"`
	AutoTranslate bool   `mapstructure:"auto_translate"` // true = 發訊息時自動翻譯；false（預設）= 使用者點擊翻譯才觸發
}

// LLMConfig LLM 語意評估配置
type LLMConfig struct {
	Provider            string  `mapstructure:"provider"` // gemini, groq
	APIKey              string  `mapstructure:"api_key"`
	Model               string  `mapstructure:"model"`                // e.g. gemini-1.5-flash, llama3-8b-8192
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"` // 0..1, 預設 0.70
	Enabled             bool    `mapstructure:"enabled"`
}

// OAuthConfig OAuth 配置
type OAuthConfig struct {
	Google GoogleOAuthConfig `mapstructure:"google"`
}

// GoogleOAuthConfig Google OAuth 配置
type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// ServerConfig 伺服器配置
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	// TrustedProxies 反向代理的 CIDR 清單。空值＝只信 RemoteAddr。
	// 千萬別填 0.0.0.0/0：ClientIP() 會改讀 X-Forwarded-For，
	// 任何人都能偽造來源 IP 繞過 IP 限流。
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 分鐘
	LogMode         bool   `mapstructure:"log_mode"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// MinioConfig Minio 物件儲存配置
type MinioConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	PublicEndpoint  string `mapstructure:"public_endpoint"` // 公開 URL 前綴（留空則沿用 Endpoint）
	AccessKey       string `mapstructure:"access_key"`
	SecretKey       string `mapstructure:"secret_key"`
	Bucket          string `mapstructure:"bucket"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	PresignExpiry   int    `mapstructure:"presign_expiry"`     // 分鐘，預設 15
	PublicRead      bool   `mapstructure:"public_read"`        // true = bucket 設為公開讀取，URL 無 signature
	MaxFileSizeMB   int64  `mapstructure:"max_file_size_mb"`   // 單檔上限，預設 50MB
	DailyUploadMax  int    `mapstructure:"daily_upload_max"`   // 每日上傳次數上限，預設 20
	DailyBytesMaxMB int64  `mapstructure:"daily_bytes_max_mb"` // 每日上傳總量上限 MB，預設 500
	FileTTLDays     int    `mapstructure:"file_ttl_days"`      // 檔案 TTL（天），0 = 永久
	LRUEvictEnabled bool   `mapstructure:"lru_evict_enabled"`  // 開啟 LRU 清理
}

// LiveKitConfig LiveKit 語音伺服器配置
type LiveKitConfig struct {
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	URL       string `mapstructure:"url"`        // LiveKit Server WebSocket URL，例如 ws://livekit:7880
	PublicURL string `mapstructure:"public_url"` // 前端連線用的公開 URL，例如 ws://localhost:7880
	TokenTTL  int    `mapstructure:"token_ttl"`  // Token 有效秒數，預設 3600
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	ExpirationHours int    `mapstructure:"expiration_hours"`
}

// LogConfig 日誌配置
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// defaultJWTSecret 僅供本機開發；release 模式下必須覆寫（見 Load）。
const (
	defaultJWTSecret = "your-secret-key-change-this-in-production"
	minJWTSecretLen  = 32
)

// usableJWTSecret 判斷 secret 是否真的被設過。
// 正常部署路徑不會走到這裡失敗：deploy.yml 先 `test -n "$JWT_SECRET"` 再
// envsubst 展開 configs/config.prod.yaml，viper 讀到的已是實際值。
// 這道檢查擋的是繞過該流程的情況——envsubst 把未設定的變數展成空字串、
// 或有人直接把 `secret: ${JWT_SECRET}` 原樣餵給 viper（viper 不展開 ${}）。
func usableJWTSecret(secret string) bool {
	return secret != defaultJWTSecret &&
		!strings.Contains(secret, "${") &&
		len(secret) >= minJWTSecretLen
}

// Load 載入配置檔案
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 設定預設值
	setDefaults()

	// 讀取環境變數
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// 如果找不到配置檔案，使用預設值
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// 開發預設值一旦被帶進正式環境，等於任何人都能自簽 token；寧可開不起來。
	if config.Server.Mode == "release" && !usableJWTSecret(config.JWT.Secret) {
		return nil, errors.New("config: jwt.secret must be overridden when server.mode=release")
	}

	return &config, nil
}

// setDefaults 設定預設配置值
func setDefaults() {
	// Server 預設值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", 10*time.Second)
	viper.SetDefault("server.write_timeout", 10*time.Second)
	viper.SetDefault("server.idle_timeout", 60*time.Second)
	// 只信任私有網段的代理（nginx / k8s ingress 都在這些網段內）。
	// 從公網直連的請求 RemoteAddr 不在其中，偽造的 X-Forwarded-For 會被忽略；
	// 走 nginx 進來的則正常還原真實 client IP，IP 限流才不會全站共用一個桶子。
	viper.SetDefault("server.trusted_proxies", []string{
		"127.0.0.1/32", "::1/128",
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7",
	})

	// Database 預設值
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "talkrealm")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", 60) // 60 分鐘
	viper.SetDefault("database.log_mode", false)

	// Redis 預設值
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// JWT 預設值
	viper.SetDefault("jwt.secret", defaultJWTSecret)
	// 單位是「小時」，不是 Duration：填 24*time.Hour 會變成 864 億小時。
	viper.SetDefault("jwt.expiration_hours", 24)

	// Log 預設值
	viper.SetDefault("log.level", "info")

	// OAuth 預設值
	viper.SetDefault(
		"oauth.google.redirect_url",
		"http://localhost:8080/api/v1/auth/google/callback",
	)

	// LiveKit 預設值
	viper.SetDefault("livekit.url", "ws://localhost:7880")
	viper.SetDefault("livekit.public_url", "ws://localhost:7880")
	viper.SetDefault("livekit.token_ttl", 3600)

	// DeepL 預設值
	viper.SetDefault("deepl.api_url", "https://api-free.deepl.com/v2")
	viper.SetDefault("deepl.enabled", false)

	// LLM 預設值
	viper.SetDefault("llm.provider", "gemini")
	viper.SetDefault("llm.model", "gemini-1.5-flash")
	viper.SetDefault("llm.similarity_threshold", 0.70)
	viper.SetDefault("llm.enabled", false)

	// Minio 預設值
	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.bucket", "talkrealm")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.presign_expiry", 15)
	viper.SetDefault("minio.public_read", false)
	viper.SetDefault("minio.max_file_size_mb", 50)
	viper.SetDefault("minio.daily_upload_max", 20)
	viper.SetDefault("minio.daily_bytes_max_mb", 500)
	viper.SetDefault("minio.file_ttl_days", 0)
	viper.SetDefault("minio.lru_evict_enabled", true)
}
