//go:build !windows

package windows

import "EnigmaNetz/Enigma-Go-Sensor/internal/capture/common"

func NewWindowsCapturer() common.Capturer {
	panic("NewWindowsCapturer called on non-Windows platform")
}
