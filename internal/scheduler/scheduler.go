package scheduler

import (
	"log"
	"time"

	"ciphomate/internal/config"
	"ciphomate/internal/device"
)

var (
	MaxRetries        = 2
	RetryDelays       = []time.Duration{30 * time.Minute, 60 * time.Minute}
	CurrentThreshold  = 20 // in mA
	LowCurrentMinutes = 5  // consecutive low current minutes before shutdown
	Interval          = 2 * time.Minute
)

func Load(cfg *config.Config) {
	MaxRetries = cfg.MaxRetries
	RetryDelays = []time.Duration{cfg.RetryDelay1, cfg.RetryDelay2}
	CurrentThreshold = cfg.CurrentThreshold
	LowCurrentMinutes = cfg.LowCurrentMinutes
	Interval = cfg.MonitorInterval
}

func MonitorUntilExpiry(expiry time.Time) {
	// start := time.Now()
	if runMonitorLoop(expiry) {
		log.Println("✅ Inching completed fully. No retries needed.")
		err := device.Switch(false)
		if err != nil {
			log.Printf("Error switching off %v", err)
		}
		return
	}

	for i := 0; i < MaxRetries && i < len(RetryDelays); i++ {
		delay := RetryDelays[i]
		log.Printf("🔁 Retry #%d scheduled after %v.", i+1, delay)
		if waitAndRetry(delay, expiry) {
			log.Printf("✅ Retry #%d succeeded.", i+1)
			return
		}
		log.Printf("❌ Retry #%d failed or skipped.", i+1)
	}
}

func runMonitorLoop(expiry time.Time) bool {
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()

	lowCurrentCount := 0
	for t := range ticker.C {
		if t.After(expiry) {
			log.Println("✅ Device remained ON until expiry.")
			return true
		}

		current, err := device.GetCurrent()
		if err != nil {
			log.Println("⚠️ Error getting current:", err)
			continue
		}

		log.Printf("🔌 Current draw: %d mA", current)

		if current < CurrentThreshold {
			lowCurrentCount++
			log.Printf("⚠️ Low current (%d checks)", lowCurrentCount)
		} else {
			lowCurrentCount = 0
		}

		// Use Interval to calculate total time before shutdown
		if time.Duration(lowCurrentCount)*Interval >= time.Duration(LowCurrentMinutes)*time.Minute {
			log.Println("❌ Turning OFF early due to sustained low current.")
			err = device.Switch(false)
			if err != nil {
				log.Printf("Error switching off %v", err)
			}
			return false
		}
	}
	return false
}

func waitAndRetry(delay time.Duration, expiry time.Time) bool {
	retryTime := time.Now().Add(delay)
	if retryTime.After(expiry) {
		log.Println("⏭️ Retry would exceed inching expiry. Skipping.")
		return false
	}

	time.Sleep(delay)

	log.Println("🔁 Retrying: Switching ON device")
	err := device.Switch(true)
	if err != nil {
		log.Println("❌ Retry switch ON failed:", err)
		return false
	}

	return runMonitorLoop(expiry)
}
