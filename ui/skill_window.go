package ui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	skillTabW         = 32
	skillTabH         = 44
	skillTabOver      = 1
	skillTabRailW     = skillTabW + skillTabOver*2
	skillTableViewW   = 396
	skillWindowWidth  = skillTabRailW + verticalTabDividerW + skillTableViewW
	skillWindowHeight = 388
	skillRowH         = 32
	skillIconSize     = 24
	skillHeaderH      = 24
	skillTableViewH   = skillWindowHeight - ROWindowTitleHeight - ROWindowFooterHeight
	skillTableBodyH   = skillTableViewH - skillHeaderH

	skillGridColumns      = 7
	skillGridCellW        = 62
	skillGridCellH        = 58
	skillGridMinRows      = 6
	skillGridViewW        = skillGridColumns*skillGridCellW + ROScrollbarGutter
	skillGridViewH        = skillGridMinRows * skillGridCellH
	skillGridWindowWidth  = skillTabRailW + verticalTabDividerW + skillGridViewW
	skillGridWindowHeight = ROWindowTitleHeight + skillGridViewH + ROWindowFooterHeight
)

const (
	skillTabFirst = iota
	skillTabSecond
	skillTabThird
	skillTabFourth
	skillTabEtc
	skillTabCount
)

var skillTabs = [...]struct {
	label string
	tab   int
}{
	{label: "1st", tab: skillTabFirst},
	{label: "2nd", tab: skillTabSecond},
	{label: "3rd", tab: skillTabThird},
	{label: "4th", tab: skillTabFourth},
	{label: "Etc", tab: skillTabEtc},
}

type SkillWindow struct {
	Window
	tab            int
	scrollY        state.Signal[float32]
	snapshot       string
	skillViewKey   string
	skillViewReady bool
	allSkills      []session.Skill
	skillsByTab    [skillTabCount][]session.Skill
	visibleTabs    []int
	lastClick      uint16
	lastClickAt    time.Time
	dragSkill      session.Skill
	dragActive     bool
	dragFrom       time.Time
	hoveredSkill   session.Skill
	hasHover       bool
	hoverX         int
	hoverY         int
	tooltip        tooltipState
	pending        map[uint16]int
	pendingOrder   []uint16
	dirty          bool
	icons          map[uint16]image.Image
	iconMiss       map[uint16]struct{}
	lastIconAssets bool
	assets         AssetProvider
	actions        GameActions
	table          *rotheme.TableViewWidget
	grid           *skillGridWidget
	gridMode       bool
	gridRows       int
	selectedLevels map[uint16]int
}

func (w *SkillWindow) Toggle(ctx Context) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if w.IsOpen() {
		w.close(ctx)
		return
	}
	w.OpenWindow(ctx)
}

func (w *SkillWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if w.IsOpen() {
		w.Publish(ctx)
		w.Raise(ctx)
		return
	}
	w.openAtDefault(ctx)
}

func (w *SkillWindow) Update(ctx Context, shortcuts *ShortcutBar, actions GameActions) bool {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.actions = actions
	if assets, ok := actions.(AssetProvider); ok && assets != nil && !w.lastIconAssets {
		w.assets = assets
		w.lastIconAssets = true
		w.snapshot = ""
	}
	if ctx.Input == nil {
		w.Publish(ctx)
		return true
	}
	w.updateTooltipHover(ctx)
	if w.UpdateDrag(ctx, shortcuts) {
		return true
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		w.close(ctx)
		w.Publish(ctx)
		return true
	}
	snapshot := w.skillSnapshot(ctx.Session)
	w.ensureSkillViewForSnapshot(ctx, snapshot)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.SetContent(w.widgetTree(ctx, actions))
	}
	w.clampScroll(ctx)
	consumed := w.Window.Update(ctx)
	if w.dirty {
		w.dirty = false
		w.refresh(ctx, actions)
		return true
	}
	if !w.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *SkillWindow) UpdateDrag(ctx Context, shortcuts *ShortcutBar) bool {
	if !w.dragActive || ctx.Input == nil {
		return false
	}
	if ctx.Input.MouseJustReleased(input.MouseButtonLeft) || !ctx.Input.MousePressed(input.MouseButtonLeft) {
		skill := w.dragSkill
		w.dragActive = false
		w.dragSkill = session.Skill{}
		if shortcuts != nil && shortcuts.AcceptSkillDrop(ctx, skill, ctx.Input.MouseX, ctx.Input.MouseY) {
			w.Publish(ctx)
			return true
		}
		w.Publish(ctx)
		return true
	}
	return true
}

