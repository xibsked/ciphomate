package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host              string
	ClientID          string
	Secret            string
	DeviceID          string
	MaxRetries        int
	RetryDelay1       time.Duration
	RetryDelay2       time.Duration
	CurrentThreshold  int
	LowCurrentMinutes int
	MonitorInterval   time.Duration
}

func Load() (*Config, error) {
	return &Config{
		Host:              os.Getenv("HOST"),
		ClientID:          os.Getenv("CLIENT_ID"),
		Secret:            os.Getenv("SECRET"),
		DeviceID:          os.Getenv("DEVICE_ID"),
		MaxRetries:        getEnvInt("MAX_RETRIES", 2),
		RetryDelay1:       getEnvDuration("RETRY_DELAY_1", 30*time.Minute),
		RetryDelay2:       getEnvDuration("RETRY_DELAY_2", 60*time.Minute),
		CurrentThreshold:  getEnvInt("CURRENT_THRESHOLD", 20),
		LowCurrentMinutes: getEnvInt("LOW_CURRENT_MINUTES", 5),
		MonitorInterval:   getEnvDuration("MONITOR_INTERVAL", 2*time.Minute),
	}, nil
}

func getEnvInt(key string, defaultVal int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
