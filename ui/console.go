package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	consoleMargin     = windowScreenMargin
	consoleWidth      = 480
	consoleHeight     = 176
	consoleMaxLines   = 9
	consoleMaxInput   = 120
	consoleMaxHistory = 20
	consoleFieldH     = 24
	consoleLineH      = 14
)

var (
	consoleColorChat        = color.RGBA{R: 235, G: 242, B: 250, A: 255}
	consoleColorSystem      = color.RGBA{R: 252, G: 221, B: 128, A: 255}
	consoleColorBlue        = color.RGBA{R: 0, G: 255, B: 255, A: 255}
	consoleColorGuild       = color.RGBA{R: 255, G: 255, B: 99, A: 255}
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

	OnGuildWindow func()

	ctx        client.Context
	window     Window
	inputField *textfield.Widget
	scrollY    state.Signal[float32]
	cacheKey   string
	messageH   int
}

func (c *ChatConsole) Active() bool {
	return c != nil && c.active
}

func (c *ChatConsole) Update(ctx client.Context) bool {
	c.ctx = ctx
	c.ensureWindow(ctx)
	c.syncActiveFromField()
	defer c.Publish(ctx)
	if ctx.Input == nil {
		return false
	}
	if ctx.Input.JustPressed(input.KeyEscape) && c.active {
		c.setActive(false)
		return true
	}
	if ctx.Input.JustPressed(input.KeyEnter) && !c.active {
		c.setActive(true)
		return true
	}
	if c.active && c.clickedOutside(ctx) {
		c.setActive(false)
		return false
	}
	if c.active && ctx.Input.JustPressed(input.KeyArrowUp) {
		c.previousInput()
		return true
	}
	if c.active && ctx.Input.JustPressed(input.KeyArrowDown) {
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
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
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
		c.window = NewWindow(width, height)
		c.window.titleHeight = 0
		c.window.SetFullRedraw(true)
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

func (c *ChatConsole) AddGuildMessage(format string, args ...any) {
	c.addMessageColor(consoleColorGuild, format, args...)
}

func (c *ChatConsole) AddColoredMessage(messageColor color.RGBA, format string, args ...any) {
	if messageColor.A == 0 {
		messageColor.A = 255
	}
	c.addMessageColor(messageColor, format, args...)
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
	if strings.HasPrefix(text, "%") {
		c.submitPartyChat(ctx, name, strings.TrimSpace(strings.TrimPrefix(text, "%")))
		return
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

func (c *ChatConsole) submitPartyChat(ctx client.Context, name string, message string) {
	if message == "" {
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		return
	}
	if err := ctx.Network.SendPartyMessage(fmt.Sprintf("%s : %s", name, message)); err != nil {
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
	case "/mineffect":
		c.submitLessEffects(ctx)
		return true
	case "/hoai":
		c.submitCompanionAI(ctx, true)
		return true
	case "/merai":
		c.submitCompanionAI(ctx, false)
		return true
	case "/memo":
		c.submitMemo(ctx)
		return true
	case "/screenshot":
		c.submitScreenshot(ctx)
		return true
	case "/organize":
		c.submitOrganizeParty(ctx, text)
		return true
	case "/guild":
		c.submitCreateGuild(ctx, text)
		return true
	case "/guildwindow", "/guildinfo":
		if c.OnGuildWindow != nil {
			c.OnGuildWindow()
		}
		c.setInput("")
		c.setActive(false)
		return true
	case "/emblem", "/guildemblem":
		c.submitGuildEmblem(ctx, text)
		return true
	case "/leave":
		c.submitLeaveParty(ctx)
		return true
	case "/invite":
		c.submitInviteParty(ctx, text)
		return true
	case "/accept":
		c.submitPartyInviteConfig(ctx, false)
		return true
	case "/refuse":
		c.submitPartyInviteConfig(ctx, true)
		return true
	case "/w", "/whisper":
		c.submitWhisper(ctx, text)
		return true
	case "/ex":
		c.submitWhisperIgnore(ctx, text, false)
		return true
	case "/in":
		c.submitWhisperIgnore(ctx, text, true)
		return true
	case "/exall":
		c.submitWhisperIgnoreAll(ctx, false)
		return true
	case "/inall":
		c.submitWhisperIgnoreAll(ctx, true)
		return true
	default:
		if emotionID, ok := db.EmotionCommandID(strings.TrimPrefix(command, "/")); ok {
			c.submitEmotion(ctx, emotionID)
			return true
		}
		return false
	}
}

func (c *ChatConsole) submitScreenshot(ctx client.Context) {
	if ctx.RequestScreenshot == nil {
		c.AddErrorMessage("screenshot failed: unavailable")
		c.setInput("")
		c.setActive(false)
		return
	}
	path, err := ctx.RequestScreenshot()
	if err != nil {
		c.AddErrorMessage("screenshot failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.AddSystemMessage("Screenshot: %s", path)
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitLessEffects(ctx client.Context) {
	if ctx.Session == nil {
		c.AddErrorMessage("mineffect failed: no session")
		c.setInput("")
		c.setActive(false)
		return
	}
	enabled := !ctx.Session.LessEffects
	if ctx.Network != nil {
		if err := ctx.Network.SendLessEffect(enabled); err != nil {
			c.AddErrorMessage("send failed: %s", err)
			c.setInput("")
			c.setActive(false)
			return
		}
	}
	ctx.Session.LessEffects = enabled
	if enabled {
		c.AddSystemMessage("Less Effects: On")
	} else {
		c.AddSystemMessage("Less Effects: Off")
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitCompanionAI(ctx client.Context, homunculus bool) {
	if ctx.Session == nil {
		c.AddErrorMessage("ai mode failed: no session")
		c.setInput("")
		c.setActive(false)
		return
	}
	label := "Mercenary"
	enabled := false
	if homunculus {
		label = "Homunculus"
		ctx.Session.HomunculusCustomAI = !ctx.Session.HomunculusCustomAI
		enabled = ctx.Session.HomunculusCustomAI
	} else {
		ctx.Session.MercenaryCustomAI = !ctx.Session.MercenaryCustomAI
		enabled = ctx.Session.MercenaryCustomAI
	}
	if enabled {
		c.AddSystemMessage("%s AI: Custom", label)
	} else {
		c.AddSystemMessage("%s AI: Default", label)
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitOrganizeParty(ctx client.Context, text string) {
	name := consoleCommandArgs(text)
	name = strings.Trim(name, `"`)
	if name == "" {
		c.AddErrorMessage("usage: /organize party_name")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendMakeParty(name); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Session != nil {
		ctx.Session.Party.Name = name
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitCreateGuild(ctx client.Context, text string) {
	name := consoleCommandArgs(text)
	name = strings.Trim(name, `"`)
	if name == "" {
		c.AddErrorMessage("usage: /guild guild_name")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Session == nil {
		c.AddErrorMessage("guild creation failed: no session")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendCreateGuild(ctx.Session.CharID, name); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	ctx.Session.PendingGuildName = name
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitGuildEmblem(ctx client.Context, text string) {
	path := strings.Trim(consoleCommandArgs(text), `"`)
	if path == "" {
		c.AddErrorMessage("usage: /emblem path/to/emblem.bmp")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		c.AddErrorMessage("emblem failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	if len(data) < 2 || data[0] != 'B' || data[1] != 'M' {
		c.AddErrorMessage("emblem failed: expected BMP file")
		c.setInput("")
		c.setActive(false)
		return
	}
	if len(data) > 1783 {
		c.AddErrorMessage("emblem failed: BMP is too large")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendGuildEmblem(data); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.AddSystemMessage("Guild emblem uploaded.")
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitLeaveParty(ctx client.Context) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendLeaveParty(); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitInviteParty(ctx client.Context, text string) {
	name := consoleCommandArgs(text)
	name = strings.Trim(name, `"`)
	if name == "" {
		c.AddErrorMessage("usage: /invite character_name")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.World == nil {
		c.AddErrorMessage("invite failed: character is not visible")
		c.setInput("")
		c.setActive(false)
		return
	}
	for _, actor := range ctx.World.Actors {
		if !strings.EqualFold(strings.TrimSpace(actor.Name), name) {
			continue
		}
		if ctx.Network == nil {
			c.AddErrorMessage("send failed: not connected")
			c.setInput("")
			c.setActive(false)
			return
		}
		if err := ctx.Network.SendPartyInvite(actor.ID, actor.Name); err != nil {
			c.AddErrorMessage("send failed: %s", err)
			c.setInput("")
			c.setActive(false)
			return
		}
		c.AddBlueMessage("%s has received an invitation to join your party.", actor.Name)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.AddErrorMessage("invite failed: character is not visible")
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitPartyInviteConfig(ctx client.Context, refuseInvites bool) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendPartyInviteConfig(refuseInvites); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Session != nil {
		ctx.Session.Party.RefuseInvites = refuseInvites
	}
	if refuseInvites {
		c.AddSystemMessage("Party invites: Refused")
	} else {
		c.AddSystemMessage("Party invites: Accepted")
	}
	c.setInput("")
	c.setActive(false)
}

func consoleCommandArgs(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), fields[0]))
}

func (c *ChatConsole) submitEmotion(ctx client.Context, emotionID uint8) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendEmotion(emotionID); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitWhisper(ctx client.Context, text string) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		c.AddErrorMessage("usage: /w name message")
		c.setInput("")
		c.setActive(false)
		return
	}
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), fields[0]))
	target, message, ok := strings.Cut(rest, " ")
	target = strings.TrimSpace(target)
	message = strings.TrimSpace(message)
	if !ok || target == "" || message == "" {
		c.AddErrorMessage("usage: /w name message")
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendWhisper(target, message); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.AddBlueMessage("[ To %s ] : %s", target, message)
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitWhisperIgnore(ctx client.Context, text string, allow bool) {
	name := consoleCommandArgs(text)
	name = strings.Trim(name, `"`)
	if name == "" {
		if allow {
			c.AddErrorMessage("usage: /in character_name")
		} else {
			c.AddErrorMessage("usage: /ex character_name")
		}
		c.setInput("")
		c.setActive(false)
		return
	}
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendWhisperIgnore(name, allow); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.setInput("")
	c.setActive(false)
}

func (c *ChatConsole) submitWhisperIgnoreAll(ctx client.Context, allow bool) {
	if ctx.Network == nil {
		c.AddErrorMessage("send failed: not connected")
		c.setInput("")
		c.setActive(false)
		return
	}
	if err := ctx.Network.SendWhisperIgnoreAll(allow); err != nil {
		c.AddErrorMessage("send failed: %s", err)
		c.setInput("")
		c.setActive(false)
		return
	}
	c.setInput("")
	c.setActive(false)
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
			primitives.Box(
				rotheme.Text(line.Text).
					Color(Color(line.Color)).
					MaxLines(1).
					Ellipsis(),
			).Height(consoleLineH),
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
			primitives.Box(messageList).
				PaddingRight(ROScrollbarGutter),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollYSignal(c.ensureScrollSignal()),
			scrollview.ScrollStep(float32(consoleLineH*3)),
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
	contentHeight := float32(lines*consoleLineH + maxInt(0, lines-1))
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
