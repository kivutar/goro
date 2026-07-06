package game

import (
	"context"
	"fmt"
	uiwidget "github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"image"
	"image/color"
	"log"
	"strings"
	"time"

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
	publishedUI       uiwidget.Widget
	create            charCreateState
	cursor            roCursorState
	quitConfirm       gameui.ConfirmModal
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

const (
	loginTransitionDuration    = 500 * time.Millisecond
	charSelectPreviewDirection = 4
	charSelectPreviewScale     = 0.92
	charSelectPreviewFeetLift  = 10
	charCreateNameMinBytes     = 4
	charCreateNameMaxBytes     = 23
	charCreateMinHairStyle     = 2
	charCreateMaxHairStyle     = 23
)

type charCreateState struct {
	slot            int
	name            string
	stats           [6]uint8
	hairStyle       int
	hairColor       int
	direction       int
	preview         *humanoidSpriteView
	previewKey      charCreatePreviewKey
	previewFailed   bool
	previewImage    image.Image
	previewImageKey charCreatePreviewKey
}

type charCreatePreviewKey struct {
	sex       byte
	hairStyle int
	hairColor int
}

const (
	createStatStr = iota
	createStatAgi
	createStatVit
	createStatInt
	createStatDex
	createStatLuk
	createStatCount
)

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
		m.publishCharacterSelectWindow(ctx)
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
		m.publishCharacterSelectWindow(ctx)
	}

	if len(conns) == 0 {
		return nil, nil
	}

	if ctx.Config.Login.AutoLogin && !m.autoAttempted {
		m.autoAttempted = true
		m.connectAndMaybeLogin(ctx, conns[m.selected], false)
	}

	for _, pkt := range ctx.Network.DrainPackets() {
		log.Printf("recv packet 0x%04X len=%d", pkt.ID, len(pkt.Data))
		m.packets = append(m.packets, pkt.String())
		if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
			m.packets = append(m.packets, "parse chat message: "+err.Error())
		} else if ok {
			addConsoleMessage(&m.console, ctx.Resources, chat)
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
			log.Printf("login map change map=%s x=%d y=%d server_move=%t addr=%s port=%d", change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port)
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
				log.Printf("account accepted aid=%d sex=%d char_servers=%d", login.AccountID, login.Sex, len(login.CharServer))
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
				log.Printf("char list characters=%d layout=%s", len(list.Characters), list.Layout)
				for _, character := range list.Characters {
					m.packets = append(m.packets, fmt.Sprintf("char slot=%d gid=%d name=%s lv=%d job=%d", character.Slot, character.ID, character.Name, character.Level, character.Job))
				}
				m.maxSlots = 9
				m.selectedSlot = 0
				if len(list.Characters) > 0 {
					m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
					m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
					m.status = "select a character"
				} else {
					m.status = "no characters"
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
				log.Printf("character created slot=%d id=%d name=%s", created.Slot, created.ID, created.Name)
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
				log.Printf("character creation refused code=%d", code)
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
				log.Printf("zone server map=%s addr=%s port=%d char_id=%d", zone.MapName, zone.Address, zone.Port, zone.CharID)
				m.connectMapServer(ctx, zone)
			}
		}
		if pkt.ID == 0x0073 || pkt.ID == 0x02EB {
			enter, err := network.ParseMapAcceptEnter(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse ZC_ACCEPT_ENTER: "+err.Error())
			} else {
				applyMapAcceptEnter(ctx, enter)
				m.status = fmt.Sprintf("entered map %s at %d,%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				log.Printf("entered map=%s x=%d y=%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				m.startWorldFade(time.Now())
			}
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
	for _, err := range ctx.Network.DrainErrors() {
		log.Printf("network frame error: %v", err)
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

func (m *LoginMode) Draw(ctx client.Context, screen *render.Image) {
	m.drawBackground(ctx, screen)
	if m.phase == loginPhaseCreate {
		m.charSelectWindow = nil
		m.drawCharacterCreate(ctx)
	} else if m.phase == loginPhaseCharacter {
		m.charCreateWindow = nil
		m.drawCharacterSelect(ctx)
	} else {
		m.charSelectWindow = nil
		m.charCreateWindow = nil
		m.drawLoginWindow(ctx)
	}
}

func (m *LoginMode) DrawOverlay(ctx client.Context, screen *render.Image) {
	now := time.Now()
	m.drawFade(ctx, screen, now)
	m.drawROCursor(screen, ctx, now)
}

func (m *LoginMode) drawROCursor(screen *render.Image, ctx client.Context, now time.Time) {
	if ctx.Input == nil {
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	m.cursor.draw(screen, ctx, m.cursorAction(ctx), now)
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
	if ctx.Input == nil || !ctx.Input.JustPressed(render.KeyEscape) {
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

func (m *LoginMode) openQuitConfirm(ctx client.Context) {
	m.quitConfirm.Open(ctx, "Exit", "Do you really want to quit?", func() {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
	}, nil)
}

func (m *LoginMode) updateFormInput(ctx client.Context) {
	m.updateLoginWindow(ctx)
	if m.loginWindow != nil {
		m.loginWindow.Update(ctx)
		m.publishUI(ctx, m.loginWindow.Widget())
	}
}

func (m *LoginMode) updateCharacterSelectInput(ctx client.Context) {
	if ctx.Input != nil && ctx.Input.JustPressed(render.KeyEscape) {
		m.cancelCharacterSelect(ctx)
		return
	}
	if ctx.Input != nil {
		if ctx.Input.JustPressed(render.KeyArrowLeft) {
			m.moveSelectedSlot(-1)
		}
		if ctx.Input.JustPressed(render.KeyArrowRight) {
			m.moveSelectedSlot(1)
		}
		if ctx.Input.JustPressed(render.KeyEnter) {
			m.submitSelectedCharacter(ctx)
		}
	}
	m.publishCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.charSelectWindow.Update(ctx)
		m.publishUI(ctx, m.charSelectWindow.Widget())
	}
}

func (m *LoginMode) updateCharacterSelectWindow(ctx client.Context) {
	opts := gameui.CharacterSelectWindowOptions{
		SelectedSlot: m.selectedSlot,
		MaxSlots:     m.maxSlots,
	}
	if ctx.Session != nil {
		opts.Characters = ctx.Session.Characters
	}
	opts.PreviewImages = m.characterSelectPreviewImages(ctx, opts)
	callbacks := gameui.CharacterSelectWindowCallbacks{
		OnSelectSlot: func(slot int) {
			m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
		},
		OnActivateSlot: func(slot int) {
			if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
				m.submitSelectedCharacter(ctx)
			} else {
				m.openCharacterCreate(ctx, slot, time.Now())
			}
		},
		OnPreviousPage: func() {
			m.moveSelectedSlot(-3)
		},
		OnNextPage: func() {
			m.moveSelectedSlot(3)
		},
		OnMake: func() {
			slot := m.selectedSlot
			if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
				if empty, hasEmpty := firstEmptyCharacterSlot(ctx.Session.Characters, m.maxSlots); hasEmpty {
					slot = empty
				}
			}
			m.openCharacterCreate(ctx, slot, time.Now())
		},
		OnOK: func() {
			m.submitSelectedCharacter(ctx)
		},
		OnCancel: func() {
			m.cancelCharacterSelect(ctx)
		},
		OnDelete: func() {
			m.status = "character deletion is not implemented yet"
		},
	}
	if m.charSelectWindow == nil {
		m.charSelectWindow = gameui.NewCharacterSelectWindow(ctx, opts, callbacks)
		return
	}
	m.charSelectWindow.SetOptions(ctx, opts)
}

func (m *LoginMode) characterSelectPreviewImages(ctx client.Context, opts gameui.CharacterSelectWindowOptions) map[int]image.Image {
	images := make(map[int]image.Image, 3)
	pageStart := gameui.CharacterSelectPage(opts.SelectedSlot) * 3
	for localSlot := 0; localSlot < 3; localSlot++ {
		slot := pageStart + localSlot
		character, ok := characterBySlot(opts.Characters, slot)
		if !ok {
			continue
		}
		if image := m.characterPreviewImage(ctx, character); image != nil {
			images[slot] = image
		}
	}
	return images
}

func (m *LoginMode) publishCharacterSelectWindow(ctx client.Context) {
	m.updateCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.publishUI(ctx, m.charSelectWindow.Widget())
	}
}

func (m *LoginMode) cancelCharacterSelect(ctx client.Context) {
	m.charSelectWindow = nil
	m.startPhaseFade(loginPhaseAccount, time.Now())
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	m.status = "char select cancelled"
}

func (m *LoginMode) prepareCharacterSelectFromSession(ctx client.Context) {
	m.maxSlots = 9
	m.selectedSlot = 0
	if ctx.Session == nil || len(ctx.Session.Characters) == 0 {
		return
	}
	m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
	}
}

func (m *LoginMode) reconnectCharacterServer(ctx client.Context) {
	if ctx.Network == nil || ctx.Session == nil || len(ctx.Session.CharServers) == 0 {
		return
	}
	server := ctx.Session.CharServers[0]
	m.connectCharServer(ctx, network.CharServer{
		Address:   server.Address,
		Port:      server.Port,
		Name:      server.Name,
		UserCount: server.UserCount,
		State:     server.State,
		Property:  server.Property,
	})
}

func (m *LoginMode) updateCharacterCreateInput(ctx client.Context) {
	m.publishCharacterCreateWindow(ctx)
	if m.charCreateWindow != nil {
		m.charCreateWindow.Update(ctx)
		m.publishUI(ctx, m.charCreateWindow.Widget())
	}
}

func (m *LoginMode) openCharacterCreate(ctx client.Context, slot int, now time.Time) {
	slot = clampCharacterSlot(slot, m.maxSlots)
	if _, occupied := characterBySlot(ctx.Session.Characters, slot); occupied {
		empty, ok := firstEmptyCharacterSlot(ctx.Session.Characters, m.maxSlots)
		if !ok {
			m.status = "no empty character slots"
			return
		}
		slot = empty
	}
	m.create = defaultCharCreateState(slot)
	m.charCreateWindow = nil
	m.status = "create a character"
	m.playConfirmSFX(ctx)
	m.startPhaseFade(loginPhaseCreate, now)
}

func defaultCharCreateState(slot int) charCreateState {
	return charCreateState{
		slot:      slot,
		stats:     [6]uint8{5, 5, 5, 5, 5, 5},
		hairStyle: 2,
		hairColor: 0,
		direction: charSelectPreviewDirection,
	}
}

func (m *LoginMode) cancelCharacterCreate(now time.Time) {
	m.create = charCreateState{}
	m.charCreateWindow = nil
	m.status = "select a character"
	m.startPhaseFade(loginPhaseCharacter, now)
}

func (m *LoginMode) submitCharacterCreate(ctx client.Context) {
	name := strings.TrimSpace(m.create.name)
	if name == "" {
		m.status = "enter a character name"
		return
	}
	if len([]byte(name)) < charCreateNameMinBytes {
		m.status = "name must be at least 4 characters"
		return
	}
	packet := network.MakeCharacter{
		Name:      name,
		Str:       m.create.stats[createStatStr],
		Agi:       m.create.stats[createStatAgi],
		Vit:       m.create.stats[createStatVit],
		Int:       m.create.stats[createStatInt],
		Dex:       m.create.stats[createStatDex],
		Luk:       m.create.stats[createStatLuk],
		Slot:      uint8(m.create.slot),
		HairColor: uint16(m.create.hairColor),
		HairStyle: uint16(m.create.hairStyle),
	}
	if err := ctx.Network.SendMakeCharacter(packet); err != nil {
		m.status = "create character failed: " + err.Error()
		return
	}
	m.playConfirmSFX(ctx)
	m.status = "creating character..."
}

func (m *LoginMode) changeCreateHairStyle(delta int) {
	m.create.hairStyle += delta
	if m.create.hairStyle < charCreateMinHairStyle {
		m.create.hairStyle = charCreateMaxHairStyle
	}
	if m.create.hairStyle > charCreateMaxHairStyle {
		m.create.hairStyle = charCreateMinHairStyle
	}
	m.create.preview = nil
	m.create.previewFailed = false
	m.create.previewImage = nil
}

func (m *LoginMode) changeCreateHairColor() {
	m.create.hairColor = (m.create.hairColor + 1) % 10
	m.create.preview = nil
	m.create.previewFailed = false
	m.create.previewImage = nil
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
	switch m.phase {
	case loginPhaseCharacter:
		m.loginWindow = nil
		m.charCreateWindow = nil
		m.publishCharacterSelectWindow(ctx)
	case loginPhaseAccount:
		m.charSelectWindow = nil
		m.charCreateWindow = nil
		m.updateLoginWindow(ctx)
	case loginPhaseCreate:
		m.loginWindow = nil
		m.charSelectWindow = nil
		m.publishCharacterCreateWindow(ctx)
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

func (m *LoginMode) drawFade(ctx client.Context, screen *render.Image, now time.Time) {
	alpha := m.fadeAlpha(now)
	if alpha == 0 {
		return
	}
	width, height := ctx.ScreenSize()
	render.DrawRect(screen, 0, 0, float64(width), float64(height), color.RGBA{A: alpha})
}

func (m *LoginMode) nextWorldMode(ctx client.Context, now time.Time) *WorldMode {
	if ctx.UIManager != nil {
		ctx.UIManager.Clear()
	}
	m.publishedUI = nil
	m.loginWindow = nil
	m.charSelectWindow = nil
	m.charCreateWindow = nil
	m.quitConfirm = gameui.ConfirmModal{}
	next := NewWorldMode()
	next.console = m.console
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
	if err := ctx.Network.SendSelectCharacter(character.Slot); err != nil {
		m.status = "select character failed: " + err.Error()
		return
	}
	m.playConfirmSFX(ctx)
	ctx.Session.CharID = character.ID
	setSelectedCharacter(ctx.Session, character)
	m.status = fmt.Sprintf("selected character %s", character.Name)
}

func (m *LoginMode) drawBackground(ctx client.Context, screen *render.Image) {
	clear(screen)
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

func (m *LoginMode) drawLoginWindow(ctx client.Context) {
	if m.loginWindow == nil {
		m.updateLoginWindow(ctx)
	}
	if m.loginWindow != nil && ctx.UIManager != nil {
		m.publishUI(ctx, m.loginWindow.Widget())
	}
}

func (m *LoginMode) updateLoginWindow(ctx client.Context) {
	if m.loginWindow == nil {
		m.loginWindow = gameui.NewLoginWindow(ctx, m.username, m.password, gameui.LoginWindowCallbacks{
			OnSubmit: func() {
				m.username = m.loginWindow.Username
				m.password = m.loginWindow.Password
				if len(ctx.Resources.ClientInfo.Connections) > 0 {
					m.connectAndMaybeLogin(ctx, ctx.Resources.ClientInfo.Connections[0], true)
				}
			},
		})
		m.publishUI(ctx, m.loginWindow.Widget())
		return
	}
	m.loginWindow.SetContext(ctx)
	m.username = m.loginWindow.Username
	m.password = m.loginWindow.Password
	m.publishUI(ctx, m.loginWindow.Widget())
}

func (m *LoginMode) publishUI(ctx client.Context, root uiwidget.Widget) {
	if ctx.UIManager == nil || root == nil || root == m.publishedUI {
		return
	}
	if m.publishedUI != nil {
		ctx.UIManager.RemoveOverlay(m.publishedUI)
	} else {
		ctx.UIManager.Clear()
	}
	ctx.UIManager.AddOverlay(root)
	m.publishedUI = root
}

func (m *LoginMode) drawCharacterSelect(ctx client.Context) {
	m.publishCharacterSelectWindow(ctx)
}

func (m *LoginMode) drawCharacterCreate(ctx client.Context) {
	m.publishCharacterCreateWindow(ctx)
}

func (m *LoginMode) publishCharacterCreateWindow(ctx client.Context) {
	m.updateCharacterCreateWindow(ctx)
	if m.charCreateWindow != nil {
		m.publishUI(ctx, m.charCreateWindow.Widget())
	}
}

func (m *LoginMode) updateCharacterCreateWindow(ctx client.Context) {
	opts := gameui.CharacterCreateWindowOptions{
		Name:    m.create.name,
		Stats:   m.create.stats,
		Preview: m.characterCreatePreviewImage(ctx),
	}
	callbacks := gameui.CharacterCreateWindowCallbacks{
		OnNameChange: func(value string) {
			m.create.name = appendCharacterNameInput("", value, charCreateNameMaxBytes)
		},
		OnSubmit: func() {
			m.submitCharacterCreate(ctx)
		},
		OnCancel: func() {
			m.cancelCharacterCreate(time.Now())
		},
		OnHairPrev: func() {
			m.changeCreateHairStyle(-1)
		},
		OnHairNext: func() {
			m.changeCreateHairStyle(1)
		},
		OnHairColor: func() {
			m.changeCreateHairColor()
		},
		OnStat: func(stat int) {
			if !bumpCreateStat(&m.create.stats, stat) {
				m.status = "stat limit reached"
			}
		},
	}
	if m.charCreateWindow == nil {
		m.charCreateWindow = gameui.NewCharacterCreateWindow(ctx, opts, callbacks)
		return
	}
	m.charCreateWindow.SetOptions(ctx, opts)
}

func (m *LoginMode) characterCreatePreviewImage(ctx client.Context) image.Image {
	const previewW, previewH = 142, 166
	key := charCreatePreviewKey{sex: ctx.Session.Sex, hairStyle: m.create.hairStyle, hairColor: m.create.hairColor}
	if m.create.previewImage != nil && m.create.previewImageKey == key {
		return m.create.previewImage
	}
	img := render.NewImage(previewW, previewH)
	view := m.characterCreatePreviewView(ctx)
	if view == nil {
		render.DebugPrintAtColor(img, "?", previewW/2-3, previewH/2, gameui.MutedTextColor)
		m.create.previewImage = img.RGBA()
		m.create.previewImageKey = key
		return m.create.previewImage
	}
	billboard, ok := humanoidBillboardForState(view, spriteState{
		actionFamily: spriteActionIdle,
		direction:    m.create.direction,
		started:      time.Now(),
		loopIdle:     true,
	}, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		m.create.previewImage = img.RGBA()
		m.create.previewImageKey = key
		return m.create.previewImage
	}
	const scale = 1.08
	bounds := billboard.image.Bounds()
	var opts render.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(
		float64(previewW)/2-float64(bounds.Dx())*scale/2,
		float64(previewH)/2-float64(bounds.Dy())*scale/2,
	)
	opts.Filter = spriteDrawFilter()
	img.DrawImage(billboard.image, &opts)
	m.create.previewImage = img.RGBA()
	m.create.previewImageKey = key
	return m.create.previewImage
}

func (m *LoginMode) characterCreatePreviewView(ctx client.Context) *humanoidSpriteView {
	key := charCreatePreviewKey{sex: ctx.Session.Sex, hairStyle: m.create.hairStyle, hairColor: m.create.hairColor}
	if m.create.preview != nil && m.create.previewKey == key {
		return m.create.preview
	}
	if m.create.previewFailed && m.create.previewKey == key {
		return nil
	}
	character := session.Character{
		ID:        1,
		Name:      m.create.name,
		Job:       0,
		Hair:      int16(m.create.hairStyle),
		HairColor: uint8(m.create.hairColor),
	}
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex)
	m.create.previewKey = key
	if view == nil {
		m.create.previewFailed = true
		log.Printf("char create sprite resources hair=%d color=%d sex=%d %s", m.create.hairStyle, m.create.hairColor, ctx.Session.Sex, status)
		return nil
	}
	m.create.preview = view
	m.create.previewFailed = false
	return view
}

func (m *LoginMode) drawCharacterPreview(screen *render.Image, ctx client.Context, character session.Character, centerX, feetY int) {
	view := m.characterPreviewView(ctx, character)
	if view == nil {
		render.DebugPrintAtColor(screen, "?", centerX-3, feetY-72, color.RGBA{R: 220, G: 220, B: 220, A: 255})
		return
	}
	billboard, ok := humanoidBillboardForState(view, spriteState{
		actionFamily: spriteActionIdle,
		direction:    charSelectPreviewDirection,
		started:      time.Now(),
		loopIdle:     true,
	}, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		return
	}
	var opts render.DrawImageOptions
	scale := charSelectPreviewScale
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(centerX)-billboard.anchorX*scale, float64(feetY)-billboard.anchorY*scale)
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(billboard.image, &opts)
}

func (m *LoginMode) characterPreviewImage(ctx client.Context, character session.Character) image.Image {
	if character.ID == 0 {
		return nil
	}
	if m.charPreviewImages == nil {
		m.charPreviewImages = make(map[uint32]image.Image)
	}
	if img := m.charPreviewImages[character.ID]; img != nil {
		return img
	}
	img := render.NewImage(139, 144)
	m.drawCharacterPreview(img, ctx, character, 139/2, 144-15-charSelectPreviewFeetLift)
	m.charPreviewImages[character.ID] = img.RGBA()
	return m.charPreviewImages[character.ID]
}

func (m *LoginMode) characterPreviewView(ctx client.Context, character session.Character) *humanoidSpriteView {
	if character.ID == 0 {
		return nil
	}
	if _, failed := m.charViewFailed[character.ID]; failed {
		return nil
	}
	if m.charViews == nil {
		m.charViews = make(map[uint32]*humanoidSpriteView)
	}
	if view := m.charViews[character.ID]; view != nil {
		return view
	}
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex)
	if view == nil {
		if m.charViewFailed == nil {
			m.charViewFailed = make(map[uint32]struct{})
		}
		m.charViewFailed[character.ID] = struct{}{}
		log.Printf("char select sprite resources char_id=%d name=%s job=%d %s", character.ID, character.Name, character.Job, status)
		return nil
	}
	m.charViews[character.ID] = view
	return view
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

func (m *LoginMode) loadCharacterSelectSkin(ctx client.Context) {
	if m.charWindow == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/win_select.bmp"); ok {
			m.charWindow = img
		}
	}
	if m.charBox == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/box_select.bmp"); ok {
			m.charBox = img
		}
	}
}