func (w *SkillWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if !w.IsOpen() {
		w.Unpublish(ctx)
		w.hideTooltip()
		return
	}
	w.Publish(ctx)
}

func (w *SkillWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if w.dragActive && screen != nil && ctx.Input != nil && time.Since(w.dragFrom) > 80*time.Millisecond && assets != nil {
		assets.DrawSkillIcon(screen, ctx.Resources, w.dragSkill, ctx.Input.MouseX-skillIconSize/2, ctx.Input.MouseY-skillIconSize/2, skillIconSize)
	}
}

func (w *SkillWindow) Publish(ctx Context) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if !w.IsOpen() {
		w.Unpublish(ctx)
		w.hideTooltip()
		return
	}
	w.Window.Publish(ctx)
}

func (w *SkillWindow) Rebind(ctx Context, actions GameActions) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx, actions)
}

func (w *SkillWindow) openAtDefault(ctx Context) {
	width, height := w.windowSize()
	w.SetSize(width, height)
	x, y := skillWindowDefaultPosition(ctx, width, height)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.ensureSkillViewForSnapshot(ctx, w.snapshot)
	w.ensureScrollSignal().Set(0)
	w.OpenAt(x, y, w.widgetTree(ctx, w.actions))
	w.Publish(ctx)
}

func (w *SkillWindow) close(ctx Context) {
	w.dragActive = false
	w.hasHover = false
	w.hideTooltip()
	w.Window.Close()
	w.Publish(ctx)
}

func (w *SkillWindow) widgetTree(ctx Context, actions GameActions) widget.Widget {
	return w.widgetTreeWithAssets(ctx, nil, actions)
}

func (w *SkillWindow) widgetTreeWithAssets(ctx Context, assets AssetProvider, actions GameActions) widget.Widget {
	if actions == nil {
		actions = w.actions
	}
	if assets == nil {
		assets = w.assets
	}
	width, height := w.windowSize()
	var body widget.Widget
	viewWidth := 0
	viewBackground := rotheme.Default.Colors.PanelBody
	if w.gridMode {
		body = w.skillGridWidget(ctx, assets, actions)
		viewWidth = skillGridViewW
		viewBackground = rotheme.Default.Colors.WindowBody
	} else {
		body = w.skillTableWidget(ctx, assets, actions)
		viewWidth = skillTableViewW
	}
	w.setTitleButtonCount(2) // Minus and close.
	return Win(
		Title("Skill Tree"),
		TitleButton(rotheme.IconButtonMinus, func() {
			w.toggleGridMode(ctx, assets, actions)
		}),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
		}),
		Size(float32(width), float32(height)),
		Content(
			verticalTabFrame(
				w.skillTabColumn(ctx, assets, actions),
				primitives.Box(body).
					Width(float32(viewWidth)).
					Height(float32(w.contentHeight())).
					Background(viewBackground),
			),
		),
		Footer(
			footerLabel(fmt.Sprintf("Skill Points: %d", maxInt(0, sessionSkillPoints(ctx.Session)-w.pendingCount()))),
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Reset", func() {
				w.clearPending()
				w.dirty = true
			}),
			rotheme.Button("Confirm", func() {
				w.confirmPending(ctx)
				w.dirty = true
			}),
		),
	)
}

func (w *SkillWindow) skillTableWidget(ctx Context, assets AssetProvider, actions GameActions) *rotheme.TableViewWidget {
	w.ensureSkillView(ctx)
	skills := w.activeSkills()
	table := rotheme.TableView(
		rotheme.TableViewColumns(skillTableColumns),
		rotheme.TableViewRowCount(len(skills)),
		rotheme.TableViewRowHeight(skillRowH),
		rotheme.TableViewHeaderHeight(skillHeaderH),
		rotheme.TableViewEmptyText("No skills received from server yet."),
		rotheme.TableViewScrollYSignal(w.ensureScrollSignal()),
		rotheme.TableViewDispatchHoverToCells(false),
		rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
			if cell.Row < 0 || cell.Row >= len(skills) {
				return rotheme.TableViewSimpleCell{Hidden: true}
			}
			return w.skillTableCell(ctx, assets, skills[cell.Row], cell)
		}),
		rotheme.TableViewOnRowEventWithContext(func(widgetCtx widget.Context, row int, e event.Event) bool {
			return w.handleSkillTableRowEvent(widgetCtx, ctx, actions, skills, row, e)
		}),
	)
	w.table = table
	w.grid = nil
	return table
}

