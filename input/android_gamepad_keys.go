package input

// AndroidGamepadButton maps Android KeyEvent constants to positional controls.
func AndroidGamepadButton(key int) (GamepadButton, bool) {
	switch key {
	case 96:
		return GamepadSouth, true
	case 97:
		return GamepadEast, true
	case 99:
		return GamepadWest, true
	case 100:
		return GamepadNorth, true
	case 102:
		return GamepadLeftShoulder, true
	case 103:
		return GamepadRightShoulder, true
	case 109:
		return GamepadBack, true
	case 108:
		return GamepadStart, true
	case 106:
		return GamepadLeftStick, true
	case 107:
		return GamepadRightStick, true
	case 19:
		return GamepadUp, true
	case 20:
		return GamepadDown, true
	case 21:
		return GamepadLeft, true
	case 22:
		return GamepadRight, true
	}
	return 0, false
}
