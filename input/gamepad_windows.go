package input

import (
	"fmt"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

type xinputBackend struct{ getState *windows.LazyProc }

func newGamepadBackend() (gamepadBackend, error) {
	for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
		proc := windows.NewLazySystemDLL(name).NewProc("XInputGetState")
		if proc.Find() == nil {
			return &xinputBackend{getState: proc}, nil
		}
	}
	return nil, fmt.Errorf("XInput is unavailable")
}

func (b *xinputBackend) poll() []GamepadSnapshot {
	var pads []GamepadSnapshot
	for index := 0; index < 4; index++ {
		var raw struct {
			Packet                       uint32
			Buttons                      uint16
			LeftTrigger, RightTrigger    uint8
			LeftX, LeftY, RightX, RightY int16
		}
		result, _, _ := b.getState.Call(uintptr(index), uintptr(unsafe.Pointer(&raw)))
		if result != 0 {
			continue
		}
		pad := GamepadSnapshot{ID: "xinput:" + strconv.Itoa(index), Name: "XInput controller " + strconv.Itoa(index+1)}
		masks := [...]uint16{0x1000, 0x2000, 0x4000, 0x8000, 0x100, 0x200, 0x20, 0x10, 0x40, 0x80, 1, 2, 4, 8}
		for button, mask := range masks {
			pad.Buttons[button] = raw.Buttons&mask != 0
		}
		pad.Axes = [GamepadAxisCount]float64{signedStick(raw.LeftX), -signedStick(raw.LeftY), signedStick(raw.RightX), -signedStick(raw.RightY), float64(raw.LeftTrigger) / 255, float64(raw.RightTrigger) / 255}
		pads = append(pads, pad)
	}
	return pads
}

func signedStick(value int16) float64 {
	if value < 0 {
		return float64(value) / 32768
	}
	return float64(value) / 32767
}

func (*xinputBackend) close() {}
