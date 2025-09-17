package supago

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/train360-corp/supago/internal/services/kong"
	"github.com/train360-corp/supago/internal/utils"
	"os"
	"path/filepath"
	"time"
)

type KeysConfig struct {
	JwtSecret          string
	PublicJwt          string
	PrivateJwt         string
	PgSodiumEncryption string
}

type StorageConfig struct {
	DataDirectory string
}

type DashboardConfig struct {
	Username string
	Password string
}

type DatabaseConfig struct {
	DataDirectory string
	Password      string
}

type LogFlareConfig struct {
	PrivateKey string
	PublicKey  string
}

type KongSMTPFromConfig struct {
	Email string
	Name  string
}

type KongSMTPConfig struct {
	Host string
	Port uint16
	User string
	Pass string
	From KongSMTPFromConfig
}

type KongURLsConfig struct {
	Site string // where the frontend Site is publicly accessible
	Kong string // where Kong is publicly accessible
}

type KongConfig struct {
	URLs KongURLsConfig
	SMTP KongSMTPConfig
}

type GlobalConfig struct {
	PlatformName string
	DebugMode    bool
}

type Config struct {
	Global    GlobalConfig
	Database  DatabaseConfig
	Storage   StorageConfig
	Dashboard DashboardConfig
	LogFlare  LogFlareConfig
	Keys      KeysConfig
	Kong      KongConfig
}

// NewRandomConfigE like NewRandomConfig, but returns an error instead of panic-ing
func NewRandomConfigE(platformName string, keyGetter EncryptionKeyGetter) (*Config, error) {
	jwtSecret := utils.RandomString(32)

	keys, err := GetJwtKeysConfig(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to construct jwt keys config: %v", err)
	}

	// patch encryption key
	if encryptionKey, err := keyGetter(); err != nil {
		return nil, err
	} else {
		keys.PgSodiumEncryption = encryptionKey
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %v", err)
	}

	return &Config{
		Global: GlobalConfig{
			PlatformName: platformName,
			DebugMode:    false,
		},
		Keys: *keys,
		Database: DatabaseConfig{
			DataDirectory: filepath.Join(wd, "postgres", "data"),
			Password:      utils.RandomString(32),
		},
		Storage: StorageConfig{
			DataDirectory: filepath.Join(wd, "storage", "data"),
		},
		Dashboard: DashboardConfig{
			Username: utils.RandomString(32),
			Password: utils.RandomString(32),
		},
		LogFlare: LogFlareConfig{
			PrivateKey: utils.RandomString(32),
			PublicKey:  utils.RandomString(32),
		},
		Kong: KongConfig{
			URLs: KongURLsConfig{
				Site: "http://127.0.0.1:3000",
				Kong: fmt.Sprintf("http://%s:8000", containerName(Config{Global: GlobalConfig{PlatformName: platformName}}, kong.ContainerName)),
			},
			SMTP: KongSMTPConfig{
				Host: "supabase-mail",
				Port: 2500,
				User: "fake_mail_user",
				Pass: "fake_mail_password",
				From: KongSMTPFromConfig{
					Email: "admin@example.com",
					Name:  "fake_sender",
				},
			},
		},
	}, nil
}

// NewRandomConfig generates a Config object using random values
// Safe base-config to customize from
func NewRandomConfig(platformName string, keyGetter EncryptionKeyGetter) *Config {
	cfg, err := NewRandomConfigE(platformName, keyGetter)
	if err != nil {
		panic(err)
	}
	return cfg
}

// GetJwtKeysConfig returns deterministic JWTs (as a KeysConfig) pre-configured for Supabase, based on a fixed secret
func GetJwtKeysConfig(jwtSecret string) (*KeysConfig, error) {

	// Fixed issued-at: 01 Jan 2025 00:00:00 UTC
	issuedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	// Expiration: 20 years later
	expiresAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(20, 0, 0).Unix()

	makeToken := func(role string) (*string, error) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"role": role,
			"iss":  "supabase",
			"iat":  issuedAt,
			"exp":  expiresAt,
		})
		signed, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			return nil, fmt.Errorf("failed to sign %s JWT: %v", role, err)
		}
		return &signed, nil
	}

	anonKey, err := makeToken("anon")
	if err != nil {
		return nil, err
	}
	serviceKey, err := makeToken("service_role")
	if err != nil {
		return nil, err
	}

	return &KeysConfig{
		JwtSecret:  jwtSecret,
		PublicJwt:  *anonKey,
		PrivateJwt: *serviceKey,
	}, nil
}
