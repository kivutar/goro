package ui

import (
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	skillWindowWidth   = 360
	skillWindowHeight  = 382
	skillWindowPad     = 12
	skillRowH          = 32
	skillIconSize      = 24
	skillHeaderH       = 52
	skillFooterH       = 36
	skillFooterButtonW = 70
	skillListH         = skillWindowHeight - ROWindowTitleHeight - skillHeaderH - skillFooterH
)

var (
	skillWindowTextColor  = TextColor
	skillWindowMutedColor = MutedTextColor
	skillWindowGoodColor  = GoodTextColor
	skillWindowErrorColor = ErrorTextColor
)

type SkillWindow struct {
	window         WindowState
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
	pending        map[uint16]int
	pendingOrder   []uint16
	icons          map[uint16]image.Image
	iconMiss       map[uint16]struct{}
	lastIconAssets bool
	assets         AssetRenderer
	actions        GameActions
}

type skillRowWidgetConfig struct {
	skill       session.Skill
	display     session.Skill
	icon        image.Image
	name        string
	row         int
	canStage    bool
	onStage     func(session.Skill)
	onPress     func(session.Skill, int, int)
	onHover     func(session.Skill, int, int)
	onHoverExit func()
}

type skillRowWidget struct {
	widget.WidgetBase
	cfg     skillRowWidgetConfig
	hovered bool
}

func (w *SkillWindow) Toggle(ctx Context) {
	w.ensureWindow()
	if w.window.IsOpen() {
		w.close(ctx)
		return
	}
	w.openAtDefault(ctx)
}

func (w *SkillWindow) Update(ctx Context, shortcuts *ShortcutBar, actions GameActions) bool {
	w.ensureWindow()
	if !w.window.IsOpen() {
		return false
	}
	w.actions = actions
	if ctx.Input == nil {
		w.Publish(ctx)
		return true
	}
	if w.dragActive {
		if ctx.Input.MouseJustReleased(render.MouseButtonLeft) || !ctx.Input.MousePressed(render.MouseButtonLeft) {
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
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.close(ctx)
		w.Publish(ctx)
		return true
	}
	w.clampScroll(ctx.Session)
	snapshot := w.skillSnapshot(ctx.Session)
	if snapshot != w.snapshot {
		w.snapshot = snapshot
		w.window.SetContent(w.widgetTree(ctx, actions))
	}
	consumed := w.window.Update(ctx)
	if !w.window.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *SkillWindow) Draw(screen *render.Image, ctx Context, assets AssetRenderer) {
	w.ensureWindow()
	if !w.window.IsOpen() {
		w.window.Unpublish(ctx)
		return
	}
	if assets != nil {
		w.assets = assets
	}
	if w.assets != nil && !w.lastIconAssets {
		w.lastIconAssets = true
		w.snapshot = w.skillSnapshot(ctx.Session)
		w.window.SetContent(w.widgetTreeWithAssets(ctx, w.assets, w.actions))
	}
	if w.dragActive && screen != nil && ctx.Input != nil && time.Since(w.dragFrom) > 80*time.Millisecond && assets != nil {
		assets.DrawSkillIcon(screen, ctx.Resources, w.dragSkill, ctx.Input.MouseX-skillIconSize/2, ctx.Input.MouseY-skillIconSize/2, skillIconSize)
	}
	if !w.dragActive && w.hasHover {
		drawSkillTooltip(screen, ctx, w.hoveredSkill, w.hoverX, w.hoverY)
	}
	w.Publish(ctx)
}

func (w *SkillWindow) Publish(ctx Context) {
	w.ensureWindow()
	if !w.window.IsOpen() {
		w.window.Unpublish(ctx)
		return
	}
	w.window.Publish(ctx)
}

func (w *SkillWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(skillWindowWidth, skillWindowHeight)
		w.window.SetCloseOnEscape(false)
	}
}

func (w *SkillWindow) openAtDefault(ctx Context) {
	x, y := skillDefaultPosition(ctx)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.window.OpenAt(x, y, w.widgetTree(ctx, w.actions))
	w.Publish(ctx)
}

func (w *SkillWindow) close(ctx Context) {
	w.dragActive = false
	w.hasHover = false
	w.window.Close()
	w.Publish(ctx)
}

