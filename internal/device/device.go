package device

import (
	"ciphomate/internal/tuya"
	"encoding/json"
	"fmt"
	"log"
)

type StatusResponse struct {
	Result []struct {
		Code  string      `json:"code"`
		Value interface{} `json:"value"`
	} `json:"result"`
}

type Command struct {
	Code  string      `json:"code"`
	Value interface{} `json:"value"`
}

type CommandRequest struct {
	Commands []Command `json:"commands"`
}

func FetchInchingTime() (int, error) {
	// return 15, nil
	path := fmt.Sprintf("/v1.0/devices/%s/status", tuya.DeviceID)
	resp, err := tuya.SendRequest("GET", path, nil)
	if err != nil {
		return 60, err
	}
	var result StatusResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return 60, err
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
	return 60, fmt.Errorf("switch_inching not found in status")
}

func GetCurrent() (int, error) {
	// return 0, nil
	path := fmt.Sprintf("/v1.0/devices/%s/status", tuya.DeviceID)
	resp, err := tuya.SendRequest("GET", path, nil)
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

func Switch(on bool) error {
	// return nil
	cmd := CommandRequest{
		Commands: []Command{{
			Code:  "switch_1",
			Value: on,
		}},
	}
	payload, _ := json.Marshal(cmd)
	path := fmt.Sprintf("/v1.0/devices/%s/commands", tuya.DeviceID)
	_, err := tuya.SendRequest("POST", path, payload)
	return err
}
