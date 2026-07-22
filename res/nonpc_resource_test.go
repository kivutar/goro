package res

import (
	"strings"
	"testing"
)

func TestNonPCSpriteResourceCandidatesNPC(t *testing.T) {
	got := NonPCSpriteResourceCandidates(47, "1_M_01", "act")
	want := []string{
		"data\\sprite\\NPC\\1_M_01.act",
		"data\\sprite\\NPC\\1_m_01.act",
		"data\\sprite\\npc\\1_M_01.act",
		"data\\sprite\\npc\\1_m_01.act",
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNonPCSpriteResourceCandidatesMonster(t *testing.T) {
	got := NonPCSpriteResourceCandidates(1002, "PORING", "spr")
	want := []string{
		"data\\sprite\\monster\\PORING.spr",
		"data\\sprite\\monster\\poring.spr",
		legacyMonsterSpriteRoot + "PORING.spr",
		legacyMonsterSpriteRoot + "poring.spr",
		"data\\sprite\\PORING.spr",
		"data\\sprite\\poring.spr",
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNonPCSpriteResourceCandidatesMercenaryCollapsesResourceSeparators(t *testing.T) {
	got := NonPCSpriteResourceCandidates(6037, `남\\검용병`, "act")
	want := []string{
		`data\sprite\인간족\몸통\남\검용병.act`,
		`data\sprite\mercenary\남\검용병.act`,
	}
	if len(got) < len(want) {
		t.Fatalf("candidate count = %d, want at least %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNonPCSpriteResourceRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	cases := []int{47, 1002, 1015, 6037}
	for _, job := range cases {
		name, ok := manager.NonPCResourceName(job)
		if !ok {
			t.Fatalf("job %d resource name missing", job)
		}
		if source, _, ok := manager.ReadFirst(NonPCSpriteResourceCandidates(job, name, "act")); !ok {
			t.Fatalf("job %d act not found for %q", job, name)
		} else {
			t.Logf("job %d act=%s", job, source)
		}
		if source, _, ok := manager.ReadFirst(NonPCSpriteResourceCandidates(job, name, "spr")); !ok {
			t.Fatalf("job %d spr not found for %q", job, name)
		} else {
			t.Logf("job %d spr=%s", job, source)
		}
	}
}

func TestZombieSpriteResourceRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	name, ok := manager.NonPCResourceName(1015)
	if !ok {
		t.Fatal("zombie resource name missing")
	}
	if name != "Zombie" && name != "zombie" {
		t.Fatalf("zombie resource name = %q", name)
	}
	source, _, ok := manager.ReadFirst(NonPCSpriteResourceCandidates(1015, name, "spr"))
	if !ok {
		t.Fatalf("zombie spr not found for %q", name)
	}
	if strings.Contains(strings.ToLower(source), "orc_zombie") {
		t.Fatalf("zombie sprite resolved to orc zombie: %s", source)
	}
	t.Logf("zombie spr=%s", source)
}
