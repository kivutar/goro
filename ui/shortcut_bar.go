package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
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
	slots     [shortcutSlots]shortcutSlotState
	loaded    bool
	path      string
	content   widget.Widget
	root      widget.Widget
	published bool
	rootX     int
	ctx       Context
	actions   GameActions
	assets    AssetProvider
	icons     map[shortcutItemIconKey]image.Image
	iconMiss  map[shortcutItemIconKey]struct{}
}

type shortcutItemIconKey struct {
	itemID     uint16
	identified bool
}

type shortcutPersistFile struct {
	Version int                   `json:"version"`
	Slots   []shortcutPersistSlot `json:"slots"`
}

type shortcutPersistSlot struct {
	Kind       string `json:"kind,omitempty"`
	ItemIndex  uint16 `json:"item_index,omitempty"`
	ItemID     uint16 `json:"item_id,omitempty"`
	Identified bool   `json:"identified,omitempty"`
	SkillID    uint16 `json:"skill_id,omitempty"`
	SkillLevel int    `json:"skill_level,omitempty"`
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
	b.ctx = ctx
	b.actions = actions
	b.assets = assets
	b.ensureContent()
	x, _ := b.bounds(ctx)
	if b.root == nil || b.rootX != x {
		old := b.root
		b.rootX = x
		b.root = primitives.Box(b.content).
			PaddingLeft(float32(x)).
			PaddingTop(8).
			Width(float32(x + shortcutBarWidth())).
			Height(float32(8 + shortcutBarHeight()))
		if b.published && old != nil {
			ctx.UIManager.RemoveOverlay(old)
			ctx.UIManager.AddOverlay(b.root)
		}
	}
	if !b.published {
		ctx.UIManager.AddOverlay(b.root)
		b.published = true
	}
	b.redraw()
}

func (b *ShortcutBar) ResetOverlay(ctx Context) {
	if b.published && ctx.UIManager != nil && b.root != nil {
		ctx.UIManager.RemoveOverlay(b.root)
	}
	b.published = false
	b.root = nil
	b.content = nil
}

func (b *ShortcutBar) AcceptItemDrop(ctx Context, item session.InventoryItem, mx, my int) bool {
	slot, ok := b.slotAt(ctx, mx, my)
	if !ok {
		return false
	}
	b.slots[slot] = shortcutSlotState{
		kind:       shortcutItem,
		itemIndex:  item.Index,
		itemID:     item.ItemID,
		identified: item.Identified,
	}
	b.save(ctx)
	b.redraw()
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
	b.slots[slot] = shortcutSlotState{
		kind:       shortcutSkill,
		skillID:    skill.ID,
		skillLevel: skill.Level,
	}
	b.save(ctx)
	b.redraw()
	return true
}

func (b *ShortcutBar) ClearDepletedItem(ctx Context, index, itemID uint16) bool {
	changed := b.clearDepletedItemSlots(index, itemID)
	if changed {
		b.save(ctx)
	}
	return changed
}

func (b *ShortcutBar) clearDepletedItemSlots(index, itemID uint16) bool {
	if index == 0 {
		return false
	}
	changed := false
	for slot, entry := range b.slots {
		if entry.kind != shortcutItem || entry.itemIndex != index {
			continue
		}
		if itemID != 0 && entry.itemID != itemID {
			continue
		}
		b.slots[slot] = shortcutSlotState{}
		changed = true
	}
	if changed {
		b.redraw()
	}
	return changed
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
		log.Printf("shortcut item use slot=%d index=%d item=%d", slot+1, item.Index, item.ItemID)
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
	return maxInt(8, (width-shortcutBarWidth())/2), 8
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
	b.content = Window(
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
		w.hovered = true
		ctx.SetCursor(widget.CursorPointer)
		w.SetNeedsRedraw(true)
		return true
	case event.MouseLeave:
		w.hovered = false
		ctx.SetCursor(widget.CursorDefault)
		w.SetNeedsRedraw(true)
	case event.MousePress:
		switch mouse.Button {
		case event.ButtonLeft:
			w.bar.activate(w.bar.ctx, w.bar.actions, w.slot)
			return true
		case event.ButtonRight:
			w.bar.slots[w.slot] = shortcutSlotState{}
			w.bar.save(w.bar.ctx)
			w.bar.redraw()
			return true
		}
	}
	return true
}

