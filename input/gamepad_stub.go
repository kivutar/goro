//go:build !windows && !linux && !darwin && !android

package input

type emptyGamepadBackend struct{}

func newGamepadBackend() (gamepadBackend, error)  { return emptyGamepadBackend{}, nil }
func (emptyGamepadBackend) drain() []GamepadFrame { return nil }
func (emptyGamepadBackend) close()                {}
