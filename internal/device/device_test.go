package device

import (
	"bytes"
	"ciphomate/internal/config"
	"ciphomate/internal/tuya"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchInchingTimeUsesDefaultWhenStatusIsUnavailable(t *testing.T) {
	var logOutput bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	tests := []struct {
		name            string
		statusJSON      string
		expectedMinutes int
	}{
		{
			name:            "empty value",
			statusJSON:      `{"result":[{"code":"switch_inching","value":""}]}`,
			expectedMinutes: 180,
		},
		{
			name:            "malformed value",
			statusJSON:      `{"result":[{"code":"switch_inching","value":"not-base64"}]}`,
			expectedMinutes: 180,
		},
		{
			name:            "missing status item",
			statusJSON:      `{"result":[]}`,
			expectedMinutes: 180,
		},
		{
			name:            "valid value",
			statusJSON:      `{"result":[{"code":"switch_inching","value":"AQEs"}]}`,
			expectedMinutes: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOutput.Reset()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v1.0/token":
					_, _ = w.Write([]byte(`{"result":{"access_token":"test-token","expire_time":3600}}`))
				case "/v1.0/devices/pump/status":
					_, _ = w.Write([]byte(tt.statusJSON))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Host:         server.URL,
				ClientID:     "test-client",
				Secret:       "test-secret",
				PumpDeviceID: "pump",
			}
			tokenManager := tuya.NewTokenManager(cfg, filepath.Join(t.TempDir(), "token.json"))
			device := NewDevice(tuya.NewTuyaClient(cfg, tokenManager), cfg.PumpDeviceID)

			got, err := device.FetchInchingTime()
			if err != nil {
				t.Fatalf("FetchInchingTime() error = %v", err)
			}
			if got != tt.expectedMinutes {
				t.Fatalf("FetchInchingTime() = %d, want %d", got, tt.expectedMinutes)
			}
			if strings.Contains(logOutput.String(), tt.statusJSON) {
				t.Errorf("raw status response leaked into logs: %s", logOutput.String())
			}
		})
	}
}
