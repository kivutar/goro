package input

import "github.com/gogpu/gpucontext"

// AndroidKey translates Android KeyEvent keycodes to the shared input model.
func AndroidKey(code int) gpucontext.Key {
	switch {
	case code >= 29 && code <= 54:
		return gpucontext.KeyA + gpucontext.Key(code-29)
	case code >= 7 && code <= 16:
		return gpucontext.Key0 + gpucontext.Key(code-7)
	case code >= 131 && code <= 142:
		return gpucontext.KeyF1 + gpucontext.Key(code-131)
	}
	switch code {
	case 4, 111:
		return gpucontext.KeyEscape
	case 19:
		return gpucontext.KeyUp
	case 20:
		return gpucontext.KeyDown
	case 21:
		return gpucontext.KeyLeft
	case 22:
		return gpucontext.KeyRight
	case 23, 66:
		return gpucontext.KeyEnter
	case 61:
		return gpucontext.KeyTab
	case 62:
		return gpucontext.KeySpace
	case 67:
		return gpucontext.KeyBackspace
	case 112:
		return gpucontext.KeyDelete
	case 92:
		return gpucontext.KeyPageUp
	case 93:
		return gpucontext.KeyPageDown
	case 122:
		return gpucontext.KeyHome
	case 123:
		return gpucontext.KeyEnd
	case 124:
		return gpucontext.KeyInsert
	case 59:
		return gpucontext.KeyLeftShift
	case 60:
		return gpucontext.KeyRightShift
	case 113:
		return gpucontext.KeyLeftControl
	case 114:
		return gpucontext.KeyRightControl
	case 57:
		return gpucontext.KeyLeftAlt
	case 58:
		return gpucontext.KeyRightAlt
	case 55:
		return gpucontext.KeyComma
	case 56:
		return gpucontext.KeyPeriod
	case 69:
		return gpucontext.KeyMinus
	case 70:
		return gpucontext.KeyEqual
	case 71:
		return gpucontext.KeyLeftBracket
	case 72:
		return gpucontext.KeyRightBracket
	case 73:
		return gpucontext.KeyBackslash
	case 74:
		return gpucontext.KeySemicolon
	case 75:
		return gpucontext.KeyApostrophe
	case 76:
		return gpucontext.KeySlash
	case 68:
		return gpucontext.KeyGrave
	}
	return gpucontext.KeyUnknown
}
