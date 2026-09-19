package input

import (
	"github.com/gogpu/gpucontext"
	"testing"
)

func TestAndroidKey(t *testing.T) {
	for _, tc := range []struct {
		code int
		want gpucontext.Key
	}{
		{29, gpucontext.KeyA}, {54, gpucontext.KeyZ},
		{7, gpucontext.Key0}, {16, gpucontext.Key9},
		{131, gpucontext.KeyF1}, {142, gpucontext.KeyF12},
		{4, gpucontext.KeyEscape}, {111, gpucontext.KeyEscape},
		{66, gpucontext.KeyEnter}, {61, gpucontext.KeyTab},
		{67, gpucontext.KeyBackspace}, {112, gpucontext.KeyDelete},
		{19, gpucontext.KeyUp}, {22, gpucontext.KeyRight},
		{59, gpucontext.KeyLeftShift}, {114, gpucontext.KeyRightControl},
		{0, gpucontext.KeyUnknown}, {28, gpucontext.KeyUnknown},
		{143, gpucontext.KeyUnknown}, {999, gpucontext.KeyUnknown},
	} {
		if got := AndroidKey(tc.code); got != tc.want {
			t.Errorf("AndroidKey(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}