func (w *SkillWindow) skillGridWidget(ctx Context, assets AssetProvider, actions GameActions) widget.Widget {
	entries := w.skillGridEntries(ctx, assets)
	grid := newSkillGridWidget(skillGridConfig{
		entries: entries,
		onPress: func(skill session.Skill, mx, my int) {
			w.pressSkill(ctx, actions, skill, mx, my)
		},
		onStage: func(skill session.Skill) {
			if !w.canStageSkill(ctx.Session, skill) {
				return
			}
			w.stageSkill(skill.ID)
			w.dirty = true
		},
		selectedLevel: w.selectedSkillLevel,
		onAdjustLevel: func(skill session.Skill, delta int) int {
			w.changeSelectedSkillLevel(skill, delta)
			return w.selectedSkillLevel(skill)
		},
		onHover: func(skill session.Skill, mx, my int) {
			w.hoveredSkill = skill
			w.hasHover = true
			w.hoverX = mx
			w.hoverY = my
			w.showTooltip(ctx, skill, mx, my)
		},
		onLeave: func() {
			w.hasHover = false
			w.hideTooltip()
		},
	})
	w.grid = grid
	w.table = nil
	w.gridRows = grid.totalRows()
	return scrollview.New(
		grid,
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
		scrollview.ScrollYSignal(w.ensureScrollSignal()),
		scrollview.ScrollStep(skillGridCellH),
	)
}

func (w *SkillWindow) skillTabColumn(ctx Context, assets AssetProvider, actions GameActions) widget.Widget {
	w.ensureSkillView(ctx)
	tabs := make([]widget.Widget, 0, len(w.visibleTabs))
	for _, tabID := range w.visibleTabs {
		tab := skillTabs[tabID]
		tabs = append(tabs, newTabWidget(tabWidgetConfig{
			label:         tab.label,
			labelRotation: rotheme.TextRotationCounterClockwise,
			active:        tab.tab == w.tab,
			width:         skillTabRailW,
			height:        skillTabH,
			onClick: func() {
				if w.tab == tab.tab {
					return
				}
				w.tab = tab.tab
				w.ensureScrollSignal().Set(0)
				w.hasHover = false
				w.hideTooltip()
				w.SetContent(w.widgetTreeWithAssets(ctx, assets, actions))
				w.Publish(ctx)
			},
		}))
	}
	return primitives.Box(tabs...).
		Width(skillTabRailW).
		Height(float32(w.contentHeight())).
		Gap(-skillTabOver)
}

func (w *SkillWindow) skillTableCell(ctx Context, assets AssetProvider, skill session.Skill, cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
	display := w.skillWithPending(skill)
	nameColor := rotheme.Default.Colors.Text
	if display.Level <= 0 {
		nameColor = rotheme.Default.Colors.MutedText
	}
	switch cell.Column.Key {
	case "icon":
		return rotheme.TableViewSimpleCell{Icon: w.skillIconImage(ctx, assets, skill)}
	case "type":
		return rotheme.TableViewSimpleCell{
			Text:  skillTypeLabel(display),
			Color: skillTypeColor(display),
		}
	case "name":
		return rotheme.TableViewSimpleCell{
			Text:  trimRunes(skillDisplayName(ctx.Resources, display), 18),
			Color: nameColor,
		}
	case "level":
		level := w.selectedSkillLevel(skill)
		text := fmt.Sprintf("%d", display.Level)
		if selectable, known := db.SkillLevelSelectable(skill.ID); known && selectable && skill.Level > 0 {
			text = fmt.Sprintf("%d/%d", level, display.Level)
		}
		return rotheme.TableViewSimpleCell{
			Text:  text,
			Align: widget.TextAlignCenter,
			Color: nameColor,
		}
	case "leveldown":
		selectable, known := db.SkillLevelSelectable(skill.ID)
		if !known || !selectable || skill.Level <= 0 {
			return rotheme.TableViewSimpleCell{Hidden: true}
		}
		return rotheme.TableViewIconButtonCell(rotheme.IconButtonLeft, w.selectedSkillLevel(skill) <= 1)
	case "levelupselect":
		selectable, known := db.SkillLevelSelectable(skill.ID)
		if !known || !selectable || skill.Level <= 0 {
			return rotheme.TableViewSimpleCell{Hidden: true}
		}
		return rotheme.TableViewIconButtonCell(rotheme.IconButtonRight, w.selectedSkillLevel(skill) >= skill.Level)
	case "sp":
		return rotheme.TableViewSimpleCell{
			Text:  fmt.Sprintf("%d", display.SPCost),
			Color: rotheme.Default.Colors.MutedText,
		}
	case "range":
		return rotheme.TableViewSimpleCell{
			Text:  fmt.Sprintf("%d", display.Range),
			Color: rotheme.Default.Colors.MutedText,
		}
	case "levelup":
		return skillLevelUpButtonCell(w.canStageSkill(ctx.Session, skill))
	default:
		return rotheme.TableViewSimpleCell{Hidden: true}
	}
}

