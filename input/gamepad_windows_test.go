package input

import (
	"testing"
	"time"
)

func TestXInputThrottlesEmptySlotsAndRepollsConnectedSlots(t *testing.T) {
	var calls [4]int
	connected := [4]bool{true}
	backend := &xinputBackend{getState: func(index uintptr, raw *xinputState) uintptr {
		calls[index]++
		if !connected[index] {
			return 1167 // ERROR_DEVICE_NOT_CONNECTED
		}
		raw.Buttons = 0x1000
		return 0
	}}
	now := time.Now()
	if pads := backend.pollAt(now); len(pads) != 1 || !pads[0].Buttons[GamepadSouth] {
		t.Fatal("connected slot was not sampled")
	}
	if calls != [4]int{1, 1, 1, 1} {
		t.Fatalf("initial discovery calls = %v", calls)
	}
	connected[1] = true
	backend.pollAt(now.Add(time.Millisecond))
	backend.pollAt(now.Add(time.Second))
	if calls != [4]int{3, 1, 1, 1} {
		t.Fatalf("empty slots were retried too soon: %v", calls)
	}
	if pads := backend.pollAt(now.Add(2 * time.Second)); len(pads) != 2 {
		t.Fatal("new controller was not discovered after the retry interval")
	}
	connected[0] = false
	if pads := backend.pollAt(now.Add(2*time.Second + time.Millisecond)); len(pads) != 1 || pads[0].ID != "xinput:1" {
		t.Fatal("disconnected controller remained in the sample")
	}
	before := calls
	backend.pollAt(now.Add(3 * time.Second))
	if calls[0] != before[0] || calls[1] != before[1]+1 || calls[2] != before[2] || calls[3] != before[3] {
		t.Fatalf("disconnect did not restore the retry delay: %v -> %v", before, calls)
	}
	connected[0] = true
	if pads := backend.pollAt(now.Add(5 * time.Second)); len(pads) != 2 {
		t.Fatal("controller did not reconnect")
	}
}
