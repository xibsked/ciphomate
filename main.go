package main

import (
	"ciphomate/internal/config"
	"ciphomate/internal/device"
	"ciphomate/internal/logging"
	"ciphomate/internal/scheduler"
	"ciphomate/internal/tuya"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "component", "app", "error", err)
		os.Exit(1)
	}

	logger, err := logging.NewTextLogger(os.Stderr, cfg.LogLevel)
	if err != nil {
		slog.Error("failed to configure logging", "component", "app", "error", err)
		os.Exit(2)
	}
	slog.SetDefault(logger)

	if err := run(cfg); err != nil {
		slog.Error("application stopped with an error", "component", "app", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	logger := slog.With("component", "app")
	logger.Info("application started",
		"pump_device_id", cfg.PumpDeviceID,
		"tank_device_id", cfg.TankDeviceID,
		"pump_current_threshold_ma", cfg.PumpCurrentThreshold,
		"tank_current_threshold_ma", cfg.TankCurrentThreshold,
		"monitor_interval", cfg.MonitorInterval,
		"max_retries", cfg.MaxRetries,
		"retry_delay", cfg.RetryDelay,
	)
	tm := tuya.NewTokenManager(cfg, "token_cache.json")
	client := tuya.NewTuyaClient(cfg, tm)
	pump := device.NewDevice(client, cfg.PumpDeviceID)
	tank := device.NewDevice(client, cfg.TankDeviceID)

	tankCurrent, err := tank.GetCurrent()
	if err != nil {
		logger.Warn("could not read tank current; continuing startup",
			"component", "device",
			"device_id", cfg.TankDeviceID,
			"error", err,
		)
	}

	if tankCurrent >= cfg.TankCurrentThreshold {
		logger.Info("tank is full; pump will not start",
			"tank_current_ma", tankCurrent,
			"threshold_ma", cfg.TankCurrentThreshold,
		)
		return nil
	}

	inchingMinutes, err := pump.FetchInchingTime()
	if err != nil {
		return fmt.Errorf("fetch pump inching time: %w", err)
	}

	expiry := time.Now().Add(time.Duration(inchingMinutes) * time.Minute)

	logger.Info("starting pump monitoring",
		"inching_minutes", inchingMinutes,
		"monitor_expires_at", expiry,
	)

	err = pump.Switch(true)
	if err != nil {
		return fmt.Errorf("switch pump on: %w", err)
	}
	monitor := scheduler.NewScheduler(cfg, pump, tank)

	go monitor.Start(expiry)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	logger.Info("waiting for shutdown signal")

	sig := <-sigChan

	logger.Info("shutdown signal received", "signal", sig.String())

	err = pump.Switch(false)
	if err != nil {
		return fmt.Errorf("switch pump off during shutdown: %w", err)
	} else {
		logger.Info("pump switched off during shutdown")
	}

	logger.Info("shutdown complete")
	return nil
}
