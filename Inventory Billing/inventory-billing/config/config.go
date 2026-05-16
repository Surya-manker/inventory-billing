package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Redis    RedisConfig
	GST      GSTConfig
	Worker   WorkerConfig
	Mail     MailConfig
	Storage  StorageConfig
}

type ServerConfig struct {
	Address string
}

type DatabaseConfig struct {
	DSN                string
	Debug              bool
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
}

type JWTConfig struct {
	Secret                string
	AccessTokenExpiryHour int // short-lived access token (default: 1 hour)
	RefreshTokenExpiryDay int // long-lived refresh token stored in Redis (default: 7 days)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type WorkerConfig struct {
	Concurrency int // number of concurrent Asynq worker goroutines
}

type MailConfig struct {
	Provider string // "smtp" | "noop"
	From     string // display address, e.g. "Acme Billing <noreply@acme.com>"
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	UseTLS   bool
}

type StorageConfig struct {
	Provider string // "local" | "s3"
	LocalDir string // base directory for LocalStorage
	LocalURL string // public URL prefix served by the static file handler
}

type GSTConfig struct {
	SellerName    string
	SellerGSTIN   string // e.g. "29ABCDE1234F1Z5"
	SellerAddress string
	StateCode     string // first 2 digits of GSTIN, used for intra/inter-state detection
}

// validate checks that all required fields are non-empty.
// Called right after the config is assembled so the process fails fast with a
// clear message instead of panicking later with a cryptic "token is empty" error.
func (c *Config) validate() error {
	var errs []string
	if c.Database.DSN == "" {
		errs = append(errs, "DATABASE_DSN is required")
	}
	if c.JWT.Secret == "" {
		errs = append(errs, "JWT_SECRET is required")
	}
	if c.Redis.Addr == "" {
		errs = append(errs, "REDIS_ADDR is required")
	}
	if len(errs) > 0 {
		return errors.New("missing required config:\n  - " + strings.Join(errs, "\n  - "))
	}
	return nil
}

func Load() (*Config, error) {
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_ADDRESS", ":8080")
	viper.SetDefault("DATABASE_DEBUG", false)
	viper.SetDefault("DATABASE_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DATABASE_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DATABASE_CONN_MAX_LIFETIME_MIN", 30)
	viper.SetDefault("JWT_ACCESS_EXPIRY_HOUR", 1)
	viper.SetDefault("JWT_REFRESH_EXPIRY_DAY", 7)
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("WORKER_CONCURRENCY", 10)
	viper.SetDefault("MAIL_PROVIDER", "noop")
	viper.SetDefault("MAIL_SMTP_PORT", 587)
	viper.SetDefault("MAIL_USE_TLS", false)
	viper.SetDefault("STORAGE_PROVIDER", "local")
	viper.SetDefault("STORAGE_LOCAL_DIR", "./data/assets")
	viper.SetDefault("STORAGE_LOCAL_URL", "http://localhost:8080/assets")
	viper.SetDefault("GST_SELLER_NAME", "My Business")
	viper.SetDefault("GST_STATE_CODE", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Address: viper.GetString("SERVER_ADDRESS"),
		},
		Database: DatabaseConfig{
			DSN:                viper.GetString("DATABASE_DSN"),
			Debug:              viper.GetBool("DATABASE_DEBUG"),
			MaxOpenConns:       viper.GetInt("DATABASE_MAX_OPEN_CONNS"),
			MaxIdleConns:       viper.GetInt("DATABASE_MAX_IDLE_CONNS"),
			ConnMaxLifetimeMin: viper.GetInt("DATABASE_CONN_MAX_LIFETIME_MIN"),
		},
		JWT: JWTConfig{
			Secret:                viper.GetString("JWT_SECRET"),
			AccessTokenExpiryHour: viper.GetInt("JWT_ACCESS_EXPIRY_HOUR"),
			RefreshTokenExpiryDay: viper.GetInt("JWT_REFRESH_EXPIRY_DAY"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Worker: WorkerConfig{
			Concurrency: viper.GetInt("WORKER_CONCURRENCY"),
		},
		Mail: MailConfig{
			Provider: viper.GetString("MAIL_PROVIDER"),
			From:     viper.GetString("MAIL_FROM"),
			SMTPHost: viper.GetString("MAIL_SMTP_HOST"),
			SMTPPort: viper.GetInt("MAIL_SMTP_PORT"),
			SMTPUser: viper.GetString("MAIL_SMTP_USER"),
			SMTPPass: viper.GetString("MAIL_SMTP_PASS"),
			UseTLS:   viper.GetBool("MAIL_USE_TLS"),
		},
		Storage: StorageConfig{
			Provider: viper.GetString("STORAGE_PROVIDER"),
			LocalDir: viper.GetString("STORAGE_LOCAL_DIR"),
			LocalURL: viper.GetString("STORAGE_LOCAL_URL"),
		},
		GST: GSTConfig{
			SellerName:    viper.GetString("GST_SELLER_NAME"),
			SellerGSTIN:   viper.GetString("GST_SELLER_GSTIN"),
			SellerAddress: viper.GetString("GST_SELLER_ADDRESS"),
			StateCode:     viper.GetString("GST_STATE_CODE"),
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
