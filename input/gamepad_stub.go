//go:build !windows && !linux && !darwin && !android

package input

type emptyGamepadBackend struct{}

func newGamepadBackend() (gamepadBackend, error)    { return emptyGamepadBackend{}, nil }
func (emptyGamepadBackend) poll() []GamepadSnapshot { return nil }
func (emptyGamepadBackend) close()                  {}
