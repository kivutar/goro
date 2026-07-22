package ui

import (
	"fmt"
	"image"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	homunculusSkillWindowWidth  = 360
	homunculusSkillWindowHeight = 300
	homunculusSkillTableViewH   = homunculusSkillWindowHeight - ROWindowTitleHeight - ROWindowFooterHeight
	homunculusSkillTableBodyH   = homunculusSkillTableViewH - skillHeaderH
)

type HomunculusSkillWindow struct {
	Window
	kind           companionSkillWindowKind
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
	homunculusID   uint32
}

type companionSkillWindowKind uint8

const (
	companionSkillWindowHomunculus companionSkillWindowKind = iota
	companionSkillWindowMercenary
)

type MercenarySkillWindow struct {
	HomunculusSkillWindow
}

func (w *MercenarySkillWindow) ensureMercenaryKind() {
	w.HomunculusSkillWindow.kind = companionSkillWindowMercenary
}

func (w *MercenarySkillWindow) Toggle(ctx Context, actions GameActions) {
	w.ensureMercenaryKind()
	w.HomunculusSkillWindow.Toggle(ctx, actions)
}

func (w *MercenarySkillWindow) Open(ctx Context, actions GameActions) {
	w.ensureMercenaryKind()
	w.HomunculusSkillWindow.Open(ctx, actions)
}

func (w *MercenarySkillWindow) Update(ctx Context, shortcuts *ShortcutBar, actions GameActions) bool {
	w.ensureMercenaryKind()
	return w.HomunculusSkillWindow.Update(ctx, shortcuts, actions)
}

func (w *MercenarySkillWindow) UpdateDrag(ctx Context, shortcuts *ShortcutBar) bool {
	w.ensureMercenaryKind()
	return w.HomunculusSkillWindow.UpdateDrag(ctx, shortcuts)
}

func (w *MercenarySkillWindow) Rebind(ctx Context, actions GameActions) {
	w.ensureMercenaryKind()
	w.HomunculusSkillWindow.Rebind(ctx, actions)
}

func (w *HomunculusSkillWindow) Toggle(ctx Context, actions GameActions) {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if w.IsOpen() {
		w.close(ctx)
		return
	}
	if ctx.Session == nil || !w.companionActive(ctx.Session) {
		return
	}
	w.bindActions(actions)
	w.syncCompanionIdentity(ctx)
	w.openAtDefault(ctx)
}

func (w *HomunculusSkillWindow) Open(ctx Context, actions GameActions) {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if ctx.Session == nil || !w.companionActive(ctx.Session) {
		return
	}
	w.bindActions(actions)
	w.syncCompanionIdentity(ctx)
	w.openAtDefault(ctx)
}

func (w *HomunculusSkillWindow) bindActions(actions GameActions) {
	if actions != nil {
		w.actions = actions
	}
	if assets, ok := actions.(AssetProvider); ok && assets != nil {
		w.assets = assets
		w.lastIconAssets = true
	}
}

