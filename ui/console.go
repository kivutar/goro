package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	consoleMargin     = 16
	consoleWidth      = 480
	consoleHeight     = 176
	consoleMaxLines   = 9
	consoleMaxInput   = 120
	consoleMaxHistory = 20
	consoleFieldH     = 24
)

var (
	consoleColorChat        = color.RGBA{R: 235, G: 242, B: 250, A: 255}
	consoleColorSystem      = color.RGBA{R: 252, G: 221, B: 128, A: 255}
	consoleColorBlue        = color.RGBA{R: 0, G: 255, B: 255, A: 255}
	consoleColorError       = color.RGBA{R: 255, G: 132, B: 132, A: 255}
	consoleColorPlaceholder = color.RGBA{R: 150, G: 165, B: 182, A: 255}
	consoleColorInput       = color.RGBA{R: 235, G: 242, B: 250, A: 255}
)

type ConsoleMessage struct {
	Text  string
	Color color.RGBA
}

type ChatConsole struct {
	active        bool
	input         string
	messages      []ConsoleMessage
	scroll        int
	history       []string
	historyIndex  int
	historyDraft  string
	lastMessage   string
	lastMessageAt time.Time

	cacheKey  string
	root      widget.Widget
	rootX     int
	rootY     int
	published bool
}

func (c *ChatConsole) Update(ctx client.Context) bool {
	defer c.Publish(ctx)
	if ctx.Input == nil {
		return false
	}
	w, h := ctx.ScreenSize()
	x, y, cw, ch := consoleBounds(w, h)
	inside := ctx.Input.MouseX >= x && ctx.Input.MouseX < x+cw && ctx.Input.MouseY >= y && ctx.Input.MouseY < y+ch
	if inside && ctx.Input.WheelY != 0 {
		c.scrollBy(ctx.Input.WheelY)
		return true
	}
	if ctx.Input.MouseJustPressed(render.MouseButtonLeft) && inside {
		c.active = true
		c.invalidate()
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) && c.active {
		c.active = false
		c.invalidate()
		return true
	}
	if ctx.Input.JustPressed(render.KeyEnter) {
		if c.active {
			c.submit(ctx)
		} else {
			c.active = true
			c.invalidate()
		}
		return true
	}
	if !c.active {
		return false
	}
	if ctx.Input.JustPressed(render.KeyArrowUp) {
		c.previousInput()
		return true
	}
	if ctx.Input.JustPressed(render.KeyArrowDown) {
		c.nextInput()
		return true
	}
	if text := ctx.Input.TextInput(); text != "" {
		c.appendInput(text)
	}
	if ctx.Input.JustPressed(render.KeyBackspace) {
		c.backspace()
	}
	return true
}

func (c *ChatConsole) Publish(ctx client.Context) {
	if ctx.UIManager == nil {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	x, y, width, height := consoleBounds(screenW, screenH)
	key := c.renderKey(width, height)
	if c.root != nil && c.cacheKey == key && c.rootX == x && c.rootY == y {
		if redraw, ok := c.root.(interface{ SetNeedsRedraw(bool) }); ok {
			redraw.SetNeedsRedraw(true)
		}
		return
	}
	if c.published && c.root != nil {
		ctx.UIManager.RemoveOverlay(c.root)
	}
	c.cacheKey = key
	c.rootX = x
	c.rootY = y
	c.root = primitives.Box(c.widgetTree(width, height)).
		PaddingLeft(float32(x)).
		PaddingTop(float32(y)).
		Width(float32(x + width)).
		Height(float32(y + height))
	ctx.UIManager.AddOverlay(c.root)
	c.published = true
}

func (c *ChatConsole) AddMessage(format string, args ...any) {
	c.addMessageColor(consoleColorChat, format, args...)
}

func (c *ChatConsole) AddSystemMessage(format string, args ...any) {
	c.addMessageColor(consoleColorSystem, format, args...)
}

func (c *ChatConsole) AddBlueMessage(format string, args ...any) {
	c.addMessageColor(consoleColorBlue, format, args...)
}

func (c *ChatConsole) AddErrorMessage(format string, args ...any) {
	c.addMessageColor(consoleColorError, format, args...)
}

func (c *ChatConsole) addMessageColor(messageColor color.RGBA, format string, args ...any) {
	text := strings.TrimSpace(fmt.Sprintf(format, args...))
	if text == "" {
		return
	}
	now := time.Now()
	if text == c.lastMessage && now.Sub(c.lastMessageAt) < time.Second {
		return
	}
	c.lastMessage = text
	c.lastMessageAt = now
	c.messages = append(c.messages, ConsoleMessage{Text: text, Color: messageColor})
	if len(c.messages) > 80 {
		copy(c.messages, c.messages[len(c.messages)-80:])
		c.messages = c.messages[:80]
	}
	c.scroll = 0
	c.ClampScroll()
	c.invalidate()
}

func (c *ChatConsole) submit(ctx client.Context) {
	text := strings.TrimSpace(c.input)
	if text == "" {
		c.active = false
		c.invalidate()
		return
	}
	c.rememberInput(text)
	if c.SubmitCommand(ctx, text) {
		return
	}
	name := "Player"
	if ctx.Session != nil && strings.TrimSpace(ctx.Session.Selected.Name) != "" {
		name = ctx.Session.Selected.Name
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		return
	}
	if err := ctx.Network.SendGlobalChat(name, text); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		return
	}
	c.input = ""
	c.active = false
	c.invalidate()
}

