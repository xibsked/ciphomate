package scheduler

import (
	"context"
	"log/slog"
	"time"

	"ciphomate/internal/config"
	"ciphomate/internal/device"
)

type Scheduler struct {
	Config *config.Config
	Pump   *device.Device
	Tank   *device.Device
}

func NewScheduler(config *config.Config, pump *device.Device, tank *device.Device) *Scheduler {
	return &Scheduler{
		Config: config,
		Pump:   pump,
		Tank:   tank,
	}
}

func (s *Scheduler) Start(ctx context.Context, expiry time.Time) {
	logger := slog.With("component", "scheduler")
	logger.Info("monitor started", "expires_at", expiry)

	result := s.monitorOnce(ctx, expiry)
	switch result {
	case monitorCompleted:
		logger.Info("monitor period completed; switching pump off")
		if err := s.Pump.SwitchContext(ctx, false); err != nil {
			logger.Error("failed to switch pump off after monitor completed", "error", err)
		}
		return
	case monitorTankFull, monitorCanceled:
		return
	}

	logger.Warn("initial monitoring failed; retries will start", "max_retries", s.Config.MaxRetries)

	for i := 0; i < s.Config.MaxRetries; i++ {
		logger.Info("waiting before retry", "retry", i+1, "retry_delay", s.Config.RetryDelay)
		timer := time.NewTimer(s.Config.RetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			logger.Info("retry wait canceled by shutdown")
			return
		case <-timer.C:
		}

		logger.Info("retry switching pump on", "retry", i+1)
		err := s.Pump.SwitchContext(ctx, true)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("retry switch canceled by shutdown", "retry", i+1)
				return
			}
			logger.Error("retry failed to switch pump on", "retry", i+1, "error", err)
			continue
		}

		timer = time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			logger.Info("retry monitoring canceled by shutdown", "retry", i+1)
			return
		case <-timer.C:
		}

		logger.Info("retry monitoring started", "retry", i+1, "expires_at", expiry)
		result = s.monitorOnce(ctx, expiry)
		switch result {
		case monitorCompleted:
			logger.Info("retry monitoring succeeded; switching pump off", "retry", i+1)
			if err := s.Pump.SwitchContext(ctx, false); err != nil {
				logger.Error("retry succeeded but pump could not be switched off", "retry", i+1, "error", err)
			}
			return
		case monitorTankFull, monitorCanceled:
			return
		}

		logger.Warn("retry monitoring failed", "retry", i+1)
	}

	logger.Error("all retries exhausted", "max_retries", s.Config.MaxRetries)
}

type monitorResult int

const (
	monitorFailed monitorResult = iota
	monitorCompleted
	monitorTankFull
	monitorCanceled
)

func (s *Scheduler) monitorOnce(ctx context.Context, expiry time.Time) monitorResult {
	logger := slog.With("component", "scheduler")
	ticker := time.NewTicker(s.Config.MonitorInterval)
	defer ticker.Stop()

	lowCurrentCount := 0
	for {
		var t time.Time
		select {
		case <-ctx.Done():
			logger.Info("monitor canceled by shutdown")
			return monitorCanceled
		case t = <-ticker.C:
		}

		if t.After(expiry) {
			logger.Info("monitor period ended", "expires_at", expiry)
			return monitorCompleted
		}

		current, err := s.Pump.GetCurrentContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("pump current request canceled by shutdown")
				return monitorCanceled
			}
			logger.Warn("could not read pump current", "device_id", s.Pump.DeviceID, "error", err)
			continue
		}

		logger.Info("pump current measured",
			"device_id", s.Pump.DeviceID,
			"current_ma", current,
			"threshold_ma", s.Config.PumpCurrentThreshold,
		)

		if current < s.Config.PumpCurrentThreshold {
			lowCurrentCount++
			logger.Warn("pump current below threshold",
				"device_id", s.Pump.DeviceID,
				"current_ma", current,
				"threshold_ma", s.Config.PumpCurrentThreshold,
				"consecutive_checks", lowCurrentCount,
				"required_minutes", s.Config.LowCurrentMinutes,
			)
		} else {
			if lowCurrentCount > 0 {
				logger.Info("pump current recovered", "previous_low_current_checks", lowCurrentCount)
			}
			lowCurrentCount = 0
		}

		if time.Duration(lowCurrentCount)*s.Config.MonitorInterval >= time.Duration(s.Config.LowCurrentMinutes)*time.Minute {
			logger.Error("early shutdown due to sustained low current",
				"consecutive_checks", lowCurrentCount,
				"required_minutes", s.Config.LowCurrentMinutes,
			)
			if err := s.Pump.SwitchContext(ctx, false); err != nil {
				logger.Error("failed to switch pump off after low-current shutdown", "error", err)
			}
			return monitorFailed
		}

		tankCurrent, err := s.Tank.GetCurrentContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("tank current request canceled by shutdown")
				return monitorCanceled
			}
			logger.Warn("could not read tank current", "device_id", s.Tank.DeviceID, "error", err)
			continue
		}

		if tankCurrent >= s.Config.TankCurrentThreshold {
			logger.Info("tank is full; switching pump off",
				"device_id", s.Tank.DeviceID,
				"tank_current_ma", tankCurrent,
				"threshold_ma", s.Config.TankCurrentThreshold,
			)
			if err := s.Pump.SwitchContext(ctx, false); err != nil {
				logger.Error("failed to switch pump off after tank-full detection", "error", err)
			}
			logger.Info("terminating after tank-full detection")
			return monitorTankFull
		}
	}
}
