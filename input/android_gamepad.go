package input

import (
	"sort"
	"strconv"
	"sync"
)

// JNI calls arrive on Android's UI thread; the game update takes snapshots.
var androidGamepads = struct {
	sync.Mutex
	pads map[int]androidGamepad
}{pads: make(map[int]androidGamepad)}

type androidGamepad struct {
	GamepadSnapshot
	keys     [GamepadButtonCount]bool
	hat      [4]bool
	triggers [2]bool
}

func AndroidGamepadDevice(id int, name string, connected bool) {
	androidGamepads.Lock()
	defer androidGamepads.Unlock()
	if !connected {
		delete(androidGamepads.pads, id)
		return
	}
	pad := androidGamepads.pads[id]
	pad.ID, pad.Name = "android:"+strconv.Itoa(id), name
	androidGamepads.pads[id] = pad
}

func AndroidGamepadKey(id, key int, down bool) {
	button, ok := AndroidGamepadButton(key)
	if !ok && key != 104 && key != 105 {
		return
	}
	androidGamepads.Lock()
	defer androidGamepads.Unlock()
	pad, exists := androidGamepads.pads[id]
	if !exists {
		return
	}
	if ok {
		pad.keys[button] = down
	} else {
		pad.triggers[key-104] = down
	}
	androidGamepads.pads[id] = pad
}

func AndroidGamepadMotion(id int, axes [GamepadAxisCount]float64, hatX, hatY float64) {
	androidGamepads.Lock()
	defer androidGamepads.Unlock()
	pad, exists := androidGamepads.pads[id]
	if !exists {
		return
	}
	pad.Axes = axes
	pad.hat = [4]bool{hatY < -0.5, hatY > 0.5, hatX < -0.5, hatX > 0.5}
	androidGamepads.pads[id] = pad
}

func AndroidResetGamepads() {
	androidGamepads.Lock()
	defer androidGamepads.Unlock()
	clear(androidGamepads.pads)
}

type androidGamepadBackend struct{}

func (*androidGamepadBackend) close() {}
func (*androidGamepadBackend) poll() []GamepadSnapshot {
	androidGamepads.Lock()
	defer androidGamepads.Unlock()
	ids := make([]int, 0, len(androidGamepads.pads))
	for id := range androidGamepads.pads {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	pads := make([]GamepadSnapshot, 0, len(ids))
	for _, id := range ids {
		device := androidGamepads.pads[id]
		pad := device.GamepadSnapshot
		pad.Buttons = device.keys
		for i, down := range device.hat {
			pad.Buttons[int(GamepadUp)+i] = pad.Buttons[int(GamepadUp)+i] || down
		}
		for i, down := range device.triggers {
			if down {
				pad.Axes[int(GamepadLeftTrigger)+i] = 1
			}
		}
		pads = append(pads, pad)
	}
	return pads
}
