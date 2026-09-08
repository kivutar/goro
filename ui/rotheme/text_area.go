package rotheme

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

const textAreaLineHeight = float32(18)

// TextAreaWidget is a small, wrapped multiline editor. It uses the same font,
// focus, clipboard and pointer capture services as the other UI inputs.
// Positions are rune offsets; maxBytes bounds the encoded text, not its glyphs.
type TextAreaWidget struct {
	widget.WidgetBase
	text         string
	maxBytes     int
	onChange     func(string)
	readOnly     bool
	cursor       int
	anchor       int
	lineHint     int // resolves caret affinity at a soft-wrap boundary
	scroll       int
	dragging     bool
	lines        []textAreaLine
	lineText     string
	lineWidth    float32
	revealCursor bool
}

type textAreaLine struct {
	start, end int
	stops      []float32
}

func TextArea(value string, maxBytes int, onChange func(string)) *TextAreaWidget {
	w := &TextAreaWidget{maxBytes: maxBytes}
	w.replaceSelection(value)
	w.cursor, w.anchor, w.revealCursor = 0, 0, false
	w.onChange = onChange
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *TextAreaWidget) Text() string              { return w.text }
func (w *TextAreaWidget) SetReadOnly(readOnly bool) { w.readOnly = readOnly }
func (w *TextAreaWidget) IsFocusable() bool         { return w.IsEnabled() }

func (w *TextAreaWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(300, 108))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *TextAreaWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	border := Default.Colors.FooterLine
	if w.IsFocused() {
		border = Default.Colors.InputFocus
	}
	canvas.DrawRect(bounds, Default.Colors.PanelBody)
	if !w.readOnly {
		drawTextFieldInsetShadow(canvas, bounds)
	}
	canvas.StrokeRect(bounds, border, 1)
	content := bounds.Inset(geometry.UniformInsets(6))
	w.wrap(canvas, content.Width())
	w.clampScroll()
	if w.revealCursor {
		w.keepCursorVisible()
		w.revealCursor = false
	}
	canvas.PushClip(content)
	defer canvas.PopClip()
	runes := []rune(w.text)
	start, end := w.selection()
	for i := w.scroll; i < len(w.lines); i++ {
		line := w.lines[i]
		y := content.Min.Y + float32(i-w.scroll)*textAreaLineHeight
		if y >= content.Max.Y {
			break
		}
		if w.IsFocused() && start != end {
			a, b := max(start, line.start), min(end, line.end)
			if a < b {
				canvas.DrawRect(geometry.NewRect(content.Min.X+line.stops[a-line.start], y, line.stops[b-line.start]-line.stops[a-line.start], textAreaLineHeight), widget.RGBA8(121, 169, 215, 110))
			}
		}
		DrawText(canvas, string(runes[line.start:line.end]), geometry.NewRect(content.Min.X, y, content.Width(), textAreaLineHeight), Default.Typography.TextSize, Default.Colors.Text, false, widget.TextAlignLeft)
	}
	if w.IsFocused() && w.IsEnabled() && start == end {
		row := w.cursorLine()
		line := w.lines[row]
		x := content.Min.X + line.stops[min(w.cursor-line.start, len(line.stops)-1)]
		y := content.Min.Y + float32(row-w.scroll)*textAreaLineHeight
		canvas.DrawLine(geometry.Pt(x, y+2), geometry.Pt(x, y+textAreaLineHeight-2), Default.Colors.Text, 1)
	}
}

func (w *TextAreaWidget) wrap(canvas widget.Canvas, width float32) {
	if len(w.lines) > 0 && w.lineText == w.text && w.lineWidth == width {
		return
	}
	w.lineText, w.lineWidth = w.text, width
	w.lineHint = -1
	w.lines = nil
	runes := []rune(w.text)
	for start := 0; ; {
		end, space := start, -1
		for end < len(runes) && runes[end] != '\n' {
			if end > start && MeasureText(canvas, string(runes[start:end+1]), Default.Typography.TextSize, false) > width {
				if space >= start {
					end = space + 1
				}
				break
			}
			if unicode.IsSpace(runes[end]) {
				space = end
			}
			end++
		}
		line := textAreaLine{start: start, end: end, stops: make([]float32, end-start+1)}
		for i := start + 1; i <= end; i++ {
			line.stops[i-start] = MeasureText(canvas, string(runes[start:i]), Default.Typography.TextSize, false)
		}
		w.lines = append(w.lines, line)
		if end == len(runes) {
			break
		}
		start = end
		if runes[end] == '\n' {
			start++
		}
	}
}

