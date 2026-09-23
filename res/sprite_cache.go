package res

import (
	"container/list"
	"fmt"
	"sync"
	"unsafe"
)

const maxSpriteResourceCacheBytes = 32 << 20

type spriteResourceKey struct {
	name    string
	kind    byte
	archive *GRF // nil means normal manager lookup, including loose overrides.
}

type spriteResourceEntry struct {
	key   spriteResourceKey
	value any
	bytes int
}

// The manager retains recently used, immutable CPU resources across view and
// map changes. Eviction drops only the cache's reference; live views remain
// valid. The budget covers estimated retained allocations, not active views or
// rendered images. Resources are assumed unchanged for this manager's lifetime.
type spriteResourceCache struct {
	mu      sync.Mutex
	entries map[spriteResourceKey]*list.Element
	lru     list.List
	bytes   int
}

// LoadACT returns a shared, read-only animation resource. Candidates retain the
// sprite loader's ordering: the first readable file wins, even if parsing fails.
// Missing and invalid files are not cached.
func (m *Manager) LoadACT(candidates []string) (*ACT, string, error) {
	return loadSpriteResource(m, nil, candidates, 'a', ParseACT, actResourceBytes)
}

// LoadSPR returns a shared, read-only sprite, including its default palette.
// Palette overrides and rendered frames belong to the caller, not this cache.
func (m *Manager) LoadSPR(candidates []string) (*SPR, string, error) {
	return loadSpriteResource(m, nil, candidates, 's', ParseSPR, sprResourceBytes)
}

// LoadArchiveACT loads an exact entry from a particular archive. This keeps
// richer monster animation candidates separate from normal resource overrides.
// The returned resource is shared and read-only.
func (m *Manager) LoadArchiveACT(archive *GRF, name string) (*ACT, error) {
	if archive == nil {
		return nil, fmt.Errorf("no archive")
	}
	act, _, err := loadSpriteResource(m, archive, []string{name}, 'a', ParseACT, actResourceBytes)
	return act, err
}

// LoadArchiveSPR loads a shared, read-only sprite from an exact archive entry,
// without consulting loose files, aliases, or other archives.
func (m *Manager) LoadArchiveSPR(archive *GRF, name string) (*SPR, error) {
	if archive == nil {
		return nil, fmt.Errorf("no archive")
	}
	spr, _, err := loadSpriteResource(m, archive, []string{name}, 's', ParseSPR, sprResourceBytes)
	return spr, err
}

// LoadPAL returns a shared, read-only palette override.
func (m *Manager) LoadPAL(candidates []string) (*Palette, string, error) {
	return loadSpriteResource(m, nil, candidates, 'p', func(data []byte) (*Palette, error) {
		palette, err := ParsePAL(data)
		return &palette, err
	}, func(p *Palette) int { return int(unsafe.Sizeof(*p)) })
}

// LoadIMF returns shared, read-only humanoid layer ordering data.
func (m *Manager) LoadIMF(candidates []string) (*IMF, string, error) {
	return loadSpriteResource(m, nil, candidates, 'i', ParseIMF, imfResourceBytes)
}

func loadSpriteResource[T any](m *Manager, archive *GRF, candidates []string, kind byte, parse func([]byte) (*T, error), size func(*T) int) (*T, string, error) {
	if m == nil {
		return nil, "", fmt.Errorf("no resource manager")
	}
	c := &m.sprites
	c.mu.Lock()
	defer c.mu.Unlock()
	read := m.ReadFile
	if archive != nil {
		read = archive.ReadFile
	}
	for _, candidate := range candidates {
		// Preserve exact spelling: differently cased loose files may coexist.
		key := spriteResourceKey{name: candidate, kind: kind, archive: archive}
		if archive != nil {
			key.name = normalizeGRFName(candidate)
		}
		if element := c.entries[key]; element != nil {
			c.lru.MoveToFront(element)
			return element.Value.(spriteResourceEntry).value.(*T), candidate, nil
		}
		data, err := read(candidate)
		if err != nil {
			continue
		}
		value, err := parse(data)
		if err != nil {
			return nil, candidate, fmt.Errorf("parse %s: %w", candidate, err)
		}
		// Include an allowance for map/list bookkeeping and the retained key.
		c.put(key, value, size(value)+len(key.name)+160, maxSpriteResourceCacheBytes)
		return value, candidate, nil
	}
	return nil, "", fmt.Errorf("not found")
}

// Called with mu held. Oversized resources remain usable without displacing
// the entire cache. The explicit limit also allows small eviction fixtures.
func (c *spriteResourceCache) put(key spriteResourceKey, value any, bytes, limit int) {
	if bytes > limit {
		return
	}
	if c.entries == nil {
		c.entries = make(map[spriteResourceKey]*list.Element)
	}
	for c.bytes+bytes > limit {
		oldest := c.lru.Back()
		entry := oldest.Value.(spriteResourceEntry)
		delete(c.entries, entry.key)
		c.lru.Remove(oldest)
		c.bytes -= entry.bytes
	}
	c.entries[key] = c.lru.PushFront(spriteResourceEntry{key: key, value: value, bytes: bytes})
	c.bytes += bytes
}

func sprResourceBytes(spr *SPR) int {
	size := int(unsafe.Sizeof(*spr)) + cap(spr.Frames)*int(unsafe.Sizeof(SPRFrame{}))
	for _, frame := range spr.Frames {
		size += cap(frame.Data)
	}
	return size
}

func actResourceBytes(act *ACT) int {
	size := int(unsafe.Sizeof(*act)) + cap(act.Actions)*int(unsafe.Sizeof(ACTAction{})) + cap(act.Sounds)*int(unsafe.Sizeof(""))
	for _, action := range act.Actions {
		size += cap(action.Animations) * int(unsafe.Sizeof(ACTAnimation{}))
		for _, anim := range action.Animations {
			size += cap(anim.Layers)*int(unsafe.Sizeof(ACTLayer{})) + cap(anim.Pos)*int(unsafe.Sizeof(ACTPosition{}))
		}
	}
	for _, sound := range act.Sounds {
		size += len(sound)
	}
	return size
}

func imfResourceBytes(imf *IMF) int {
	size := int(unsafe.Sizeof(*imf)) + cap(imf.Layers)*int(unsafe.Sizeof(IMFLayer{}))
	for _, layer := range imf.Layers {
		size += cap(layer.Actions) * int(unsafe.Sizeof(IMFAction{}))
		for _, action := range layer.Actions {
			size += cap(action.Motions) * int(unsafe.Sizeof(IMFMotion{}))
		}
	}
	return size
}
