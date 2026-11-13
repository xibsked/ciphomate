package device

import (
	"ciphomate/internal/tuya"
	"encoding/json"
	"fmt"
	"log/slog"
)

const defaultInchingMinutes = 180

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
	path := fmt.Sprintf("/v1.0/devices/%s/status", d.DeviceID)
	resp, err := d.Client.SendRequest("GET", path, nil)
	if err != nil {
		return defaultInchingMinutes, fmt.Errorf("fetch status for device %q: %w", d.DeviceID, err)
	}
	var result StatusResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return defaultInchingMinutes, fmt.Errorf("decode status for device %q: %w", d.DeviceID, err)
	}

	slog.Debug("device status received",
		"component", "device",
		"device_id", d.DeviceID,
		"status_item_count", len(result.Result),
	)
	for _, item := range result.Result {
		if item.Code == "switch_inching" {
			enabled, seconds, err := DecodeSwitchInching(item.Value)
			if err != nil {
				slog.Warn("could not decode switch inching value; using default",
					"component", "device",
					"device_id", d.DeviceID,
					"status_code", item.Code,
					"default_minutes", defaultInchingMinutes,
					"error", err,
				)
				return defaultInchingMinutes, nil
			}

			minutes := 0
			if enabled {
				minutes = seconds / 60
			} else {
				minutes = defaultInchingMinutes
			}

			slog.Info("device inching time determined",
				"component", "device",
				"device_id", d.DeviceID,
				"enabled", enabled,
				"duration_seconds", seconds,
				"duration_minutes", minutes,
			)
			return minutes, nil

		}
	}
	slog.Warn("switch inching status missing; using default",
		"component", "device",
		"device_id", d.DeviceID,
		"default_minutes", defaultInchingMinutes,
	)
	return defaultInchingMinutes, nil
}

func (d *Device) GetCurrent() (int, error) {
	// return 0, nil
	path := fmt.Sprintf("/v1.0/devices/%s/status", d.DeviceID)
	resp, err := d.Client.SendRequest("GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("fetch status for device %q: %w", d.DeviceID, err)
	}
	var result StatusResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return 0, fmt.Errorf("decode status for device %q: %w", d.DeviceID, err)
	}

	for _, item := range result.Result {
		if item.Code == "cur_current" {
			switch value := item.Value.(type) {
			case float64:
				return int(value), nil
			case int:
				return value, nil
			default:
				return 0, fmt.Errorf("unexpected current value type for device %q: %T", d.DeviceID, item.Value)
			}
		}
	}
	return 0, fmt.Errorf("cur_current not found in status for device %q", d.DeviceID)
}

func (d *Device) Switch(on bool) error {
	cmd := CommandRequest{
		Commands: []Command{{
			Code:  "switch_1",
			Value: on,
		}},
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("encode switch command for device %q: %w", d.DeviceID, err)
	}
	path := fmt.Sprintf("/v1.0/devices/%s/commands", d.DeviceID)
	_, err = d.Client.SendRequest("POST", path, payload)
	if err != nil {
		slog.Error("device switch command failed",
			"component", "device",
			"device_id", d.DeviceID,
			"on", on,
			"error", err,
		)
		return fmt.Errorf("switch device %q to %t: %w", d.DeviceID, on, err)
	}
	slog.Info("device switch command succeeded",
		"component", "device",
		"device_id", d.DeviceID,
		"on", on,
	)
	return nil
}
