package supabase

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/train360-corp/supago/pkg/services/meta"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
	"github.com/train360-corp/supago/pkg/utils"
	"time"
)

type Config struct {
	DatabaseDataDirectory string
	DatabasePassword      string
	JwtSecret             string
	PublicJwtKey          string
	PrivateJwtKey         string
}

// GetJwts returns deterministic JWTs configured for supabase based on a fixed secret
func GetJwts(jwtSecret string) (*struct {
	Public  string
	Private string
}, error) {

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

	return &struct {
		Public  string
		Private string
	}{Public: *anonKey, Private: *serviceKey}, nil
}

func GetRandomConfig() (*Config, error) {
	jwtSecret := utils.RandomString(32)
	if jwts, err := GetJwts(jwtSecret); err != nil {
		return nil, err
	} else {
		return &Config{
			DatabaseDataDirectory: utils.GetTempDir(),
			DatabasePassword:      utils.RandomString(32),
			JwtSecret:             jwtSecret,
			PublicJwtKey:          jwts.Public,
			PrivateJwtKey:         jwts.Private,
		}, nil
	}
}

func GetServices(config *Config) (*[]types.Service, error) {

	var services []types.Service

	// add postgres
	if db, err := postgres.Service(config.DatabaseDataDirectory, config.DatabasePassword, config.JwtSecret); err != nil {
		return nil, err
	} else {
		services = append(services, *db)
	}

	// add meta
	services = append(services, *meta.Service(config.DatabasePassword))

	return &services, nil
}