func skillLevelUpButtonCell(visible bool) rotheme.TableViewSimpleCell {
	if !visible {
		return rotheme.TableViewSimpleCell{Hidden: true}
	}
	return rotheme.TableViewIconButtonCell(rotheme.IconButtonPlus, false)
}

func (w *SkillWindow) handleSkillTableRowEvent(widgetCtx widget.Context, ctx Context, actions GameActions, skills []session.Skill, row int, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok || row < 0 || row >= len(skills) {
		return false
	}
	skill := skills[row]
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		display := w.skillWithPending(skill)
		if display.Level > 0 || w.canStageSkill(ctx.Session, skill) {
			widgetCtx.SetCursor(widget.CursorPointer)
		}
		mx, my := int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y)
		if ctx.Input != nil {
			mx, my = ctx.Input.MouseX, ctx.Input.MouseY
		}
		w.hoveredSkill = skill
		w.hasHover = true
		w.hoverX = mx
		w.hoverY = my
		w.showTooltip(ctx, skill, mx, my)
		return false
	case event.MousePress:
		if mouse.Button != event.ButtonLeft {
			return false
		}
		if selectable, known := db.SkillLevelSelectable(skill.ID); known && selectable && skill.Level > 0 {
			if skillTableButtonBounds(row, "leveldown").Contains(mouse.Position) {
				w.adjustSelectedSkillLevel(widgetCtx, skill, row, -1)
				return true
			}
			if skillTableButtonBounds(row, "levelupselect").Contains(mouse.Position) {
				w.adjustSelectedSkillLevel(widgetCtx, skill, row, 1)
				return true
			}
		}
		if skillTableButtonBounds(row, "levelup").Contains(mouse.Position) {
			if !w.canStageSkill(ctx.Session, skill) {
				glog.Debugf("skill level up ignored id=%d: no points or maxed", skill.ID)
				return true
			}
			w.stageSkill(skill.ID)
			w.dirty = true
			return true
		}
		w.pressSkill(ctx, actions, skill, int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
		return true
	}
	return false
}

func (w *SkillWindow) refresh(ctx Context, actions GameActions) {
	if actions != nil {
		w.actions = actions
	}
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.ensureSkillViewForSnapshot(ctx, w.snapshot)
	w.SetContent(w.widgetTreeWithAssets(ctx, w.assets, w.actions))
	w.clampScroll(ctx)
	w.Publish(ctx)
}

