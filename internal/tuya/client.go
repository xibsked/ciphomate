package tuya

import (
	"bytes"
	"ciphomate/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func NewTuyaClient(cfg *config.Config, tm *TokenManager) *TuyaClient {
	return &TuyaClient{
		tokenManager: tm,
		config:       cfg,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *TuyaClient) SendRequest(method, path string, body []byte) ([]byte, error) {
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, err
	}
	clientID := c.config.ClientID
	host := c.config.Host
	secret := c.config.Secret

	t := fmt.Sprint(time.Now().UnixNano() / 1e6)
	contentHash := Sha256(body)
	stringToSign := fmt.Sprintf("%s\n%s\n\n%s", method, contentHash, path)
	signStr := clientID + token + t + stringToSign
	sign := strings.ToUpper(HmacSha256(signStr, secret))

	req, _ := http.NewRequest(method, host+path, bytes.NewReader(body))
	req.Header.Set("client_id", clientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", t)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("access_token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func Sha256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func HmacSha256(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
