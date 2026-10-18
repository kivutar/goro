package ui

import (
	"image"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	autoSpellWindowWidth  = 312
	autoSpellTableHeaderH = 24
	autoSpellRowH         = 32
	autoSpellRows         = 7
	autoSpellWindowHeight = ROWindowTitleHeight + autoSpellTableHeaderH + autoSpellRows*autoSpellRowH + ROWindowFooterHeight
)

type autoSpellChoice struct {
	skillID uint16
	name    string
	icon    image.Image
}

type AutoSpellWindow struct {
	Window
	scrollY      state.Signal[float32]
	selectedRow  int
	choices      []autoSpellChoice
	lastClickAt  time.Time
	lastClickRow int
	selection    uint16
	selectionSet bool
}

func (w *AutoSpellWindow) OpenList(ctx Context, list network.AutoSpellList, assets AssetProvider) {
	w.EnsureWindow(autoSpellWindowWidth, autoSpellWindowHeight)
	w.choices = make([]autoSpellChoice, 0, len(list.SkillIDs))
	for _, skillID := range list.SkillIDs {
		skill := session.Skill{ID: skillID}
		choice := autoSpellChoice{
			skillID: skillID,
			name:    skillDisplayName(ctx.Resources, skill),
		}
		if assets != nil {
			choice.icon = assets.SkillIconImage(ctx.Resources, skill)
		}
		w.choices = append(w.choices, choice)
	}
	w.selectedRow = 0
	w.lastClickRow = -1
	w.lastClickAt = time.Time{}
	w.selection = 0
	w.selectionSet = false
	w.ensureScrollSignal().Set(0)
	if len(w.choices) == 0 {
		w.Close()
		w.Publish(ctx)
		return
	}
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *AutoSpellWindow) Update(ctx Context) bool {
	w.EnsureWindow(autoSpellWindowWidth, autoSpellWindowHeight)
	if !w.IsOpen() {
		return false
	}
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		w.cancel(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	keyboardConsumed := false
	if w.IsOpen() {
		keyboardConsumed = updateSelectionTableKeyboard(ctx, &w.selectedRow, len(w.choices), autoSpellRows, autoSpellRowH, w.ensureScrollSignal(), func() {
			w.confirm(ctx)
		})
		if keyboardConsumed && w.IsOpen() {
			w.RebindContent(ctx, w.widgetTree(ctx))
		}
	}
	if w.IsOpen() {
		w.updateDoubleClick(ctx)
	}
	w.Publish(ctx)
	return consumed || keyboardConsumed
}

func (w *AutoSpellWindow) PopSelection() (uint16, bool) {
	if !w.selectionSet {
		return 0, false
	}
	skillID := w.selection
	w.selection = 0
	w.selectionSet = false
	return skillID, true
}

func (w *AutoSpellWindow) Reset(ctx Context) {
	w.selection = 0
	w.selectionSet = false
	w.choices = nil
	if w.IsOpen() {
		w.Close()
	}
	w.Publish(ctx)
}

func (w *AutoSpellWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Auto Spell"),
		CloseButton(true),
		OnClose(func() {
			w.cancel(ctx)
		}),
		Size(autoSpellWindowWidth, autoSpellWindowHeight),
		Content(
			primitives.Box(w.tableWidget()).
				Height(autoSpellTableHeight()).
				Background(rotheme.Default.Colors.PanelBody),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Cancel", func() {
				w.cancel(ctx)
			}),
			rotheme.Button("OK", func() {
				w.confirm(ctx)
			}),
		),
	)
}

func (w *AutoSpellWindow) tableWidget() *rotheme.TableViewWidget {
	rows := make([]itemTableRow, len(w.choices))
	for i, choice := range w.choices {
		rows[i] = itemTableRow{name: choice.name, icon: choice.icon}
	}
	return itemTableView(
		rows,
		"Skill",
		autoSpellRowH,
		autoSpellTableHeaderH,
		"No spells available",
		w.ensureScrollSignal(),
		w.selectedRow,
		func(row int) {
			w.selectedRow = row
		},
	)
}

func (w *AutoSpellWindow) confirm(ctx Context) {
	if w.selectedRow < 0 || w.selectedRow >= len(w.choices) {
		return
	}
	w.selectAndClose(ctx, w.choices[w.selectedRow].skillID)
}

func (w *AutoSpellWindow) cancel(ctx Context) {
	w.selectAndClose(ctx, 0)
}

func (w *AutoSpellWindow) selectAndClose(ctx Context, skillID uint16) {
	w.selection = skillID
	w.selectionSet = true
	w.Close()
	w.Publish(ctx)
}

func (w *AutoSpellWindow) updateDoubleClick(ctx Context) {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return
	}
	row, ok := w.rowAtMouse(ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return
	}
	now := time.Now()
	if w.lastClickRow == row && now.Sub(w.lastClickAt) <= 360*time.Millisecond {
		w.selectedRow = row
		w.lastClickRow = -1
		w.lastClickAt = time.Time{}
		w.confirm(ctx)
		return
	}
	w.lastClickRow = row
	w.lastClickAt = now
}

func (w *AutoSpellWindow) rowAtMouse(mouseX, mouseY int) (int, bool) {
	tableX := w.x
	tableY := w.y + ROWindowTitleHeight
	rowY := tableY + autoSpellTableHeaderH
	if !pointInRect(mouseX, mouseY, tableX, rowY, scrollbarSafeIntWidth(autoSpellWindowWidth), autoSpellRows*autoSpellRowH) {
		return 0, false
	}
	row := int((float32(mouseY-rowY) + w.ensureScrollSignal().Get()) / autoSpellRowH)
	if row < 0 || row >= len(w.choices) {
		return 0, false
	}
	return row, true
}

func (w *AutoSpellWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func autoSpellTableHeight() float32 {
	return autoSpellTableHeaderH + autoSpellRows*autoSpellRowH
}
