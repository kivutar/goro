package game

import (
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type WorldMode struct {
	walkCooldownUntil time.Time
	nextHeldWalkAt    time.Time
	tickCooldown      int
	camera            followCamera
	cameraShakeStart  time.Time
	cameraShakeEnd    time.Time
	whitePixel        *render.Image
	tileCursor        *render.Image
	textures          map[string]*render.Image
	textureMiss       map[string]struct{}
	imageCache        map[string]image.Image
	imageMiss         map[string]struct{}
	strEffects        map[string]*res.STR
	strEffectMiss     map[string]struct{}
	playerView        *humanoidSpriteView
	shadowView        *spriteView
	shadowViewMiss    bool
	cartViews         map[int]*spriteView
	cartViewMiss      map[int]struct{}
	cursorView        *spriteView
	cursorViewMiss    bool
	slotMachineView   *spriteView
	slotMachineMiss   bool
	cursorFallback    *render.Image
	cursorAction      int
	cursorStarted     time.Time
	damageNumberView  *spriteView
	damageNumberMiss  bool
	damageNumbers     map[string]*spriteBillboard
	damageMsgView     *spriteView
	damageMsgMiss     bool
	itemMarker        *render.Image
	itemViews         map[itemSpriteKey]*spriteView
	itemViewMiss      map[itemSpriteKey]struct{}
	effectViews       map[string]*spriteView
	effectViewMiss    map[string]struct{}
	actorViews        map[actorSpriteKey]*humanoidSpriteView
	actorViewMiss     map[actorSpriteKey]struct{}
	nonPCViews        map[int]*spriteView
	nonPCViewMiss     map[int]struct{}
	petAccessoryIDs   map[uint32]uint32
	petAccessoryViews map[petAccessorySpriteKey]*spriteView
	petAccessoryMiss  map[petAccessorySpriteKey]struct{}
	rsmMeshCache      map[int][]retainedWorldMesh
	rsmNodeMatrices   map[*res.RSM]map[string]mat4
	rsmAnimNodes      map[animatedRSMNodeKey]map[string]mat4
	rsmBoundsCache    map[rsmBoundsCacheKey]rsmBounds
	rsmFaceMetaCache  map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta
	rsmPlacementGrid  *rsmPlacementGrid
	gndMeshCache      *gndRetainedMeshCache
	pendingWarp       bool
	pendingAttack     attackIntent
	pendingPickup     pickupIntent
	pendingSkill      pendingSkillTarget
	pendingSkillText  pendingSkillTextTarget
	pendingPetCapture petCaptureState
	petProperty       network.PetProperty
	hasPetProperty    bool
	petOldFullness    uint16
	petLastTalk       time.Time
	petInfoRequested  bool
	petID             uint32
	petSlotMachine    petSlotMachineState
	pickupReqItemID   uint32
	lockedAttackID    uint32
	attackFocusID     uint32
	attackFocusStart  time.Time
	lastAttackAt      time.Time
	lastChaseAt       time.Time
	actorAnims        map[uint32]actorAnimation
	damageFloaters    []damageFloater
	worldEffects      []worldEffect
	actorCastBars     map[uint32]actorCastBar
	scheduledSounds   []scheduledSound
	scheduledStops    []scheduledActorStop
	scheduledResumes  []scheduledWalkResume
	mapSoundNext      map[int]time.Time
	mapWeatherSounds  map[int]time.Time
	actorDeaths       map[uint32]time.Time
	actorSoundFrames  map[uint32]actorSoundFrame
	actorLife         map[uint32]actorLife
	actorNameReqAt    map[uint32]time.Time
	guildEmblems      map[uint32]guildEmblem
	speechBubbles     map[uint32]speechBubble
	gndNormalSource   *res.GND
	gndTopNormals     [][4]modelPoint3
	ui                worldUI
	pendingChatRoom   network.ChatRoomCreate
	pendingTradeName  string
	mapFade           mapFadeState
	hoveredWalk       hoveredWalkCellCache
	bot               *luaBot
	companionAI       companionAISystem
}

type worldUI struct {
	minimap           gameui.Minimap
	statusIcons       gameui.StatusIcons
	console           gameui.ChatConsole
	npcDialog         gameui.NPCDialog
	escapeMenu        gameui.EscapeMenu
	teleportModal     gameui.TeleportModal
	deathModal        gameui.DeathModal
	disconnectDialog  gameui.ConfirmModal
	friendRequest     gameui.ConfirmModal
	friendConfirm     gameui.ConfirmModal
	partyRequest      gameui.ConfirmModal
	guildRequest      gameui.ConfirmModal
	tradeRequest      gameui.ConfirmModal
	characterWindow   gameui.CharacterWindow
	basicMenu         gameui.BasicMenu
	inventoryBag      gameui.InventoryBagWindow
	equipmentWindow   gameui.EquipmentWindow
	viewEquipWindow   gameui.ViewEquipmentWindow
	storageWindow     gameui.StorageWindow
	cartWindow        gameui.CartWindow
	changeCartWindow  gameui.ChangeCartWindow
	shopWindow        gameui.ShopWindow
	vendingWindow     gameui.VendingWindow
	itemInfoWindow    gameui.ItemInfoWindow
	identifyWindow    gameui.IdentifyWindow
	cardWindow        gameui.CardCompositionWindow
	makingArrow       gameui.MakingArrowWindow
	petEggWindow      gameui.PetEggWindow
	petInfoWindow     gameui.PetInfoWindow
	petContext        gameui.PetContextMenu
	petConfirm        gameui.ConfirmModal
	homunculusInfo    gameui.HomunculusInfoWindow
	homunculusContext gameui.HomunculusContextMenu
	homunculusConfirm gameui.ConfirmModal
	statsWindow       gameui.StatsWindow
	skillWindow       gameui.SkillWindow
	friendsWindow     gameui.FriendsWindow
	guildWindow       gameui.GuildWindow
	friendSettings    gameui.FriendSettingsWindow
	whisperWindow     gameui.WhisperWindow
	chatRoomCreate    gameui.ChatRoomCreateWindow
	chatRoom          gameui.ChatRoomWindow
	partySettings     gameui.PartySettingsWindow
	partyCreate       gameui.PartyCreateWindow
	partyInvite       gameui.PartyInviteWindow
	partyInfo         gameui.ConfirmModal
	skillTextPrompt   gameui.TextPromptWindow
	playerContext     gameui.PlayerContextMenu
	tradeWindow       gameui.TradeWindow
	settingsWindow    gameui.SettingsWindow
	shortcutBar       gameui.ShortcutBar
}

type actorSpriteKey struct {
	job         int
	head        int
	sex         byte
	bodyPalette int
	headPalette int
	weapon      int
	shield      int
	headTop     int
	headMid     int
	headLow     int
}

type pickupIntent struct {
	itemID  uint32
	expires time.Time
	readyAt time.Time
}

type pendingSkillTarget struct {
	skill    session.Skill
	maxLevel int
	targetID uint32
	expires  time.Time
	readyAt  time.Time
	source   string
	started  time.Time
}

type pendingSkillTextTarget struct {
	skill  session.Skill
	x      int
	y      int
	source string
}

type mapFadePhase int

const (
	mapFadeNone mapFadePhase = iota
	mapFadeOut
	mapFadeHold
	mapFadeIn
)

type mapFadeState struct {
	phase     mapFadePhase
	started   time.Time
	change    network.MapChange
	hasChange bool
}

const (
	mapFadeOutDuration       = 220 * time.Millisecond
	mapFadeInDuration        = 340 * time.Millisecond
	actorNameRequestCooldown = time.Second
	defaultRSMLoadLimit      = 128
)

var (
	quadIndices012023 = []uint16{0, 1, 2, 0, 2, 3}
	quadIndices012213 = []uint16{0, 1, 2, 2, 1, 3}
)

func NewWorldMode() *WorldMode {
	return &WorldMode{}
}

func (m *WorldMode) Name() string {
	return "world"
}

func (m *WorldMode) Enter(ctx client.Context) {
	now := time.Now()
	if m.mapFade.phase == mapFadeNone {
		m.startMapFadeIn(now)
	} else if m.mapFade.started.IsZero() {
		m.mapFade.started = now
	}
	zoom, zoomTarget := m.camera.zoom, m.camera.zoomTarget
	m.camera.Reset()
	m.camera.zoom = zoom
	m.camera.zoomTarget = zoomTarget
	ctx.World.GAT = nil
	ctx.World.GND = nil
	ctx.World.RSW = nil
	ctx.World.RSM = nil
	ctx.World.RSMFail = 0
	m.textures = make(map[string]*render.Image)
	m.textureMiss = make(map[string]struct{})
	m.playerView = nil
	m.shadowView = nil
	m.shadowViewMiss = false
	m.cursorView = nil
	m.cursorViewMiss = false
	m.cursorFallback = nil
	m.cursorAction = cursorActionDefault
	m.cursorStarted = now
	m.damageNumberView = nil
	m.damageNumberMiss = false
	m.damageNumbers = make(map[string]*spriteBillboard)
	m.damageMsgView = nil
	m.damageMsgMiss = false
	m.itemMarker = nil
	m.itemViews = make(map[itemSpriteKey]*spriteView)
	m.itemViewMiss = make(map[itemSpriteKey]struct{})
	m.effectViews = make(map[string]*spriteView)
	m.effectViewMiss = make(map[string]struct{})
	m.actorViews = make(map[actorSpriteKey]*humanoidSpriteView)
	m.actorViewMiss = make(map[actorSpriteKey]struct{})
	m.nonPCViews = make(map[int]*spriteView)
	m.nonPCViewMiss = make(map[int]struct{})
	m.petAccessoryIDs = make(map[uint32]uint32)
	m.petAccessoryViews = make(map[petAccessorySpriteKey]*spriteView)
	m.petAccessoryMiss = make(map[petAccessorySpriteKey]struct{})
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	m.rsmNodeMatrices = make(map[*res.RSM]map[string]mat4)
	m.rsmAnimNodes = make(map[animatedRSMNodeKey]map[string]mat4)
	m.rsmBoundsCache = make(map[rsmBoundsCacheKey]rsmBounds)
	m.rsmFaceMetaCache = make(map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta)
	m.rsmPlacementGrid = nil
	m.gndMeshCache = nil
	m.pendingWarp = false
	m.pendingAttack = attackIntent{}
	m.pendingPickup = pickupIntent{}
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	m.pickupReqItemID = 0
	m.lockedAttackID = 0
	m.clearAttackFocus()
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
	m.actorAnims = make(map[uint32]actorAnimation)
	m.damageFloaters = nil
	m.scheduledSounds = nil
	m.scheduledStops = nil
	m.mapSoundNext = make(map[int]time.Time)
	m.mapWeatherSounds = make(map[int]time.Time)
	m.actorDeaths = make(map[uint32]time.Time)
	m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	m.actorLife = make(map[uint32]actorLife)
	m.speechBubbles = make(map[uint32]speechBubble)
	m.syncCurrentActorEffectStateEffects(ctx)
	m.ui.npcDialog.ResetPublished(ctx)
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	playerStatus := ""
	character := ctx.Session.SelectedCharacter()
	if view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex); view != nil {
		m.playerView = view
		playerStatus = status
	} else {
		playerStatus = status
	}
	if view, status := loadActorShadowSpriteView(ctx.Resources); view != nil {
		m.shadowView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.shadowViewMiss = true
		glog.Warnf("actor shadow resources unavailable: %s", status)
	}
	m.cartViews = make(map[int]*spriteView)
	m.cartViewMiss = make(map[int]struct{})
	if view, status := loadCursorSpriteView(ctx.Resources); view != nil {
		m.cursorView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.cursorViewMiss = true
		glog.Warnf("cursor resources unavailable: %s", status)
	}
	render.SetCursorMode(render.CursorModeHidden)
	glog.Debugf("player sprite resources char_id=%d name=%s job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, character.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
	m.rebindPersistentUI(ctx)
	if ctx.World.MapName == "" {
		return
	}

	gat, _, err := loadGAT(ctx.Resources, ctx.World.MapName)
	if err != nil {
		return
	}
	ctx.World.GAT = gat
	if gnd, _, err := loadGND(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.GND = gnd
	} else {
		ctx.World.GND = nil
	}
	if rsw, rswSource, err := loadRSW(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.RSW = rsw
		ctx.World.RSM, ctx.World.RSMFail = loadRSMModels(ctx.Resources, rsw, defaultRSMLoadLimit)
		m.playMapBGM(ctx, rswSource)
	} else {
		ctx.World.RSW = nil
		ctx.World.RSM = nil
		ctx.World.RSMFail = 0
		m.playMapBGM(ctx, ctx.World.MapName)
	}
	if err := ctx.Network.SendLoadEndAck(); err == nil {
		m.tickCooldown = 1
	}
}

func (m *WorldMode) rebindPersistentUI(ctx client.Context) {
	m.ui.console.OnGuildWindow = func() { m.toggleGuildWindow(ctx) }
	m.ui.guildWindow.EmblemImage = func(ctx client.Context) image.Image {
		if ctx.Session == nil || m.guildEmblems == nil {
			return nil
		}
		guildID := ctx.Session.GuildID
		version := ctx.Session.EmblemVersion
		if guildID == 0 {
			guildID = ctx.Session.Guild.ID
		}
		if version == 0 {
			version = ctx.Session.Guild.EmblemVersion
		}
		if guildID == 0 || version == 0 {
			return nil
		}
		emblem := m.guildEmblems[guildID]
		if emblem.image == nil || emblem.version < version {
			return nil
		}
		return emblem.image.RGBA()
	}
	m.setGuildEmblemOptions(ctx)
	m.ui.basicMenu.Rebind(ctx, m.basicMenuCallbacks(ctx))
	m.ui.inventoryBag.Rebind(ctx, &m.ui.itemInfoWindow, &m.ui.cartWindow)
	m.ui.equipmentWindow.Rebind(ctx, &m.ui.itemInfoWindow, &m.ui.cartWindow, m)
	m.ui.cartWindow.Rebind(ctx, &m.ui.itemInfoWindow)
	m.ui.itemInfoWindow.Rebind(ctx, m)
	m.ui.statsWindow.Rebind(ctx)
	m.ui.skillWindow.Rebind(ctx, m)
	m.ui.friendsWindow.Rebind(ctx)
	m.ui.guildWindow.Rebind(ctx)
	m.ui.friendSettings.Rebind(ctx)
	m.ui.partySettings.Rebind(ctx)
	m.ui.partyCreate.Rebind(ctx)
	m.ui.partyInvite.Rebind(ctx)
	m.ui.chatRoomCreate.Rebind(ctx)
	m.ui.chatRoom.Rebind(ctx)
	m.ui.skillTextPrompt.Rebind(ctx)
	m.ui.settingsWindow.Rebind(ctx)
	m.ui.homunculusInfo.Rebind(ctx)
	m.ui.whisperWindow.Rebind(ctx)
	m.ui.shortcutBar.ResetOverlay(ctx)
}

func (m *WorldMode) playMapBGM(ctx client.Context, rswName string) {
	if ctx.Audio == nil {
		return
	}
	_, err := ctx.Audio.PlayMap(rswName)
	if err != nil {
		glog.Warnf("bgm failed map=%s: %v", rswName, err)
		return
	}
}

func (m *WorldMode) Update(ctx client.Context) (Mode, error) {
	now := time.Now()
	if m.mapFade.phase == mapFadeOut {
		if !m.mapFadeElapsed(now) {
			return nil, nil
		}
		if m.mapFade.hasChange {
			change := m.mapFade.change
			m.mapFade = mapFadeState{phase: mapFadeHold, started: now}
			next := m.handleMapChange(ctx, change)
			if next != nil {
				return next, nil
			}
			if !m.pendingWarp {
				m.startMapFadeIn(now)
			}
			return nil, nil
		}
		m.startMapFadeIn(now)
	}
	if m.mapFade.phase == mapFadeIn && m.mapFadeElapsed(now) {
		m.mapFade = mapFadeState{}
	}

	if m.updateDisconnectDialog(ctx) {
		return nil, nil
	}

	for _, pkt := range ctx.Network.DrainPackets() {
		if handleDisconnectPacket(ctx, &m.ui.disconnectDialog, pkt) {
			continue
		}
		if hotkeys, ok, err := network.ParseHotkeyList(pkt); err != nil {
			glog.Errorf("parse hotkey list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyHotkeyList(ctx, hotkeys)
			m.ui.shortcutBar.SyncFromSession(ctx)
			continue
		}
		if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
			glog.Errorf("parse chat message 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatMessage(ctx, chat, now)
			continue
		}
		if whisper, ok, err := network.ParseWhisperMessage(pkt); err != nil {
			glog.Errorf("parse whisper message 0x%04X: %v", pkt.ID, err)
		} else if ok {
			addWhisperMessage(&m.ui.console, whisper)
			m.addWhisperWindowIncoming(ctx, whisper)
			continue
		}
		if ack, ok, err := network.ParseWhisperAck(pkt); err != nil {
			glog.Errorf("parse whisper ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			addWhisperAck(&m.ui.console, ctx.Resources, ack)
			m.addWhisperWindowAck(ctx, ack)
			continue
		}
		if ack, ok, err := network.ParseWhisperIgnoreAck(pkt); err != nil {
			glog.Errorf("parse whisper ignore ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			addWhisperIgnoreAck(&m.ui.console, ack)
			continue
		}
		if ack, ok, err := network.ParseChatRoomCreateAck(pkt); err != nil {
			glog.Errorf("parse chat room create ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomCreateAck(ctx, ack)
			continue
		}
		if board, ok, err := network.ParseChatRoomBoard(pkt); err != nil {
			glog.Errorf("parse chat room board 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyChatRoomBoard(ctx, board)
			continue
		}
		if destroy, ok, err := network.ParseChatRoomDestroy(pkt); err != nil {
			glog.Errorf("parse chat room destroy 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyChatRoomDestroy(ctx, destroy)
			continue
		}
		if enter, ok, err := network.ParseChatRoomEnter(pkt); err != nil {
			glog.Errorf("parse chat room enter 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomEnter(ctx, enter)
			continue
		}
		if joined, ok, err := network.ParseChatRoomMemberJoin(pkt); err != nil {
			glog.Errorf("parse chat room member join 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomMemberJoin(ctx, joined)
			continue
		}
		if left, ok, err := network.ParseChatRoomMemberLeave(pkt); err != nil {
			glog.Errorf("parse chat room member leave 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomMemberLeave(ctx, left)
			continue
		}
		if change, ok, err := network.ParseChatRoomChange(pkt); err != nil {
			glog.Errorf("parse chat room change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomChange(ctx, change)
			continue
		}
		if role, ok, err := network.ParseChatRoomRoleChange(pkt); err != nil {
			glog.Errorf("parse chat room role change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleChatRoomRoleChange(ctx, role)
			continue
		}
		if emotion, ok, err := network.ParseEmotionNotify(pkt); err != nil {
			glog.Errorf("parse emotion 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyEmotionNotify(ctx, emotion)
			continue
		}
		if change, ok, err := network.ParseMapChange(pkt); err != nil {
			glog.Errorf("parse map change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.startMapFadeOut(change, time.Now())
			return nil, nil
		}
		if enter, err := network.ParseMapAcceptEnter(pkt); err == nil {
			applyMapAcceptEnter(ctx, enter)
			sendLessEffectPreference(ctx)
			if m.pendingWarp {
				m.pendingWarp = false
				return m.nextWorldMode(), nil
			}
			continue
		}
		if ack, ok, err := network.ParseActorNameAck(pkt); err != nil {
			glog.Errorf("parse actor name ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyActorNameAck(ctx, ack)
			continue
		}
		if ack, ok, err := network.ParseRestartAck(pkt); err != nil {
			glog.Errorf("parse restart ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if m.ui.deathModal.ApplyRestartAck(ack) {
				return m.nextCharacterSelectMode(ctx), nil
			}
			if m.ui.escapeMenu.ApplyRestartAck(ack) {
				return m.nextCharacterSelectMode(ctx), nil
			}
			continue
		}
		if ack, ok, err := network.ParseQuitGameAck(pkt); err != nil {
			glog.Errorf("parse quit game ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyQuitGameAck(ctx, ack)
			continue
		}
		if dialog, ok, err := network.ParseNPCDialog(pkt); err != nil {
			glog.Errorf("parse npc dialog 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.npcDialog.Apply(dialog)
			continue
		}
		if ack, ok, err := network.ParseSelfMoveAck(pkt); err != nil {
			glog.Errorf("parse self move ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySelfMoveAck(ctx, ack)
			m.clearLocalActorAction(ctx)
			glog.Debugf("walk ack from=%d,%d to=%d,%d tick=%d", ack.FromX, ack.FromY, ack.ToX, ack.ToY, ack.ServerTick)
			m.continuePendingAttack(ctx, "walk ack")
			m.continuePendingPickup(ctx, "walk ack")
			m.skills().ContinuePendingTarget(ctx, "walk ack")
			continue
		}
		if position, ok, err := network.ParseActorSetPosition(pkt); err != nil {
			glog.Errorf("parse actor set position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if isLocalActor(ctx, position.ID) {
				glog.Debugf("local position fix id=%d x=%d y=%d", position.ID, position.X, position.Y)
			}
			applyActorSetPosition(ctx, position)
			if isLocalActor(ctx, position.ID) {
				m.continuePendingAttack(ctx, "position fix")
				m.continuePendingPickup(ctx, "position fix")
				m.skills().ContinuePendingTarget(ctx, "position fix")
			}
			continue
		}
		if position, ok, err := network.ParseActorJumpPosition(pkt); err != nil {
			glog.Errorf("parse actor jump position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if isLocalActor(ctx, position.ID) {
				glog.Debugf("local jump position id=%d x=%d y=%d", position.ID, position.X, position.Y)
			}
			applyActorJumpPosition(ctx, position)
			continue
		}
		if item, ok, err := network.ParseFloorItemEntry(pkt); err != nil {
			glog.Errorf("parse floor item entry 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyFloorItemEntry(ctx, item)
			continue
		}
		if disappear, ok, err := network.ParseFloorItemDisappear(pkt); err != nil {
			glog.Errorf("parse floor item disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyFloorItemDisappear(ctx, disappear)
			continue
		}
		if pickup, ok, err := network.ParseItemPickupAck(pkt); err != nil {
			glog.Errorf("parse item pickup ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyItemPickupAck(ctx, pickup)
			if pickup.Result == 0 {
				message := formatPickupConsoleMessage(ctx.Resources, pickup)
				glog.Debugf("console pickup message item_id=%d amount=%d text=%q", pickup.ItemID, pickup.Amount, message)
				m.ui.console.AddBlueMessage("%s", message)
			} else {
				m.ui.console.AddErrorMessage("Pickup failed item %d result=%d", pickup.ItemID, pickup.Result)
			}
			continue
		}
		if useAck, ok, err := network.ParseUseItemAck(pkt); err != nil {
			glog.Errorf("parse use item ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("use item ack index=%d item=%d aid=%d amount=%d result=%d", useAck.Index, useAck.ItemID, useAck.AID, useAck.Amount, useAck.Result)
			m.addItemUseEffect(ctx, useAck)
			applyUseItemAck(ctx, useAck)
			if useAck.Result != 0 && useAck.Amount == 0 && m.ui.shortcutBar.ClearDepletedItem(ctx, useAck.Index, useAck.ItemID) {
				glog.Debugf("shortcut item depleted index=%d item=%d", useAck.Index, useAck.ItemID)
			}
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if _, ok, err := network.ParsePetCaptureStart(pkt); err != nil {
			glog.Errorf("parse pet capture start 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.startPetCapture(ctx)
			continue
		}
		if petCapture, ok, err := network.ParsePetCaptureResult(pkt); err != nil {
			glog.Errorf("parse pet capture result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetCaptureResult(ctx, petCapture)
			continue
		}
		if petProperty, ok, err := network.ParsePetProperty(pkt); err != nil {
			glog.Errorf("parse pet property 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetProperty(ctx, petProperty)
			continue
		}
		if petFeed, ok, err := network.ParsePetFeedResult(pkt); err != nil {
			glog.Errorf("parse pet feed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetFeedResult(ctx, petFeed)
			continue
		}
		if petState, ok, err := network.ParsePetStateChange(pkt); err != nil {
			glog.Errorf("parse pet state 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetStateChange(ctx, petState)
			continue
		}
		if petEggs, ok, err := network.ParsePetEggList(pkt); err != nil {
			glog.Errorf("parse pet egg list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetEggList(ctx, petEggs)
			continue
		}
		if petAction, ok, err := network.ParsePetAction(pkt); err != nil {
			glog.Errorf("parse pet action 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyPetAction(ctx, petAction)
			continue
		}
		if identifyList, ok, err := network.ParseItemIdentifyList(pkt); err != nil {
			glog.Errorf("parse item identify list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("item identify list indexes=%v", identifyList.Indexes)
			m.ui.identifyWindow.OpenList(ctx, identifyList)
			continue
		}
		if identifyAck, ok, err := network.ParseItemIdentifyAck(pkt); err != nil {
			glog.Errorf("parse item identify ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("item identify ack index=%d success=%v", identifyAck.Index, identifyAck.Success)
			applyItemIdentifyAck(ctx, identifyAck)
			m.ui.identifyWindow.ApplyAck(ctx, identifyAck)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if arrowList, ok, err := network.ParseMakingArrowList(pkt); err != nil {
			glog.Errorf("parse making arrow list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("making arrow list items=%v", arrowList.ItemIDs)
			m.ui.makingArrow.OpenList(ctx, arrowList)
			continue
		}
		if compositionList, ok, err := network.ParseItemCompositionList(pkt); err != nil {
			glog.Errorf("parse item composition list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("item composition list indexes=%v", compositionList.Indexes)
			m.ui.cardWindow.OpenList(ctx, m.ui.inventoryBag.PendingCardIndex(), compositionList)
			continue
		}
		if compositionAck, ok, err := network.ParseItemCompositionAck(pkt); err != nil {
			glog.Errorf("parse item composition ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("item composition ack equip_index=%d card_index=%d success=%v", compositionAck.EquipIndex, compositionAck.CardIndex, compositionAck.Success)
			applyItemCompositionAck(ctx, compositionAck)
			m.ui.cardWindow.ApplyAck(ctx, compositionAck)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if items, ok, err := network.ParseInventoryItemList(pkt); err != nil {
			glog.Errorf("parse inventory item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryItemList(ctx, items)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if itemDelete, ok, err := network.ParseInventoryItemDelete(pkt); err != nil {
			glog.Errorf("parse inventory item delete 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryItemDelete(ctx, itemDelete)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if equipAck, ok, err := network.ParseInventoryEquipAck(pkt); err != nil {
			glog.Errorf("parse inventory equip ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryEquipAck(ctx, equipAck)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if arrow, ok, err := network.ParseEquippedArrow(pkt); err != nil {
			glog.Errorf("parse equipped arrow 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("equipped arrow index=%d", arrow.Index)
			applyEquippedArrow(ctx, arrow)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if storageItems, ok, err := network.ParseStorageItemList(pkt); err != nil {
			glog.Errorf("parse storage item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemList(ctx, storageItems)
			m.ui.storageWindow.OpenWindow(ctx)
			continue
		}
		if cartItems, ok, err := network.ParseCartItemList(pkt); err != nil {
			glog.Errorf("parse cart item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartItemList(ctx, cartItems)
			m.ui.cartWindow.ClampScroll(ctx.Session)
			m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
			continue
		}
		if storageAmount, ok, err := network.ParseStorageAmount(pkt); err != nil {
			glog.Errorf("parse storage amount 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageAmount(ctx, storageAmount)
			m.ui.storageWindow.OpenWindow(ctx)
			continue
		}
		if cartAmount, ok, err := network.ParseCartAmount(pkt); err != nil {
			glog.Errorf("parse cart amount 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartAmount(ctx, cartAmount)
			m.ui.cartWindow.ClampScroll(ctx.Session)
			m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
			continue
		}
		if friends, ok, err := network.ParseFriendsList(pkt); err != nil {
			glog.Errorf("parse friends list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendsList(ctx, friends)
			continue
		}
		if friendState, ok, err := network.ParseFriendState(pkt); err != nil {
			glog.Errorf("parse friend state 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendState(ctx, friendState)
			continue
		}
		if friendRequest, ok, err := network.ParseFriendRequest(pkt); err != nil {
			glog.Errorf("parse friend request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openFriendRequest(ctx, friendRequest)
			continue
		}
		if friendAdded, ok, err := network.ParseFriendAddResult(pkt); err != nil {
			glog.Errorf("parse friend add result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendAddResult(ctx, friendAdded)
			m.addFriendResultMessage(friendAdded)
			continue
		}
		if friendDeleted, ok, err := network.ParseFriendDelete(pkt); err != nil {
			glog.Errorf("parse friend delete 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if friend, removed := applyFriendDelete(ctx, friendDeleted); removed {
				name := friend.Name
				if strings.TrimSpace(name) == "" {
					name = "Friend"
				}
				m.ui.console.AddSystemMessage("%s removed from your friend list.", name)
			}
			continue
		}
		if partyCreate, ok, err := network.ParsePartyCreateResult(pkt); err != nil {
			glog.Errorf("parse party create result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handlePartyCreateResult(ctx, partyCreate)
			continue
		}
		if partyList, ok, err := network.ParsePartyList(pkt); err != nil {
			glog.Errorf("parse party list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyList(ctx, partyList)
			continue
		}
		if partyInvite, ok, err := network.ParsePartyInviteRequest(pkt); err != nil {
			glog.Errorf("parse party invite request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openPartyInviteRequest(ctx, partyInvite)
			continue
		}
		if partyInviteAnswer, ok, err := network.ParsePartyInviteAnswer(pkt); err != nil {
			glog.Errorf("parse party invite answer 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handlePartyInviteAnswer(partyInviteAnswer)
			continue
		}
		if partyOption, ok, err := network.ParsePartyOption(pkt); err != nil {
			glog.Errorf("parse party option 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyOption(ctx, partyOption)
			continue
		}
		if partyConfig, ok, err := network.ParsePartyInviteConfig(pkt); err != nil {
			glog.Errorf("parse party invite config 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyInviteConfig(ctx, partyConfig)
			m.ui.partySettings.Rebind(ctx)
			continue
		}
		if guildBelonging, ok, err := network.ParseGuildBelonging(pkt); err != nil {
			glog.Errorf("parse guild belonging 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildBelonging(ctx, guildBelonging)
			m.requestActorGuildEmblem(ctx, guildBelonging.GuildID, guildBelonging.EmblemVersion)
			continue
		}
		if guildInfo, ok, err := network.ParseGuildInfo(pkt); err != nil {
			glog.Errorf("parse guild info 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildDetails(ctx, guildInfo)
			m.requestActorGuildEmblem(ctx, guildInfo.GuildID, guildInfo.EmblemVersion)
			continue
		}
		if guildMembers, ok, err := network.ParseGuildMembers(pkt); err != nil {
			glog.Errorf("parse guild members 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildMembers(ctx, guildMembers)
			continue
		}
		if guildMember, ok, err := network.ParseGuildMemberInfo(pkt); err != nil {
			glog.Errorf("parse guild member info 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildMember(ctx, guildMember)
			continue
		}
		if memberPositions, ok, err := network.ParseGuildMemberPositions(pkt); err != nil {
			glog.Errorf("parse guild member positions 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildMemberPositions(ctx, memberPositions)
			m.ui.guildWindow.Refresh(ctx)
			continue
		}
		if guildPositions, ok, err := network.ParseGuildPositions(pkt); err != nil {
			glog.Errorf("parse guild positions 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildPositions(ctx, guildPositions)
			continue
		}
		if guildPositionNames, ok, err := network.ParseGuildPositionNames(pkt); err != nil {
			glog.Errorf("parse guild position names 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildPositionNames(ctx, guildPositionNames)
			continue
		}
		if guildSkills, ok, err := network.ParseGuildSkillInfo(pkt); err != nil {
			glog.Errorf("parse guild skills 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildSkills(ctx, guildSkills)
			continue
		}
		if guildExpelHistory, ok, err := network.ParseGuildExpelHistory(pkt); err != nil {
			glog.Errorf("parse guild expel history 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyLocalGuildExpelHistory(ctx, guildExpelHistory)
			continue
		}
		if guildNotice, ok, err := network.ParseGuildNotice(pkt); err != nil {
			glog.Errorf("parse guild notice 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleGuildNotice(ctx, guildNotice)
			continue
		}
		if guildEmblem, ok, err := network.ParseGuildEmblemImage(pkt); err != nil {
			glog.Errorf("parse guild emblem 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyGuildEmblemImage(ctx, guildEmblem)
			continue
		}
		if guildEmblemChange, ok, err := network.ParseGuildEmblemChange(pkt); err != nil {
			glog.Errorf("parse guild emblem change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyGuildEmblemChange(ctx, guildEmblemChange)
			continue
		}
		if guildCreate, ok, err := network.ParseGuildCreationResult(pkt); err != nil {
			glog.Errorf("parse guild create result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleGuildCreationResult(ctx, guildCreate)
			continue
		}
		if guildInvite, ok, err := network.ParseGuildInviteRequest(pkt); err != nil {
			glog.Errorf("parse guild invite request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openGuildInviteRequest(ctx, guildInvite)
			continue
		}
		if guildInviteAck, ok, err := network.ParseGuildInviteAck(pkt); err != nil {
			glog.Errorf("parse guild invite ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleGuildInviteAck(guildInviteAck)
			continue
		}
		if partyMember, ok, err := network.ParsePartyMemberJoin(pkt); err != nil {
			glog.Errorf("parse party member join 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberJoin(ctx, partyMember)
			continue
		}
		if partyLeave, ok, err := network.ParsePartyMemberLeave(pkt); err != nil {
			glog.Errorf("parse party member leave 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if !applyPartyMemberLeave(ctx, partyLeave) {
				m.ui.console.AddErrorMessage("Cannot leave party on this map.")
			}
			continue
		}
		if partyHP, ok, err := network.ParsePartyMemberHP(pkt); err != nil {
			glog.Errorf("parse party member hp 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberHP(ctx, partyHP)
			continue
		}
		if partyPosition, ok, err := network.ParsePartyMemberPosition(pkt); err != nil {
			glog.Errorf("parse party member position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberPosition(ctx, partyPosition)
			continue
		}
		if partyChat, ok, err := network.ParsePartyChat(pkt); err != nil {
			glog.Errorf("parse party chat 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyChat(ctx, partyChat, &m.ui.console)
			continue
		}
		if tradeRequest, ok, err := network.ParseTradeRequest(pkt); err != nil {
			glog.Errorf("parse trade request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openTradeRequest(ctx, tradeRequest)
			continue
		}
		if showEquip, ok, err := network.ParseShowEquipConfig(pkt); err != nil {
			glog.Errorf("parse show equip config 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if ctx.Session != nil {
				ctx.Session.ShowEquip = showEquip
			}
			continue
		}
		if lessEffects, ok, err := network.ParseLessEffect(pkt); err != nil {
			glog.Errorf("parse less effect 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if ctx.Session != nil {
				ctx.Session.LessEffects = lessEffects
			}
			continue
		}
		if viewedEquip, ok, err := network.ParseViewedEquipment(pkt); err != nil {
			glog.Errorf("parse viewed equipment 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.viewEquipWindow.Open(ctx, viewedEquip, m)
			continue
		}
		if tradeResponse, ok, err := network.ParseTradeResponse(pkt); err != nil {
			glog.Errorf("parse trade response 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleTradeResponse(ctx, tradeResponse)
			continue
		}
		if tradeItem, ok, err := network.ParseTradeItem(pkt); err != nil {
			glog.Errorf("parse trade item 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.tradeWindow.AddReceivedItem(ctx, tradeItem)
			continue
		}
		if tradeAck, ok, err := network.ParseTradeAddItemAck(pkt); err != nil {
			glog.Errorf("parse trade add item ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.tradeWindow.AddOwnItemAck(ctx, tradeAck)
			continue
		}
		if tradeConclude, ok, err := network.ParseTradeConclude(pkt); err != nil {
			glog.Errorf("parse trade conclude 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.tradeWindow.SetConcluded(ctx, tradeConclude.Other)
			continue
		}
		if network.ParseTradeCanceled(pkt) {
			m.ui.tradeWindow.Close(ctx)
			m.ui.console.AddErrorMessage("Trade canceled.")
			continue
		}
		if tradeExec, ok, err := network.ParseTradeExec(pkt); err != nil {
			glog.Errorf("parse trade exec 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleTradeExec(ctx, tradeExec)
			continue
		}
		if network.ParseTradeUndo(pkt) {
			m.ui.tradeWindow.Undo(ctx)
			continue
		}
		if storageItem, ok, err := network.ParseStorageItemAdded(pkt); err != nil {
			glog.Errorf("parse storage item added 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemAdded(ctx, storageItem)
			m.ui.storageWindow.OpenWindow(ctx)
			m.ui.storageWindow.ClampScroll(ctx.Session)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemAdded(pkt); err != nil {
			glog.Errorf("parse cart item added 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("cart item added index=%d item=%d amount=%d", cartItem.Index, cartItem.ItemID, cartItem.Amount)
			applyCartItemAdded(ctx, cartItem)
			m.ui.cartWindow.ClampScroll(ctx.Session)
			m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
			m.ui.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if ack, ok, err := network.ParseCartAddAck(pkt); err != nil {
			glog.Errorf("parse cart add ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			switch ack.Result {
			case 0:
				m.ui.console.AddErrorMessage("Cart is overweight.")
				glog.Warnf("cart add rejected result=%d reason=weight", ack.Result)
			case 1:
				m.ui.console.AddErrorMessage("Cart has too many items.")
				glog.Warnf("cart add rejected result=%d reason=count", ack.Result)
			default:
				m.ui.console.AddErrorMessage("Cart add failed.")
				glog.Warnf("cart add rejected result=%d", ack.Result)
			}
			continue
		}
		if storageItem, ok, err := network.ParseStorageItemRemoved(pkt); err != nil {
			glog.Errorf("parse storage item removed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemRemoved(ctx, storageItem)
			m.ui.storageWindow.ClampScroll(ctx.Session)
			m.ui.storageWindow.Refresh(ctx, &m.ui.itemInfoWindow)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemRemoved(pkt); err != nil {
			glog.Errorf("parse cart item removed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartItemRemoved(ctx, cartItem)
			m.ui.cartWindow.ClampScroll(ctx.Session)
			m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
			continue
		}
		if vendOpen, ok, err := network.ParseVendingOpenRequest(pkt); err != nil {
			glog.Errorf("parse vending open request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("vending open request max_items=%d", vendOpen.MaxItems)
			m.ui.vendingWindow.OpenSetup(ctx, vendOpen)
			continue
		}
		if board, ok, err := network.ParseVendingBoard(pkt); err != nil {
			glog.Errorf("parse vending board 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyVendingBoard(ctx, board)
			continue
		}
		if board, ok, err := network.ParseVendingBoardDisappear(pkt); err != nil {
			glog.Errorf("parse vending board disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyVendingBoardDisappear(ctx, board)
			continue
		}
		if vendList, ok, err := network.ParseVendingItemList(pkt); err != nil {
			glog.Errorf("parse vending item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if vendList.Own {
				m.ui.vendingWindow.ApplyOwnList(ctx, vendList)
			} else {
				m.ui.vendingWindow.OpenBuy(ctx, vendList)
			}
			continue
		}
		if vendResult, ok, err := network.ParseVendingPurchaseResult(pkt); err != nil {
			glog.Errorf("parse vending purchase result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.vendingWindow.ApplyPurchaseResult(ctx, vendResult)
			continue
		}
		if sold, ok, err := network.ParseVendingSoldItem(pkt); err != nil {
			glog.Errorf("parse vending sold item 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.vendingWindow.ApplySoldItem(ctx, sold)
			continue
		}
		if network.ParseStorageClosed(pkt) {
			applyStorageClosed(ctx)
			m.ui.storageWindow.SetOpen(false)
			continue
		}
		if network.ParseCartClosed(pkt) {
			applyCartClosed(ctx)
			m.ui.cartWindow.SetOpen(false)
			continue
		}
		if deal, ok, err := network.ParseShopDealSelection(pkt); err != nil {
			glog.Errorf("parse shop deal selection 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.shopWindow.OpenDeal(deal, ctx)
			continue
		}
		if sellList, ok, err := network.ParseShopSellList(pkt); err != nil {
			glog.Errorf("parse shop sell list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.shopWindow.OpenSell(sellList, ctx)
			continue
		}
		if buyList, ok, err := network.ParseShopBuyList(pkt); err != nil {
			glog.Errorf("parse shop buy list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.shopWindow.OpenBuy(buyList, ctx)
			continue
		}
		if result, ok, err := network.ParseShopResult(pkt); err != nil {
			glog.Errorf("parse shop result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.shopWindow.ApplyResult(ctx, result)
			if result.Sell && result.Result == 0 {
				m.ui.console.AddBlueMessage("The deal has successfully completed.")
			}
			continue
		}
		if vanish, ok, err := network.ParseActorVanish(pkt); err != nil {
			glog.Errorf("parse actor vanish 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorVanish(ctx, vanish)
			if m.pendingAttack.targetID == vanish.ID {
				m.pendingAttack = attackIntent{}
			}
			if m.lockedAttackID == vanish.ID {
				m.clearLockedAttack()
			}
			if m.attackFocusID == vanish.ID {
				m.clearAttackFocus()
			}
			continue
		}
		if look, ok, err := network.ParseActorLookChange(pkt); err != nil {
			glog.Errorf("parse actor look change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if m.applySkillUnitLookChange(ctx, look) {
				continue
			}
			if applyActorLookChange(ctx, look) {
				if view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.SelectedCharacter(), ctx.Session.Sex); view != nil {
					m.playerView = view
					glog.Debugf("player sprite changed type=%d value=%d %s", look.Type, look.Value, status)
				} else {
					m.playerView = nil
					glog.Warnf("player sprite reload failed after look change type=%d value=%d: %s", look.Type, look.Value, status)
				}
			}
			continue
		}
		if direction, ok, err := network.ParseActorDirectionChange(pkt); err != nil {
			glog.Errorf("parse actor direction change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyActorDirectionChange(ctx, direction)
			continue
		}
		if state, ok, err := network.ParseActorStateChange(pkt); err != nil {
			glog.Errorf("parse actor state change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorStateChange(ctx, state)
			continue
		}
		if bladeStop, ok, err := network.ParseActorBladeStop(pkt); err != nil {
			glog.Errorf("parse actor blade stop 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorBladeStop(ctx, bladeStop)
			continue
		}
		if action, ok, err := network.ParseActorActionNotify(pkt); err != nil {
			glog.Errorf("parse actor action 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorActionNotify(ctx, action)
			continue
		}
		if life, ok, err := network.ParseActorHPUpdate(pkt); err != nil {
			glog.Errorf("parse actor hp 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorHPUpdate(life)
			continue
		}
		if snapshot, ok, err := network.ParseStatusSnapshot(pkt); err != nil {
			glog.Errorf("parse status snapshot 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStatusSnapshot(ctx, snapshot)
			continue
		}
		if ack, ok, err := network.ParseStatusChangeAck(pkt); err != nil {
			glog.Errorf("parse status change ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.ui.statsWindow.ApplyStatusChangeAck(ctx, ack)
			continue
		}
		if statusEffect, ok, err := network.ParseStatusEffectChange(pkt); err != nil {
			glog.Errorf("parse status effect change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyStatusEffectChange(ctx, statusEffect)
			continue
		}
		if hom, ok, err := network.ParseHomunculusProperty(pkt); err != nil {
			glog.Errorf("parse homunculus property 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyHomunculusProperty(ctx, hom)
			continue
		}
		if feed, ok, err := network.ParseHomunculusFeedResult(pkt); err != nil {
			glog.Errorf("parse homunculus feed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			glog.Debugf("homunculus feed result success=%t item=%d", feed.Result, feed.ItemID)
			m.applyHomunculusFeedResultMessage(ctx, feed)
			continue
		}
		if homState, ok, err := network.ParseHomunculusStateChange(pkt); err != nil {
			glog.Errorf("parse homunculus state 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyHomunculusStateChange(ctx, homState)
			continue
		}
		if homParam, ok, err := network.ParseHomunculusParamChange(pkt); err != nil {
			glog.Errorf("parse homunculus param 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyHomunculusParamChange(ctx, homParam)
			continue
		}
		if homSkills, ok, err := network.ParseHomunculusSkillInfoList(pkt); err != nil {
			glog.Errorf("parse homunculus skill list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyHomunculusSkillInfoList(ctx, homSkills)
			continue
		}
		if homSkill, ok, err := network.ParseHomunculusSkillInfoUpdate(pkt); err != nil {
			glog.Errorf("parse homunculus skill update 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyHomunculusSkillInfoUpdate(ctx, homSkill)
			continue
		}
		if merc, ok, err := network.ParseMercenaryProperty(pkt); err != nil {
			glog.Errorf("parse mercenary property 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyMercenaryProperty(ctx, merc)
			continue
		}
		if mercParam, ok, err := network.ParseMercenaryParamChange(pkt); err != nil {
			glog.Errorf("parse mercenary param 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyMercenaryParamChange(ctx, mercParam)
			continue
		}
		if mercSkills, ok, err := network.ParseMercenarySkillInfoList(pkt); err != nil {
			glog.Errorf("parse mercenary skill list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyMercenarySkillInfoList(ctx, mercSkills)
			continue
		}
		if mercSkill, ok, err := network.ParseMercenarySkillInfoUpdate(pkt); err != nil {
			glog.Errorf("parse mercenary skill update 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyMercenarySkillInfoUpdate(ctx, mercSkill)
			continue
		}
		if list, ok, err := network.ParseSkillInfoList(pkt); err != nil {
			glog.Errorf("parse skill list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySkillInfoList(ctx, list)
			m.ui.skillWindow.ClampScroll(ctx.Session)
			continue
		}
		if update, ok, err := network.ParseSkillInfoUpdate(pkt); err != nil {
			glog.Errorf("parse skill update 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySkillInfoUpdate(ctx, update)
			m.ui.skillWindow.ClampScroll(ctx.Session)
			continue
		}
		if auto, ok, err := network.ParseAutoRunSkill(pkt); err != nil {
			glog.Errorf("parse auto-run skill 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.skills().ApplyAutoRun(ctx, auto)
			continue
		}
		if warpList, ok, err := network.ParseWarpPointList(pkt); err != nil {
			glog.Errorf("parse warp point list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyWarpPointList(ctx, warpList)
			continue
		}
		if memo, ok, err := network.ParseRememberWarpPointAck(pkt); err != nil {
			glog.Errorf("parse remember warp point ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyRememberWarpPointAck(ctx, memo)
			continue
		}
		if fail, ok, err := network.ParseSkillFailAck(pkt); err != nil {
			glog.Errorf("parse skill fail ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillFailAck(ctx, fail)
			continue
		}
		if cast, ok, err := network.ParseSkillCastNotify(pkt); err != nil {
			glog.Errorf("parse skill cast 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillCastNotify(ctx, cast)
			continue
		}
		if groundSkill, ok, err := network.ParseGroundSkillNotify(pkt); err != nil {
			glog.Errorf("parse ground skill 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyGroundSkillNotify(ctx, groundSkill)
			continue
		}
		if skillUnit, ok, err := network.ParseSkillUnitEntry(pkt); err != nil {
			glog.Errorf("parse skill unit 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillUnitEntry(ctx, skillUnit)
			continue
		}
		if skillUnit, ok, err := network.ParseSkillUnitDisappear(pkt); err != nil {
			glog.Errorf("parse skill unit disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillUnitDisappear(skillUnit)
			continue
		}
		if effect, ok, err := network.ParseSpecialEffectNotify(pkt); err != nil {
			glog.Errorf("parse special effect 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySpecialEffectNotify(ctx, effect)
			continue
		}
		if mvp, ok, err := network.ParseMVPNotify(pkt); err != nil {
			glog.Errorf("parse mvp effect 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyMVPNotify(ctx, mvp)
			continue
		}
		if skill, ok, err := network.ParseSkillNoDamageNotify(pkt); err != nil {
			glog.Errorf("parse skill nodamage 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillNoDamageNotify(ctx, skill)
			continue
		}
		if failure, ok, err := network.ParseAttackFailureForDistance(pkt); err != nil {
			glog.Errorf("parse attack distance failure 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyAttackFailureForDistance(ctx, failure)
			continue
		}
		if recovery, ok, err := network.ParseRecovery(pkt); err != nil {
			glog.Errorf("parse recovery 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyRecovery(ctx, recovery)
			continue
		}
		if change, ok, err := network.ParseParameterChange(pkt); err != nil {
			glog.Errorf("parse parameter change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyParameterChange(ctx, change)
			if change.VarID == network.StatusHP {
				m.clearLocalDeathStateIfAlive(ctx)
			}
			continue
		}
		if entry, ok, err := network.ParseActorEntry(pkt); err != nil {
			glog.Errorf("parse actor entry 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.clearActorDeath(entry.ID)
			m.upsertNetworkActor(ctx, entry)
			m.applyWarpPortalEntry(ctx, entry)
		}
	}
	networkErrors := ctx.Network.DrainErrors()
	if handleNetworkDisconnectErrors(ctx, &m.ui.disconnectDialog, networkErrors) {
		return nil, nil
	}
	for _, err := range networkErrors {
		glog.Errorf("network frame error: %v", err)
	}

	m.updatePendingAttack(ctx, "update", false)
	m.processPendingAttack(ctx)
	m.updatePendingPickup(ctx, "update", false)
	m.processPendingPickup(ctx)
	m.skills().UpdatePendingTarget(ctx, "update", false)
	m.skills().ProcessPendingTarget(ctx)
	m.processLockedAttack(ctx)
	now = time.Now()
	m.cleanupDeadActors(ctx, now)
	m.processScheduledActorStops(ctx, now)
	m.processScheduledWalkResumes(ctx, now)
	m.processActorMotionSounds(ctx, now)
	m.processMapSounds(ctx, now)
	m.playDueScheduledSounds(ctx, now)

	if m.tickCooldown > 0 {
		m.tickCooldown--
	}
	if m.tickCooldown == 0 {
		if err := ctx.Network.SendTick(uint32(time.Now().UnixMilli())); err == nil {
			m.tickCooldown = 300
		} else {
			m.tickCooldown = 60
		}
	}

	m.camera.Update(ctx, now)
	if m.mapFade.phase == mapFadeHold {
		return nil, nil
	}
	if m.skills().CancelFromInput(ctx) {
		return nil, nil
	}
	if m.cancelPetCaptureFromInput(ctx) {
		return nil, nil
	}
	if m.openEscapeMenuFromInput(ctx) {
		return nil, nil
	}
	m.skills().AdjustPendingLevelFromWheel(ctx)
	petContextConsumed := m.ui.petContext.Update(ctx)
	if action := m.ui.petContext.PopAction(); action.Kind != gameui.PetContextActionNone {
		m.handlePetContextAction(ctx, action)
		return nil, nil
	}
	if petContextConsumed {
		return nil, nil
	}
	if m.openPetContextFromInput(ctx, now) {
		return nil, nil
	}
	homunculusContextConsumed := m.ui.homunculusContext.Update(ctx)
	if action := m.ui.homunculusContext.PopAction(); action.Kind != gameui.HomunculusContextActionNone {
		m.handleHomunculusContextAction(ctx, action)
		return nil, nil
	}
	if homunculusContextConsumed {
		return nil, nil
	}
	if m.openHomunculusContextFromInput(ctx, now) {
		return nil, nil
	}
	playerContextConsumed := m.ui.playerContext.Update(ctx)
	switch action := m.ui.playerContext.PopAction(); action.Kind {
	case gameui.PlayerContextActionAddFriend:
		m.sendAddFriend(ctx, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteParty:
		m.sendPartyInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteGuild:
		m.sendGuildInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionTrade:
		m.sendTradeRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionSeeEquipment:
		m.sendViewEquipmentRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	}
	if playerContextConsumed {
		return nil, nil
	}
	if m.handleCompanionAICommandClick(ctx, now) {
		return nil, nil
	}
	if m.openPlayerContextFromInput(ctx, now) {
		return nil, nil
	}
	if !m.ui.escapeMenu.IsOpen() && !m.ui.teleportModal.IsOpen() && !m.ui.deathModal.IsOpen() && !m.ui.friendRequest.IsOpen() && !m.ui.friendConfirm.IsOpen() && !m.ui.partyRequest.IsOpen() && !m.ui.guildRequest.IsOpen() && !m.ui.tradeRequest.IsOpen() && !m.ui.settingsWindow.IsOpen() && !m.ui.identifyWindow.IsOpen() && !m.ui.petEggWindow.IsOpen() && !m.ui.petInfoWindow.IsOpen() && !m.ui.petConfirm.IsOpen() && !m.ui.homunculusInfo.IsOpen() && !m.ui.homunculusConfirm.IsOpen() {
		m.updateCameraRotation(ctx)
	}
	if m.updatePetSlotMachine(ctx) {
		return nil, nil
	}
	if m.ui.escapeMenu.IsOpen() {
		if m.ui.escapeMenu.Update(ctx) {
			m.handleEscapeMenuAction(ctx)
			return nil, nil
		}
	}
	if m.ui.friendRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.friendConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.partyRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.guildRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.partyInfo.Update(ctx) {
		return nil, nil
	}
	if m.ui.tradeRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.petConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.petInfoWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.homunculusConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.homunculusInfo.Update(ctx) {
		if action := m.ui.homunculusInfo.PopAction(); action.Kind != gameui.HomunculusInfoActionNone {
			m.handleHomunculusInfoAction(ctx, action)
		}
		return nil, nil
	}
	if m.ui.deathModal.Update(ctx) {
		return nil, nil
	}
	if m.ui.teleportModal.Update(ctx, m) {
		return nil, nil
	}
	if m.ui.npcDialog.Update(ctx) {
		return nil, nil
	}
	if m.updateWhisperWindow(ctx) {
		return nil, nil
	}
	if m.updateChatRoomWindows(ctx) {
		return nil, nil
	}
	if m.updatePartyHelperWindows(ctx) {
		return nil, nil
	}
	if m.updateSkillTextPrompt(ctx) {
		return nil, nil
	}
	if m.ui.console.Update(ctx) {
		return nil, nil
	}
	if m.ui.settingsWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return nil, nil
	}
	if m.ui.characterWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.itemInfoWindow.Update(ctx, m) {
		return nil, nil
	}
	if m.ui.identifyWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.cardWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.makingArrow.Update(ctx) {
		return nil, nil
	}
	if m.ui.petEggWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.inventoryBag.UpdateDrag(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.tradeWindow, &m.ui.equipmentWindow) {
		return nil, nil
	}
	if m.ui.storageWindow.UpdateDrag(ctx, &m.ui.inventoryBag, &m.ui.cartWindow) {
		return nil, nil
	}
	if m.ui.cartWindow.UpdateDrag(ctx, &m.ui.inventoryBag, &m.ui.storageWindow) {
		return nil, nil
	}
	if m.ui.skillWindow.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.shortcutBar.Update(ctx, m) {
		return nil, nil
	}
	if m.ui.inventoryBag.Update(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.tradeWindow, &m.ui.equipmentWindow, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.tradeWindow.Update(ctx, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.equipmentWindow.Update(ctx, &m.ui.itemInfoWindow, &m.ui.cartWindow, m) {
		return nil, nil
	}
	if m.ui.viewEquipWindow.Update(ctx, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.storageWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.cartWindow, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.cartWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.storageWindow, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.changeCartWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.shopWindow.Update(ctx, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.vendingWindow.Update(ctx, &m.ui.itemInfoWindow) {
		return nil, nil
	}
	if m.ui.skillWindow.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.toggleGuildWindowFromInput(ctx) {
		return nil, nil
	}
	if m.ui.friendsWindow.Update(ctx) {
		switch action := m.ui.friendsWindow.PopAction(); action.Kind {
		case gameui.FriendsWindowActionPartySettings:
			m.ui.partySettings.Open(ctx)
		case gameui.FriendsWindowActionPartyCreate:
			m.ui.partyCreate.Open(ctx)
		case gameui.FriendsWindowActionPartyInvite:
			m.ui.partyInvite.Open(ctx)
		case gameui.FriendsWindowActionPartyLeave:
			if ctx.Network == nil {
				m.ui.console.AddErrorMessage("Leave party failed: not connected.")
			} else if err := ctx.Network.SendLeaveParty(); err != nil {
				m.ui.console.AddErrorMessage("Leave party failed.")
				glog.Warnf("leave party failed: %v", err)
			}
		case gameui.FriendsWindowActionFriendWhisper:
			m.ui.whisperWindow.Open(ctx, action.Friend.Name)
		case gameui.FriendsWindowActionFriendDelete:
			m.openDeleteFriendConfirm(ctx, action.Friend)
		case gameui.FriendsWindowActionFriendSettings:
			m.ui.friendSettings.Open(ctx)
		case gameui.FriendsWindowActionFriendBlockWhisper:
			name := strings.TrimSpace(action.Friend.Name)
			if ctx.Network == nil {
				m.ui.console.AddErrorMessage("Block whisper failed: not connected.")
			} else if err := ctx.Network.SendWhisperIgnore(name, false); err != nil {
				m.ui.console.AddErrorMessage("Block whisper failed.")
				glog.Warnf("block whisper failed name=%q: %v", name, err)
			}
		case gameui.FriendsWindowActionPartyMemberInfo:
			m.openPartyMemberInfo(ctx, action.PartyMember)
		case gameui.FriendsWindowActionPartyMemberWhisper:
			m.ui.whisperWindow.Open(ctx, action.PartyMember.Name)
		case gameui.FriendsWindowActionPartyMemberExpel:
			m.openExpelPartyMemberConfirm(ctx, action.PartyMember)
		}
		return nil, nil
	}
	if m.ui.guildWindow.Update(ctx) {
		if action := m.ui.guildWindow.PopAction(); action.RequestMenu {
			m.requestGuildWindowTab(ctx, action.MenuTab)
		} else if action.SelectedEmblemPath != "" {
			m.uploadGuildEmblem(ctx, action.SelectedEmblemPath)
		} else if len(action.MemberPositions) > 0 {
			m.changeGuildMemberPositions(ctx, action.MemberPositions)
		} else if len(action.LevelUpSkillIDs) > 0 {
			m.levelUpGuildSkills(ctx, action.LevelUpSkillIDs)
		} else if action.UpdatePositions {
			m.updateGuildPositions(ctx, action.Positions)
		} else if action.UpdateNotice {
			m.updateGuildNotice(ctx, action.NoticeSubject, action.Notice)
		}
		return nil, nil
	}
	if m.ui.friendSettings.Update(ctx) {
		return nil, nil
	}
	if m.ui.partySettings.Update(ctx) {
		return nil, nil
	}
	if m.ui.statsWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.basicMenu.Update(ctx, m.basicMenuCallbacks(ctx)) {
		return nil, nil
	}
	m.ui.minimap.Update(ctx)
	removeExpiredStatusEffects(ctx.Session, now)
	m.ui.statusIcons.Update(ctx, now)
	m.updateCompanionAI(ctx, now)
	m.updateBot(ctx, now)
	pointerBlocked := uiPointerBlocked(ctx)
	if !pointerBlocked {
		m.updateCameraZoom(ctx)
	}

	if !pointerBlocked && ctx.Input.MouseJustPressed(input.MouseButtonLeft) && m.walkReady(now) {
		m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		if m.handlePetCaptureClick(ctx, projection, now) {
			return nil, nil
		}
		if m.pendingSkill.skill.ID != 0 {
			m.skills().HandleClick(ctx, projection, now)
			return nil, nil
		}
		playerX, playerY := currentPlayerCell(ctx, now)
		if actor, ok := m.hoveredVendingBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click vending target mouse=%d,%d id=%d name=%q shop=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.VendingName, playerX, playerY, actor.X, actor.Y)
			m.requestVendingList(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := m.hoveredChatRoomBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click chat room target mouse=%d,%d id=%d name=%q room=%d title=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.ChatRoomID, actor.ChatRoomTitle, playerX, playerY, actor.X, actor.Y)
			m.requestChatRoomEnter(ctx, actor)
			return nil, nil
		}
		if item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click pickup target mouse=%d,%d id=%d item_id=%d amount=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, item.ID, item.ItemID, item.Amount, playerX, playerY, item.X, item.Y)
			m.clearLockedAttack()
			m.clearAttackFocus()
			m.requestPickup(ctx, item, "click")
			return nil, nil
		}
		if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			glog.Debugf("click attack target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.requestAttack(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			glog.Debugf("click npc talk target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.clearAttackFocus()
			m.requestNPCTalk(ctx, actor, "click")
			return nil, nil
		}
		if targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok {
			glog.Debugf("click walk target mouse=%d,%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, playerX, playerY, targetX, targetY)
			m.clearLockedAttack()
			m.clearAttackFocus()
			if shouldUseTurnOnlyGroundClick(ctx) {
				m.requestChangeDirection(ctx, targetX, targetY, "click")
				return nil, nil
			}
			m.requestWalk(ctx, targetX, targetY, "click")
		}
	}
	if m.updateHeldWalk(ctx, pointerBlocked, now) {
		return nil, nil
	}
	return nil, nil
}

func (m *WorldMode) handleEscapeMenuAction(ctx client.Context) {
	switch m.ui.escapeMenu.ConsumeAction() {
	case gameui.EscapeMenuActionCharacterSelect:
		m.ui.escapeMenu.RequestCharacterSelect(ctx)
	case gameui.EscapeMenuActionSettings:
		m.ui.settingsWindow.OpenWindow(ctx)
	}
}

func (m *WorldMode) applyQuitGameAck(ctx client.Context, ack network.QuitGameAck) {
	handled := m.ui.escapeMenu.ApplyQuitGameAck(ack)
	if m.ui.deathModal.ApplyQuitGameAck(ack) {
		handled = true
	}
	if ack.Allowed {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
		return
	}
	if handled {
		m.ui.console.AddErrorMessage("You cannot exit the game right now.")
	}
}

func (m *WorldMode) openEscapeMenuFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.escapeMenu.IsOpen() || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	if m.ui.deathModal.IsOpen() || m.ui.teleportModal.IsOpen() || m.ui.friendRequest.IsOpen() || m.ui.friendConfirm.IsOpen() || m.ui.partyRequest.IsOpen() || m.ui.guildRequest.IsOpen() || m.ui.tradeRequest.IsOpen() {
		return false
	}
	m.ui.escapeMenu.Toggle(ctx)
	return true
}

func (m *WorldMode) basicMenuCallbacks(ctx client.Context) gameui.BasicMenuCallbacks {
	return gameui.BasicMenuCallbacks{
		OnStatus: func() { m.ui.statsWindow.Toggle(ctx) },
		OnOption: func() { m.ui.escapeMenu.Toggle(ctx) },
		OnItems:  func() { m.ui.inventoryBag.Toggle(ctx) },
		OnEquip:  func() { m.ui.equipmentWindow.Toggle(ctx) },
		OnSkill:  func() { m.ui.skillWindow.Toggle(ctx) },
		OnMap:    func() { m.ui.minimap.Toggle(ctx) },
		OnComm:   func() { m.ui.chatRoomCreate.Open(ctx) },
		OnFriend: func() { m.ui.friendsWindow.Toggle(ctx) },
	}
}

func (m *WorldMode) toggleGuildWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.console.Active() {
		return false
	}
	if !ctx.Input.Pressed(input.KeyAlt) || !ctx.Input.JustPressed(input.KeyG) {
		return false
	}
	m.toggleGuildWindow(ctx)
	return true
}

func (m *WorldMode) toggleGuildWindow(ctx client.Context) {
	wasOpen := m.ui.guildWindow.IsOpen()
	m.setGuildEmblemOptions(ctx)
	m.ui.guildWindow.Toggle(ctx)
	if wasOpen || !m.ui.guildWindow.IsOpen() || ctx.Network == nil {
		return
	}
	m.requestGuildWindowTab(ctx, 0)
}

const maxGuildMenuRequestType = 4

func (m *WorldMode) requestGuildWindowTab(ctx client.Context, tab uint32) {
	if tab > maxGuildMenuRequestType {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild info request failed.")
		return
	}
	if err := ctx.Network.SendGuildMenuRequest(tab); err != nil {
		m.ui.console.AddErrorMessage("Guild info request failed.")
		glog.Warnf("guild menu request failed tab=%d: %v", tab, err)
	}
	if tab == 0 {
		m.requestSessionGuildEmblem(ctx)
	}
}

func (m *WorldMode) requestSessionGuildEmblem(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	guildID := ctx.Session.GuildID
	version := ctx.Session.EmblemVersion
	if guildID == 0 {
		guildID = ctx.Session.Guild.ID
	}
	if version == 0 {
		version = ctx.Session.Guild.EmblemVersion
	}
	m.requestGuildEmblem(ctx, guildID, version, true)
}

func (m *WorldMode) handleMapChange(ctx client.Context, change network.MapChange) Mode {
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.clearLocalActorAction(ctx)
	m.scheduledStops = nil
	m.ui.npcDialog.ResetPublished(ctx)
	m.ui.teleportModal = gameui.TeleportModal{}
	m.clearLocalDeathState(ctx)
	currentMap := ctx.World.MapName
	reuseLoadedMap := !change.ServerMove && sameLoadedMap(ctx, change.MapName)
	glog.Debugf("map change current=%s target=%s x=%d y=%d server_move=%t addr=%s port=%d reuse_loaded=%t", currentMap, change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port, reuseLoadedMap)
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	applyWarpPosition(ctx, change.X, change.Y)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	if reuseLoadedMap {
		zoom, zoomTarget := m.camera.zoom, m.camera.zoomTarget
		m.camera.Reset()
		m.camera.zoom = zoom
		m.camera.zoomTarget = zoomTarget
		m.camera.Update(ctx, time.Now())
		if ctx.Network != nil {
			if err := ctx.Network.SendLoadEndAck(); err != nil {
				glog.Warnf("same-map warp load ack failed map=%s x=%d y=%d: %v", change.MapName, change.X, change.Y, err)
			} else {
				m.tickCooldown = 1
			}
		}
		return nil
	}
	if change.ServerMove {
		ctx.Session.Zone.Address = change.Address
		ctx.Session.Zone.Port = change.Port
		dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := ctx.Network.Connect(dialCtx, change.Address, int(change.Port))
		cancel()
		if err != nil {
			glog.Warnf("map reconnect failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			openConnectionFailedDialog(ctx, &m.ui.disconnectDialog)
			return nil
		}
		if err := ctx.Network.SendMapServerEnter(ctx.Session.AccountID, ctx.Session.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex); err != nil {
			glog.Warnf("map re-enter failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			message := disconnectMessageText(ctx.Resources, disconnectMessage{2, "Disconnected from Server."})
			openDisconnectDialog(ctx, &m.ui.disconnectDialog, message)
			return nil
		}
		m.pendingWarp = true
		return nil
	}
	return m.nextWorldMode()
}

func (m *WorldMode) nextWorldMode() *WorldMode {
	next := NewWorldMode()
	next.camera.zoom = m.camera.zoom
	next.camera.zoomTarget = m.camera.zoomTarget
	next.ui.console = m.ui.console
	next.ui.characterWindow = m.ui.characterWindow
	next.ui.basicMenu = m.ui.basicMenu
	next.ui.inventoryBag = m.ui.inventoryBag
	next.ui.equipmentWindow = m.ui.equipmentWindow
	next.ui.cartWindow = m.ui.cartWindow
	next.ui.itemInfoWindow = m.ui.itemInfoWindow
	next.ui.cardWindow = m.ui.cardWindow
	next.ui.petEggWindow = m.ui.petEggWindow
	next.ui.petInfoWindow = m.ui.petInfoWindow
	next.ui.homunculusInfo = m.ui.homunculusInfo
	next.petProperty = m.petProperty
	next.hasPetProperty = m.hasPetProperty
	next.petOldFullness = m.petOldFullness
	next.petLastTalk = m.petLastTalk
	next.ui.statsWindow = m.ui.statsWindow
	next.ui.skillWindow = m.ui.skillWindow
	next.ui.friendsWindow = m.ui.friendsWindow
	next.ui.guildWindow = m.ui.guildWindow
	next.ui.friendSettings = m.ui.friendSettings
	next.ui.whisperWindow = m.ui.whisperWindow
	next.ui.chatRoomCreate = m.ui.chatRoomCreate
	next.ui.chatRoom = m.ui.chatRoom
	next.pendingChatRoom = m.pendingChatRoom
	next.ui.partySettings = m.ui.partySettings
	next.ui.settingsWindow = m.ui.settingsWindow
	next.ui.partyCreate = m.ui.partyCreate
	next.ui.partyInvite = m.ui.partyInvite
	next.ui.skillTextPrompt = m.ui.skillTextPrompt
	next.ui.shortcutBar = m.ui.shortcutBar
	next.ui.minimap = m.ui.minimap
	next.startMapFadeIn(time.Now())
	m.companionAI.close()
	return next
}

func (m *WorldMode) nextCharacterSelectMode(ctx client.Context) *LoginMode {
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	if ctx.Session != nil {
		ctx.Session.Playing = false
		ctx.Session.Storage = session.Storage{}
		ctx.Session.Cart = session.Cart{}
	}
	next := NewCharacterSelectMode(ctx, m.ui.console)
	return next
}

func (m *WorldMode) updateDisconnectDialog(ctx client.Context) bool {
	if !m.ui.disconnectDialog.IsOpen() {
		return false
	}
	return m.ui.disconnectDialog.Update(ctx)
}

func sameLoadedMap(ctx client.Context, mapName string) bool {
	if ctx.World == nil || ctx.World.MapName == "" || mapName == "" {
		return false
	}
	if !strings.EqualFold(ctx.World.MapName, mapName) {
		return false
	}
	return ctx.World.GND != nil || ctx.World.GAT != nil
}

func (m *WorldMode) requestNPCTalk(ctx client.Context, actor worldstate.Actor, source string) {
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	m.clearLockedAttack()
	if err := ctx.Network.SendNPCContact(actor.ID); err == nil {
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		playerX, playerY := currentPlayerCell(ctx, time.Now())
		glog.Warnf("%s npc talk failed target=%d player=%d,%d target=%d,%d: %v", source, actor.ID, playerX, playerY, actor.X, actor.Y, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) Draw(ctx client.Context, screen *render.Frame) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	fog := sceneFogFromMap(ctx.Resources, ctx.World.MapName, ctx.Config.Fog)
	clearWorldScene(screen, ctx.World.MapName)
	var actorOverlays []sceneActorDrawEntry
	screen.SetCamera3D(projection.RenderCameraWithFog(fog))
	vertexFog := sceneFog{}

	if ctx.World.GND != nil {
		m.drawGNDMeshes(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection)
		m.drawGNDWater(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection, now, vertexFog)
		if !ctx.Config.Render.NoUI {
			m.drawTileCursor(screen, ctx, projection)
		}
		if ctx.World.RSW != nil && len(ctx.World.RSM) > 0 {
			actorOverlays = m.drawSceneModelsAndActors(screen, ctx, projection, vertexFog, now)
		} else {
			m.drawGroundItems(screen, ctx, projection, now)
			actorOverlays = m.drawSceneActors(screen, ctx, projection)
		}
	} else if ctx.World.GAT != nil {
		m.drawGroundItems(screen, ctx, projection, now)
		actorOverlays = m.drawSceneActors(screen, ctx, projection)
	}

	if !ctx.Config.Render.NoUI {
		m.drawSceneActorOverlays(screen, ctx, projection, now, actorOverlays)
	}
	m.drawRSWEffects(screen, ctx, projection, now)
	m.drawMapWeatherEffects(screen, ctx, projection, now)
	m.drawWorldEffects(screen, ctx, projection, now)
	m.drawDamageFloaters(screen, ctx, projection, now)

	if !ctx.Config.Render.NoUI {
		m.ui.inventoryBag.Draw(screen, ctx, m)
		m.ui.storageWindow.Draw(screen, ctx, m)
		m.ui.cartWindow.Draw(screen, ctx, m)
		m.ui.shopWindow.Draw(screen, ctx, m)
		m.ui.vendingWindow.Draw(screen, ctx, m)
		m.ui.skillWindow.Draw(screen, ctx, m)
		m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
		m.ui.deathModal.Draw(screen, ctx, width, height)
	}
}

func (m *WorldMode) DrawOverlay(ctx client.Context, screen *render.Frame) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	m.drawMapFade(screen, now)
	if !ctx.Config.Render.NoUI {
		m.drawUIDragGhosts(screen, ctx)
		m.drawPetSlotMachine(screen, ctx, now)
		m.drawROCursor(screen, ctx, projection, now)
	}
}

func (m *WorldMode) DrawUIOverlay(ctx client.Context, screen *render.Frame) {
	if ctx.Config.Render.NoUI {
		return
	}
	m.ui.inventoryBag.DrawTooltip(screen)
	m.ui.equipmentWindow.DrawTooltip(screen)
	m.ui.itemInfoWindow.DrawTooltip(screen)
	m.ui.skillWindow.DrawTooltip(screen)
	m.ui.guildWindow.DrawTooltip(screen)
	m.ui.shortcutBar.DrawTooltip(screen)
}

func (m *WorldMode) drawUIDragGhosts(screen *render.Frame, ctx client.Context) {
	m.ui.inventoryBag.DrawDragGhost(screen, ctx, m)
	m.ui.storageWindow.DrawDragGhost(screen, ctx, m)
	m.ui.cartWindow.DrawDragGhost(screen, ctx, m)
	m.ui.shopWindow.DrawDragGhost(screen, ctx, m)
	m.ui.vendingWindow.DrawDragGhost(screen, ctx, m)
	m.ui.skillWindow.DrawDragGhost(screen, ctx, m)
}

func clickedAttackTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actorCanBeAttackClicked(ctx, actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
		scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.x), float64(point.y), scale) {
			continue
		}
		dx := float64(point.x) - float64(mouseX)
		dy := float64(point.y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedSkillTarget(ctx client.Context, projection sceneProjection, skill session.Skill, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range skillTargetCandidates(ctx, skill) {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actorCanBeSkillTargeted(ctx, skill, actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
		scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.x), float64(point.y), scale) {
			continue
		}
		dx := float64(point.x) - float64(mouseX)
		dy := float64(point.y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedPlayerTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actorCanOpenPlayerContext(ctx, actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
		scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.x), float64(point.y), scale) {
			continue
		}
		dx := float64(point.x) - float64(mouseX)
		dy := float64(point.y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func skillTargetCandidates(ctx client.Context, skill session.Skill) []worldstate.Actor {
	if ctx.World == nil {
		return nil
	}
	candidates := make([]worldstate.Actor, 0, len(ctx.World.Actors)+1)
	if skillTargetFlagsIncludeSelfPick(skill.Type) {
		if actor, ok, _ := actorForCombatID(ctx, localSkillTarget(ctx)); ok {
			candidates = append(candidates, actor)
		}
	}
	for _, actor := range ctx.World.Actors {
		candidates = append(candidates, actor)
	}
	return candidates
}

func skillTargetFlagsIncludeSelfPick(flags uint32) bool {
	return flags&(skillTargetFriend|skillTargetSelf) != 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampGameInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type sceneDrawEntry struct {
	depth       float64
	actorIndex  int
	shadowIndex int
	itemIndex   int
}

func (m *WorldMode) drawSceneModelsAndActors(screen *render.Frame, ctx client.Context, projection sceneProjection, fog sceneFog, now time.Time) []sceneActorDrawEntry {
	m.drawRSMModels(screen, ctx.Resources, ctx.World.RSW, ctx.World.RSM, ctx.World.GND, projection, fog, now)
	actors := m.collectSceneActorEntries(screen, ctx, projection)
	items := m.collectSceneItemEntries(screen, ctx, projection, now)
	entries := make([]sceneDrawEntry, 0, len(actors)+len(items))
	for i, item := range items {
		entries = append(entries, sceneDrawEntry{depth: item.depth, actorIndex: -1, shadowIndex: -1, itemIndex: i})
	}
	for i, actor := range actors {
		if actor.castShadow {
			entries = append(entries, sceneDrawEntry{depth: actor.shadowDepth, actorIndex: -1, shadowIndex: i, itemIndex: -1})
		}
		entries = append(entries, sceneDrawEntry{depth: actor.depth, actorIndex: i, shadowIndex: -1, itemIndex: -1})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].depth > entries[j].depth
	})
	for _, entry := range entries {
		if entry.shadowIndex >= 0 {
			m.drawActorShadowEntry(screen, ctx, projection, actors[entry.shadowIndex])
			continue
		}
		if entry.itemIndex >= 0 {
			m.drawGroundItemEntry3D(screen, projection, items[entry.itemIndex])
			continue
		}
		m.drawSceneActorEntry(screen, ctx, projection, actors[entry.actorIndex])
	}
	return actors
}

func loadGAT(manager *res.Manager, mapName string) (*res.GAT, string, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".gat",
		"data/" + base + ".gat",
		base + ".gat",
	}
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err != nil {
			continue
		}
		gat, err := res.ParseGAT(data)
		if err != nil {
			return nil, candidate, err
		}
		return gat, candidate, nil
	}
	return nil, "", fmt.Errorf("gat not found for map %s", mapName)
}

func loadRSW(manager *res.Manager, mapName string) (*res.RSW, string, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".rsw",
		"data/" + base + ".rsw",
		base + ".rsw",
	}
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err != nil {
			continue
		}
		rsw, err := res.ParseRSW(data)
		if err != nil {
			return nil, candidate, err
		}
		return rsw, candidate, nil
	}
	return nil, "", fmt.Errorf("rsw not found for map %s", mapName)
}

func loadRSMModels(manager *res.Manager, rsw *res.RSW, limit int) (map[string]*res.RSM, int) {
	if rsw == nil || limit == 0 {
		return nil, 0
	}
	models := make(map[string]*res.RSM)
	failures := 0
	for _, placement := range rsw.Models {
		if placement.Filename == "" {
			continue
		}
		if _, ok := models[placement.Filename]; ok {
			continue
		}
		if limit > 0 && len(models) >= limit {
			break
		}

		rsm, err := loadRSMModel(manager, placement.Filename)
		if err != nil {
			failures++
			continue
		}
		models[placement.Filename] = rsm
	}
	return models, failures
}

func loadRSMModel(manager *res.Manager, filename string) (*res.RSM, error) {
	var data []byte
	for _, candidate := range res.RSMModelCandidates(filename) {
		raw, err := manager.ReadFile(candidate)
		if err == nil {
			data = raw
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("rsm not found: %s", filename)
	}
	return res.ParseRSM(data)
}

type screenPoint struct {
	x float32
	y float32
}

type texturePoint struct {
	u float32
	v float32
}

func quadNormal(verts [4]modelPoint3) modelPoint3 {
	normal := normalize3(cross3(sub3(verts[1], verts[0]), sub3(verts[2], verts[0])))
	if normal == (modelPoint3{}) {
		return modelPoint3{y: 1}
	}
	return normal
}

type sceneLighting struct {
	direction modelPoint3
	diffuse   modelPoint3
	ambient   modelPoint3
	env       modelPoint3
	opacity   float64
}

func sceneLightingFromRSW(rsw *res.RSW) sceneLighting {
	longitude := 45.0
	latitude := 45.0
	diffuse := modelPoint3{x: 1, y: 1, z: 1}
	ambient := modelPoint3{}
	opacity := 1.0
	if rsw != nil {
		longitude = float64(rsw.Light.Longitude)
		latitude = float64(rsw.Light.Latitude)
		diffuse = modelPoint3{x: float64(rsw.Light.Diffuse[0]), y: float64(rsw.Light.Diffuse[1]), z: float64(rsw.Light.Diffuse[2])}
		ambient = modelPoint3{x: float64(rsw.Light.Ambient[0]), y: float64(rsw.Light.Ambient[1]), z: float64(rsw.Light.Ambient[2])}
		opacity = float64(rsw.Light.Opacity)
	}
	longitude = degreesToRadians(longitude)
	latitude = degreesToRadians(latitude)
	dir := normalize3(modelPoint3{
		x: -math.Cos(longitude) * math.Sin(latitude),
		y: -math.Cos(latitude),
		z: -math.Sin(longitude) * math.Sin(latitude),
	})
	if dir == (modelPoint3{}) {
		dir = normalize3(modelPoint3{x: -0.5, y: -0.7, z: -0.5})
	}
	diffuse = clampUnitPoint(diffuse)
	ambient = clampUnitPoint(ambient)
	opacity = math.Max(0, math.Min(1, opacity))
	env := modelPoint3{
		x: 1 - (1-ambient.x)*(1-diffuse.x),
		y: 1 - (1-ambient.y)*(1-diffuse.y),
		z: 1 - (1-ambient.z)*(1-diffuse.z),
	}
	return sceneLighting{
		direction: dir,
		diffuse:   diffuse,
		ambient:   ambient,
		env:       clampUnitPoint(env),
		opacity:   opacity,
	}
}

func (l sceneLighting) groundScale(normal modelPoint3) modelPoint3 {
	weight := math.Max(dot3(normalize3(normal), l.direction), 0)
	return modelPoint3{
		x: clampUnit(l.ambient.x+l.diffuse.x*weight) * clampUnit(l.env.x),
		y: clampUnit(l.ambient.y+l.diffuse.y*weight) * clampUnit(l.env.y),
		z: clampUnit(l.ambient.z+l.diffuse.z*weight) * clampUnit(l.env.z),
	}
}

func (l sceneLighting) modelScale(normal modelPoint3) modelPoint3 {
	weight := math.Max(dot3(normalize3(normal), l.direction), 0.5)
	return l.modelScaleFromWeight(weight)
}

func (l sceneLighting) modelScaleNormalized(normal modelPoint3) modelPoint3 {
	weight := math.Max(dot3(normal, l.direction), 0.5)
	return l.modelScaleFromWeight(weight)
}

func (l sceneLighting) modelScaleFromWeight(weight float64) modelPoint3 {
	return modelPoint3{
		x: clampUnit(l.ambient.x+l.diffuse.x*weight) * clampUnit(l.env.x),
		y: clampUnit(l.ambient.y+l.diffuse.y*weight) * clampUnit(l.env.y),
		z: clampUnit(l.ambient.z+l.diffuse.z*weight) * clampUnit(l.env.z),
	}
}

func clampUnitPoint(point modelPoint3) modelPoint3 {
	return modelPoint3{x: clampUnit(point.x), y: clampUnit(point.y), z: clampUnit(point.z)}
}

func clampUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func textureColor(name string) color.RGBA {
	if name == "" {
		return color.RGBA{R: 78, G: 86, B: 78, A: 255}
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.ToLower(name)))
	value := hash.Sum32()
	return color.RGBA{
		R: 70 + uint8(value&0x3f),
		G: 80 + uint8((value>>8)&0x4f),
		B: 68 + uint8((value>>16)&0x4f),
		A: 255,
	}
}

func clampColor(value float64) uint8 {
	return uint8(min(255, max(0, int(value))))
}

func drawColoredSurfaceTints3DAlpha(screen *render.Frame, white *render.Image, verts [4]modelPoint3, indices []uint16, colors [4]color.RGBA) {
	drawColoredSurfaceTints3DWithOptions(screen, white, verts, indices, colors, triangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}

func drawColoredSurfaceTints3DWithOptions(screen *render.Frame, white *render.Image, verts [4]modelPoint3, indices []uint16, colors [4]color.RGBA, options *render.DrawTrianglesOptions) {
	vertices := []render.Vertex3D{
		coloredSurfaceVertex3D(verts[0], 0, 0, colors[0]),
		coloredSurfaceVertex3D(verts[1], 1, 0, colors[1]),
		coloredSurfaceVertex3D(verts[2], 1, 1, colors[2]),
		coloredSurfaceVertex3D(verts[3], 0, 1, colors[3]),
	}
	screen.DrawTriangles3DOwned(vertices, indices, white, options)
}

func drawTexturedSurface3DAlpha(screen *render.Frame, texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA) {
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, indices, tints, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
}

func drawTexturedSurface3DWithOptions(screen *render.Frame, texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
	bounds := texture.Bounds()
	w := float32(bounds.Dx())
	h := float32(bounds.Dy())
	vertices := []render.Vertex3D{
		texturedSurfaceVertex3D(verts[0], uvs[0], tints[0], w, h),
		texturedSurfaceVertex3D(verts[1], uvs[1], tints[1], w, h),
		texturedSurfaceVertex3D(verts[2], uvs[2], tints[2], w, h),
		texturedSurfaceVertex3D(verts[3], uvs[3], tints[3], w, h),
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

func coloredSurfaceVertex3D(point modelPoint3, u, v float32, tint color.RGBA) render.Vertex3D {
	return render.Vertex3D{
		X:      float32(point.x),
		Y:      float32(point.y),
		Z:      float32(point.z),
		SrcX:   u,
		SrcY:   v,
		ColorR: float32(tint.R) / 255,
		ColorG: float32(tint.G) / 255,
		ColorB: float32(tint.B) / 255,
		ColorA: float32(tint.A) / 255,
		DepthX: float32(point.x),
		DepthY: float32(point.y),
		DepthZ: float32(point.z),
	}
}

func texturedSurfaceVertex3D(point modelPoint3, uv texturePoint, tint color.RGBA, textureWidth, textureHeight float32) render.Vertex3D {
	return render.Vertex3D{
		X:      float32(point.x),
		Y:      float32(point.y),
		Z:      float32(point.z),
		SrcX:   uv.u * textureWidth,
		SrcY:   uv.v * textureHeight,
		ColorR: float32(tint.R) / 255,
		ColorG: float32(tint.G) / 255,
		ColorB: float32(tint.B) / 255,
		ColorA: float32(tint.A) / 255,
		DepthX: float32(point.x),
		DepthY: float32(point.y),
		DepthZ: float32(point.z),
	}
}
