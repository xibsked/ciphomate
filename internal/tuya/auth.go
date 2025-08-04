package tuya

import (
	"ciphomate/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	// Try to load from cache
	if err := tm.loadTokenFromCache(); err == nil && time.Now().Before(tm.ExpiresAt) {
		log.Println("✅ Using cached Tuya token.")
		return tm.Token, nil
	}

	log.Println("🔁 Fetching new Tuya token...")
	resp, err := fetchToken(tm.Config)
	if err != nil {
		return "", err
	}

	tm.Token = resp.AccessToken
	tm.ExpiresAt = time.Now().Add(time.Duration(resp.ExpireTime) * time.Second)

	// Save to cache
	_ = tm.saveTokenToCache()

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

	req, _ := http.NewRequest("GET", host+path, nil)
	req.Header.Set("client_id", clientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", t)
	req.Header.Set("sign_method", "HMAC-SHA256")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Println("Token API Response:", string(body))

	var result struct {
		Result TokenResponse `json:"result"`
	}
	err = json.Unmarshal(body, &result)
	return result.Result, err
}

func (tm *TokenManager) loadTokenFromCache() error {
	data, err := ioutil.ReadFile(tm.CachePath)
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
	return ioutil.WriteFile(tm.CachePath, data, 0644)
}
