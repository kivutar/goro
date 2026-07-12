package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kivutar/goro/session"
)

func TestShortcutSlotPersistRoundTrip(t *testing.T) {
	item := shortcutSlotState{
		kind:       shortcutItem,
		itemIndex:  12,
		itemID:     601,
		identified: true,
	}
	if got := shortcutSlotFromPersist(item.persist()); got != item {
		t.Fatalf("item slot = %+v, want %+v", got, item)
	}

	skill := shortcutSlotState{
		kind:       shortcutSkill,
		skillID:    6,
		skillLevel: 2,
	}
	if got := shortcutSlotFromPersist(skill.persist()); got != skill {
		t.Fatalf("skill slot = %+v, want %+v", got, skill)
	}
}

func TestShortcutStatePathUsesSelectedCharacter(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	s := &session.Session{
		AccountID: 2000000,
		CharID:    150001,
		Selected:  session.Character{ID: 150001, Name: "Osmotar"},
	}

	path, legacy, err := shortcutStatePath(s)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(config, "goro", "shortcuts", "char-150001.json")
	if path != want {
		t.Fatalf("shortcut path = %q, want %q", path, want)
	}
	if legacy != filepath.Join(config, "goro", "shortcuts.json") {
		t.Fatalf("legacy path = %q", legacy)
	}
}

func TestShortcutStatePathFallsBackToSanitizedCharacterName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &session.Session{Selected: session.Character{Name: "A/B C"}}

	path, _, err := shortcutStatePath(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, filepath.Join("goro", "shortcuts", "name-A_B_C.json")) {
		t.Fatalf("shortcut path = %q", path)
	}
}

func TestShortcutLoadMigratesLegacyFileToCharacterPath(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	legacy := filepath.Join(config, "goro", "shortcuts.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(`{"version":1,"slots":[{"kind":"skill","skill_id":6,"skill_level":2}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := Context{Session: &session.Session{Selected: session.Character{ID: 150001}}}

	bar := &ShortcutBar{}
	bar.Load(ctx)
	if bar.slots[0].kind != shortcutSkill || bar.slots[0].skillID != 6 || bar.slots[0].skillLevel != 2 {
		t.Fatalf("migrated slot = %+v", bar.slots[0])
	}
	if !strings.HasSuffix(bar.path, filepath.Join("goro", "shortcuts", "char-150001.json")) {
		t.Fatalf("bar path = %q", bar.path)
	}

	bar.save(ctx)
	if _, err := os.Stat(bar.path); err != nil {
		t.Fatalf("character shortcut file not written: %v", err)
	}
}

func TestInventoryItemForShortcutFallsBackToItemID(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 8, ItemID: 501, Amount: 2},
				{Index: 14, ItemID: 601, Amount: 3},
			},
		},
	}

	item, ok := inventoryItemForShortcut(s, 99, 601)
	if !ok {
		t.Fatal("item not found")
	}
	if item.Index != 14 || item.ItemID != 601 {
		t.Fatalf("item = %+v", item)
	}
}

func TestInventoryItemForShortcutRejectsReusedIndexWithDifferentItem(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 12, ItemID: 602, Amount: 1},
			},
		},
	}

	item, ok := inventoryItemForShortcut(s, 12, 501)
	if ok {
		t.Fatalf("shortcut resolved reused index to wrong item: %+v", item)
	}
}

func TestShortcutBarClearsDepletedItem(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[2] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 501}
	bar.slots[3] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 602}
	bar.slots[4] = shortcutSlotState{kind: shortcutItem, itemIndex: 14, itemID: 501}

	if !bar.clearDepletedItemSlots(12, 501) {
		t.Fatal("depleted shortcut was not cleared")
	}
	if bar.slots[2].kind != shortcutEmpty {
		t.Fatalf("slot 3 kind = %d, want empty", bar.slots[2].kind)
	}
	if bar.slots[3].kind != shortcutItem || bar.slots[4].kind != shortcutItem {
		t.Fatalf("unrelated shortcuts were cleared: slot4=%+v slot5=%+v", bar.slots[3], bar.slots[4])
	}
}

func TestSkillForShortcutUsesSelectedLevel(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: 6, Level: 8, Range: 9}},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2})
	if !ok {
		t.Fatal("skill not found")
	}
	if skill.Level != 2 {
		t.Fatalf("shortcut skill level = %d, want selected level 2", skill.Level)
	}
}

func TestSkillForShortcutFallsBackAndClampsLevel(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: 6, Level: 4, Range: 9}},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6})
	if !ok {
		t.Fatal("legacy skill shortcut not found")
	}
	if skill.Level != 4 {
		t.Fatalf("legacy shortcut level = %d, want learned level 4", skill.Level)
	}

	skill, ok = skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 9})
	if !ok {
		t.Fatal("clamped skill shortcut not found")
	}
	if skill.Level != 4 {
		t.Fatalf("clamped shortcut level = %d, want learned level 4", skill.Level)
	}
}

func TestShortcutSkillTooltipUsesHotkeyAndName(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[1] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	bar.ctx = Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session: &session.Session{
			Skills: session.Skills{
				List: []session.Skill{{ID: 6, Level: 8, Name: "Provoke"}},
			},
		},
	}

	bar.showTooltip(1)
	if !bar.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if got := bar.tooltipText(1); got != "[ F2 ] Provoke" {
		t.Fatalf("tooltip text = %q", got)
	}
	if got := bar.tooltip.Text(); got != "[ F2 ] Provoke" {
		t.Fatalf("published tooltip text = %q", got)
	}
}

func TestShortcutTooltipUnpublishesForEmptySlot(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[0] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	bar.ctx = Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session: &session.Session{
			Skills: session.Skills{
				List: []session.Skill{{ID: 6, Level: 2, Name: "Provoke"}},
			},
		},
	}

	bar.showTooltip(0)
	if !bar.tooltip.Open() {
		t.Fatal("tooltip did not open")
	}
	bar.showTooltip(1)
	if bar.tooltip.Open() {
		t.Fatal("tooltip is still open after hovering empty slot")
	}
}
