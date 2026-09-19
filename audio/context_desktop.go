//go:build nofakecgo && !android

package audio

import "github.com/ebitengine/oto/v3"

func newAudioContext(options *oto.NewContextOptions) (*oto.Context, chan struct{}, int, error) {
	context, ready, err := oto.NewContext(options)
	return context, ready, options.SampleRate, err
}