func (w *SkillWindow) widgetTree(ctx Context, actions GameActions) widget.Widget {
	return w.widgetTreeWithAssets(ctx, nil, actions)
}

func (w *SkillWindow) widgetTreeWithAssets(ctx Context, assets AssetRenderer, actions GameActions) widget.Widget {
	if actions == nil {
		actions = w.actions
	}
	if assets == nil {
		assets = w.assets
	}
	return Window(
		Title("Skill Tree"),
		CloseButton(true),
		OnClose(func() {
			w.close(ctx)
		}),
		Size(skillWindowWidth, skillWindowHeight),
		FooterHeight(skillFooterH),
		FooterPadding(10),
		Content(
			primitives.Box(
				rotheme.Text(fmt.Sprintf("Skill Points : %d", maxInt(0, sessionSkillPoints(ctx.Session)-w.pendingCount()))),
				w.skillHeader(),
				primitives.Box(
					scrollview.New(
						w.skillList(ctx, assets, actions),
						scrollview.DirectionOpt(scrollview.Vertical),
						scrollview.ScrollYSignal(w.ensureScrollSignal()),
						scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
						scrollview.ScrollStep(skillRowH),
					),
				).
					Height(skillListH),
			).
				Padding(skillWindowPad).
				Gap(5),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					w.clearPending()
					w.refresh(ctx, actions)
				}).Width(skillFooterButtonW),
				rotheme.Button("Confirm", func() {
					w.confirmPending(ctx)
					w.refresh(ctx, actions)
				}).Width(skillFooterButtonW),
			).
				Gap(8).
				CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func (w *SkillWindow) skillHeader() widget.Widget {
	return primitives.HBox(
		primitives.Box().Width(38),
		primitives.Box(rotheme.Text("Name").Color(rotheme.Default.Colors.MutedText)).Width(154),
		primitives.Box(rotheme.Text("Lv").Color(rotheme.Default.Colors.MutedText)).Width(40),
		primitives.Box(rotheme.Text("SP").Color(rotheme.Default.Colors.MutedText)).Width(38),
		primitives.Box(rotheme.Text("Range").Color(rotheme.Default.Colors.MutedText)).Width(56),
	).Height(14)
}

func (w *SkillWindow) skillList(ctx Context, assets AssetRenderer, actions GameActions) widget.Widget {
	skills := sessionSkills(ctx.Session)
	if len(skills) == 0 {
		return primitives.Box(
			rotheme.Text("No skills received from server yet.").
				Color(rotheme.Default.Colors.MutedText),
		).Padding(8)
	}
	rows := make([]widget.Widget, 0, len(skills))
	for row, skill := range skills {
		display := w.skillWithPending(skill)
		rows = append(rows, newSkillRowWidget(skillRowWidgetConfig{
			skill:    skill,
			display:  display,
			icon:     w.skillIconImage(ctx, assets, skill),
			name:     skillDisplayName(ctx.Resources, display),
			row:      row,
			canStage: w.canStageSkill(ctx.Session, skill),
			onStage: func(skill session.Skill) {
				if !w.canStageSkill(ctx.Session, skill) {
					log.Printf("skill level up ignored id=%d: no points or maxed", skill.ID)
					return
				}
				w.stageSkill(skill.ID)
				w.refresh(ctx, actions)
			},
			onPress: func(skill session.Skill, mx, my int) {
				w.pressSkill(ctx, actions, skill, mx, my)
			},
			onHover: func(skill session.Skill, mx, my int) {
				w.hoveredSkill = skill
				w.hasHover = true
				w.hoverX = mx
				w.hoverY = my
			},
			onHoverExit: func() {
				w.hasHover = false
			},
		}))
	}
	return primitives.Box(rows...)
}

func (w *SkillWindow) refresh(ctx Context, actions GameActions) {
	if actions != nil {
		w.actions = actions
	}
	w.clampScroll(ctx.Session)
	w.snapshot = w.skillSnapshot(ctx.Session)
	w.window.SetContent(w.widgetTreeWithAssets(ctx, w.assets, w.actions))
	w.Publish(ctx)
}

