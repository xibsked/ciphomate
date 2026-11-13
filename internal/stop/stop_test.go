package stop_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestStopUsesGracefulTuyaAcceptanceWithoutFallback(t *testing.T) {
	dir, appBinary, stopScript := setupScripts(t)
	pidFile := filepath.Join(dir, "app.pid")
	resultFile := filepath.Join(dir, "stop-result")
	callsFile := filepath.Join(dir, "fallback-calls")
	app := exec.Command(appBinary)
	app.Env = helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0")
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if app.ProcessState == nil {
			_ = app.Process.Kill()
			_, _ = app.Process.Wait()
		}
	})
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(app.Process.Pid)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := runStop(stopScript, helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0"))
	if err != nil {
		t.Fatalf("stop script failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "graceful_off_accepted_by_tuya") {
		t.Fatalf("graceful Tuya acceptance was not reported: %s", output)
	}
	if _, err := os.Stat(callsFile); !os.IsNotExist(err) {
		t.Fatalf("fallback should not run after graceful Tuya acceptance; stat err=%v", err)
	}
	if _, err := app.Process.Wait(); err != nil {
		t.Fatalf("graceful app exit: %v", err)
	}
}

func TestStopUsesOneOffFallbackForStalePID(t *testing.T) {
	dir, appBinary, stopScript := setupScripts(t)
	pidFile := filepath.Join(dir, "app.pid")
	resultFile := filepath.Join(dir, "stop-result")
	callsFile := filepath.Join(dir, "fallback-calls")
	if err := os.WriteFile(pidFile, []byte("999999999\n"), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := runStop(stopScript, helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0"))
	if err != nil {
		t.Fatalf("stop script failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "off_fallback_accepted_by_tuya") {
		t.Fatalf("fallback Tuya acceptance was not reported: %s", output)
	}
	if got := countLines(t, callsFile); got != 1 {
		t.Fatalf("fallback OFF request count = %d, want 1", got)
	}
}

func TestStopFallsBackWhenGracefulTuyaOffFails(t *testing.T) {
	dir, appBinary, stopScript := setupScripts(t)
	pidFile := filepath.Join(dir, "app.pid")
	resultFile := filepath.Join(dir, "stop-result")
	callsFile := filepath.Join(dir, "fallback-calls")
	app := exec.Command(appBinary, "fail-term")
	app.Env = helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0")
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if app.ProcessState == nil {
			_ = app.Process.Kill()
			_, _ = app.Process.Wait()
		}
	})
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(app.Process.Pid)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := runStop(stopScript, helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0"))
	if err != nil {
		t.Fatalf("stop script failed after successful fallback: %v\n%s", err, output)
	}
	if !strings.Contains(output, "graceful_off_not_confirmed") ||
		!strings.Contains(output, "off_fallback_accepted_by_tuya") {
		t.Fatalf("expected graceful failure and fallback success, got:\n%s", output)
	}
	if got := countLines(t, callsFile); got != 1 {
		t.Fatalf("fallback OFF request count = %d, want 1", got)
	}
	if err := app.Wait(); err == nil {
		t.Fatal("fake app should report graceful OFF failure")
	}
}

func TestStopReportsFallbackFailure(t *testing.T) {
	dir, appBinary, stopScript := setupScripts(t)
	pidFile := filepath.Join(dir, "app.pid")
	resultFile := filepath.Join(dir, "stop-result")
	callsFile := filepath.Join(dir, "fallback-calls")

	output, err := runStop(stopScript, helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "9"))
	if err == nil {
		t.Fatalf("stop script succeeded despite failed Tuya OFF; output:\n%s", output)
	}
	if !strings.Contains(output, "scheduled_stop_not_confirmed_by_tuya") {
		t.Fatalf("failure outcome was not clearly reported: %s", output)
	}
	if got := countLines(t, callsFile); got != 1 {
		t.Fatalf("fallback OFF request count = %d, want 1", got)
	}
}

func TestStopFallsBackThenForceTerminatesHungApp(t *testing.T) {
	dir, appBinary, stopScript := setupScripts(t)
	pidFile := filepath.Join(dir, "app.pid")
	resultFile := filepath.Join(dir, "stop-result")
	callsFile := filepath.Join(dir, "fallback-calls")
	app := exec.Command(appBinary, "ignore-term")
	app.Env = helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0")
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if app.ProcessState == nil {
			_ = app.Process.Kill()
			_, _ = app.Process.Wait()
		}
	})
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(app.Process.Pid)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	env := helperEnv(dir, appBinary, pidFile, resultFile, callsFile, "0")
	env = append(env, "STOP_GRACE_SECONDS=1")
	output, err := runStop(stopScript, env)
	if err != nil {
		t.Fatalf("stop script failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "off_fallback_accepted_by_tuya") ||
		!strings.Contains(output, "app_force_terminated") {
		t.Fatalf("expected fallback then force termination, got:\n%s", output)
	}
	if got := countLines(t, callsFile); got != 1 {
		t.Fatalf("fallback OFF request count = %d, want 1", got)
	}
	if err := app.Wait(); err == nil {
		t.Fatal("hung app exited without the forced termination")
	}
}

func setupScripts(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	appBinary := filepath.Join(dir, "fake-ciphomate")
	stopScript, err := filepath.Abs(filepath.Join("..", "..", "scripts", "stop.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/bash
set -u
if [[ "${1:-}" == "--off" ]]; then
  printf 'fallback\n' >> "$CALLS_FILE"
  exit "$OFF_EXIT_CODE"
fi
if [[ "${1:-}" == "ignore-term" ]]; then
  trap '' TERM
elif [[ "${1:-}" == "fail-term" ]]; then
  trap 'printf "failed\n" > "$STOP_RESULT_FILE"; exit 1' TERM
else
  trap 'printf "accepted\n" > "$STOP_RESULT_FILE"; exit 0' TERM
fi
while :; do
  read -r -t 0.1 _ || true
done
`
	if err := os.WriteFile(appBinary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return dir, appBinary, stopScript
}

func helperEnv(dir, appBinary, pidFile, resultFile, callsFile, offExitCode string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"PID_FILE=" + pidFile,
		"STOP_RESULT_FILE=" + resultFile,
		"CALLS_FILE=" + callsFile,
		"OFF_EXIT_CODE=" + offExitCode,
		"STOP_GRACE_SECONDS=2",
		"CIPHOMATE_BIN=" + appBinary,
		"CIPHOMATE_DIR=" + dir,
		"CIPHOMATE_EXE_PATH=" + bashExecutable(),
	}
}

func bashExecutable() string {
	path, err := exec.LookPath("bash")
	if err != nil {
		return "/usr/bin/bash"
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func runStop(script string, env []string) (string, error) {
	cmd := exec.Command("bash", script)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fallback call log: %v", err)
	}
	return strings.Count(string(data), "\n")
}
