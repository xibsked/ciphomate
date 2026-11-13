package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ciphomate/internal/config"
)

func TestShutdownDuringStartupSendsOffAfterCanceledOn(t *testing.T) {
	runShutdownTest(t, true, true)
}

func TestShutdownDuringMonitoringCancelsSchedulerBeforeOff(t *testing.T) {
	runShutdownTest(t, false, true)
}

func TestShutdownReportsTuyaOffFailure(t *testing.T) {
	runShutdownTest(t, false, false)
}

func runShutdownTest(t *testing.T, blockOn, offSucceeds bool) {
	t.Helper()

	var (
		mu     sync.Mutex
		events []bool
	)
	onRequest := make(chan struct{})
	var onOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0/token":
			writeJSON(t, w, map[string]any{
				"result": map[string]any{"access_token": "test-token", "expire_time": 3600},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1.0/devices/tank/status":
			writeJSON(t, w, map[string]any{
				"result": []map[string]any{{"code": "cur_current", "value": 0}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1.0/devices/pump/status":
			writeJSON(t, w, map[string]any{
				"result": []map[string]any{{"code": "switch_inching", "value": "AQEs"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1.0/devices/pump/commands":
			var body struct {
				Commands []struct {
					Value bool `json:"value"`
				} `json:"commands"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			value := body.Commands[0].Value
			mu.Lock()
			events = append(events, value)
			mu.Unlock()
			if value {
				onOnce.Do(func() { close(onRequest) })
				if blockOn {
					<-r.Context().Done()
					return
				}
			}
			success := value || offSucceeds
			writeJSON(t, w, map[string]any{"success": success})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs = &lockedBuffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	stopResult := filepath.Join(t.TempDir(), "stop-result")
	t.Setenv("STOP_RESULT_FILE", stopResult)
	cfg := &config.Config{
		Host:                 server.URL,
		ClientID:             "test-client",
		Secret:               "test-secret",
		PumpDeviceID:         "pump",
		TankDeviceID:         "tank",
		TokenCachePath:       filepath.Join(t.TempDir(), "token.json"),
		PumpCurrentThreshold: 5000,
		TankCurrentThreshold: 100,
		MaxRetries:           2,
		RetryDelay:           time.Hour,
		LowCurrentMinutes:    2,
		MonitorInterval:      time.Hour,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, cfg) }()

	select {
	case <-onRequest:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for pump ON request")
	}
	if !blockOn {
		deadline := time.Now().Add(5 * time.Second)
		for !logs.Contains(`msg="monitor started"`) && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if !logs.Contains(`msg="monitor started"`) {
			cancel()
			t.Fatal("monitor did not start before shutdown")
		}
	}
	cancel()

	select {
	case err := <-done:
		if offSucceeds && err != nil {
			t.Fatalf("run() error = %v", err)
		}
		if !offSucceeds && err == nil {
			t.Fatal("run() succeeded despite Tuya rejecting OFF")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not finish after shutdown cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) < 2 || !events[0] || events[len(events)-1] {
		t.Fatalf("switch command sequence = %v, want ON followed by OFF", events)
	}
	wantResult := "accepted\n"
	if !offSucceeds {
		wantResult = "failed\n"
	}
	if result, err := os.ReadFile(stopResult); err != nil || string(result) != wantResult {
		t.Fatalf("stop result = %q, err=%v; want %q", result, err, wantResult)
	}
	for i, value := range events {
		if i > 0 && value {
			t.Fatalf("ON command was sent after shutdown began: %v", events)
		}
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("write JSON response: %v", err)
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  []byte
}

func (b *lockedBuffer) Contains(value string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Contains(string(b.b), value)
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.b = append(b.b, p...)
	return len(p), nil
}
