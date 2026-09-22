//go:build nofakecgo

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kivutar/goro/res"
)

func (b *BGM) SFXVolume() float64 {
	if b == nil {
		return 0
	}
	return b.sfxVolume
}

func (b *BGM) SetSFXVolume(volume float64) {
	if b == nil {
		return
	}
	volume = clampVolume(volume)
	b.sfxVolume = volume
	for _, player := range b.sfxPlayers {
		if player != nil {
			player.SetVolume(volume)
		}
	}
}

func (b *BGM) PlaySFX(path string) (string, error) {
	return b.PlaySFXVolume(path, 1)
}

// PreloadSFX prepares a sound without playing it or opening the audio device.
// Call it during map loading for sounds that must be ready during gameplay.
func (b *BGM) PreloadSFX(path string) error {
	if b == nil || b.disabled || b.sfxVolume <= 0 {
		return nil
	}
	path = normalizeSFXPath(path)
	if path == "" {
		return nil
	}
	_, err := b.loadSFX(path, b.sfxSampleRate())
	return err
}

func (b *BGM) sfxSampleRate() int {
	if b.sampleRate > 0 {
		return b.sampleRate
	}
	return defaultSampleRate
}

func (b *BGM) PlaySFXVolume(path string, volume float64) (string, error) {
	if b == nil || b.disabled || b.sfxVolume <= 0 || volume <= 0 {
		return "", nil
	}
	path = normalizeSFXPath(path)
	if path == "" {
		return "", nil
	}
	sampleRate := b.sfxSampleRate()
	sound, err := b.loadSFX(path, sampleRate)
	if err != nil {
		return sound.source, err
	}
	context := b.ensureContext(sampleRate)
	if context == nil {
		return sound.source, fmt.Errorf("audio context unavailable")
	}
	player := context.NewPlayer(bytes.NewReader(sound.pcm))
	player.SetVolume(b.sfxVolume * clampVolume(volume))
	b.trimSFXPlayers()
	b.sfxPlayers = append(b.sfxPlayers, player)
	player.Play()
	return sound.source, nil
}

func (b *BGM) loadSFX(path string, sampleRate int) (decodedSFX, error) {
	key := sfxCacheKey{path: strings.ToLower(normalizeSFXPath(path)), sampleRate: sampleRate}
	if sound, ok := b.sfxCache.get(key); ok {
		return sound, nil
	}
	data, source, err := readSFXFile(b.resources, path)
	if err != nil {
		return decodedSFX{}, err
	}
	sound := decodedSFX{source: source}
	pcm, sourceRate, err := decodeWAVToPCM16Stereo(data)
	if err != nil {
		return sound, fmt.Errorf("decode sfx %s: %w", source, err)
	}
	if sourceRate != sampleRate {
		pcm, err = resamplePCM16Stereo(pcm, sourceRate, sampleRate)
		if err != nil {
			return sound, fmt.Errorf("resample sfx %s: %w", source, err)
		}
	}
	sound.pcm = pcm
	b.sfxCache.put(key, sound)
	return sound, nil
}

func (b *BGM) trimSFXPlayers() {
	if len(b.sfxPlayers) == 0 {
		return
	}
	active := b.sfxPlayers[:0]
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		if player.IsPlaying() {
			active = append(active, player)
			continue
		}
	}
	const maxSFXPlayers = 32
	for len(active) > maxSFXPlayers {
		player := active[0]
		active = active[1:]
		if player != nil {
			player.Pause()
		}
	}
	b.sfxPlayers = active
}

func (b *BGM) stopSFX() {
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		player.Pause()
	}
	b.sfxPlayers = nil
}

