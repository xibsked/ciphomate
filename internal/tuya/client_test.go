package tuya

import (
	"bytes"
	"ciphomate/internal/config"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSendRequestReportsHTTPStatusWithoutLoggingResponseBody(t *testing.T) {
	const responseBody = "sensitive provider response body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	cachePath := writeTestTokenCache(t)
	var output bytes.Buffer
	setTestLogger(t, &output)

	cfg := &config.Config{Host: server.URL, ClientID: "test-client", Secret: "test-secret"}
	manager := NewTokenManager(cfg, cachePath)
	client := NewTuyaClient(cfg, manager)

	_, err := client.SendRequest(http.MethodGet, "/status", nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("SendRequest() error = %v, want HTTP 401 error", err)
	}
	if strings.Contains(err.Error(), responseBody) {
		t.Fatalf("response body included in returned error: %v", err)
	}
	if strings.Contains(output.String(), responseBody) {
		t.Fatalf("response body leaked into logs: %s", output.String())
	}
	if !strings.Contains(output.String(), "http_status=401") {
		t.Errorf("HTTP status missing from diagnostics: %s", output.String())
	}
}

func TestSendRequestReportsTuyaFailureWithoutLoggingResponseBody(t *testing.T) {
	const responseBody = `{"success":false,"code":1010,"msg":"sensitive provider response body"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	cachePath := writeTestTokenCache(t)
	var output bytes.Buffer
	setTestLogger(t, &output)

	cfg := &config.Config{Host: server.URL, ClientID: "test-client", Secret: "test-secret"}
	manager := NewTokenManager(cfg, cachePath)
	client := NewTuyaClient(cfg, manager)

	_, err := client.SendRequest(http.MethodPost, "/commands", []byte(`{"commands":[]}`))
	if err == nil || !strings.Contains(err.Error(), "success=false") {
		t.Fatalf("SendRequest() error = %v, want Tuya API failure", err)
	}
	if strings.Contains(err.Error(), responseBody) || strings.Contains(output.String(), responseBody) {
		t.Fatalf("provider response leaked into error or logs: %v\n%s", err, output.String())
	}
}

func writeTestTokenCache(t *testing.T) string {
	t.Helper()

	cachePath := filepath.Join(t.TempDir(), "token.json")
	cache, err := json.Marshal(TokenCache{
		AccessToken: "test-access-token",
		ExpireTime:  time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, cache, 0600); err != nil {
		t.Fatal(err)
	}
	return cachePath
}

func setTestLogger(t *testing.T, output *bytes.Buffer) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}
