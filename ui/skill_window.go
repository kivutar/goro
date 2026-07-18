package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"strings"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	skillWindowWidth  = 360
	skillWindowHeight = 388
	skillRowH         = 32
	skillIconSize     = 24
	skillHeaderH      = 24
	skillTableViewH   = skillWindowHeight - ROWindowTitleHeight - ROWindowFooterHeight
	skillTableBodyH   = skillTableViewH - skillHeaderH
)

type SkillWindow struct {
	Window
	scrollY        state.Signal[float32]
	snapshot       string
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
}

func (w *SkillWindow) Toggle(ctx Context) {
	w.EnsureWindow(skillWindowWidth, skillWindowHeight)
	if w.IsOpen() {
		w.close(ctx)
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
	w.clampScroll(ctx)
	snapshot := w.skillSnapshot(ctx.Session)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.SetContent(w.widgetTree(ctx, actions))
	}
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
	x, y := skillDefaultPosition(ctx)
	w.snapshot = w.skillSnapshot(ctx.Session)
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
	return Win(
		Title("Skill Tree"),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
		}),
		Size(skillWindowWidth, skillWindowHeight),
		Content(
			primitives.Box(
				w.skillTableWidget(ctx, assets, actions),
			).
				Height(skillTableViewH).
				Background(rotheme.Default.Colors.PanelBody),
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
	skills := w.visibleSkills(ctx)
	return rotheme.TableView(
		rotheme.TableViewColumns(skillTableColumns),
		rotheme.TableViewRowCount(len(skills)),
		rotheme.TableViewRowHeight(skillRowH),
		rotheme.TableViewHeaderHeight(skillHeaderH),
		rotheme.TableViewEmptyText("No skills received from server yet."),
		rotheme.TableViewScrollYSignal(w.ensureScrollSignal()),
		rotheme.TableViewInvalidateHover(false),
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
		return rotheme.TableViewSimpleCell{
			Text:  fmt.Sprintf("%d", display.Level),
			Color: nameColor,
		}
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
		return rotheme.TableViewIconButtonCell(rotheme.IconButtonPlus, !w.canStageSkill(ctx.Session, skill))
	default:
		return rotheme.TableViewSimpleCell{Hidden: true}
	}
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
		if skillTableLevelUpButtonBounds(row).Contains(mouse.Position) {
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
	w.clampScroll(ctx)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.SetContent(w.widgetTreeWithAssets(ctx, w.assets, w.actions))
	w.Publish(ctx)
}

func (w *SkillWindow) pressSkill(ctx Context, actions GameActions, skill session.Skill, mx, my int) {
	if skill.Level <= 0 {
		glog.Debugf("skill use ignored id=%d: not learned", skill.ID)
		return
	}
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
	x, y := w.skillTableBodyOrigin()
	if !pointInRect(mouseX, mouseY, x, y, scrollbarSafeIntWidth(skillWindowWidth), skillTableBodyH) {
		return session.Skill{}, false
	}
	row := int((float32(mouseY-y) + w.ensureScrollSignal().Get()) / skillRowH)
	skills := w.visibleSkills(ctx)
	if row < 0 || row >= len(skills) {
		return session.Skill{}, false
	}
	return skills[row], true
}

func (w *SkillWindow) skillTableBodyOrigin() (int, int) {
	return w.x, w.y + ROWindowTitleHeight + skillHeaderH
}

func (w *SkillWindow) hideTooltip() {
	w.tooltip.Hide()
}

func (w *SkillWindow) DrawTooltip(screen *render.Frame) {
	w.tooltip.Draw(screen)
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
	w.clampScrollCount(len(w.visibleSkills(ctx)))
}

func (w *SkillWindow) clampScrollCount(skillCount int) {
	maxScroll := float32(maxInt(0, skillCount*skillRowH-skillTableBodyH))
	scroll := w.ensureScrollSignal()
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *SkillWindow) ClampScroll(s *session.Session) {
	w.clampScrollCount(len(sessionSkills(s)))
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
	return fmt.Sprintf("points=%d;pending=%v;skills=%v", s.Skills.Points, w.pending, s.Skills.List)
}

func (w *SkillWindow) visibleSkills(ctx Context) []session.Skill {
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
	for _, skillID := range db.SkillTreeSkillIDs(int(ctx.Session.Selected.Job)) {
		if seen[skillID] {
			continue
		}
		if skill, ok := byID[skillID]; ok {
			ordered = append(ordered, skill)
			seen[skillID] = true
			continue
		}
		if !w.skillRequirementsMet(levels, skillID) {
			continue
		}
		skill := w.lockedSkill(ctx, skillID)
		ordered = append(ordered, skill)
		seen[skillID] = true
		levels[skillID] = w.pendingFor(skillID)
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

func (w *SkillWindow) skillRequirementsMet(levels map[uint16]int, skillID uint16) bool {
	requirements := db.SkillRequirements[skillID]
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
	width, height := ctx.ScreenSize()
	x := minInt(characterWindowX+characterWindowWidth+12, maxInt(8, width-skillWindowWidth-8))
	y := minInt(characterWindowY, maxInt(8, height-skillWindowHeight-8))
	return maxInt(8, x), maxInt(8, y)
}

var skillTableColumns = []rotheme.TableViewColumn{
	{Key: "icon", Width: 34},
	{Key: "type", Width: 16},
	{Key: "name", Title: "Name", Width: 142},
	{Key: "level", Title: "Lv", Width: 40},
	{Key: "sp", Title: "SP", Width: 38},
	{Key: "range", Title: "Range", Width: 56},
	{Key: "levelup", Width: 22},
	{Key: "fill", Flex: 1},
}

func skillTableLevelUpButtonBounds(row int) geometry.Rect {
	x := float32(0)
	for _, col := range skillTableColumns {
		if col.Key == "levelup" {
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
	if skill.MaxLevel > 0 {
		return skill.MaxLevel
	}
	if maxLevel, ok := db.SkillMaxLevel(skill.ID); ok {
		return maxLevel
	}
	return 0
}