func (w *SkillWindow) toggleGridMode(ctx Context, assets AssetProvider, actions GameActions) {
	w.gridMode = !w.gridMode
	w.dragActive = false
	w.hasHover = false
	w.hideTooltip()
	w.ensureScrollSignal().Set(0)
	width, height := w.windowSize()
	w.SetSize(width, height)
	screenW, screenH := ctx.ScreenSize()
	w.x = clampWindowInt(w.x, windowScreenMargin, maxInt(windowScreenMargin, screenW-width-windowScreenMargin))
	w.y = clampWindowInt(w.y, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	w.SetContent(w.widgetTreeWithAssets(ctx, assets, actions))
	w.Publish(ctx)
}

func (w *SkillWindow) pressSkill(ctx Context, actions GameActions, skill session.Skill, mx, my int) {
	if skill.Level <= 0 {
		glog.Debugf("skill use ignored id=%d: not learned", skill.ID)
		return
	}
	if !skillCanUseShortcut(skill) {
		glog.Debugf("skill use ignored id=%d: passive skill", skill.ID)
		return
	}
	skill.Level = w.selectedSkillLevel(skill)
	now := time.Now()
	if w.lastClick == skill.ID && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.lastClick = 0
		w.lastClickAt = time.Time{}
		if actions == nil {
			glog.Warnf("skill use failed id=%d: no game actions", skill.ID)
			return
		}
		if err := actions.UseShortcutSkill(ctx, skill); err != nil {
			glog.Warnf("skill use failed id=%d: %v", skill.ID, err)
		}
		return
	}
	w.lastClick = skill.ID
	w.lastClickAt = now
	w.dragSkill = skill
	w.dragActive = true
	w.hideTooltip()
	w.dragFrom = now
	w.hoverX = mx
	w.hoverY = my
}

func (w *SkillWindow) showTooltip(ctx Context, skill session.Skill, mx, my int) {
	if w.dragActive || skill.ID == 0 {
		w.hideTooltip()
		return
	}
	const tooltipW = 292
	text := skillTooltipText(ctx, skill)
	w.tooltip.ShowBox(ctx, text, mx+16+tooltipW/2, my+18, my-6, tooltipW, 24)
}

func (w *SkillWindow) updateTooltipHover(ctx Context) {
	if !w.hasHover || ctx.Input == nil {
		return
	}
	skill, ok := w.skillAtMouse(ctx, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok || skill.ID != w.hoveredSkill.ID {
		w.hasHover = false
		w.hideTooltip()
	}
}

func (w *SkillWindow) skillAtMouse(ctx Context, mouseX, mouseY int) (session.Skill, bool) {
	if w.gridMode {
		return w.skillGridAtMouse(mouseX, mouseY)
	}
	x, y := w.skillTableBodyOrigin()
	if !pointInRect(mouseX, mouseY, x, y, scrollbarSafeIntWidth(skillTableViewW), skillTableBodyH) {
		return session.Skill{}, false
	}
	row := int((float32(mouseY-y) + w.ensureScrollSignal().Get()) / skillRowH)
	w.ensureSkillView(ctx)
	skills := w.activeSkills()
	if row < 0 || row >= len(skills) {
		return session.Skill{}, false
	}
	return skills[row], true
}

func (w *SkillWindow) skillTableBodyOrigin() (int, int) {
	return w.x + skillTabRailW + verticalTabDividerW, w.y + ROWindowTitleHeight + skillHeaderH
}

func (w *SkillWindow) hideTooltip() {
	w.tooltip.Hide()
}

func (w *SkillWindow) DrawTooltip(ctx Context, screen *render.Frame) {
	w.tooltip.Draw(ctx, screen)
}

func (w *SkillWindow) skillIconImage(ctx Context, assets AssetProvider, skill session.Skill) image.Image {
	if assets == nil || skill.ID == 0 {
		return nil
	}
	if w.icons != nil {
		if img := w.icons[skill.ID]; img != nil {
			return img
		}
	}
	if _, ok := w.iconMiss[skill.ID]; ok {
		return nil
	}
	img := assets.SkillIconImage(ctx.Resources, skill, skillIconSize)
	if img == nil {
		if w.iconMiss == nil {
			w.iconMiss = make(map[uint16]struct{})
		}
		w.iconMiss[skill.ID] = struct{}{}
		return nil
	}
	if w.icons == nil {
		w.icons = make(map[uint16]image.Image)
	}
	w.icons[skill.ID] = img
	return img
}

func (w *SkillWindow) clearPending() {
	w.pending = nil
	w.pendingOrder = nil
}

func (w *SkillWindow) pendingCount() int {
	total := 0
	for _, count := range w.pending {
		total += count
	}
	return total
}

func (w *SkillWindow) pendingFor(skillID uint16) int {
	if w.pending == nil {
		return 0
	}
	return w.pending[skillID]
}

func (w *SkillWindow) skillWithPending(skill session.Skill) session.Skill {
	skill.Level += w.pendingFor(skill.ID)
	return skill
}

func (w *SkillWindow) selectedSkillLevel(skill session.Skill) int {
	if skill.Level <= 0 {
		return 0
	}
	selectable, known := db.SkillLevelSelectable(skill.ID)
	if !known || !selectable {
		return skill.Level
	}
	level := skill.Level
	if w.selectedLevels != nil {
		if selected := w.selectedLevels[skill.ID]; selected > 0 {
			level = selected
		}
	}
	return maxInt(1, minInt(skill.Level, level))
}

func (w *SkillWindow) adjustSelectedSkillLevel(ctx widget.Context, skill session.Skill, row, delta int) bool {
	if !w.changeSelectedSkillLevel(skill, delta) {
		return false
	}
	if w.table != nil && ctx != nil {
		w.table.InvalidateRow(ctx, row)
	}
	return true
}

func (w *SkillWindow) changeSelectedSkillLevel(skill session.Skill, delta int) bool {
	selectable, known := db.SkillLevelSelectable(skill.ID)
	if !known || !selectable || skill.Level <= 0 || delta == 0 {
		return false
	}
	current := w.selectedSkillLevel(skill)
	next := maxInt(1, minInt(skill.Level, current+delta))
	if next == current {
		return false
	}
	if w.selectedLevels == nil {
		w.selectedLevels = make(map[uint16]int)
	}
	if next == skill.Level {
		delete(w.selectedLevels, skill.ID)
	} else {
		w.selectedLevels[skill.ID] = next
	}
	return true
}

func (w *SkillWindow) stageSkill(skillID uint16) {
	if w.pending == nil {
		w.pending = make(map[uint16]int)
	}
	if w.pending[skillID] == 0 {
		w.pendingOrder = append(w.pendingOrder, skillID)
	}
	w.pending[skillID]++
}

func (w *SkillWindow) canStageSkill(s *session.Session, skill session.Skill) bool {
	if !canIncreaseSkill(s, skill) {
		return false
	}
	pending := w.pendingFor(skill.ID)
	if maxLevel := skillMaxLevel(skill); maxLevel > 0 {
		return skill.Level+pending < maxLevel && w.pendingCount() < sessionSkillPoints(s)
	}
	return w.pendingCount() < sessionSkillPoints(s)
}

func (w *SkillWindow) confirmPending(ctx Context) {
	if len(w.pendingOrder) == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("skill level up failed: not connected")
		return
	}
	for _, skillID := range w.pendingOrder {
		for i := 0; i < w.pending[skillID]; i++ {
			if err := ctx.Network.SendSkillLevelUp(skillID); err != nil {
				glog.Warnf("skill level up failed id=%d: %v", skillID, err)
				return
			}
		}
	}
	w.clearPending()
}

