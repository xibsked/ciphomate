package config

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var envPath string

func init() {
	flag.StringVar(&envPath, "env", "", "Optional path to .env file")
}

func Load() (*Config, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if envPath != "" {
		err := godotenv.Load(envPath)
		if err != nil {
			slog.Warn("could not load environment file", "component", "config", "path", envPath, "error", err)
		} else {
			slog.Info("loaded environment file", "component", "config", "path", envPath)
		}
	} else {
		err := godotenv.Load()
		if err != nil {
			slog.Debug("no default .env file loaded; using process environment", "component", "config", "error", err)
		} else {
			slog.Info("loaded environment file", "component", "config", "path", ".env")
		}
	}
	return &Config{
		Host:                 os.Getenv("HOST"),
		ClientID:             os.Getenv("CLIENT_ID"),
		Secret:               os.Getenv("SECRET"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		PumpDeviceID:         os.Getenv("PUMP_DEVICE_ID"),
		TankDeviceID:         os.Getenv("TANK_DEVICE_ID"),
		MaxRetries:           getEnvInt("MAX_RETRIES", 2),
		RetryDelay:           getEnvDuration("RETRY_DELAY", 30*time.Minute),
		PumpCurrentThreshold: getEnvInt("PUMP_CURRENT_THRESHOLD", 5000),
		TankCurrentThreshold: getEnvInt("TANK_CURRENT_THRESHOLD", 100),
		LowCurrentMinutes:    getEnvInt("LOW_CURRENT_MINUTES", 2),
		MonitorInterval:      getEnvDuration("MONITOR_INTERVAL", 1*time.Minute),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		} else {
			slog.Warn("invalid integer environment value; using default",
				"component", "config",
				"variable", key,
				"default", defaultVal,
				"error", err,
			)
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		} else {
			slog.Warn("invalid duration environment value; using default",
				"component", "config",
				"variable", key,
				"default", defaultVal,
				"error", err,
			)
		}
	}
	return defaultVal
}
