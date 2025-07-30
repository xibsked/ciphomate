package scheduler

import (
	"log"
	"time"

	"ciphomate/internal/config"
	"ciphomate/internal/device"
)

type Scheduler struct {
	Config *config.Config
	Device *device.Device
}

func NewScheduler(config *config.Config, device *device.Device) *Scheduler {
	return &Scheduler{
		Config: config,
		Device: device,
	}
}

func (s *Scheduler) MonitorUntilExpiry(expiry time.Time) {
	// start := time.Now()
	if s.runMonitorLoop(expiry) {
		log.Println("✅ Inching completed fully. No retries needed.")
		err := s.Device.Switch(false)
		if err != nil {
			log.Printf("Error switching off %v", err)
		}
		return
	}

	retryDelays := []time.Duration{s.Config.RetryDelay1, s.Config.RetryDelay2}

	for i := 0; i < s.Config.MaxRetries && i < len(retryDelays); i++ {
		delay := retryDelays[i]
		log.Printf("🔁 Retry #%d scheduled after %v.", i+1, delay)
		if s.waitAndRetry(delay, expiry) {
			log.Printf("✅ Retry #%d succeeded.", i+1)
			return
		}
		log.Printf("❌ Retry #%d failed or skipped.", i+1)
	}
}

func (s *Scheduler) runMonitorLoop(expiry time.Time) bool {
	ticker := time.NewTicker(s.Config.MonitorInterval)
	defer ticker.Stop()

	lowCurrentCount := 0
	for t := range ticker.C {
		if t.After(expiry) {
			log.Println("✅ Device remained ON until expiry.")
			return true
		}

		current, err := s.Device.GetCurrent()
		if err != nil {
			log.Println("⚠️ Error getting current:", err)
			continue
		}

		log.Printf("🔌 Current draw: %d mA", current)

		if current < s.Config.CurrentThreshold {
			lowCurrentCount++
			log.Printf("⚠️ Low current (%d checks)", lowCurrentCount)
		} else {
			lowCurrentCount = 0
		}

		// Use Interval to calculate total time before shutdown
		if time.Duration(lowCurrentCount)*s.Config.MonitorInterval >= time.Duration(s.Config.LowCurrentMinutes)*time.Minute {
			log.Println("❌ Turning OFF early due to sustained low current.")
			err = s.Device.Switch(false)
			if err != nil {
				log.Printf("Error switching off %v", err)
			}
			return false
		}
	}
	return false
}

func (s *Scheduler) waitAndRetry(delay time.Duration, expiry time.Time) bool {
	retryTime := time.Now().Add(delay)
	if retryTime.After(expiry) {
		log.Println("⏭️ Retry would exceed inching expiry. Skipping.")
		return false
	}

	time.Sleep(delay)

	log.Println("🔁 Retrying: Switching ON device")
	err := s.Device.Switch(true)
	if err != nil {
		log.Println("❌ Retry switch ON failed:", err)
		return false
	}

	return s.runMonitorLoop(expiry)
}
