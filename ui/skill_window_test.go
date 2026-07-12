package ui

import (
	"image"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestCanIncreaseSkillRequiresPointsAndFlag(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 1}}
	if !canIncreaseSkill(s, session.Skill{ID: 1, Upgradable: true}) {
		t.Fatal("expected skill to be increasable")
	}
	if canIncreaseSkill(s, session.Skill{ID: 1}) {
		t.Fatal("skill without upgradable flag should not increase")
	}
	s.Skills.Points = 0
	if canIncreaseSkill(s, session.Skill{ID: 1, Upgradable: true}) {
		t.Fatal("skill without points should not increase")
	}
}

func TestSkillWindowCanStageSkillHonorsMaxLevel(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 3}}
	window := &SkillWindow{}
	skill := session.Skill{ID: 5, Level: 9, MaxLevel: 10, Upgradable: true}
	if !window.canStageSkill(s, skill) {
		t.Fatal("expected level 9/10 skill to allow one staged level")
	}
	window.stageSkill(skill.ID)
	if window.canStageSkill(s, skill) {
		t.Fatal("level 9/10 skill should not allow staging past level 10")
	}
	if canIncreaseSkill(s, session.Skill{ID: 5, Level: 10, MaxLevel: 10, Upgradable: true}) {
		t.Fatal("max-level skill should not increase")
	}
}

func TestSkillWindowCanStageSkillWithoutKnownMaxAllowsAvailablePoints(t *testing.T) {
	s := &session.Session{Skills: session.Skills{Points: 3}}
	window := &SkillWindow{}
	skill := session.Skill{ID: 999, Level: 1, Upgradable: true}
	for i := 0; i < s.Skills.Points; i++ {
		if !window.canStageSkill(s, skill) {
			t.Fatalf("expected unknown max skill to allow staged level %d", i+1)
		}
		window.stageSkill(skill.ID)
	}
	if window.canStageSkill(s, skill) {
		t.Fatal("unknown max skill should not allow staging past available points")
	}
}

func TestSkillWindowDoubleClickUsesSharedSkillController(t *testing.T) {
	ctx := Context{ScreenW: 800, ScreenH: 600}
	mode := &skillWindowTestRenderer{}
	window := &SkillWindow{}
	skill := session.Skill{ID: 6, Type: 1, Level: 2, Range: 9}

	window.pressSkill(ctx, mode, skill, 20, 30)
	if mode.used.ID != 0 {
		t.Fatalf("skill used after first click = %+v, want none", mode.used)
	}

	window.pressSkill(ctx, mode, skill, 20, 30)
	if mode.used.ID != 6 || mode.used.Level != 2 {
		t.Fatalf("used skill = %+v, want provoke level 2", mode.used)
	}
}

func TestSkillDragReleaseOverShortcutStoresSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{
		loaded: true,
		path:   filepath.Join(t.TempDir(), "shortcuts.json"),
	}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(render.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(render.MouseButtonLeft, false)

	window := &SkillWindow{
		dragSkill:  session.Skill{ID: 46, Level: 10},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	if !window.UpdateDrag(Context{Input: inputState, ScreenW: 800, ScreenH: 600}, bar) {
		t.Fatal("skill drag release was not consumed")
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != 46 || got.skillLevel != 10 {
		t.Fatalf("shortcut slot = %+v, want double strafe level 10", got)
	}
}

func TestSkillTooltipUsesSharedOverlayState(t *testing.T) {
	window := &SkillWindow{}
	ctx := Context{ScreenW: 800, ScreenH: 600}
	skill := session.Skill{ID: 6, Name: "Provoke", Level: 2, Range: 9}

	window.showTooltip(ctx, skill, 100, 120)
	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	text := window.tooltip.Text()
	if !strings.Contains(text, "Provoke") || !strings.Contains(text, "Lv 2") || !strings.Contains(text, "Range: 9") {
		t.Fatalf("tooltip text = %q", text)
	}

	window.hideTooltip()
	if window.tooltip.Open() {
		t.Fatal("tooltip should be closed")
	}
}

type skillWindowTestRenderer struct {
	used session.Skill
}

func (r *skillWindowTestRenderer) DrawInventoryItemIcon(*render.Image, *res.Manager, session.InventoryItem, int, int) {
}

func (r *skillWindowTestRenderer) DrawSkillIcon(*render.Image, *res.Manager, session.Skill, int, int, int) {
}

func (r *skillWindowTestRenderer) SkillIconImage(*res.Manager, session.Skill, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) ItemInfoIllustrationImage(*res.Manager, session.InventoryItem, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) EquipmentPreviewImage(Context, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) EquipmentPreviewImageForCharacter(Context, session.Character, byte, int, int) image.Image {
	return nil
}

func (r *skillWindowTestRenderer) UseShortcutSkill(_ Context, skill session.Skill) error {
	r.used = skill
	return nil
}

func (r *skillWindowTestRenderer) AddTeleportEffect(Context) {}