func (w *SkillWindow) clampScroll(ctx Context) {
	w.ensureSkillView(ctx)
	if w.gridMode {
		w.clampScrollHeight(w.gridRows*skillGridCellH, skillGridViewH)
		return
	}
	w.clampScrollCount(len(w.activeSkills()))
}

func (w *SkillWindow) clampScrollHeight(contentHeight, viewHeight int) {
	maxScroll := float32(maxInt(0, contentHeight-viewHeight))
	scroll := w.ensureScrollSignal()
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *SkillWindow) clampScrollCount(skillCount int) {
	w.clampScrollHeight(skillCount*skillRowH, skillTableBodyH)
}

func (w *SkillWindow) ClampScroll(s *session.Session) {
	w.clampScroll(Context{Session: s})
}

func (w *SkillWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *SkillWindow) skillSnapshot(s *session.Session) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("job=%d;points=%d;pending=%v;skills=%v", s.Selected.Job, s.Skills.Points, w.pending, s.Skills.List)
}

func (w *SkillWindow) visibleSkills(ctx Context) []session.Skill {
	w.ensureSkillView(ctx)
	return w.allSkills
}

func (w *SkillWindow) ensureSkillView(ctx Context) {
	w.ensureSkillViewForSnapshot(ctx, w.skillSnapshot(ctx.Session))
}

