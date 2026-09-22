//go:build nofakecgo

package audio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/res"
)

func TestLoadSFXCachesDecodedAndResampledAudio(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bird.wav")
	wav := testWAV(1, 8, 22050, []byte{0, 128, 255, 128})
	if err := os.WriteFile(path, wav, 0600); err != nil {
		t.Fatal(err)
	}
	b := NewBGM(&res.Manager{Root: root}, false, 0, 1, false)
	if err := b.PreloadSFX("bird.wav"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	first, err := b.loadSFX("bird.wav", 44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.pcm) != 32 {
		t.Fatalf("resampled PCM length = %d, want 32", len(first.pcm))
	}
	repeated, err := b.loadSFX("BIRD.WAV", 44100)
	if err != nil {
		t.Fatalf("cached playback reread the file: %v", err)
	}
	if first.source != repeated.source || &first.pcm[0] != &repeated.pcm[0] {
		t.Fatal("repeated playback did not reuse decoded PCM")
	}
	if _, err := b.loadSFX("bird.wav", 22050); err == nil {
		t.Fatal("different output rate reused incompatible PCM")
	}
	if b.context != nil {
		t.Fatal("loading a sound initialized the playback device")
	}
}

func TestPreloadSFXSkipsDisabledOrMutedAudio(t *testing.T) {
	for _, b := range []*BGM{nil, NewBGM(nil, false, 0, 1, true), NewBGM(nil, false, 0, 0, false)} {
		if err := b.PreloadSFX("missing.wav"); err != nil {
			t.Fatalf("disabled or muted audio attempted to load a sound: %v", err)
		}
	}
}

func TestLoadSFXRetriesDecodeErrors(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bird.wav")
	if err := os.WriteFile(path, []byte("invalid WAV"), 0600); err != nil {
		t.Fatal(err)
	}
	b := NewBGM(&res.Manager{Root: root}, false, 0, 1, false)
	if _, err := b.loadSFX("bird.wav", 44100); err == nil {
		t.Fatal("malformed WAV succeeded")
	}
	if err := os.WriteFile(path, testWAV(1, 8, 44100, []byte{128, 255}), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.loadSFX("bird.wav", 44100); err != nil {
		t.Fatalf("decode failure was cached: %v", err)
	}
}

func TestSFXCacheEvictsLeastRecentlyUsedWithoutChangingPlayback(t *testing.T) {
	var cache sfxCache
	pcm := make([]byte, maxSFXCacheBytes/2)
	pcm[0], pcm[1] = 10, 20
	a := sfxCacheKey{path: "a", sampleRate: 44100}
	b := sfxCacheKey{path: "b", sampleRate: 44100}
	c := sfxCacheKey{path: "c", sampleRate: 44100}
	cache.put(a, decodedSFX{pcm: pcm})
	cache.put(b, decodedSFX{pcm: pcm})
	sound, _ := cache.get(b)
	first, second := bytes.NewReader(sound.pcm), bytes.NewReader(sound.pcm)
	if value, err := first.ReadByte(); err != nil || value != 10 {
		t.Fatalf("first player initial sample = %d, %v", value, err)
	}
	cache.get(a) // a is most recently used, so b must be evicted.
	cache.put(c, decodedSFX{pcm: pcm})
	if _, ok := cache.get(b); ok {
		t.Fatal("least recently used sound was not evicted")
	}
	if _, ok := cache.get(a); !ok || cache.bytes > maxSFXCacheBytes {
		t.Fatal("cache exceeded its budget or evicted the recently used sound")
	}
	if value, err := first.ReadByte(); err != nil || value != 20 {
		t.Fatalf("eviction interrupted first player: %d, %v", value, err)
	}
	if value, err := second.ReadByte(); err != nil || value != 10 {
		t.Fatalf("overlapping player lost its independent position: %d, %v", value, err)
	}
	cache.put(a, decodedSFX{pcm: []byte{30}})
	if cache.bytes != len(pcm)+1 {
		t.Fatalf("replacement accounting = %d, want %d", cache.bytes, len(pcm)+1)
	}
}

func TestSFXCacheDoesNotRetainOversizedSounds(t *testing.T) {
	var cache sfxCache
	key := sfxCacheKey{path: "large", sampleRate: 44100}
	cache.put(key, decodedSFX{pcm: make([]byte, maxSFXCacheBytes+1)})
	if _, ok := cache.get(key); ok || cache.bytes != 0 {
		t.Fatal("cache retained a sound larger than its memory budget")
	}
}
