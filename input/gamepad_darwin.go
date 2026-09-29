package input

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

type gcBackend struct {
	controller objc.ID
	selectors  map[string]objc.SEL
	devices    map[objc.ID]*gcGamepad
}

type gcGamepad struct {
	controller objc.ID // Retained while handlers reference its button elements.
	buttons    [GamepadButtonCount]objc.ID
	mu         sync.Mutex
	frame      GamepadFrame
	closed     bool
}

func newGamepadBackend() (gamepadBackend, error) {
	// Use the system framework through purego, retaining CGO-free cross builds.
	// Do not dlclose an Objective-C framework after registering its classes.
	if _, err := purego.Dlopen("/System/Library/Frameworks/GameController.framework/GameController", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return nil, err
	}
	controller := objc.ID(objc.GetClass("GCController"))
	if controller == 0 {
		return nil, fmt.Errorf("GameController framework is unavailable")
	}
	return &gcBackend{controller: controller, selectors: make(map[string]objc.SEL), devices: make(map[objc.ID]*gcGamepad)}, nil
}

func (b *gcBackend) sel(name string) objc.SEL {
	if sel, ok := b.selectors[name]; ok {
		return sel
	}
	sel := objc.RegisterName(name)
	b.selectors[name] = sel
	return sel
}

func (b *gcBackend) get(object objc.ID, name string) objc.ID {
	if object == 0 {
		return 0
	}
	sel := b.sel(name)
	if object.Send(b.sel("respondsToSelector:"), sel) == 0 {
		return 0
	}
	return object.Send(sel)
}

func (b *gcBackend) value(element objc.ID) float64 {
	if element == 0 {
		return 0
	}
	return float64(objc.Send[float32](element, b.sel("value")))
}

func (b *gcBackend) drain() []GamepadFrame {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(b.sel("alloc")).Send(b.sel("init"))
	defer pool.Send(b.sel("drain"))
	return b.drainControllers(b.controller.Send(b.sel("controllers")))
}

func (b *gcBackend) drainControllers(controllers objc.ID) []GamepadFrame {
	count := int(controllers.Send(b.sel("count")))
	connected := make(map[objc.ID]bool, count)
	var pads []GamepadFrame
	for i := 0; i < count; i++ {
		controller := controllers.Send(b.sel("objectAtIndex:"), uintptr(i))
		profile := b.get(controller, "extendedGamepad")
		if profile == 0 {
			continue
		}
		connected[controller] = true
		device := b.devices[controller]
		if device == nil {
			device = b.watch(controller, profile)
			b.devices[controller] = device
		}
		left, right := b.get(profile, "leftThumbstick"), b.get(profile, "rightThumbstick")
		axes := [GamepadAxisCount]float64{
			b.value(b.get(left, "xAxis")), -b.value(b.get(left, "yAxis")),
			b.value(b.get(right, "xAxis")), -b.value(b.get(right, "yAxis")),
			b.value(b.get(profile, "leftTrigger")), b.value(b.get(profile, "rightTrigger")),
		}
		device.mu.Lock()
		device.frame.Axes = axes
		pad := device.frame
		device.frame.Changes = nil
		device.mu.Unlock()
		pads = append(pads, pad)
	}
	for controller, device := range b.devices {
		if !connected[controller] {
			b.unwatch(device)
			delete(b.devices, controller)
		}
	}
	return pads
}

func (b *gcBackend) watch(controller, profile objc.ID) *gcGamepad {
	d := &gcGamepad{
		controller: controller.Send(b.sel("retain")),
		frame:      GamepadFrame{ID: "gc:" + strconv.FormatUint(uint64(controller), 16), Name: "GameController"},
	}
	if name := b.get(controller, "vendorName"); name != 0 {
		d.frame.Name = objc.Send[string](name, b.sel("UTF8String"))
	}
	for button, name := range [...]string{"buttonA", "buttonB", "buttonX", "buttonY", "leftShoulder", "rightShoulder", "buttonOptions", "buttonMenu", "leftThumbstickButton", "rightThumbstickButton"} {
		d.buttons[button] = b.get(profile, name)
	}
	dpad := b.get(profile, "dpad")
	for i, name := range [...]string{"up", "down", "left", "right"} {
		d.buttons[int(GamepadUp)+i] = b.get(dpad, name)
	}
	for button, element := range d.buttons {
		if element == 0 {
			continue
		}
		d.frame.Buttons[button] = objc.Send[bool](element, b.sel("isPressed"))
		// Use the event's pressed value, not a later read of the element. Both
		// edges may be delivered after a slow frame has finished rendering.
		handler := objc.NewBlock(func(_ objc.Block, _ objc.ID, _ float32, pressed bool) {
			d.mu.Lock()
			defer d.mu.Unlock()
			if !d.closed {
				d.frame.setButton(GamepadButton(button), pressed)
			}
		})
		element.Send(b.sel("setPressedChangedHandler:"), handler)
		handler.Release() // The property copies/retains the block.
	}
	return d
}

func (b *gcBackend) unwatch(d *gcGamepad) {
	d.mu.Lock()
	d.closed = true // Reject any callback already queued during disconnection.
	d.frame.Changes = nil
	d.mu.Unlock()
	for _, element := range d.buttons {
		if element != 0 {
			element.Send(b.sel("setPressedChangedHandler:"), objc.Block(0))
		}
	}
	d.controller.Send(b.sel("release"))
}

func (b *gcBackend) close() {
	for controller, device := range b.devices {
		b.unwatch(device)
		delete(b.devices, controller)
	}
}
