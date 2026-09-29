//go:build linux && !android

package input

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestEvdevDiscoverySkipsNonControllers(t *testing.T) {
	root := t.TempDir()
	devices, sysfs := filepath.Join(root, "dev"), filepath.Join(root, "sys")
	if err := os.MkdirAll(devices, 0o755); err != nil {
		t.Fatal(err)
	}
	controller := "1000000000000 0 0 0 0"
	if strconv.IntSize == 32 {
		controller = "10000 0 0 0 0 0 0 0 0 0"
	}
	for name, capabilities := range map[string]string{
		"event0": "0",                  // A switch/audio event node.
		"event1": "e520 10000 0 0 0 0", // Touchpad; opening it may wake hardware.
		"event2": "400 0 0 0 0 0",      // Touchscreen.
		"event3": controller,
		"event4": "invalid", // Unreadable/malformed metadata is not a controller.
	} {
		if err := os.WriteFile(filepath.Join(devices, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(sysfs, name, "device/capabilities")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "key"), []byte(capabilities), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{filepath.Join(devices, "event3")}
	if got := evdevGamepadPaths(devices, sysfs); !reflect.DeepEqual(got, want) {
		t.Fatalf("discovery candidates = %v, want %v", got, want)
	}
	// Repeated scans must not grow the candidate set to include other devices.
	if got := evdevGamepadPaths(devices, sysfs); !reflect.DeepEqual(got, want) {
		t.Fatalf("rescan candidates = %v", got)
	}
}

func TestEvdevSysfsCapabilityWordSizes(t *testing.T) {
	for _, test := range []struct {
		bits     string
		wordBits int
		want     bool
	}{
		{"1000000000000 0 0 0 0", 64, true},
		{"10000 0 0 0 0 0 0 0 0 0", 32, true},
		{"e520 10000 0 0 0 0", 64, false},
		{"0", 64, false},
		{"invalid 0 0 0 0", 64, false},
	} {
		if got := evdevCapability([]byte(test.bits), 0x130, test.wordBits); got != test.want {
			t.Fatalf("%+v: got %v", test, got)
		}
	}
}

func TestEvdevAxisNormalization(t *testing.T) {
	for _, test := range []struct {
		value, min, max int32
		trigger         bool
		want            float64
	}{
		{0, 0, 255, false, -1}, {255, 0, 255, false, 1}, {128, 0, 256, false, 0},
		{-32768, -32768, 32767, false, -1}, {32767, -32768, 32767, false, 1},
		{-32768, -32768, 32767, true, 0}, {32767, -32768, 32767, true, 1},
		{512, 0, 1024, true, 0.5}, {1, 0, 0, false, 0}, {300, 0, 255, true, 1},
	} {
		if got := normalizeGamepadAxis(test.value, test.min, test.max, test.trigger); got != test.want {
			t.Fatalf("%+v: got %v", test, got)
		}
	}
}