func (w *HomunculusSkillWindow) Update(ctx Context, shortcuts *ShortcutBar, actions GameActions) bool {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if !w.IsOpen() {
		return false
	}
	w.actions = actions
	if assets, ok := actions.(AssetProvider); ok && assets != nil && !w.lastIconAssets {
		w.assets = assets
		w.lastIconAssets = true
		w.snapshot = ""
	}
	if ctx.Session == nil || !w.companionActive(ctx.Session) {
		w.close(ctx)
		return true
	}
	w.syncCompanionIdentity(ctx)
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

func (w *HomunculusSkillWindow) UpdateDrag(ctx Context, shortcuts *ShortcutBar) bool {
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

func (w *HomunculusSkillWindow) Draw(screen *render.Frame, ctx Context, assets AssetProvider) {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if !w.IsOpen() {
		w.Unpublish(ctx)
		w.hideTooltip()
		return
	}
	w.Publish(ctx)
}

func (w *HomunculusSkillWindow) DrawDragGhost(screen *render.Frame, ctx Context, assets AssetProvider) {
	if w.dragActive && screen != nil && ctx.Input != nil && time.Since(w.dragFrom) > 80*time.Millisecond && assets != nil {
		assets.DrawSkillIcon(screen, ctx.Resources, w.dragSkill, ctx.Input.MouseX-skillIconSize/2, ctx.Input.MouseY-skillIconSize/2, skillIconSize)
	}
}

func (w *HomunculusSkillWindow) DrawTooltip(screen *render.Frame) {
	w.tooltip.Draw(screen)
}

func (w *HomunculusSkillWindow) Publish(ctx Context) {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if !w.IsOpen() {
		w.Unpublish(ctx)
		w.hideTooltip()
		return
	}
	w.Window.Publish(ctx)
}

func (w *HomunculusSkillWindow) Rebind(ctx Context, actions GameActions) {
	w.EnsureWindow(homunculusSkillWindowWidth, homunculusSkillWindowHeight)
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx, actions)
}

func (w *HomunculusSkillWindow) openAtDefault(ctx Context) {
	x, y := homunculusSkillDefaultPosition(ctx)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.OpenAt(x, y, w.widgetTree(ctx, w.actions))
	w.Publish(ctx)
}

func (w *HomunculusSkillWindow) close(ctx Context) {
	w.Close()
	w.Publish(ctx)
}

func (w *HomunculusSkillWindow) Close() {
	w.dragActive = false
	w.hasHover = false
	w.hideTooltip()
	w.Window.Close()
}

func (w *HomunculusSkillWindow) widgetTree(ctx Context, actions GameActions) widget.Widget {
	return w.widgetTreeWithAssets(ctx, nil, actions)
}

func (w *HomunculusSkillWindow) widgetTreeWithAssets(ctx Context, assets AssetProvider, actions GameActions) widget.Widget {
	if actions == nil {
		actions = w.actions
	}
	if assets == nil {
		assets = w.assets
	}
	return Win(
		Title(w.title()),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
		}),
		Size(homunculusSkillWindowWidth, homunculusSkillWindowHeight),
		Content(
			primitives.Box(
				w.skillTableWidget(ctx, assets, actions),
			).
				Height(homunculusSkillTableViewH).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			footerLabel(fmt.Sprintf("Skill Points: %d", maxInt(0, w.companionSkillPoints(ctx.Session)-w.pendingCount()))),
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

func (w *HomunculusSkillWindow) skillTableWidget(ctx Context, assets AssetProvider, actions GameActions) *rotheme.TableViewWidget {
	skills := w.visibleSkills(ctx)
	return rotheme.TableView(
		rotheme.TableViewColumns(skillTableColumns),
		rotheme.TableViewRowCount(len(skills)),
		rotheme.TableViewRowHeight(skillRowH),
		rotheme.TableViewHeaderHeight(skillHeaderH),
		rotheme.TableViewEmptyText(fmt.Sprintf("No %s skills received from server yet.", w.kindLabelLower())),
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
}

func (w *HomunculusSkillWindow) skillTableCell(ctx Context, assets AssetProvider, skill session.Skill, cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
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

func (w *HomunculusSkillWindow) handleSkillTableRowEvent(widgetCtx widget.Context, ctx Context, actions GameActions, skills []session.Skill, row int, e event.Event) bool {
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
				glog.Debugf("%s skill level up ignored id=%d: no points or maxed", w.kindLabelLower(), skill.ID)
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

func (w *HomunculusSkillWindow) refresh(ctx Context, actions GameActions) {
	if actions != nil {
		w.actions = actions
	}
	w.clampScroll(ctx)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.SetContent(w.widgetTreeWithAssets(ctx, w.assets, w.actions))
	w.Publish(ctx)
}

func (w *HomunculusSkillWindow) pressSkill(ctx Context, actions GameActions, skill session.Skill, mx, my int) {
	if skill.Level <= 0 {
		glog.Debugf("%s skill use ignored id=%d: not learned", w.kindLabelLower(), skill.ID)
		return
	}
	if skill.Type == 0 {
		glog.Debugf("%s skill use ignored id=%d: passive skill", w.kindLabelLower(), skill.ID)
		return
	}
	now := time.Now()
	if w.lastClick == skill.ID && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.lastClick = 0
		w.lastClickAt = time.Time{}
		if actions == nil {
			glog.Warnf("%s skill use failed id=%d: no game actions", w.kindLabelLower(), skill.ID)
			return
		}
		if err := actions.UseShortcutSkill(ctx, skill); err != nil {
			glog.Warnf("%s skill use failed id=%d: %v", w.kindLabelLower(), skill.ID, err)
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

func (w *HomunculusSkillWindow) clearPending() {
	w.pending = nil
	w.pendingOrder = nil
}

func (w *HomunculusSkillWindow) pendingCount() int {
	total := 0
	for _, count := range w.pending {
		total += count
	}
	return total
}

func (w *HomunculusSkillWindow) pendingFor(skillID uint16) int {
	if w.pending == nil {
		return 0
	}
	return w.pending[skillID]
}

func (w *HomunculusSkillWindow) skillWithPending(skill session.Skill) session.Skill {
	skill.Level += w.pendingFor(skill.ID)
	return skill
}

func (w *HomunculusSkillWindow) stageSkill(skillID uint16) {
	if w.pending == nil {
		w.pending = make(map[uint16]int)
	}
	if w.pending[skillID] == 0 {
		w.pendingOrder = append(w.pendingOrder, skillID)
	}
	w.pending[skillID]++
}

func (w *HomunculusSkillWindow) canStageSkill(s *session.Session, skill session.Skill) bool {
	if s == nil || w.companionSkillPoints(s) <= 0 || !skill.Upgradable {
		return false
	}
	pending := w.pendingFor(skill.ID)
	if maxLevel := skillMaxLevel(skill); maxLevel > 0 {
		return skill.Level+pending < maxLevel && w.pendingCount() < w.companionSkillPoints(s)
	}
	return w.pendingCount() < w.companionSkillPoints(s)
}

func (w *HomunculusSkillWindow) confirmPending(ctx Context) {
	if len(w.pendingOrder) == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("%s skill level up failed: not connected", w.kindLabelLower())
		return
	}
	for _, skillID := range w.pendingOrder {
		for i := 0; i < w.pending[skillID]; i++ {
			if err := ctx.Network.SendSkillLevelUp(skillID); err != nil {
				glog.Warnf("%s skill level up failed id=%d: %v", w.kindLabelLower(), skillID, err)
				return
			}
		}
	}
	w.clearPending()
}

func (w *HomunculusSkillWindow) syncCompanionIdentity(ctx Context) {
	if ctx.Session == nil || !w.companionActive(ctx.Session) {
		return
	}
	id := w.companion(ctx.Session).ID
	if w.homunculusID != 0 && id != 0 && w.homunculusID != id {
		w.clearPending()
		w.snapshot = ""
	}
	if id != 0 {
		w.homunculusID = id
	}
}

func (w *HomunculusSkillWindow) syncHomunculusIdentity(ctx Context) {
	w.kind = companionSkillWindowHomunculus
	w.syncCompanionIdentity(ctx)
}

func (w *HomunculusSkillWindow) showTooltip(ctx Context, skill session.Skill, mx, my int) {
	if w.dragActive || skill.ID == 0 {
		w.hideTooltip()
		return
	}
	const tooltipW = 292
	text := skillTooltipText(ctx, skill)
	w.tooltip.ShowBox(ctx, text, mx+16+tooltipW/2, my+18, my-6, tooltipW, 24)
}

func (w *HomunculusSkillWindow) updateTooltipHover(ctx Context) {
	if !w.hasHover || ctx.Input == nil {
		return
	}
	skill, ok := w.skillAtMouse(ctx, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok || skill.ID != w.hoveredSkill.ID {
		w.hasHover = false
		w.hideTooltip()
	}
}

func (w *HomunculusSkillWindow) skillAtMouse(ctx Context, mouseX, mouseY int) (session.Skill, bool) {
	x, y := w.skillTableBodyOrigin()
	if !pointInRect(mouseX, mouseY, x, y, scrollbarSafeIntWidth(homunculusSkillWindowWidth), homunculusSkillTableBodyH) {
		return session.Skill{}, false
	}
	row := int((float32(mouseY-y) + w.ensureScrollSignal().Get()) / skillRowH)
	skills := w.visibleSkills(ctx)
	if row < 0 || row >= len(skills) {
		return session.Skill{}, false
	}
	return skills[row], true
}

func (w *HomunculusSkillWindow) skillTableBodyOrigin() (int, int) {
	return w.x, w.y + ROWindowTitleHeight + skillHeaderH
}

func (w *HomunculusSkillWindow) hideTooltip() {
	w.tooltip.Hide()
}

func (w *HomunculusSkillWindow) skillIconImage(ctx Context, assets AssetProvider, skill session.Skill) image.Image {
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

func (w *HomunculusSkillWindow) clampScroll(ctx Context) {
	w.clampScrollCount(len(w.visibleSkills(ctx)))
}

func (w *HomunculusSkillWindow) clampScrollCount(skillCount int) {
	maxScroll := float32(maxInt(0, skillCount*skillRowH-homunculusSkillTableBodyH))
	scroll := w.ensureScrollSignal()
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *HomunculusSkillWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *HomunculusSkillWindow) skillSnapshot(s *session.Session) string {
	if s == nil {
		return ""
	}
	companion := w.companion(s)
	return fmt.Sprintf("kind=%d;id=%d;active=%t;points=%d;pending=%v;skills=%v", w.kind, companion.ID, companion.Active, companion.Skills.Points, w.pending, companion.Skills.List)
}

func (w *HomunculusSkillWindow) visibleSkills(ctx Context) []session.Skill {
	return append([]session.Skill(nil), w.companionSkills(ctx.Session)...)
}

func homunculusSkillDefaultPosition(ctx Context) (int, int) {
	width, height := ctx.ScreenSize()
	x := minInt(characterWindowX+characterWindowWidth+12, maxInt(windowScreenMargin, width-homunculusSkillWindowWidth-windowScreenMargin))
	y := minInt(characterWindowY+32, maxInt(windowScreenMargin, height-homunculusSkillWindowHeight-windowScreenMargin))
	return maxInt(windowScreenMargin, x), maxInt(windowScreenMargin, y)
}

func (w *HomunculusSkillWindow) title() string {
	if w.kind == companionSkillWindowMercenary {
		return "Mercenary Skills"
	}
	return "Homunculus Skills"
}

func (w *HomunculusSkillWindow) kindLabelLower() string {
	if w.kind == companionSkillWindowMercenary {
		return "mercenary"
	}
	return "homunculus"
}

func (w *HomunculusSkillWindow) companion(s *session.Session) session.Companion {
	if s == nil {
		return session.Companion{}
	}
	if w.kind == companionSkillWindowMercenary {
		return s.Mercenary
	}
	return s.Homunculus
}

func (w *HomunculusSkillWindow) companionActive(s *session.Session) bool {
	return w.companion(s).Active
}

func (w *HomunculusSkillWindow) companionSkills(s *session.Session) []session.Skill {
	return w.companion(s).Skills.List
}

func (w *HomunculusSkillWindow) companionSkillPoints(s *session.Session) int {
	return w.companion(s).Skills.Points
}

func companionSkillByID(companion session.Companion, skillID uint16) (session.Skill, bool) {
	if skillID == 0 {
		return session.Skill{}, false
	}
	for _, skill := range companion.Skills.List {
		if skill.ID == skillID {
			return skill, true
		}
	}
	return session.Skill{}, false
}
