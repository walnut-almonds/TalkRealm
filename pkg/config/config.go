package config

import (
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
}

// ServerConfig 伺服器配置
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
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

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	ExpireTime time.Duration `mapstructure:"expire_time"`
}

// LogConfig 日誌配置
type LogConfig struct {
	Level string `mapstructure:"level"`
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
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// 如果找不到配置檔案，使用預設值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
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
	viper.SetDefault("jwt.secret", "your-secret-key-change-this-in-production")
	viper.SetDefault("jwt.expire_time", 24*time.Hour)

	// Log 預設值
	viper.SetDefault("log.level", "info")
}
