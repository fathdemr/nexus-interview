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
	JWT      JWTConfig      `mapstructure:"jwt"`
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

// DSN builds a PostgreSQL connection string from the config fields.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// JWTConfig holds RSA key material for token signing and verification.
type JWTConfig struct {
	// PrivateKey is the RSA private key in PEM format used to sign tokens.
	// Newline sequences (\n) are normalised automatically on load.
	PrivateKey string `mapstructure:"privateKey"`

	// PublicKey is the RSA public key in PEM format used to verify tokens.
	// Newline sequences (\n) are normalised automatically on load.
	PublicKey string `mapstructure:"publicKey"`

	// ExpirationHours is the token lifetime in hours.
	// Example: 24
	ExpirationHours int `mapstructure:"expirationHours"`
}

// Load reads config.yaml from the working directory and unmarshals it into Config.
func Load() (Config, error) {
	viper.SetConfigName("config")
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
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.JWT.ExpirationHours == 0 {
		cfg.JWT.ExpirationHours = 24
	}

	return cfg, nil
}
