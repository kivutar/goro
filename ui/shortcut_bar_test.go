package ui

import (
	"testing"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestShortcutSlotHotkeyRoundTrip(t *testing.T) {
	item := shortcutSlotState{kind: shortcutItem, itemID: 601, identified: true}
	itemHotkey := item.hotkey()
	if itemHotkey.Type != network.HotkeyTypeItem || itemHotkey.ID != 601 || itemHotkey.Level != 0 {
		t.Fatalf("item hotkey = %+v", itemHotkey)
	}
	if got := shortcutSlotFromHotkey(session.HotkeySlot{Type: itemHotkey.Type, ID: itemHotkey.ID, Level: itemHotkey.Level}); got.kind != shortcutItem || got.itemID != 601 || !got.identified {
		t.Fatalf("item slot = %+v", got)
	}

	skill := shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	skillHotkey := skill.hotkey()
	if skillHotkey.Type != network.HotkeyTypeSkill || skillHotkey.ID != 6 || skillHotkey.Level != 2 {
		t.Fatalf("skill hotkey = %+v", skillHotkey)
	}
	if got := shortcutSlotFromHotkey(session.HotkeySlot{Type: skillHotkey.Type, ID: skillHotkey.ID, Level: skillHotkey.Level}); got != skill {
		t.Fatalf("skill slot = %+v, want %+v", got, skill)
	}
}

func TestShortcutBarSyncsFromSessionHotkeys(t *testing.T) {
	ctx := Context{Session: &session.Session{Hotkeys: session.Hotkeys{
		Loaded:  true,
		Version: 3,
		Slots: []session.HotkeySlot{
			{Type: network.HotkeyTypeSkill, ID: 6, Level: 2},
			{Type: network.HotkeyTypeItem, ID: 501},
		},
	}}}

	bar := &ShortcutBar{}
	bar.SyncFromSession(ctx)
	if bar.slots[0] != (shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}) {
		t.Fatalf("slot 1 = %+v", bar.slots[0])
	}
	if bar.slots[1].kind != shortcutItem || bar.slots[1].itemID != 501 {
		t.Fatalf("slot 2 = %+v", bar.slots[1])
	}
	if bar.hotkeyVersion != 3 {
		t.Fatalf("hotkey version = %d, want 3", bar.hotkeyVersion)
	}
}

func TestShortcutDropMarksShortcutOverlayDirty(t *testing.T) {
	app := &shortcutInvalidatingApp{}
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIApp:     app,
		UIManager: &shortcutInvalidatingManager{},
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.Publish(ctx, nil, nil)
	if bar.root == nil {
		t.Fatal("shortcut bar root was not published")
	}
	x, y := bar.slotBounds(ctx, 0)

	if !bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2}, x+1, y+1) {
		t.Fatal("skill drop was not accepted")
	}
	if app.invalidates != 1 {
		t.Fatalf("shortcut drop invalidates = %d, want 1", app.invalidates)
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != 6 || got.skillLevel != 2 {
		t.Fatalf("slot = %+v", got)
	}
}

type shortcutInvalidatingManager struct {
	overlays []widget.Widget
}

func (m *shortcutInvalidatingManager) AddOverlay(root widget.Widget) {
	m.overlays = append(m.overlays, root)
}

func (m *shortcutInvalidatingManager) RemoveOverlay(root widget.Widget) {
	for i, overlay := range m.overlays {
		if overlay == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			return
		}
	}
}

func (m *shortcutInvalidatingManager) Clear() {
	m.overlays = nil
}

type shortcutInvalidatingApp struct {
	invalidates int
}

func (a *shortcutInvalidatingApp) SetUIRoot(widget.Widget) {}

func (a *shortcutInvalidatingApp) Frame() {}

func (a *shortcutInvalidatingApp) Invalidate() {
	a.invalidates++
}

func (a *shortcutInvalidatingApp) Cursor() widget.CursorType {
	return widget.CursorDefault
}

func (a *shortcutInvalidatingApp) HoveredWidget() widget.Widget {
	return nil
}

func TestShortcutPublishDoesNotKeepOverlayDirty(t *testing.T) {
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.Publish(ctx, nil, nil)
	if clear, ok := bar.root.(interface {
		ClearRedraw()
		ClearSceneDirty()
	}); ok {
		clear.ClearRedraw()
		clear.ClearSceneDirty()
	} else {
		t.Fatal("shortcut overlay cannot simulate clean boundary")
	}

	bar.Publish(ctx, nil, nil)
	if redraw, ok := bar.root.(interface{ NeedsRedraw() bool }); !ok || redraw.NeedsRedraw() {
		t.Fatal("shortcut publish dirtied an unchanged overlay")
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

func TestShortcutBarKeepsDepletedItemShortcut(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[2] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 501}
	bar.slots[3] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 602}
	bar.slots[4] = shortcutSlotState{kind: shortcutItem, itemIndex: 14, itemID: 501}

	if bar.ClearDepletedItem(Context{}, 12, 501) {
		t.Fatal("depleted shortcut should not be cleared locally")
	}
	if bar.slots[2].kind != shortcutItem || bar.slots[3].kind != shortcutItem || bar.slots[4].kind != shortcutItem {
		t.Fatalf("shortcut slots changed: slot3=%+v slot4=%+v slot5=%+v", bar.slots[2], bar.slots[3], bar.slots[4])
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

func TestSkillForShortcutResolvesHomunculusSkills(t *testing.T) {
	s := &session.Session{
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillHvanCaprice, Level: 4, Type: 1, Name: "Caprice"}},
			},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillHvanCaprice, skillLevel: 2})
	if !ok {
		t.Fatal("homunculus shortcut skill not found")
	}
	if skill.ID != db.SkillHvanCaprice || skill.Level != 2 || skill.Name != "Caprice" {
		t.Fatalf("homunculus shortcut skill = %+v", skill)
	}
}

func TestSkillForShortcutPrefersMercenaryThenHomunculusBeforePlayer(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: db.SkillMsBash, Level: 10, Type: 1, Name: "Player"}},
		},
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillMsBash, Level: 4, Type: 1, Name: "Homunculus"}},
			},
		},
		Mercenary: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillMsBash, Level: 2, Type: 1, Name: "Mercenary"}},
			},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillMsBash})
	if !ok {
		t.Fatal("shortcut skill not found")
	}
	if skill.Name != "Mercenary" || skill.Level != 2 {
		t.Fatalf("shortcut skill = %+v, want mercenary first", skill)
	}

	s.Mercenary.Active = false
	skill, ok = skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillMsBash})
	if !ok {
		t.Fatal("shortcut skill not found after mercenary deactivated")
	}
	if skill.Name != "Homunculus" || skill.Level != 4 {
		t.Fatalf("shortcut skill = %+v, want homunculus second", skill)
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
