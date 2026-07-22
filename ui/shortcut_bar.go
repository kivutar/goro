package ui

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	shortcutSlots    = 9
	shortcutSlot     = 34
	shortcutGap      = 2
	shortcutPad      = 3
	shortcutLabelH   = 12
	shortcutLabelGap = 1
)

type shortcutKind int

const (
	shortcutEmpty shortcutKind = iota
	shortcutItem
	shortcutSkill
)

type shortcutSlotState struct {
	kind       shortcutKind
	itemIndex  uint16
	itemID     uint16
	identified bool
	skillID    uint16
	skillLevel int
}

type ShortcutBar struct {
	slots         [shortcutSlots]shortcutSlotState
	hotkeyVersion int
	content       widget.Widget
	root          widget.Widget
	published     bool
	rootX         int
	ctx           Context
	actions       GameActions
	assets        AssetProvider
	icons         map[shortcutItemIconKey]image.Image
	iconMiss      map[shortcutItemIconKey]struct{}
	tooltip       tooltipState
}

type shortcutItemIconKey struct {
	itemID     uint16
	identified bool
}

func (b *ShortcutBar) Update(ctx Context, actions GameActions) bool {
	if ctx.Input == nil {
		return false
	}
	assets, _ := actions.(AssetProvider)
	b.Publish(ctx, actions, assets)
	for i := 0; i < shortcutSlots; i++ {
		if ctx.Input.JustPressed(shortcutKey(i)) {
			b.activate(ctx, actions, i)
			b.redraw()
			return true
		}
	}
	return b.pointInside(ctx, ctx.Input.MouseX, ctx.Input.MouseY)
}

func (b *ShortcutBar) Publish(ctx Context, actions GameActions, assets AssetProvider) {
	if ctx.UIManager == nil {
		return
	}
	b.SyncFromSession(ctx)
	b.ctx = ctx
	b.actions = actions
	b.assets = assets
	b.ensureContent()
	x, _ := b.bounds(ctx)
	if b.root == nil || b.rootX != x {
		old := b.root
		b.rootX = x
		b.root = positionedWidget(b.content, x, 8, shortcutBarWidth(), shortcutBarHeight())
		if b.published && old != nil {
			ctx.UIManager.RemoveOverlay(old)
			ctx.UIManager.AddOverlay(b.root)
		}
	}
	if !b.published {
		ctx.UIManager.AddOverlay(b.root)
		b.published = true
	}
}

func (b *ShortcutBar) ResetOverlay(ctx Context) {
	if b.published && ctx.UIManager != nil && b.root != nil {
		ctx.UIManager.RemoveOverlay(b.root)
	}
	b.hideTooltip()
	b.published = false
	b.root = nil
	b.content = nil
}

func (b *ShortcutBar) DrawTooltip(screen *render.Frame) {
	b.tooltip.Draw(screen)
}

func (b *ShortcutBar) AcceptItemDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	slot, ok := b.slotAt(ctx, mx, my)
	if !ok {
		return false
	}
	b.ctx = ctx
	b.slots[slot] = shortcutSlotState{
		kind:       shortcutItem,
		itemIndex:  item.Index,
		itemID:     item.ItemID,
		identified: item.Identified,
	}
	b.sendSlotChange(ctx, slot)
	b.redraw()
	b.invalidate(ctx)
	return true
}

func (b *ShortcutBar) AcceptSkillDrop(ctx Context, skill session.Skill, mx, my int) bool {
	slot, ok := b.slotAt(ctx, mx, my)
	if !ok {
		return false
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return true
	}
	b.ctx = ctx
	b.slots[slot] = shortcutSlotState{
		kind:       shortcutSkill,
		skillID:    skill.ID,
		skillLevel: skill.Level,
	}
	b.sendSlotChange(ctx, slot)
	b.redraw()
	b.invalidate(ctx)
	return true
}

func (b *ShortcutBar) ClearDepletedItem(ctx Context, index, itemID uint16) bool {
	b.redraw()
	return false
}

func (b *ShortcutBar) activate(ctx Context, actions GameActions, slot int) {
	if slot < 0 || slot >= len(b.slots) {
		return
	}
	entry := b.slots[slot]
	switch entry.kind {
	case shortcutItem:
		item, ok := inventoryItemForShortcut(ctx.Session, entry.itemIndex, entry.itemID)
		if !ok {
			return
		}
		if err := useInventoryItem(ctx, item); err != nil {
			return
		}
		glog.Debugf("shortcut item use slot=%d index=%d item=%d", slot+1, item.Index, item.ItemID)
	case shortcutSkill:
		skill, ok := skillForShortcut(ctx.Session, entry)
		if !ok {
			return
		}
		if actions == nil {
			return
		}
		if err := actions.UseShortcutSkill(ctx, skill); err != nil {
			return
		}
	default:
	}
	b.redraw()
}

