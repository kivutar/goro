package ui

import (
	"fmt"
	"image/color"
	"strings"
	"unicode/utf8"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	npcDialogWidth       = 360
	npcDialogHeight      = 260
	npcDialogPad         = 12
	npcDialogButtonW     = 78
	npcDialogLineH       = 14
	npcDialogMaxMessages = 32
	npcMenuWidth         = 260
	npcMenuMinRows       = 4
	npcMenuMaxRows       = 8
	npcMenuRowH          = 24
	npcMenuPad           = 8
	npcMenuFooterH       = 32
	npcMenuMinHeight     = ROWindowTitleHeight + npcMenuPad*2 + npcMenuMinRows*npcMenuRowH + npcMenuFooterH
)

var (
	npcDialogTextColor   = TextColor
	npcDialogMutedColor  = MutedTextColor
	npcDialogOptionColor = TextColor
)

type npcDialogAction int

const (
	npcDialogActionNone npcDialogAction = iota
	npcDialogActionNext
	npcDialogActionClose
	npcDialogActionMenu
)

type NPCDialog struct {
	open        bool
	npcID       uint32
	lines       []string
	options     []string
	action      npcDialogAction
	clearOnText bool
	status      string

	dialogWindow WindowState
	menuWindow   WindowState
	dirty        bool
}

type npcDialogTextRun struct {
	text  string
	color color.RGBA
}

func (d *NPCDialog) Apply(packet network.NPCDialog) {
	switch packet.Kind {
	case network.NPCDialogSay:
		if d.clearOnText {
			d.lines = d.lines[:0]
			d.clearOnText = false
		}
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionNone
		d.options = nil
		if packet.Message != "" {
			d.lines = append(d.lines, packet.Message)
			if len(d.lines) > npcDialogMaxMessages {
				d.lines = append([]string(nil), d.lines[len(d.lines)-npcDialogMaxMessages:]...)
			}
		}
		d.dirty = true
	case network.NPCDialogNext:
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionNext
		d.options = nil
		d.dirty = true
	case network.NPCDialogClose:
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionClose
		d.options = nil
		d.dirty = true
	case network.NPCDialogMenu:
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionMenu
		d.options = append([]string(nil), packet.Options...)
		d.dirty = true
	case network.NPCDialogClear:
		if !d.open || d.npcID == 0 || packet.NPCID == 0 || d.npcID == packet.NPCID {
			d.Reset()
		}
	}
}

func (d *NPCDialog) Reset() {
	d.closeWindows()
	d.open = false
	d.npcID = 0
	d.lines = nil
	d.options = nil
	d.action = npcDialogActionNone
	d.clearOnText = false
	d.status = ""
	d.dirty = true
}

func (d *NPCDialog) ResetPublished(ctx Context) {
	d.Reset()
	d.publish(ctx)
}

func (d *NPCDialog) Update(ctx Context) bool {
	if !d.open {
		if d.dialogWindow.published != nil || d.menuWindow.published != nil {
			d.publish(ctx)
			return true
		}
		return false
	}
	if ctx.Input == nil {
		return false
	}
	if d.openWindows(ctx) {
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		if d.action == npcDialogActionMenu {
			d.choose(ctx, 255)
		} else {
			d.close(ctx)
		}
		d.publish(ctx)
		return true
	}
	if ctx.Input.JustPressed(render.KeyEnter) {
		switch d.action {
		case npcDialogActionNext:
			d.next(ctx)
		case npcDialogActionClose:
			d.close(ctx)
		}
		d.publish(ctx)
		return true
	}

	consumed := false
	if d.action == npcDialogActionMenu && d.menuWindow.Update(ctx) {
		consumed = true
	}
	if d.dialogWindow.Update(ctx) {
		consumed = true
	}
	if consumed {
		d.publish(ctx)
		return true
	}
	return false
}

func (d *NPCDialog) next(ctx Context) {
	if ctx.Network == nil {
		d.status = "not connected"
		d.dirty = true
		d.refresh(ctx)
		return
	}
	if err := ctx.Network.SendNPCNext(d.npcID); err != nil {
		d.status = err.Error()
		d.dirty = true
		d.refresh(ctx)
		return
	}
	d.action = npcDialogActionNone
	d.clearOnText = true
	d.status = ""
	d.dirty = true
	d.refresh(ctx)
}

func (d *NPCDialog) close(ctx Context) {
	if ctx.Network != nil && d.npcID != 0 {
		if err := ctx.Network.SendNPCClose(d.npcID); err != nil {
			d.status = err.Error()
			d.dirty = true
			d.refresh(ctx)
			return
		}
	}
	d.Reset()
	d.publish(ctx)
}

