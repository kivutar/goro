package rotheme

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestTextAreaMultilineEditingAndByteLimit(t *testing.T) {
	var changed string
	w := TextArea("", 8, func(s string) { changed = s })
	w.SetFocused(true)
	ctx := widget.NewContext()
	for _, r := range "éabc" {
		w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'é', event.ModNone))
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'z', event.ModNone))
	if w.Text() != "éabc\né" || changed != w.Text() || len(w.Text()) != 8 {
		t.Fatalf("multiline text = %q, changed = %q", w.Text(), changed)
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyBackspace, 0, event.ModNone))
	if w.Text() != "éabc\n" || !utf8.ValidString(w.Text()) {
		t.Fatalf("backspace damaged UTF-8: %q", w.Text())
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyA, 0, event.ModCtrl))
	w.replaceSelection("ààààà")
	if w.Text() != "àààà" {
		t.Fatalf("selection replacement = %q", w.Text())
	}
	for _, key := range []event.Key{event.KeyTab, event.KeyEscape} {
		if w.Event(ctx, event.NewKeyEvent(event.KeyPress, key, 0, event.ModNone)) {
			t.Fatalf("editor swallowed navigation key %v", key)
		}
	}
}

func TestTextAreaNormalizesPastedAndReceivedText(t *testing.T) {
	w := TextArea("a\r\nb\rc\t\x00d", 0, nil)
	if w.Text() != "a\nb\nc d" {
		t.Fatalf("received text = %q", w.Text())
	}
	w.moveCursor(utf8.RuneCountInString(w.Text()), false)
	w.replaceSelection("\r\n\t\x01é")
	if w.Text() != "a\nb\nc d\n é" {
		t.Fatalf("pasted text = %q", w.Text())
	}
}

func TestTextAreaReadOnlyDoesNotMutateTextOrSelection(t *testing.T) {
	w := TextArea("One\nTwo", 0, nil)
	w.SetReadOnly(true)
	w.SetFocused(true)
	w.moveCursor(2, false)
	for _, key := range []event.Key{event.KeyBackspace, event.KeyDelete, event.KeyEnter} {
		w.Event(widget.NewContext(), event.NewKeyEvent(event.KeyPress, key, 0, event.ModNone))
	}
	w.replaceSelection("replace")
	if w.Text() != "One\nTwo" || w.anchor != 2 || w.cursor != 2 {
		t.Fatalf("read-only state = %q %d:%d", w.Text(), w.anchor, w.cursor)
	}
}

func TestTextAreaWrapAndScroll(t *testing.T) {
	w := TextArea("ab cd ef\n\nlast\n", 0, nil)
	ctx := widget.NewContext()
	width := Default.Typography.TextSize*0.5*4 + 12
	w.Layout(ctx, geometry.Tight(geometry.Sz(width, 48)))
	w.Draw(ctx, &uitest.MockCanvas{})
	var lines []string
	runes := []rune(w.Text())
	for _, l := range w.lines {
		lines = append(lines, string(runes[l.start:l.end]))
	}
	if got := strings.Join(lines, "|"); got != "ab |cd |ef||last|" {
		t.Fatalf("wrapped lines = %q", got)
	}
	old := &w.lines[0]
	w.Draw(ctx, &uitest.MockCanvas{})
	if old != &w.lines[0] {
		t.Fatal("unchanged draw remeasured text")
	}
	w.SetFocused(true)
	w.Event(ctx, event.NewWheelEvent(geometry.Pt(0, 1), geometry.Point{}, geometry.Point{}, event.ModNone))
	w.Draw(ctx, &uitest.MockCanvas{})
	if w.scroll != 3 {
		t.Fatalf("wheel scroll was undone by caret: %d", w.scroll)
	}
	w.Layout(ctx, geometry.Tight(geometry.Sz(400, 300)))
	w.Draw(ctx, &uitest.MockCanvas{})
	if w.scroll != 0 {
		t.Fatalf("resize left content scrolled out of view: %d", w.scroll)
	}
	w.moveCursor(len(runes), false)
	w.Layout(ctx, geometry.Tight(geometry.Sz(width, 48)))
	w.Draw(ctx, &uitest.MockCanvas{})
	if w.scroll != 4 {
		t.Fatalf("caret not scrolled into view: %d", w.scroll)
	}
}

func TestTextAreaWordNavigationAndDeletion(t *testing.T) {
	w := TextArea("one  two three", 0, nil)
	w.SetFocused(true)
	ctx := widget.NewContext()
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModCtrl))
	if w.cursor != 5 {
		t.Fatalf("next word = %d", w.cursor)
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModCtrl))
	if w.cursor != 9 {
		t.Fatalf("next word = %d", w.cursor)
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyBackspace, 0, event.ModCtrl))
	if w.Text() != "one  three" || w.cursor != 5 {
		t.Fatalf("delete word = %q, cursor %d", w.Text(), w.cursor)
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModCtrl))
	if w.cursor != 0 {
		t.Fatalf("previous word = %d", w.cursor)
	}
}

func TestTextAreaEndStaysOnSoftWrappedLine(t *testing.T) {
	w := TextArea("ab cd ef", 0, nil)
	ctx := widget.NewContext()
	w.SetFocused(true)
	w.Layout(ctx, geometry.Tight(geometry.Sz(Default.Typography.TextSize*2+12, 80)))
	w.Draw(ctx, &uitest.MockCanvas{})
	w.moveCursor(1, false)
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEnd, 0, event.ModNone))
	if w.cursor != 3 || w.cursorLine() != 0 {
		t.Fatalf("End moved off the visual line: cursor=%d line=%d", w.cursor, w.cursorLine())
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyHome, 0, event.ModNone))
	if w.cursor != 0 {
		t.Fatalf("Home after End went to a different line: %d", w.cursor)
	}
	// Clicking beyond a wrapped line selects its end, not the next line's
	// start. Vertical navigation preserves that same visual-line affinity.
	w.moveCursorAt(geometry.Pt(w.Bounds().Max.X, 8), false)
	w.Draw(ctx, &uitest.MockCanvas{})
	if w.cursor != 3 || w.cursorLine() != 0 {
		t.Fatalf("click at line end = cursor %d, line %d", w.cursor, w.cursorLine())
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyDown, 0, event.ModNone))
	if w.cursor != 6 || w.cursorLine() != 1 {
		t.Fatalf("Down at line end = cursor %d, line %d", w.cursor, w.cursorLine())
	}
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUp, 0, event.ModNone))
	if w.cursor != 3 || w.cursorLine() != 0 {
		t.Fatalf("Up at line end = cursor %d, line %d", w.cursor, w.cursorLine())
	}
}

func TestTextAreaInvalidatesScreenCoordinates(t *testing.T) {
	w := TextArea("", 20, nil)
	w.SetFocused(true)
	w.SetBounds(geometry.NewRect(5, 7, 120, 40))
	canvas := &uitest.MockCanvas{}
	canvas.PushTransform(geometry.Pt(150, 200))
	widget.StampScreenOrigin(w, canvas)
	ctx := widget.NewContext()
	var damage geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) { damage = r })
	w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'x', event.ModNone))
	if damage != w.ScreenBounds() {
		t.Fatalf("redraw damage = %v, want screen bounds %v", damage, w.ScreenBounds())
	}
}