func (w *SkillWindow) ensureSkillViewForSnapshot(ctx Context, snapshot string) {
	if w.skillViewReady && snapshot == w.skillViewKey {
		return
	}
	w.skillViewReady = true
	w.skillViewKey = snapshot
	job := db.JobNovice
	if ctx.Session != nil {
		job = int(ctx.Session.Selected.Job)
	}
	groups := db.SkillTreeSkillGroups(job)
	w.allSkills = w.buildVisibleSkills(ctx, job, groups)
	for i := range w.skillsByTab {
		w.skillsByTab[i] = w.skillsByTab[i][:0]
	}

	tabBySkill := make(map[uint16]int)
	available := [skillTabCount]bool{}
	for _, group := range groups {
		if group.ClassLevel < 1 || group.ClassLevel > 4 {
			continue
		}
		tab := group.ClassLevel - 1
		available[tab] = true
		for _, skillID := range group.SkillIDs {
			tabBySkill[skillID] = tab
		}
	}
	available[skillTabEtc] = true
	for _, skill := range w.allSkills {
		tab, ok := tabBySkill[skill.ID]
		if !ok {
			tab = skillTabEtc
		}
		w.skillsByTab[tab] = append(w.skillsByTab[tab], skill)
	}

	w.visibleTabs = w.visibleTabs[:0]
	for _, tab := range skillTabs {
		if available[tab.tab] {
			w.visibleTabs = append(w.visibleTabs, tab.tab)
		}
	}
	if !available[w.tab] {
		w.tab = w.visibleTabs[0]
		w.ensureScrollSignal().Set(0)
	}
}

func (w *SkillWindow) activeSkills() []session.Skill {
	if w.tab < 0 || w.tab >= len(w.skillsByTab) {
		return nil
	}
	return w.skillsByTab[w.tab]
}

func (w *SkillWindow) buildVisibleSkills(ctx Context, job int, groups []db.SkillTreeGroup) []session.Skill {
	sessionList := sessionSkills(ctx.Session)
	skills := append([]session.Skill(nil), sessionList...)
	if ctx.Session == nil {
		return skills
	}
	levels := make(map[uint16]int, len(sessionList))
	byID := make(map[uint16]session.Skill, len(sessionList))
	for _, skill := range sessionList {
		byID[skill.ID] = skill
		levels[skill.ID] = skill.Level + w.pendingFor(skill.ID)
	}

	ordered := make([]session.Skill, 0, len(sessionList))
	seen := make(map[uint16]bool, len(sessionList))
	for _, group := range groups {
		for _, skillID := range group.SkillIDs {
			if seen[skillID] {
				continue
			}
			if skill, ok := byID[skillID]; ok {
				ordered = append(ordered, skill)
				seen[skillID] = true
				continue
			}
			if !w.skillRequirementsMet(job, levels, skillID) {
				continue
			}
			skill := w.lockedSkill(ctx, skillID)
			ordered = append(ordered, skill)
			seen[skillID] = true
			levels[skillID] = w.pendingFor(skillID)
		}
	}
	for _, skill := range sessionList {
		if !seen[skill.ID] {
			ordered = append(ordered, skill)
		}
	}
	return ordered
}

func (w *SkillWindow) lockedSkill(ctx Context, skillID uint16) session.Skill {
	skill := session.Skill{ID: skillID, Upgradable: true}
	if ctx.Resources != nil {
		if maxLevel, ok := ctx.Resources.SkillMaxLevel(int(skillID)); ok {
			skill.MaxLevel = maxLevel
		}
	}
	return skill
}

func (w *SkillWindow) skillRequirementsMet(job int, levels map[uint16]int, skillID uint16) bool {
	requirements := db.SkillRequirementsForJob(job, skillID)
	if len(requirements) == 0 {
		return false
	}
	for _, requirement := range requirements {
		if levels[requirement.SkillID] < requirement.Level {
			return false
		}
	}
	return true
}

func skillDefaultPosition(ctx Context) (int, int) {
	return skillWindowDefaultPosition(ctx, skillWindowWidth, skillWindowHeight)
}

func skillWindowDefaultPosition(ctx Context, windowWidth, windowHeight int) (int, int) {
	width, height := ctx.ScreenSize()
	x := maxInt(windowScreenMargin, (width-windowWidth)/2)
	y := maxInt(windowScreenMargin, (height-windowHeight)/2)
	return x, y
}

func (w *SkillWindow) windowSize() (int, int) {
	if w.gridMode {
		return skillGridWindowWidth, skillGridWindowHeight
	}
	return skillWindowWidth, skillWindowHeight
}

func (w *SkillWindow) contentHeight() int {
	_, height := w.windowSize()
	return height - ROWindowTitleHeight - ROWindowFooterHeight
}

