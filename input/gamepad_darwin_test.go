package input

import (
	"runtime"
	"slices"
	"testing"

	"github.com/ebitengine/purego"
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
	// Deliver real framework callbacks on a serial queue and drain it explicitly;
	// this test process does not run an application's main dispatch loop.
	var createQueue func(string, uintptr) uintptr
	var syncQueue func(uintptr, uintptr, uintptr)
	var releaseQueue func(uintptr)
	purego.RegisterLibFunc(&createQueue, purego.RTLD_DEFAULT, "dispatch_queue_create")
	purego.RegisterLibFunc(&syncQueue, purego.RTLD_DEFAULT, "dispatch_sync_f")
	purego.RegisterLibFunc(&releaseQueue, purego.RTLD_DEFAULT, "dispatch_release")
	queue := createQueue("goro.gamepad.test", 0)
	defer releaseQueue(queue)
	controller.Send(b.sel("setHandlerQueue:"), queue)
	barrier := purego.NewCallback(func(uintptr) {})
	flush := func() { syncQueue(queue, 0, barrier) }
	controllers := objc.ID(objc.GetClass("NSArray")).Send(b.sel("arrayWithObject:"), controller)
	if len(b.drainControllers(controllers)) != 1 {
		t.Fatal("controller not discovered")
	}
	device := b.devices[controller]
	profile := b.get(controller, "extendedGamepad")
	button := b.get(profile, "buttonA")
	handler := objc.Block(button.Send(b.sel("pressedChangedHandler"))).Copy()
	if handler == 0 {
		t.Fatal("button handler not installed")
	}
	defer handler.Release()
	dpad := b.get(profile, "dpad")

	// Set native element values; GameController must call our installed blocks.
	button.Send(b.sel("setValue:"), float32(1))
	dpad.Send(b.sel("setValueForXAxis:yAxis:"), float32(1), float32(0))
	button.Send(b.sel("setValue:"), float32(0))
	dpad.Send(b.sel("setValueForXAxis:yAxis:"), float32(0), float32(0))
	flush()
	axis := b.get(b.get(profile, "leftThumbstick"), "xAxis")
	axis.Send(b.sel("setValue:"), float32(0.625))
	frame := b.drainControllers(controllers)[0]
	want := []GamepadButtonChange{
		{GamepadSouth, true}, {GamepadRight, true},
		{GamepadSouth, false}, {GamepadRight, false},
	}
	if !slices.Equal(frame.Changes, want) || frame.Buttons[GamepadSouth] || frame.Axes[GamepadLeftX] != 0.625 {
		t.Fatalf("drained frame = %+v", frame)
	}
	if len(b.drainControllers(controllers)[0].Changes) != 0 {
		t.Fatal("tap replayed")
	}
	button.Send(b.sel("setValue:"), float32(1))
	button.Send(b.sel("setValue:"), float32(0))
	flush()
	if !slices.Equal(frame.Changes, want) {
		t.Fatal("later input mutated a drained frame")
	}
	b.drainControllers(0) // Disconnect with a pending tap.
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