func (d *NPCDialog) choose(ctx Context, choice int) {
	if ctx.Network == nil {
		d.status = "not connected"
		d.dirty = true
		d.refresh(ctx)
		return
	}
	cancel := choice < 1 || choice > 254
	if choice < 1 {
		choice = 255
	}
	if choice > 255 {
		choice = 255
	}
	if err := ctx.Network.SendNPCMenuChoice(d.npcID, uint8(choice)); err != nil {
		d.status = err.Error()
		d.dirty = true
		d.refresh(ctx)
		return
	}
	if cancel || choice == 255 {
		d.Reset()
		d.publish(ctx)
		return
	}
	d.action = npcDialogActionNone
	d.options = nil
	d.status = ""
	d.dirty = true
	d.refresh(ctx)
}

func (d *NPCDialog) ensureWindows(ctx Context) {
	width, height := ctx.ScreenSize()
	x, y, w, h := npcDialogBounds(width, height)
	if d.dialogWindow.width == 0 {
		d.dialogWindow = NewWindowState(w, h)
		d.dialogWindow.SetCloseOnEscape(false)
		d.dialogWindow.OpenAt(x, y, d.dialogTree(ctx, w, h))
	} else {
		if d.dialogWindow.width != w || d.dialogWindow.height != h {
			d.dirty = true
		}
		d.dialogWindow.SetSize(w, h)
		if d.dialogWindow.SetAutoPosition(x, y) {
			d.dirty = true
		}
	}
	menuX, menuY, menuW, menuH := d.menuBounds(width, height, d.dialogWindow.x, d.dialogWindow.y, w, h)
	if d.menuWindow.width == 0 {
		d.menuWindow = NewWindowState(menuW, menuH)
		d.menuWindow.SetCloseOnEscape(false)
		d.menuWindow.SetAutoPosition(menuX, menuY)
	} else {
		if d.menuWindow.width != menuW || d.menuWindow.height != menuH {
			d.dirty = true
		}
		d.menuWindow.SetSize(menuW, menuH)
		if d.menuWindow.SetAutoPosition(menuX, menuY) {
			d.dirty = true
		}
	}
}

func (d *NPCDialog) openWindows(ctx Context) bool {
	d.ensureWindows(ctx)
	changed := d.dirty
	if !d.dialogWindow.IsOpen() {
		d.dialogWindow.OpenAt(d.dialogWindow.x, d.dialogWindow.y, d.dialogTree(ctx, d.dialogWindow.width, d.dialogWindow.height))
		changed = true
	} else if d.dirty {
		d.dialogWindow.SetContent(d.dialogTree(ctx, d.dialogWindow.width, d.dialogWindow.height))
	}
	if d.action == npcDialogActionMenu {
		if !d.menuWindow.IsOpen() {
			d.menuWindow.OpenAt(d.menuWindow.x, d.menuWindow.y, d.menuTree(ctx, d.menuWindow.width, d.menuWindow.height))
			changed = true
		} else if d.dirty {
			d.menuWindow.SetContent(d.menuTree(ctx, d.menuWindow.width, d.menuWindow.height))
		}
	} else if d.menuWindow.IsOpen() {
		d.menuWindow.Close()
		changed = true
	}
	d.dirty = false
	if changed {
		d.publish(ctx)
	}
	return changed
}

func (d *NPCDialog) closeWindows() {
	if d.dialogWindow.IsOpen() {
		d.dialogWindow.Close()
	}
	if d.menuWindow.IsOpen() {
		d.menuWindow.Close()
	}
}

func (d *NPCDialog) refresh(ctx Context) {
	if !d.open || !d.dialogWindow.IsOpen() {
		return
	}
	d.openWindows(ctx)
}

func (d *NPCDialog) publish(ctx Context) {
	if ctx.UIManager == nil {
		return
	}
	if !d.open || !d.dialogWindow.IsOpen() {
		d.dialogWindow.Unpublish(ctx)
		d.menuWindow.Unpublish(ctx)
		return
	}
	d.dialogWindow.Publish(ctx)
	if d.action != npcDialogActionMenu || !d.menuWindow.IsOpen() {
		d.menuWindow.Unpublish(ctx)
		return
	}
	d.menuWindow.Publish(ctx)
}

