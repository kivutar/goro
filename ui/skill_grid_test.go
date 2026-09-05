package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestSkillGridKeepsFixedPositionsAndMinimumRows(t *testing.T) {
	grid := newSkillGridWidget(skillGridConfig{entries: []skillGridEntry{
		{position: 14, skill: session.Skill{ID: db.SkillNVTrickdead}},
		{position: 0, skill: session.Skill{ID: db.SkillNVBasic}},
	}})
	grid.Layout(widget.NewContext(), geometry.Constraints{
		MinWidth:  skillGridViewW,
		MaxWidth:  skillGridViewW,
		MinHeight: 0,
		MaxHeight: geometry.Infinity,
	})

	if got := grid.totalRows(); got != skillGridMinRows {
		t.Fatalf("grid rows = %d, want minimum %d", got, skillGridMinRows)
	}
	if entry, ok := grid.entryAtPosition(14); !ok || entry.skill.ID != db.SkillNVTrickdead {
		t.Fatalf("position 14 = %+v, %v; want Trick Dead", entry, ok)
	}
	if got := grid.cellBounds(14).Min.Y; got != 2*skillGridCellH {
		t.Fatalf("position 14 y = %.1f, want %d", got, 2*skillGridCellH)
	}
}

func TestSkillGridUpgradableCellStagesInsteadOfPressingSkill(t *testing.T) {
	skill := session.Skill{ID: db.SkillSMBash, Level: 1, Upgradable: true}
	staged := 0
	pressed := 0
	grid := newSkillGridWidget(skillGridConfig{
		entries: []skillGridEntry{{position: 0, skill: skill, display: skill, canStage: true}},
		onStage: func(session.Skill) { staged++ },
		onPress: func(session.Skill, int, int) { pressed++ },
	})
	grid.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(skillGridViewW, skillGridViewH)))
	slot := skillGridSlotBounds(grid.cellBounds(0))
	point := geometry.Pt(slot.Min.X+1, slot.Min.Y+1)
	grid.Event(widget.NewContext(), event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		event.ButtonStateLeft,
		point,
		point,
		event.ModNone,
	))

	if staged != 1 || pressed != 0 {
		t.Fatalf("stage click = staged %d pressed %d, want 1 and 0", staged, pressed)
	}
}

func TestSkillGridUpgradableCellUsesBlueBackground(t *testing.T) {
	skill := session.Skill{ID: db.SkillSMBash, Level: 1, Upgradable: true}
	grid := newSkillGridWidget(skillGridConfig{
		entries: []skillGridEntry{{position: 0, skill: skill, display: skill, canStage: true}},
	})
	grid.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(skillGridViewW, skillGridViewH)))
	canvas := &uitest.MockCanvas{}
	grid.Draw(widget.NewContext(), canvas)

	if len(canvas.RoundRects) == 0 {
		t.Fatal("grid drew no skill slot backgrounds")
	}
	if got := canvas.RoundRects[0].Color; got != rotheme.Default.Colors.ButtonHover {
		t.Fatalf("upgradable slot color = %+v, want %+v", got, rotheme.Default.Colors.ButtonHover)
	}
}

func TestSkillGridFallbackPositionsAreDense(t *testing.T) {
	skills := []session.Skill{
		{ID: db.SkillNVBasic},
		{ID: db.SkillNVFirstaid},
		{ID: db.SkillNVTrickdead},
	}
	positions := skillGridPositions(nil, db.JobNovice, skillTabFirst, skills)
	for i, skill := range skills {
		if got := positions[skill.ID]; got != i {
			t.Fatalf("skill %d position = %d, want %d", skill.ID, got, i)
		}
	}
}

func TestSkillWindowGridIncludesUnavailableTreeSkills(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Job: db.JobAlchemist},
		Skills: session.Skills{List: []session.Skill{
			{ID: db.SkillAMLearningpotion, Level: 1},
		}},
	}
	window := &SkillWindow{tab: skillTabSecond, gridMode: true}
	ctx := Context{Session: s}
	window.ensureSkillView(ctx)
	skills := window.gridSkills(ctx)

	if !containsSkill(skills, db.SkillAMLearningpotion) || !containsSkill(skills, db.SkillAMPharmacy) {
		t.Fatalf("grid skills omit the complete Alchemist tree: %+v", skills)
	}
	pharmacy := skills[skillIndex(skills, db.SkillAMPharmacy)]
	if pharmacy.Level != 0 || pharmacy.Upgradable {
		t.Fatalf("unavailable Pharmacy = %+v, want visible but locked", pharmacy)
	}
}

func TestSkillWindowTogglesBetweenListAndGridSizes(t *testing.T) {
	ctx := Context{
		Session: &session.Session{Selected: session.Character{Job: db.JobNovice}},
		ScreenW: 800,
		ScreenH: 600,
	}
	window := &SkillWindow{}
	window.EnsureWindow(skillWindowWidth, skillWindowHeight)
	window.OpenAt(40, 50, window.widgetTree(ctx, nil))

	window.toggleGridMode(ctx, nil, nil)
	if !window.gridMode || window.width != skillGridWindowWidth || window.height != skillGridWindowHeight || window.grid == nil {
		t.Fatalf("expanded window = mode %t size %dx%d grid %p", window.gridMode, window.width, window.height, window.grid)
	}
	window.content.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(skillGridWindowWidth, skillGridWindowHeight)))
	window.content.Draw(widget.NewContext(), &uitest.MockCanvas{})

	window.toggleGridMode(ctx, nil, nil)
	if window.gridMode || window.width != skillWindowWidth || window.height != skillWindowHeight || window.table == nil {
		t.Fatalf("compact window = mode %t size %dx%d table %p", window.gridMode, window.width, window.height, window.table)
	}
}
