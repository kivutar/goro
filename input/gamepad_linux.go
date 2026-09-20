//go:build linux && !android

package input

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/kivutar/goro/glog"
	"golang.org/x/sys/unix"
)

// evdev uses the kernel's positional gamepad mapping, not joystick indices.
// Polling EVIOCGKEY/EVIOCGABS also recovers correctly after dropped events.
type evdevBackend struct {
	devices  []*evdevGamepad
	nextScan time.Time
	warned   bool
}
type evdevGamepad struct {
	file *os.File
	name string
	axes [64]bool
}

func newGamepadBackend() (gamepadBackend, error) {
	return newPollingGamepads(&evdevBackend{}), nil
}

func evdevRead(fd uintptr, number uint, data []byte) error {
	request := uintptr(0x80000000 | uint(len(data))<<16 | uint('E')<<8 | number)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(&data[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

func evdevBit(bits []byte, code int) bool { return bits[code/8]&(1<<uint(code%8)) != 0 }

// Opening an event node can wake its hardware and block in the driver even
// with O_NONBLOCK. Read capabilities from sysfs first so controller discovery
// never opens unrelated keyboards, touchpads, touchscreens or audio devices.
func evdevGamepadPaths(deviceRoot, sysfsRoot string) []string {
	paths, _ := filepath.Glob(filepath.Join(deviceRoot, "event*"))
	gamepads := paths[:0]
	for _, path := range paths {
		bits, err := os.ReadFile(filepath.Join(sysfsRoot, filepath.Base(path), "device/capabilities/key"))
		if err == nil && evdevCapability(bits, 0x130, strconv.IntSize) {
			gamepads = append(gamepads, path)
		}
	}
	return gamepads
}

// Sysfs prints a bitmap as space-separated unsigned-long words, highest first.
// Supported Linux release targets (amd64 and arm64) both use 64-bit words.
func evdevCapability(data []byte, bit, wordBits int) bool {
	words := strings.Fields(string(data))
	index := len(words) - 1 - bit/wordBits
	if index < 0 {
		return false
	}
	word, err := strconv.ParseUint(words[index], 16, wordBits)
	return err == nil && word&(uint64(1)<<uint(bit%wordBits)) != 0
}

func (b *evdevBackend) scan() {
	paths := evdevGamepadPaths("/dev/input", "/sys/class/input")
	for _, path := range paths {
		known := false
		for _, device := range b.devices {
			if device.file.Name() == path {
				known = true
				break
			}
		}
		if known {
			continue
		}
		file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK, 0)
		if err != nil {
			if os.IsPermission(err) && !b.warned {
				glog.Warnf("gamepad cannot read %s: %v", path, err)
				b.warned = true
			}
			continue
		}
		var keys [96]byte
		if evdevRead(file.Fd(), 0x21, keys[:]) != nil || !evdevBit(keys[:], 0x130) {
			file.Close()
			continue
		}
		device := &evdevGamepad{file: file}
		var name [128]byte
		_ = evdevRead(file.Fd(), 0x06, name[:])
		device.name = string(bytes.TrimRight(name[:], "\x00"))
		var axes [8]byte
		_ = evdevRead(file.Fd(), 0x23, axes[:])
		for axis := range device.axes {
			device.axes[axis] = evdevBit(axes[:], axis)
		}
		b.devices = append(b.devices, device)
	}
}

func (b *evdevBackend) poll() []GamepadSnapshot {
	if time.Now().After(b.nextScan) {
		b.scan()
		b.nextScan = time.Now().Add(time.Second)
	}
	var pads []GamepadSnapshot
	alive := b.devices[:0]
	for _, device := range b.devices {
		pad, err := device.snapshot()
		if err != nil {
			device.file.Close()
			continue
		}
		alive = append(alive, device)
		pads = append(pads, pad)
	}
	b.devices = alive
	return pads
}

func (d *evdevGamepad) snapshot() (GamepadSnapshot, error) {
	pad := GamepadSnapshot{ID: d.file.Name(), Name: d.name}
	var keys [96]byte
	if err := evdevRead(d.file.Fd(), 0x18, keys[:]); err != nil {
		return pad, err
	}
	codes := [...]int{0x130, 0x131, 0x134, 0x133, 0x136, 0x137, 0x13a, 0x13b, 0x13d, 0x13e, 0x220, 0x221, 0x222, 0x223}
	for button, code := range codes {
		pad.Buttons[button] = evdevBit(keys[:], code)
	}
	// Modern drivers use RX/RY for the right stick; older DInput drivers use Z/RZ.
	rightX, rightY, leftTrigger, rightTrigger := 3, 4, 2, 5
	if !d.axes[3] || !d.axes[4] {
		rightX, rightY, leftTrigger, rightTrigger = 2, 5, -1, -1
	}
	for axis, code := range [...]int{0, 1, rightX, rightY, leftTrigger, rightTrigger} {
		value, err := d.axis(code, GamepadAxis(axis) >= GamepadLeftTrigger)
		if err != nil {
			return pad, err
		}
		pad.Axes[axis] = value
	}
	x, err := d.axis(16, false)
	if err != nil {
		return pad, err
	}
	y, err := d.axis(17, false)
	if err != nil {
		return pad, err
	}
	pad.Buttons[GamepadLeft] = pad.Buttons[GamepadLeft] || x < -0.5
	pad.Buttons[GamepadRight] = pad.Buttons[GamepadRight] || x > 0.5
	pad.Buttons[GamepadUp] = pad.Buttons[GamepadUp] || y < -0.5
	pad.Buttons[GamepadDown] = pad.Buttons[GamepadDown] || y > 0.5
	if evdevBit(keys[:], 0x138) {
		pad.Axes[GamepadLeftTrigger] = 1
	}
	if evdevBit(keys[:], 0x139) {
		pad.Axes[GamepadRightTrigger] = 1
	}
	return pad, nil
}

func (d *evdevGamepad) axis(code int, trigger bool) (float64, error) {
	if code < 0 || !d.axes[code] {
		return 0, nil
	}
	var abs [24]byte // struct input_absinfo: value, min, max, fuzz, flat, resolution
	if err := evdevRead(d.file.Fd(), uint(0x40+code), abs[:]); err != nil {
		return 0, err
	}
	value := int32(binary.NativeEndian.Uint32(abs[0:4]))
	minimum := int32(binary.NativeEndian.Uint32(abs[4:8]))
	maximum := int32(binary.NativeEndian.Uint32(abs[8:12]))
	return normalizeGamepadAxis(value, minimum, maximum, trigger), nil
}

func normalizeGamepadAxis(value, minimum, maximum int32, trigger bool) float64 {
	if maximum <= minimum {
		return 0
	}
	value01 := (float64(value) - float64(minimum)) / (float64(maximum) - float64(minimum))
	value01 = max(0, min(1, value01))
	if trigger {
		return value01
	}
	return value01*2 - 1
}

func (b *evdevBackend) close() {
	for _, device := range b.devices {
		device.file.Close()
	}
	b.devices = nil
}