func (w *SkillWindow) pressSkill(ctx Context, actions GameActions, skill session.Skill, mx, my int) {
	if skill.Level <= 0 {
		log.Printf("skill use ignored id=%d: not learned", skill.ID)
		return
	}
	now := time.Now()
	if w.lastClick == skill.ID && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.lastClick = 0
		w.lastClickAt = time.Time{}
		if actions == nil {
			log.Printf("skill use failed id=%d: no game actions", skill.ID)
			return
		}
		if err := actions.UseShortcutSkill(ctx, skill); err != nil {
			log.Printf("skill use failed id=%d: %v", skill.ID, err)
		}
		return
	}
	w.lastClick = skill.ID
	w.lastClickAt = now
	w.dragSkill = skill
	w.dragActive = true
	w.dragFrom = now
	w.hoverX = mx
	w.hoverY = my
}

func (w *SkillWindow) skillIconImage(ctx Context, assets AssetRenderer, skill session.Skill) image.Image {
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
	return w.pendingCount() < sessionSkillPoints(s)
}

func (w *SkillWindow) confirmPending(ctx Context) {
	if len(w.pendingOrder) == 0 {
		return
	}
	if ctx.Network == nil {
		log.Printf("skill level up failed: not connected")
		return
	}
	for _, skillID := range w.pendingOrder {
		for i := 0; i < w.pending[skillID]; i++ {
			if err := ctx.Network.SendSkillLevelUp(skillID); err != nil {
				log.Printf("skill level up failed id=%d: %v", skillID, err)
				return
			}
		}
	}
	w.clearPending()
}

func (w *SkillWindow) clampScroll(s *session.Session) {
	maxScroll := float32(maxInt(0, len(sessionSkills(s))*skillRowH-skillListH))
	scroll := w.ensureScrollSignal()
	switch value := scroll.Get(); {
	case value < 0:
		scroll.Set(0)
	case value > maxScroll:
		scroll.Set(maxScroll)
	}
}

func (w *SkillWindow) ClampScroll(s *session.Session) {
	w.clampScroll(s)
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

func skillDefaultPosition(ctx Context) (int, int) {
	width, height := ctx.ScreenSize()
	x := minInt(characterWindowX+characterWindowWidth+12, maxInt(8, width-skillWindowWidth-8))
	y := minInt(characterWindowY, maxInt(8, height-skillWindowHeight-8))
	return maxInt(8, x), maxInt(8, y)
}

func newSkillRowWidget(cfg skillRowWidgetConfig) *skillRowWidget {
	w := &skillRowWidget{cfg: cfg}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *skillRowWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(skillWindowWidth-skillWindowPad*2, skillRowH))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *skillRowWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	fill := widget.RGBA8(246, 249, 253, 255)
	if w.cfg.row%2 == 1 {
		fill = rotheme.Default.Colors.PanelBody
	}
	if w.hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(geometry.NewRect(bounds.Min.X, bounds.Min.Y, bounds.Width(), bounds.Height()-2), fill)
	if w.cfg.icon != nil {
		iconBounds := w.cfg.icon.Bounds()
		iconW := float32(iconBounds.Dx())
		iconH := float32(iconBounds.Dy())
		canvas.DrawImage(w.cfg.icon, geometry.Pt(bounds.Min.X+4+(skillIconSize-iconW)/2, bounds.Min.Y+4+(skillIconSize-iconH)/2))
	} else {
		icon := geometry.NewRect(bounds.Min.X+4, bounds.Min.Y+4, skillIconSize, skillIconSize)
		canvas.DrawRect(icon, widget.RGBA8(54, 62, 80, 235))
		canvas.DrawRect(geometry.NewRect(icon.Min.X+2, icon.Min.Y+2, icon.Width()-4, icon.Height()-4), widget.RGBA8(92, 110, 150, 220))
		rotheme.DrawText(canvas, "S", icon, rotheme.Default.Typography.TextSize, widget.RGBA8(238, 238, 245, 255), false, widget.TextAlignCenter)
	}
	typeColor := widget.RGBA8(34, 142, 158, 255)
	typeLabel := "P"
	if w.cfg.display.Type != 0 {
		typeColor = widget.RGBA8(44, 92, 184, 255)
		typeLabel = "A"
	}
	nameColor := rotheme.Default.Colors.Text
	if w.cfg.display.Level <= 0 {
		nameColor = rotheme.Default.Colors.MutedText
	}
	rotheme.DrawText(canvas, typeLabel, geometry.NewRect(bounds.Min.X+34, bounds.Min.Y+8, 12, 14), rotheme.Default.Typography.TextSize, typeColor, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, trimRunes(w.cfg.name, 18), geometry.NewRect(bounds.Min.X+50, bounds.Min.Y+8, 142, 14), rotheme.Default.Typography.TextSize, nameColor, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, fmt.Sprintf("%d", w.cfg.display.Level), geometry.NewRect(bounds.Min.X+192, bounds.Min.Y+8, 34, 14), rotheme.Default.Typography.TextSize, nameColor, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, fmt.Sprintf("%d", w.cfg.display.SPCost), geometry.NewRect(bounds.Min.X+232, bounds.Min.Y+8, 36, 14), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, fmt.Sprintf("%d", w.cfg.display.Range), geometry.NewRect(bounds.Min.X+282, bounds.Min.Y+8, 34, 14), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignLeft)
	rotheme.DrawIconButton(canvas, w.plusBounds(), rotheme.IconButtonPlus, w.hovered, !w.cfg.canStage)
}

