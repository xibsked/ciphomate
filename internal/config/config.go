package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Host     string `env:"HOST" envDefault:""`
	ClientID string `env:"CLIENT_ID" envDefault:""`
	Secret   string `env:"SECRET" envDefault:""`
	DeviceID string `env:"DEVICE_ID" envDefault:""`

	MaxRetries        int           `env:"MAX_RETRIES" envDefault:"2"`
	RetryDelay1       time.Duration `env:"RETRY_DELAY_1" envDefault:"30m"`
	RetryDelay2       time.Duration `env:"RETRY_DELAY_2" envDefault:"60m"`
	CurrentThreshold  int           `env:"CURRENT_THRESHOLD" envDefault:"20"`
	LowCurrentMinutes int           `env:"LOW_CURRENT_MINUTES" envDefault:"5"`
	MonitorInterval   time.Duration `env:"MONITOR_INTERVAL" envDefault:"2m"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
