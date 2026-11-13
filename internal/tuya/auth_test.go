package tuya

import (
	"bytes"
	"ciphomate/internal/config"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTokenDoesNotLogTokenResponse(t *testing.T) {
	const accessToken = "access-token-must-not-be-logged"
	const refreshToken = "refresh-token-must-not-be-logged"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken + `","expire_time":3600}}`))
	}))
	defer server.Close()

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	cachePath := filepath.Join(t.TempDir(), "token.json")
	manager := NewTokenManager(&config.Config{
		Host:     server.URL,
		ClientID: "test-client",
		Secret:   "test-secret",
	}, cachePath)

	token, err := manager.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if token != accessToken {
		t.Fatalf("GetToken() = %q, want access token", token)
	}

	logs := output.String()
	if strings.Contains(logs, accessToken) || strings.Contains(logs, refreshToken) {
		t.Fatalf("token data leaked into logs: %s", logs)
	}
	if !strings.Contains(logs, "Tuya access token acquired") {
		t.Fatalf("success event missing from logs: %s", logs)
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat token cache: %v", err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Errorf("token cache permissions = %o, want no group/other access", info.Mode().Perm())
	}
}