func (d *NPCDialog) dialogTree(ctx Context, width, height int) widget.Widget {
	contentHeight := height - ROWindowTitleHeight
	footer := widget.Widget(nil)
	if d.action == npcDialogActionNext || d.action == npcDialogActionClose {
		contentHeight -= 40
		label := "Next"
		action := d.next
		if d.action == npcDialogActionClose {
			label = "Close"
			action = d.close
		}
		footer = primitives.HBox(
			primitives.Expanded(primitives.Box()),
			rotheme.Button(label, func() {
				action(ctx)
			}).Width(npcDialogButtonW),
		).CrossAlign(primitives.CrossAxisCenter)
	}
	lines := d.dialogLineWidgets(width, contentHeight)
	if len(lines) == 0 {
		lines = append(lines, rotheme.Text(""))
	}
	if d.status != "" {
		lines = append(lines, rotheme.Text(trimRunes(d.status, 64)).Color(npcDialogWidgetColor(ErrorTextColor)))
	} else if d.action == npcDialogActionNone {
		lines = append(lines,
			primitives.Box(
				primitives.Expanded(primitives.Box()),
				rotheme.Text("Waiting...").Color(npcDialogWidgetColor(npcDialogMutedColor)),
			),
		)
	}
	options := []WindowOption{
		Title(d.title(ctx)),
		CloseButton(false),
		Size(float32(width), float32(height)),
		Content(
			primitives.Box(lines...).
				Padding(npcDialogPad).
				Gap(2),
		),
	}
	if footer != nil {
		options = append(options,
			FooterHeight(40),
			Footer(footer),
		)
	}
	return Window(options...)
}

func (d *NPCDialog) dialogLineWidgets(width, contentHeight int) []widget.Widget {
	lineMax := maxInt(12, (width-2*npcDialogPad)/7)
	maxLines := maxInt(1, (contentHeight-2*npcDialogPad)/npcDialogLineH)
	wrapped := wrapNPCDialogLines(d.lines, lineMax)
	if len(wrapped) > maxLines {
		wrapped = wrapped[len(wrapped)-maxLines:]
	}
	lines := make([]widget.Widget, 0, len(wrapped))
	for _, line := range wrapped {
		lines = append(lines, npcDialogTextLine(line))
	}
	return lines
}

func (d *NPCDialog) menuTree(ctx Context, width, height int) widget.Widget {
	rows := make([]widget.Widget, 0, maxInt(1, len(d.options)))
	for i, option := range d.options {
		choice := i + 1
		rows = append(rows,
			rotheme.Button(fmt.Sprintf("%d. %s", choice, npcDialogRunsPlainText(npcDialogTextRuns(option, npcDialogOptionColor))), func() {
				d.choose(ctx, choice)
			}).Height(npcMenuRowH),
		)
	}
	if len(rows) == 0 {
		rows = append(rows, rotheme.Text("No options.").Color(npcDialogWidgetColor(npcDialogMutedColor)))
	}
	return Window(
		Title("Choose"),
		CloseButton(false),
		Size(float32(width), float32(height)),
		Content(
			primitives.Box(
				scrollview.New(
					primitives.Box(rows...).
						Gap(2).
						CrossAlign(primitives.CrossAxisStretch),
					scrollview.DirectionOpt(scrollview.Vertical),
					scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
					scrollview.ScrollStep(npcMenuRowH),
				),
			).Padding(npcMenuPad),
		),
		FooterHeight(32),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					d.choose(ctx, 255)
				}).Width(68),
			).CrossAlign(primitives.CrossAxisCenter),
		),
	)
}

func npcDialogTextLine(runs []npcDialogTextRun) widget.Widget {
	parts := make([]widget.Widget, 0, len(runs))
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		parts = append(parts,
			rotheme.Text(run.text).
				Color(npcDialogWidgetColor(run.color)).
				LineHeight(npcDialogLineH/rotheme.Default.Typography.TextSize),
		)
	}
	if len(parts) == 0 {
		return primitives.Box().Height(npcDialogLineH)
	}
	return primitives.HBox(parts...).
		Height(npcDialogLineH).
		CrossAlign(primitives.CrossAxisCenter)
}

func npcDialogRunsPlainText(runs []npcDialogTextRun) string {
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.text)
	}
	return b.String()
}

func npcDialogWidgetColor(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}

func npcDialogBounds(width, height int) (int, int, int, int) {
	w := minInt(npcDialogWidth, maxInt(260, width-40))
	h := minInt(npcDialogHeight, maxInt(130, height-40))
	x := (width - w) / 2
	y := (height - h) / 2
	if y < 16 {
		y = 16
	}
	return x, y, w, h
}

