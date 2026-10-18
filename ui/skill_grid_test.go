package ui

import (
	"fmt"
	"maps"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestSkillGridRequirements(t *testing.T) {
	for _, tc := range []struct {
		name  string
		job   int
		skill uint16
		want  map[uint16]int
	}{
		{"blessing", db.JobAcolyte, db.SkillALBlessing, map[uint16]int{
			db.SkillALBlessing: 0, db.SkillALDp: 5,
		}},
		{"recursive", db.JobAcolyte, db.SkillALPneuma, map[uint16]int{
			db.SkillALPneuma: 0, db.SkillALWarp: 4, db.SkillALTeleport: 2, db.SkillALRuwach: 1,
		}},
		{"shared prerequisites keep highest level", db.JobKnight, db.SkillKNBowlingbash, map[uint16]int{
			db.SkillKNBowlingbash: 0, db.SkillSMBash: 10, db.SkillSMMagnum: 3,
			db.SkillSMTwohand: 5, db.SkillSMSword: 1, db.SkillKNTwohandquicken: 10, db.SkillKNAutocounter: 5,
		}},
		{"job override", db.JobCrusader, db.SkillALDp, map[uint16]int{
			db.SkillALDp: 0, db.SkillALCure: 1, db.SkillCRTrust: 5,
		}},
		{"no prerequisites", db.JobAcolyte, db.SkillALDp, map[uint16]int{
			db.SkillALDp: 0,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := skillGridRequirements(tc.job, tc.skill); !maps.Equal(got, tc.want) {
				t.Fatalf("requirements = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSkillGridHoverInvalidatesOnlyAffectedCells(t *testing.T) {
	ctx := uitest.NewMockContext()
	grid := newSkillGridWidget(skillGridConfig{
		job: db.JobAcolyte,
		entries: []skillGridEntry{
			{position: 0, skill: session.Skill{ID: db.SkillALBlessing}},
			{position: 1, skill: session.Skill{ID: db.SkillALDp}},
			{position: 2, skill: session.Skill{ID: db.SkillALHeal}},
		},
	})
	grid.Layout(ctx, geometry.Tight(geometry.Sz(skillGridViewW, skillGridViewH)))
	for _, step := range []struct {
		name      string
		position  int
		eventType event.MouseEventType
		want      map[uint16]int
		dirty     []int
	}{
		{"hover blessing", 0, event.MouseMove, map[uint16]int{db.SkillALBlessing: 0, db.SkillALDp: 5}, []int{0, 1}},
		{"same skill", 0, event.MouseMove, map[uint16]int{db.SkillALBlessing: 0, db.SkillALDp: 5}, nil},
		{"another skill", 2, event.MouseMove, map[uint16]int{db.SkillALHeal: 0}, []int{0, 1, 2}},
		{"empty cell", 3, event.MouseMove, nil, []int{2}},
		{"hover again", 0, event.MouseMove, map[uint16]int{db.SkillALBlessing: 0, db.SkillALDp: 5}, []int{0, 1}},
		{"leave grid", 0, event.MouseLeave, nil, []int{0, 1}},
	} {
		t.Run(step.name, func(t *testing.T) {
			ctx.InvalidatedRects = nil
			point := skillGridSlotBounds(grid.cellBounds(step.position)).Center()
			grid.Event(ctx, event.NewMouseEvent(step.eventType, event.ButtonNone, 0, point, point, 0))
			if !maps.Equal(grid.requiredLevels, step.want) {
				t.Fatalf("hover requirements = %v, want %v", grid.requiredLevels, step.want)
			}
			var wantRects []geometry.Rect
			for _, position := range step.dirty {
				wantRects = append(wantRects, grid.cellBounds(position))
			}
			if !slices.Equal(ctx.InvalidatedRects, wantRects) {
				t.Fatalf("invalidated = %v, want %v", ctx.InvalidatedRects, wantRects)
			}
		})
	}
}

func TestSkillGridHoverDrawsRequiredLevelEvenWhenAlreadyLearned(t *testing.T) {
	for _, learnedLevel := range []int{0, 5, 10} {
		skill := session.Skill{ID: db.SkillALDp, Level: learnedLevel}
		grid := newSkillGridWidget(skillGridConfig{
			job: db.JobAcolyte,
			entries: []skillGridEntry{
				{position: 0, skill: session.Skill{ID: db.SkillALBlessing}},
				{position: 1, skill: skill, display: skill, canStage: true},
			},
		})
		ctx := widget.NewContext()
		grid.Layout(ctx, geometry.Tight(geometry.Sz(skillGridViewW, skillGridViewH)))
		grid.setHover(ctx, 0, skillGridPartCell, true)
		canvas := &uitest.MockCanvas{}
		grid.Draw(ctx, canvas)
		for _, position := range []int{0, 1} {
			bounds := skillGridSlotBounds(grid.cellBounds(position))
			if !slices.ContainsFunc(canvas.RoundRects, func(call uitest.DrawRoundRectCall) bool {
				return call.Bounds == bounds && call.Color == rotheme.Default.Colors.SkillRequired
			}) {
				t.Fatalf("level %d: slot %d has no prerequisite highlight", learnedLevel, position)
			}
		}
		for _, call := range canvas.RoundRects {
			if call.Color == rotheme.Default.Colors.SkillRequired &&
				call.Bounds != skillGridSlotBounds(grid.cellBounds(0)) &&
				call.Bounds != skillGridSlotBounds(grid.cellBounds(1)) {
				t.Fatalf("level %d: highlight extends beyond the icon squares: %v", learnedLevel, call.Bounds)
			}
		}
		var badges []uitest.DrawStyledTextCall
		for _, call := range canvas.StyledTexts {
			if call.Style.Bold {
				badges = append(badges, call)
			}
		}
		if len(badges) != 1 || badges[0].Text != "5" || !grid.cellBounds(1).Contains(badges[0].Bounds.Center()) {
			t.Fatalf("level %d: required-level badges = %+v, want 5 on Divine Protection only", learnedLevel, badges)
		}
		grid.setHover(ctx, -1, skillGridPartCell, false)
		canvas.Reset()
		grid.Draw(ctx, canvas)
		for _, call := range canvas.RoundRects {
			if call.Color == rotheme.Default.Colors.SkillRequired {
				t.Fatalf("level %d: prerequisite highlight remains after leaving", learnedLevel)
			}
		}
	}
}

func TestSkillGridMovingWithinSkillDoesNotAllocate(t *testing.T) {
	grid := newSkillGridWidget(skillGridConfig{
		job:     db.JobAcolyte,
		entries: []skillGridEntry{{position: 0, skill: session.Skill{ID: db.SkillALBlessing}}},
	})
	ctx := widget.NewContext()
	grid.setHover(ctx, 0, skillGridPartCell, true)
	if allocs := testing.AllocsPerRun(100, func() {
		grid.setHover(ctx, 0, skillGridPartCell, true)
	}); allocs != 0 {
		t.Fatalf("unchanged hover allocations = %g, want 0", allocs)
	}
}

func TestSkillWindowGridUsesSelectedJobForRequirements(t *testing.T) {
	ctx := Context{Session: &session.Session{Selected: session.Character{Job: db.JobCrusader}}}
	window := &SkillWindow{tab: skillTabSecond, gridMode: true}
	window.skillGridWidget(ctx, nil, nil)
	if got := window.grid.cfg.job; got != db.JobCrusader {
		t.Fatalf("grid job = %d, want Crusader", got)
	}
}

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
		onPress: func(session.Skill) { pressed++ },
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

func TestSkillGridTextLayoutKeepsLearnedAndLockedSkillsInPlace(t *testing.T) {
	manager, err := res.NewManagerFS(fstest.MapFS{
		"data/skilltreeview.txt": {Data: []byte(`NOVICE
{
NV_BASIC#0#9@
NV_FIRSTAID#7#1@
NV_TRICKDEAD#14#1@
}
ACOLYTE
{
AL_HEAL#2#10@
AL_BLESSING#11#5#10@
}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range []int{0, 1} {
		skills := []session.Skill{
			{ID: db.SkillALBlessing, Level: level},
			{ID: db.SkillNVBasic, Level: 9},
			{ID: db.SkillALHeal, Level: 3},
			{ID: db.SkillNVFirstaid, Level: 1},
			{ID: db.SkillNVTrickdead, Level: level},
		}
		want := map[uint16]int{
			db.SkillNVBasic: 0, db.SkillNVFirstaid: 7, db.SkillNVTrickdead: 14,
			db.SkillALHeal: 2, db.SkillALBlessing: 11,
		}
		got := skillGridPositions(manager, db.JobMonk, skillTabFirst, skills)
		if !maps.Equal(got, want) {
			t.Fatalf("level %d: positions = %v, want %v", level, got, want)
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
	skills := window.activeSkills()

	if !containsSkill(skills, db.SkillAMLearningpotion) || !containsSkill(skills, db.SkillAMPharmacy) {
		t.Fatalf("grid skills omit the complete Alchemist tree: %+v", skills)
	}
	pharmacy := skills[skillIndex(skills, db.SkillAMPharmacy)]
	if pharmacy.Level != 0 || pharmacy.Upgradable {
		t.Fatalf("unavailable Pharmacy = %+v, want visible but locked", pharmacy)
	}
}

func TestSkillWindowFullTreeSurvivesViewChanges(t *testing.T) {
	for _, job := range []int16{db.JobAcolyte, db.JobMonk, db.JobMonkH, db.JobWizard} {
		t.Run(fmt.Sprint(job), func(t *testing.T) {
			ctx := Context{
				Session: &session.Session{Selected: session.Character{Job: job}, Skills: session.Skills{Points: 10}},
				ScreenW: 800, ScreenH: 600,
			}
			window := &SkillWindow{}
			window.OpenWindow(ctx)
			for _, group := range db.SkillTreeSkillGroups(int(job)) {
				window.skillTabColumn(ctx, nil, nil).Children()[group.ClassLevel-1].(*tabWidget).cfg.onClick()
				for range 3 { // List, grid, then list again.
					skills := window.activeSkills()
					if len(skills) != len(group.SkillIDs) {
						t.Fatalf("job %d tab %d grid=%t has %d skills, want %d", job, group.ClassLevel, window.gridMode, len(skills), len(group.SkillIDs))
					}
					for i, id := range group.SkillIDs {
						skill := skills[i]
						if skill.ID != id || skill.Level != 0 || window.canStageSkill(ctx.Session, skill) {
							t.Fatalf("skill %d = %+v, want visible and locked", id, skill)
						}
					}
					if window.gridMode {
						for _, entry := range window.grid.cfg.entries {
							if entry.canStage || entry.skill != skills[skillIndex(skills, entry.skill.ID)] {
								t.Fatalf("grid skill differs from list: %+v", entry)
							}
						}
					} else {
						window.ensureScrollSignal().Set(0)
						canvas := &uitest.MockCanvas{}
						window.content.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(skillWindowWidth, skillWindowHeight)))
						window.content.Draw(widget.NewContext(), canvas)
						if !slices.ContainsFunc(canvas.StyledTexts, func(call uitest.DrawStyledTextCall) bool {
							return call.Text == skillDisplayName(nil, skills[0])
						}) {
							t.Fatal("list did not draw its first locked skill")
						}
						window.ensureScrollSignal().Set(10000)
						window.ClampScroll(ctx.Session)
						if want := float32(maxInt(0, len(skills)*skillRowH-skillTableBodyH)); window.scrollY.Get() != want {
							t.Fatalf("list scroll limit = %v, want %v", window.scrollY.Get(), want)
						}
					}
					window.toggleGridMode(ctx, nil, nil)
				}
				window.toggleGridMode(ctx, nil, nil)
			}
		})
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
