package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		value string
		want  slog.Level
	}{
		{value: "debug", want: slog.LevelDebug},
		{value: "info", want: slog.LevelInfo},
		{value: "", want: slog.LevelInfo},
		{value: "warn", want: slog.LevelWarn},
		{value: "warning", want: slog.LevelWarn},
		{value: "error", want: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := ParseLevel(tt.value)
			if err != nil {
				t.Fatalf("ParseLevel(%q) error = %v", tt.value, err)
			}
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseLevelRejectsUnknownValue(t *testing.T) {
	if _, err := ParseLevel("verbose"); err == nil {
		t.Fatal("ParseLevel(verbose) succeeded, want error")
	}
}

func TestNewTextLoggerFiltersByLevel(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewTextLogger(&output, "warn")
	if err != nil {
		t.Fatal(err)
	}

	logger.Debug("debug event")
	logger.Info("info event")
	logger.Warn("warn event", "component", "test")

	logs := output.String()
	if strings.Contains(logs, "debug event") || strings.Contains(logs, "info event") {
		t.Fatalf("below-threshold logs were emitted: %s", logs)
	}
	if !strings.Contains(logs, "level=WARN") || !strings.Contains(logs, "component=test") {
		t.Fatalf("structured warning not found in output: %s", logs)
	}
}
