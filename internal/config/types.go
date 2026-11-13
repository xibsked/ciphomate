package config

import (
	"time"
)

type Config struct {
	OffOnly              bool
	Host                 string
	ClientID             string
	Secret               string
	LogLevel             string
	TokenCachePath       string
	PumpDeviceID         string
	TankDeviceID         string
	MaxRetries           int
	RetryDelay           time.Duration
	PumpCurrentThreshold int
	TankCurrentThreshold int
	LowCurrentMinutes    int
	MonitorInterval      time.Duration
}
