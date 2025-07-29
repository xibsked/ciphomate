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

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or unable to load it — relying on OS environment")
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	log.Printf("✅ Loaded config: %+v", cfg)

	tuya.InitAuth(cfg)

	inchingMinutes, err := device.FetchInchingTime()
	if err != nil {
		log.Fatalf("Error fetching inching time: %v", err)
	}
	expiry := time.Now().Add(time.Duration(inchingMinutes) * time.Minute)

	log.Println("Triggering initial power ON and monitoring...")
	err = device.Switch(true)
	if err != nil {
		log.Fatalf("Failed to switch ON device: %v", err)
	}
	scheduler.Load(cfg)
	go scheduler.MonitorUntilExpiry(expiry)

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Program is running. Waiting for signal to exit...")

	<-sigChan // Block here until a signal is received

	log.Println("Signal received. Turning off device and exiting...")

	// ✅ Graceful shutdown
	err = device.Switch(false)
	if err != nil {
		log.Printf("Error switching off device: %v", err)
	} else {
		log.Println("Device turned OFF successfully.")
	}

	log.Println("Shutdown complete.")
}