func (c *ChatConsole) SubmitCommand(ctx client.Context, text string) bool {
	if !strings.HasPrefix(text, "/") {
		return false
	}
	command := strings.ToLower(strings.Fields(text)[0])
	switch command {
	case "/sit":
		c.submitSitStand(ctx, !consolePlayerSitting(ctx))
		return true
	case "/stand":
		c.submitSitStand(ctx, false)
		return true
	case "/noshift", "/ns":
		if ctx.Session == nil {
			c.AddErrorMessage("noshift failed: no session")
			c.input = ""
			c.active = false
			c.invalidate()
			return true
		}
		ctx.Session.NoShift = !ctx.Session.NoShift
		if ctx.Session.NoShift {
			c.AddSystemMessage("No Shift: On")
		} else {
			c.AddSystemMessage("No Shift: Off")
		}
		c.input = ""
		c.active = false
		c.invalidate()
		return true
	case "/memo":
		c.submitMemo(ctx)
		return true
	default:
		return false
	}
}

func (c *ChatConsole) submitMemo(ctx client.Context) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.input = ""
		c.active = false
		c.invalidate()
		return
	}
	if err := ctx.Network.SendRememberWarpPoint(); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		return
	}
	c.input = ""
	c.active = false
	c.invalidate()
}

func (c *ChatConsole) submitSitStand(ctx client.Context, sit bool) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		return
	}
	targetID := consoleLocalActorID(ctx)
	if targetID == 0 {
		c.AddErrorMessage("send failed: missing local actor")
		return
	}
	action := network.ActionStandUp
	if sit {
		action = network.ActionSitDown
	}
	if err := ctx.Network.SendActionRequest(targetID, action); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		return
	}
	if ctx.World != nil {
		ctx.World.Player.Sitting = sit
		if sit {
			ctx.World.Player.Moving = false
		}
	}
	c.input = ""
	c.active = false
	c.invalidate()
}

func consolePlayerSitting(ctx client.Context) bool {
	return ctx.World != nil && ctx.World.Player.Sitting
}

func consoleLocalActorID(ctx client.Context) uint32 {
	if ctx.Session != nil {
		if ctx.Session.AccountID != 0 {
			return ctx.Session.AccountID
		}
		if ctx.Session.CharID != 0 {
			return ctx.Session.CharID
		}
	}
	if ctx.World != nil {
		return ctx.World.Player.ID
	}
	return 0
}

func (c *ChatConsole) appendInput(text string) {
	if text == "" {
		return
	}
	c.historyIndex = 0
	c.historyDraft = ""
	runes := []rune(c.input + text)
	if len(runes) > consoleMaxInput {
		runes = runes[:consoleMaxInput]
	}
	c.input = string(runes)
	c.invalidate()
}

func (c *ChatConsole) backspace() {
	runes := []rune(c.input)
	if len(runes) == 0 {
		return
	}
	c.historyIndex = 0
	c.historyDraft = ""
	c.input = string(runes[:len(runes)-1])
	c.invalidate()
}

func (c *ChatConsole) rememberInput(text string) {
	if text == "" {
		return
	}
	if len(c.history) == 0 || c.history[len(c.history)-1] != text {
		c.history = append(c.history, text)
		if len(c.history) > consoleMaxHistory {
			copy(c.history, c.history[len(c.history)-consoleMaxHistory:])
			c.history = c.history[:consoleMaxHistory]
		}
	}
	c.historyIndex = 0
	c.historyDraft = ""
}

func (c *ChatConsole) previousInput() {
	if len(c.history) == 0 {
		return
	}
	if c.historyIndex == 0 {
		c.historyDraft = c.input
		c.historyIndex = len(c.history)
	} else if c.historyIndex > 1 {
		c.historyIndex--
	}
	c.input = c.history[c.historyIndex-1]
	c.invalidate()
}

