//go:build nofakecgo

package audio

const maxSFXCacheBytes = 32 << 20

type decodedSFX struct {
	pcm    []byte
	source string
}

type sfxCacheKey struct {
	path       string
	sampleRate int
}

type sfxCacheEntry struct {
	sound    decodedSFX
	lastUsed uint64
}

// sfxCache is owned by the game thread. Players get independent readers of
// immutable PCM, so overlapping playback and cache eviction do not interfere.
type sfxCache struct {
	entries map[sfxCacheKey]sfxCacheEntry
	bytes   int
	clock   uint64
}

func (c *sfxCache) get(key sfxCacheKey) (decodedSFX, bool) {
	entry, ok := c.entries[key]
	if !ok {
		return decodedSFX{}, false
	}
	c.clock++
	entry.lastUsed = c.clock
	c.entries[key] = entry
	return entry.sound, true
}

func (c *sfxCache) put(key sfxCacheKey, sound decodedSFX) {
	if len(sound.pcm) == 0 || len(sound.pcm) > maxSFXCacheBytes {
		return
	}
	if c.entries == nil {
		c.entries = make(map[sfxCacheKey]sfxCacheEntry)
	}
	if previous, ok := c.entries[key]; ok {
		c.bytes -= len(previous.sound.pcm)
		delete(c.entries, key)
	}
	for c.bytes+len(sound.pcm) > maxSFXCacheBytes {
		var oldest sfxCacheKey
		lastUsed := ^uint64(0)
		for candidate, entry := range c.entries {
			if entry.lastUsed < lastUsed {
				oldest, lastUsed = candidate, entry.lastUsed
			}
		}
		c.bytes -= len(c.entries[oldest].sound.pcm)
		delete(c.entries, oldest)
	}
	c.clock++
	c.entries[key] = sfxCacheEntry{sound: sound, lastUsed: c.clock}
	c.bytes += len(sound.pcm)
}
