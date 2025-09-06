package supabase

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/train360-corp/supago/pkg/services/analytics"
	"github.com/train360-corp/supago/pkg/services/kong"
	"github.com/train360-corp/supago/pkg/services/meta"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/services/postgrest"
	"github.com/train360-corp/supago/pkg/services/realtime"
	"github.com/train360-corp/supago/pkg/services/studio"
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
	DashboardUsername     string
	DashboardPassword     string
	LogFlarePrivateKey    string
	LogFlarePublicKey     string
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
			DashboardUsername:     utils.RandomString(32),
			DashboardPassword:     utils.RandomString(32),
			LogFlarePrivateKey:    utils.RandomString(32),
			LogFlarePublicKey:     utils.RandomString(32),
		}, nil
	}
}

func GetServices(config *Config) (*[]types.Service, error) {

	var services []types.Service

	// add postgres
	if db, err := postgres.Service(config.DatabaseDataDirectory, config.DatabasePassword, config.JwtSecret); err != nil {
		return nil, fmt.Errorf("failed to construct postgrest service: %v", err)
	} else {
		services = append(services, *db)
	}

	// add kong
	if gateway, err := kong.Service(kong.Props{
		Keys: kong.Keys{
			Public:  config.PublicJwtKey,
			Private: config.PrivateJwtKey,
		},
		Dashboard: kong.Dashboard{
			Username: config.DashboardUsername,
			Password: config.DashboardPassword,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to construct gateway service: %v", err)
	} else {
		services = append(services, *gateway)
	}

	// add remaining services
	services = append(services,
		*analytics.Service(config.DatabasePassword, config.LogFlarePublicKey, config.LogFlarePrivateKey),
		*meta.Service(config.DatabasePassword),
		*postgrest.Service(config.DatabasePassword, config.JwtSecret),
		*realtime.Service(config.DatabasePassword, config.PublicJwtKey, config.JwtSecret),
		*studio.Service(studio.Props{
			Keys: studio.Keys{
				Public:  config.PublicJwtKey,
				Private: config.PrivateJwtKey,
				Secret:  config.JwtSecret,
			},
			Database: studio.Database{
				Password: config.DatabasePassword,
			},
			LogFlare: studio.LogFlare{
				Password: config.LogFlarePrivateKey,
			},
		}),
	)

	return &services, nil
}
