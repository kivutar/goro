package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
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
)

type ConsoleMessage struct {
	Text  string
	Color color.RGBA
}

type ChatConsole struct {
	active        bool
	input         string
	messages      []ConsoleMessage
	history       []string
	historyIndex  int
	historyDraft  string
	lastMessage   string
	lastMessageAt time.Time

	ctx        client.Context
	window     WindowState
	inputField *textfield.Widget
	scrollY    state.Signal[float32]
	cacheKey   string
	messageH   int
}

func (c *ChatConsole) Update(ctx client.Context) bool {
	c.ctx = ctx
	c.ensureWindow(ctx)
	c.syncActiveFromField()
	defer c.Publish(ctx)
	if ctx.Input == nil {
		return false
	}
	if ctx.Input.JustPressed(render.KeyEscape) && c.active {
		c.setActive(false)
		return true
	}
	if ctx.Input.JustPressed(render.KeyEnter) && !c.active {
		c.setActive(true)
		return true
	}
	if c.active && c.clickedOutside(ctx) {
		c.setActive(false)
		return false
	}
	if c.active && ctx.Input.JustPressed(render.KeyArrowUp) {
		c.previousInput()
		return true
	}
	if c.active && ctx.Input.JustPressed(render.KeyArrowDown) {
		c.nextInput()
		return true
	}
	consumed := c.window.Update(ctx)
	c.syncActiveFromField()
	return consumed || c.active
}

func (c *ChatConsole) Publish(ctx client.Context) {
	c.ensureWindow(ctx)
	c.window.Publish(ctx)
}

func (c *ChatConsole) clickedOutside(ctx client.Context) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return false
	}
	x, y, width, height := consoleBounds(ctx.ScreenSize())
	return !pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, x, y, width, height)
}

func (c *ChatConsole) ensureWindow(ctx client.Context) {
	screenW, screenH := ctx.ScreenSize()
	x, y, width, height := consoleBounds(screenW, screenH)
	key := c.renderKey(width, height)
	if c.window.width == 0 {
		c.window = NewWindowState(width, height)
		c.window.titleHeight = 0
		c.window.SetCloseOnEscape(false)
	}
	c.window.SetAutoPosition(x, y)
	c.window.SetSize(width, height)
	if !c.window.IsOpen() {
		c.cacheKey = key
		c.window.OpenAt(x, y, c.widgetTree(width, height))
		return
	}
	if c.cacheKey != key {
		c.cacheKey = key
		c.window.SetContent(c.widgetTree(width, height))
	}
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
	c.invalidate()
	c.scrollToBottom()
}

func (c *ChatConsole) submit(ctx client.Context) {
	text := strings.TrimSpace(c.input)
	if text == "" {
		c.setActive(false)
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
	c.setInput("")
	c.setActive(false)
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
			c.setInput("")
			c.setActive(false)
			return true
		}
		ctx.Session.NoShift = !ctx.Session.NoShift
		if ctx.Session.NoShift {
			c.AddSystemMessage("No Shift: On")
		} else {
			c.AddSystemMessage("No Shift: Off")
		}
		c.setInput("")
		c.setActive(false)
		return true
	case "/noctrl", "/nc":
		if ctx.Session == nil {
			c.AddErrorMessage("noctrl failed: no session")
			c.setInput("")
			c.setActive(false)
			return true
		}
		ctx.Session.NoCtrl = !ctx.Session.NoCtrl
		if ctx.Session.NoCtrl {
			c.AddSystemMessage("No Ctrl: On")
		} else {
			c.AddSystemMessage("No Ctrl: Off")
		}
		c.setInput("")
		c.setActive(false)
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
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendRememberWarpPoint(); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		return
	}
	c.setInput("")
	c.setActive(false)
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
	c.setInput("")
	c.setActive(false)
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
	c.setInput(c.history[c.historyIndex-1])
}

