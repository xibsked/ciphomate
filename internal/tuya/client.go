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
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	Host     = ""
	ClientID = ""
	Secret   = ""
	DeviceID = ""
)

func Load(cfg *config.Config) {
	Host = cfg.Host
	ClientID = cfg.ClientID
	Secret = cfg.Secret
	DeviceID = cfg.DeviceID
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpireTime   int    `json:"expire_time"`
	RefreshToken string `json:"refresh_token"`
	UID          string `json:"uid"`
}

func GetToken() (TokenResponse, error) {
	path := "/v1.0/token?grant_type=1"
	t := fmt.Sprint(time.Now().UnixNano() / 1e6)
	contentHash := Sha256([]byte(""))
	stringToSign := fmt.Sprintf("GET\n%s\n\n%s", contentHash, path)
	signStr := ClientID + t + stringToSign
	sign := strings.ToUpper(HmacSha256(signStr, Secret))

	req, _ := http.NewRequest("GET", Host+path, nil)
	req.Header.Set("client_id", ClientID)
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

func SendRequest(method, path string, body []byte) ([]byte, error) {
	t := fmt.Sprint(time.Now().UnixNano() / 1e6)
	contentHash := Sha256(body)
	stringToSign := fmt.Sprintf("%s\n%s\n\n%s", method, contentHash, path)
	signStr := ClientID + Token + t + stringToSign
	sign := strings.ToUpper(HmacSha256(signStr, Secret))

	req, _ := http.NewRequest(method, Host+path, bytes.NewReader(body))
	req.Header.Set("client_id", ClientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", t)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("access_token", Token)
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
