package game

import (
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
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
	speechBubbles     map[uint32]speechBubble
	gndNormalSource   *res.GND
	gndTopNormals     [][4]modelPoint3
	minimap           gameui.Minimap
	statusIcons       gameui.StatusIcons
	console           gameui.ChatConsole
	npcDialog         gameui.NPCDialog
	escapeMenu        gameui.EscapeMenu
	teleportModal     gameui.TeleportModal
	deathModal        gameui.DeathModal
	friendRequest     gameui.ConfirmModal
	partyRequest      gameui.ConfirmModal
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
	statsWindow       gameui.StatsWindow
	skillWindow       gameui.SkillWindow
	friendsWindow     gameui.FriendsWindow
	partySettings     gameui.PartySettingsWindow
	playerContext     gameui.PlayerContextMenu
	tradeWindow       gameui.TradeWindow
	pendingTradeName  string
	settingsWindow    gameui.SettingsWindow
	shortcutBar       gameui.ShortcutBar
	mapFade           mapFadeState
	hoveredWalk       hoveredWalkCellCache
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

type scheduledActorStop struct {
	id         uint32
	at         time.Time
	resumeWalk bool
	resumeAt   time.Time
}

type scheduledWalkResume struct {
	id  uint32
	at  time.Time
	toX int
	toY int
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
	zoom := m.camera.zoom
	m.camera.Reset()
	m.camera.zoom = zoom
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
	m.shortcutBar.Load(ctx)
	m.npcDialog.ResetPublished(ctx)
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	playerStatus := ""
	character := selectedCharacter(ctx.Session)
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
		log.Printf("actor shadow resources unavailable: %s", status)
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
		log.Printf("cursor resources unavailable: %s", status)
	}
	render.SetCursorMode(render.CursorModeHidden)
	log.Printf("player sprite resources char_id=%d name=%s job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, character.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
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
	m.basicMenu.Rebind(ctx, m.basicMenuCallbacks(ctx))
	m.inventoryBag.Rebind(ctx, &m.itemInfoWindow, &m.cartWindow)
	m.equipmentWindow.Rebind(ctx, &m.itemInfoWindow, &m.cartWindow, m)
	m.cartWindow.Rebind(ctx, &m.itemInfoWindow)
	m.itemInfoWindow.Rebind(ctx, m)
	m.statsWindow.Rebind(ctx)
	m.skillWindow.Rebind(ctx, m)
	m.friendsWindow.Rebind(ctx)
	m.partySettings.Rebind(ctx)
	m.settingsWindow.Rebind(ctx)
	m.shortcutBar.ResetOverlay(ctx)
}

func (m *WorldMode) playMapBGM(ctx client.Context, rswName string) {
	if ctx.Audio == nil {
		return
	}
	_, err := ctx.Audio.PlayMap(rswName)
	if err != nil {
		log.Printf("bgm failed map=%s: %v", rswName, err)
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

	for _, pkt := range ctx.Network.DrainPackets() {
		if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
			log.Printf("parse chat message 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySpeechBubble(ctx, chat, now)
			addConsoleMessage(&m.console, ctx.Resources, chat)
			continue
		}
		if whisper, ok, err := network.ParseWhisperMessage(pkt); err != nil {
			log.Printf("parse whisper message 0x%04X: %v", pkt.ID, err)
		} else if ok {
			addWhisperMessage(&m.console, whisper)
			continue
		}
		if ack, ok, err := network.ParseWhisperAck(pkt); err != nil {
			log.Printf("parse whisper ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			addWhisperAck(&m.console, ctx.Resources, ack)
			continue
		}
		if emotion, ok, err := network.ParseEmotionNotify(pkt); err != nil {
			log.Printf("parse emotion 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyEmotionNotify(ctx, emotion)
			continue
		}
		if change, ok, err := network.ParseMapChange(pkt); err != nil {
			log.Printf("parse map change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.startMapFadeOut(change, time.Now())
			return nil, nil
		}
		if enter, err := network.ParseMapAcceptEnter(pkt); err == nil {
			applyMapAcceptEnter(ctx, enter)
			if m.pendingWarp {
				m.pendingWarp = false
				return m.nextWorldMode(), nil
			}
			continue
		}
		if ack, ok, err := network.ParseActorNameAck(pkt); err != nil {
			log.Printf("parse actor name ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyActorNameAck(ctx, ack)
			continue
		}
		if ack, ok, err := network.ParseRestartAck(pkt); err != nil {
			log.Printf("parse restart ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if m.deathModal.ApplyRestartAck(ack) {
				return m.nextCharacterSelectMode(ctx), nil
			}
			if m.escapeMenu.ApplyRestartAck(ack) {
				return m.nextCharacterSelectMode(ctx), nil
			}
			continue
		}
		if dialog, ok, err := network.ParseNPCDialog(pkt); err != nil {
			log.Printf("parse npc dialog 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.npcDialog.Apply(dialog)
			continue
		}
		if ack, ok, err := network.ParseSelfMoveAck(pkt); err != nil {
			log.Printf("parse self move ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySelfMoveAck(ctx, ack)
			m.clearLocalActorAction(ctx)
			log.Printf("walk ack from=%d,%d to=%d,%d tick=%d", ack.FromX, ack.FromY, ack.ToX, ack.ToY, ack.ServerTick)
			m.continuePendingAttack(ctx, "walk ack")
			m.continuePendingPickup(ctx, "walk ack")
			m.skills().ContinuePendingTarget(ctx, "walk ack")
			continue
		}
		if position, ok, err := network.ParseActorSetPosition(pkt); err != nil {
			log.Printf("parse actor set position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if isLocalActor(ctx, position.ID) {
				log.Printf("local position fix id=%d x=%d y=%d", position.ID, position.X, position.Y)
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
			log.Printf("parse actor jump position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if isLocalActor(ctx, position.ID) {
				log.Printf("local jump position id=%d x=%d y=%d", position.ID, position.X, position.Y)
			}
			applyActorJumpPosition(ctx, position)
			continue
		}
		if item, ok, err := network.ParseFloorItemEntry(pkt); err != nil {
			log.Printf("parse floor item entry 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyFloorItemEntry(ctx, item)
			continue
		}
		if disappear, ok, err := network.ParseFloorItemDisappear(pkt); err != nil {
			log.Printf("parse floor item disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyFloorItemDisappear(ctx, disappear)
			continue
		}
		if pickup, ok, err := network.ParseItemPickupAck(pkt); err != nil {
			log.Printf("parse item pickup ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyItemPickupAck(ctx, pickup)
			if pickup.Result == 0 {
				message := formatPickupConsoleMessage(ctx.Resources, pickup)
				log.Printf("console pickup message item_id=%d amount=%d text=%q", pickup.ItemID, pickup.Amount, message)
				m.console.AddBlueMessage("%s", message)
			} else {
				m.console.AddErrorMessage("Pickup failed item %d result=%d", pickup.ItemID, pickup.Result)
			}
			continue
		}
		if useAck, ok, err := network.ParseUseItemAck(pkt); err != nil {
			log.Printf("parse use item ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("use item ack index=%d item=%d aid=%d amount=%d result=%d", useAck.Index, useAck.ItemID, useAck.AID, useAck.Amount, useAck.Result)
			m.addItemUseEffect(ctx, useAck)
			applyUseItemAck(ctx, useAck)
			if useAck.Result != 0 && useAck.Amount == 0 && m.shortcutBar.ClearDepletedItem(ctx, useAck.Index, useAck.ItemID) {
				log.Printf("shortcut item depleted index=%d item=%d", useAck.Index, useAck.ItemID)
			}
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if identifyList, ok, err := network.ParseItemIdentifyList(pkt); err != nil {
			log.Printf("parse item identify list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("item identify list indexes=%v", identifyList.Indexes)
			m.identifyWindow.OpenList(ctx, identifyList)
			continue
		}
		if identifyAck, ok, err := network.ParseItemIdentifyAck(pkt); err != nil {
			log.Printf("parse item identify ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("item identify ack index=%d success=%v", identifyAck.Index, identifyAck.Success)
			applyItemIdentifyAck(ctx, identifyAck)
			m.identifyWindow.ApplyAck(ctx, identifyAck)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if items, ok, err := network.ParseInventoryItemList(pkt); err != nil {
			log.Printf("parse inventory item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryItemList(ctx, items)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if itemDelete, ok, err := network.ParseInventoryItemDelete(pkt); err != nil {
			log.Printf("parse inventory item delete 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryItemDelete(ctx, itemDelete)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if equipAck, ok, err := network.ParseInventoryEquipAck(pkt); err != nil {
			log.Printf("parse inventory equip ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyInventoryEquipAck(ctx, equipAck)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if arrow, ok, err := network.ParseEquippedArrow(pkt); err != nil {
			log.Printf("parse equipped arrow 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("equipped arrow index=%d", arrow.Index)
			applyEquippedArrow(ctx, arrow)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if storageItems, ok, err := network.ParseStorageItemList(pkt); err != nil {
			log.Printf("parse storage item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemList(ctx, storageItems)
			m.storageWindow.OpenWindow(ctx)
			continue
		}
		if cartItems, ok, err := network.ParseCartItemList(pkt); err != nil {
			log.Printf("parse cart item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartItemList(ctx, cartItems)
			m.cartWindow.ClampScroll(ctx.Session)
			m.cartWindow.Refresh(ctx, &m.itemInfoWindow)
			continue
		}
		if storageAmount, ok, err := network.ParseStorageAmount(pkt); err != nil {
			log.Printf("parse storage amount 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageAmount(ctx, storageAmount)
			m.storageWindow.OpenWindow(ctx)
			continue
		}
		if cartAmount, ok, err := network.ParseCartAmount(pkt); err != nil {
			log.Printf("parse cart amount 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartAmount(ctx, cartAmount)
			m.cartWindow.ClampScroll(ctx.Session)
			m.cartWindow.Refresh(ctx, &m.itemInfoWindow)
			continue
		}
		if friends, ok, err := network.ParseFriendsList(pkt); err != nil {
			log.Printf("parse friends list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendsList(ctx, friends)
			continue
		}
		if friendState, ok, err := network.ParseFriendState(pkt); err != nil {
			log.Printf("parse friend state 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendState(ctx, friendState)
			continue
		}
		if friendRequest, ok, err := network.ParseFriendRequest(pkt); err != nil {
			log.Printf("parse friend request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openFriendRequest(ctx, friendRequest)
			continue
		}
		if friendAdded, ok, err := network.ParseFriendAddResult(pkt); err != nil {
			log.Printf("parse friend add result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendAddResult(ctx, friendAdded)
			m.addFriendResultMessage(friendAdded)
			continue
		}
		if friendDeleted, ok, err := network.ParseFriendDelete(pkt); err != nil {
			log.Printf("parse friend delete 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyFriendDelete(ctx, friendDeleted)
			continue
		}
		if partyCreate, ok, err := network.ParsePartyCreateResult(pkt); err != nil {
			log.Printf("parse party create result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handlePartyCreateResult(ctx, partyCreate)
			continue
		}
		if partyList, ok, err := network.ParsePartyList(pkt); err != nil {
			log.Printf("parse party list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyList(ctx, partyList)
			continue
		}
		if partyInvite, ok, err := network.ParsePartyInviteRequest(pkt); err != nil {
			log.Printf("parse party invite request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openPartyInviteRequest(ctx, partyInvite)
			continue
		}
		if partyInviteAnswer, ok, err := network.ParsePartyInviteAnswer(pkt); err != nil {
			log.Printf("parse party invite answer 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handlePartyInviteAnswer(partyInviteAnswer)
			continue
		}
		if partyOption, ok, err := network.ParsePartyOption(pkt); err != nil {
			log.Printf("parse party option 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyOption(ctx, partyOption)
			continue
		}
		if partyMember, ok, err := network.ParsePartyMemberJoin(pkt); err != nil {
			log.Printf("parse party member join 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberJoin(ctx, partyMember)
			continue
		}
		if partyLeave, ok, err := network.ParsePartyMemberLeave(pkt); err != nil {
			log.Printf("parse party member leave 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if !applyPartyMemberLeave(ctx, partyLeave) {
				m.console.AddErrorMessage("Cannot leave party on this map.")
			}
			continue
		}
		if partyHP, ok, err := network.ParsePartyMemberHP(pkt); err != nil {
			log.Printf("parse party member hp 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberHP(ctx, partyHP)
			continue
		}
		if partyPosition, ok, err := network.ParsePartyMemberPosition(pkt); err != nil {
			log.Printf("parse party member position 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyMemberPosition(ctx, partyPosition)
			continue
		}
		if partyChat, ok, err := network.ParsePartyChat(pkt); err != nil {
			log.Printf("parse party chat 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyPartyChat(ctx, partyChat, &m.console)
			continue
		}
		if tradeRequest, ok, err := network.ParseTradeRequest(pkt); err != nil {
			log.Printf("parse trade request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.openTradeRequest(ctx, tradeRequest)
			continue
		}
		if showEquip, ok, err := network.ParseShowEquipConfig(pkt); err != nil {
			log.Printf("parse show equip config 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if ctx.Session != nil {
				ctx.Session.ShowEquip = showEquip
			}
			continue
		}
		if viewedEquip, ok, err := network.ParseViewedEquipment(pkt); err != nil {
			log.Printf("parse viewed equipment 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.viewEquipWindow.Open(ctx, viewedEquip, m)
			continue
		}
		if tradeResponse, ok, err := network.ParseTradeResponse(pkt); err != nil {
			log.Printf("parse trade response 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleTradeResponse(ctx, tradeResponse)
			continue
		}
		if tradeItem, ok, err := network.ParseTradeItem(pkt); err != nil {
			log.Printf("parse trade item 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.tradeWindow.AddReceivedItem(ctx, tradeItem)
			continue
		}
		if tradeAck, ok, err := network.ParseTradeAddItemAck(pkt); err != nil {
			log.Printf("parse trade add item ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.tradeWindow.AddOwnItemAck(ctx, tradeAck)
			continue
		}
		if tradeConclude, ok, err := network.ParseTradeConclude(pkt); err != nil {
			log.Printf("parse trade conclude 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.tradeWindow.SetConcluded(ctx, tradeConclude.Other)
			continue
		}
		if network.ParseTradeCanceled(pkt) {
			m.tradeWindow.Close(ctx)
			m.console.AddErrorMessage("Trade canceled.")
			continue
		}
		if tradeExec, ok, err := network.ParseTradeExec(pkt); err != nil {
			log.Printf("parse trade exec 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.handleTradeExec(ctx, tradeExec)
			continue
		}
		if network.ParseTradeUndo(pkt) {
			m.tradeWindow.Undo(ctx)
			continue
		}
		if storageItem, ok, err := network.ParseStorageItemAdded(pkt); err != nil {
			log.Printf("parse storage item added 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemAdded(ctx, storageItem)
			m.storageWindow.OpenWindow(ctx)
			m.storageWindow.ClampScroll(ctx.Session)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemAdded(pkt); err != nil {
			log.Printf("parse cart item added 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("cart item added index=%d item=%d amount=%d", cartItem.Index, cartItem.ItemID, cartItem.Amount)
			applyCartItemAdded(ctx, cartItem)
			m.cartWindow.ClampScroll(ctx.Session)
			m.cartWindow.Refresh(ctx, &m.itemInfoWindow)
			m.inventoryBag.ClampScroll(ctx.Session)
			continue
		}
		if ack, ok, err := network.ParseCartAddAck(pkt); err != nil {
			log.Printf("parse cart add ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			switch ack.Result {
			case 0:
				m.console.AddErrorMessage("Cart is overweight.")
				log.Printf("cart add rejected result=%d reason=weight", ack.Result)
			case 1:
				m.console.AddErrorMessage("Cart has too many items.")
				log.Printf("cart add rejected result=%d reason=count", ack.Result)
			default:
				m.console.AddErrorMessage("Cart add failed.")
				log.Printf("cart add rejected result=%d", ack.Result)
			}
			continue
		}
		if storageItem, ok, err := network.ParseStorageItemRemoved(pkt); err != nil {
			log.Printf("parse storage item removed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemRemoved(ctx, storageItem)
			m.storageWindow.ClampScroll(ctx.Session)
			m.storageWindow.Refresh(ctx, &m.itemInfoWindow)
			continue
		}
		if cartItem, ok, err := network.ParseCartItemRemoved(pkt); err != nil {
			log.Printf("parse cart item removed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyCartItemRemoved(ctx, cartItem)
			m.cartWindow.ClampScroll(ctx.Session)
			m.cartWindow.Refresh(ctx, &m.itemInfoWindow)
			continue
		}
		if vendOpen, ok, err := network.ParseVendingOpenRequest(pkt); err != nil {
			log.Printf("parse vending open request 0x%04X: %v", pkt.ID, err)
		} else if ok {
			log.Printf("vending open request max_items=%d", vendOpen.MaxItems)
			m.vendingWindow.OpenSetup(ctx, vendOpen)
			continue
		}
		if board, ok, err := network.ParseVendingBoard(pkt); err != nil {
			log.Printf("parse vending board 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyVendingBoard(ctx, board)
			continue
		}
		if board, ok, err := network.ParseVendingBoardDisappear(pkt); err != nil {
			log.Printf("parse vending board disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyVendingBoardDisappear(ctx, board)
			continue
		}
		if vendList, ok, err := network.ParseVendingItemList(pkt); err != nil {
			log.Printf("parse vending item list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if vendList.Own {
				m.vendingWindow.ApplyOwnList(ctx, vendList)
			} else {
				m.vendingWindow.OpenBuy(ctx, vendList)
			}
			continue
		}
		if vendResult, ok, err := network.ParseVendingPurchaseResult(pkt); err != nil {
			log.Printf("parse vending purchase result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.vendingWindow.ApplyPurchaseResult(ctx, vendResult)
			continue
		}
		if sold, ok, err := network.ParseVendingSoldItem(pkt); err != nil {
			log.Printf("parse vending sold item 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.vendingWindow.ApplySoldItem(ctx, sold)
			continue
		}
		if network.ParseStorageClosed(pkt) {
			applyStorageClosed(ctx)
			m.storageWindow.SetOpen(false)
			continue
		}
		if network.ParseCartClosed(pkt) {
			applyCartClosed(ctx)
			m.cartWindow.SetOpen(false)
			continue
		}
		if deal, ok, err := network.ParseShopDealSelection(pkt); err != nil {
			log.Printf("parse shop deal selection 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.shopWindow.OpenDeal(deal, ctx)
			continue
		}
		if sellList, ok, err := network.ParseShopSellList(pkt); err != nil {
			log.Printf("parse shop sell list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.shopWindow.OpenSell(sellList, ctx)
			continue
		}
		if buyList, ok, err := network.ParseShopBuyList(pkt); err != nil {
			log.Printf("parse shop buy list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.shopWindow.OpenBuy(buyList, ctx)
			continue
		}
		if result, ok, err := network.ParseShopResult(pkt); err != nil {
			log.Printf("parse shop result 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.shopWindow.ApplyResult(ctx, result)
			if result.Sell && result.Result == 0 {
				m.console.AddBlueMessage("The deal has successfully completed.")
			}
			continue
		}
		if vanish, ok, err := network.ParseActorVanish(pkt); err != nil {
			log.Printf("parse actor vanish 0x%04X: %v", pkt.ID, err)
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
			log.Printf("parse actor look change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			if m.applySkillUnitLookChange(ctx, look) {
				continue
			}
			if applyActorLookChange(ctx, look) {
				if view, status := loadPlayerHumanoidSpriteView(ctx.Resources, selectedCharacter(ctx.Session), ctx.Session.Sex); view != nil {
					m.playerView = view
					log.Printf("player sprite changed type=%d value=%d %s", look.Type, look.Value, status)
				} else {
					m.playerView = nil
					log.Printf("player sprite reload failed after look change type=%d value=%d: %s", look.Type, look.Value, status)
				}
			}
			continue
		}
		if direction, ok, err := network.ParseActorDirectionChange(pkt); err != nil {
			log.Printf("parse actor direction change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyActorDirectionChange(ctx, direction)
			continue
		}
		if state, ok, err := network.ParseActorStateChange(pkt); err != nil {
			log.Printf("parse actor state change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorStateChange(ctx, state)
			continue
		}
		if bladeStop, ok, err := network.ParseActorBladeStop(pkt); err != nil {
			log.Printf("parse actor blade stop 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorBladeStop(ctx, bladeStop)
			continue
		}
		if action, ok, err := network.ParseActorActionNotify(pkt); err != nil {
			log.Printf("parse actor action 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorActionNotify(ctx, action)
			continue
		}
		if life, ok, err := network.ParseActorHPUpdate(pkt); err != nil {
			log.Printf("parse actor hp 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyActorHPUpdate(life)
			continue
		}
		if snapshot, ok, err := network.ParseStatusSnapshot(pkt); err != nil {
			log.Printf("parse status snapshot 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStatusSnapshot(ctx, snapshot)
			continue
		}
		if ack, ok, err := network.ParseStatusChangeAck(pkt); err != nil {
			log.Printf("parse status change ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.statsWindow.ApplyStatusChangeAck(ctx, ack)
			continue
		}
		if statusEffect, ok, err := network.ParseStatusEffectChange(pkt); err != nil {
			log.Printf("parse status effect change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyStatusEffectChange(ctx, statusEffect)
			continue
		}
		if list, ok, err := network.ParseSkillInfoList(pkt); err != nil {
			log.Printf("parse skill list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySkillInfoList(ctx, list)
			m.skillWindow.ClampScroll(ctx.Session)
			continue
		}
		if update, ok, err := network.ParseSkillInfoUpdate(pkt); err != nil {
			log.Printf("parse skill update 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applySkillInfoUpdate(ctx, update)
			m.skillWindow.ClampScroll(ctx.Session)
			continue
		}
		if auto, ok, err := network.ParseAutoRunSkill(pkt); err != nil {
			log.Printf("parse auto-run skill 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.skills().ApplyAutoRun(ctx, auto)
			continue
		}
		if warpList, ok, err := network.ParseWarpPointList(pkt); err != nil {
			log.Printf("parse warp point list 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyWarpPointList(ctx, warpList)
			continue
		}
		if memo, ok, err := network.ParseRememberWarpPointAck(pkt); err != nil {
			log.Printf("parse remember warp point ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyRememberWarpPointAck(ctx, memo)
			continue
		}
		if fail, ok, err := network.ParseSkillFailAck(pkt); err != nil {
			log.Printf("parse skill fail ack 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillFailAck(ctx, fail)
			continue
		}
		if cast, ok, err := network.ParseSkillCastNotify(pkt); err != nil {
			log.Printf("parse skill cast 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillCastNotify(ctx, cast)
			continue
		}
		if groundSkill, ok, err := network.ParseGroundSkillNotify(pkt); err != nil {
			log.Printf("parse ground skill 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyGroundSkillNotify(ctx, groundSkill)
			continue
		}
		if skillUnit, ok, err := network.ParseSkillUnitEntry(pkt); err != nil {
			log.Printf("parse skill unit 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillUnitEntry(ctx, skillUnit)
			continue
		}
		if skillUnit, ok, err := network.ParseSkillUnitDisappear(pkt); err != nil {
			log.Printf("parse skill unit disappear 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillUnitDisappear(skillUnit)
			continue
		}
		if effect, ok, err := network.ParseSpecialEffectNotify(pkt); err != nil {
			log.Printf("parse special effect 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySpecialEffectNotify(ctx, effect)
			continue
		}
		if skill, ok, err := network.ParseSkillNoDamageNotify(pkt); err != nil {
			log.Printf("parse skill nodamage 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applySkillNoDamageNotify(ctx, skill)
			continue
		}
		if failure, ok, err := network.ParseAttackFailureForDistance(pkt); err != nil {
			log.Printf("parse attack distance failure 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyAttackFailureForDistance(ctx, failure)
			continue
		}
		if recovery, ok, err := network.ParseRecovery(pkt); err != nil {
			log.Printf("parse recovery 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyRecovery(ctx, recovery)
			continue
		}
		if change, ok, err := network.ParseParameterChange(pkt); err != nil {
			log.Printf("parse parameter change 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.applyParameterChange(ctx, change)
			if change.VarID == network.StatusHP {
				m.clearLocalDeathStateIfAlive(ctx)
			}
			continue
		}
		if entry, ok, err := network.ParseActorEntry(pkt); err != nil {
			log.Printf("parse actor entry 0x%04X: %v", pkt.ID, err)
		} else if ok {
			m.clearActorDeath(entry.ID)
			m.upsertNetworkActor(ctx, entry)
			m.applyWarpPortalEntry(ctx, entry)
		}
	}
	for _, err := range ctx.Network.DrainErrors() {
		log.Printf("network frame error: %v", err)
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
	if m.openEscapeMenuFromInput(ctx) {
		return nil, nil
	}
	m.skills().AdjustPendingLevelFromWheel(ctx)
	playerContextConsumed := m.playerContext.Update(ctx)
	switch action := m.playerContext.PopAction(); action.Kind {
	case gameui.PlayerContextActionAddFriend:
		m.sendAddFriend(ctx, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteParty:
		m.sendPartyInvite(ctx, action.ActorID, action.Name)
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
	if m.openPlayerContextFromInput(ctx, now) {
		return nil, nil
	}
	if !m.escapeMenu.IsOpen() && !m.teleportModal.IsOpen() && !m.deathModal.IsOpen() && !m.friendRequest.IsOpen() && !m.partyRequest.IsOpen() && !m.tradeRequest.IsOpen() && !m.settingsWindow.IsOpen() && !m.identifyWindow.IsOpen() {
		m.updateCameraRotation(ctx)
	}
	if m.escapeMenu.IsOpen() {
		if m.escapeMenu.Update(ctx) {
			m.handleEscapeMenuAction(ctx)
			return nil, nil
		}
	}
	if m.friendRequest.Update(ctx) {
		return nil, nil
	}
	if m.partyRequest.Update(ctx) {
		return nil, nil
	}
	if m.tradeRequest.Update(ctx) {
		return nil, nil
	}
	if m.deathModal.Update(ctx) {
		return nil, nil
	}
	if m.teleportModal.Update(ctx, m) {
		return nil, nil
	}
	if m.npcDialog.Update(ctx) {
		return nil, nil
	}
	if m.console.Update(ctx) {
		return nil, nil
	}
	if m.settingsWindow.Update(ctx) {
		return nil, nil
	}
	if m.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return nil, nil
	}
	if m.characterWindow.Update(ctx) {
		return nil, nil
	}
	if m.itemInfoWindow.Update(ctx, m) {
		return nil, nil
	}
	if m.identifyWindow.Update(ctx) {
		return nil, nil
	}
	if m.inventoryBag.UpdateDrag(ctx, &m.shortcutBar, &m.storageWindow, &m.cartWindow, &m.tradeWindow) {
		return nil, nil
	}
	if m.storageWindow.UpdateDrag(ctx, &m.inventoryBag, &m.cartWindow) {
		return nil, nil
	}
	if m.cartWindow.UpdateDrag(ctx, &m.inventoryBag, &m.storageWindow) {
		return nil, nil
	}
	if m.skillWindow.UpdateDrag(ctx, &m.shortcutBar) {
		return nil, nil
	}
	if m.shortcutBar.Update(ctx, m) {
		return nil, nil
	}
	if m.inventoryBag.Update(ctx, &m.shortcutBar, &m.storageWindow, &m.cartWindow, &m.tradeWindow, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.tradeWindow.Update(ctx, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.equipmentWindow.Update(ctx, &m.itemInfoWindow, &m.cartWindow, m) {
		return nil, nil
	}
	if m.viewEquipWindow.Update(ctx, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.storageWindow.Update(ctx, &m.inventoryBag, &m.cartWindow, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.cartWindow.Update(ctx, &m.inventoryBag, &m.storageWindow, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.changeCartWindow.Update(ctx) {
		return nil, nil
	}
	if m.shopWindow.Update(ctx, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.vendingWindow.Update(ctx, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.skillWindow.Update(ctx, &m.shortcutBar, m) {
		return nil, nil
	}
	if m.friendsWindow.Update(ctx) {
		switch m.friendsWindow.PopAction() {
		case gameui.FriendsWindowActionPartySettings:
			m.partySettings.Open(ctx)
		case gameui.FriendsWindowActionPartyLeave:
			if ctx.Network == nil {
				m.console.AddErrorMessage("Leave party failed: not connected.")
			} else if err := ctx.Network.SendLeaveParty(); err != nil {
				m.console.AddErrorMessage("Leave party failed.")
				log.Printf("leave party failed: %v", err)
			}
		}
		return nil, nil
	}
	if m.partySettings.Update(ctx) {
		return nil, nil
	}
	if m.statsWindow.Update(ctx) {
		return nil, nil
	}
	if m.basicMenu.Update(ctx, m.basicMenuCallbacks(ctx)) {
		return nil, nil
	}
	m.minimap.Update(ctx)
	removeExpiredStatusEffects(ctx.Session, now)
	m.statusIcons.Update(ctx, now)
	pointerBlocked := uiPointerBlocked(ctx)
	if !pointerBlocked {
		m.updateCameraZoom(ctx)
	}

	if !pointerBlocked && ctx.Input.MouseJustPressed(render.MouseButtonLeft) && m.walkReady(now) {
		m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		if m.pendingSkill.skill.ID != 0 {
			m.skills().HandleClick(ctx, projection, now)
			return nil, nil
		}
		playerX, playerY := currentPlayerCell(ctx, now)
		if actor, ok := m.hoveredVendingBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			log.Printf("click vending target mouse=%d,%d id=%d name=%q shop=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.VendingName, playerX, playerY, actor.X, actor.Y)
			m.requestVendingList(ctx, actor, "click")
			return nil, nil
		}
		if item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			log.Printf("click pickup target mouse=%d,%d id=%d item_id=%d amount=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, item.ID, item.ItemID, item.Amount, playerX, playerY, item.X, item.Y)
			m.clearLockedAttack()
			m.clearAttackFocus()
			m.requestPickup(ctx, item, "click")
			return nil, nil
		}
		if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			log.Printf("click attack target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.requestAttack(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			log.Printf("click npc talk target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.clearAttackFocus()
			m.requestNPCTalk(ctx, actor, "click")
			return nil, nil
		}
		if targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok {
			log.Printf("click walk target mouse=%d,%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, playerX, playerY, targetX, targetY)
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
	switch m.escapeMenu.ConsumeAction() {
	case gameui.EscapeMenuActionCharacterSelect:
		m.escapeMenu.RequestCharacterSelect(ctx)
	case gameui.EscapeMenuActionSettings:
		m.settingsWindow.OpenWindow(ctx)
	}
}

func (m *WorldMode) openEscapeMenuFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.escapeMenu.IsOpen() || !ctx.Input.JustPressed(render.KeyEscape) {
		return false
	}
	if m.deathModal.IsOpen() || m.teleportModal.IsOpen() || m.friendRequest.IsOpen() || m.partyRequest.IsOpen() || m.tradeRequest.IsOpen() {
		return false
	}
	m.escapeMenu.Toggle(ctx)
	return true
}

func (m *WorldMode) basicMenuCallbacks(ctx client.Context) gameui.BasicMenuCallbacks {
	return gameui.BasicMenuCallbacks{
		OnStatus: func() { m.statsWindow.Toggle(ctx) },
		OnOption: func() { m.escapeMenu.Toggle(ctx) },
		OnItems:  func() { m.inventoryBag.Toggle(ctx) },
		OnEquip:  func() { m.equipmentWindow.Toggle(ctx) },
		OnSkill:  func() { m.skillWindow.Toggle(ctx) },
		OnMap:    func() { m.minimap.Toggle(ctx) },
		OnFriend: func() { m.friendsWindow.Toggle(ctx) },
	}
}

func (m *WorldMode) handleMapChange(ctx client.Context, change network.MapChange) Mode {
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.clearLocalActorAction(ctx)
	m.scheduledStops = nil
	m.npcDialog.ResetPublished(ctx)
	m.teleportModal = gameui.TeleportModal{}
	m.clearLocalDeathState(ctx)
	currentMap := ctx.World.MapName
	reuseLoadedMap := !change.ServerMove && sameLoadedMap(ctx, change.MapName)
	log.Printf("map change current=%s target=%s x=%d y=%d server_move=%t addr=%s port=%d reuse_loaded=%t", currentMap, change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port, reuseLoadedMap)
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	applyWarpPosition(ctx, change.X, change.Y)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	if reuseLoadedMap {
		zoom := m.camera.zoom
		m.camera.Reset()
		m.camera.zoom = zoom
		m.camera.Update(ctx, time.Now())
		if ctx.Network != nil {
			if err := ctx.Network.SendLoadEndAck(); err != nil {
				log.Printf("same-map warp load ack failed map=%s x=%d y=%d: %v", change.MapName, change.X, change.Y, err)
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
			log.Printf("map reconnect failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			return nil
		}
		if err := ctx.Network.SendMapServerEnter(ctx.Session.AccountID, ctx.Session.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex); err != nil {
			log.Printf("map re-enter failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
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
	next.console = m.console
	next.characterWindow = m.characterWindow
	next.basicMenu = m.basicMenu
	next.inventoryBag = m.inventoryBag
	next.equipmentWindow = m.equipmentWindow
	next.cartWindow = m.cartWindow
	next.itemInfoWindow = m.itemInfoWindow
	next.statsWindow = m.statsWindow
	next.skillWindow = m.skillWindow
	next.friendsWindow = m.friendsWindow
	next.settingsWindow = m.settingsWindow
	next.shortcutBar = m.shortcutBar
	next.minimap = m.minimap
	next.startMapFadeIn(time.Now())
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
	next := NewCharacterSelectMode(ctx, m.console)
	return next
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
		log.Printf("%s npc talk failed target=%d player=%d,%d target=%d,%d: %v", source, actor.ID, playerX, playerY, actor.X, actor.Y, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) humanoidSpriteViewForActor(ctx client.Context, actor worldstate.Actor) *humanoidSpriteView {
	if isLocalActor(ctx, actor.ID) {
		return m.playerView
	}
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := actorSpriteKey{
		job:         int(actor.Job),
		head:        int(actor.Head),
		sex:         actor.Sex,
		bodyPalette: int(actor.BodyPal),
		headPalette: int(actor.HeadPal),
		weapon:      weapon,
		shield:      shield,
		headTop:     int(actor.HeadTop),
		headMid:     int(actor.HeadMid),
		headLow:     int(actor.HeadLow),
	}
	return m.actorViews[key]
}

func (m *WorldMode) Draw(ctx client.Context, screen *render.Image) {
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
			m.drawTileCursor(screen, ctx, projection, now)
		}
		if ctx.World.RSW != nil && len(ctx.World.RSM) > 0 {
			actorOverlays = m.drawSceneModelsAndActors(screen, ctx, projection, vertexFog, now)
		} else {
			m.drawGroundItems(screen, ctx, projection, now)
			actorOverlays = m.drawSceneActors(screen, ctx, projection)
		}
	} else if ctx.World.GAT != nil {
		drawGAT(screen, ctx.World.GAT, ctx.World.Player.X, ctx.World.Player.Y)
		m.drawGroundItems(screen, ctx, projection, now)
		actorOverlays = m.drawSceneActors(screen, ctx, projection)
	} else {
		const tile = 32
		for x := 0; x < width; x += tile {
			render.DrawLine(screen, float64(x), 0, float64(x), float64(height), render.ColorGrid)
		}
		for y := 0; y < height; y += tile {
			render.DrawLine(screen, 0, float64(y), float64(width), float64(y), render.ColorGrid)
		}
	}

	if !ctx.Config.Render.NoUI {
		m.drawSceneActorOverlays(screen, ctx, projection, now, actorOverlays)
	}
	m.drawRSWEffects(screen, ctx, projection, now)
	m.drawMapWeatherEffects(screen, ctx, projection, now)
	m.drawWorldEffects(screen, ctx, projection, now)
	m.drawDamageFloaters(screen, ctx, projection, now)

	if !ctx.Config.Render.NoUI {
		m.inventoryBag.Draw(screen, ctx, m)
		m.storageWindow.Draw(screen, ctx, m)
		m.cartWindow.Draw(screen, ctx, m)
		m.shopWindow.Draw(screen, ctx, m)
		m.vendingWindow.Draw(screen, ctx, m)
		m.skillWindow.Draw(screen, ctx, m)
		m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
		m.deathModal.Draw(screen, ctx, width, height)
	}
}

func (m *WorldMode) DrawOverlay(ctx client.Context, screen *render.Image) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	m.drawMapFade(screen, now)
	if !ctx.Config.Render.NoUI {
		m.drawUIDragGhosts(screen, ctx)
		m.drawROCursor(screen, ctx, projection, now)
	}
}

func (m *WorldMode) DrawUIOverlay(ctx client.Context, screen *render.Image) {
	if ctx.Config.Render.NoUI {
		return
	}
	m.inventoryBag.DrawTooltip(screen)
	m.equipmentWindow.DrawTooltip(screen)
	m.shortcutBar.DrawTooltip(screen)
}

func (m *WorldMode) drawUIDragGhosts(screen *render.Image, ctx client.Context) {
	m.inventoryBag.DrawDragGhost(screen, ctx, m)
	m.storageWindow.DrawDragGhost(screen, ctx, m)
	m.cartWindow.DrawDragGhost(screen, ctx, m)
	m.shopWindow.DrawDragGhost(screen, ctx, m)
	m.vendingWindow.DrawDragGhost(screen, ctx, m)
	m.skillWindow.DrawDragGhost(screen, ctx, m)
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
		actorX, actorY := actor.RenderPosition(now)
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
		actorX, actorY := actor.RenderPosition(now)
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
		actorX, actorY := actor.RenderPosition(now)
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

func (m *WorldMode) drawSceneModelsAndActors(screen *render.Image, ctx client.Context, projection sceneProjection, fog sceneFog, now time.Time) []sceneActorDrawEntry {
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
			m.drawActorShadowEntry(screen, projection, actors[entry.shadowIndex])
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

func drawColoredSurfaceTints3DAlpha(screen, white *render.Image, verts [4]modelPoint3, indices []uint16, colors [4]color.RGBA) {
	drawColoredSurfaceTints3DWithOptions(screen, white, verts, indices, colors, triangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}

func drawColoredSurfaceTints3DWithOptions(screen, white *render.Image, verts [4]modelPoint3, indices []uint16, colors [4]color.RGBA, options *render.DrawTrianglesOptions) {
	vertices := []render.Vertex3D{
		coloredSurfaceVertex3D(verts[0], 0, 0, colors[0]),
		coloredSurfaceVertex3D(verts[1], 1, 0, colors[1]),
		coloredSurfaceVertex3D(verts[2], 1, 1, colors[2]),
		coloredSurfaceVertex3D(verts[3], 0, 1, colors[3]),
	}
	screen.DrawTriangles3DOwned(vertices, indices, white, options)
}

func drawTexturedSurface3DAlpha(screen, texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA) {
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, indices, tints, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
}

func drawTexturedSurface3DWithOptions(screen, texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
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

func drawGAT(screen *render.Image, gat *res.GAT, playerX, playerY int) {
	const tile = 10
	width := screen.Bounds().Dx()
	height := screen.Bounds().Dy()
	tilesX := width/tile + 2
	tilesY := height/tile + 2
	startX := playerX - tilesX/2
	startY := playerY - tilesY/2

	for sy := 0; sy < tilesY; sy++ {
		mapY := startY + sy
		for sx := 0; sx < tilesX; sx++ {
			mapX := startX + sx
			cell, ok := gat.Cell(mapX, mapY)
			c := color.RGBA{R: 22, G: 25, B: 32, A: 255}
			if ok {
				switch {
				case cell.Type&res.GATTypeWater != 0:
					c = color.RGBA{R: 38, G: 84, B: 112, A: 255}
				case cell.Type&res.GATTypeWalkable != 0:
					c = color.RGBA{R: 54, G: 75, B: 54, A: 255}
				case cell.Type&res.GATTypeSnipable != 0:
					c = color.RGBA{R: 87, G: 77, B: 42, A: 255}
				default:
					c = color.RGBA{R: 54, G: 45, B: 48, A: 255}
				}
			}
			render.DrawRect(screen, float64(sx*tile), float64(sy*tile), tile-1, tile-1, c)
		}
	}
}