func (c *ChatConsole) nextInput() {
	if c.historyIndex == 0 {
		return
	}
	if c.historyIndex < len(c.history) {
		c.historyIndex++
		c.setInput(c.history[c.historyIndex-1])
	} else {
		c.historyIndex = 0
		c.setInput(c.historyDraft)
		c.historyDraft = ""
	}
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
	field := primitives.Box(c.inputWidget()).
		Width(float32(contentWidth)).
		Height(consoleFieldH).
		CrossAlign(primitives.CrossAxisStretch)
	messageHeight := maxInt(20, height-16-consoleFieldH-4)
	c.messageH = messageHeight
	messageList := primitives.Box(messageWidgets...).
		Width(float32(contentWidth)).
		Gap(1).
		CrossAlign(primitives.CrossAxisStretch)
	scrollY := consoleBottomScrollY(len(messageWidgets), messageHeight)
	c.ensureScrollSignal().Set(scrollY)
	messages := primitives.Box(
		scrollview.New(
			messageList,
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollYSignal(c.ensureScrollSignal()),
			scrollview.ScrollStep(float32(rotheme.Default.Typography.TextSize*3)),
		),
	).
		Width(float32(contentWidth)).
		Height(float32(messageHeight)).
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
	out := make([]ConsoleMessage, 0, len(c.messages))
	out = append(out, c.messages...)
	return out
}

func consoleBottomScrollY(lines int, viewportHeight int) float32 {
	if lines <= 0 {
		return 0
	}
	contentHeight := float32(lines)*rotheme.Default.Typography.TextSize + float32(maxInt(0, lines-1))
	scrollY := contentHeight - float32(viewportHeight)
	if scrollY < 0 {
		return 0
	}
	return scrollY
}

func (c *ChatConsole) renderKey(width, height int) string {
	return fmt.Sprintf("%dx%d:%s", width, height, c.messagesKey())
}

func (c *ChatConsole) invalidate() {
	c.cacheKey = ""
}

func (c *ChatConsole) messagesKey() string {
	var b strings.Builder
	for _, msg := range c.messages {
		fmt.Fprintf(&b, "%02x%02x%02x%02x:%s\n", msg.Color.R, msg.Color.G, msg.Color.B, msg.Color.A, msg.Text)
	}
	return b.String()
}

func (c *ChatConsole) inputWidget() *textfield.Widget {
	if c.inputField != nil {
		return c.inputField
	}
	c.inputField = rotheme.TextField(
		c.input,
		textfield.TypeText,
		func(value string) {
			c.input = value
			c.historyIndex = 0
			c.historyDraft = ""
			c.scrollToBottom()
		},
		func(string) {
			c.submit(c.ctx)
		},
		textfield.MaxLength(consoleMaxInput),
		textfield.Placeholder("Press Enter to chat"),
	)
	c.inputField.SetFocused(c.active)
	return c.inputField
}

func (c *ChatConsole) setInput(text string) {
	c.input = text
	if c.inputField != nil && c.inputField.Text() != text {
		c.inputField.SetText(text)
	}
	c.scrollToBottom()
	c.invalidate()
}

func (c *ChatConsole) setActive(active bool) {
	c.active = active
	if c.inputField != nil {
		c.inputField.SetFocused(active)
	}
	if active {
		c.scrollToBottom()
	}
	c.invalidate()
}

func (c *ChatConsole) syncActiveFromField() {
	if c.inputField != nil {
		c.active = c.inputField.IsFocused()
	}
}

func (c *ChatConsole) ensureScrollSignal() state.Signal[float32] {
	if c.scrollY == nil {
		c.scrollY = state.NewSignal[float32](0)
	}
	return c.scrollY
}

func (c *ChatConsole) scrollToBottom() {
	if c.messageH <= 0 {
		return
	}
	c.ensureScrollSignal().Set(consoleBottomScrollY(len(c.visibleLines()), c.messageH))
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
