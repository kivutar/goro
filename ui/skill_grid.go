package ui

import (
	"fmt"
	"image"
	"sort"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const skillGridSlotSize = 28

type skillGridEntry struct {
	position int
	skill    session.Skill
	display  session.Skill
	name     string
	icon     image.Image
	canStage bool
}

type skillGridConfig struct {
	entries       []skillGridEntry
	onPress       func(session.Skill, int, int)
	onStage       func(session.Skill)
	selectedLevel func(session.Skill) int
	onAdjustLevel func(session.Skill, int) int
	onHover       func(session.Skill, int, int)
	onLeave       func()
}

type skillGridPart uint8

const (
	skillGridPartCell skillGridPart = iota
	skillGridPartLevelDown
	skillGridPartLevelUp
)

type skillGridWidget struct {
	widget.WidgetBase
	cfg             skillGridConfig
	entryByPosition map[int]int
	hoveredPosition int
	hoveredPart     skillGridPart
	rows            int
}

func newSkillGridWidget(cfg skillGridConfig) *skillGridWidget {
	sort.SliceStable(cfg.entries, func(i, j int) bool {
		return cfg.entries[i].position < cfg.entries[j].position
	})
	w := &skillGridWidget{
		cfg:             cfg,
		entryByPosition: make(map[int]int, len(cfg.entries)),
		hoveredPosition: -1,
		rows:            skillGridMinRows,
	}
	for i, entry := range cfg.entries {
		w.entryByPosition[entry.position] = i
		w.rows = maxInt(w.rows, entry.position/skillGridColumns+1)
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *skillGridWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(skillGridViewW, float32(w.totalRows()*skillGridCellH)))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *skillGridWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.DrawRect(bounds, rotheme.Default.Colors.WindowBody)
	startRow, endRow := w.visibleRows(canvas)
	for row := startRow; row < endRow; row++ {
		for col := 0; col < skillGridColumns; col++ {
			position := row*skillGridColumns + col
			cell := w.cellBounds(position)
			entry, occupied := w.entryAtPosition(position)
			w.drawCell(canvas, cell, entry, occupied, position == w.hoveredPosition)
		}
	}
}

func (w *skillGridWidget) drawCell(canvas widget.Canvas, cell geometry.Rect, entry skillGridEntry, occupied, hovered bool) {
	if hovered && occupied {
		highlight := rotheme.Default.Colors.ButtonHover
		highlight.A = 0.45
		canvas.DrawRoundRect(cell.Inset(geometry.UniformInsets(1)), highlight, 4)
	}

	slot := skillGridSlotBounds(cell)
	slotColor := rotheme.Default.Colors.WindowBody
	if occupied && entry.canStage {
		slotColor = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRoundRect(slot, slotColor, 3)
	outline := rotheme.Default.Colors.FooterLine
	outline.A *= 0.25
	canvas.StrokeRoundRect(slot, outline, 3, 1)
	if !occupied {
		return
	}

	textColor := rotheme.Default.Colors.Text
	if entry.display.Level <= 0 {
		textColor = rotheme.Default.Colors.MutedText
	}
	rotheme.DrawText(
		canvas,
		trimRunes(entry.name, 10),
		geometry.NewRect(cell.Min.X+1, cell.Min.Y, cell.Width()-2, 14),
		rotheme.Default.Typography.TextSize,
		textColor,
		false,
		widget.TextAlignCenter,
	)
	if entry.icon != nil {
		iconBounds := geometry.NewRect(slot.Min.X+2, slot.Min.Y+2, skillIconSize, skillIconSize)
		canvas.DrawImage(entry.icon, iconBounds.Min)
		if entry.display.Level <= 0 && !entry.canStage {
			shade := rotheme.Default.Colors.WindowBody
			shade.A = 0.5
			canvas.DrawRect(iconBounds, shade)
		}
	}

	w.drawLevel(canvas, cell, entry, textColor, hovered)
}

func (w *skillGridWidget) drawLevel(canvas widget.Canvas, cell geometry.Rect, entry skillGridEntry, color widget.Color, hovered bool) {
	selectable, known := db.SkillLevelSelectable(entry.skill.ID)
	if !known || !selectable || entry.skill.Level <= 0 {
		rotheme.DrawText(
			canvas,
			fmt.Sprintf("Lv %d", entry.display.Level),
			geometry.NewRect(cell.Min.X+1, cell.Max.Y-15, cell.Width()-2, 14),
			rotheme.Default.Typography.TextSize,
			color,
			false,
			widget.TextAlignCenter,
		)
		return
	}
	level := entry.skill.Level
	if w.cfg.selectedLevel != nil {
		level = w.cfg.selectedLevel(entry.skill)
	}
	down := skillGridPartBounds(cell, skillGridPartLevelDown)
	up := skillGridPartBounds(cell, skillGridPartLevelUp)
	rotheme.DrawIconButton(canvas, down, rotheme.IconButtonLeft, hovered && w.hoveredPart == skillGridPartLevelDown, level <= 1)
	rotheme.DrawIconButton(canvas, up, rotheme.IconButtonRight, hovered && w.hoveredPart == skillGridPartLevelUp, level >= entry.skill.Level)
	rotheme.DrawText(
		canvas,
		fmt.Sprintf("%d/%d", level, entry.skill.Level),
		geometry.NewRect(down.Max.X, cell.Max.Y-15, up.Min.X-down.Max.X, 14),
		rotheme.Default.Typography.TextSize,
		color,
		false,
		widget.TextAlignCenter,
	)
}

func (w *skillGridWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	position := w.positionAt(mouse.Position)
	entry, occupied := w.entryAtPosition(position)
	part := skillGridPartCell
	if occupied {
		part = skillGridHitPart(entry, w.cellBounds(position), mouse.Position)
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		w.setHover(ctx, position, part, occupied)
		if occupied {
			ctx.SetCursor(widget.CursorPointer)
			if w.cfg.onHover != nil {
				w.cfg.onHover(entry.skill, int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
			}
		} else {
			ctx.SetCursor(widget.CursorDefault)
			if w.cfg.onLeave != nil {
				w.cfg.onLeave()
			}
		}
		return true
	case event.MouseLeave:
		w.setHover(ctx, -1, skillGridPartCell, false)
		ctx.SetCursor(widget.CursorDefault)
		if w.cfg.onLeave != nil {
			w.cfg.onLeave()
		}
		return false
	case event.MousePress:
		if mouse.Button != event.ButtonLeft || !occupied {
			return true
		}
		switch part {
		case skillGridPartLevelDown:
			w.adjustLevel(ctx, position, entry.skill, -1)
		case skillGridPartLevelUp:
			w.adjustLevel(ctx, position, entry.skill, 1)
		default:
			if entry.canStage && w.cfg.onStage != nil {
				w.cfg.onStage(entry.skill)
			} else if w.cfg.onPress != nil {
				w.cfg.onPress(entry.skill, int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
			}
		}
		return true
	}
	return true
}

func (w *skillGridWidget) adjustLevel(ctx widget.Context, position int, skill session.Skill, delta int) {
	if w.cfg.onAdjustLevel == nil || w.cfg.selectedLevel == nil || delta == 0 {
		return
	}
	before := w.cfg.selectedLevel(skill)
	after := w.cfg.onAdjustLevel(skill, delta)
	if before == after {
		return
	}
	w.SetNeedsRedraw(true)
	ctx.InvalidateRect(w.cellBounds(position))
}

func (w *skillGridWidget) setHover(ctx widget.Context, position int, part skillGridPart, occupied bool) {
	if !occupied {
		position = -1
		part = skillGridPartCell
	}
	if w.hoveredPosition == position && w.hoveredPart == part {
		return
	}
	old := w.hoveredPosition
	w.hoveredPosition = position
	w.hoveredPart = part
	w.SetNeedsRedraw(true)
	if old >= 0 {
		ctx.InvalidateRect(w.cellBounds(old))
	}
	if position >= 0 {
		ctx.InvalidateRect(w.cellBounds(position))
	}
}

func skillGridHitPart(entry skillGridEntry, cell geometry.Rect, point geometry.Point) skillGridPart {
	selectable, known := db.SkillLevelSelectable(entry.skill.ID)
	if !known || !selectable || entry.skill.Level <= 0 {
		return skillGridPartCell
	}
	if skillGridPartBounds(cell, skillGridPartLevelDown).Contains(point) {
		return skillGridPartLevelDown
	}
	if skillGridPartBounds(cell, skillGridPartLevelUp).Contains(point) {
		return skillGridPartLevelUp
	}
	return skillGridPartCell
}

func skillGridSlotBounds(cell geometry.Rect) geometry.Rect {
	return geometry.NewRect(
		cell.Min.X+(cell.Width()-skillGridSlotSize)/2,
		cell.Min.Y+14,
		skillGridSlotSize,
		skillGridSlotSize,
	)
}

func skillGridPartBounds(cell geometry.Rect, part skillGridPart) geometry.Rect {
	switch part {
	case skillGridPartLevelDown:
		return geometry.NewRect(cell.Min.X+1, cell.Max.Y-rotheme.IconButtonSize, rotheme.IconButtonSize, rotheme.IconButtonSize)
	case skillGridPartLevelUp:
		return geometry.NewRect(cell.Max.X-rotheme.IconButtonSize-1, cell.Max.Y-rotheme.IconButtonSize, rotheme.IconButtonSize, rotheme.IconButtonSize)
	default:
		return cell
	}
}

func (w *skillGridWidget) cellBounds(position int) geometry.Rect {
	bounds := w.Bounds()
	col := position % skillGridColumns
	row := position / skillGridColumns
	return geometry.NewRect(
		bounds.Min.X+float32(col*skillGridCellW),
		bounds.Min.Y+float32(row*skillGridCellH),
		skillGridCellW,
		skillGridCellH,
	)
}

func (w *skillGridWidget) positionAt(point geometry.Point) int {
	local := point.Sub(w.Bounds().Min)
	if local.X < 0 || local.Y < 0 || local.X >= skillGridColumns*skillGridCellW || local.Y >= float32(w.totalRows()*skillGridCellH) {
		return -1
	}
	return int(local.Y)/skillGridCellH*skillGridColumns + int(local.X)/skillGridCellW
}

func (w *skillGridWidget) entryAtPosition(position int) (skillGridEntry, bool) {
	index, ok := w.entryByPosition[position]
	if !ok || index < 0 || index >= len(w.cfg.entries) {
		return skillGridEntry{}, false
	}
	return w.cfg.entries[index], true
}

func (w *skillGridWidget) totalRows() int {
	return maxInt(skillGridMinRows, w.rows)
}

func (w *skillGridWidget) visibleRows(canvas widget.Canvas) (int, int) {
	clip := canvas.ClipBounds()
	if clip.IsEmpty() {
		return 0, 0
	}
	offset := canvas.TransformOffset()
	localTop := clip.Min.Y - offset.Y
	localBottom := clip.Max.Y - offset.Y
	if localBottom <= 0 || localTop >= float32(w.totalRows()*skillGridCellH) {
		return 0, 0
	}
	start := maxInt(0, int(localTop)/skillGridCellH)
	end := minInt(w.totalRows(), int(localBottom)/skillGridCellH+1)
	return start, maxInt(start, end)
}

func (w *SkillWindow) skillGridEntries(ctx Context, assets AssetProvider) []skillGridEntry {
	w.ensureSkillView(ctx)
	skills := w.gridSkills(ctx)
	positions := skillGridPositions(ctx.Resources, selectedJob(ctx.Session), w.tab, skills)
	entries := make([]skillGridEntry, 0, len(skills))
	for _, skill := range skills {
		entries = append(entries, skillGridEntry{
			position: positions[skill.ID],
			skill:    skill,
			display:  w.skillWithPending(skill),
			name:     skillDisplayName(ctx.Resources, skill),
			icon:     w.skillIconImage(ctx, assets, skill),
			canStage: w.canStageSkill(ctx.Session, skill),
		})
	}
	return entries
}

func (w *SkillWindow) gridSkills(ctx Context) []session.Skill {
	if w.tab == skillTabEtc {
		return append([]session.Skill(nil), w.activeSkills()...)
	}
	job := selectedJob(ctx.Session)
	groups := db.SkillTreeSkillGroups(job)
	classLevel := w.tab + 1
	var skillIDs []uint16
	for _, group := range groups {
		if group.ClassLevel == classLevel {
			skillIDs = group.SkillIDs
			break
		}
	}
	byID := make(map[uint16]session.Skill, len(w.allSkills))
	for _, skill := range w.allSkills {
		byID[skill.ID] = skill
	}
	seen := make(map[uint16]bool, len(skillIDs))
	skills := make([]session.Skill, 0, len(skillIDs))
	for _, skillID := range skillIDs {
		if seen[skillID] {
			continue
		}
		seen[skillID] = true
		if skill, ok := byID[skillID]; ok {
			skills = append(skills, skill)
			continue
		}
		skill := w.lockedSkill(ctx, skillID)
		skill.Upgradable = false
		skills = append(skills, skill)
	}
	return skills
}

func skillGridPositions(manager *res.Manager, job, tab int, skills []session.Skill) map[uint16]int {
	desired := make(map[uint16]bool, len(skills))
	for _, skill := range skills {
		desired[skill.ID] = true
	}
	positions := make(map[uint16]int, len(skills))
	if manager != nil && tab != skillTabEtc {
		for _, layoutJob := range db.SkillTreeLayoutJobs(job, tab+1) {
			layout, ok := manager.SkillTreePositions(layoutJob)
			if !ok {
				continue
			}
			for skillID, position := range layout {
				id := uint16(skillID)
				if desired[id] && position >= 0 {
					positions[id] = position
				}
			}
		}
	}

	occupied := make(map[int]bool, len(skills))
	for _, skill := range skills {
		if position, ok := positions[skill.ID]; ok && !occupied[position] {
			occupied[position] = true
			continue
		}
		delete(positions, skill.ID)
	}
	next := 0
	for _, skill := range skills {
		if _, ok := positions[skill.ID]; ok {
			continue
		}
		for occupied[next] {
			next++
		}
		positions[skill.ID] = next
		occupied[next] = true
		next++
	}
	return positions
}

func (w *SkillWindow) skillGridAtMouse(mouseX, mouseY int) (session.Skill, bool) {
	if w.grid == nil {
		return session.Skill{}, false
	}
	x := w.x + skillTabRailW + verticalTabDividerW
	y := w.y + ROWindowTitleHeight
	if !pointInRect(mouseX, mouseY, x, y, skillGridColumns*skillGridCellW, skillGridViewH) {
		return session.Skill{}, false
	}
	col := (mouseX - x) / skillGridCellW
	row := int((float32(mouseY-y) + w.ensureScrollSignal().Get()) / skillGridCellH)
	entry, ok := w.grid.entryAtPosition(row*skillGridColumns + col)
	return entry.skill, ok
}

func selectedJob(s *session.Session) int {
	if s == nil {
		return db.JobNovice
	}
	return int(s.Selected.Job)
}
