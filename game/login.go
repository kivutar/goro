package game

import (
	"context"
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

type LoginMode struct {
	selected          int
	phase             loginPhase
	status            string
	packets           []string
	console           gameui.ChatConsole
	autoAttempted     bool
	autoCharAttempted bool
	fade              loginFadeState
	username          string
	password          string
	background        *render.Image
	bgTiles           []*render.Image
	bgSource          string
	bgLoaded          bool
	bgmStarted        bool
	selectedSlot      int
	maxSlots          int
	charViews         map[uint32]*humanoidSpriteView
	charViewFailed    map[uint32]struct{}
	charPreviewImages map[uint32]image.Image
	charWindow        *render.Image
	charBox           *render.Image
	loginWindow       *gameui.LoginWindow
	charSelectWindow  *gameui.CharacterSelectWindow
	charCreateWindow  *gameui.CharacterCreateWindow
	charDeleteConfirm gameui.ConfirmModal
	charDeletePrompt  gameui.TextPromptWindow
	deleteCharID      uint32
	create            charCreateState
	cursor            roCursorState
	quitConfirm       gameui.ConfirmModal
	disconnectDialog  gameui.ConfirmModal
}

type loginPhase int

const (
	loginPhaseAccount loginPhase = iota
	loginPhaseCharacter
	loginPhaseCreate
)

type loginFadePhase int

const (
	loginFadeNone loginFadePhase = iota
	loginFadeOut
	loginFadeIn
)

type loginFadeState struct {
	phase      loginFadePhase
	started    time.Time
	target     loginPhase
	hasTarget  bool
	enterWorld bool
}

const loginTransitionDuration = 500 * time.Millisecond
const loginConfirmSFX = "버튼소리.wav"

func NewLoginMode() *LoginMode {
	return &LoginMode{status: "select a server", maxSlots: 9}
}

func NewCharacterSelectMode(ctx client.Context, console gameui.ChatConsole) *LoginMode {
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.status = "select a character"
	mode.autoAttempted = true
	mode.console = console
	mode.prepareCharacterSelectFromSession(ctx)
	return mode
}

func (m *LoginMode) Name() string {
	return "login"
}

func (m *LoginMode) Enter(ctx client.Context) {
	m.clearLoginWindows(ctx)
	if m.username == "" {
		m.username = ctx.Config.Login.Username
	}
	if m.password == "" {
		m.password = ctx.Config.Login.Password
	}
	m.loadBackground(ctx)
	m.loadCharacterSelectSkin(ctx)
	m.cursor.ensureLoaded(ctx)
	render.SetCursorMode(render.CursorModeHidden)
	m.playLoginBGM(ctx)
	if m.phase == loginPhaseCharacter {
		m.prepareCharacterSelectFromSession(ctx)
		m.showCharacterSelectWindow(ctx)
		m.reconnectCharacterServer(ctx)
	}
	if m.phase == loginPhaseAccount && len(ctx.Resources.ClientInfo.Connections) == 0 {
		m.status = "no login servers discovered"
	}
}

func (m *LoginMode) Update(ctx client.Context) (Mode, error) {
	now := time.Now()
	if m.updateFade(ctx, now) {
		return m.nextWorldMode(ctx, now), nil
	}

	conns := ctx.Resources.ClientInfo.Connections
	fading := m.fade.phase != loginFadeNone
	if !fading {
		if m.updateQuitConfirm(ctx) {
			// The confirmation modal is modal: no keyboard or mouse input should
			// leak into the login form, character list, or creation controls.
		} else if m.updateDisconnectDialog(ctx) {
		} else if m.updateCharacterDelete(ctx) {
		} else if m.updatePhaseEscape(ctx, now) {
		} else if m.phase == loginPhaseCreate {
			m.updateCharacterCreateInput(ctx)
		} else if m.phase == loginPhaseCharacter {
			m.updateCharacterSelectInput(ctx)
		} else {
			m.updateFormInput(ctx)
		}

	}
	if fading && m.phase == loginPhaseCharacter {
		m.showCharacterSelectWindow(ctx)
	}

	if len(conns) == 0 {
		return nil, nil
	}

	if ctx.Config.Login.AutoLogin && !m.autoAttempted {
		m.autoAttempted = true
		m.connectAndMaybeLogin(ctx, conns[m.selected], false)
	}

	for _, pkt := range ctx.Network.DrainPackets() {
		glog.Debugf("recv packet 0x%04X len=%d", pkt.ID, len(pkt.Data))
		if handleDisconnectPacket(ctx, &m.disconnectDialog, pkt) {
			continue
		}
		m.packets = append(m.packets, pkt.String())
		if hotkeys, ok, err := network.ParseHotkeyList(pkt); err != nil {
			m.packets = append(m.packets, "parse hotkey list: "+err.Error())
		} else if ok {
			applyHotkeyList(ctx, hotkeys)
			continue
		}
		if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
			m.packets = append(m.packets, "parse chat message: "+err.Error())
		} else if ok {
			addConsoleMessage(&m.console, ctx.Resources, chat)
			continue
		}
		if whisper, ok, err := network.ParseWhisperMessage(pkt); err != nil {
			m.packets = append(m.packets, "parse whisper message: "+err.Error())
		} else if ok {
			addWhisperMessage(&m.console, whisper)
			continue
		}
		if ack, ok, err := network.ParseWhisperAck(pkt); err != nil {
			m.packets = append(m.packets, "parse whisper ack: "+err.Error())
		} else if ok {
			addWhisperAck(&m.console, ctx.Resources, ack)
			continue
		}
		if ack, ok, err := network.ParseWhisperIgnoreAck(pkt); err != nil {
			m.packets = append(m.packets, "parse whisper ignore ack: "+err.Error())
		} else if ok {
			addWhisperIgnoreAck(&m.console, ack)
			continue
		}
		if change, ok, err := network.ParseMapChange(pkt); err != nil {
			m.packets = append(m.packets, "parse ZC_NPCACK_MAPMOVE: "+err.Error())
		} else if ok {
			ctx.World.MapName = change.MapName
			ctx.Session.Zone.MapName = change.MapName
			applyWarpPosition(ctx, change.X, change.Y)
			ctx.Session.Playing = true
			m.status = fmt.Sprintf("map change: %s at %d,%d", change.MapName, change.X, change.Y)
			glog.Debugf("login map change map=%s x=%d y=%d server_move=%t addr=%s port=%d", change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port)
			if change.ServerMove {
				ctx.Session.Zone.Address = change.Address
				ctx.Session.Zone.Port = change.Port
				m.connectMapServer(ctx, network.ZoneServerNotify{
					CharID:  ctx.Session.CharID,
					MapName: change.MapName,
					Address: change.Address,
					Port:    change.Port,
				})
			} else {
				m.startWorldFade(time.Now())
			}
			continue
		}
		if pkt.ID == 0x0069 {
			login, err := network.ParseAccountAcceptLogin(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse AC_ACCEPT_LOGIN: "+err.Error())
			} else {
				ctx.Session.AccountID = login.AccountID
				ctx.Session.AuthCode = login.AuthCode
				ctx.Session.UserLevel = login.UserLevel
				ctx.Session.Sex = login.Sex
				ctx.Session.CharServers = convertCharServers(login.CharServer)
				m.status = fmt.Sprintf("account accepted: aid=%d char_servers=%d", login.AccountID, len(login.CharServer))
				glog.Infof("account accepted aid=%d sex=%d char_servers=%d", login.AccountID, login.Sex, len(login.CharServer))
				for _, server := range login.CharServer {
					m.packets = append(m.packets, fmt.Sprintf("char %s %s:%d users=%d", server.Name, server.Address, server.Port, server.UserCount))
				}
				if len(login.CharServer) > 0 {
					m.connectCharServer(ctx, login.CharServer[0])
				}
			}
		}
		if pkt.ID == 0x006B {
			list, err := network.ParseCharList(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_ENTER: "+err.Error())
			} else {
				ctx.Session.Characters = convertCharacters(list.Characters)
				m.status = fmt.Sprintf("char list: %d characters (%s)", len(list.Characters), list.Layout)
				glog.Debugf("char list characters=%d layout=%s", len(list.Characters), list.Layout)
				for _, character := range list.Characters {
					m.packets = append(m.packets, fmt.Sprintf("char slot=%d gid=%d name=%s lv=%d job=%d", character.Slot, character.ID, character.Name, character.Level, character.Job))
				}
				m.maxSlots = 9
				m.selectedSlot = 0
				if len(list.Characters) > 0 {
					m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
					m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
					if configured := ctx.Config.Login.CharSlot; configured >= 0 {
						m.selectedSlot = clampCharacterSlot(configured, m.maxSlots)
					}
					m.status = "select a character"
				} else {
					m.status = "no characters"
				}
				if m.autoSelectCharacter(ctx) {
					continue
				}
				m.startPhaseFade(loginPhaseCharacter, time.Now())
			}
		}
		if pkt.ID == 0x006D {
			character, err := network.ParseMakeCharacterAccept(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_MAKECHAR: "+err.Error())
			} else {
				created := convertCharacter(character)
				ctx.Session.Characters = upsertCharacter(ctx.Session.Characters, created)
				m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
				m.selectedSlot = int(created.Slot)
				m.create = charCreateState{}
				m.charViews = nil
				m.charViewFailed = nil
				m.charPreviewImages = nil
				m.status = fmt.Sprintf("created character %s", created.Name)
				glog.Infof("character created slot=%d id=%d name=%s", created.Slot, created.ID, created.Name)
				m.startPhaseFade(loginPhaseCharacter, time.Now())
			}
			continue
		}
		if pkt.ID == 0x006E {
			code, err := network.ParseMakeCharacterRefuse(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_REFUSE_MAKECHAR: "+err.Error())
			} else {
				m.status = describeMakeCharacterRefuse(code)
				glog.Debugf("character creation refused code=%d", code)
			}
			continue
		}
		if pkt.ID == 0x006F {
			if err := network.ParseDeleteCharacterAccept(pkt); err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_DELETECHAR: "+err.Error())
			} else {
				m.applyDeleteCharacterAccept(ctx)
			}
			continue
		}
		if pkt.ID == 0x0070 {
			code, err := network.ParseDeleteCharacterRefuse(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_REFUSE_DELETECHAR: "+err.Error())
			} else {
				m.applyDeleteCharacterRefuse(ctx, code)
			}
			continue
		}
		if pkt.ID == 0x0071 {
			zone, err := network.ParseZoneServerNotify(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_NOTIFY_ZONESVR: "+err.Error())
			} else {
				ctx.Session.CharID = zone.CharID
				ctx.Session.Zone = session.ZoneServer{
					Address: zone.Address,
					Port:    zone.Port,
					MapName: zone.MapName,
				}
				ctx.World.MapName = zone.MapName
				m.status = fmt.Sprintf("zone server: %s %s:%d", zone.MapName, zone.Address, zone.Port)
				glog.Debugf("zone server map=%s addr=%s port=%d char_id=%d", zone.MapName, zone.Address, zone.Port, zone.CharID)
				m.connectMapServer(ctx, zone)
			}
		}
		if pkt.ID == 0x0073 || pkt.ID == 0x02EB {
			enter, err := network.ParseMapAcceptEnter(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse ZC_ACCEPT_ENTER: "+err.Error())
			} else {
				applyMapAcceptEnter(ctx, enter)
				sendLessEffectPreference(ctx)
				m.status = fmt.Sprintf("entered map %s at %d,%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				glog.Infof("entered map=%s x=%d y=%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				m.startWorldFade(time.Now())
			}
		}
		if friends, ok, err := network.ParseFriendsList(pkt); err != nil {
			m.packets = append(m.packets, "parse friends list: "+err.Error())
		} else if ok {
			applyFriendsList(ctx, friends)
			continue
		}
		if friendState, ok, err := network.ParseFriendState(pkt); err != nil {
			m.packets = append(m.packets, "parse friend state: "+err.Error())
		} else if ok {
			applyFriendState(ctx, friendState)
			continue
		}
		if cartItems, ok, err := network.ParseCartItemList(pkt); err != nil {
			m.packets = append(m.packets, "parse cart item list: "+err.Error())
		} else if ok {
			glog.Debugf("login cart item list items=%d", len(cartItems))
			applyCartItemList(ctx, cartItems)
			continue
		}
		if cartAmount, ok, err := network.ParseCartAmount(pkt); err != nil {
			m.packets = append(m.packets, "parse cart amount: "+err.Error())
		} else if ok {
			glog.Debugf("login cart amount count=%d/%d weight=%d/%d", cartAmount.Amount, cartAmount.MaxAmount, cartAmount.Weight, cartAmount.MaxWeight)
			applyCartAmount(ctx, cartAmount)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemAdded(pkt); err != nil {
			m.packets = append(m.packets, "parse cart item added: "+err.Error())
		} else if ok {
			glog.Debugf("login cart item added index=%d item=%d amount=%d", cartItem.Index, cartItem.ItemID, cartItem.Amount)
			applyCartItemAdded(ctx, cartItem)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemRemoved(pkt); err != nil {
			m.packets = append(m.packets, "parse cart item removed: "+err.Error())
		} else if ok {
			applyCartItemRemoved(ctx, cartItem)
			continue
		}
		if network.ParseCartClosed(pkt) {
			applyCartClosed(ctx)
			continue
		}
		if board, ok, err := network.ParseVendingBoard(pkt); err != nil {
			m.packets = append(m.packets, "parse vending board: "+err.Error())
		} else if ok {
			applyVendingBoardToWorld(ctx, board)
			glog.Debugf("login vending board actor=%d name=%q", board.OwnerAID, board.Name)
			continue
		}
		if board, ok, err := network.ParseVendingBoardDisappear(pkt); err != nil {
			m.packets = append(m.packets, "parse vending board disappear: "+err.Error())
		} else if ok {
			applyVendingBoardDisappearToWorld(ctx, board)
			glog.Debugf("login vending board removed actor=%d", board.OwnerAID)
			continue
		}
		if board, ok, err := network.ParseChatRoomBoard(pkt); err != nil {
			m.packets = append(m.packets, "parse chat room board: "+err.Error())
		} else if ok {
			applyChatRoomBoardToWorld(ctx, board)
			glog.Debugf("login chat room board actor=%d room=%d title=%q", board.OwnerID, board.RoomID, board.Title)
			continue
		}
		if destroy, ok, err := network.ParseChatRoomDestroy(pkt); err != nil {
			m.packets = append(m.packets, "parse chat room destroy: "+err.Error())
		} else if ok {
			applyChatRoomDestroyToWorld(ctx, destroy)
			glog.Debugf("login chat room board removed room=%d", destroy.RoomID)
			continue
		}
		if entry, ok, err := network.ParseActorEntry(pkt); err != nil {
			m.packets = append(m.packets, "parse actor entry: "+err.Error())
		} else if ok {
			upsertNetworkActor(ctx, entry)
		}
		if len(m.packets) > 8 {
			m.packets = m.packets[len(m.packets)-8:]
		}
	}
	networkErrors := ctx.Network.DrainErrors()
	if handleNetworkDisconnectErrors(ctx, &m.disconnectDialog, networkErrors) {
		return nil, nil
	}
	for _, err := range networkErrors {
		glog.Errorf("network frame error: %v", err)
		m.packets = append(m.packets, "frame error: "+err.Error())
		if len(m.packets) > 8 {
			m.packets = m.packets[len(m.packets)-8:]
		}
	}

	if m.updateFade(ctx, time.Now()) {
		return m.nextWorldMode(ctx, time.Now()), nil
	}
	return nil, nil
}

func (m *LoginMode) Draw(ctx client.Context, screen *render.Frame) {
	m.drawBackground(ctx, screen)
	if m.phase == loginPhaseCreate {
		m.drawCharacterCreate(ctx)
	} else if m.phase == loginPhaseCharacter {
		m.drawCharacterSelect(ctx)
	} else {
		m.drawLoginWindow(ctx)
	}
}

func (m *LoginMode) DrawOverlay(ctx client.Context, screen *render.Frame) {
	now := time.Now()
	m.drawFade(ctx, screen, now)
	if ctx.Config.Render.NoUI {
		return
	}
	m.drawROCursor(screen, ctx, now)
}

func (m *LoginMode) drawROCursor(screen *render.Frame, ctx client.Context, now time.Time) {
	if ctx.Input == nil {
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	m.cursor.draw(screen, ctx, m.cursorAction(ctx), now, 0, 0)
}

func (m *LoginMode) cursorAction(ctx client.Context) int {
	if ctx.Input == nil {
		return cursorActionDefault
	}
	if action, ok := uiCursorAction(ctx); ok {
		return action
	}
	return cursorActionDefault
}

func (m *LoginMode) updatePhaseEscape(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	switch m.phase {
	case loginPhaseCreate:
		m.cancelCharacterCreate(now)
	case loginPhaseCharacter:
		m.startPhaseFade(loginPhaseAccount, now)
		if ctx.Network != nil {
			ctx.Network.Close()
		}
		m.status = "char select cancelled"
	case loginPhaseAccount:
		m.updateLoginWindow(ctx)
		m.openQuitConfirm(ctx)
	}
	return true
}

func (m *LoginMode) updateQuitConfirm(ctx client.Context) bool {
	if !m.quitConfirm.IsOpen() {
		return false
	}
	return m.quitConfirm.Update(ctx)
}

func (m *LoginMode) updateDisconnectDialog(ctx client.Context) bool {
	if !m.disconnectDialog.IsOpen() {
		return false
	}
	return m.disconnectDialog.Update(ctx)
}

func (m *LoginMode) openQuitConfirm(ctx client.Context) {
	m.quitConfirm.Open(ctx, "Exit", "Do you really want to quit?", func() {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
	}, nil)
}

func (m *LoginMode) startPhaseFade(target loginPhase, now time.Time) {
	if m.fade.phase != loginFadeNone && m.fade.hasTarget && m.fade.target == target && !m.fade.enterWorld {
		return
	}
	if m.phase == target && m.fade.phase == loginFadeNone {
		return
	}
	m.fade = loginFadeState{
		phase:     loginFadeOut,
		started:   now,
		target:    target,
		hasTarget: true,
	}
}

func (m *LoginMode) startWorldFade(now time.Time) {
	if m.fade.phase != loginFadeNone && m.fade.enterWorld {
		return
	}
	m.fade = loginFadeState{
		phase:      loginFadeOut,
		started:    now,
		enterWorld: true,
	}
}

func (m *LoginMode) updateFade(ctx client.Context, now time.Time) bool {
	switch m.fade.phase {
	case loginFadeOut:
		if now.Sub(m.fade.started) < loginTransitionDuration {
			return false
		}
		if m.fade.enterWorld {
			return true
		}
		if m.fade.hasTarget {
			m.phase = m.fade.target
			m.publishPhaseWindow(ctx)
		}
		m.fade = loginFadeState{phase: loginFadeIn, started: now}
	case loginFadeIn:
		if now.Sub(m.fade.started) >= loginTransitionDuration {
			m.fade = loginFadeState{}
		}
	}
	return false
}

func (m *LoginMode) publishPhaseWindow(ctx client.Context) {
	m.clearLoginWindows(ctx)
	switch m.phase {
	case loginPhaseCharacter:
		m.showCharacterSelectWindow(ctx)
	case loginPhaseAccount:
		m.updateLoginWindow(ctx)
	case loginPhaseCreate:
		m.showCharacterCreateWindow(ctx)
	}
}

func (m *LoginMode) clearLoginWindows(ctx client.Context) {
	m.loginWindow = nil
	m.charSelectWindow = nil
	m.charCreateWindow = nil
	if ctx.UIManager != nil {
		ctx.UIManager.Clear()
	}
}

func (m *LoginMode) fadeAlpha(now time.Time) uint8 {
	if m.fade.started.IsZero() {
		return 0
	}
	switch m.fade.phase {
	case loginFadeOut:
		return clampColor(255 * clampUnit(float64(now.Sub(m.fade.started))/float64(loginTransitionDuration)))
	case loginFadeIn:
		return clampColor(255 * (1 - clampUnit(float64(now.Sub(m.fade.started))/float64(loginTransitionDuration))))
	default:
		return 0
	}
}

func (m *LoginMode) drawFade(ctx client.Context, screen *render.Frame, now time.Time) {
	alpha := m.fadeAlpha(now)
	if alpha == 0 {
		return
	}
	width, height := ctx.ScreenSize()
	render.DrawRect(screen, 0, 0, float64(width), float64(height), color.RGBA{A: alpha})
}

func (m *LoginMode) nextWorldMode(ctx client.Context, now time.Time) *WorldMode {
	m.clearLoginWindows(ctx)
	m.quitConfirm = gameui.ConfirmModal{}
	next := NewWorldMode()
	next.ui.console = m.console
	next.startMapFadeIn(now)
	return next
}

func (m *LoginMode) moveSelectedSlot(delta int) {
	m.selectedSlot = clampCharacterSlot(m.selectedSlot+delta, m.maxSlots)
}

func (m *LoginMode) submitSelectedCharacter(ctx client.Context) {
	character, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot)
	if !ok {
		m.status = "empty character slot"
		return
	}
	if ctx.Network == nil {
		m.status = "select character failed: not connected"
		return
	}
	if err := ctx.Network.SendSelectCharacter(character.Slot); err != nil {
		m.status = "select character failed: " + err.Error()
		return
	}
	m.playConfirmSFX(ctx)
	ctx.Session.SelectCharacter(character)
	m.status = fmt.Sprintf("selected character %s", character.Name)
}

func (m *LoginMode) drawBackground(ctx client.Context, screen *render.Frame) {
	screen.Fill(color.Black)
	width, height := ctx.ScreenSize()
	if width <= 0 || height <= 0 {
		return
	}
	if len(m.bgTiles) == 12 {
		cellW := float64(width) / 4
		cellH := float64(height) / 3
		for i, tile := range m.bgTiles {
			if tile == nil {
				continue
			}
			b := tile.Bounds()
			if b.Dx() <= 0 || b.Dy() <= 0 {
				continue
			}
			var opts render.DrawImageOptions
			opts.GeoM.Scale(cellW/float64(b.Dx()), cellH/float64(b.Dy()))
			opts.GeoM.Translate(float64(i%4)*cellW, float64(i/4)*cellH)
			opts.Filter = render.FilterLinear
			screen.DrawImage(tile, &opts)
		}
		return
	}
	if m.background == nil {
		render.DrawRect(screen, 0, 0, float64(width), float64(height), color.RGBA{R: 10, G: 13, B: 22, A: 255})
		return
	}
	b := m.background.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Scale(float64(width)/float64(b.Dx()), float64(height)/float64(b.Dy()))
	opts.Filter = render.FilterLinear
	screen.DrawImage(m.background, &opts)
}

func (m *LoginMode) loadBackground(ctx client.Context) {
	if m.bgLoaded {
		return
	}
	m.bgLoaded = true
	for _, set := range loginBackgroundSets(ctx.Config.Packet.ClientDate) {
		if len(set) == 1 {
			img, source, ok := loadLoginBackgroundImage(ctx.Resources, set[0])
			if ok {
				m.background = img
				m.bgSource = source
				return
			}
			continue
		}
		tiles := make([]*render.Image, 0, len(set))
		sources := make([]string, 0, len(set))
		ok := true
		for _, name := range set {
			img, source, loaded := loadLoginBackgroundImage(ctx.Resources, name)
			if !loaded {
				ok = false
				break
			}
			tiles = append(tiles, img)
			sources = append(sources, source)
		}
		if ok {
			m.bgTiles = tiles
			m.bgSource = fmt.Sprintf("%d login tiles", len(sources))
			return
		}
	}
	m.bgSource = "fallback"
}

func (m *LoginMode) playLoginBGM(ctx client.Context) {
	if m.bgmStarted || ctx.Audio == nil {
		return
	}
	m.bgmStarted = true
	_ = ctx.Audio.Play("01.mp3")
}

func (m *LoginMode) playConfirmSFX(ctx client.Context) {
	if ctx.Audio == nil {
		return
	}
	source, err := ctx.Audio.PlaySFX(loginConfirmSFX)
	if err == nil {
		if source != "" {
			glog.Debugf("sfx playing path=%s source=%s", loginConfirmSFX, source)
		}
	}
}

func loadLoginBackgroundImage(manager *res.Manager, name string) (*render.Image, string, bool) {
	for _, candidate := range loginInterfaceCandidates(name) {
		img, source, err := res.LoadImageExact(manager, []string{candidate})
		if err == nil {
			return render.NewImageFromImage(img), source, true
		}
	}
	return nil, "", false
}

func loginBackgroundSets(clientDate int) [][]string {
	tiles2018 := []string{
		"t_배경1-1.bmp", "t_배경1-2.bmp", "t_배경1-3.bmp", "t_배경1-4.bmp",
		"t_배경2-1.bmp", "t_배경2-2.bmp", "t_배경2-3.bmp", "t_배경2-4.bmp",
		"t_배경3-1.bmp", "t_배경3-2.bmp", "t_배경3-3.bmp", "t_배경3-4.bmp",
	}
	sets := make([][]string, 0, 3)
	if clientDate >= 20221207 {
		sets = append(sets, []string{"t_login.jpg"})
	}
	if clientDate >= 20181114 {
		sets = append(sets, tiles2018)
	}
	sets = append(sets, []string{"bgi_temp.bmp"}, []string{"t_login.jpg"}, tiles2018)
	return sets
}

func loginInterfaceCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	const ui = "data\\texture\\유저인터페이스\\"
	candidates := []string{
		ui + name,
		strings.ReplaceAll(ui, "\\", "/") + name,
		"texture\\유저인터페이스\\" + name,
		"data\\texture\\interface\\" + name,
		"data/texture/interface/" + name,
		name,
	}
	return uniqueLoginStrings(candidates)
}

func uniqueLoginStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (m *LoginMode) connectAndMaybeLogin(ctx client.Context, conn res.Connection, userConfirmed bool) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, conn.Address, conn.Port)
	cancel()
	if err != nil {
		m.status = err.Error()
		openConnectionFailedDialog(ctx, &m.disconnectDialog)
		return
	}

	m.status = ctx.Network.Status()
	glog.Infof("connected login server %s:%d", conn.Address, conn.Port)
	if strings.TrimSpace(m.username) == "" && m.password == "" {
		return
	}

	err = ctx.Network.SendAccountLogin(
		m.username,
		m.password,
		uint32(conn.Version),
		0,
	)
	if err != nil {
		m.status = "login packet failed: " + err.Error()
		return
	}
	if userConfirmed {
		m.playConfirmSFX(ctx)
	}
	m.status = "CA_LOGIN sent"
	glog.Debugf("sent CA_LOGIN user=%s version=%d", m.username, conn.Version)
}

