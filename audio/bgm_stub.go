//go:build !nofakecgo

package audio

import "github.com/kivutar/goro/res"

type BGM struct {
	bgmVolume float64
	sfxVolume float64
	disabled  bool
}

func NewBGM(_ *res.Manager, _ bool, bgmVolume, sfxVolume float64, disabled bool) *BGM {
	return &BGM{
		bgmVolume: clampVolume(bgmVolume),
		sfxVolume: clampVolume(sfxVolume),
		disabled:  disabled,
	}
}

func (b *BGM) Enabled() bool {
	return false
}

func (b *BGM) SetEnabled(bool) {}

func (b *BGM) Volume() float64 {
	return b.BGMVolume()
}

func (b *BGM) BGMVolume() float64 {
	if b == nil {
		return 0
	}
	return b.bgmVolume
}

func (b *BGM) SFXVolume() float64 {
	if b == nil {
		return 0
	}
	return b.sfxVolume
}

func (b *BGM) SetVolume(volume float64) {
	b.SetBGMVolume(volume)
	b.SetSFXVolume(volume)
}

func (b *BGM) SetBGMVolume(volume float64) {
	if b != nil {
		b.bgmVolume = clampVolume(volume)
	}
}

func (b *BGM) SetSFXVolume(volume float64) {
	if b != nil {
		b.sfxVolume = clampVolume(volume)
	}
}

func (b *BGM) PlayMap(string) (string, error) {
	return "", nil
}

func (b *BGM) Play(string) error {
	return nil
}

func (b *BGM) PlaySFX(path string) (string, error) {
	return b.PlaySFXVolume(path, 1)
}

func (b *BGM) PlaySFXVolume(string, float64) (string, error) {
	return "", nil
}

func (b *BGM) Stop() {}
