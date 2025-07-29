package tuya

import (
	"ciphomate/internal/config"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
)

type TokenCache struct {
	AccessToken string    `json:"access_token"`
	ExpireTime  time.Time `json:"expire_time"`
}

var (
	Token           string
	tokenCachePath  = "token_cache.json"
	tokenValidUntil time.Time
)

// InitAuth initializes and caches token
func InitAuth(config *config.Config) {
	Load(config)
	if tryLoadTokenFromCache() {
		log.Println("✅ Using cached Tuya token.")
		return
	}

	log.Println("🔁 Fetching new Tuya token...")
	resp, err := GetToken()
	if err != nil {
		log.Fatalf("Failed to fetch Tuya token: %v", err)
	}

	Token = resp.AccessToken
	tokenValidUntil = time.Now().Add(time.Duration(resp.ExpireTime) * time.Second)

	cache := TokenCache{
		AccessToken: Token,
		ExpireTime:  tokenValidUntil,
	}
	data, _ := json.Marshal(cache)
	_ = ioutil.WriteFile(tokenCachePath, data, 0644)
}

func tryLoadTokenFromCache() bool {
	data, err := ioutil.ReadFile(tokenCachePath)
	if err != nil {
		return false
	}

	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return false
	}

	if time.Now().After(cache.ExpireTime) {
		log.Println("⚠️ Tuya token expired.")
		return false
	}

	Token = cache.AccessToken
	tokenValidUntil = cache.ExpireTime
	return true
}