func (w *skillRowWidget) Event(ctx widget.Context, e event.Event) bool {
	ev, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		w.hovered = true
		if w.cfg.display.Level > 0 || w.cfg.canStage {
			ctx.SetCursor(widget.CursorPointer)
		}
		if w.cfg.onHover != nil {
			w.cfg.onHover(w.cfg.skill, int(ev.GlobalPosition.X), int(ev.GlobalPosition.Y))
		}
		w.SetNeedsRedraw(true)
		return false
	case event.MouseLeave:
		w.hovered = false
		if w.cfg.onHoverExit != nil {
			w.cfg.onHoverExit()
		}
		w.SetNeedsRedraw(true)
		return false
	case event.MousePress:
		if ev.Button != event.ButtonLeft {
			return false
		}
		if w.plusBounds().Contains(ev.Position) {
			if w.cfg.onStage != nil && w.cfg.canStage {
				w.cfg.onStage(w.cfg.skill)
			}
			return true
		}
		if w.cfg.onPress != nil {
			w.cfg.onPress(w.cfg.skill, int(ev.GlobalPosition.X), int(ev.GlobalPosition.Y))
		}
		return true
	}
	return false
}

func (w *skillRowWidget) Children() []widget.Widget {
	return nil
}

func (w *skillRowWidget) plusBounds() geometry.Rect {
	bounds := w.Bounds()
	return geometry.NewRect(bounds.Max.X-rotheme.IconButtonSize-7, bounds.Min.Y+7, rotheme.IconButtonSize, rotheme.IconButtonSize)
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

func drawSkillTooltip(screen *render.Image, ctx Context, skill session.Skill, mouseX, mouseY int) {
	if screen == nil {
		return
	}
	const tooltipW = 292
	name := skillDisplayName(ctx.Resources, skill)
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
	wrapped := wrapItemInfoLines(lines, 38)
	tooltipH := 12 + itemInfoLineH*(len(wrapped)+1)
	x := mouseX + 16
	y := mouseY + 18
	screenW, screenH := ctx.ScreenSize()
	x = clampInventoryWindowInt(x, 8, maxInt(8, screenW-tooltipW-8))
	y = clampInventoryWindowInt(y, 8, maxInt(8, screenH-tooltipH-8))
	DrawSurface(screen, x, y, tooltipW, tooltipH, PanelBodyColor, WindowBorderColor)
	render.DebugPrintAtColor(screen, trimRunes(name, 38), x+7, y+6, skillWindowTextColor)
	lineY := y + 6 + itemInfoLineH
	for i, line := range wrapped {
		color := skillWindowMutedColor
		if i >= 4 {
			color = skillWindowTextColor
		}
		render.DebugPrintAtColor(screen, trimRunes(line, 38), x+7, lineY+i*itemInfoLineH, color)
	}
}

func canIncreaseSkill(s *session.Session, skill session.Skill) bool {
	return s != nil && s.Skills.Points > 0 && skill.Upgradable
}
