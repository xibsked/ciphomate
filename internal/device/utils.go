package device

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

func DecodeSwitchInching(value interface{}) (enabled bool, durationSeconds int, err error) {
	var base64Val string

	switch v := value.(type) {
	case string:
		base64Val = v
	case []byte:
		base64Val = string(v)
	default:
		return false, 0, fmt.Errorf("unexpected type for switch_inching value: %T", value)
	}

	data, err := base64.StdEncoding.DecodeString(base64Val)
	if err != nil {
		return false, 0, fmt.Errorf("base64 decode error: %w", err)
	}

	if len(data) < 3 {
		return false, 0, fmt.Errorf("unexpected data length: got %d bytes", len(data))
	}

	enabled = data[0] == 1
	durationRawSeconds := binary.BigEndian.Uint16(data[1:3])
	durationSeconds = int(durationRawSeconds)

	return enabled, durationSeconds, nil
}
