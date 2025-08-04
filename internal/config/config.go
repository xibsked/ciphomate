package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var envPath string

func init() {
	flag.StringVar(&envPath, "env", "", "Optional path to .env file")
	flag.Parse()
	if envPath != "" {
		err := godotenv.Load(envPath)
		if err != nil {
			log.Printf("⚠️ Could not load .env from '%s': %v", envPath, err)
		} else {
			log.Printf("✅ Loaded .env from: %s", envPath)
		}
	} else {
		err := godotenv.Load()
		if err != nil {
			log.Println("No .env file found or unable to load it — relying on OS environment")
		} else {
			log.Println("Loading .env from default location")
		}
	}
}

func Load() (*Config, error) {
	return &Config{
		Host:                 os.Getenv("HOST"),
		ClientID:             os.Getenv("CLIENT_ID"),
		Secret:               os.Getenv("SECRET"),
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