func (b *ShortcutBar) redraw() {
	if redraw, ok := b.content.(interface{ SetNeedsRedraw(bool) }); ok {
		redraw.SetNeedsRedraw(true)
	}
	if redraw, ok := b.root.(interface{ SetNeedsRedraw(bool) }); ok {
		redraw.SetNeedsRedraw(true)
	}
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

func (b *ShortcutBar) Load(ctx Context) {
	if b.loaded {
		return
	}
	b.loaded = true
	path, legacyPath, err := shortcutStatePath(ctx.Session)
	if err != nil {
		log.Printf("shortcut load skipped: %v", err)
		return
	}
	b.path = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && legacyPath != "" && legacyPath != path {
			data, err = os.ReadFile(legacyPath)
			if err != nil {
				if !os.IsNotExist(err) {
					log.Printf("shortcut legacy load failed path=%s: %v", legacyPath, err)
				}
				return
			}
			log.Printf("shortcut bar migrating legacy path=%s target=%s", legacyPath, path)
		} else {
			if !os.IsNotExist(err) {
				log.Printf("shortcut load failed path=%s: %v", path, err)
			}
			return
		}
	}
	var saved shortcutPersistFile
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("shortcut load parse failed path=%s: %v", path, err)
		return
	}
	for i := 0; i < len(saved.Slots) && i < shortcutSlots; i++ {
		b.slots[i] = shortcutSlotFromPersist(saved.Slots[i])
	}
	log.Printf("shortcut bar loaded path=%s slots=%d", path, len(saved.Slots))
}

func (b *ShortcutBar) save(ctx Context) {
	if !b.loaded {
		b.Load(ctx)
	}
	path := b.path
	if path == "" {
		var err error
		path, _, err = shortcutStatePath(ctx.Session)
		if err != nil {
			log.Printf("shortcut save skipped: %v", err)
			return
		}
		b.path = path
		b.loaded = true
	}
	if path == "" {
		log.Printf("shortcut save skipped: no character selected")
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("shortcut save mkdir failed path=%s: %v", path, err)
		return
	}
	saved := shortcutPersistFile{
		Version: 1,
		Slots:   make([]shortcutPersistSlot, shortcutSlots),
	}
	for i := 0; i < shortcutSlots; i++ {
		saved.Slots[i] = b.slots[i].persist()
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		log.Printf("shortcut save marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		log.Printf("shortcut save failed path=%s: %v", path, err)
	}
}

func shortcutStatePath(s *session.Session) (string, string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}
	legacy := filepath.Join(dir, "goro", "shortcuts.json")
	key := shortcutCharacterKey(s)
	if key == "" {
		return legacy, legacy, nil
	}
	return filepath.Join(dir, "goro", "shortcuts", key+".json"), legacy, nil
}

func shortcutCharacterKey(s *session.Session) string {
	if s == nil {
		return ""
	}
	if s.Selected.ID != 0 {
		return fmt.Sprintf("char-%d", s.Selected.ID)
	}
	if s.CharID != 0 {
		return fmt.Sprintf("char-%d", s.CharID)
	}
	name := strings.TrimSpace(s.Selected.Name)
	if name == "" {
		return ""
	}
	sanitized := sanitizeShortcutPathPart(name)
	if sanitized == "" {
		return ""
	}
	return "name-" + sanitized
}

func sanitizeShortcutPathPart(value string) string {
	var out strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	return strings.Trim(out.String(), "_")
}

func shortcutSlotFromPersist(saved shortcutPersistSlot) shortcutSlotState {
	switch saved.Kind {
	case "item":
		return shortcutSlotState{
			kind:       shortcutItem,
			itemIndex:  saved.ItemIndex,
			itemID:     saved.ItemID,
			identified: saved.Identified,
		}
	case "skill":
		return shortcutSlotState{
			kind:       shortcutSkill,
			skillID:    saved.SkillID,
			skillLevel: saved.SkillLevel,
		}
	default:
		return shortcutSlotState{}
	}
}

func (s shortcutSlotState) persist() shortcutPersistSlot {
	switch s.kind {
	case shortcutItem:
		return shortcutPersistSlot{
			Kind:       "item",
			ItemIndex:  s.itemIndex,
			ItemID:     s.itemID,
			Identified: s.identified,
		}
	case shortcutSkill:
		return shortcutPersistSlot{
			Kind:       "skill",
			SkillID:    s.skillID,
			SkillLevel: s.skillLevel,
		}
	default:
		return shortcutPersistSlot{}
	}
}

func shortcutKey(slot int) render.Key {
	switch slot {
	case 0:
		return render.KeyF1
	case 1:
		return render.KeyF2
	case 2:
		return render.KeyF3
	case 3:
		return render.KeyF4
	case 4:
		return render.KeyF5
	case 5:
		return render.KeyF6
	case 6:
		return render.KeyF7
	case 7:
		return render.KeyF8
	default:
		return render.KeyF9
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
	skill, ok := skillByID(s, entry.skillID)
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