func (b *ShortcutBar) slotAt(ctx Context, mx, my int) (int, bool) {
	for i := 0; i < shortcutSlots; i++ {
		x, y := b.slotBounds(ctx, i)
		if pointInRect(mx, my, x, y, shortcutSlot, shortcutSlot) {
			return i, true
		}
	}
	return 0, false
}

func (b *ShortcutBar) pointInside(ctx Context, mx, my int) bool {
	x, y := b.bounds(ctx)
	return pointInRect(mx, my, x, y, shortcutBarWidth(), shortcutBarHeight())
}

func (b *ShortcutBar) bounds(ctx Context) (int, int) {
	width, _ := ctx.ScreenSize()
	return maxInt(windowScreenMargin, (width-shortcutBarWidth())/2), windowScreenMargin
}

func (b *ShortcutBar) slotBounds(ctx Context, slot int) (int, int) {
	x, y := b.bounds(ctx)
	return x + shortcutPad + slot*(shortcutSlot+shortcutGap), y + shortcutPad
}

func shortcutBarWidth() int {
	return shortcutSlots*shortcutSlot + (shortcutSlots-1)*shortcutGap + shortcutPad*2
}

func shortcutBarHeight() int {
	return shortcutPad*2 + shortcutSlot + shortcutLabelGap + shortcutLabelH
}

func (b *ShortcutBar) ensureContent() {
	if b.content != nil {
		return
	}
	columns := make([]widget.Widget, 0, shortcutSlots)
	for i := 0; i < shortcutSlots; i++ {
		columns = append(columns, b.slotColumn(i))
	}
	b.content = Win(
		TitleBar(false),
		Radius(0),
		Size(float32(shortcutBarWidth()), float32(shortcutBarHeight())),
		Content(
			primitives.Box(
				primitives.HBox(columns...).
					Gap(shortcutGap).
					CrossAlign(primitives.CrossAxisStart),
			).
				Padding(shortcutPad).
				CrossAlign(primitives.CrossAxisStretch),
		),
	)
}

func (b *ShortcutBar) slotColumn(slot int) widget.Widget {
	return primitives.Box(
		newShortcutSlotButton(b, slot),
		shortcutKeyLabel(slot),
	).
		Width(shortcutSlot).
		Gap(shortcutLabelGap).
		CrossAlign(primitives.CrossAxisStretch)
}

func shortcutKeyLabel(slot int) widget.Widget {
	return primitives.Box(
		primitives.Expanded(primitives.Box()),
		rotheme.Text(fmt.Sprintf("F%d", slot+1)).
			Color(rotheme.Default.Colors.MutedText).
			Align(widget.TextAlignCenter).
			LineHeight(shortcutLabelH),
		primitives.Expanded(primitives.Box()),
	).
		Height(shortcutLabelH).
		CrossAlign(primitives.CrossAxisStretch)
}

type shortcutSlotButton struct {
	widget.WidgetBase
	bar     *ShortcutBar
	slot    int
	hovered bool
}

func newShortcutSlotButton(bar *ShortcutBar, slot int) *shortcutSlotButton {
	w := &shortcutSlotButton{bar: bar, slot: slot}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *shortcutSlotButton) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(shortcutSlot, shortcutSlot))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *shortcutSlotButton) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.bar == nil {
		return
	}
	bounds := w.Bounds()
	fill := rotheme.Default.Colors.Button
	if w.hovered {
		fill = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(bounds, fill)
	canvas.StrokeRect(bounds, rotheme.Default.Colors.ButtonBorder, 1)
	w.drawContent(canvas, bounds)
}

func (w *shortcutSlotButton) drawContent(canvas widget.Canvas, bounds geometry.Rect) {
	entry := w.bar.slots[w.slot]
	switch entry.kind {
	case shortcutItem:
		item := session.InventoryItem{ItemID: entry.itemID, Index: entry.itemIndex, Identified: entry.identified, Amount: 1}
		if live, ok := inventoryItemForShortcut(w.bar.ctx.Session, entry.itemIndex, entry.itemID); ok {
			item = live
		}
		if icon := w.bar.itemIconImage(w.bar.ctx.Resources, item); icon != nil {
			canvas.DrawImage(icon, geometry.Pt(bounds.Min.X+5, bounds.Min.Y+5))
		}
		if item.Amount > 1 {
			rotheme.DrawText(
				canvas,
				fmt.Sprintf("%d", item.Amount),
				geometry.NewRect(bounds.Min.X+1, bounds.Max.Y-15, shortcutSlot-3, 12),
				rotheme.Default.Typography.TextSize,
				rotheme.Default.Colors.Text,
				false,
				widget.TextAlignRight,
			)
		}
	case shortcutSkill:
		skill, _ := skillForShortcut(w.bar.ctx.Session, entry)
		if skill.ID == 0 {
			skill = session.Skill{ID: entry.skillID, Level: entry.skillLevel}
		}
		if w.bar.assets != nil {
			if icon := w.bar.assets.SkillIconImage(w.bar.ctx.Resources, skill, 24); icon != nil {
				canvas.DrawImage(icon, geometry.Pt(bounds.Min.X+5, bounds.Min.Y+5))
			}
		}
		if skill.Level > 0 {
			rotheme.DrawText(
				canvas,
				fmt.Sprintf("Lv%d", maxInt(1, skill.Level)),
				geometry.NewRect(bounds.Min.X+3, bounds.Min.Y+1, shortcutSlot-6, 12),
				rotheme.Default.Typography.TextSize,
				Color(TitleTextColor),
				false,
				widget.TextAlignLeft,
			)
		}
	}
}