func (m *LoginMode) playLoginBGM(ctx client.Context) {
	if m.bgmStarted || ctx.Audio == nil {
		return
	}
	m.bgmStarted = true
	for _, path := range []string{"01.mp3", "BGM\\01.mp3", "bgm\\01.mp3"} {
		if err := ctx.Audio.Play(path); err == nil {
			return
		}
	}
}

func (m *LoginMode) playConfirmSFX(ctx client.Context) {
	playLoginSFXFirst(ctx, loginConfirmSFXCandidates()...)
}

func playLoginSFXFirst(ctx client.Context, paths ...string) {
	if ctx.Audio == nil {
		return
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		source, err := ctx.Audio.PlaySFX(path)
		if err == nil {
			if source != "" {
				log.Printf("sfx playing path=%s source=%s", path, source)
			}
			return
		}
	}
}

func loginConfirmSFXCandidates() []string {
	const koreanButtonSound = "\xB9\xF6\xC6\xB0\xBC\xD2\xB8\xAE.wav"
	return []string{
		koreanButtonSound,
		"wav\\" + koreanButtonSound,
		"click.wav",
		"button.wav",
		"btnok.wav",
		"btn_ok.wav",
		"ok.wav",
		"enter.wav",
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

func charSelectMaxSlots(characters []session.Character) int {
	maxSlots := 9
	for _, character := range characters {
		if int(character.Slot)+1 > maxSlots {
			maxSlots = int(character.Slot) + 1
		}
	}
	if maxSlots%3 != 0 {
		maxSlots += 3 - maxSlots%3
	}
	return maxSlots
}

func firstOccupiedCharacterSlot(characters []session.Character) int {
	if len(characters) == 0 {
		return 0
	}
	slot := int(characters[0].Slot)
	for _, character := range characters[1:] {
		if int(character.Slot) < slot {
			slot = int(character.Slot)
		}
	}
	return slot
}

func firstEmptyCharacterSlot(characters []session.Character, maxSlots int) (int, bool) {
	if maxSlots <= 0 {
		maxSlots = 9
	}
	occupied := make(map[int]struct{}, len(characters))
	for _, character := range characters {
		occupied[int(character.Slot)] = struct{}{}
	}
	for slot := 0; slot < maxSlots; slot++ {
		if _, ok := occupied[slot]; !ok {
			return slot, true
		}
	}
	return 0, false
}

func clampCharacterSlot(slot, maxSlots int) int {
	if maxSlots <= 0 {
		maxSlots = 1
	}
	if slot < 0 {
		return 0
	}
	if slot >= maxSlots {
		return maxSlots - 1
	}
	return slot
}

func characterBySlot(characters []session.Character, slot int) (session.Character, bool) {
	for _, character := range characters {
		if int(character.Slot) == slot {
			return character, true
		}
	}
	return session.Character{}, false
}

func upsertCharacter(characters []session.Character, character session.Character) []session.Character {
	for i := range characters {
		if characters[i].ID == character.ID || characters[i].Slot == character.Slot {
			characters[i] = character
			return characters
		}
	}
	return append(characters, character)
}

func pairedCreateStat(stat int) int {
	switch stat {
	case createStatStr:
		return createStatInt
	case createStatInt:
		return createStatStr
	case createStatAgi:
		return createStatLuk
	case createStatLuk:
		return createStatAgi
	case createStatVit:
		return createStatDex
	case createStatDex:
		return createStatVit
	default:
		return -1
	}
}

func bumpCreateStat(stats *[createStatCount]uint8, stat int) bool {
	if stats == nil || stat < 0 || stat >= createStatCount {
		return false
	}
	pair := pairedCreateStat(stat)
	if pair < 0 || (*stats)[stat] >= 9 || (*stats)[pair] <= 1 {
		return false
	}
	(*stats)[stat]++
	(*stats)[pair]--
	return true
}

func appendCharacterNameInput(current, input string, maxBytes int) string {
	if maxBytes <= 0 {
		return current
	}
	out := current
	for _, r := range input {
		if r < 32 || r == 127 {
			continue
		}
		next := out + string(r)
		if len([]byte(next)) > maxBytes {
			break
		}
		out = next
	}
	return out
}

func describeMakeCharacterRefuse(code uint8) string {
	switch code {
	case 0:
		return "character name already exists"
	case 1:
		return "account is underaged"
	case 2:
		return "invalid character name"
	default:
		return fmt.Sprintf("character creation refused (%d)", code)
	}
}

func loginBackgroundSets(clientDate int) [][]string {
	tiles2018 := []string{
		"t_\xB9\xE8\xB0\xE61-1.bmp", "t_\xB9\xE8\xB0\xE61-2.bmp", "t_\xB9\xE8\xB0\xE61-3.bmp", "t_\xB9\xE8\xB0\xE61-4.bmp",
		"t_\xB9\xE8\xB0\xE62-1.bmp", "t_\xB9\xE8\xB0\xE62-2.bmp", "t_\xB9\xE8\xB0\xE62-3.bmp", "t_\xB9\xE8\xB0\xE62-4.bmp",
		"t_\xB9\xE8\xB0\xE63-1.bmp", "t_\xB9\xE8\xB0\xE63-2.bmp", "t_\xB9\xE8\xB0\xE63-3.bmp", "t_\xB9\xE8\xB0\xE63-4.bmp",
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
	const ui = "data\\texture\\\xC0\xAF\xC0\xFA\xC0\xCE\xC5\xCD\xC6\xE4\xC0\xCC\xBD\xBA\\"
	candidates := []string{
		ui + name,
		strings.ReplaceAll(ui, "\\", "/") + name,
		"texture\\\xC0\xAF\xC0\xFA\xC0\xCE\xC5\xCD\xC6\xE4\xC0\xCC\xBD\xBA\\" + name,
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
		return
	}

	m.status = ctx.Network.Status()
	log.Printf("connected login server %s:%d", conn.Address, conn.Port)
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
	log.Printf("sent CA_LOGIN user=%s version=%d", m.username, conn.Version)
}

func (m *LoginMode) connectCharServer(ctx client.Context, server network.CharServer) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, server.Address, int(server.Port))
	cancel()
	if err != nil {
		m.status = "char connect failed: " + err.Error()
		return
	}

	err = ctx.Network.SendCharServerEnter(ctx.Session.AccountID, ctx.Session.AuthCode, ctx.Session.UserLevel, ctx.Session.Sex)
	if err != nil {
		m.status = "CA_ENTER failed: " + err.Error()
		return
	}
	m.status = "CA_ENTER sent to char server"
	log.Printf("sent CA_ENTER account_id=%d addr=%s port=%d", ctx.Session.AccountID, server.Address, server.Port)
}

func (m *LoginMode) connectMapServer(ctx client.Context, zone network.ZoneServerNotify) {
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, zone.Address, int(zone.Port))
	cancel()
	if err != nil {
		m.status = "map connect failed: " + err.Error()
		return
	}

	err = ctx.Network.SendMapServerEnter(ctx.Session.AccountID, zone.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex)
	if err != nil {
		m.status = "CZ_ENTER2 failed: " + err.Error()
		return
	}
	m.status = "CZ_ENTER2 sent to map server"
	log.Printf("sent CZ_ENTER2 account_id=%d char_id=%d addr=%s port=%d", ctx.Session.AccountID, zone.CharID, zone.Address, zone.Port)
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
	}
}

func setSelectedCharacter(sessionState *session.Session, character session.Character) {
	sessionState.Selected = character
	sessionState.Vitals = sessionVitalsFromCharacter(character)
	sessionState.Progress = sessionProgressFromCharacter(character)
	sessionState.Inventory.Zeny = character.Money
}
