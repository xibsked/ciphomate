package config

import (
	"time"
)

type Config struct {
	Host              string
	ClientID          string
	Secret            string
	PumpDeviceID      string
	TankDeviceID      string
	MaxRetries        int
	RetryDelay1       time.Duration
	RetryDelay2       time.Duration
	CurrentThreshold  int
	LowCurrentMinutes int
	MonitorInterval   time.Duration
}
