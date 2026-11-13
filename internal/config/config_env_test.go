package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestComposeEnvFileReachesApplicationAndCron(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	envContents := `HOST=https://api.example.test
CLIENT_ID=test-client
SECRET='fake secret # value = test'
PUMP_DEVICE_ID=pump-test
TANK_DEVICE_ID=tank-test
START_TIME=06:15
STOP_TIME=22:45
TZ=Asia/Kolkata
LOG_LEVEL=debug
TOKEN_CACHE_PATH=/run/test-token-cache.json
MAX_RETRIES=3
RETRY_DELAY=45m
PUMP_CURRENT_THRESHOLD=4000
TANK_CURRENT_THRESHOLD=120
LOW_CURRENT_MINUTES=3
MONITOR_INTERVAL=30s
`
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(envContents), 0600); err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"HOST":                   "https://api.example.test",
		"CLIENT_ID":              "test-client",
		"SECRET":                 "fake secret # value = test",
		"PUMP_DEVICE_ID":         "pump-test",
		"TANK_DEVICE_ID":         "tank-test",
		"START_TIME":             "06:15",
		"STOP_TIME":              "22:45",
		"TZ":                     "Asia/Kolkata",
		"LOG_LEVEL":              "debug",
		"TOKEN_CACHE_PATH":       "/run/test-token-cache.json",
		"MAX_RETRIES":            "3",
		"RETRY_DELAY":            "45m",
		"PUMP_CURRENT_THRESHOLD": "4000",
		"TANK_CURRENT_THRESHOLD": "120",
		"LOW_CURRENT_MINUTES":    "3",
		"MONITOR_INTERVAL":       "30s",
	}

	command := exec.Command(
		"docker", "compose",
		"--project-directory", projectDir,
		"-f", filepath.Join(repoRoot, "compose.yml"),
		"config", "--format", "json",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose config failed: %v\n%s", err, output)
	}

	var rendered struct {
		Services map[string]struct {
			Environment map[string]string `json:"environment"`
		} `json:"services"`
	}
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode docker compose config: %v\n%s", err, output)
	}

	service, ok := rendered.Services["ciphomate"]
	if !ok {
		t.Fatal("docker compose config has no ciphomate service")
	}
	for name, want := range expected {
		if got := service.Environment[name]; got != want {
			t.Errorf("Compose environment %s = %q, want %q", name, got, want)
		}
	}

	cronEnvPath := filepath.Join(projectDir, "cron-env.sh")
	capture := exec.Command("bash", filepath.Join(repoRoot, "scripts", "write-cron-env.sh"), cronEnvPath)
	capture.Env = []string{"PATH=" + os.Getenv("PATH")}
	for name, value := range expected {
		capture.Env = append(capture.Env, name+"="+value)
	}
	if output, err := capture.CombinedOutput(); err != nil {
		t.Fatalf("write cron environment: %v\n%s", err, output)
	}

	restore := exec.Command("bash", "-c", `printf '%s' "$SECRET"`)
	restore.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"BASH_ENV=" + cronEnvPath,
	}
	output, err = restore.Output()
	if err != nil {
		t.Fatalf("restore cron environment: %v", err)
	}
	if got := string(output); got != expected["SECRET"] {
		t.Errorf("cron-restored SECRET = %q, want %q", got, expected["SECRET"])
	}

	for _, name := range []string{
		"HOST", "CLIENT_ID", "SECRET", "LOG_LEVEL", "TOKEN_CACHE_PATH", "PUMP_DEVICE_ID", "TANK_DEVICE_ID",
		"MAX_RETRIES", "RETRY_DELAY", "PUMP_CURRENT_THRESHOLD",
		"TANK_CURRENT_THRESHOLD", "LOW_CURRENT_MINUTES", "MONITOR_INTERVAL",
	} {
		t.Setenv(name, expected[name])
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != expected["HOST"] ||
		cfg.ClientID != expected["CLIENT_ID"] ||
		cfg.Secret != expected["SECRET"] ||
		cfg.LogLevel != expected["LOG_LEVEL"] ||
		cfg.TokenCachePath != expected["TOKEN_CACHE_PATH"] ||
		cfg.PumpDeviceID != expected["PUMP_DEVICE_ID"] ||
		cfg.TankDeviceID != expected["TANK_DEVICE_ID"] ||
		cfg.MaxRetries != 3 ||
		cfg.RetryDelay != 45*time.Minute ||
		cfg.PumpCurrentThreshold != 4000 ||
		cfg.TankCurrentThreshold != 120 ||
		cfg.LowCurrentMinutes != 3 ||
		cfg.MonitorInterval != 30*time.Second {
		t.Errorf("loaded app config does not match Compose environment: %+v", cfg)
	}
}
