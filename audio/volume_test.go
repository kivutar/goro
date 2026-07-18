//go:build nofakecgo

package audio

import "testing"

func TestBGMAndSFXVolumesAreSeparate(t *testing.T) {
	bgm := NewBGM(nil, true, 0.25, 0.75, false)
	if bgm.BGMVolume() != 0.25 {
		t.Fatalf("bgm volume = %v, want 0.25", bgm.BGMVolume())
	}
	if bgm.SFXVolume() != 0.75 {
		t.Fatalf("sfx volume = %v, want 0.75", bgm.SFXVolume())
	}

	bgm.SetBGMVolume(0.4)
	if bgm.BGMVolume() != 0.4 || bgm.SFXVolume() != 0.75 {
		t.Fatalf("volumes after bgm change = %v/%v", bgm.BGMVolume(), bgm.SFXVolume())
	}
	bgm.SetSFXVolume(0.6)
	if bgm.BGMVolume() != 0.4 || bgm.SFXVolume() != 0.6 {
		t.Fatalf("volumes after sfx change = %v/%v", bgm.BGMVolume(), bgm.SFXVolume())
	}
}

func TestAudioVolumesClamp(t *testing.T) {
	bgm := NewBGM(nil, true, -1, 2, false)
	if bgm.BGMVolume() != 0 {
		t.Fatalf("bgm volume = %v, want 0", bgm.BGMVolume())
	}
	if bgm.SFXVolume() != 1 {
		t.Fatalf("sfx volume = %v, want 1", bgm.SFXVolume())
	}
}

func TestMutedSFXDoesNotInitializeAudio(t *testing.T) {
	bgm := NewBGM(nil, false, 0, 0, false)
	path, err := bgm.PlaySFXVolume("missing.wav", 1)
	if err != nil {
		t.Fatalf("muted SFX returned error: %v", err)
	}
	if path != "" {
		t.Fatalf("muted SFX path = %q, want empty", path)
	}
}

func TestDisabledAudioCannotBeReenabled(t *testing.T) {
	bgm := NewBGM(nil, true, 0.5, 0.5, true)
	if bgm.Enabled() {
		t.Fatal("disabled audio reported BGM enabled")
	}
	bgm.SetEnabled(true)
	if bgm.Enabled() {
		t.Fatal("disabled audio was reenabled")
	}
	path, err := bgm.PlaySFXVolume("missing.wav", 1)
	if err != nil {
		t.Fatalf("disabled SFX returned error: %v", err)
	}
	if path != "" {
		t.Fatalf("disabled SFX path = %q, want empty", path)
	}
}