var skillTableColumns = []rotheme.TableViewColumn{
	{Key: "icon", Width: 34},
	{Key: "type", Width: 16},
	{Key: "name", Title: "Name", Width: 142},
	{Key: "leveldown", Width: 18},
	{Key: "level", Title: "Lv", Width: 40, Align: widget.TextAlignCenter},
	{Key: "levelupselect", Width: 18},
	{Key: "sp", Title: "SP", Width: 38},
	{Key: "range", Title: "Range", Width: 56},
	{Key: "levelup", Width: 22},
	{Key: "fill", Flex: 1},
}

func skillTableButtonBounds(row int, key string) geometry.Rect {
	return skillTableButtonBoundsForColumns(skillTableColumns, row, key)
}

func skillTableButtonBoundsForColumns(columns []rotheme.TableViewColumn, row int, key string) geometry.Rect {
	x := float32(0)
	for _, col := range columns {
		if col.Key == key {
			return geometry.NewRect(
				x+(col.Width-rotheme.IconButtonSize)/2,
				float32(row)*skillRowH+(skillRowH-rotheme.IconButtonSize)/2,
				rotheme.IconButtonSize,
				rotheme.IconButtonSize,
			)
		}
		x += col.Width
	}
	return geometry.Rect{}
}

func skillTableLevelUpButtonBounds(row int) geometry.Rect {
	return skillTableButtonBounds(row, "levelup")
}

func skillTypeLabel(skill session.Skill) string {
	if skill.Type == 0 {
		return "P"
	}
	return "A"
}

func skillTypeColor(skill session.Skill) widget.Color {
	if skill.Type == 0 {
		return widget.RGBA8(34, 142, 158, 255)
	}
	return widget.RGBA8(44, 92, 184, 255)
}

func sessionSkills(s *session.Session) []session.Skill {
	if s == nil {
		return nil
	}
	return s.Skills.List
}

func sessionSkillPoints(s *session.Session) int {
	if s == nil {
		return 0
	}
	return s.Skills.Points
}

func skillLabel(skill session.Skill) string {
	if strings.TrimSpace(skill.Name) != "" {
		return skill.Name
	}
	return fmt.Sprintf("Skill %d", skill.ID)
}

func skillDisplayName(manager *res.Manager, skill session.Skill) string {
	if manager != nil {
		if name, ok := manager.SkillDisplayName(int(skill.ID)); ok {
			return name
		}
	}
	return skillLabel(skill)
}

func skillTooltipText(ctx Context, skill session.Skill) string {
	name := trimRunes(skillDisplayName(ctx.Resources, skill), 38)
	lines := append([]string{name}, skillTooltipLines(ctx, skill)...)
	return strings.Join(lines, "\n")
}

func skillTooltipLines(ctx Context, skill session.Skill) []string {
	lines := []string{
		fmt.Sprintf("Lv %d", skill.Level),
	}
	if skill.SPCost > 0 {
		lines = append(lines, fmt.Sprintf("SP Cost: %d", skill.SPCost))
	}
	if skill.Range > 0 {
		lines = append(lines, fmt.Sprintf("Range: %d", skill.Range))
	}
	hasDescription := false
	if ctx.Resources != nil {
		if desc, ok := ctx.Resources.SkillDescription(int(skill.ID)); ok {
			hasDescription = true
			lines = append(lines, "")
			for _, line := range desc {
				clean := strings.TrimSpace(stripItemInfoColorCodes(strings.ReplaceAll(line, "_", " ")))
				if clean == "" {
					lines = append(lines, "")
					continue
				}
				lines = append(lines, clean)
			}
		}
	}
	if !hasDescription {
		lines = append(lines, "", "No description available.")
	}
	return wrapItemInfoLines(lines, 38)
}

func canIncreaseSkill(s *session.Session, skill session.Skill) bool {
	maxLevel := skillMaxLevel(skill)
	return s != nil && s.Skills.Points > 0 && skill.Upgradable && (maxLevel <= 0 || skill.Level < maxLevel)
}

func skillMaxLevel(skill session.Skill) int {
	if maxLevel, ok := db.SkillMaxLevel(skill.ID); ok {
		return maxLevel
	}
	if skill.MaxLevel > 0 {
		return skill.MaxLevel
	}
	return 0
}