func (w *shortcutSlotButton) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok || w.bar == nil {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		if !w.hovered {
			w.hovered = true
			w.bar.redraw()
		}
		ctx.SetCursor(widget.CursorPointer)
		w.bar.showTooltip(w.slot)
		return true
	case event.MouseLeave:
		if w.hovered {
			w.hovered = false
			w.bar.redraw()
		}
		ctx.SetCursor(widget.CursorDefault)
		w.bar.hideTooltip()
	case event.MousePress:
		switch mouse.Button {
		case event.ButtonLeft:
			w.bar.activate(w.bar.ctx, w.bar.actions, w.slot)
			return true
		case event.ButtonRight:
			w.bar.slots[w.slot] = shortcutSlotState{}
			w.bar.sendSlotChange(w.bar.ctx, w.slot)
			w.bar.hideTooltip()
			w.bar.redraw()
			w.bar.invalidate(w.bar.ctx)
			return true
		}
	}
	return true
}

func (b *ShortcutBar) showTooltip(slot int) {
	if slot < 0 || slot >= shortcutSlots {
		return
	}
	text := b.tooltipText(slot)
	if text == "" {
		b.hideTooltip()
		return
	}
	barX, barY := b.bounds(b.ctx)
	b.tooltip.Show(b.ctx, text, barX+shortcutBarWidth()/2, barY+shortcutBarHeight()+2, barY-2)
}

func (b *ShortcutBar) tooltipText(slot int) string {
	if slot < 0 || slot >= shortcutSlots {
		return ""
	}
	entry := b.slots[slot]
	if entry.kind != shortcutSkill {
		return ""
	}
	skill, ok := skillForShortcut(b.ctx.Session, entry)
	if !ok {
		skill = session.Skill{ID: entry.skillID, Level: entry.skillLevel}
	}
	if skill.ID == 0 {
		return ""
	}
	name := trimRunes(skillDisplayName(b.ctx.Resources, skill), 38)
	if name == "" {
		return ""
	}
	return fmt.Sprintf("[ F%d ] %s", slot+1, name)
}

func (b *ShortcutBar) hideTooltip() {
	b.tooltip.Hide()
}

func (b *ShortcutBar) redraw() {
	if redraw, ok := b.content.(interface{ SetNeedsRedraw(bool) }); ok {
		redraw.SetNeedsRedraw(true)
	}
	if redraw, ok := b.root.(interface{ SetNeedsRedraw(bool) }); ok {
		redraw.SetNeedsRedraw(true)
	}
}

func (b *ShortcutBar) invalidate(ctx Context) {
	if ctx.UIApp == nil {
		return
	}
	ctx.UIApp.Invalidate()
}

func (b *ShortcutBar) itemIconImage(manager *res.Manager, item session.InventoryItem) image.Image {
	if manager == nil || item.ItemID == 0 {
		return nil
	}
	key := shortcutItemIconKey{itemID: item.ItemID, identified: item.Identified}
	if b.icons != nil {
		if img := b.icons[key]; img != nil {
			return img
		}
	}
	if _, ok := b.iconMiss[key]; ok {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(item.ItemID), item.Identified)
	if !ok {
		b.markIconMiss(key)
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		b.markIconMiss(key)
		return nil
	}
	if b.icons == nil {
		b.icons = make(map[shortcutItemIconKey]image.Image)
	}
	b.icons[key] = img
	return img
}

func (b *ShortcutBar) markIconMiss(key shortcutItemIconKey) {
	if b.iconMiss == nil {
		b.iconMiss = make(map[shortcutItemIconKey]struct{})
	}
	b.iconMiss[key] = struct{}{}
}

func (b *ShortcutBar) SyncFromSession(ctx Context) {
	if ctx.Session == nil || !ctx.Session.Hotkeys.Loaded || b.hotkeyVersion == ctx.Session.Hotkeys.Version {
		return
	}
	for i := range b.slots {
		b.slots[i] = shortcutSlotState{}
		if i < len(ctx.Session.Hotkeys.Slots) {
			b.slots[i] = shortcutSlotFromHotkey(ctx.Session.Hotkeys.Slots[i])
		}
	}
	b.hotkeyVersion = ctx.Session.Hotkeys.Version
	b.redraw()
	b.invalidate(ctx)
}

