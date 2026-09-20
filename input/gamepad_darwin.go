package input

import (
	"fmt"
	"strconv"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

type gcBackend struct {
	controller objc.ID
	selectors  map[string]objc.SEL
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
	return &gcBackend{controller: controller, selectors: make(map[string]objc.SEL)}, nil
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

func (b *gcBackend) poll() []GamepadSnapshot {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(b.sel("alloc")).Send(b.sel("init"))
	defer pool.Send(b.sel("drain"))
	controllers := b.controller.Send(b.sel("controllers"))
	count := int(controllers.Send(b.sel("count")))
	var pads []GamepadSnapshot
	for i := 0; i < count; i++ {
		controller := controllers.Send(b.sel("objectAtIndex:"), uintptr(i))
		profile := b.get(controller, "extendedGamepad")
		if profile == 0 {
			continue
		}
		pad := GamepadSnapshot{ID: "gc:" + strconv.FormatUint(uint64(controller), 16), Name: "GameController"}
		if name := b.get(controller, "vendorName"); name != 0 {
			pad.Name = objc.Send[string](name, b.sel("UTF8String"))
		}
		buttons := [...]string{"buttonA", "buttonB", "buttonX", "buttonY", "leftShoulder", "rightShoulder", "buttonOptions", "buttonMenu", "leftThumbstickButton", "rightThumbstickButton"}
		for button, name := range buttons {
			pad.Buttons[button] = b.value(b.get(profile, name)) > 0.5
		}
		dpad := b.get(profile, "dpad")
		for i, name := range [...]string{"up", "down", "left", "right"} {
			pad.Buttons[int(GamepadUp)+i] = b.value(b.get(dpad, name)) > 0.5
		}
		left, right := b.get(profile, "leftThumbstick"), b.get(profile, "rightThumbstick")
		pad.Axes = [GamepadAxisCount]float64{
			b.value(b.get(left, "xAxis")), -b.value(b.get(left, "yAxis")),
			b.value(b.get(right, "xAxis")), -b.value(b.get(right, "yAxis")),
			b.value(b.get(profile, "leftTrigger")), b.value(b.get(profile, "rightTrigger")),
		}
		pads = append(pads, pad)
	}
	return pads
}

func (*gcBackend) close() {}
