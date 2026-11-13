package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q: expected debug, info, warn, or error", value)
	}
}

func NewTextLogger(output io.Writer, levelName string) (*slog.Logger, error) {
	level, err := ParseLevel(levelName)
	if err != nil {
		return nil, err
	}

	return slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: level,
	})), nil
}