func shortcutSlotFromHotkey(h session.HotkeySlot) shortcutSlotState {
	switch h.Type {
	case network.HotkeyTypeItem:
		if h.ID == 0 {
			return shortcutSlotState{}
		}
		return shortcutSlotState{
			kind:       shortcutItem,
			itemID:     uint16(h.ID),
			identified: true,
		}
	case network.HotkeyTypeSkill:
		if h.ID == 0 {
			return shortcutSlotState{}
		}
		return shortcutSlotState{
			kind:       shortcutSkill,
			skillID:    uint16(h.ID),
			skillLevel: int(h.Level),
		}
	default:
		return shortcutSlotState{}
	}
}

func (b *ShortcutBar) sendSlotChange(ctx Context, slot int) {
	if slot < 0 || slot >= shortcutSlots {
		return
	}
	hotkey := b.slots[slot].hotkey()
	if ctx.Network != nil {
		if err := ctx.Network.SendHotkey(uint16(slot), hotkey); err != nil {
			glog.Warnf("shortcut hotkey save failed slot=%d: %v", slot+1, err)
		}
	}
	if ctx.Session != nil {
		setSessionHotkey(ctx.Session, slot, hotkey)
		b.hotkeyVersion = ctx.Session.Hotkeys.Version
	}
}

func setSessionHotkey(s *session.Session, slot int, hotkey network.HotkeySlot) {
	if s == nil || slot < 0 {
		return
	}
	if len(s.Hotkeys.Slots) <= slot {
		next := make([]session.HotkeySlot, slot+1)
		copy(next, s.Hotkeys.Slots)
		s.Hotkeys.Slots = next
	}
	s.Hotkeys.Slots[slot] = session.HotkeySlot{Type: hotkey.Type, ID: hotkey.ID, Level: hotkey.Level}
	s.Hotkeys.Loaded = true
	s.Hotkeys.Version++
}

func (s shortcutSlotState) hotkey() network.HotkeySlot {
	switch s.kind {
	case shortcutItem:
		return network.HotkeySlot{Type: network.HotkeyTypeItem, ID: uint32(s.itemID)}
	case shortcutSkill:
		return network.HotkeySlot{Type: network.HotkeyTypeSkill, ID: uint32(s.skillID), Level: uint16(maxInt(0, s.skillLevel))}
	default:
		return network.HotkeySlot{}
	}
}

func shortcutKey(slot int) input.Key {
	switch slot {
	case 0:
		return input.KeyF1
	case 1:
		return input.KeyF2
	case 2:
		return input.KeyF3
	case 3:
		return input.KeyF4
	case 4:
		return input.KeyF5
	case 5:
		return input.KeyF6
	case 6:
		return input.KeyF7
	case 7:
		return input.KeyF8
	default:
		return input.KeyF9
	}
}

func inventoryItemByIndex(s *session.Session, index uint16) (session.InventoryItem, bool) {
	if s == nil {
		return session.InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if item.Index != index || item.Amount == 0 {
			continue
		}
		return item, true
	}
	return session.InventoryItem{}, false
}

func inventoryItemForShortcut(s *session.Session, index, itemID uint16) (session.InventoryItem, bool) {
	if item, ok := inventoryItemByIndex(s, index); ok {
		if itemID == 0 || item.ItemID == itemID {
			return item, true
		}
	}
	if s == nil || itemID == 0 {
		return session.InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if item.ItemID != itemID || item.Amount == 0 {
			continue
		}
		return item, true
	}
	return session.InventoryItem{}, false
}

func skillForShortcut(s *session.Session, entry shortcutSlotState) (session.Skill, bool) {
	if entry.kind != shortcutSkill {
		return session.Skill{}, false
	}
	skill, ok := shortcutSkillByID(s, entry.skillID)
	if !ok || skill.Level <= 0 {
		return session.Skill{}, false
	}
	level := entry.skillLevel
	if level <= 0 || level > skill.Level {
		level = skill.Level
	}
	skill.Level = level
	return skill, true
}

func shortcutSkillByID(s *session.Session, skillID uint16) (session.Skill, bool) {
	if s == nil {
		return session.Skill{}, false
	}
	if s.Mercenary.Active {
		if skill, ok := companionSkillByID(s.Mercenary, skillID); ok {
			return skill, true
		}
	}
	if s.Homunculus.Active {
		if skill, ok := companionSkillByID(s.Homunculus, skillID); ok {
			return skill, true
		}
	}
	if skill, ok := skillByID(s, skillID); ok {
		return skill, true
	}
	return session.Skill{}, false
}
