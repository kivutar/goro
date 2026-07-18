package ui

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/kivutar/goro/db"
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
	skill := session.Skill{ID: db.SkillSMBash, Level: 9, Upgradable: true}
	if !window.canStageSkill(s, skill) {
		t.Fatal("expected level 9/10 skill to allow one staged level")
	}
	window.stageSkill(skill.ID)
	if window.canStageSkill(s, skill) {
		t.Fatal("level 9/10 skill should not allow staging past level 10")
	}
	if canIncreaseSkill(s, session.Skill{ID: db.SkillSMBash, Level: 10, Upgradable: true}) {
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

func TestSkillWindowShowsPendingUnlockedSkillBeforeConfirm(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSwordman},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMBash, Level: 4, MaxLevel: 10, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	if containsSkill(window.visibleSkills(Context{Session: s}), db.SkillSMMagnum) {
		t.Fatal("magnum break should not be visible before bash reaches level 5")
	}
	window.stageSkill(db.SkillSMBash)
	if !containsSkill(window.visibleSkills(Context{Session: s}), db.SkillSMMagnum) {
		t.Fatal("magnum break should be visible after staged bash level satisfies prerequisites")
	}
}

func TestSkillWindowShowsSuperNoviceThunderstorm(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSuperNovice},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillMGLightningbolt, Level: 3, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	if containsSkill(window.visibleSkills(Context{Session: s}), db.SkillMGThunderstorm) {
		t.Fatal("thunderstorm should not be visible before lightning bolt reaches level 4")
	}
	window.stageSkill(db.SkillMGLightningbolt)
	if !containsSkill(window.visibleSkills(Context{Session: s}), db.SkillMGThunderstorm) {
		t.Fatal("super novice should see thunderstorm after staged lightning bolt level satisfies prerequisites")
	}
}

func TestSkillWindowOrdersPendingUnlocksBySkillTree(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobSuperNovice},
		Skills: session.Skills{
			Points: 1,
			List: []session.Skill{
				{ID: db.SkillSMBash, Level: 1, Upgradable: true},
				{ID: db.SkillMGLightningbolt, Level: 3, Upgradable: true},
				{ID: db.SkillMGFirebolt, Level: 1, Upgradable: true},
				{ID: db.SkillALHeal, Level: 1, Upgradable: true},
			},
		},
	}
	window := &SkillWindow{}

	window.stageSkill(db.SkillMGLightningbolt)
	skills := window.visibleSkills(Context{Session: s})
	thunderstorm := skillIndex(skills, db.SkillMGThunderstorm)
	firebolt := skillIndex(skills, db.SkillMGFirebolt)
	heal := skillIndex(skills, db.SkillALHeal)
	if thunderstorm < 0 {
		t.Fatal("expected thunderstorm to be visible")
	}
	if !(firebolt < thunderstorm && thunderstorm < heal) {
		t.Fatalf("thunderstorm index = %d, firebolt = %d, heal = %d; expected skill-tree order", thunderstorm, firebolt, heal)
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

func TestSkillWindowSkillAtMouseUsesTableViewBody(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{
				{ID: 1},
				{ID: 2},
			},
		},
	}
	window := &SkillWindow{}
	window.EnsureWindow(skillWindowWidth, skillWindowHeight)
	window.x = 20
	window.y = 30
	ctx := Context{Session: s}

	if _, ok := window.skillAtMouse(ctx, window.x+8, window.y+ROWindowTitleHeight+skillHeaderH-1); ok {
		t.Fatal("header should not hit a skill row")
	}

	skill, ok := window.skillAtMouse(ctx, window.x+8, window.y+ROWindowTitleHeight+skillHeaderH+1)
	if !ok || skill.ID != 1 {
		t.Fatalf("skill at top row = %+v, %v; want id 1", skill, ok)
	}

	window.ensureScrollSignal().Set(skillRowH)
	skill, ok = window.skillAtMouse(ctx, window.x+8, window.y+ROWindowTitleHeight+skillHeaderH+1)
	if !ok || skill.ID != 2 {
		t.Fatalf("skill at scrolled top row = %+v, %v; want id 2", skill, ok)
	}
}

func TestSkillWindowTablePlusStagesSkill(t *testing.T) {
	skill := session.Skill{ID: db.SkillSMBash, Level: 1, MaxLevel: 10, Upgradable: true}
	s := &session.Session{Skills: session.Skills{Points: 1}}
	window := &SkillWindow{}
	bounds := skillTableLevelUpButtonBounds(0)

	consumed := window.handleSkillTableRowEvent(
		nil,
		Context{Session: s},
		nil,
		[]session.Skill{skill},
		0,
		event.NewMouseEvent(
			event.MousePress,
			event.ButtonLeft,
			event.ButtonStateLeft,
			geometry.Pt(bounds.Min.X+1, bounds.Min.Y+1),
			geometry.Pt(100, 120),
			0,
		),
	)

	if !consumed {
		t.Fatal("plus press was not consumed")
	}
	if got := window.pendingFor(skill.ID); got != 1 {
		t.Fatalf("pending skill levels = %d, want 1", got)
	}
	if !window.dirty {
		t.Fatal("plus press should mark the window dirty")
	}
}

func TestSkillDragReleaseOverShortcutStoresSkill(t *testing.T) {
	inputState := input.NewState()
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(Context{ScreenW: 800, ScreenH: 600}, 0)
	inputState.SetMousePosition(x+shortcutSlot/2, y+shortcutSlot/2)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

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

func (r *skillWindowTestRenderer) DrawInventoryItemIcon(*render.Frame, *res.Manager, session.InventoryItem, int, int) {
}

func (r *skillWindowTestRenderer) DrawSkillIcon(*render.Frame, *res.Manager, session.Skill, int, int, int) {
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

func containsSkill(skills []session.Skill, skillID uint16) bool {
	return skillIndex(skills, skillID) >= 0
}

func skillIndex(skills []session.Skill, skillID uint16) int {
	for i, skill := range skills {
		if skill.ID == skillID {
			return i
		}
	}
	return -1
}
