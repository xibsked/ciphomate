package tuya

import (
	"bytes"
	"ciphomate/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
		return nil, fmt.Errorf("get Tuya access token: %w", err)
	}
	clientID := c.config.ClientID
	host := c.config.Host
	secret := c.config.Secret

	t := fmt.Sprint(time.Now().UnixNano() / 1e6)
	contentHash := Sha256(body)
	stringToSign := fmt.Sprintf("%s\n%s\n\n%s", method, contentHash, path)
	signStr := clientID + token + t + stringToSign
	sign := strings.ToUpper(HmacSha256(signStr, secret))

	req, err := http.NewRequest(method, host+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Tuya %s request: %w", method, err)
	}
	req.Header.Set("client_id", clientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", t)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("access_token", token)
	req.Header.Set("Content-Type", "application/json")

	startedAt := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("Tuya request failed",
			"component", "tuya",
			"method", method,
			"path", path,
			"duration", time.Since(startedAt),
			"error", err,
		)
		return nil, fmt.Errorf("Tuya %s %s request failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read Tuya response",
			"component", "tuya",
			"method", method,
			"path", path,
			"http_status", resp.StatusCode,
			"duration", time.Since(startedAt),
			"error", err,
		)
		return nil, fmt.Errorf("read Tuya %s %s response: %w", method, path, err)
	}

	slog.Debug("Tuya request completed",
		"component", "tuya",
		"method", method,
		"path", path,
		"http_status", resp.StatusCode,
		"duration", time.Since(startedAt),
	)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Tuya %s %s returned HTTP %d", method, path, resp.StatusCode)
	}
	var apiResponse struct {
		Success *bool `json:"success"`
	}
	if err := json.Unmarshal(responseBody, &apiResponse); err == nil && apiResponse.Success != nil && !*apiResponse.Success {
		slog.Warn("Tuya API reported request failure",
			"component", "tuya",
			"method", method,
			"path", path,
			"http_status", resp.StatusCode,
		)
		return nil, fmt.Errorf("Tuya %s %s reported success=false", method, path)
	}
	return responseBody, nil
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
