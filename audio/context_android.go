package audio

import (
	"sync"

	"github.com/ebitengine/oto/v3"
)

// Android may recreate the game on each Activity resume. Oto permits only
// one context per process, so keep its device and sample rate across games.
var androidAudio struct {
	sync.Mutex
	context    *oto.Context
	ready      chan struct{}
	sampleRate int
}

func newAudioContext(options *oto.NewContextOptions) (*oto.Context, chan struct{}, int, error) {
	androidAudio.Lock()
	defer androidAudio.Unlock()
	if androidAudio.context == nil {
		context, ready, err := oto.NewContext(options)
		if err != nil {
			return nil, nil, 0, err
		}
		androidAudio.context = context
		androidAudio.ready = ready
		androidAudio.sampleRate = options.SampleRate
	}
	return androidAudio.context, androidAudio.ready, androidAudio.sampleRate, nil
}