func decodeWAVToPCM16Stereo(data []byte) ([]byte, int, error) {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	pos := 12
	var channels, bitsPerSample uint16
	var sampleRate uint32
	var pcm []byte
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(data) {
			return nil, 0, fmt.Errorf("invalid wav chunk %q size %d", id, size)
		}
		chunk := data[pos : pos+size]
		switch id {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, 0, fmt.Errorf("short fmt chunk")
			}
			format := binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(chunk[14:16])
			if format != 1 {
				return nil, 0, fmt.Errorf("unsupported wav format %d", format)
			}
		case "data":
			pcm = append([]byte(nil), chunk...)
		}
		pos += size
		if pos%2 != 0 {
			pos++
		}
	}
	if sampleRate == 0 || len(pcm) == 0 {
		return nil, 0, fmt.Errorf("missing wav fmt or data")
	}
	switch bitsPerSample {
	case 8:
		var err error
		pcm, err = pcm8ToPCM16Stereo(pcm, channels)
		if err != nil {
			return nil, 0, err
		}
	case 16:
		var err error
		pcm, err = pcm16ToStereo(pcm, channels)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fmt.Errorf("unsupported wav bits per sample %d", bitsPerSample)
	}
	return pcm, int(sampleRate), nil
}

func pcm8ToPCM16Stereo(src []byte, channels uint16) ([]byte, error) {
	switch channels {
	case 1:
		dst := make([]byte, len(src)*4)
		for frame, sample := range src {
			value := unsignedPCM8ToPCM16(sample)
			writePCM16(dst, frame, 0, value)
			writePCM16(dst, frame, 1, value)
		}
		return dst, nil
	case 2:
		if len(src)%2 != 0 {
			return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
		}
		dst := make([]byte, len(src)*2)
		for frame := 0; frame < len(src)/2; frame++ {
			writePCM16(dst, frame, 0, unsignedPCM8ToPCM16(src[frame*2]))
			writePCM16(dst, frame, 1, unsignedPCM8ToPCM16(src[frame*2+1]))
		}
		return dst, nil
	default:
		return nil, fmt.Errorf("unsupported wav channels %d", channels)
	}
}

func pcm16ToStereo(src []byte, channels uint16) ([]byte, error) {
	switch channels {
	case 1:
		return monoPCM16ToStereo(src)
	case 2:
		if len(src)%4 != 0 {
			return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
		}
		return src, nil
	default:
		return nil, fmt.Errorf("unsupported wav channels %d", channels)
	}
}

func monoPCM16ToStereo(src []byte) ([]byte, error) {
	if len(src)%2 != 0 {
		return nil, fmt.Errorf("invalid mono pcm length %d", len(src))
	}
	dst := make([]byte, len(src)*2)
	for frame := 0; frame < len(src)/2; frame++ {
		sample := int16(uint16(src[frame*2]) | uint16(src[frame*2+1])<<8)
		writePCM16(dst, frame, 0, sample)
		writePCM16(dst, frame, 1, sample)
	}
	return dst, nil
}

func unsignedPCM8ToPCM16(sample byte) int16 {
	return int16((int(sample) - 128) << 8)
}

func readSFXFile(manager *res.Manager, path string) ([]byte, string, error) {
	if manager == nil {
		return nil, "", fmt.Errorf("resource manager is nil")
	}
	for _, candidate := range sfxPathCandidates(path) {
		data, err := manager.ReadFileExact(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("sfx not found: %s", path)
}

func normalizeSFXPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.ReplaceAll(path, "/", "\\")
	path = strings.TrimPrefix(path, ".\\")
	path = strings.TrimPrefix(path, "data\\")
	if path == "" {
		return ""
	}
	if filepath.Ext(path) == "" {
		path += ".wav"
	}
	return path
}

func sfxPathCandidates(path string) []string {
	normalized := normalizeSFXPath(path)
	if normalized == "" {
		return nil
	}
	slash := strings.ReplaceAll(normalized, "\\", "/")
	candidates := []string{normalized, slash}
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "wav\\") {
		candidates = append(candidates,
			"data\\"+normalized,
			"data/"+slash,
		)
	} else {
		candidates = append(candidates,
			"wav\\"+normalized,
			"wav/"+slash,
			"data\\wav\\"+normalized,
			"data/wav/"+slash,
		)
	}
	return uniqueStrings(candidates)
}