func (m *LoginMode) connectCharServer(ctx client.Context, server network.CharServer) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, server.Address, int(server.Port))
	cancel()
	if err != nil {
		m.status = "char connect failed: " + err.Error()
		openConnectionFailedDialog(ctx, &m.disconnectDialog)
		return
	}

	err = ctx.Network.SendCharServerEnter(ctx.Session.AccountID, ctx.Session.AuthCode, ctx.Session.UserLevel, ctx.Session.Sex)
	if err != nil {
		m.status = "CA_ENTER failed: " + err.Error()
		return
	}
	m.status = "CA_ENTER sent to char server"
	glog.Debugf("sent CA_ENTER account_id=%d addr=%s port=%d", ctx.Session.AccountID, server.Address, server.Port)
}

func (m *LoginMode) connectMapServer(ctx client.Context, zone network.ZoneServerNotify) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, zone.Address, int(zone.Port))
	cancel()
	if err != nil {
		m.status = "map connect failed: " + err.Error()
		openConnectionFailedDialog(ctx, &m.disconnectDialog)
		return
	}

	err = ctx.Network.SendMapServerEnter(ctx.Session.AccountID, zone.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex)
	if err != nil {
		m.status = "CZ_ENTER2 failed: " + err.Error()
		return
	}
	m.status = "CZ_ENTER2 sent to map server"
	glog.Debugf("sent CZ_ENTER2 account_id=%d char_id=%d addr=%s port=%d", ctx.Session.AccountID, zone.CharID, zone.Address, zone.Port)
}

