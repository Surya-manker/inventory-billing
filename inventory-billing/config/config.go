package config

import "github.com/spf13/viper"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Redis    RedisConfig
	GST      GSTConfig
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

type GSTConfig struct {
	SellerName    string
	SellerGSTIN   string // e.g. "29ABCDE1234F1Z5"
	SellerAddress string
	StateCode     string // first 2 digits of GSTIN, used for intra/inter-state detection
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
	viper.SetDefault("GST_SELLER_NAME", "My Business")
	viper.SetDefault("GST_STATE_CODE", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	return &Config{
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
		GST: GSTConfig{
			SellerName:    viper.GetString("GST_SELLER_NAME"),
			SellerGSTIN:   viper.GetString("GST_SELLER_GSTIN"),
			SellerAddress: viper.GetString("GST_SELLER_ADDRESS"),
			StateCode:     viper.GetString("GST_STATE_CODE"),
		},
	}, nil
}