func (d *NPCDialog) menuBounds(width, height, dialogX, dialogY, dialogW, dialogH int) (int, int, int, int) {
	w := minInt(npcMenuWidth, maxInt(220, width-40))
	rows := maxInt(npcMenuMinRows, minInt(len(d.options), npcMenuMaxRows))
	h := maxInt(npcMenuMinHeight, ROWindowTitleHeight+npcMenuPad*2+rows*npcMenuRowH+npcMenuFooterH)
	x := dialogX + (dialogW-w)/2
	y := dialogY + dialogH + 8
	x = clampWindowInt(x, 8, maxInt(8, width-w-8))
	y = clampWindowInt(y, 8, maxInt(8, height-h-8))
	return x, y, w, h
}

func (d *NPCDialog) title(ctx Context) string {
	name := ""
	if ctx.World != nil && d.npcID != 0 {
		if actor, ok := ctx.World.Actors[d.npcID]; ok {
			name = strings.TrimSpace(actor.Name)
		}
	}
	if name == "" {
		name = "NPC"
	}
	return name
}

func wrapNPCDialogLines(lines []string, maxRunes int) [][]npcDialogTextRun {
	if maxRunes < 8 {
		maxRunes = 8
	}
	var out [][]npcDialogTextRun
	for _, line := range lines {
		for _, split := range strings.Split(line, "\n") {
			out = append(out, wrapNPCDialogTextRuns(npcDialogTextRuns(split, npcDialogTextColor), maxRunes)...)
		}
	}
	return out
}

type npcDialogColoredRune struct {
	r     rune
	color color.RGBA
}

func npcDialogTextRuns(text string, base color.RGBA) []npcDialogTextRun {
	current := base
	var runs []npcDialogTextRun
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		runs = append(runs, npcDialogTextRun{text: b.String(), color: current})
		b.Reset()
	}
	for i := 0; i < len(text); {
		if c, ok := parseNPCDialogColorCode(text, i, base); ok {
			flush()
			current = c
			i += 7
			continue
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		b.WriteRune(r)
		i += size
	}
	flush()
	return runs
}

func parseNPCDialogColorCode(text string, at int, base color.RGBA) (color.RGBA, bool) {
	if at+7 > len(text) || text[at] != '^' {
		return color.RGBA{}, false
	}
	var value [6]byte
	for i := 0; i < 6; i++ {
		c := text[at+1+i]
		if !isNPCDialogHex(c) {
			return color.RGBA{}, false
		}
		value[i] = c
	}
	if strings.EqualFold(string(value[:]), "000000") {
		return base, true
	}
	return color.RGBA{
		R: npcDialogHexByte(value[0], value[1]),
		G: npcDialogHexByte(value[2], value[3]),
		B: npcDialogHexByte(value[4], value[5]),
		A: 255,
	}, true
}

func isNPCDialogHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

func npcDialogHexByte(hi, lo byte) uint8 {
	return npcDialogHexNibble(hi)<<4 | npcDialogHexNibble(lo)
}

func npcDialogHexNibble(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return 0
	}
}

func wrapNPCDialogTextRuns(runs []npcDialogTextRun, maxRunes int) [][]npcDialogTextRun {
	chars := npcDialogRunsToRunes(runs)
	if len(chars) == 0 {
		return nil
	}
	var out [][]npcDialogTextRun
	for len(chars) > maxRunes {
		breakAt := maxRunes
		for i := maxRunes - 1; i > 0; i-- {
			if chars[i].r == ' ' || chars[i].r == '\t' {
				breakAt = i
				break
			}
		}
		out = append(out, npcDialogRunesToRuns(chars[:breakAt]))
		chars = chars[breakAt:]
		for len(chars) > 0 && (chars[0].r == ' ' || chars[0].r == '\t') {
			chars = chars[1:]
		}
	}
	if len(chars) > 0 {
		out = append(out, npcDialogRunesToRuns(chars))
	}
	return out
}

func npcDialogRunsToRunes(runs []npcDialogTextRun) []npcDialogColoredRune {
	var chars []npcDialogColoredRune
	for _, run := range runs {
		for _, r := range run.text {
			chars = append(chars, npcDialogColoredRune{r: r, color: run.color})
		}
	}
	return chars
}

func npcDialogRunesToRuns(chars []npcDialogColoredRune) []npcDialogTextRun {
	if len(chars) == 0 {
		return nil
	}
	runs := []npcDialogTextRun{{color: chars[0].color}}
	var b strings.Builder
	current := chars[0].color
	for _, char := range chars {
		if char.color != current {
			runs[len(runs)-1].text = b.String()
			b.Reset()
			current = char.color
			runs = append(runs, npcDialogTextRun{color: current})
		}
		b.WriteRune(char.r)
	}
	runs[len(runs)-1].text = b.String()
	return runs
}