func convertCharServers(servers []network.CharServer) []session.CharServer {
	out := make([]session.CharServer, 0, len(servers))
	for _, server := range servers {
		out = append(out, session.CharServer{
			Address:   server.Address,
			Port:      server.Port,
			Name:      server.Name,
			UserCount: server.UserCount,
			State:     server.State,
			Property:  server.Property,
		})
	}
	return out
}

func convertCharacters(characters []network.Character) []session.Character {
	out := make([]session.Character, 0, len(characters))
	for _, character := range characters {
		out = append(out, convertCharacter(character))
	}
	return out
}

func convertCharacter(character network.Character) session.Character {
	return session.Character{
		ID:        character.ID,
		Money:     character.Money,
		Name:      character.Name,
		Slot:      character.Slot,
		Level:     character.Level,
		JobLevel:  character.JobLevel,
		Job:       character.Job,
		HP:        character.HP,
		MaxHP:     character.MaxHP,
		SP:        character.SP,
		MaxSP:     character.MaxSP,
		Str:       character.Str,
		Agi:       character.Agi,
		Vit:       character.Vit,
		Int:       character.Int,
		Dex:       character.Dex,
		Luk:       character.Luk,
		Hair:      character.Hair,
		HairColor: character.HairColor,
		HeadPal:   character.HeadPal,
		BodyPal:   character.BodyPal,
		Weapon:    character.Weapon,
		Shield:    character.Shield,
		HeadTop:   character.HeadTop,
		HeadMid:   character.HeadMid,
		HeadLow:   character.HeadLow,
		Option:    character.Option,
	}
}
