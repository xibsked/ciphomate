package main

import (
	"ciphomate/internal/config"
	"ciphomate/internal/device"
	"ciphomate/internal/scheduler"
	"ciphomate/internal/tuya"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	log.Printf("✅ Loaded config: %+v", cfg)

	tm := tuya.NewTokenManager(cfg, "token_cache.json")
	client := tuya.NewTuyaClient(cfg, tm)
	pump := device.NewDevice(client, cfg.PumpDeviceID)

	inchingMinutes, err := pump.FetchInchingTime()
	if err != nil {
		log.Fatalf("Error fetching inching time: %v", err)
	}
	expiry := time.Now().Add(time.Duration(inchingMinutes) * time.Minute)

	log.Println("Triggering initial power ON and monitoring...")
	err = pump.Switch(true)
	if err != nil {
		log.Fatalf("Failed to switch ON device: %v", err)
	}
	scheduler := scheduler.NewScheduler(cfg, pump)

	go scheduler.Start(expiry)

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Program is running. Waiting for signal to exit...")

	<-sigChan // Block here until a signal is received

	log.Println("Signal received. Turning off device and exiting...")

	// ✅ Graceful shutdown
	err = pump.Switch(false)
	if err != nil {
		log.Printf("Error switching off device: %v", err)
	} else {
		log.Println("Device turned OFF successfully.")
	}

	log.Println("Shutdown complete.")
}
