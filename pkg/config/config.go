package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration loaded from config.yaml.
// Add new fields here as the project grows; never read viper directly outside this package.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"db"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Swagger  SwaggerConfig  `mapstructure:"swagger"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	// Port is the TCP port the server listens on.
	// Example: "8080"
	Port string `mapstructure:"port"`

	// Mode controls Gin's run mode.
	// Values: "debug" | "release" | "test"
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	// Host is the database server address.
	// Example: "localhost" or "nexus-db.xxxx.rds.amazonaws.com"
	Host string `mapstructure:"host"`

	// Port is the database server port.
	// Example: "5432"
	Port string `mapstructure:"port"`

	// User is the PostgreSQL role used to authenticate.
	// Example: "nexus_user"
	User string `mapstructure:"user"`

	// Password is the PostgreSQL role password.
	Password string `mapstructure:"password"`

	// Name is the target database name.
	// Example: "nexus_interview"
	Name string `mapstructure:"name"`

	// SSLMode controls TLS behaviour for the connection.
	// Values: "disable" | "require" | "verify-full"
	SSLMode string `mapstructure:"sslmode"`
}

type SwaggerConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// DSN builds a PostgreSQL connection string from the config fields.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// JWTConfig holds RSA key material for token signing and verification.
type JWTConfig struct {
	PrivateKey string `mapstructure:"privateKey"`
	PublicKey  string `mapstructure:"publicKey"`

	// AccessTokenExpirationMinutes is the short-lived access token lifetime.
	// Example: 15
	AccessTokenExpirationMinutes int `mapstructure:"accessTokenExpirationMinutes"`

	// RefreshTokenExpirationDays is the long-lived refresh token lifetime.
	// Example: 7
	RefreshTokenExpirationDays int `mapstructure:"refreshTokenExpirationDays"`

	// CookieDomain is the domain attribute set on the access token cookie.
	// Example: "nexus-interview.com" — leave empty for localhost.
	CookieDomain string `mapstructure:"cookieDomain"`

	// CookieSecure controls the Secure flag on the access token cookie.
	// Set to true in production (HTTPS only).
	CookieSecure bool `mapstructure:"cookieSecure"`
}

// RedisConfig holds connection parameters for the Redis cache.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	// TLS enables TLS for the Redis connection.
	// Required for AWS ElastiCache Serverless — set to true in production.
	TLS bool `mapstructure:"tls"`
}

// Load reads config.yaml from the working directory and unmarshals it into Config.
func Load() (Config, error) {
	viper.SetConfigName("config.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	// Normalise PEM newlines — config.yaml may store them as literal \n
	cfg.JWT.PrivateKey = strings.ReplaceAll(cfg.JWT.PrivateKey, `\n`, "\n")
	cfg.JWT.PublicKey = strings.ReplaceAll(cfg.JWT.PublicKey, `\n`, "\n")

	if cfg.Server.Port == "" {
		cfg.Server.Port = "5075"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.JWT.AccessTokenExpirationMinutes == 0 {
		cfg.JWT.AccessTokenExpirationMinutes = 15
	}
	if cfg.JWT.RefreshTokenExpirationDays == 0 {
		cfg.JWT.RefreshTokenExpirationDays = 7
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}

	return cfg, nil
}