func (w *TextAreaWidget) selection() (int, int) {
	return min(w.anchor, w.cursor), max(w.anchor, w.cursor)
}

func (w *TextAreaWidget) cursorLine() int {
	if w.lineHint >= 0 && w.lineHint < len(w.lines) {
		line := w.lines[w.lineHint]
		if w.cursor >= line.start && w.cursor <= line.end {
			return w.lineHint
		}
	}
	for i := len(w.lines) - 1; i >= 0; i-- {
		if w.cursor >= w.lines[i].start {
			return i
		}
	}
	return 0
}

func (w *TextAreaWidget) keepCursorVisible() {
	if len(w.lines) == 0 {
		return
	}
	visible := max(1, int((w.Bounds().Height()-12)/textAreaLineHeight))
	row := w.cursorLine()
	if row < w.scroll {
		w.scroll = row
	}
	if row >= w.scroll+visible {
		w.scroll = row - visible + 1
	}
	w.clampScroll()
}

func (w *TextAreaWidget) clampScroll() {
	visible := max(1, int((w.Bounds().Height()-12)/textAreaLineHeight))
	w.scroll = max(0, min(w.scroll, len(w.lines)-visible))
}

func (w *TextAreaWidget) cursorAt(p geometry.Point) (pos, row int) {
	if len(w.lines) == 0 {
		return 0, -1
	}
	content := w.Bounds().Inset(geometry.UniformInsets(6))
	row = max(0, min(w.scroll+int((p.Y-content.Min.Y)/textAreaLineHeight), len(w.lines)-1))
	line := w.lines[row]
	x := p.X - content.Min.X
	for i := 1; i < len(line.stops); i++ {
		if x < (line.stops[i-1]+line.stops[i])/2 {
			return line.start + i - 1, row
		}
	}
	return line.end, row
}

func (w *TextAreaWidget) moveCursorAt(p geometry.Point, extend bool) {
	pos, row := w.cursorAt(p)
	w.moveCursor(pos, extend)
	w.lineHint = row
}

func (w *TextAreaWidget) moveCursor(pos int, extend bool) {
	w.revealCursor = true
	w.lineHint = -1
	w.cursor = max(0, min(pos, utf8.RuneCountInString(w.text)))
	if !extend {
		w.anchor = w.cursor
	}
}

func cleanTextAreaInput(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, value)
}

func (w *TextAreaWidget) replaceSelection(value string) bool {
	if w.readOnly {
		return false
	}
	value = cleanTextAreaInput(value)
	runes := []rune(w.text)
	a, b := w.selection()
	if w.maxBytes > 0 {
		available := max(0, w.maxBytes-len(string(runes[:a]))-len(string(runes[b:])))
		if len(value) > available {
			cut := available
			for cut > 0 && !utf8.RuneStart(value[cut]) {
				cut--
			}
			value = value[:cut]
			if value == "" {
				return false
			}
		}
	}
	next := string(runes[:a]) + value + string(runes[b:])
	if next == w.text {
		return false
	}
	w.text = next
	w.moveCursor(a+utf8.RuneCountInString(value), false)
	// The next draw measures only the changed text. No canvas is retained.
	w.lines = nil
	if w.onChange != nil {
		w.onChange(w.text)
	}
	return true
}

func (w *TextAreaWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsEnabled() {
		return false
	}
	handled := false
	switch e := e.(type) {
	case *event.WheelEvent:
		if e.DeltaY() > 0 {
			w.scroll += 3
		} else if e.DeltaY() < 0 {
			w.scroll -= 3
		}
		w.clampScroll()
		handled = true
	case *event.MouseEvent:
		switch e.MouseType {
		case event.MouseEnter:
			ctx.SetCursor(widget.CursorText)
			return true
		case event.MouseLeave:
			ctx.SetCursor(widget.CursorDefault)
			return true
		case event.MousePress:
			if e.Button != event.ButtonLeft {
				return true
			}
			ctx.RequestFocus(w)
			w.moveCursorAt(e.Position, e.Modifiers().IsShift())
			w.dragging = true
			if capture, ok := ctx.(widget.PointerCapturer); ok {
				capture.CapturePointer(w)
			}
			handled = true
		case event.MouseMove, event.MouseDrag:
			if w.dragging {
				// Captured events bypass parent coordinate transforms. The
				// global position is stable on both captured and normal paths.
				position := e.GlobalPosition.Sub(w.ScreenBounds().Min).Add(w.Bounds().Min)
				w.moveCursorAt(position, true)
				handled = true
			}
		case event.MouseRelease:
			if e.Button == event.ButtonLeft && w.dragging {
				w.dragging = false
				if capture, ok := ctx.(widget.PointerCapturer); ok {
					capture.ReleasePointer(w)
				}
				handled = true
			}
		}
	case *event.KeyEvent:
		if !w.IsFocused() || e.KeyType == event.KeyRelease {
			return false
		}
		handled = w.key(e)
		w.revealCursor = handled
	}
	if handled {
		w.SetNeedsRedraw(true)
		ctx.InvalidateRect(w.ScreenBounds())
	}
	return handled
}

