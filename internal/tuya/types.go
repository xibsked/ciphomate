package tuya

import (
	"ciphomate/internal/config"
	"net/http"
	"time"
)

type TokenCache struct {
	AccessToken string    `json:"access_token"`
	ExpireTime  time.Time `json:"expire_time"`
}

type TokenManager struct {
	Config    *config.Config
	Token     string
	ExpiresAt time.Time
	CachePath string
}

type TuyaClient struct {
	tokenManager *TokenManager
	config       *config.Config
	httpClient   *http.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpireTime   int    `json:"expire_time"`
	RefreshToken string `json:"refresh_token"`
	UID          string `json:"uid"`
}
