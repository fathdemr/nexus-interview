package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration loaded from environment variables.
// Never read viper directly outside this package.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Swagger  SwaggerConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host    string
	Port    string
	User    string
	Pass    string
	Name    string
	SSLMode string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Pass, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr string
	Pass string
	DB   int
	TLS  bool
}

type JWTConfig struct {
	PrivateKey                   string
	PublicKey                    string
	AccessTokenExpirationMinutes int
	RefreshTokenExpirationDays   int
	CookieDomain                 string
	CookieSecure                 bool
}

type SwaggerConfig struct {
	Username string
	Password string
}

// Load reads configuration exclusively from environment variables.
// A .env file in the working directory is loaded as a fallback for local development;
// real environment variables always take precedence.
func Load() (Config, error) {
	viper.AutomaticEnv()

	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig() // .env is optional — missing file is not an error

	cfg := Config{
		Server: ServerConfig{
			Port: viper.GetString("SERVER_PORT"),
			Mode: viper.GetString("SERVER_MODE"),
		},
		Database: DatabaseConfig{
			Host:    viper.GetString("DB_HOST"),
			Port:    viper.GetString("DB_PORT"),
			User:    viper.GetString("DB_USER"),
			Pass:    viper.GetString("DB_PASS"),
			Name:    viper.GetString("DB_NAME"),
			SSLMode: viper.GetString("DB_SSLMODE"),
		},
		Redis: RedisConfig{
			Addr: viper.GetString("REDIS_ADDR"),
			Pass: viper.GetString("REDIS_PASS"),
			DB:   viper.GetInt("REDIS_DB"),
			TLS:  viper.GetBool("REDIS_TLS"),
		},
		JWT: JWTConfig{
			PrivateKey:                   normalizeKey(viper.GetString("PRIVATE_KEY")),
			PublicKey:                    normalizeKey(viper.GetString("PUBLIC_KEY")),
			AccessTokenExpirationMinutes: viper.GetInt("ACCESS_TOKEN_EXPIRATION_MINUTES"),
			RefreshTokenExpirationDays:   viper.GetInt("REFRESH_TOKEN_EXPIRATION_DAYS"),
			CookieDomain:                 viper.GetString("COOKIE_DOMAIN"),
			CookieSecure:                 viper.GetBool("COOKIE_SECURE"),
		},
		Swagger: SwaggerConfig{
			Username: viper.GetString("SWAGGER_USERNAME"),
			Password: viper.GetString("SWAGGER_PASS"),
		},
	}

	setDefaults(&cfg)

	return cfg, nil
}

// setDefaults fills in safe fallback values for optional fields.
func setDefaults(cfg *Config) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	if cfg.JWT.AccessTokenExpirationMinutes == 0 {
		cfg.JWT.AccessTokenExpirationMinutes = 15
	}
	if cfg.JWT.RefreshTokenExpirationDays == 0 {
		cfg.JWT.RefreshTokenExpirationDays = 7
	}
}

// normalizeKey converts escaped newlines in PEM keys stored as single-line env vars.
func normalizeKey(raw string) string {
	return strings.ReplaceAll(raw, `\n`, "\n")
}