func (w *TextAreaWidget) key(e *event.KeyEvent) bool {
	ctrl, shift := e.Modifiers().IsCtrl(), e.Modifiers().IsShift()
	a, b := w.selection()
	runes := []rune(w.text)
	switch {
	case ctrl && e.Key == event.KeyA:
		w.anchor = 0
		w.moveCursor(len(runes), true)
	case ctrl && (e.Key == event.KeyC || e.Key == event.KeyX):
		if a != b {
			widget.ClipboardWrite(string(runes[a:b]))
			if e.Key == event.KeyX {
				w.replaceSelection("")
			}
		}
	case ctrl && e.Key == event.KeyV:
		w.replaceSelection(widget.ClipboardRead())
	case e.Key == event.KeyEnter || e.Key == event.KeyNumpadEnter:
		w.replaceSelection("\n")
	case e.Key == event.KeyBackspace:
		if !w.readOnly && a == b && a > 0 {
			w.anchor = a - 1
			if ctrl {
				w.anchor = textAreaWordBoundary(runes, a, -1)
			}
		}
		w.replaceSelection("")
	case e.Key == event.KeyDelete:
		if !w.readOnly && a == b && b < len(runes) {
			w.anchor = b + 1
			if ctrl {
				w.anchor = textAreaWordBoundary(runes, b, 1)
			}
		}
		w.replaceSelection("")
	case e.Key == event.KeyLeft || e.Key == event.KeyRight:
		pos := w.cursor
		dir := 1
		if e.Key == event.KeyLeft {
			dir = -1
		}
		if !shift && a != b {
			if dir < 0 {
				pos = a
			} else {
				pos = b
			}
		} else {
			if ctrl {
				pos = textAreaWordBoundary(runes, pos, dir)
			} else {
				pos += dir
			}
		}
		w.moveCursor(pos, shift)
	case e.Key == event.KeyUp || e.Key == event.KeyDown:
		if len(w.lines) > 0 {
			row := w.cursorLine()
			col := w.cursor - w.lines[row].start
			if e.Key == event.KeyUp {
				row = max(0, row-1)
			} else {
				row = min(len(w.lines)-1, row+1)
			}
			w.moveCursor(min(w.lines[row].start+col, w.lines[row].end), shift)
			w.lineHint = row
		}
	case e.Key == event.KeyHome || e.Key == event.KeyEnd:
		pos := 0
		row := -1
		if e.Key == event.KeyEnd {
			pos = len(runes)
		}
		if !ctrl && len(w.lines) > 0 {
			row = w.cursorLine()
			line := w.lines[row]
			pos = line.start
			if e.Key == event.KeyEnd {
				pos = line.end
			}
		}
		w.moveCursor(pos, shift)
		w.lineHint = row
	case e.Key == event.KeyTab || e.Key == event.KeyEscape:
		return false
	case e.HasRune() && !ctrl && e.Rune >= ' ':
		w.replaceSelection(string(e.Rune))
	default:
		return false
	}
	return true
}

func textAreaWordBoundary(runes []rune, pos, dir int) int {
	if dir < 0 {
		for pos > 0 && unicode.IsSpace(runes[pos-1]) {
			pos--
		}
		for pos > 0 && !unicode.IsSpace(runes[pos-1]) {
			pos--
		}
	} else {
		for pos < len(runes) && !unicode.IsSpace(runes[pos]) {
			pos++
		}
		for pos < len(runes) && unicode.IsSpace(runes[pos]) {
			pos++
		}
	}
	return pos
}
