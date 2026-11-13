package tuya

import (
	"ciphomate/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func NewTokenManager(cfg *config.Config, cachePath string) *TokenManager {
	return &TokenManager{
		Config:    cfg,
		CachePath: cachePath,
	}
}
func (tm *TokenManager) GetToken() (string, error) {
	if err := tm.loadTokenFromCache(); err == nil {
		if time.Now().Before(tm.ExpiresAt) {
			slog.Debug("using cached Tuya access token", "component", "tuya.auth", "expires_at", tm.ExpiresAt)
			return tm.Token, nil
		}
		slog.Debug("cached Tuya access token expired", "component", "tuya.auth", "expires_at", tm.ExpiresAt)
	} else {
		slog.Debug("Tuya token cache unavailable; requesting a token", "component", "tuya.auth", "error", err)
	}

	slog.Info("requesting Tuya access token", "component", "tuya.auth")
	resp, err := fetchToken(tm.Config)
	if err != nil {
		return "", err
	}

	tm.Token = resp.AccessToken
	tm.ExpiresAt = time.Now().Add(time.Duration(resp.ExpireTime) * time.Second)
	slog.Info("Tuya access token acquired", "component", "tuya.auth", "expires_at", tm.ExpiresAt)

	if err := tm.saveTokenToCache(); err != nil {
		slog.Warn("could not save Tuya token cache",
			"component", "tuya.auth",
			"cache_path", tm.CachePath,
			"error", err,
		)
	}

	return tm.Token, nil
}

func fetchToken(config *config.Config) (TokenResponse, error) {
	clientID := config.ClientID
	host := config.Host
	secret := config.Secret

	path := "/v1.0/token?grant_type=1"
	t := fmt.Sprint(time.Now().UnixNano() / 1e6)
	contentHash := Sha256([]byte(""))
	stringToSign := fmt.Sprintf("GET\n%s\n\n%s", contentHash, path)
	signStr := clientID + t + stringToSign
	sign := strings.ToUpper(HmacSha256(signStr, secret))

	req, err := http.NewRequest("GET", host+path, nil)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("create Tuya token request: %w", err)
	}
	req.Header.Set("client_id", clientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", t)
	req.Header.Set("sign_method", "HMAC-SHA256")

	client := &http.Client{Timeout: 10 * time.Second}
	startedAt := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("Tuya token request failed",
			"component", "tuya.auth",
			"duration", time.Since(startedAt),
			"error", err,
		)
		return TokenResponse{}, fmt.Errorf("Tuya token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("read Tuya token response: %w", err)
	}
	slog.Debug("Tuya token request completed",
		"component", "tuya.auth",
		"http_status", resp.StatusCode,
		"duration", time.Since(startedAt),
	)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TokenResponse{}, fmt.Errorf("Tuya token endpoint returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Result TokenResponse `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return TokenResponse{}, fmt.Errorf("decode Tuya token response: %w", err)
	}
	if result.Result.AccessToken == "" {
		return TokenResponse{}, fmt.Errorf("Tuya token response did not contain an access token")
	}
	return result.Result, nil
}

func (tm *TokenManager) loadTokenFromCache() error {
	data, err := os.ReadFile(tm.CachePath)
	if err != nil {
		return err
	}

	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}

	tm.Token = cache.AccessToken
	tm.ExpiresAt = cache.ExpireTime
	return nil
}

func (tm *TokenManager) saveTokenToCache() error {
	cache := TokenCache{
		AccessToken: tm.Token,
		ExpireTime:  tm.ExpiresAt,
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tm.CachePath, data, 0600); err != nil {
		return err
	}
	return os.Chmod(tm.CachePath, 0600)
}
