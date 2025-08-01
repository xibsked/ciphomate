package scheduler

import (
	"log"
	"os"
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

func (s *Scheduler) Start(expiry time.Time) {
	log.Printf("🕒 Starting monitor. Expiry at %v", expiry)

	success := s.monitorOnce(expiry)
	if success {
		log.Println("✅ Initial monitor passed. Turning off device.")
		_ = s.Pump.Switch(false)
		return
	}

	log.Println("🔁 Initial monitor failed. Starting retries...")

	for i := 0; i < s.Config.MaxRetries; i++ {
		log.Printf("🕒 Waiting %v before retry #%d", s.Config.RetryDelay, i+1)
		time.Sleep(s.Config.RetryDelay)

		log.Printf("🔁 Retry #%d: Switching ON device", i+1)
		err := s.Pump.Switch(true)
		if err != nil {
			log.Printf("❌ Retry #%d switch ON failed: %v", i+1, err)
			continue
		}

		// Optional: wait a bit before reading current
		time.Sleep(2 * time.Second)

		log.Printf("🔍 Retry #%d monitoring started (for %v)...", i+1, expiry)
		if s.monitorOnce(expiry) {
			log.Printf("✅ Retry #%d succeeded. Turning off device.", i+1)
			_ = s.Pump.Switch(false)
			return
		}

		log.Printf("❌ Retry #%d failed. Continuing...", i+1)
	}

	log.Println("🚫 All retries exhausted.")
}

func (s *Scheduler) monitorOnce(expiry time.Time) bool {
	ticker := time.NewTicker(s.Config.MonitorInterval)
	defer ticker.Stop()

	lowCurrentCount := 0
	for t := range ticker.C {
		if t.After(expiry) {
			log.Println("⏰ Monitor period ended naturally.")
			return true
		}

		current, err := s.Pump.GetCurrent()
		if err != nil {
			log.Println("⚠️ Error reading current:", err)
			continue
		}

		log.Printf("🔌 Current draw: %d mA", current)

		if current < s.Config.PumpCurrentThreshold {
			lowCurrentCount++
			log.Printf("⚠️ Low current (%d checks)", lowCurrentCount)
		} else {
			lowCurrentCount = 0
		}

		if time.Duration(lowCurrentCount)*s.Config.MonitorInterval >= time.Duration(s.Config.LowCurrentMinutes)*time.Minute {
			log.Println("❌ Early shutdown due to sustained low current.")
			s.Pump.Switch(false)
			return false
		}

		tankCurrent, err := s.Tank.GetCurrent()
		if err != nil {
			log.Printf("Cannot get current of tank: %+v", err)
		}

		if tankCurrent >= s.Config.TankCurrentThreshold {
			log.Printf("Tank level detection current is %d", tankCurrent)
			log.Printf("Tank level detection current threshold is %d", s.Config.TankCurrentThreshold)
			log.Printf("Tank is full, switching the pump off.")

			s.Pump.Switch(false)

			log.Printf("Terminating the ciphomate")
			os.Exit(0)
		}
	}
	return false
}
