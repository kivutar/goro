package res

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSpriteCacheFixture(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSpriteResourceCacheSharesSuccessfulLoads(t *testing.T) {
	root := t.TempDir()
	data, _ := actCountFixture(2, 5)
	writeSpriteCacheFixture(t, root, "shared.act", data)
	m := &Manager{Root: root}
	first, _, err := m.LoadACT([]string{"shared.act"})
	if err != nil {
		t.Fatal(err)
	}
	// A cache hit must avoid even reading the file. Different candidate lists
	// can still share the same winning resource.
	writeSpriteCacheFixture(t, root, "shared.act", []byte("invalid"))
	second, source, err := m.LoadACT([]string{"missing.act", "shared.act"})
	if err != nil || second != first || source != "shared.act" {
		t.Fatalf("resource not shared: source=%q err=%v", source, err)
	}
	other := &Manager{Root: root}
	if _, _, err := other.LoadACT([]string{"shared.act"}); err == nil {
		t.Fatal("separate manager reused the first manager's cache")
	}
	if _, _, err := m.LoadSPR([]string{"shared.act"}); err == nil {
		t.Fatal("cache ignored the requested resource type")
	}
}

func TestSpriteResourceCachePreservesCandidatePriorityAndRetriesFailures(t *testing.T) {
	root := t.TempDir()
	m := &Manager{Root: root}
	data, _ := actCountFixture(2, 5)
	writeSpriteCacheFixture(t, root, "fallback.act", data)
	candidates := []string{"preferred.act", "fallback.act"}
	fallback, source, err := m.LoadACT(candidates)
	if err != nil || source != "fallback.act" {
		t.Fatalf("missing fallback: source=%q err=%v", source, err)
	}
	writeSpriteCacheFixture(t, root, "preferred.act", []byte("invalid"))
	if _, source, err := m.LoadACT(candidates); err == nil || source != "preferred.act" {
		t.Fatalf("cached fallback hid a malformed preferred resource: source=%q err=%v", source, err)
	}
	writeSpriteCacheFixture(t, root, "preferred.act", data)
	preferred, source, err := m.LoadACT(candidates)
	if err != nil || source != "preferred.act" || preferred == fallback {
		t.Fatalf("failed load was cached or candidate priority changed: source=%q err=%v", source, err)
	}
}

func TestSpriteResourceCacheKeepsLooseOverrideAndAliasLookup(t *testing.T) {
	root := t.TempDir()
	archiveData, _ := actCountFixture(1, 1)
	if err := writeTestGRF(filepath.Join(root, "test.grf"), "shared.act", archiveData); err != nil {
		t.Fatal(err)
	}
	archive, err := OpenGRF(filepath.Join(root, "test.grf"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = archive.Close() })
	looseData, _ := actCountFixture(2, 5)
	writeSpriteCacheFixture(t, root, "shared.act", looseData)
	writeSpriteCacheFixture(t, root, "resnametable.txt", []byte("alias.act#shared.act#\n"))
	m := &Manager{Root: root, Archives: []*GRF{archive}}
	for _, name := range []string{"shared.act", "alias.act"} {
		act, source, err := m.LoadACT([]string{name})
		if err != nil || act.VersionMajor != 2 || source != name {
			t.Fatalf("lookup changed for %q: source=%q err=%v", name, source, err)
		}
		if again, _, err := m.LoadACT([]string{name}); err != nil || again != act {
			t.Fatalf("lookup not cached for %q: %v", name, err)
		}
	}
	act, err := m.LoadArchiveACT(archive, "shared.act")
	if err != nil || act.VersionMajor != 1 {
		t.Fatalf("archive lookup reused a loose override: %v", err)
	}
	if again, err := m.LoadArchiveACT(archive, "SHARED.ACT"); err != nil || again != act {
		t.Fatalf("archive resource not shared: %v", err)
	}
	// A second archive may contain the same path with different content.
	if err := writeTestGRF(filepath.Join(root, "second.grf"), "shared.act", looseData); err != nil {
		t.Fatal(err)
	}
	second, err := OpenGRF(filepath.Join(root, "second.grf"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	if other, err := m.LoadArchiveACT(second, "shared.act"); err != nil || other == act || other.VersionMajor != 2 {
		t.Fatalf("archive caches were mixed: %v", err)
	}
}

func TestSpriteResourceCacheEvictionKeepsLiveResources(t *testing.T) {
	root := t.TempDir()
	data, _ := actCountFixture(2, 5)
	for _, name := range []string{"a.act", "b.act", "c.act"} {
		writeSpriteCacheFixture(t, root, name, data)
	}
	m := &Manager{Root: root}
	a, _, _ := m.LoadACT([]string{"a.act"})
	b, _, _ := m.LoadACT([]string{"b.act"})
	budget := m.sprites.bytes
	// Touch A through the public loader, so B must be evicted first.
	if again, _, err := m.LoadACT([]string{"a.act"}); err != nil || again != a {
		t.Fatal("A not cached")
	}
	c, err := ParseACT(data)
	if err != nil {
		t.Fatal(err)
	}
	key := spriteResourceKey{name: "c.act", kind: 'a'}
	m.sprites.put(key, c, budget/2, budget)
	if len(m.sprites.entries) != 2 || m.sprites.bytes != budget || m.sprites.entries[spriteResourceKey{name: "b.act", kind: 'a'}] != nil {
		t.Fatal("did not evict the least recently used resource within the budget")
	}
	if len(b.Actions) != 1 || b.Actions[0].Animations[0].Layers[0].X != 12 {
		t.Fatal("eviction invalidated a live resource")
	}
	if again, _, err := m.LoadACT([]string{"b.act"}); err != nil || again == b {
		t.Fatal("evicted resource was not reloaded")
	}
	before := m.sprites.bytes
	m.sprites.put(spriteResourceKey{name: "oversized", kind: 'a'}, c, before+1, before)
	if m.sprites.bytes != before || m.sprites.entries[key] == nil {
		t.Fatal("oversized resource displaced cached resources")
	}
}
