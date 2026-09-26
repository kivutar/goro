package input

func newGamepadBackend() (gamepadBackend, error) { return &androidGamepadBackend{}, nil }
