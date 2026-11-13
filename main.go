package main

import (
	"ciphomate/internal/config"
	"ciphomate/internal/device"
	"ciphomate/internal/logging"
	"ciphomate/internal/scheduler"
	"ciphomate/internal/tuya"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

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

	if cfg.OffOnly {
		if err := switchPumpOff(cfg); err != nil {
			slog.Error("one-shot pump OFF request failed", "component", "app", "error", err)
			os.Exit(1)
		}
		slog.Info("one-shot pump OFF request accepted by Tuya", "component", "app")
		return
	}

	if err := run(ctx, cfg); err != nil {
		slog.Error("application stopped with an error", "component", "app", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
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
	tm := tuya.NewTokenManager(cfg, cfg.TokenCachePath)
	client := tuya.NewTuyaClient(cfg, tm)
	pump := device.NewDevice(client, cfg.PumpDeviceID)
	tank := device.NewDevice(client, cfg.TankDeviceID)

	tankCurrent, err := tank.GetCurrentContext(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return shutdownPump(pump, "shutdown during tank-current startup check")
		}
		logger.Warn("could not read tank current; continuing startup",
			"component", "device",
			"device_id", cfg.TankDeviceID,
			"error", err,
		)
	}

	if ctx.Err() != nil {
		return shutdownPump(pump, "shutdown during startup")
	}

	if tankCurrent >= cfg.TankCurrentThreshold {
		logger.Info("tank is full; pump will not start",
			"tank_current_ma", tankCurrent,
			"threshold_ma", cfg.TankCurrentThreshold,
		)
		return nil
	}

	inchingMinutes, err := pump.FetchInchingTimeContext(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return shutdownPump(pump, "shutdown during inching-time lookup")
		}
		return fmt.Errorf("fetch pump inching time: %w", err)
	}

	if ctx.Err() != nil {
		return shutdownPump(pump, "shutdown before pump start")
	}

	expiry := time.Now().Add(time.Duration(inchingMinutes) * time.Minute)

	logger.Info("starting pump monitoring",
		"inching_minutes", inchingMinutes,
		"monitor_expires_at", expiry,
	)

	err = pump.SwitchContext(ctx, true)
	if err != nil {
		if ctx.Err() != nil {
			return shutdownPump(pump, "shutdown during pump start")
		}
		return fmt.Errorf("switch pump on: %w", err)
	}
	if ctx.Err() != nil {
		return shutdownPump(pump, "shutdown after pump start")
	}
	monitor := scheduler.NewScheduler(cfg, pump, tank)

	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		monitor.Start(ctx, expiry)
	}()

	logger.Info("waiting for shutdown signal")

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received", "error", ctx.Err())
	case <-monitorDone:
		logger.Info("monitor completed; waiting for shutdown signal")
		<-ctx.Done()
		logger.Info("shutdown signal received", "error", ctx.Err())
	}

	<-monitorDone
	return shutdownPump(pump, "application shutdown")
}

func shutdownPump(pump *device.Device, reason string) error {
	logger := slog.With("component", "app")
	logger.Info("requesting pump OFF from Tuya", "reason", reason)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pump.SwitchContext(ctx, false); err != nil {
		writeStopResult("failed")
		return fmt.Errorf("request pump OFF during %s: %w", reason, err)
	}
	writeStopResult("accepted")
	logger.Info("Tuya accepted pump OFF request", "reason", reason)
	logger.Info("shutdown complete")
	return nil
}

func switchPumpOff(cfg *config.Config) error {
	tm := tuya.NewTokenManager(cfg, cfg.TokenCachePath)
	client := tuya.NewTuyaClient(cfg, tm)
	pump := device.NewDevice(client, cfg.PumpDeviceID)
	return shutdownPump(pump, "one-shot stop fallback")
}

func writeStopResult(result string) {
	path := os.Getenv("STOP_RESULT_FILE")
	if path == "" {
		path = "/run/ciphomate-stop-result"
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(result+"\n"), 0600); err != nil {
		slog.Warn("could not write stop result marker", "path", path, "error", err)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		slog.Warn("could not publish stop result marker", "path", path, "error", err)
		_ = os.Remove(tmpPath)
	}
}
