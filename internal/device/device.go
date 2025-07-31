package device

import (
	"ciphomate/internal/tuya"
	"encoding/json"
	"fmt"
	"log"
)

type Device struct {
	Client   *tuya.TuyaClient
	DeviceID string
}

func NewDevice(client *tuya.TuyaClient, deviceID string) *Device {
	return &Device{
		Client:   client,
		DeviceID: deviceID,
	}
}

func (d *Device) FetchInchingTime() (int, error) {
	defaultInching := 180

	// return defaultInching, nil

	path := fmt.Sprintf("/v1.0/devices/%s/status", d.DeviceID)
	resp, err := d.Client.SendRequest("GET", path, nil)
	if err != nil {
		return defaultInching, err
	}
	var result StatusResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return defaultInching, err
	}

	log.Printf("result: %+v", result)
	for _, item := range result.Result {
		if item.Code == "switch_inching" {
			enabled, seconds, err := DecodeSwitchInching(item.Value)
			if err != nil {
				log.Printf("Failed to decode switch_inching: %v", err)
				continue
			}
			minutes := seconds / 60
			log.Printf("Inching enabled: %v", enabled)
			log.Printf("Inching time (from device): %d minutes", minutes)
			return minutes, nil
		}
	}
	return defaultInching, fmt.Errorf("switch_inching not found in status")
}

func (d *Device) GetCurrent() (int, error) {
	// return 0, nil
	path := fmt.Sprintf("/v1.0/devices/%s/status", d.DeviceID)
	resp, err := d.Client.SendRequest("GET", path, nil)
	if err != nil {
		return 0, err
	}
	var result StatusResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return 0, err
	}

	for _, item := range result.Result {
		if item.Code == "cur_current" {
			return int(item.Value.(float64)), nil
		}
	}
	return 0, fmt.Errorf("cur_current not found")
}

func (d *Device) Switch(on bool) error {
	// return nil
	cmd := CommandRequest{
		Commands: []Command{{
			Code:  "switch_1",
			Value: on,
		}},
	}
	payload, _ := json.Marshal(cmd)
	path := fmt.Sprintf("/v1.0/devices/%s/commands", d.DeviceID)
	_, err := d.Client.SendRequest("POST", path, payload)
	return err
}
