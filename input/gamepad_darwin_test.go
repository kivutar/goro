package input

import (
	"runtime"
	"slices"
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestGameControllerQueuesNativeButtonCallbacks(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	backend, err := newGamepadBackend()
	if err != nil {
		t.Fatal(err)
	}
	b := backend.(*gcBackend)
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(b.sel("alloc")).Send(b.sel("init"))
	defer pool.Send(b.sel("drain"))
	defer b.close()

	// Apple's mutable snapshot supplies real GameController button objects,
	// without requiring a physical controller on the macOS CI runner.
	controller := b.controller.Send(b.sel("controllerWithExtendedGamepad"))
	if controller == 0 {
		t.Fatal("could not create a controller snapshot")
	}
	controllers := objc.ID(objc.GetClass("NSArray")).Send(b.sel("arrayWithObject:"), controller)
	if len(b.drainControllers(controllers)) != 1 {
		t.Fatal("controller not discovered")
	}
	device := b.devices[controller]
	button := device.buttons[GamepadSouth]
	handler := objc.Block(button.Send(b.sel("pressedChangedHandler"))).Copy()
	if handler == 0 {
		t.Fatal("button handler not installed")
	}
	defer handler.Release()
	right := device.buttons[GamepadRight]
	rightHandler := objc.Block(right.Send(b.sel("pressedChangedHandler")))
	if rightHandler == 0 {
		t.Fatal("D-pad handler not installed")
	}

	// Both presses and releases arrive before the game drains its next frame.
	handler.Invoke(button, float32(1), true)
	rightHandler.Invoke(right, float32(1), true)
	handler.Invoke(button, float32(0), false)
	rightHandler.Invoke(right, float32(0), false)
	handler.Invoke(button, float32(1), true)
	handler.Invoke(button, float32(0), false)
	profile := b.get(controller, "extendedGamepad")
	axis := b.get(b.get(profile, "leftThumbstick"), "xAxis")
	axis.Send(b.sel("setValue:"), float32(0.625))
	frame := b.drainControllers(controllers)[0]
	want := []GamepadButtonChange{
		{GamepadSouth, true}, {GamepadRight, true},
		{GamepadSouth, false}, {GamepadRight, false},
		{GamepadSouth, true}, {GamepadSouth, false},
	}
	if !slices.Equal(frame.Changes, want) || frame.Buttons[GamepadSouth] || frame.Axes[GamepadLeftX] != 0.625 {
		t.Fatalf("drained frame = %+v", frame)
	}
	if len(b.drainControllers(controllers)[0].Changes) != 0 {
		t.Fatal("tap replayed")
	}
	handler.Invoke(button, float32(0), true) // The pressed parameter is authoritative.
	held := b.drainControllers(controllers)[0]
	if !held.Buttons[GamepadSouth] || !slices.Equal(held.Changes, []GamepadButtonChange{{GamepadSouth, true}}) {
		t.Fatal("callback state was overwritten by a polled button value")
	}
	if !slices.Equal(frame.Changes, want) {
		t.Fatal("later callbacks mutated a drained frame")
	}

	handler.Invoke(button, float32(0), false)
	b.drainControllers(0) // Disconnect with a pending release.
	if len(b.devices) != 0 || button.Send(b.sel("pressedChangedHandler")) != 0 {
		t.Fatal("disconnect retained controller or callback")
	}
	handler.Invoke(button, float32(1), true) // Already queued when detached.
	if len(device.frame.Changes) != 0 {
		t.Fatal("late callback recreated pending input")
	}
	if frame := b.drainControllers(controllers)[0]; len(frame.Changes) != 0 || frame.Buttons[GamepadSouth] {
		t.Fatal("reconnect inherited the old controller's pending input")
	}
	b.close()
	if button.Send(b.sel("pressedChangedHandler")) != 0 {
		t.Fatal("close retained callback")
	}
	b.close() // Idempotent cleanup.
}