func (c *ChatConsole) nextInput() {
	if c.historyIndex == 0 {
		return
	}
	if c.historyIndex < len(c.history) {
		c.historyIndex++
		c.input = c.history[c.historyIndex-1]
	} else {
		c.historyIndex = 0
		c.input = c.historyDraft
		c.historyDraft = ""
	}
	c.invalidate()
}

func (c *ChatConsole) widgetTree(width, height int) widget.Widget {
	contentWidth := maxInt(1, width-16)
	messageWidgets := make([]widget.Widget, 0, consoleMaxLines)
	for _, line := range c.visibleLines() {
		messageWidgets = append(messageWidgets,
			rotheme.Text(line.Text).
				Color(Color(line.Color)).
				MaxLines(1).
				Ellipsis(),
		)
	}
	fieldText := c.inputText()
	if c.active && time.Now().UnixMilli()/500%2 == 0 {
		fieldText += "|"
	}
	field := primitives.Box(
		rotheme.Text(fieldText).
			Color(Color(consoleColorInput)).
			MaxLines(1).
			Ellipsis(),
	).
		Width(float32(contentWidth)).
		Height(consoleFieldH).
		PaddingXY(6, 3).
		Background(widget.RGBA8(5, 8, 13, 205)).
		BorderStyle(1, widget.RGBA8(190, 208, 230, 120)).
		CrossAlign(primitives.CrossAxisStretch)
	messageHeight := maxInt(20, height-16-consoleFieldH-4)
	messages := primitives.Box(messageWidgets...).
		Width(float32(contentWidth)).
		Height(float32(messageHeight)).
		Gap(1).
		CrossAlign(primitives.CrossAxisStretch)
	return primitives.Box(messages, field).
		Width(float32(width)).
		Height(float32(height)).
		PaddingXY(8, 6).
		Gap(4).
		Background(widget.RGBA8(14, 18, 24, 188)).
		BorderStyle(1, widget.RGBA8(180, 198, 218, 95)).
		Rounded(WindowRadius).
		CrossAlign(primitives.CrossAxisStretch)
}

func (c *ChatConsole) visibleLines() []ConsoleMessage {
	if len(c.messages) == 0 {
		return []ConsoleMessage{{Text: "Server messages will appear here.", Color: consoleColorPlaceholder}}
	}
	c.ClampScroll()
	start := len(c.messages) - consoleMaxLines - c.scroll
	if start < 0 {
		start = 0
	}
	end := minInt(len(c.messages), start+consoleMaxLines)
	out := make([]ConsoleMessage, 0, end-start)
	out = append(out, c.messages[start:end]...)
	return out
}

func (c *ChatConsole) renderKey(width, height int) string {
	blink := int64(0)
	if c.active {
		blink = time.Now().UnixMilli() / 500
	}
	return fmt.Sprintf("%dx%d:%t:%s:%s:%d:%d", width, height, c.active, c.input, c.messagesKey(), blink, c.scroll)
}

func (c *ChatConsole) invalidate() {
	c.cacheKey = ""
}

func (c *ChatConsole) scrollBy(wheelY float64) {
	step := int(math.Ceil(math.Abs(wheelY))) * 3
	if step < 1 {
		step = 1
	}
	if wheelY > 0 {
		c.scrollLines(step)
	} else {
		c.scrollLines(-step)
	}
}

func (c *ChatConsole) scrollLines(lines int) {
	c.scroll += lines
	c.ClampScroll()
	c.invalidate()
}

func (c *ChatConsole) ClampScroll() {
	maxScroll := maxInt(0, len(c.messages)-consoleMaxLines)
	if c.scroll < 0 {
		c.scroll = 0
	}
	if c.scroll > maxScroll {
		c.scroll = maxScroll
	}
}

func (c *ChatConsole) messagesKey() string {
	var b strings.Builder
	for _, msg := range c.messages {
		fmt.Fprintf(&b, "%02x%02x%02x%02x:%s\n", msg.Color.R, msg.Color.G, msg.Color.B, msg.Color.A, msg.Text)
	}
	return b.String()
}

func (c *ChatConsole) inputText() string {
	if c.active || c.input != "" {
		return c.input
	}
	return "Press Enter to chat"
}

func (c *ChatConsole) Messages() []ConsoleMessage {
	if c == nil {
		return nil
	}
	out := make([]ConsoleMessage, len(c.messages))
	copy(out, c.messages)
	return out
}

func consoleBounds(screenW, screenH int) (x, y, w, h int) {
	w = minInt(consoleWidth, maxInt(260, screenW-2*consoleMargin))
	h = minInt(consoleHeight, maxInt(128, screenH-2*consoleMargin))
	x = consoleMargin
	y = maxInt(consoleMargin, screenH-h-consoleMargin)
	return x, y, w, h
}
