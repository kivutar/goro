package game

import (
	"context"
	"fmt"
	"github.com/kivutar/goro/client"
	"hash/fnv"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type WorldMode struct {
	status           string
	walkCooldown     int
	tickCooldown     int
	camera           followCamera
	cameraShakeStart time.Time
	cameraShakeEnd   time.Time
	whitePixel       *render.Image
	tileCursor       *render.Image
	textures         map[string]*render.Image
	textureMiss      map[string]struct{}
	strEffects       map[string]*res.STR
	strEffectMiss    map[string]struct{}
	playerView       *humanoidSpriteView
	shadowView       *playerSpriteView
	shadowViewMiss   bool
	cursorView       *playerSpriteView
	cursorViewMiss   bool
	cursorFallback   *render.Image
	cursorAction     int
	cursorStarted    time.Time
	damageNumberView *playerSpriteView
	damageNumberMiss bool
	damageNumbers    map[string]*spriteBillboard
	damageMsgView    *playerSpriteView
	damageMsgMiss    bool
	itemMarker       *render.Image
	itemViews        map[itemSpriteKey]*playerSpriteView
	itemViewMiss     map[itemSpriteKey]struct{}
	effectViews      map[string]*playerSpriteView
	effectViewMiss   map[string]struct{}
	actorViews       map[actorSpriteKey]*humanoidSpriteView
	actorViewMiss    map[actorSpriteKey]struct{}
	nonPCViews       map[int]*playerSpriteView
	nonPCViewMiss    map[int]struct{}
	rsmWorldCache    map[int][]modelWorldTriangle
	rsmMeshCache     map[int][]retainedWorldMesh
	rsmNodeMatrices  map[*res.RSM]map[string]mat4
	rsmBoundsCache   map[rsmBoundsCacheKey]rsmBounds
	gndMeshCache     *gndRetainedMeshCache
	pendingWarp      bool
	pendingAttack    attackIntent
	pendingPickup    pickupIntent
	pendingSkill     pendingSkillTarget
	pickupReqItemID  uint32
	lockedAttackID   uint32
	lastAttackAt     time.Time
	lastChaseAt      time.Time
	actorAnims       map[uint32]actorAnimation
	damageFloaters   []damageFloater
	worldEffects     []worldEffect
	scheduledSounds  []scheduledSound
	actorDeaths      map[uint32]time.Time
	actorSoundFrames map[uint32]actorSoundFrame
	actorLife        map[uint32]actorLife
	actorNameReqAt   map[uint32]time.Time
	gndNormalSource  *res.GND
	gndTopNormals    [][4]modelPoint3
	minimap          gameui.Minimap
	statusIcons      gameui.StatusIcons
	console          gameui.ChatConsole
	npcDialog        gameui.NPCDialog
	escapeMenu       gameui.EscapeMenu
	teleportModal    gameui.TeleportModal
	deathModal       gameui.DeathModal
	characterWindow  gameui.CharacterWindow
	basicMenu        gameui.BasicMenu
	inventoryBag     gameui.InventoryBagWindow
	equipmentWindow  gameui.EquipmentWindow
	storageWindow    gameui.StorageWindow
	shopWindow       gameui.ShopWindow
	itemInfoWindow   gameui.ItemInfoWindow
	identifyWindow   gameui.IdentifyWindow
	statsWindow      gameui.StatsWindow
	skillWindow      gameui.SkillWindow
	settingsWindow   gameui.SettingsWindow
	shortcutBar      gameui.ShortcutBar
	mapFade          mapFadeState
}

type actorSpriteKey struct {
	job     int
	head    int
	sex     byte
	weapon  int
	shield  int
	headTop int
	headMid int
	headLow int
}

type attackIntent struct {
	targetID uint32
	expires  time.Time
	readyAt  time.Time
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

type damageFloater struct {
	actorID uint32
	x       int
	y       int
	text    string
	color   color.RGBA
	kind    damageFloaterKind
	starts  time.Time
	expires time.Time
}

type damageFloaterKind int

const (
	damageFloaterNormal damageFloaterKind = iota
	damageFloaterCritical
	damageFloaterIncoming
	damageFloaterRecoveryHP
	damageFloaterRecoverySP
	damageFloaterMiss
)

type scheduledSound struct {
	at    time.Time
	paths []string
}

type actorSoundFrame struct {
	actionFamily int
	motion       int
	soundIndex   int
}

type actorAnimation struct {
	actionFamily   int
	started        time.Time
	duration       time.Duration
	holdFinal      bool
	fixedMotion    int
	hasFixedMotion bool
}

type actorLife struct {
	hp        int
	maxHP     int
	sp        int
	maxSP     int
	hasSP     bool
	player    bool
	estimated bool
	fromTiny  bool
	updatedAt time.Time
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

const attackRetryInterval = 1200 * time.Millisecond

const (
	defaultAttackAnimationDuration = 600 * time.Millisecond
	defaultHitAnimationDuration    = 250 * time.Millisecond
	defaultDeathAnimationDuration  = 900 * time.Millisecond
	maxCombatAnimationDuration     = 5 * time.Second
	nonPCDeathFadeDuration         = 5 * time.Second
	mapFadeOutDuration             = 220 * time.Millisecond
	mapFadeInDuration              = 340 * time.Millisecond
	actorNameRequestCooldown       = time.Second
	defaultRSMLoadLimit            = 128
	defaultCameraFollowFactor      = 0.1
	defaultCameraWheelZoomStep     = 1.12
	defaultCameraWheelZoomUnits    = 15
	defaultCameraPinchZoomScale    = 240
	defaultCameraMinZoom           = 65.0
	defaultCameraMaxZoom           = 165.0
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
	m.status = "loading map"
	now := time.Now()
	if m.mapFade.phase == mapFadeNone {
		m.startMapFadeIn(now)
	} else if m.mapFade.started.IsZero() {
		m.mapFade.started = now
	}
	m.camera.Reset()
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
	m.itemViews = make(map[itemSpriteKey]*playerSpriteView)
	m.itemViewMiss = make(map[itemSpriteKey]struct{})
	m.effectViews = make(map[string]*playerSpriteView)
	m.effectViewMiss = make(map[string]struct{})
	m.actorViews = make(map[actorSpriteKey]*humanoidSpriteView)
	m.actorViewMiss = make(map[actorSpriteKey]struct{})
	m.nonPCViews = make(map[int]*playerSpriteView)
	m.nonPCViewMiss = make(map[int]struct{})
	m.rsmWorldCache = make(map[int][]modelWorldTriangle)
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	m.rsmNodeMatrices = make(map[*res.RSM]map[string]mat4)
	m.rsmBoundsCache = make(map[rsmBoundsCacheKey]rsmBounds)
	m.gndMeshCache = nil
	m.pendingWarp = false
	m.pendingAttack = attackIntent{}
	m.pendingPickup = pickupIntent{}
	m.pendingSkill = pendingSkillTarget{}
	m.pickupReqItemID = 0
	m.lockedAttackID = 0
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
	m.actorAnims = make(map[uint32]actorAnimation)
	m.damageFloaters = nil
	m.scheduledSounds = nil
	m.actorDeaths = make(map[uint32]time.Time)
	m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	m.actorLife = make(map[uint32]actorLife)
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
	if ctx.World.MapName == "" {
		m.status = "no map selected"
		return
	}

	gat, source, err := loadGAT(ctx.Resources, ctx.World.MapName)
	if err != nil {
		m.status = err.Error()
		return
	}
	ctx.World.GAT = gat
	m.status = fmt.Sprintf("loaded %s %dx%d", source, gat.Width, gat.Height)
	if gnd, gndSource, err := loadGND(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.GND = gnd
		m.status = fmt.Sprintf("loaded %s %dx%d", gndSource, gnd.Width, gnd.Height)
	} else {
		ctx.World.GND = nil
		m.status += " gnd: " + err.Error()
	}
	if rsw, rswSource, err := loadRSW(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.RSW = rsw
		ctx.World.RSM, ctx.World.RSMFail = loadRSMModels(ctx.Resources, rsw, defaultRSMLoadLimit)
		m.status += fmt.Sprintf(" rsw=%s", rswSource)
		m.playMapBGM(ctx, rswSource)
	} else {
		ctx.World.RSW = nil
		ctx.World.RSM = nil
		ctx.World.RSMFail = 0
		m.status += " rsw: " + err.Error()
		m.playMapBGM(ctx, ctx.World.MapName)
	}
	if err := ctx.Network.SendLoadEndAck(); err != nil {
		m.status += " load-ack failed: " + err.Error()
	} else {
		m.tickCooldown = 1
	}
	if playerStatus != "" {
		m.status += " " + playerStatus
	}
}

func (m *WorldMode) playMapBGM(ctx client.Context, rswName string) {
	if ctx.Audio == nil {
		return
	}
	path, err := ctx.Audio.PlayMap(rswName)
	if err != nil {
		m.status += " bgm: " + err.Error()
		log.Printf("bgm failed map=%s: %v", rswName, err)
		return
	}
	if path != "" {
		m.status += " bgm=" + path
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
			addConsoleMessage(&m.console, ctx.Resources, chat)
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
			m.status = fmt.Sprintf("entered map %s at %d,%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
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
			m.status = fmt.Sprintf("walk ack: %d,%d -> %d,%d", ack.FromX, ack.FromY, ack.ToX, ack.ToY)
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
				m.status = fmt.Sprintf("position fix: %d,%d", position.X, position.Y)
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
		if storageAmount, ok, err := network.ParseStorageAmount(pkt); err != nil {
			log.Printf("parse storage amount 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageAmount(ctx, storageAmount)
			m.storageWindow.OpenWindow(ctx)
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
		if storageItem, ok, err := network.ParseStorageItemRemoved(pkt); err != nil {
			log.Printf("parse storage item removed 0x%04X: %v", pkt.ID, err)
		} else if ok {
			applyStorageItemRemoved(ctx, storageItem)
			m.storageWindow.ClampScroll(ctx.Session)
			continue
		}
		if network.ParseStorageClosed(pkt) {
			applyStorageClosed(ctx)
			m.storageWindow.SetOpen(false)
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
			upsertNetworkActor(ctx, entry)
			m.applyWarpPortalEntry(ctx, entry)
		}
	}
	for _, err := range ctx.Network.DrainErrors() {
		log.Printf("network frame error: %v", err)
	}

	m.processPendingAttack(ctx)
	m.processPendingPickup(ctx)
	m.skills().ProcessPendingTarget(ctx)
	m.processLockedAttack(ctx)
	now = time.Now()
	m.cleanupDeadActors(ctx, now)
	m.processActorMotionSounds(ctx, now)
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
	m.skills().AdjustPendingLevelFromWheel(ctx)
	if !m.escapeMenu.IsOpen() && !m.teleportModal.IsOpen() && !m.deathModal.IsOpen() && !m.settingsWindow.IsOpen() && !m.identifyWindow.IsOpen() {
		m.updateCameraRotation(ctx)
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
		switch m.escapeMenu.ConsumeAction() {
		case gameui.EscapeMenuActionCharacterSelect:
			m.escapeMenu.RequestCharacterSelect(ctx)
		case gameui.EscapeMenuActionSettings:
			m.settingsWindow.OpenWindow(ctx)
		}
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
	if m.shortcutBar.Update(ctx, m) {
		return nil, nil
	}
	if m.inventoryBag.Update(ctx, &m.shortcutBar, &m.storageWindow, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.equipmentWindow.Update(ctx, &m.itemInfoWindow, m) {
		return nil, nil
	}
	if m.storageWindow.Update(ctx, &m.inventoryBag, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.shopWindow.Update(ctx, &m.itemInfoWindow) {
		return nil, nil
	}
	if m.skillWindow.Update(ctx, &m.shortcutBar, m) {
		return nil, nil
	}
	if m.statsWindow.Update(ctx) {
		return nil, nil
	}
	if m.basicMenu.Update(ctx) {
		if action := m.basicMenu.PopAction(); action != "" {
			m.handleBasicMenuAction(ctx, action)
		}
		return nil, nil
	}
	m.minimap.Update(ctx)
	removeExpiredStatusEffects(ctx.Session, now)
	m.statusIcons.Update(ctx, now)
	if action := m.basicMenu.PopAction(); action != "" {
		m.handleBasicMenuAction(ctx, action)
		return nil, nil
	}
	m.updateCameraZoom(ctx)

	dx, dy := 0, 0
	if ctx.Input.Pressed(render.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.Pressed(render.KeyArrowRight) {
		dx++
	}
	if ctx.Input.Pressed(render.KeyArrowUp) {
		dy--
	}
	if ctx.Input.Pressed(render.KeyArrowDown) {
		dy++
	}
	if m.walkCooldown > 0 {
		m.walkCooldown--
	}
	if ctx.Input.MouseJustPressed(render.MouseButtonLeft) && m.walkCooldown == 0 {
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		if m.pendingSkill.skill.ID != 0 {
			m.skills().HandleClick(ctx, projection, now)
			return nil, nil
		}
		if item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			log.Printf("click pickup target mouse=%d,%d id=%d item_id=%d amount=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, item.ID, item.ItemID, item.Amount, ctx.World.Player.X, ctx.World.Player.Y, item.X, item.Y)
			m.clearLockedAttack()
			m.requestPickup(ctx, item, "click")
			return nil, nil
		}
		if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			log.Printf("click attack target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y)
			m.requestAttack(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			log.Printf("click npc talk target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y)
			m.requestNPCTalk(ctx, actor, "click")
			return nil, nil
		}
		if targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok {
			log.Printf("click walk target mouse=%d,%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, ctx.World.Player.X, ctx.World.Player.Y, targetX, targetY)
			m.clearLockedAttack()
			if shouldUseTurnOnlyGroundClick(ctx) {
				m.requestChangeDirection(ctx, targetX, targetY, "click")
				return nil, nil
			}
			m.requestWalk(ctx, targetX, targetY, "click")
		}
	}
	if (dx != 0 || dy != 0) && m.walkCooldown == 0 {
		targetX := ctx.World.Player.X + dx
		targetY := ctx.World.Player.Y + dy
		m.clearLockedAttack()
		m.requestWalk(ctx, targetX, targetY, "key")
	}
	return nil, nil
}

func (m *WorldMode) handleBasicMenuAction(ctx client.Context, action string) {
	switch action {
	case "status":
		m.statsWindow.Toggle(ctx)
	case "option":
		m.escapeMenu.OpenMenu()
	case "skill":
		m.skillWindow.Toggle(ctx)
	case "items":
		m.inventoryBag.Toggle(ctx)
	case "equip":
		m.equipmentWindow.Toggle(ctx)
	}
}

func (m *WorldMode) handleMapChange(ctx client.Context, change network.MapChange) Mode {
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
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
				m.status = "same-map warp load-ack failed: " + err.Error()
				log.Printf("same-map warp load ack failed map=%s x=%d y=%d: %v", change.MapName, change.X, change.Y, err)
			} else {
				m.tickCooldown = 1
				m.status = fmt.Sprintf("warped on %s at %d,%d", change.MapName, change.X, change.Y)
			}
		} else {
			m.status = fmt.Sprintf("warped on %s at %d,%d", change.MapName, change.X, change.Y)
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
			m.status = "map reconnect failed: " + err.Error()
			log.Printf("map reconnect failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			return nil
		}
		if err := ctx.Network.SendMapServerEnter(ctx.Session.AccountID, ctx.Session.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex); err != nil {
			m.status = "map re-enter failed: " + err.Error()
			log.Printf("map re-enter failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			return nil
		}
		m.pendingWarp = true
		m.status = fmt.Sprintf("waiting for map enter: %s %s:%d", change.MapName, change.Address, change.Port)
		return nil
	}
	return m.nextWorldMode()
}

func (m *WorldMode) nextWorldMode() *WorldMode {
	next := NewWorldMode()
	next.camera.zoom = m.camera.zoom
	next.console = m.console
	next.characterWindow = m.characterWindow
	next.shortcutBar = m.shortcutBar
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
	}
	next := NewCharacterSelectMode(ctx, m.console)
	return next
}

func (m *WorldMode) startMapFadeOut(change network.MapChange, now time.Time) {
	m.mapFade = mapFadeState{
		phase:     mapFadeOut,
		started:   now,
		change:    change,
		hasChange: true,
	}
	if change.MapName != "" {
		m.status = fmt.Sprintf("leaving for %s", change.MapName)
	} else {
		m.status = "leaving map"
	}
}

func (m *WorldMode) startMapFadeIn(now time.Time) {
	m.mapFade = mapFadeState{phase: mapFadeIn, started: now}
}

func (m *WorldMode) mapFadeElapsed(now time.Time) bool {
	switch m.mapFade.phase {
	case mapFadeOut:
		return now.Sub(m.mapFade.started) >= mapFadeOutDuration
	case mapFadeIn:
		return now.Sub(m.mapFade.started) >= mapFadeInDuration
	default:
		return false
	}
}

func (m *WorldMode) mapFadeAlpha(now time.Time) uint8 {
	if m.mapFade.started.IsZero() {
		return 0
	}
	switch m.mapFade.phase {
	case mapFadeOut:
		return clampColor(255 * clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeOutDuration)))
	case mapFadeHold:
		return 255
	case mapFadeIn:
		return clampColor(255 * (1 - clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeInDuration))))
	default:
		return 0
	}
}

func (m *WorldMode) drawMapFade(screen *render.Image, now time.Time) {
	alpha := m.mapFadeAlpha(now)
	if alpha == 0 {
		return
	}
	bounds := screen.Bounds()
	render.DrawRect(screen, 0, 0, float64(bounds.Dx()), float64(bounds.Dy()), color.RGBA{A: alpha})
}

func formatConsoleMessage(manager *res.Manager, chat network.ChatMessage) string {
	if chat.Text != "" {
		return chat.Text
	}
	if chat.MessageID < 0 {
		return ""
	}
	text := ""
	if manager != nil {
		text, _ = manager.MsgString(chat.MessageID)
	}
	if text == "" {
		text = fmt.Sprintf("message #%d", chat.MessageID)
	}
	if chat.Value != 0 {
		if strings.Contains(text, "%") {
			text = fmt.Sprintf(text, chat.Value)
		} else {
			text = fmt.Sprintf("%s %d", text, chat.Value)
		}
	}
	if chat.SkillID != 0 {
		text = fmt.Sprintf("skill %d: %s", chat.SkillID, text)
	}
	return text
}

func addConsoleMessage(console *gameui.ChatConsole, manager *res.Manager, chat network.ChatMessage) {
	if console == nil {
		return
	}
	text := formatConsoleMessage(manager, chat)
	if text == "" {
		return
	}
	if chat.Text == "" || !strings.Contains(text, " : ") {
		console.AddSystemMessage("%s", text)
		return
	}
	console.AddMessage("%s", text)
}

func formatPickupConsoleMessage(manager *res.Manager, pickup network.ItemPickupAck) string {
	itemName := fmt.Sprintf("item %d", pickup.ItemID)
	if manager != nil {
		if name, ok := manager.ItemDisplayName(int(pickup.ItemID), pickup.Identified); ok && name != "" {
			itemName = name
		}
	}
	amount := int(pickup.Amount)
	if amount <= 0 {
		amount = 1
	}
	template := ""
	if manager != nil {
		template, _ = manager.MsgString(153)
	}
	if template == "" {
		template = "You got %s %d."
	}
	if strings.Contains(template, "%s") {
		template = strings.Replace(template, "%s", itemName, 1)
	} else {
		template = strings.TrimSpace(template + " " + itemName)
	}
	if strings.Contains(template, "%d") {
		template = strings.Replace(template, "%d", fmt.Sprintf("%d", amount), 1)
	} else if amount != 1 {
		template = strings.TrimSpace(fmt.Sprintf("%s %d", template, amount))
	}
	return template
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

func (m *WorldMode) requestWalk(ctx client.Context, targetX, targetY int, source string) {
	if !walkTargetInBounds(ctx, targetX, targetY) {
		m.status = fmt.Sprintf("%s walk blocked by map bounds: %d,%d", source, targetX, targetY)
		m.walkCooldown = 12
		return
	}
	log.Printf("%s walk request from=%d,%d to=%d,%d", source, ctx.World.Player.X, ctx.World.Player.Y, targetX, targetY)
	if err := ctx.Network.SendWalkToXY(targetX, targetY); err == nil {
		m.status = fmt.Sprintf("%s walk request: %d,%d", source, targetX, targetY)
		m.walkCooldown = 12
	} else {
		m.status = source + " walk request failed: " + err.Error()
		log.Printf("%s walk request failed from=%d,%d to=%d,%d: %v", source, ctx.World.Player.X, ctx.World.Player.Y, targetX, targetY, err)
		m.walkCooldown = 30
	}
}

func shouldUseTurnOnlyGroundClick(ctx client.Context) bool {
	if ctx.World == nil || ctx.Input == nil {
		return false
	}
	return ctx.World.Player.Sitting || ctx.Input.Pressed(render.KeyShift)
}

func (m *WorldMode) requestChangeDirection(ctx client.Context, targetX, targetY int, source string) {
	if ctx.Network == nil {
		m.status = "change direction failed: not connected"
		m.walkCooldown = 30
		return
	}
	if ctx.World == nil {
		return
	}
	targetDir := directionFromDelta(ctx.World.Player.X, ctx.World.Player.Y, targetX, targetY, ctx.World.Player.Dir)
	headDir, bodyDir, ok := resolveTurnOnlyDirection(ctx.World.Player.Dir, int(ctx.World.Player.HeadDir), targetDir)
	if !ok {
		return
	}
	log.Printf("%s change direction request player=%d,%d target=%d,%d head_dir=%d dir=%d", source, ctx.World.Player.X, ctx.World.Player.Y, targetX, targetY, headDir, bodyDir)
	if err := ctx.Network.SendChangeDirection(headDir, bodyDir); err != nil {
		m.status = source + " change direction failed: " + err.Error()
		log.Printf("%s change direction failed target=%d,%d head_dir=%d dir=%d: %v", source, targetX, targetY, headDir, bodyDir, err)
		m.walkCooldown = 30
		return
	}
	m.status = fmt.Sprintf("%s change direction: head=%d dir=%d", source, headDir, bodyDir)
	m.applyLocalDirection(ctx, headDir, bodyDir)
	m.walkCooldown = 6
}

func resolveTurnOnlyDirection(currentBodyDir int, currentHeadDir int, targetDir int) (uint8, uint8, bool) {
	bodyDir := normalizeDirectionIndex(currentBodyDir)
	headDir := normalizeHeadDir(currentHeadDir)
	targetDir = normalizeDirectionIndex(targetDir)
	delta := normalizeDirectionIndex(bodyDir - targetDir)

	resolvedBodyDir := bodyDir
	resolvedHeadDir := 0
	switch delta {
	case 0, 4:
		resolvedBodyDir = targetDir
	case 1:
		if headDir != 1 {
			resolvedHeadDir = 1
		} else {
			resolvedBodyDir = targetDir
		}
	case 2, 3:
		resolvedBodyDir = normalizeDirectionIndex(targetDir + 1)
		resolvedHeadDir = 1
	case 7:
		if headDir != 2 {
			resolvedHeadDir = 2
		} else {
			resolvedBodyDir = targetDir
		}
	case 5, 6:
		resolvedBodyDir = normalizeDirectionIndex(targetDir - 1)
		resolvedHeadDir = 2
	default:
		return 0, 0, false
	}
	return uint8(normalizeHeadDir(resolvedHeadDir)), uint8(resolvedBodyDir), true
}

func normalizeHeadDir(headDir int) int {
	if headDir < 0 {
		return 0
	}
	if headDir > 2 {
		return 2
	}
	return headDir
}

func (m *WorldMode) applyLocalDirection(ctx client.Context, headDir, dir uint8) {
	if ctx.World == nil {
		return
	}
	ctx.World.Player.Dir = int(dir & 7)
	ctx.World.Player.HeadDir = uint8(normalizeHeadDir(int(headDir)))
	ctx.World.Dir = ctx.World.Player.Dir
	if ctx.Session != nil {
		ctx.Session.PlayerDir = ctx.World.Player.Dir
	}
}

func (m *WorldMode) requestAttack(ctx client.Context, actor worldstate.Actor, source string) {
	if ctx.Network == nil {
		m.status = "attack request failed: not connected"
		m.walkCooldown = 30
		return
	}
	m.lockAttack(actor.ID)
	attackRange := currentNormalAttackRange(ctx)
	if attackTargetWithinRange(ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange) {
		m.sendAttackAction(ctx, actor, source)
		return
	}
	targetX, targetY, ok := attackApproachCell(ctx, actor, attackRange)
	if !ok {
		m.status = fmt.Sprintf("%s attack chase blocked: %d", source, actor.ID)
		log.Printf("%s attack chase blocked target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange)
		m.walkCooldown = 12
		return
	}
	m.pendingAttack = attackIntent{
		targetID: actor.ID,
		expires:  time.Now().Add(8 * time.Second),
	}
	log.Printf("%s attack chase target=%d player=%d,%d target=%d,%d range=%d chase=%d,%d", source, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange, targetX, targetY)
	m.requestWalk(ctx, targetX, targetY, source+" attack chase")
}

func (m *WorldMode) requestNPCTalk(ctx client.Context, actor worldstate.Actor, source string) {
	if ctx.Network == nil {
		m.status = "npc talk failed: not connected"
		m.walkCooldown = 30
		return
	}
	m.clearLockedAttack()
	if err := ctx.Network.SendNPCContact(actor.ID); err == nil {
		m.status = fmt.Sprintf("%s npc talk request: %d", source, actor.ID)
		m.walkCooldown = 12
	} else {
		m.status = source + " npc talk failed: " + err.Error()
		log.Printf("%s npc talk failed target=%d player=%d,%d target=%d,%d: %v", source, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, err)
		m.walkCooldown = 30
	}
}

func (m *WorldMode) lockAttack(targetID uint32) {
	if targetID == 0 || m.lockedAttackID == targetID {
		return
	}
	m.lockedAttackID = targetID
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
}

func (m *WorldMode) clearLockedAttack() {
	m.lockedAttackID = 0
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
}

func (m *WorldMode) continuePendingAttack(ctx client.Context, source string) {
	if m.pendingAttack.targetID == 0 {
		return
	}
	now := time.Now()
	if now.After(m.pendingAttack.expires) {
		log.Printf("%s pending attack expired target=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		log.Printf("%s pending attack target vanished id=%d", source, m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	if !attackTargetWithinRange(ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange) {
		log.Printf("%s pending attack still out of range target=%d player=%d,%d target=%d,%d range=%d", source, actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange)
		return
	}
	readyAt := pendingAttackReadyAt(ctx.World.Player, now)
	if m.pendingAttack.readyAt.IsZero() || readyAt.After(m.pendingAttack.readyAt) {
		m.pendingAttack.readyAt = readyAt
	}
	log.Printf("%s pending attack scheduled target=%d delay_ms=%d", source, actor.ID, maxInt(0, int(m.pendingAttack.readyAt.Sub(now).Milliseconds())))
}

func (m *WorldMode) processPendingAttack(ctx client.Context) {
	if m.pendingAttack.targetID == 0 || m.pendingAttack.readyAt.IsZero() {
		return
	}
	now := time.Now()
	if now.After(m.pendingAttack.expires) {
		log.Printf("pending attack expired target=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	if now.Before(m.pendingAttack.readyAt) {
		return
	}
	actor, ok := ctx.World.Actors[m.pendingAttack.targetID]
	if !ok {
		log.Printf("pending attack target vanished id=%d", m.pendingAttack.targetID)
		m.pendingAttack = attackIntent{}
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	if !attackTargetWithinRange(ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange) {
		log.Printf("pending attack became out of range target=%d player=%d,%d target=%d,%d range=%d", actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange)
		m.pendingAttack.readyAt = time.Time{}
		m.requestAttack(ctx, actor, "pending")
		return
	}
	m.pendingAttack = attackIntent{}
	m.sendAttackAction(ctx, actor, "pending")
}

func (m *WorldMode) processLockedAttack(ctx client.Context) {
	if m.lockedAttackID == 0 || ctx.Network == nil {
		return
	}
	if m.pendingAttack.targetID == m.lockedAttackID {
		return
	}
	now := time.Now()
	if ctx.World.Player.IsMovingAt(now) {
		return
	}
	actor, ok := ctx.World.Actors[m.lockedAttackID]
	if !ok {
		log.Printf("locked attack target vanished id=%d", m.lockedAttackID)
		m.clearLockedAttack()
		return
	}
	if !actorCanBeAttackClicked(ctx, actor) {
		log.Printf("locked attack target no longer attackable id=%d object_type=%d", actor.ID, actor.ObjectType)
		m.clearLockedAttack()
		return
	}
	attackRange := currentNormalAttackRange(ctx)
	if attackTargetWithinRange(ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange) {
		if !attackRetryDue(m.lastAttackAt, now) {
			return
		}
		log.Printf("locked attack retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange)
		m.sendAttackAction(ctx, actor, "locked")
		return
	}
	if !attackRetryDue(m.lastChaseAt, now) {
		return
	}
	m.lastChaseAt = now
	log.Printf("locked attack chase retry target=%d player=%d,%d target=%d,%d range=%d", actor.ID, ctx.World.Player.X, ctx.World.Player.Y, actor.X, actor.Y, attackRange)
	m.requestAttack(ctx, actor, "locked")
}

func attackRetryDue(last time.Time, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= attackRetryInterval
}

func pendingAttackReadyAt(player worldstate.Actor, now time.Time) time.Time {
	readyAt := now.Add(60 * time.Millisecond)
	if player.IsMovingAt(now) && player.MoveDuration > 0 {
		walkReadyAt := player.MoveStarted.Add(player.MoveDuration).Add(60 * time.Millisecond)
		if walkReadyAt.After(readyAt) {
			readyAt = walkReadyAt
		}
	}
	return readyAt
}

func (m *WorldMode) sendAttackAction(ctx client.Context, actor worldstate.Actor, source string) {
	if err := ctx.Network.SendActionRequest(actor.ID, network.ActionAttack); err == nil {
		m.status = fmt.Sprintf("%s attack request: %d", source, actor.ID)
		m.lastAttackAt = time.Now()
		m.walkCooldown = 12
	} else {
		m.status = source + " attack request failed: " + err.Error()
		log.Printf("%s attack request failed target=%d action=%d: %v", source, actor.ID, network.ActionAttack, err)
		m.walkCooldown = 30
	}
}

func (m *WorldMode) applyActorActionNotify(ctx client.Context, action network.ActorActionNotify) {
	log.Printf("actor action src=%d dst=%d skill=%d level=%d damage=%d left_damage=%d hits=%d action=%d src_speed=%d dst_speed=%d tick=%d", action.SourceID, action.TargetID, action.SkillID, action.SkillLevel, action.Damage, action.LeftDamage, action.HitCount, action.Action, action.SourceSpeed, action.TargetSpeed, action.ServerTick)
	now := time.Now()
	if action.Action == network.ActorActionPickupItem {
		m.applyActorPickupActionNotify(ctx, action, now)
		return
	}
	if action.Action == network.ActionSitDown || action.Action == network.ActionStandUp {
		m.applyActorSitStandActionNotify(ctx, action)
		return
	}
	source, sourceOK, sourceLocal := actorForCombatID(ctx, action.SourceID)
	target, targetOK, targetLocal := actorForCombatID(ctx, action.TargetID)
	if sourceOK && targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		source.Dir = directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	}
	attackDuration := combatDuration(action.SourceSpeed, defaultAttackAnimationDuration)
	attackFamily := spriteActionNonPCAttack
	if sourceOK {
		attackFamily = skillActionFamilyForActor(source, action.SkillID)
		m.startCombatAnimation(ctx, action.SourceID, attackFamily, now, attackDuration)
	}
	hitDelay := combatDuration(action.SourceSpeed, 0)
	if sourceOK && !res.HasPlayerJobToken(int(source.Job)) {
		if actionDef, ok := m.nonPCResolvedAction(ctx, source, attackFamily); ok {
			hitDelay = combatHitDelayFromAction(actionDef, attackDuration)
			if sound := actionSoundName(m.nonPCActionACT(ctx, source), actionDef, firstActionSoundMotion(actionDef)); sound != "" {
				m.scheduleSound(now.Add(hitDelay), sound)
			}
		}
	}
	hitAt := now.Add(hitDelay)
	if targetOK && actionHasHitReaction(action) {
		if hitAt.Before(now) {
			hitAt = now
		}
		m.addSkillBeginEffect(ctx, action, now)
		m.addNormalAttackBeforeHitEffect(ctx, action, source, sourceOK, now)
		m.addSkillBeforeHitEffect(ctx, action, now)
		if skillTargetUsesHitReaction(action, sourceLocal, targetLocal) {
			m.startCombatAnimation(ctx, action.TargetID, hurtActionFamilyForActor(target), hitAt, combatDuration(action.TargetSpeed, defaultHitAnimationDuration))
		}
		m.scheduleSound(hitAt, combatHitSFXCandidates(source, sourceOK, target, targetOK)...)
		m.addSkillEffect(ctx, action, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		m.applyCombatLifeFallback(ctx, target, targetLocal, action, hitAt)
		if targetLocal {
			ctx.World.Player.Moving = false
		}
	}
	x, y := ctx.World.Player.X, ctx.World.Player.Y
	if targetOK {
		x, y = target.X, target.Y
	} else if isLocalActor(ctx, action.TargetID) {
		x, y = ctx.World.Player.X, ctx.World.Player.Y
	}
	m.addActionDamageFloaters(action, targetLocal, sourceLocal, x, y, hitAt)
}

func (m *WorldMode) addSkillBeginEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillBeginEffectIDs(action.SkillID) {
		actorID := action.SourceID
		if effectDetachesLocalActor(effectID) && isLocalActor(ctx, actorID) {
			actorID = 0
		}
		if m.addWorldEffectAt(ctx, effectID, actorID, starts) {
			log.Printf("skill begin effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addNormalAttackBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, source worldstate.Actor, sourceOK bool, starts time.Time) {
	if action.SkillID != 0 || !sourceOK || !actorUsesBow(ctx.Resources, source) {
		return
	}
	if m.addWorldEffectBetweenAt(ctx, effectArrowShot, action.TargetID, action.SourceID, starts) {
		log.Printf("normal attack before-hit effect src=%d target=%d effect=%d", action.SourceID, action.TargetID, effectArrowShot)
	}
}

func actorUsesBow(manager *res.Manager, actor worldstate.Actor) bool {
	if actor.Weapon <= 0 {
		return false
	}
	return res.PlayerWeaponViewID(manager, int(actor.Weapon)) == 11
}

func (m *WorldMode) addSkillEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	for _, effectID := range skillEffectIDs(action.SkillID) {
		if m.addWorldEffectBetweenAt(ctx, effectID, action.TargetID, action.SourceID, starts) {
			log.Printf("skill effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
	for _, effectID := range skillEffectOnCasterIDs(action.SkillID) {
		if m.addWorldEffectAt(ctx, effectID, action.SourceID, starts) {
			log.Printf("skill caster effect skill=%d src=%d target=%d effect=%d", action.SkillID, action.SourceID, action.TargetID, effectID)
		}
	}
}

func (m *WorldMode) addSkillBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 {
		return
	}
	effectIDs := skillBeforeHitEffectIDs(action.SkillID)
	selfEffectIDs := skillBeforeHitEffectSelfIDs(action.SkillID)
	if len(effectIDs) == 0 && len(selfEffectIDs) == 0 {
		return
	}
	count := actionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(multiHitDelay * time.Duration(i))
		for _, effectID := range effectIDs {
			if m.addWorldEffectBetweenAt(ctx, effectID, action.TargetID, action.SourceID, effectStarts) {
				log.Printf("skill before-hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range selfEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				log.Printf("skill before-hit self effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

func (m *WorldMode) addSkillHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) {
	if action.SkillID == 0 || action.Damage == 0 {
		return
	}
	effectIDs := skillHitEffectIDs(action.SkillID)
	casterEffectIDs := skillHitEffectOnCasterIDs(action.SkillID)
	if len(effectIDs) == 0 && len(casterEffectIDs) == 0 {
		return
	}
	count := actionVisualHitCount(action)
	for i := 0; i < count; i++ {
		effectStarts := starts.Add(multiHitDelay * time.Duration(i))
		for _, effectID := range effectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.TargetID, effectStarts) {
				log.Printf("skill hit effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
		for _, effectID := range casterEffectIDs {
			if m.addWorldEffectAt(ctx, effectID, action.SourceID, effectStarts) {
				log.Printf("skill hit caster effect skill=%d src=%d target=%d effect=%d hit=%d/%d", action.SkillID, action.SourceID, action.TargetID, effectID, i+1, count)
			}
		}
	}
}

const multiHitDelay = 200 * time.Millisecond

func actionVisualHitCount(action network.ActorActionNotify) int {
	if action.HitCount == 0 {
		return 1
	}
	return maxInt(1, int(action.HitCount))
}

func (m *WorldMode) addActionDamageFloaters(action network.ActorActionNotify, targetLocal, sourceLocal bool, x, y int, hitAt time.Time) {
	text, kind, floaterColor := actionDamageFloater(action, targetLocal, sourceLocal)
	if text == "" {
		return
	}
	count := actionVisualHitCount(action)
	if count <= 1 || action.Damage+action.LeftDamage <= 0 || kind == damageFloaterCritical {
		m.damageFloaters = append(m.damageFloaters, damageFloater{
			actorID: action.TargetID,
			x:       x,
			y:       y,
			text:    text,
			color:   floaterColor,
			kind:    kind,
			starts:  hitAt,
			expires: hitAt.Add(damageFloaterDuration(kind)),
		})
		return
	}
	total := int(action.Damage + action.LeftDamage)
	base := total / count
	rem := total % count
	for i := 0; i < count; i++ {
		value := base
		if i < rem {
			value++
		}
		starts := hitAt.Add(multiHitDelay * time.Duration(i))
		m.damageFloaters = append(m.damageFloaters, damageFloater{
			actorID: action.TargetID,
			x:       x,
			y:       y,
			text:    strconv.Itoa(value),
			color:   floaterColor,
			kind:    kind,
			starts:  starts,
			expires: starts.Add(damageFloaterDuration(kind)),
		})
	}
}

func (m *WorldMode) applyActorPickupActionNotify(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	source, sourceOK, sourceLocal := actorForCombatID(ctx, action.SourceID)
	if !sourceOK {
		return
	}
	if ctx.World != nil {
		if item, ok := ctx.World.Items[action.TargetID]; ok {
			dir := directionFromDelta(source.X, source.Y, item.X, item.Y, source.Dir)
			if sourceLocal {
				ctx.World.Player.Dir = dir
				ctx.World.Dir = dir
				if ctx.Session != nil {
					ctx.Session.PlayerDir = dir
				}
			} else {
				source.Dir = dir
				ctx.World.UpsertActor(source)
			}
		}
	}
	m.startCombatAnimation(ctx, action.SourceID, spriteActionPickup, now, pickupAnimationDuration)
}

func (m *WorldMode) applyActorSitStandActionNotify(ctx client.Context, action network.ActorActionNotify) {
	id := action.SourceID
	if id == 0 {
		id = action.TargetID
	}
	if id == 0 || ctx.World == nil {
		return
	}
	sitting := action.Action == network.ActionSitDown
	if isLocalActor(ctx, id) {
		ctx.World.Player.Sitting = sitting
		if sitting {
			ctx.World.Player.Moving = false
		}
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	actor.Sitting = sitting
	if sitting {
		actor.Moving = false
	}
	ctx.World.UpsertActor(actor)
	if !sitting {
		actor = ctx.World.Actors[id]
		actor.Sitting = false
		ctx.World.Actors[id] = actor
	}
}

func (m *WorldMode) applyActorHPUpdate(update network.ActorHPUpdate) {
	if update.ID == 0 || update.MaxHP <= 0 {
		return
	}
	hp := update.HP
	if hp < 0 {
		hp = 0
	}
	if hp > update.MaxHP {
		hp = update.MaxHP
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	m.actorLife[update.ID] = actorLife{
		hp:        hp,
		maxHP:     update.MaxHP,
		estimated: false,
		fromTiny:  update.Tiny,
		updatedAt: time.Now(),
	}
	log.Printf("actor hp id=%d hp=%d max_hp=%d tiny=%t", update.ID, hp, update.MaxHP, update.Tiny)
}

func (m *WorldMode) applyCombatLifeFallback(ctx client.Context, target worldstate.Actor, targetLocal bool, action network.ActorActionNotify, hitAt time.Time) {
	if targetLocal || !actorCanBeAttackClicked(ctx, target) {
		return
	}
	damage := int(action.Damage + action.LeftDamage)
	if damage <= 0 {
		return
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	life, ok := m.actorLife[target.ID]
	if !ok || life.maxHP <= 0 {
		maxHP := estimatedMonsterMaxHP(int(target.Job))
		if maxHP <= 0 {
			maxHP = estimatedUnknownMonsterMaxHP(damage)
		}
		life = actorLife{hp: maxHP, maxHP: maxHP, estimated: true}
	}
	if life.fromTiny {
		return
	}
	life.hp -= damage
	if life.hp < 0 {
		life.hp = 0
	}
	life.updatedAt = hitAt
	m.actorLife[target.ID] = life
}

func estimatedUnknownMonsterMaxHP(damage int) int {
	if damage <= 0 {
		return 100
	}
	return max(100, damage*3)
}

func estimatedMonsterMaxHP(job int) int {
	if hp, ok := preRenewalMonsterMaxHP[job]; ok {
		return hp
	}
	return 0
}

var preRenewalMonsterMaxHP = map[int]int{
	1001: 1109,  // Scorpion
	1002: 50,    // Poring
	1004: 169,   // Hornet
	1005: 155,   // Familiar
	1007: 63,    // Fabre
	1008: 427,   // Pupa
	1009: 92,    // Condor
	1010: 95,    // Willow
	1011: 67,    // Chonchon
	1012: 133,   // Roda Frog
	1013: 919,   // Wolf
	1014: 510,   // Spore
	1015: 534,   // Zombie
	1016: 3040,  // Archer Skeleton
	1018: 595,   // Creamy
	1019: 531,   // Peco Peco
	1020: 405,   // Mandragora
	1023: 1400,  // Orc Warrior
	1024: 426,   // Wormtail
	1025: 471,   // Boa
	1026: 2872,  // Munak
	1028: 2334,  // Soldier Skeleton
	1031: 344,   // Poporing
	1033: 693,   // Elder Willow
	1034: 2152,  // Thara Frog
	1036: 5418,  // Ghoul
	1040: 3900,  // Golem
	1041: 5176,  // Mummy
	1044: 3952,  // Obeaune
	1045: 6900,  // Marc
	1047: 420,   // Peco Peco Egg
	1048: 48,    // Thief Bug Egg
	1049: 80,    // Picky
	1050: 83,    // Picky
	1051: 126,   // Thief Bug
	1052: 198,   // Rocker
	1053: 170,   // Thief Bug Female
	1054: 583,   // Thief Bug Male
	1055: 610,   // Muka
	1056: 641,   // Smokie
	1057: 879,   // Yoyo
	1058: 926,   // Metaller
	1060: 1619,  // Bigfoot
	1062: 69,    // Santa Poring
	1063: 60,    // Lunatic
	1064: 1648,  // Megalodon
	1065: 11990, // Strouf
	1066: 1017,  // Vadon
	1067: 1620,  // Cornutus
	1068: 660,   // Hydra
	1069: 4299,  // Swordfish
	1070: 507,   // Kukre
	1071: 1676,  // Pirate Skeleton
	1073: 2451,  // Crab
	1074: 920,   // Shellfish
	1076: 234,   // Skeleton
	1077: 665,   // Poison Spore
	1078: 10,    // Red Plant
	1079: 10,    // Blue Plant
	1080: 10,    // Green Plant
	1081: 10,    // Yellow Plant
	1082: 10,    // White Plant
	1083: 20,    // Shining Plant
	1084: 15,    // Black Mushroom
	1085: 15,    // Red Mushroom
	1094: 495,   // Ambernite
	1095: 688,   // Andre
	1097: 420,   // Ant Egg
	1104: 817,   // Coco
	1105: 760,   // Deniro
	1113: 55,    // Drops
	1114: 140,   // Dustiness
	1117: 500,   // Baby Desert Wolf
	1166: 457,   // Savage Babe
	1175: 284,   // Tarou
}

func actorForCombatID(ctx client.Context, id uint32) (worldstate.Actor, bool, bool) {
	if ctx.World == nil || id == 0 {
		return worldstate.Actor{}, false, false
	}
	if isLocalActor(ctx, id) {
		actor := ctx.World.Player
		character := selectedCharacter(ctx.Session)
		actor.ID = id
		actor.Job = character.Job
		actor.Head = character.Hair
		actor.Weapon = character.Weapon
		actor.Shield = character.Shield
		actor.HeadTop = character.HeadTop
		actor.HeadMid = character.HeadMid
		actor.HeadLow = character.HeadLow
		actor.Sex = ctx.Session.Sex
		actor.Appearance = true
		return actor, true, true
	}
	actor, ok := ctx.World.Actors[id]
	return actor, ok, false
}

func (m *WorldMode) faceCombatSource(ctx client.Context, source worldstate.Actor, sourceLocal bool, target worldstate.Actor) {
	dir := directionFromDelta(source.X, source.Y, target.X, target.Y, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	ctx.World.UpsertActor(source)
}

func (m *WorldMode) startActorAnimation(id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(id, actionFamily, started, duration, false)
}

func (m *WorldMode) startHeldActorAnimation(id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimationWithOptions(id, actionFamily, started, duration, true)
}

func (m *WorldMode) startActorAnimationWithOptions(id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	if id == 0 || actionFamily < 0 {
		return
	}
	if m.actorAnims == nil {
		m.actorAnims = make(map[uint32]actorAnimation)
	}
	m.actorAnims[id] = actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		holdFinal:    holdFinal,
	}
}

func (m *WorldMode) startCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startActorAnimation(id, actionFamily, started, duration)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startActorAnimation(ctx.Session.AccountID, actionFamily, started, duration)
	m.startActorAnimation(ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) startHeldCombatAnimation(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration) {
	m.startHeldActorAnimation(id, actionFamily, started, duration)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.startHeldActorAnimation(ctx.Session.AccountID, actionFamily, started, duration)
	m.startHeldActorAnimation(ctx.Session.CharID, actionFamily, started, duration)
}

func (m *WorldMode) clearLocalDeathStateIfAlive(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	if ctx.Session.Vitals.HP <= 0 && ctx.Session.Selected.HP <= 0 {
		return
	}
	m.clearLocalDeathState(ctx)
}

func (m *WorldMode) clearLocalDeathState(ctx client.Context) {
	m.deathModal.Reset()
	if ctx.Session == nil || m.actorAnims == nil {
		return
	}
	m.clearActorDeathAnimation(ctx.Session.AccountID)
	m.clearActorDeathAnimation(ctx.Session.CharID)
}

func (m *WorldMode) clearActorDeathAnimation(id uint32) {
	if id == 0 || m.actorAnims == nil {
		return
	}
	anim, ok := m.actorAnims[id]
	if !ok {
		return
	}
	if anim.actionFamily != spriteActionPCDeath && anim.actionFamily != spriteActionNonPCDeath {
		return
	}
	delete(m.actorAnims, id)
}

func (m *WorldMode) actorAnimation(id uint32, now time.Time) (actorAnimation, bool) {
	if m.actorAnims == nil || id == 0 {
		return actorAnimation{}, false
	}
	anim, ok := m.actorAnims[id]
	if !ok {
		return actorAnimation{}, false
	}
	if anim.duration <= 0 {
		anim.duration = defaultAttackAnimationDuration
	}
	if now.Before(anim.started) {
		return actorAnimation{}, false
	}
	if !now.Before(anim.started.Add(anim.duration)) {
		if anim.holdFinal {
			return anim, true
		}
		delete(m.actorAnims, id)
		return actorAnimation{}, false
	}
	return anim, true
}

func attackActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		if isSecondPCAttack(int(actor.Job), actor.Sex, int(actor.Weapon)) {
			return spriteActionPCAttack3
		}
		return spriteActionPCAttack2
	}
	return spriteActionNonPCAttack
}

func skillActionFamilyForActor(actor worldstate.Actor, skillID uint16) int {
	if skillID == 0 {
		return attackActionFamilyForActor(actor)
	}
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionNonPCAttack
	}
	switch skillAction(skillID) {
	case roBrowserSkillActionAttack:
		return attackActionFamilyForActor(actor)
	case roBrowserSkillActionReadyFight:
		return spriteActionPCReadyFight
	default:
		return spriteActionPCSkill
	}
}

func skillCastActionFamilyForActor(actor worldstate.Actor, skillID uint16) int {
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionNonPCAttack
	}
	return spriteActionPCReadyFight
}

func skillTargetUsesHitReaction(action network.ActorActionNotify, sourceLocal, targetLocal bool) bool {
	if action.SkillID > 0 && sourceLocal && targetLocal && action.Action == network.ActorActionSkill {
		return false
	}
	return true
}

func hurtActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCHurt
	}
	return spriteActionNonPCHurt
}

func deathActionFamilyForActor(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return spriteActionPCDeath
	}
	return spriteActionNonPCDeath
}

func isSecondPCAttack(job int, sex byte, weaponValue int) bool {
	weaponType := res.PlayerWeaponType(weaponValue)
	switch job {
	case 0, 23, 4001, 4045:
		if sex != 0 {
			return weaponType == 2 || weaponType == 3 || (weaponType >= 6 && weaponType <= 10) || weaponType == 23
		}
		return weaponType == 1
	case 1, 7, 13, 14, 21:
		return weaponType >= 4 && weaponType <= 5
	case 2, 5:
		return weaponType == 1
	case 3:
		return weaponType != 11
	case 6, 11, 17, 19, 20:
		return weaponType == 11
	case 8:
		return weaponType == 15
	case 10, 18:
		return weaponType == 2 || (weaponType > 5 && weaponType <= 8)
	case 12:
		return weaponType == 16 || (weaponType > 24 && weaponType <= 30)
	case 15:
		return weaponType == 0 || weaponType == 12
	case 16:
		return weaponType == 5 || weaponType == 10 || weaponType == 15 || weaponType == 23
	case 24:
		return weaponType >= 18 && weaponType <= 21
	case 25:
		return weaponType == 22
	default:
		return false
	}
}

func combatDuration(speed int32, fallback time.Duration) time.Duration {
	if speed <= 0 {
		return fallback
	}
	duration := time.Duration(speed) * time.Millisecond
	if duration > maxCombatAnimationDuration {
		return maxCombatAnimationDuration
	}
	return duration
}

func actionAnimationDuration(action res.ACTAction, fallback time.Duration) time.Duration {
	if len(action.Animations) == 0 {
		return fallback
	}
	delayMS := float64(action.DelayMS)
	if delayMS <= 0 {
		delayMS = 150
	}
	duration := time.Duration(delayMS * float64(time.Millisecond) * float64(len(action.Animations)))
	if duration <= 0 {
		return fallback
	}
	if duration > maxCombatAnimationDuration {
		return maxCombatAnimationDuration
	}
	return duration
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func combatHitDelayFromAction(action res.ACTAction, duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	motion := firstActionSoundMotion(action)
	if motion >= 0 && len(action.Animations) > 0 {
		return duration * time.Duration(motion) / time.Duration(len(action.Animations))
	}
	return duration / 2
}

func firstActionSoundMotion(action res.ACTAction) int {
	for index, animation := range action.Animations {
		if animation.Sound >= 0 {
			return index
		}
	}
	return -1
}

func actionSoundName(act *res.ACT, action res.ACTAction, motion int) string {
	if act == nil || motion < 0 || motion >= len(action.Animations) {
		return ""
	}
	soundIndex := action.Animations[motion].Sound
	if soundIndex < 0 || soundIndex >= len(act.Sounds) {
		return ""
	}
	sound := strings.TrimSpace(act.Sounds[soundIndex])
	if strings.EqualFold(sound, "atk") {
		return ""
	}
	return sound
}

func (m *WorldMode) nonPCResolvedAction(ctx client.Context, actor worldstate.Actor, actionFamily int) (res.ACTAction, bool) {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return res.ACTAction{}, false
	}
	_, action, ok := resolveSpriteAction(view.act, actionFamily, actor.Dir)
	return action, ok
}

func (m *WorldMode) nonPCActionACT(ctx client.Context, actor worldstate.Actor) *res.ACT {
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return nil
	}
	return view.act
}

func (m *WorldMode) actorActionDuration(ctx client.Context, actor worldstate.Actor, actionFamily int, fallback time.Duration) time.Duration {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return fallback
	}
	if action, ok := m.nonPCResolvedAction(ctx, actor, actionFamily); ok {
		return actionAnimationDuration(action, fallback)
	}
	return fallback
}

func combatHitSFXCandidates(source worldstate.Actor, sourceOK bool, target worldstate.Actor, targetOK bool) []string {
	if targetOK && res.HasPlayerJobToken(int(target.Job)) {
		return []string{"player_clothes.wav", "player_wooden_male.wav", "player_metal.wav"}
	}
	if sourceOK && res.HasPlayerJobToken(int(source.Job)) {
		return weaponHitSFXCandidates(res.PlayerWeaponType(int(source.Weapon)))
	}
	return []string{"_enemy_hit_normal1.wav", "_enemy_hit_normal2.wav", "_enemy_hit_normal3.wav", "_enemy_hit_normal4.wav"}
}

func weaponHitSFXCandidates(weaponType int) []string {
	switch weaponType {
	case 1, 2, 3:
		return []string{"_hit_sword.wav", "_enemy_hit_normal1.wav"}
	case 4, 5:
		return []string{"_hit_spear.wav", "_enemy_hit_normal1.wav"}
	case 6, 7:
		return []string{"_hit_axe.wav", "_enemy_hit_normal1.wav"}
	case 11:
		return []string{"_hit_arrow.wav", "_enemy_hit_normal1.wav"}
	case 0, 8, 9, 10, 15, 23:
		return []string{"_hit_mace.wav", "_enemy_hit_normal1.wav"}
	default:
		return []string{"_enemy_hit_normal1.wav", "_enemy_hit_normal2.wav", "_enemy_hit_normal3.wav", "_enemy_hit_normal4.wav"}
	}
}

func (m *WorldMode) scheduleSound(at time.Time, paths ...string) {
	clean := paths[:0]
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			clean = append(clean, path)
		}
	}
	if len(clean) == 0 {
		return
	}
	m.scheduledSounds = append(m.scheduledSounds, scheduledSound{
		at:    at,
		paths: append([]string(nil), clean...),
	})
}

func (m *WorldMode) playDueScheduledSounds(ctx client.Context, now time.Time) {
	if len(m.scheduledSounds) == 0 {
		return
	}
	active := m.scheduledSounds[:0]
	for _, sound := range m.scheduledSounds {
		if now.Before(sound.at) {
			active = append(active, sound)
			continue
		}
		m.playSFXFirst(ctx, sound.paths...)
	}
	m.scheduledSounds = active
}

func (m *WorldMode) processActorMotionSounds(ctx client.Context, now time.Time) {
	if ctx.World == nil || len(ctx.World.Actors) == 0 {
		return
	}
	for _, actor := range ctx.World.Actors {
		m.processNonPCMotionSound(ctx, actor, now)
	}
}

func (m *WorldMode) processNonPCMotionSound(ctx client.Context, actor worldstate.Actor, now time.Time) {
	if actor.ID == 0 || res.HasPlayerJobToken(int(actor.Job)) || !actorWithinSoundRange(ctx, actor, now) {
		return
	}
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil || view.act == nil {
		return
	}
	state := m.nonPCSpriteState(actor, now)
	switch state.actionFamily {
	case spriteActionNonPCAttack, spriteActionNonPCHurt:
		return
	}
	_, action, ok := resolveSpriteAction(view.act, state.actionFamily, state.direction)
	if !ok || len(action.Animations) == 0 {
		return
	}
	motion := bodyMotionForState(action, state, view.started, now)
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	soundIndex := action.Animations[motion].Sound
	current := actorSoundFrame{actionFamily: state.actionFamily, motion: motion, soundIndex: soundIndex}
	if m.actorSoundFrames == nil {
		m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	}
	if previous, ok := m.actorSoundFrames[actor.ID]; ok && previous == current {
		return
	}
	m.actorSoundFrames[actor.ID] = current
	if soundIndex < 0 {
		return
	}
	if sound := actionSoundName(view.act, action, motion); sound != "" {
		m.scheduleSound(now, sound)
	}
}

func actorWithinSoundRange(ctx client.Context, actor worldstate.Actor, now time.Time) bool {
	if ctx.World == nil {
		return false
	}
	actorX, actorY := actor.RenderPosition(now)
	playerX, playerY := ctx.World.Player.RenderPosition(now)
	const soundRangeCells = 25
	return math.Hypot(actorX-playerX, actorY-playerY) <= soundRangeCells
}

func (m *WorldMode) playSFXFirst(ctx client.Context, paths ...string) {
	if ctx.Audio == nil {
		return
	}
	var lastErr error
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
		lastErr = err
	}
	if lastErr != nil {
		log.Printf("sfx failed paths=%v: %v", paths, lastErr)
	}
}

func actionHasHitReaction(action network.ActorActionNotify) bool {
	if action.Action == 4 || action.Action == 9 || action.Action == 11 {
		return false
	}
	return action.Damage > 0 || action.LeftDamage > 0
}

func (m *WorldMode) applyAttackFailureForDistance(ctx client.Context, failure network.AttackFailureForDistance) {
	attackRange := maxInt(1, failure.AttackRange)
	if ctx.Session != nil {
		ctx.Session.AttackRange = attackRange
	}
	log.Printf("attack distance failure target=%d server_player=%d,%d server_target=%d,%d range=%d client_player=%d,%d", failure.TargetID, failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange, ctx.World.Player.X, ctx.World.Player.Y)
	ctx.World.SetPlayerPosition(failure.SourceX, failure.SourceY, ctx.World.Player.Dir)
	if actor, ok := ctx.World.Actors[failure.TargetID]; ok {
		actor.X = failure.TargetX
		actor.Y = failure.TargetY
		actor.Moving = false
		actor.FromX = failure.TargetX
		actor.FromY = failure.TargetY
		actor.ToX = failure.TargetX
		actor.ToY = failure.TargetY
		actor.MovePath = nil
		ctx.World.UpsertActor(actor)
	}
	if m.lockedAttackID != failure.TargetID && m.pendingAttack.targetID != failure.TargetID {
		return
	}
	m.pendingAttack = attackIntent{}
	m.lastAttackAt = time.Now()
	if !attackTargetWithinRange(failure.SourceX, failure.SourceY, failure.TargetX, failure.TargetY, attackRange) {
		if actor, ok := ctx.World.Actors[failure.TargetID]; ok {
			m.requestAttack(ctx, actor, "attack failure")
		}
	}
}

func applyParameterChange(ctx client.Context, change network.ParameterChange) {
	if ctx.Session == nil {
		return
	}
	value := int(change.Value)
	switch change.VarID {
	case network.StatusBaseExp:
		ctx.Session.Progress.BaseExp = change.Value
	case network.StatusJobExp:
		ctx.Session.Progress.JobExp = change.Value
	case network.StatusHP:
		ctx.Session.Vitals.HP = value
		ctx.Session.Selected.HP = clampInt16(value)
	case network.StatusMaxHP:
		ctx.Session.Vitals.MaxHP = value
		ctx.Session.Selected.MaxHP = clampInt16(value)
	case network.StatusSP:
		ctx.Session.Vitals.SP = value
		ctx.Session.Selected.SP = clampInt16(value)
	case network.StatusMaxSP:
		ctx.Session.Vitals.MaxSP = value
		ctx.Session.Selected.MaxSP = clampInt16(value)
	case network.StatusPoint:
		ctx.Session.Stats.Points = value
	case network.StatusBaseLevel:
		ctx.Session.Progress.BaseLevel = value
		ctx.Session.Selected.Level = clampInt16(value)
	case network.StatusSkillPoint:
		ctx.Session.Skills.Points = value
	case network.StatusStr, network.StatusAgi, network.StatusVit, network.StatusInt, network.StatusDex, network.StatusLuk:
		setSessionStat(ctx.Session, change.VarID, value)
	case network.StatusUStr, network.StatusUAgi, network.StatusUVit, network.StatusUInt, network.StatusUDex, network.StatusULuk:
		setSessionStatCost(ctx.Session, change.VarID, value)
	case network.StatusZeny:
		ctx.Session.Inventory.Zeny = change.Value
	case network.StatusNextBaseExp:
		ctx.Session.Progress.NextBaseExp = change.Value
	case network.StatusNextJobExp:
		ctx.Session.Progress.NextJobExp = change.Value
	case network.StatusWeight:
		ctx.Session.Inventory.Weight = value
	case network.StatusMaxWeight:
		ctx.Session.Inventory.MaxWeight = value
	case network.StatusJobLevel:
		ctx.Session.Progress.JobLevel = value
		ctx.Session.Selected.JobLevel = clampInt16(value)
	default:
		return
	}
	log.Printf("parameter change var=%d value=%d hp=%d/%d sp=%d/%d base_lv=%d job_lv=%d base_exp=%d/%d job_exp=%d/%d zeny=%d weight=%d/%d",
		change.VarID,
		change.Value,
		ctx.Session.Vitals.HP,
		ctx.Session.Vitals.MaxHP,
		ctx.Session.Vitals.SP,
		ctx.Session.Vitals.MaxSP,
		ctx.Session.Progress.BaseLevel,
		ctx.Session.Progress.JobLevel,
		ctx.Session.Progress.BaseExp,
		ctx.Session.Progress.NextBaseExp,
		ctx.Session.Progress.JobExp,
		ctx.Session.Progress.NextJobExp,
		ctx.Session.Inventory.Zeny,
		ctx.Session.Inventory.Weight,
		ctx.Session.Inventory.MaxWeight)
}

func (m *WorldMode) applyParameterChange(ctx client.Context, change network.ParameterChange) {
	if ctx.Session == nil {
		return
	}
	previousHP := ctx.Session.Vitals.HP
	previousSP := ctx.Session.Vitals.SP
	previousBaseLevel := ctx.Session.Progress.BaseLevel
	previousJobLevel := ctx.Session.Progress.JobLevel
	applyParameterChange(ctx, change)
	if change.Value <= 0 {
		return
	}
	previousValues := map[uint16]int{
		network.StatusHP:        previousHP,
		network.StatusSP:        previousSP,
		network.StatusBaseLevel: previousBaseLevel,
		network.StatusJobLevel:  previousJobLevel,
	}
	if visual, ok := statusVisualEffects[change.VarID]; ok {
		visual.applyParameterChange(ctx, m, previousValues[change.VarID])
	}
}

var (
	recoveryHPColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	recoverySPColor = color.RGBA{R: 0, G: 0, B: 255, A: 255}
)

const (
	recoveryHPSFX = "_heal_effect.wav"
	recoverySPSFX = "effect\\\xC8\xED\xB1\xE2.wav"
)

var recoverySFXFallbacks = []string{"effect\\priest_recovery.wav"}

type statusVisualEffect struct {
	current       func(*session.Session) int
	recover       func(*session.Session, int) bool
	recovery      bool
	recoveryColor color.RGBA
	recoveryKind  damageFloaterKind
	recoverySFX   []string
	clearsDeath   bool
	levelEffectID int
}

var statusVisualEffects = map[uint16]statusVisualEffect{
	network.StatusHP: {
		current:       func(s *session.Session) int { return s.Vitals.HP },
		recover:       recoverSessionHP,
		recovery:      true,
		recoveryColor: recoveryHPColor,
		recoveryKind:  damageFloaterRecoveryHP,
		recoverySFX:   []string{recoveryHPSFX},
		clearsDeath:   true,
	},
	network.StatusSP: {
		current:       func(s *session.Session) int { return s.Vitals.SP },
		recover:       recoverSessionSP,
		recovery:      true,
		recoveryColor: recoverySPColor,
		recoveryKind:  damageFloaterRecoverySP,
		recoverySFX:   []string{recoverySPSFX},
	},
	network.StatusBaseLevel: {
		current:       func(s *session.Session) int { return s.Progress.BaseLevel },
		levelEffectID: effectBaseLevelUp,
	},
	network.StatusJobLevel: {
		current:       func(s *session.Session) int { return s.Progress.JobLevel },
		levelEffectID: effectJobLevelUp,
	},
}

func (v statusVisualEffect) applyParameterChange(ctx client.Context, mode *WorldMode, previous int) {
	if v.current == nil || mode == nil || ctx.Session == nil {
		return
	}
	current := v.current(ctx.Session)
	if v.recovery {
		delta := current - previous
		if delta > 0 {
			mode.addLocalRecoveryFloater(ctx, delta, v.recoveryColor, v.recoveryKind)
			mode.scheduleSound(time.Now(), v.sfxCandidates()...)
		}
		return
	}
	if v.levelEffectID > 0 && current > previous {
		mode.addWorldEffectIfMissing(ctx, v.levelEffectID, localSkillTarget(ctx))
	}
}

func (v statusVisualEffect) sfxCandidates() []string {
	if len(v.recoverySFX) == 0 {
		return append([]string(nil), recoverySFXFallbacks...)
	}
	paths := append([]string(nil), v.recoverySFX...)
	return append(paths, recoverySFXFallbacks...)
}

func recoverSessionHP(s *session.Session, amount int) bool {
	maxHP := s.Vitals.MaxHP
	if maxHP <= 0 {
		maxHP = int(s.Selected.MaxHP)
	}
	next := s.Vitals.HP + amount
	if maxHP > 0 && next > maxHP {
		next = maxHP
	}
	s.Vitals.HP = next
	s.Selected.HP = clampInt16(next)
	return true
}

func recoverSessionSP(s *session.Session, amount int) bool {
	maxSP := s.Vitals.MaxSP
	if maxSP <= 0 {
		maxSP = int(s.Selected.MaxSP)
	}
	next := s.Vitals.SP + amount
	if maxSP > 0 && next > maxSP {
		next = maxSP
	}
	s.Vitals.SP = next
	s.Selected.SP = clampInt16(next)
	return true
}

func (m *WorldMode) applyRecovery(ctx client.Context, recovery network.Recovery) {
	if ctx.Session == nil || recovery.Amount <= 0 {
		return
	}
	visual, ok := statusVisualEffects[recovery.StatusID]
	if !ok || visual.recover == nil {
		return
	}
	if visual.recover(ctx.Session, recovery.Amount) {
		m.addLocalRecoveryFloater(ctx, recovery.Amount, visual.recoveryColor, visual.recoveryKind)
		if visual.clearsDeath {
			m.clearLocalDeathStateIfAlive(ctx)
		}
		m.scheduleSound(time.Now(), visual.sfxCandidates()...)
	}
	log.Printf("recovery status=%d amount=%d hp=%d/%d sp=%d/%d", recovery.StatusID, recovery.Amount, ctx.Session.Vitals.HP, ctx.Session.Vitals.MaxHP, ctx.Session.Vitals.SP, ctx.Session.Vitals.MaxSP)
}

func (m *WorldMode) addLocalRecoveryFloater(ctx client.Context, amount int, floaterColor color.RGBA, kind damageFloaterKind) {
	if ctx.World == nil || amount <= 0 {
		return
	}
	now := time.Now()
	actorID := uint32(0)
	if ctx.Session != nil {
		actorID = ctx.Session.AccountID
		if actorID == 0 {
			actorID = ctx.Session.CharID
		}
	}
	m.damageFloaters = append(m.damageFloaters, damageFloater{
		actorID: actorID,
		x:       ctx.World.Player.X,
		y:       ctx.World.Player.Y,
		text:    fmt.Sprintf("%d", amount),
		color:   floaterColor,
		kind:    kind,
		starts:  now,
		expires: now.Add(damageFloaterDuration(kind)),
	})
}

func (m *WorldMode) addTargetRecoveryFloater(ctx client.Context, actorID uint32, amount int, floaterColor color.RGBA, kind damageFloaterKind, now time.Time) {
	if ctx.World == nil || actorID == 0 || amount <= 0 {
		return
	}
	x, y, ok := effectAnchor(ctx, actorID)
	if !ok {
		return
	}
	m.damageFloaters = append(m.damageFloaters, damageFloater{
		actorID: actorID,
		x:       x,
		y:       y,
		text:    strconv.Itoa(amount),
		color:   floaterColor,
		kind:    kind,
		starts:  now,
		expires: now.Add(damageFloaterDuration(kind)),
	})
}

func clampInt16(value int) int16 {
	if value < -32768 {
		return -32768
	}
	if value > 32767 {
		return 32767
	}
	return int16(value)
}

var (
	damageFloaterWhite  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterYellow = color.RGBA{R: 230, G: 230, B: 38, A: 255}
	damageFloaterRed    = color.RGBA{R: 255, G: 64, B: 64, A: 255}
)

func actionDamageFloater(action network.ActorActionNotify, targetLocal, sourceLocal bool) (string, damageFloaterKind, color.RGBA) {
	total := action.Damage + action.LeftDamage
	if total > 0 {
		if action.Action == 10 || action.Action == 13 {
			return strconv.Itoa(int(total)), damageFloaterCritical, damageFloaterYellow
		}
		if targetLocal && !sourceLocal {
			return strconv.Itoa(int(total)), damageFloaterIncoming, damageFloaterRed
		}
		return strconv.Itoa(int(total)), damageFloaterNormal, damageFloaterWhite
	}
	if action.Action == 11 {
		return "miss", damageFloaterMiss, damageFloaterWhite
	}
	if action.Action == 0 || action.Action == 7 {
		return "miss", damageFloaterMiss, damageFloaterWhite
	}
	return "", damageFloaterNormal, color.RGBA{}
}

func damageFloaterDuration(kind damageFloaterKind) time.Duration {
	switch kind {
	case damageFloaterMiss:
		return 800 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
}

func currentNormalAttackRange(ctx client.Context) int {
	attackRange := 1
	if ctx.Session != nil {
		attackRange = maxInt(attackRange, ctx.Session.AttackRange)
		attackRange = maxInt(attackRange, normalAttackRangeFromEquippedItems(ctx.Session, ctx.Resources))
	}
	return maxInt(1, attackRange)
}

func attackTargetWithinRange(playerX, playerY, targetX, targetY, attackRange int) bool {
	return maxInt(absInt(playerX-targetX), absInt(playerY-targetY)) <= maxInt(1, attackRange)
}

func attackApproachCell(ctx client.Context, actor worldstate.Actor, attackRange int) (int, int, bool) {
	attackRange = maxInt(1, attackRange)
	if attackRange > 1 {
		if x, y, ok := rangedAttackApproachCell(ctx, actor, attackRange); ok {
			return x, y, true
		}
	}
	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x := actor.X + dx
			y := actor.Y + dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			distance := math.Hypot(float64(x-ctx.World.Player.X), float64(y-ctx.World.Player.Y))
			if distance < bestDistance {
				bestDistance = distance
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, bestDistance < math.Inf(1)
}

func rangedAttackApproachCell(ctx client.Context, actor worldstate.Actor, attackRange int) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	playerX := ctx.World.Player.X
	playerY := ctx.World.Player.Y
	stepX := approachSign(actor.X - playerX)
	stepY := approachSign(actor.Y - playerY)
	preferredX := actor.X - stepX*attackRange
	preferredY := actor.Y - stepY*attackRange
	type candidate struct {
		x                 int
		y                 int
		sourceDistance    int
		preferredDistance int
	}
	candidates := make([]candidate, 0, (attackRange*2+1)*(attackRange*2+1))
	for dy := -attackRange; dy <= attackRange; dy++ {
		for dx := -attackRange; dx <= attackRange; dx++ {
			ringDistance := maxInt(absInt(dx), absInt(dy))
			if ringDistance == 0 || ringDistance > attackRange {
				continue
			}
			x := actor.X + dx
			y := actor.Y + dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			if ctx.World.GAT != nil && !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			candidates = append(candidates, candidate{
				x:                 x,
				y:                 y,
				sourceDistance:    maxInt(absInt(x-playerX), absInt(y-playerY)),
				preferredDistance: maxInt(absInt(x-preferredX), absInt(y-preferredY)),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.sourceDistance != right.sourceDistance {
			return left.sourceDistance < right.sourceDistance
		}
		if left.preferredDistance != right.preferredDistance {
			return left.preferredDistance < right.preferredDistance
		}
		if left.y != right.y {
			return left.y < right.y
		}
		return left.x < right.x
	})
	for _, candidate := range candidates {
		return candidate.x, candidate.y, true
	}
	return 0, 0, false
}

func approachSign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func (m *WorldMode) drawDamageFloaters(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time) {
	if len(m.damageFloaters) == 0 {
		return
	}
	active := m.damageFloaters[:0]
	for _, floater := range m.damageFloaters {
		if now.After(floater.expires) {
			continue
		}
		active = append(active, floater)
		if now.Before(floater.starts) {
			continue
		}
		x, y := float64(floater.x), float64(floater.y)
		if actor, ok := ctx.World.Actors[floater.actorID]; ok {
			x, y = actor.RenderPosition(now)
		} else if isLocalActor(ctx, floater.actorID) {
			x, y = ctx.World.Player.RenderPosition(now)
		}
		progress := damageFloaterProgress(floater, now)
		dx, dy, zLift, scale, alpha := damageFloaterPlacement(floater.kind, progress)
		floaterColor := damageFloaterColor(floater.kind, floater.color)
		terrainZ := terrainHeightAt(ctx.World, x, y)
		worldX := cellCenter(x) + dx
		worldY := cellCenter(y) + dy
		screenScale := actorBillboardScreenScale(projection, worldX, worldY, terrainZ)
		if floater.kind == damageFloaterMiss {
			if billboard, ok := m.damageMessageBillboard(ctx, 0, 0); ok {
				drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, 1, floaterColor)
				continue
			}
		}
		if floater.kind == damageFloaterCritical {
			if billboard, ok := m.damageMessageBillboard(ctx, 2, 0); ok {
				drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift+0.05, screenScale*scale*0.6, alpha, 1, color.RGBA{R: 168, G: 168, B: 168, A: 255})
			}
		}
		if billboard, ok := m.damageNumberBillboard(ctx, floater.text); ok {
			drawSpriteBillboardTintAlphaOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, 1, floaterColor)
			continue
		}
		point := projection.Project(worldX, worldY, terrainZ+zLift)
		debugTextColor(screen, withAlpha(floaterColor, alpha), int(point.x)-8, int(point.y)-40, "%s", floater.text)
	}
	m.damageFloaters = active
}

func clearWorldScene(screen *render.Image, mapName string) {
	screen.Fill(worldSceneClearColor(mapName))
}

func worldSceneClearColor(mapName string) color.RGBA {
	normalized := normalizeMapNameForSceneClear(mapName)
	for _, name := range []string{
		"yuno.rsw",
		"valkyrie.rsw",
		"rwc01.rsw",
		"himinn.rsw",
		"airplane.rsw",
		"airplane01.rsw",
		"schgld.rsw",
		"bat_fild02.rsw",
		"que_qsch01.rsw",
		"que_qsch02.rsw",
		"que_qsch03.rsw",
		"que_qsch04.rsw",
		"que_qsch05.rsw",
		"que_qaru01.rsw",
		"que_qaru02.rsw",
		"que_qaru03.rsw",
		"que_qaru04.rsw",
		"que_qaru05.rsw",
		"bat_b01.rsw",
		"bat_b02.rsw",
	} {
		if normalized == name {
			return color.RGBA{R: 0x99, G: 0xcc, B: 0xff, A: 255}
		}
	}
	for _, name := range []string{"gonryun.rsw", "gon_dun02.rsw", "ra_temsky.rsw", "que_temsky.rsw"} {
		if normalized == name {
			return color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}
		}
	}
	switch normalized {
	case "thana_boss.rsw":
		return color.RGBA{R: 0xe0, G: 0xd5, B: 0xc2, A: 255}
	case "5@tower.rsw", "5tower.rsw":
		return color.RGBA{R: 0x33, G: 0x00, B: 0x33, A: 255}
	default:
		return color.RGBA{A: 255}
	}
}

func normalizeMapNameForSceneClear(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if index := strings.LastIndexAny(name, `\/`); index >= 0 {
		name = name[index+1:]
	}
	switch {
	case strings.HasSuffix(name, ".gat"):
		return strings.TrimSuffix(name, ".gat") + ".rsw"
	case name != "" && !strings.Contains(name, "."):
		return name + ".rsw"
	default:
		return name
	}
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
		m.drawTileCursor(screen, ctx, projection, now)
		if ctx.World.RSW != nil && len(ctx.World.RSM) > 0 {
			actorOverlays = m.drawSceneModelsAndActors(screen, ctx, projection, vertexFog)
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
			drawLine(screen, float64(x), 0, float64(x), float64(height), render.ColorGrid)
		}
		for y := 0; y < height; y += tile {
			drawLine(screen, 0, float64(y), float64(width), float64(y), render.ColorGrid)
		}
	}

	m.drawSceneActorOverlays(screen, ctx, projection, now, actorOverlays)
	m.drawRSWEffects(screen, ctx, projection, now)
	m.drawWorldEffects(screen, ctx, projection, now)
	m.drawDamageFloaters(screen, ctx, projection, now)

	m.inventoryBag.Draw(screen, ctx, m)
	m.storageWindow.Draw(screen, ctx, m)
	m.shopWindow.Draw(screen, ctx, m)
	m.skillWindow.Draw(screen, ctx, m)
	m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
	m.deathModal.Draw(screen, ctx, width, height)
}

func (m *WorldMode) DrawOverlay(ctx client.Context, screen *render.Image) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	m.drawMapFade(screen, now)
	m.drawROCursor(screen, ctx, projection, now)
}

type followCamera struct {
	initialized bool
	x           float64
	y           float64
	z           float64
	yawOffset   float64
	zoom        float64
}

func (c *followCamera) Reset() {
	*c = followCamera{}
}

func (c *followCamera) Update(ctx client.Context, now time.Time) {
	targetX, targetY, targetZ := playerCameraTarget(ctx.World, now)
	if !c.initialized {
		c.x = targetX
		c.y = targetY
		c.z = targetZ
		c.initialized = true
		c.store(ctx)
		return
	}
	factor := cameraFollowFactor()
	c.x += (targetX - c.x) * factor
	c.y += (targetY - c.y) * factor
	c.z += (targetZ - c.z) * factor
	c.store(ctx)
}

func (c *followCamera) Rotate(delta float64) {
	c.yawOffset = normalizeCameraYaw(c.yawOffset + delta)
}

func (c *followCamera) ZoomBy(factor float64) {
	if factor <= 0 || !isFinite(factor) {
		return
	}
	c.zoom = clampCameraZoom(c.currentZoom() * factor)
}

func (c *followCamera) ZoomByDelta(delta float64) {
	if delta == 0 || !isFinite(delta) {
		return
	}
	c.zoom = clampCameraZoom(c.currentZoom() + delta)
}

func (c *followCamera) currentZoom() float64 {
	if c.zoom <= 0 || !isFinite(c.zoom) {
		return sceneCameraZoom()
	}
	return clampCameraZoom(c.zoom)
}

func (c *followCamera) ResetRotation() {
	c.yawOffset = 0
}

func (c *followCamera) Projection(ctx client.Context, width, height int, now time.Time) sceneProjection {
	return c.ProjectionWithOffset(ctx, width, height, now, 0, 0)
}

func (c *followCamera) ProjectionWithOffset(ctx client.Context, width, height int, now time.Time, offsetX, offsetY float64) sceneProjection {
	if !c.initialized {
		c.Update(ctx, now)
	}
	c.store(ctx)
	yawOffset := c.yawOffset
	if cameraRotationLockedForMap(ctx) {
		c.ResetRotation()
		yawOffset = 0
	}
	return newSceneProjectionForTargetYawZoom(width, height, c.x+offsetX, c.y+offsetY, c.z, cameraYawForMap(ctx)+yawOffset, cameraZoomForMap(ctx, c.currentZoom()))
}

func (c *followCamera) store(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	ctx.World.Camera.X = c.x
	ctx.World.Camera.Y = c.y
}

func (m *WorldMode) sceneProjection(ctx client.Context, width, height int, now time.Time) sceneProjection {
	offsetX, offsetY := m.cameraShakeOffset(now)
	return m.camera.ProjectionWithOffset(ctx, width, height, now, offsetX, offsetY)
}

func (m *WorldMode) startCameraShake(starts time.Time, duration time.Duration) {
	if duration <= 0 {
		return
	}
	m.cameraShakeStart = starts
	m.cameraShakeEnd = starts.Add(duration)
}

func (m *WorldMode) cameraShakeOffset(now time.Time) (float64, float64) {
	if m.cameraShakeStart.IsZero() || !now.Before(m.cameraShakeEnd) {
		return 0, 0
	}
	duration := m.cameraShakeEnd.Sub(m.cameraShakeStart)
	if duration <= 0 {
		return 0, 0
	}
	elapsed := now.Sub(m.cameraShakeStart)
	progress := clampFloat(float64(elapsed)/float64(duration), 0, 1)
	amplitude := 0.18 * (1 - progress)
	seconds := float64(elapsed) / float64(time.Second)
	return math.Sin(seconds*120*2*math.Pi) * amplitude, math.Cos(seconds*150*2*math.Pi) * amplitude
}

func (m *WorldMode) updateCameraRotation(ctx client.Context) {
	if cameraRotationLockedForMap(ctx) {
		m.camera.ResetRotation()
		return
	}
	delta := 0.0
	if ctx.Input.MousePressed(render.MouseButtonRight) {
		screenW, _ := ctx.ScreenSize()
		delta += cameraDragYawDelta(ctx.Input.MouseDX, screenW)
	}
	if delta != 0 {
		m.camera.Rotate(delta)
	}
}

func (m *WorldMode) updateCameraZoom(ctx client.Context) {
	if cameraZoomLockedForMap(ctx) {
		return
	}
	factor := 1.0
	if ctx.Input.WheelY != 0 {
		m.camera.ZoomByDelta(cameraWheelZoomDelta(ctx.Input.WheelY))
	}
	if ctx.Input.PinchDelta != 0 {
		factor *= cameraPinchZoomFactor(ctx.Input.PinchDelta)
	}
	if factor != 1 {
		m.camera.ZoomBy(factor)
	}
}

func playerCameraTarget(world *worldstate.World, now time.Time) (float64, float64, float64) {
	if world == nil {
		return 0.5, 0.5, 0
	}
	playerX, playerY := world.Player.RenderPosition(now)
	return cellCenter(playerX), cellCenter(playerY), cameraTargetHeightAt(world, playerX, playerY)
}

func cameraFollowFactor() float64 {
	return defaultCameraFollowFactor
}

func cameraYawForMap(ctx client.Context) float64 {
	if viewPoint, ok := lockedCameraViewPointForMap(ctx); ok {
		return -float64(viewPoint.InitialLongitude)
	}
	if cameraRotationLockedForMap(ctx) {
		return -45
	}
	return sceneCameraYaw()
}

func cameraRotationLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if _, ok := lockedCameraViewPointForMap(ctx); ok {
		return true
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func lockedCameraViewPointForMap(ctx client.Context) (res.CameraViewPoint, bool) {
	if ctx.Resources == nil || ctx.World == nil {
		return res.CameraViewPoint{}, false
	}
	viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName)
	if !ok || !viewPoint.LocksLongitude() {
		return res.CameraViewPoint{}, false
	}
	return viewPoint, true
}

func cameraZoomForMap(ctx client.Context, outdoorZoom float64) float64 {
	if ctx.Resources == nil || ctx.World == nil {
		return clampCameraZoom(outdoorZoom)
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok && viewPoint.DistanceScope <= 0 {
		if viewPoint.InitialDistance > 0 {
			return clampCameraZoom(float64(viewPoint.InitialDistance))
		}
		return sceneCameraZoom()
	}
	if ctx.Resources.IsIndoorMap(ctx.World.MapName) {
		return sceneCameraZoom()
	}
	return clampCameraZoom(outdoorZoom)
}

func cameraZoomLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok {
		return viewPoint.DistanceScope <= 0
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func cameraDragYawDelta(mouseDX, screenWidth int) float64 {
	if mouseDX == 0 || screenWidth <= 0 {
		return 0
	}
	return -(float64(mouseDX) / float64(screenWidth)) * 720
}

func cameraWheelZoomFactor(wheelY float64) float64 {
	if wheelY == 0 || !isFinite(wheelY) {
		return 1
	}
	return math.Pow(cameraZoomWheelStep(), -wheelY)
}

func cameraWheelZoomDelta(wheelY float64) float64 {
	if wheelY == 0 || !isFinite(wheelY) {
		return 0
	}
	return -wheelY * cameraZoomWheelUnits()
}

func cameraPinchZoomFactor(delta float64) float64 {
	if delta == 0 || !isFinite(delta) {
		return 1
	}
	return math.Exp(-delta / cameraPinchZoomScale())
}

func cameraZoomWheelStep() float64 {
	return defaultCameraWheelZoomStep
}

func cameraZoomWheelUnits() float64 {
	return defaultCameraWheelZoomUnits
}

func cameraPinchZoomScale() float64 {
	return defaultCameraPinchZoomScale
}

func clampCameraZoom(zoom float64) float64 {
	if !isFinite(zoom) || zoom <= 0 {
		zoom = defaultSceneCameraZoom
	}
	return math.Max(defaultCameraMinZoom, math.Min(defaultCameraMaxZoom, zoom))
}

func normalizeCameraYaw(yaw float64) float64 {
	yaw = math.Mod(yaw, 360)
	if yaw <= -180 {
		yaw += 360
	}
	if yaw > 180 {
		yaw -= 360
	}
	return yaw
}

func upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if isLocalActor(ctx, entry.ID) {
		return
	}
	dir := entry.Dir
	if entry.Moving {
		dir = directionFromDelta(entry.FromX, entry.FromY, entry.ToX, entry.ToY, dir)
	}
	ctx.World.UpsertActor(worldstate.Actor{
		ID:            entry.ID,
		X:             entry.X,
		Y:             entry.Y,
		Dir:           dir,
		Job:           entry.Job,
		Head:          entry.Head,
		Weapon:        entry.Weapon,
		Shield:        entry.Shield,
		HeadTop:       entry.HeadTop,
		HeadMid:       entry.HeadMid,
		HeadLow:       entry.HeadLow,
		Sex:           entry.Sex,
		HeadDir:       entry.HeadDir,
		Appearance:    entry.Appearance,
		Moving:        entry.Moving,
		FromX:         entry.FromX,
		FromY:         entry.FromY,
		ToX:           entry.ToX,
		ToY:           entry.ToY,
		ObjectType:    entry.ObjectType,
		HasObjectType: entry.HasObjectType,
		Speed:         entry.Speed,
	})
}

func (m *WorldMode) applyWarpPortalEntry(ctx client.Context, entry network.ActorEntry) {
	if !isWarpPortalJob(entry.Job) {
		return
	}
	m.addWorldEffectIfMissing(ctx, effectPortal, entry.ID)
}

func isWarpPortalJob(job int16) bool {
	return job == 128 || job == 129
}

func (m *WorldMode) applyActorVanish(ctx client.Context, vanish network.ActorVanish) {
	log.Printf("actor vanish id=%d reason=%d", vanish.ID, vanish.Reason)
	if vanish.Reason == 1 {
		m.startActorDeath(ctx, vanish.ID)
		return
	}
	ctx.World.RemoveActor(vanish.ID)
	delete(m.actorAnims, vanish.ID)
	delete(m.actorDeaths, vanish.ID)
	delete(m.actorSoundFrames, vanish.ID)
	delete(m.actorLife, vanish.ID)
}

func (m *WorldMode) startActorDeath(ctx client.Context, id uint32) {
	actor, ok, local := actorForCombatID(ctx, id)
	if !ok {
		if !local {
			ctx.World.RemoveActor(id)
		}
		return
	}
	now := time.Now()
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.WalkDistance = 0
	if local {
		ctx.World.Player.Moving = false
		m.deathModal.OpenDeath()
	} else {
		ctx.World.UpsertActor(actor)
	}
	actionFamily := deathActionFamilyForActor(actor)
	deathDuration := m.actorActionDuration(ctx, actor, actionFamily, defaultDeathAnimationDuration)
	visibleDuration := deathDuration
	if !local {
		visibleDuration = maxDuration(deathDuration, nonPCDeathFadeDuration)
	}
	if local {
		m.startHeldCombatAnimation(ctx, id, actionFamily, now, deathDuration)
	} else {
		m.startCombatAnimation(ctx, id, actionFamily, now, visibleDuration)
	}
	if !local {
		if m.actorDeaths == nil {
			m.actorDeaths = make(map[uint32]time.Time)
		}
		m.actorDeaths[id] = now.Add(visibleDuration)
		if m.actorLife != nil {
			if life, ok := m.actorLife[id]; ok {
				life.hp = 0
				life.updatedAt = now
				m.actorLife[id] = life
			}
		}
	}
	log.Printf("actor death id=%d job=%d local=%t action=%d death_ms=%d remove_ms=%d", id, actor.Job, local, actionFamily, deathDuration.Milliseconds(), visibleDuration.Milliseconds())
}

func (m *WorldMode) cleanupDeadActors(ctx client.Context, now time.Time) {
	if len(m.actorDeaths) == 0 || ctx.World == nil {
		return
	}
	for id, removeAt := range m.actorDeaths {
		if now.Before(removeAt) {
			continue
		}
		ctx.World.RemoveActor(id)
		delete(m.actorDeaths, id)
		delete(m.actorAnims, id)
		delete(m.actorSoundFrames, id)
		delete(m.actorLife, id)
		if m.pendingAttack.targetID == id {
			m.pendingAttack = attackIntent{}
		}
		if m.lockedAttackID == id {
			m.clearLockedAttack()
		}
		log.Printf("actor death removed id=%d", id)
	}
}

func (m *WorldMode) clearActorDeath(id uint32) {
	delete(m.actorDeaths, id)
	delete(m.actorAnims, id)
	delete(m.actorSoundFrames, id)
}

func (m *WorldMode) actorDeathAlpha(id uint32, now time.Time) float64 {
	removeAt, ok := m.actorDeaths[id]
	if !ok {
		return 1
	}
	started := now
	if anim, ok := m.actorAnims[id]; ok && !anim.started.IsZero() {
		started = anim.started
	}
	total := removeAt.Sub(started)
	if total <= 0 {
		return 0
	}
	elapsed := now.Sub(started)
	if elapsed <= 0 {
		return 1
	}
	alpha := 1 - float64(elapsed)/float64(total)
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func applyActorLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.ID == 0 {
		return false
	}
	if isLocalActor(ctx, look.ID) {
		applyCharacterLookChange(ctx.Session, look)
		applyWorldActorLookChange(&ctx.World.Player, look)
		return true
	}
	actor, ok := ctx.World.Actors[look.ID]
	if !ok {
		actor = worldstate.Actor{ID: look.ID, Appearance: true}
	}
	applyWorldActorLookChange(&actor, look)
	ctx.World.UpsertActor(actor)
	return false
}

func applyCharacterLookChange(sessionState *session.Session, look network.ActorLookChange) {
	update := func(character *session.Character) {
		switch look.Type {
		case 0:
			character.Job = int16(look.Value)
		case 1:
			character.Hair = int16(look.Value)
		case 2:
			weapon, shield := res.NormalizePlayerWeaponShield(int(look.Value&0xFFFF), int((look.Value>>16)&0xFFFF))
			character.Weapon = int16(weapon)
			character.Shield = int16(shield)
		case 3:
			character.HeadLow = int16(look.Value)
		case 4:
			character.HeadTop = int16(look.Value)
		case 5:
			character.HeadMid = int16(look.Value)
		case 6:
			character.HeadPal = int16(look.Value)
			if look.Value <= 255 {
				character.HairColor = uint8(look.Value)
			}
		case 7:
			character.BodyPal = int16(look.Value)
		case 8:
			character.Shield = int16(look.Value)
		}
	}
	update(&sessionState.Selected)
	for index := range sessionState.Characters {
		if sessionState.Characters[index].ID == sessionState.CharID || sessionState.Characters[index].ID == sessionState.Selected.ID {
			update(&sessionState.Characters[index])
		}
	}
}

func applyWorldActorLookChange(actor *worldstate.Actor, look network.ActorLookChange) {
	actor.Appearance = true
	switch look.Type {
	case 0:
		if actorHasMobObjectType(*actor) && res.HasPlayerJobToken(int(look.Value)) {
			log.Printf("ignored mob look-base player job id=%d old_job=%d value=%d", actor.ID, actor.Job, look.Value)
			return
		}
		actor.Job = int16(look.Value)
	case 1:
		actor.Head = int16(look.Value)
	case 2:
		weapon, shield := res.NormalizePlayerWeaponShield(int(look.Value&0xFFFF), int((look.Value>>16)&0xFFFF))
		actor.Weapon = int16(weapon)
		actor.Shield = int16(shield)
	case 3:
		actor.HeadLow = int16(look.Value)
	case 4:
		actor.HeadTop = int16(look.Value)
	case 5:
		actor.HeadMid = int16(look.Value)
	case 8:
		actor.Shield = int16(look.Value)
	}
}

func actorHasMobObjectType(actor worldstate.Actor) bool {
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeMob, actorObjectTypeNPCABR, actorObjectTypeNPCBionic:
		return true
	default:
		return false
	}
}

func applyActorDirectionChange(ctx client.Context, direction network.ActorDirectionChange) {
	if ctx.World == nil || direction.ID == 0 {
		return
	}
	dir := int(direction.Dir & 7)
	headDir := uint8(normalizeHeadDir(int(direction.HeadDir)))
	if isLocalActor(ctx, direction.ID) {
		ctx.World.Player.Dir = dir
		ctx.World.Player.HeadDir = headDir
		ctx.World.Dir = dir
		if ctx.Session != nil {
			ctx.Session.PlayerDir = dir
		}
		return
	}
	actor, ok := ctx.World.Actors[direction.ID]
	if !ok {
		return
	}
	actor.Dir = dir
	actor.HeadDir = headDir
	ctx.World.Actors[direction.ID] = actor
}

func isLocalActor(ctx client.Context, id uint32) bool {
	return ctx.Session != nil && id != 0 && (id == ctx.Session.AccountID || id == ctx.Session.CharID)
}

func applySelfMoveAck(ctx client.Context, ack network.SelfMoveAck) {
	dir := directionFromDelta(ack.FromX, ack.FromY, ack.ToX, ack.ToY, ctx.World.Dir)
	ctx.World.SetPlayerMovement(ack.FromX, ack.FromY, ack.ToX, ack.ToY, dir)
	ctx.Session.PlayerX = ack.ToX
	ctx.Session.PlayerY = ack.ToY
}

func applyMapAcceptEnter(ctx client.Context, enter network.MapAcceptEnter) {
	ctx.Session.PlayerX = enter.X
	ctx.Session.PlayerY = enter.Y
	ctx.Session.PlayerDir = enter.Dir
	ctx.Session.Playing = true
	ctx.World.SetPlayerPosition(enter.X, enter.Y, enter.Dir)
}

func applyWarpPosition(ctx client.Context, x, y int) {
	dir := ctx.World.Dir
	if ctx.Session.PlayerDir != 0 {
		dir = ctx.Session.PlayerDir
	}
	ctx.Session.PlayerX = x
	ctx.Session.PlayerY = y
	ctx.World.SetPlayerPosition(x, y, dir)
}

func applyActorSetPosition(ctx client.Context, position network.ActorSetPosition) {
	if isLocalActor(ctx, position.ID) {
		ctx.World.SetPlayerPosition(position.X, position.Y, ctx.World.Dir)
		ctx.Session.PlayerX = position.X
		ctx.Session.PlayerY = position.Y
		return
	}
	ctx.World.UpsertActor(worldstate.Actor{
		ID: position.ID,
		X:  position.X,
		Y:  position.Y,
	})
}

func applyActorNameAck(ctx client.Context, ack network.ActorNameAck) {
	name := sanitizeActorName(ack.Name)
	if name == "" || ctx.World == nil {
		return
	}
	if isLocalActor(ctx, ack.ID) {
		ctx.World.Player.Name = name
		return
	}
	actor, ok := ctx.World.Actors[ack.ID]
	if !ok {
		return
	}
	actor.Name = name
	ctx.World.Actors[ack.ID] = actor
}

func walkTargetInBounds(ctx client.Context, x, y int) bool {
	if x < 0 || y < 0 {
		return false
	}
	if ctx.World.GAT != nil {
		return x < ctx.World.GAT.Width && y < ctx.World.GAT.Height
	}
	if ctx.World.GND != nil {
		return x < ctx.World.GND.Width && y < ctx.World.GND.Height
	}
	return x <= 1023 && y <= 1023
}

func directionFromDelta(fromX, fromY, toX, toY int, fallback int) int {
	return worldstate.DirectionFromDelta(fromX, fromY, toX, toY, normalizeDirectionIndex(fallback))
}

func cameraTargetHeightAt(world *worldstate.World, x, y float64) float64 {
	return terrainHeightAt(world, x, y)
}

func clickedWalkTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
	minX, maxX, minY, maxY, ok := walkTargetSearchBounds(ctx)
	if !ok {
		return 0, 0, false
	}

	if x, y, ok := clickedWalkCellByProjectedPolygon(ctx, projection, mouseX, mouseY, minX, maxX, minY, maxY); ok {
		return x, y, true
	}

	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			point := projection.Project(cellCenter(float64(x)), cellCenter(float64(y)), terrainHeightAt(ctx.World, float64(x), float64(y)))
			dx := float64(point.x) - float64(mouseX)
			dy := float64(point.y) - float64(mouseY)
			distance := dx*dx + dy*dy
			if distance < bestDistance {
				bestDistance = distance
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, bestDistance < math.Inf(1)
}

func hoveredWalkCell(ctx client.Context, projection sceneProjection, mouseX, mouseY int) (int, int, bool) {
	minX, maxX, minY, maxY, ok := walkTargetSearchBounds(ctx)
	if !ok {
		return 0, 0, false
	}
	return clickedWalkCellByProjectedPolygon(ctx, projection, mouseX, mouseY, minX, maxX, minY, maxY)
}

func walkTargetSearchBounds(ctx client.Context) (int, int, int, int, bool) {
	if ctx.World == nil {
		return 0, 0, 0, 0, false
	}
	radius := clickWalkSearchRadius()
	minX := maxInt(0, ctx.World.Player.X-radius)
	maxX := ctx.World.Player.X + radius
	minY := maxInt(0, ctx.World.Player.Y-radius)
	maxY := ctx.World.Player.Y + radius
	if ctx.World.GAT != nil {
		maxX = minInt(maxX, ctx.World.GAT.Width-1)
		maxY = minInt(maxY, ctx.World.GAT.Height-1)
	} else if ctx.World.GND != nil {
		maxX = minInt(maxX, ctx.World.GND.Width*2-1)
		maxY = minInt(maxY, ctx.World.GND.Height*2-1)
	}
	return minX, maxX, minY, maxY, minX <= maxX && minY <= maxY
}

func clickedWalkCellByProjectedPolygon(ctx client.Context, projection sceneProjection, mouseX, mouseY, minX, maxX, minY, maxY int) (int, int, bool) {
	if ctx.World == nil || ctx.World.GAT == nil {
		return 0, 0, false
	}
	bestX, bestY := 0, 0
	bestDepth := math.Inf(1)
	found := false
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !ctx.World.GAT.Walkable(x, y) {
				continue
			}
			points, depth, ok := projectedGATCell(projection, ctx.World.GAT, x, y)
			if !ok {
				continue
			}
			if !pointInProjectedGATCell(float64(mouseX), float64(mouseY), points) {
				continue
			}
			if !found || depth < bestDepth {
				found = true
				bestDepth = depth
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, found
}

func (m *WorldMode) drawTileCursor(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil || ctx.World == nil || ctx.World.GAT == nil {
		return
	}
	x, y, ok := hoveredWalkCell(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return
	}
	verts, ok := tileCursorCellVerts(ctx.World.GAT, x, y, now)
	if !ok {
		return
	}
	points := projectTileCursorVerts(projection, verts)
	if quadHasInvalidPoint(points) || quadOutside(points, float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())) {
		return
	}
	drawTileCursorSurface3D(screen, m.tileCursorTexture(), verts)
}

func tileCursorCellVerts(gat *res.GAT, x, y int, now time.Time) ([4]modelPoint3, bool) {
	if gat == nil {
		return [4]modelPoint3{}, false
	}
	cell, ok := gat.Cell(x, y)
	if !ok {
		return [4]modelPoint3{}, false
	}
	lift := tileCursorLift(now)
	return [4]modelPoint3{
		{x: float64(x), y: float64(cell.Heights[0]) + lift, z: float64(y)},
		{x: float64(x + 1), y: float64(cell.Heights[1]) + lift, z: float64(y)},
		{x: float64(x), y: float64(cell.Heights[2]) + lift, z: float64(y + 1)},
		{x: float64(x + 1), y: float64(cell.Heights[3]) + lift, z: float64(y + 1)},
	}, true
}

func projectTileCursorVerts(projection sceneProjection, verts [4]modelPoint3) [4]screenPoint {
	return [4]screenPoint{
		projection.Project(verts[0].x, verts[0].z, verts[0].y),
		projection.Project(verts[1].x, verts[1].z, verts[1].y),
		projection.Project(verts[2].x, verts[2].z, verts[2].y),
		projection.Project(verts[3].x, verts[3].z, verts[3].y),
	}
}

func tileCursorLift(now time.Time) float64 {
	seconds := float64(now.UnixNano()) / float64(time.Second)
	return 0.018 + 0.006*math.Sin(seconds*math.Pi*2/1.2)
}

func (m *WorldMode) tileCursorTexture() *render.Image {
	if m.tileCursor != nil {
		return m.tileCursor
	}
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dist := minInt(minInt(x, y), minInt(size-1-x, size-1-y))
			alpha := uint8(0)
			switch {
			case dist < 3:
				alpha = 190
			case dist < 6:
				alpha = 100
			case dist < 11:
				alpha = 32
			}
			if x == y || x == size-1-y {
				alpha = maxUint8(alpha, 34)
			}
			if alpha > 0 {
				img.SetRGBA(x, y, color.RGBA{R: 180, G: 230, B: 255, A: alpha})
			}
		}
	}
	m.tileCursor = render.NewImageFromImage(img)
	return m.tileCursor
}

func drawTileCursorSurface3D(screen, texture *render.Image, verts [4]modelPoint3) {
	if texture == nil {
		return
	}
	tints := [4]color.RGBA{
		{R: 255, G: 255, B: 255, A: 210},
		{R: 255, G: 255, B: 255, A: 210},
		{R: 255, G: 255, B: 255, A: 210},
		{R: 255, G: 255, B: 255, A: 210},
	}
	uvs := [4]texturePoint{
		{u: 0, v: 0},
		{u: 1, v: 0},
		{u: 0, v: 1},
		{u: 1, v: 1},
	}
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012213, tints, triangleDrawOptions(render.FilterLinear, render.AddressClampToZero))
}

func projectedGATCell(projection sceneProjection, gat *res.GAT, x, y int) ([4]screenPoint, float64, bool) {
	cell, ok := gat.Cell(x, y)
	if !ok {
		return [4]screenPoint{}, 0, false
	}
	verts := [4]modelPoint3{
		{x: float64(x), y: float64(cell.Heights[0]), z: float64(y)},
		{x: float64(x + 1), y: float64(cell.Heights[1]), z: float64(y)},
		{x: float64(x), y: float64(cell.Heights[2]), z: float64(y + 1)},
		{x: float64(x + 1), y: float64(cell.Heights[3]), z: float64(y + 1)},
	}
	points := [4]screenPoint{
		projection.Project(verts[0].x, verts[0].z, verts[0].y),
		projection.Project(verts[1].x, verts[1].z, verts[1].y),
		projection.Project(verts[2].x, verts[2].z, verts[2].y),
		projection.Project(verts[3].x, verts[3].z, verts[3].y),
	}
	depth := math.Inf(1)
	for _, vert := range verts {
		depth = math.Min(depth, projection.Depth(vert.x, vert.z, vert.y))
	}
	return points, depth, true
}

func pointInProjectedGATCell(x, y float64, points [4]screenPoint) bool {
	return pointInScreenTriangle(x, y, points[0], points[1], points[2]) ||
		pointInScreenTriangle(x, y, points[2], points[1], points[3])
}

func pointInScreenTriangle(x, y float64, a, b, c screenPoint) bool {
	d1 := screenTriangleSign(x, y, a, b)
	d2 := screenTriangleSign(x, y, b, c)
	d3 := screenTriangleSign(x, y, c, a)
	hasNegative := d1 < 0 || d2 < 0 || d3 < 0
	hasPositive := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNegative && hasPositive)
}

func screenTriangleSign(x, y float64, a, b screenPoint) float64 {
	return (x-float64(b.x))*(float64(a.y)-float64(b.y)) - (float64(a.x)-float64(b.x))*(y-float64(b.y))
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

func actorCanBeSkillTargeted(ctx client.Context, skill session.Skill, actor worldstate.Actor) bool {
	if actor.ID == 0 || isWarpActor(actor) {
		return false
	}
	targetFlags, ok := skillTargetFlagsForActor(ctx, actor)
	if !ok {
		return false
	}
	if skill.Type&targetFlags != 0 {
		if isLocalActor(ctx, actor.ID) && skill.Type&skillTargetEnemy != 0 {
			return false
		}
		return true
	}
	if skillTargetOverrideActive(ctx) || skillTargetMapStateAllowsMismatch(ctx, actor) {
		if isLocalActor(ctx, actor.ID) && skill.Type&skillTargetEnemy != 0 {
			return false
		}
		return true
	}
	return false
}

func skillTargetFlagsIncludeSelfPick(flags uint32) bool {
	return flags&(skillTargetFriend|skillTargetSelf) != 0
}

func skillTargetFlagsForActor(ctx client.Context, actor worldstate.Actor) (uint32, bool) {
	if actor.ID == 0 || isWarpActor(actor) {
		return 0, false
	}
	if isLocalActor(ctx, actor.ID) {
		return skillTargetFriend, true
	}
	if actor.HasObjectType {
		switch actor.ObjectType {
		case actorObjectTypePC, actorObjectTypeElemental:
			return skillTargetFriend, true
		case actorObjectTypeHomunculus, actorObjectTypeMercenary:
			return skillTargetFriend | skillTargetHomun, true
		case actorObjectTypeMob, actorObjectTypeUnit, actorObjectTypeNPCABR, actorObjectTypeNPCBionic:
			return skillTargetEnemy | skillTargetPet, true
		default:
			return 0, false
		}
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return skillTargetFriend, true
	}
	return 0, false
}

func skillTargetMapStateAllowsMismatch(ctx client.Context, actor worldstate.Actor) bool {
	// roBrowser allows target-type mismatches on PvP/GvG maps. Goro does not yet
	// parse map state packets, so keep the rule isolated until that state exists.
	return false
}

func actorCanBeAttackClicked(ctx client.Context, actor worldstate.Actor) bool {
	if isLocalActor(ctx, actor.ID) {
		return false
	}
	if actor.ID == 0 || !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeMob, actorObjectTypeNPCABR, actorObjectTypeNPCBionic:
		return true
	default:
		return false
	}
}

func pointInActorPickBounds(mouseX, mouseY, centerX, centerY, scale float64) bool {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	left := centerX - 44*scale
	right := centerX + 44*scale
	top := centerY - float64(humanoidBillboardAnchorY)*scale
	bottom := centerY + 20*scale
	return mouseX >= left && mouseX <= right && mouseY >= top && mouseY <= bottom
}

func clickWalkSearchRadius() int {
	return 70
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

type sceneActorDrawEntry struct {
	actor       worldstate.Actor
	screenX     float64
	screenY     float64
	worldX      float64
	worldY      float64
	worldZ      float64
	scale       float64
	shadow      float64
	castShadow  bool
	shadowX     float64
	shadowY     float64
	shadowScale float64
	shadowDepth float64
	depth       float64
	isPlayer    bool
	hidden      bool
}

const (
	actorBillboardCellWorldUnits  = 5.0
	actorBillboardWorldHeightUnit = 1.0 * actorBillboardCellWorldUnits
	actorJobWarpPortal            = 45
	actorJobClearNPC              = 844
	actorObjectTypePC             = 0
	actorObjectTypeMob            = 5
	actorObjectTypeNPC            = 6
	actorObjectTypeHomunculus     = 8
	actorObjectTypeMercenary      = 9
	actorObjectTypeElemental      = 10
	actorObjectTypeUnit           = 11
	actorObjectTypeNPC2           = 12
	actorObjectTypeNPCABR         = 13
	actorObjectTypeNPCBionic      = 14
)

var monsterShadowSize = map[int]float64{
	111:  0.0,
	139:  0.0,
	1004: 0.5,
	1005: 0.5,
	1007: 0.5,
	1008: 0.3,
	1009: 0.7,
	1011: 0.5,
	1013: 1.2,
	1018: 0.7,
	1019: 1.2,
	1020: 0.0,
	1025: 0.0,
	1030: 0.0,
	1035: 0.5,
	1037: 0.0,
	1039: 1.2,
	1040: 2.0,
	1042: 0.5,
	1046: 0.0,
	1047: 0.2,
	1048: 0.2,
	1049: 0.3,
	1050: 0.3,
	1051: 0.3,
	1056: 0.7,
	1057: 0.7,
	1061: 1.5,
	1063: 0.5,
	1069: 1.2,
	1070: 0.3,
	1072: 0.5,
	1074: 0.5,
	1078: 0.0,
	1079: 0.0,
	1080: 0.0,
	1081: 0.0,
	1082: 0.0,
	1083: 0.0,
	1084: 0.0,
	1085: 0.0,
	1087: 1.2,
	1089: 1.5,
	1090: 1.0,
	1091: 0.5,
	1092: 1.2,
	1094: 0.7,
	1095: 0.5,
	1097: 0.2,
	1098: 2.0,
	1101: 0.5,
	1102: 1.2,
	1103: 0.3,
	1104: 0.7,
	1105: 0.7,
	1106: 1.2,
	1107: 0.7,
	1108: 0.7,
	1109: 0.7,
	1110: 0.7,
	1111: 0.5,
	1114: 0.5,
	1115: 1.2,
	1121: 0.7,
	1127: 0.0,
	1129: 0.5,
	1131: 0.0,
	1138: 0.0,
	1139: 0.5,
	1140: 1.2,
	1141: 0.5,
	1142: 0.5,
	1143: 0.5,
	1145: 0.5,
	1147: 1.5,
	1149: 1.5,
	1155: 0.5,
	1156: 0.5,
	1158: 0.7,
	1159: 1.2,
	1160: 0.7,
	1161: 0.5,
	1162: 0.5,
	1167: 0.5,
	1170: 0.7,
	1174: 0.5,
	1175: 0.5,
	1176: 0.7,
	1182: 0.0,
	1183: 0.5,
	1184: 0.5,
	1186: 2.0,
	1190: 1.2,
	1192: 1.5,
	1193: 2.0,
	1194: 0.5,
	1195: 0.5,
	1199: 0.5,
	1201: 1.2,
	1202: 1.5,
	1203: 0.5,
	1204: 0.5,
	1208: 1.2,
	1209: 0.7,
	1211: 0.5,
	1214: 0.7,
	1219: 5.0,
}

func (m *WorldMode) drawSceneActors(screen *render.Image, ctx client.Context, projection sceneProjection) []sceneActorDrawEntry {
	entries := m.collectSceneActorEntries(screen, ctx, projection)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].depth > entries[j].depth
	})
	for _, entry := range entries {
		m.drawActorShadowEntry(screen, projection, entry)
	}
	for _, entry := range entries {
		m.drawSceneActorEntry(screen, ctx, projection, entry)
	}
	return entries
}

type sceneDrawEntry struct {
	depth       float64
	actorIndex  int
	shadowIndex int
	itemIndex   int
}

func (m *WorldMode) drawSceneModelsAndActors(screen *render.Image, ctx client.Context, projection sceneProjection, fog sceneFog) []sceneActorDrawEntry {
	m.drawRSMModels(screen, ctx.Resources, ctx.World.RSW, ctx.World.RSM, ctx.World.GND, projection, fog)
	actors := m.collectSceneActorEntries(screen, ctx, projection)
	items := m.collectSceneItemEntries(screen, ctx, projection, time.Now())
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

func (m *WorldMode) drawSceneActorOverlays(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time, entries []sceneActorDrawEntry) {
	for _, entry := range entries {
		m.drawActorLifeBar(screen, ctx, entry)
	}
	m.drawHoveredLocalPlayerNameLabel(screen, ctx, entries)
	m.drawHoveredActorNameLabel(screen, ctx, projection, now)
}

func (m *WorldMode) drawHoveredLocalPlayerNameLabel(screen *render.Image, ctx client.Context, entries []sceneActorDrawEntry) {
	if ctx.Input == nil {
		return
	}
	for _, entry := range entries {
		if !entry.isPlayer {
			continue
		}
		if !pointInActorPickBounds(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), entry.screenX, entry.screenY, entry.scale) {
			return
		}
		labelY := actorNameLabelY(entry.screenY, entry.scale)
		if life, ok := m.actorLifeForDisplay(ctx, entry.actor); ok {
			labelY = actorNameBelowLifeBarY(entry.screenY, entry.scale, life)
		}
		drawActorNameLabelAtY(screen, actorDisplayName(ctx, entry.actor, true), entry.screenX, labelY, actorNameLabelColor(entry.actor, true))
		return
	}
}

func (m *WorldMode) collectSceneActorEntries(screen *render.Image, ctx client.Context, projection sceneProjection) []sceneActorDrawEntry {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	entries := make([]sceneActorDrawEntry, 0, len(ctx.World.Actors)+1)
	player := ctx.World.Player
	player.ID = ctx.Session.CharID
	character := selectedCharacter(ctx.Session)
	player.Job = character.Job
	player.Head = character.Hair
	player.Sex = ctx.Session.Sex
	if character.Name != "" {
		player.Name = character.Name
	}
	player.Dir = ctx.World.Dir
	entries = appendActorDrawEntry(entries, ctx.World, projection, player, true, now, width, height)
	entries[len(entries)-1].hidden = localActorHidden(ctx)
	for _, actor := range ctx.World.Actors {
		if actor.ID == ctx.Session.AccountID || actor.ID == ctx.Session.CharID {
			continue
		}
		entries = appendActorDrawEntry(entries, ctx.World, projection, actor, false, now, width, height)
	}
	return entries
}

func (m *WorldMode) drawSceneActorEntry(screen *render.Image, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry) {
	cameraYaw := projection.cameraYaw
	alpha := 1.0
	if entry.hidden {
		alpha = 0.35
	}
	if entry.isPlayer {
		if m.drawPlayerSprite3D(ctx, screen, projection, entry, entry.actor.Dir, cameraYaw, entry.shadow, alpha) {
			return
		}
		drawPanel(screen, entry.screenX-6, entry.screenY-6, 24, 24)
		return
	}
	if visual := specialNPCVisualForActor(ctx, entry.actor); visual != specialNPCVisualNone {
		if m.drawSpecialNPCVisual(screen, ctx, projection, entry, visual, time.Now()) {
			return
		}
	}
	if isWarpActor(entry.actor) {
		if m.whitePixel == nil {
			m.whitePixel = render.NewImage(1, 1)
			m.whitePixel.Fill(color.White)
		}
		drawWarpZoneEffect(screen, m.whitePixel, m.effectTexture(ctx.Resources, "ring_blue"), entry.worldX, entry.worldY, entry.worldZ, time.Now())
		return
	}
	if m.drawActorSprite3D(screen, ctx, projection, entry, cameraYaw, entry.shadow) {
		return
	}
	drawActorMarker(screen, entry.screenX-6, entry.screenY-20, entry.actor, time.Now())
}

func (m *WorldMode) drawActorShadowEntry(screen *render.Image, projection sceneProjection, entry sceneActorDrawEntry) {
	if !entry.castShadow || m.shadowView == nil || m.shadowViewMiss {
		return
	}
	if entry.hidden {
		return
	}
	now := time.Now()
	if m.actorShadowSuppressed(entry.actor, now) {
		return
	}
	scale := entry.scale * entry.shadowScale
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return
	}
	drawFixedSpriteBillboardAlphaFlat3D(screen, projection, m.shadowView, entry.worldX, entry.worldY, entry.worldZ+0.03, scale, m.actorDeathAlpha(entry.actor.ID, now), entry.shadow)
}

func appendActorDrawEntry(entries []sceneActorDrawEntry, world *worldstate.World, projection sceneProjection, actor worldstate.Actor, isPlayer bool, now time.Time, screenWidth, screenHeight int) []sceneActorDrawEntry {
	actorX, actorY := actor.RenderPosition(now)
	actor.Dir = actor.RenderDirection(now)
	terrainZ := terrainHeightAt(world, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	worldX := cellCenter(actorX)
	worldY := cellCenter(actorY)
	scale := actorBillboardScreenScale(projection, worldX, worldY, terrainZ)
	if actorAnchorOutsideViewport(float64(point.x), float64(point.y), screenWidth, screenHeight, scale) {
		return entries
	}
	depth := actorBillboardSortDepth(projection, worldX, worldY, terrainZ)
	shadowDepth := projection.Depth(worldX, worldY, terrainZ+0.05)
	shadowPoint := projection.Project(worldX, worldY, terrainZ+0.05)
	return append(entries, sceneActorDrawEntry{
		actor:       actor,
		screenX:     float64(point.x),
		screenY:     float64(point.y),
		worldX:      worldX,
		worldY:      worldY,
		worldZ:      terrainZ,
		scale:       scale,
		shadow:      actorShadowFactor(world, actorX, actorY),
		castShadow:  actorCastsShadow(actor),
		shadowX:     float64(shadowPoint.x),
		shadowY:     float64(shadowPoint.y),
		shadowScale: actorShadowSize(actor),
		shadowDepth: shadowDepth,
		depth:       depth,
		isPlayer:    isPlayer,
	})
}

func actorAnchorOutsideViewport(anchorX, anchorY float64, screenWidth, screenHeight int, scale float64) bool {
	left, right, top, bottom := actorViewportCullMargins(scale)
	return anchorX < -left || anchorX > float64(screenWidth)+right || anchorY < -top || anchorY > float64(screenHeight)+bottom
}

func actorViewportCullMargins(scale float64) (left, right, top, bottom float64) {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	side := math.Max(128, float64(humanoidBillboardWidth)*scale)
	// Most entity pixels are above the feet/anchor. The lower screen edge needs
	// a larger margin so tall sprites do not disappear while their body is still
	// visible and only their feet are outside the viewport.
	topMargin := math.Max(96, float64(humanoidBillboardHeight-humanoidBillboardAnchorY)*scale*2)
	bottomMargin := math.Max(192, float64(humanoidBillboardAnchorY)*scale*1.6)
	return side, side, topMargin, bottomMargin
}

func actorCastsShadow(actor worldstate.Actor) bool {
	if isWarpActor(actor) || actorJobHasSpecialNoShadow(int(actor.Job)) {
		return false
	}
	return actorShadowSize(actor) > 0
}

func (m *WorldMode) actorShadowSuppressed(actor worldstate.Actor, now time.Time) bool {
	if actor.Sitting {
		return true
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		switch anim.actionFamily {
		case spriteActionSit, spriteActionPCDeath, spriteActionNonPCDeath:
			return true
		}
	}
	return false
}

func actorShadowSize(actor worldstate.Actor) float64 {
	if size, ok := monsterShadowSize[int(actor.Job)]; ok {
		return size
	}
	return 1
}

func actorShadowFactor(world *worldstate.World, x, y float64) float64 {
	if world == nil || world.GND == nil {
		return 1
	}
	shadowX, shadowY := gndShadowMapPoint(x, y)
	total := 0
	for dy := -3; dy < 3; dy++ {
		for dx := -3; dx < 3; dx++ {
			total += int(gndShadowMapAlpha(world.GND, shadowX+dx, shadowY+dy))
		}
	}
	return clampUnit(float64(total) / (6 * 6 * 255))
}

func gndShadowMapPoint(x, y float64) (int, int) {
	x += 0.5
	y += 0.5
	shadowX := int(math.Floor(x/2)) * 8
	shadowY := int(math.Floor(y/2)) * 8
	localX := 0
	if int(x)&1 != 0 {
		localX = 4
	}
	localY := 0
	if int(y)&1 != 0 {
		localY = 4
	}
	localX += int(math.Floor((x - math.Floor(x)) * 4))
	localY += int(math.Floor((y - math.Floor(y)) * 4))
	shadowX += minInt(localX, 6)
	shadowY += minInt(localY, 6)
	return shadowX, shadowY
}

func gndShadowMapAlpha(gnd *res.GND, shadowX, shadowY int) uint8 {
	if gnd == nil || shadowX < 0 || shadowY < 0 || shadowX >= gnd.Width*8 || shadowY >= gnd.Height*8 {
		return 255
	}
	cellX := shadowX / 8
	cellY := shadowY / 8
	localX := shadowX % 8
	localY := shadowY % 8
	cell, ok := gnd.Cell(cellX, cellY)
	if !ok || cell.Top < 0 {
		return 255
	}
	surface, ok := gnd.Surface(cell.Top)
	if !ok {
		return 255
	}
	lightmap, ok := gnd.Lightmap(surface.LightmapID)
	if !ok {
		return 255
	}
	return lightmap.Alpha[localY][localX]
}

func actorBillboardSortDepth(projection sceneProjection, x, y, z float64) float64 {
	footDepth := projection.Depth(x, y, z)
	topDepth := projection.Depth(x, y, z+actorBillboardWorldHeightUnit)
	if topDepth <= 0 || !isFinite(topDepth) {
		return footDepth
	}
	return math.Min(footDepth, topDepth)
}

func actorDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if isPlayer {
		if name := sanitizeActorName(selectedCharacterName(ctx.Session)); name != "" {
			return name
		}
		return sanitizeActorName(actor.Name)
	}
	if isWarpActor(actor) {
		return ""
	}
	if name := sanitizeActorName(actor.Name); name != "" {
		return name
	}
	if res.HasPlayerJobToken(int(actor.Job)) || ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.JobResourceName(int(actor.Job)); ok {
		return displayNameFromResource(resourceName)
	}
	return ""
}

func (m *WorldMode) drawHoveredActorNameLabel(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil || ctx.World == nil {
		return
	}
	if _, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
		return
	}
	actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok || isWarpActor(actor) {
		return
	}
	label := m.hoveredActorDisplayName(ctx, actor, now)
	if label == "" {
		return
	}
	actorX, actorY := actor.RenderPosition(now)
	terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
	drawActorNameLabel(screen, label, float64(point.x), float64(point.y), scale, actorNameLabelColor(actor, isLocalActor(ctx, actor.ID)))
}

func (m *WorldMode) hoveredActorDisplayName(ctx client.Context, actor worldstate.Actor, now time.Time) string {
	if isLocalActor(ctx, actor.ID) {
		return actorDisplayName(ctx, actor, true)
	}
	if name := sanitizeActorName(actor.Name); name != "" {
		return name
	}
	if shouldUseServerNameForHoverActor(actor) {
		m.requestActorName(ctx, actor.ID, now)
		if name := actorResourceDisplayName(ctx, actor); name != "" {
			return name
		}
		if res.HasPlayerJobToken(int(actor.Job)) {
			return "Player"
		}
		if isMonsterLikeHoverActor(actor) {
			return "Monster"
		}
		return "NPC"
	}
	if name := actorResourceDisplayName(ctx, actor); name != "" {
		return name
	}
	return "Entity"
}

func actorResourceDisplayName(ctx client.Context, actor worldstate.Actor) string {
	if ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.JobResourceName(int(actor.Job)); ok {
		return displayNameFromResource(resourceName)
	}
	return ""
}

func (m *WorldMode) requestActorName(ctx client.Context, id uint32, now time.Time) {
	if id == 0 || ctx.Network == nil || isLocalActor(ctx, id) {
		return
	}
	if m.actorNameReqAt == nil {
		m.actorNameReqAt = make(map[uint32]time.Time)
	}
	if previous, ok := m.actorNameReqAt[id]; ok && now.Sub(previous) < actorNameRequestCooldown {
		return
	}
	if err := ctx.Network.SendNameRequest(id); err != nil {
		log.Printf("send name request failed id=%d: %v", id, err)
		return
	}
	m.actorNameReqAt[id] = now
}

func shouldUseServerNameForHoverActor(actor worldstate.Actor) bool {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return true
	}
	if isMonsterLikeHoverActor(actor) {
		return true
	}
	return actor.HasObjectType && actor.ObjectType == actorObjectTypeNPC
}

func isMonsterLikeHoverActor(actor worldstate.Actor) bool {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	job := int(actor.Job)
	return job >= 1000 && (job < 6001 || job > 6047)
}

func selectedCharacterName(s *session.Session) string {
	if s == nil {
		return ""
	}
	return selectedCharacter(s).Name
}

func sanitizeActorName(name string) string {
	name = strings.TrimSpace(name)
	if hash := strings.IndexByte(name, '#'); hash >= 0 {
		name = strings.TrimSpace(name[:hash])
	}
	if strings.EqualFold(name, "actor") {
		return ""
	}
	return name
}

func displayNameFromResource(name string) string {
	name = strings.TrimSpace(strings.TrimSuffix(name, ".spr"))
	name = strings.TrimSuffix(name, ".act")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ToLower(strings.TrimSpace(name))
	fields := strings.Fields(name)
	for i, field := range fields {
		fields[i] = titleASCIIWord(field)
	}
	return strings.Join(fields, " ")
}

func isWarpActor(actor worldstate.Actor) bool {
	return actor.Job == actorJobWarpPortal
}

func titleASCIIWord(word string) string {
	if word == "" {
		return ""
	}
	if word[0] < 'a' || word[0] > 'z' {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func actorNameLabelColor(actor worldstate.Actor, isPlayer bool) color.RGBA {
	if isPlayer {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	switch {
	case actor.HasObjectType && actor.ObjectType == actorObjectTypeNPC:
		return color.RGBA{R: 148, G: 189, B: 247, A: 255}
	case isMonsterLikeHoverActor(actor):
		return color.RGBA{R: 255, G: 198, B: 198, A: 255}
	default:
		return color.RGBA{R: 248, G: 248, B: 248, A: 255}
	}
}

func drawActorNameLabel(screen *render.Image, label string, centerX, baseY, scale float64, foreground color.RGBA) {
	drawActorNameLabelAtY(screen, label, centerX, actorNameLabelY(baseY, scale), foreground)
}

func drawActorNameLabelAtY(screen *render.Image, label string, centerX, labelY float64, foreground color.RGBA) {
	label = sanitizeActorName(label)
	if label == "" {
		return
	}
	outline := color.RGBA{A: 196}
	text := render.OutlinedTextImage(label, foreground, outline)
	if text == nil {
		return
	}
	x := int(math.Round(centerX)) - text.Bounds().Dx()/2
	y := int(math.Round(labelY))
	render.DrawOutlinedTextAt(screen, label, x, y, foreground, outline)
}

func actorNameLabelY(baseY, scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	return baseY + 13*scale
}

func actorLifeBarY(baseY, scale float64) float64 {
	return actorNameLabelY(baseY, scale) + 14
}

func actorLifeBarHeight(life actorLife) float64 {
	if life.hasSP {
		return 9
	}
	return 5
}

func actorNameBelowLifeBarY(baseY, scale float64, life actorLife) float64 {
	return actorLifeBarY(baseY, scale) + actorLifeBarHeight(life) + 3
}

func (m *WorldMode) drawActorLifeBar(screen *render.Image, ctx client.Context, entry sceneActorDrawEntry) {
	life, ok := m.actorLifeForDisplay(ctx, entry.actor)
	if !ok {
		return
	}
	ratio := float64(life.hp) / float64(life.maxHP)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	const width = 60.0
	height := actorLifeBarHeight(life)
	x := math.Round(entry.screenX - width/2)
	y := math.Round(actorLifeBarY(entry.screenY, entry.scale))
	fillWidth := math.Round((width - 2) * ratio)
	fill := color.RGBA{R: 255, G: 0, B: 231, A: 255}
	if life.player {
		fill = gameui.PlayerHPBarColor
		if ratio < 0.25 {
			fill = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		}
	} else if ratio < 0.25 {
		fill = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
	render.DrawRect(screen, x, y, width, height, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, width-2, height-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		render.DrawRect(screen, x+1, y+1, fillWidth, 3, fill)
	}
	if life.hasSP {
		spRatio := float64(life.sp) / float64(life.maxSP)
		if spRatio < 0 {
			spRatio = 0
		} else if spRatio > 1 {
			spRatio = 1
		}
		render.DrawRect(screen, x, y+4, width, 1, color.RGBA{R: 16, G: 24, B: 156, A: 255})
		if spWidth := math.Round((width - 2) * spRatio); spWidth > 0 {
			render.DrawRect(screen, x+1, y+5, spWidth, 3, gameui.PlayerSPBarColor)
		}
	}
}

func (m *WorldMode) actorLifeForDisplay(ctx client.Context, actor worldstate.Actor) (actorLife, bool) {
	if actor.ID == 0 {
		return actorLife{}, false
	}
	if specialNPCVisualForActor(ctx, actor) != specialNPCVisualNone {
		return actorLife{}, false
	}
	if isLocalActor(ctx, actor.ID) {
		return localPlayerLifeForDisplay(ctx)
	}
	// Monster HP bars are a 2012+ client feature. The 2008 client exposes
	// monster HP through WZ_ESTIMATION/Sense instead, so keep the combat HP
	// cache hidden from the normal actor overlay.
	return actorLife{}, false
}

func (m *WorldMode) monsterLifeForSense(actorID uint32) (actorLife, bool) {
	if m.actorLife == nil {
		return actorLife{}, false
	}
	life, ok := m.actorLife[actorID]
	if !ok || life.maxHP <= 0 || life.hp < 0 {
		return actorLife{}, false
	}
	return life, true
}

func localPlayerLifeForDisplay(ctx client.Context) (actorLife, bool) {
	if ctx.Session == nil {
		return actorLife{}, false
	}
	hp := ctx.Session.Vitals.HP
	maxHP := ctx.Session.Vitals.MaxHP
	sp := ctx.Session.Vitals.SP
	maxSP := ctx.Session.Vitals.MaxSP
	if maxHP <= 0 {
		character := selectedCharacter(ctx.Session)
		hp = int(character.HP)
		maxHP = int(character.MaxHP)
	}
	if maxSP <= 0 {
		character := selectedCharacter(ctx.Session)
		sp = int(character.SP)
		maxSP = int(character.MaxSP)
	}
	if maxHP <= 0 {
		return actorLife{}, false
	}
	return actorLife{
		hp:     clampGameInt(hp, 0, maxHP),
		maxHP:  maxHP,
		sp:     clampGameInt(sp, 0, maxSP),
		maxSP:  maxSP,
		hasSP:  maxSP > 0,
		player: true,
	}, true
}

func (m *WorldMode) drawActorSprite3D(screen *render.Image, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.actor
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return m.drawNonPCSprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
	}
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := actorSpriteKey{
		job:     int(actor.Job),
		head:    int(actor.Head),
		sex:     actor.Sex,
		weapon:  weapon,
		shield:  shield,
		headTop: int(actor.HeadTop),
		headMid: int(actor.HeadMid),
		headLow: int(actor.HeadLow),
	}
	if _, ok := m.actorViewMiss[key]; ok {
		return false
	}
	view, ok := m.actorViews[key]
	if !ok {
		loaded, status := loadHumanoidSpriteViewWithAppearance(ctx.Resources, humanoidAppearance{
			job:     key.job,
			head:    key.head,
			sex:     key.sex,
			weapon:  key.weapon,
			shield:  key.shield,
			headTop: key.headTop,
			headMid: key.headMid,
			headLow: key.headLow,
		}, "actor")
		if loaded == nil {
			m.actorViewMiss[key] = struct{}{}
			log.Printf("actor sprite unavailable id=%d job=%d head=%d sex=%d: %s", actor.ID, key.job, key.head, key.sex, status)
			return false
		}
		m.actorViews[key] = loaded
		view = loaded
		log.Printf("actor sprite resources id=%d job=%d head=%d sex=%d %s", actor.ID, key.job, key.head, key.sex, status)
	}
	now := time.Now()
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    actor.Dir,
		headDir:      actor.HeadDir,
		headTurn:     true,
		cameraYaw:    cameraYaw,
		moving:       actor.IsMovingAt(now),
		moveSpeedMS:  actor.Speed,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actor.RenderWalkDistance(now)
	} else if actor.Sitting {
		state.actionFamily = spriteActionSit
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		state.actionFamily = anim.actionFamily
		state.started = anim.started
		state.loop = false
		state.moving = false
		state.fixedMotion = anim.fixedMotion
		state.hasFixedMotion = anim.hasFixedMotion
	}
	billboard, ok := humanoidBillboardForState(view, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, 1, shadow)
	return true
}

func (m *WorldMode) drawNonPCSprite3D(screen *render.Image, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.actor
	view := m.nonPCSpriteView(ctx, actor)
	if view == nil {
		return false
	}
	now := time.Now()
	state := m.nonPCSpriteState(actor, now)
	state.cameraYaw = cameraYaw
	billboard, ok := singleSpriteBillboardForState(view, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, m.actorDeathAlpha(actor.ID, now), shadow)
	return true
}

func (m *WorldMode) nonPCSpriteState(actor worldstate.Actor, now time.Time) spriteState {
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    actor.Dir,
		moving:       actor.IsMovingAt(now),
		loopIdle:     true,
		moveSpeedMS:  actor.Speed,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actor.RenderWalkDistance(now)
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		state.actionFamily = anim.actionFamily
		state.started = anim.started
		state.loop = false
		state.moving = false
		state.loopIdle = false
		state.fixedMotion = anim.fixedMotion
		state.hasFixedMotion = anim.hasFixedMotion
	}
	return state
}

func (m *WorldMode) nonPCSpriteView(ctx client.Context, actor worldstate.Actor) *playerSpriteView {
	job := int(actor.Job)
	if _, ok := m.nonPCViewMiss[job]; ok {
		return nil
	}
	if m.nonPCViews == nil {
		m.nonPCViews = make(map[int]*playerSpriteView)
	}
	view, ok := m.nonPCViews[job]
	if ok {
		return view
	}
	if ctx.Resources == nil {
		return nil
	}
	loaded, status := loadNonPCSpriteView(ctx.Resources, job, "nonpc")
	if loaded == nil {
		if m.nonPCViewMiss == nil {
			m.nonPCViewMiss = make(map[int]struct{})
		}
		m.nonPCViewMiss[job] = struct{}{}
		log.Printf("nonpc sprite unavailable id=%d job=%d: %s", actor.ID, job, status)
		return nil
	}
	m.nonPCViews[job] = loaded
	log.Printf("nonpc sprite resources id=%d job=%d %s", actor.ID, job, status)
	return loaded
}

func actorBillboardScreenScale(projection sceneProjection, x, y, z float64) float64 {
	base := projection.Project(x, y, z)
	top := projection.Project(x, y, z+actorBillboardWorldHeightUnit)
	projectedHeight := math.Hypot(float64(top.x-base.x), float64(top.y-base.y))
	if projectedHeight <= 0 || math.IsNaN(projectedHeight) || math.IsInf(projectedHeight, 0) {
		return 1
	}
	return projectedHeight / float64(humanoidBillboardAnchorY)
}

func drawWarpZoneEffect(screen, white, ringTexture *render.Image, x, y, z float64, now time.Time) {
	const (
		segments       = 64
		ringCount      = 4
		baseRadius     = 0.25
		radiusRange    = 1.18
		bandWidth      = 0.34
		cycleSeconds   = 4.0
		bottomBaseSize = 0.95
		topBaseSize    = 1.58
		heightBase     = 1.10
		groundLift     = 0.04
	)
	z += groundLift
	seconds := float64(now.UnixNano()) / float64(time.Second)

	for i := 0; i < ringCount; i++ {
		phase := math.Mod(seconds+float64(i), cycleSeconds) / cycleSeconds
		sizeFactor := 1 - phase
		heightFactor := phase * 2
		if phase > 0.5 {
			heightFactor = (1 - phase) * 2
		}
		alpha := uint8(102 * warpCycleFade(phase))
		drawWorldCylinderBand(screen, white, ringTexture, x, y, z, bottomBaseSize*sizeFactor, topBaseSize*sizeFactor, heightBase*heightFactor, color.RGBA{R: 155, G: 205, B: 255, A: alpha}, segments)
	}
	drawWorldRadialGradient(screen, white, x, y, z, 0.18, 0.85, color.RGBA{R: 170, G: 210, B: 255, A: 54}, segments)
	for i := 0; i < ringCount; i++ {
		phase := math.Mod(seconds*0.55+float64(i)/ringCount, 1)
		radius := baseRadius + phase*radiusRange
		alpha := uint8(155 * (1 - phase))
		if alpha < 28 {
			alpha = 28
		}
		drawWorldSoftRing(screen, white, x, y, z, radius, bandWidth, color.RGBA{R: 185, G: 215, B: 255, A: alpha}, segments)
	}
	pulse := 0.5 + 0.5*math.Sin(seconds*2.4)
	drawWorldSoftRing(screen, white, x, y, z, 0.35+pulse*0.06, 0.26, color.RGBA{R: 235, G: 245, B: 255, A: 150}, segments)
}

func warpCycleFade(phase float64) float64 {
	switch {
	case phase < 0.25:
		return phase / 0.25
	case phase > 0.75:
		return (1 - phase) / 0.25
	default:
		return 1
	}
}

func drawWorldRadialGradient(screen, white *render.Image, x, y, z, innerRadius, outerRadius float64, c color.RGBA, segments int) {
	drawWorldRingBand(screen, white, x, y, z, innerRadius, outerRadius, c.A, 0, c, segments)
}

func drawWorldSoftRing(screen, white *render.Image, x, y, z, radius, width float64, c color.RGBA, segments int) {
	inner := math.Max(0, radius-width*0.5)
	mid := math.Max(inner+0.01, radius)
	outer := math.Max(mid+0.01, radius+width*0.5)
	drawWorldRingBand(screen, white, x, y, z, inner, mid, 0, c.A, c, segments)
	drawWorldRingBand(screen, white, x, y, z, mid, outer, c.A, 0, c, segments)
}

func drawWorldCylinderBand(screen, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int) {
	if segments < 3 || bottomRadius <= 0.01 || topRadius <= 0.01 || height <= 0.01 || c.A == 0 {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	tint := c
	srcW, srcH := float32(1), float32(1)
	source := white
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	for i := 0; i <= segments; i++ {
		u := float32(i) / float32(segments)
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		vertices = append(vertices,
			warpEffectTexturedVertex3D(x+cosine*bottomRadius, y+sine*bottomRadius, z, u*srcW, srcH, tint),
			warpEffectTexturedVertex3D(x+cosine*topRadius, y+sine*topRadius, z+height, u*srcW, 0, tint),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, source, options)
}

func drawWorldRingBand(screen, white *render.Image, x, y, z, innerRadius, outerRadius float64, innerAlpha, outerAlpha uint8, c color.RGBA, segments int) {
	if segments < 3 || outerRadius <= innerRadius {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	innerColor := c
	outerColor := c
	innerColor.A = innerAlpha
	outerColor.A = outerAlpha
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		vertices = append(vertices,
			warpEffectVertex3D(x+cosine*innerRadius, y+sine*innerRadius, z, innerColor),
			warpEffectVertex3D(x+cosine*outerRadius, y+sine*outerRadius, z, outerColor),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := triangleDrawOptions(render.FilterNearest, render.AddressUnsafe)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, white, options)
}

func warpEffectTexturedVertex3D(x, y, z float64, srcX, srcY float32, c color.RGBA) render.Vertex3D {
	point := modelPoint3{x: x, y: z, z: y}
	return render.Vertex3D{
		X:      float32(point.x),
		Y:      float32(point.y),
		Z:      float32(point.z),
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: float32(c.R) / 255,
		ColorG: float32(c.G) / 255,
		ColorB: float32(c.B) / 255,
		ColorA: float32(c.A) / 255,
		DepthX: float32(point.x),
		DepthY: float32(point.y),
		DepthZ: float32(point.z),
	}
}

func warpEffectVertex3D(x, y, z float64, c color.RGBA) render.Vertex3D {
	return warpEffectTexturedVertex3D(x, y, z, 0, 0, c)
}

func drawActorMarker(screen *render.Image, x, y float64, actor worldstate.Actor, now time.Time) {
	col := color.RGBA{R: 82, G: 166, B: 255, A: 230}
	if actor.Job >= 1000 {
		col = color.RGBA{R: 229, G: 102, B: 72, A: 230}
	}
	if actor.IsMovingAt(now) {
		col = color.RGBA{R: 235, G: 190, B: 80, A: 230}
	}
	render.DrawRect(screen, x, y, 12, 18, col)
	render.DrawRect(screen, x+3, y-4, 6, 6, col)
	debugText(screen, int(x-12), int(y-16), "%d", actor.Job)
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

func loadGND(manager *res.Manager, mapName string) (*res.GND, string, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".gnd",
		"data/" + base + ".gnd",
		base + ".gnd",
	}
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err != nil {
			continue
		}
		gnd, err := res.ParseGND(data)
		if err != nil {
			return nil, candidate, err
		}
		return gnd, candidate, nil
	}
	return nil, "", fmt.Errorf("gnd not found for map %s", mapName)
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

func (m *WorldMode) drawGNDWater(screen *render.Image, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection sceneProjection, now time.Time, fog sceneFog) {
	width := screen.Bounds().Dx()
	height := screen.Bounds().Dy()
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, width, height)
	if !ok {
		return
	}
	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok || cell.Top < 0 {
				continue
			}
			if waterDraw, ok := newGNDWaterDraw(projection, x, y, cell, gnd, rsw, now); ok {
				m.drawWaterSurface(screen, manager, waterDraw, projection, fog)
			}
		}
	}
}

func gndDrawBounds(gnd *res.GND, projection sceneProjection, screenWidth, screenHeight int) (int, int, int, int, bool) {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 {
		return 0, 0, 0, 0, false
	}

	centerX := gndTileFromWorld(projection.playerX)
	centerY := gndTileFromWorld(projection.playerY)
	if minWorldX, maxWorldX, minWorldY, maxWorldY, ok := cameraGroundFootprint(projection, screenWidth, screenHeight); ok {
		const margin = 24
		startX := minInt(gndTileFromWorld(minWorldX), centerX) - margin
		endX := maxInt(gndTileFromWorld(maxWorldX), centerX) + margin
		startY := minInt(gndTileFromWorld(minWorldY), centerY) - margin
		endY := maxInt(gndTileFromWorld(maxWorldY), centerY) + margin
		return clampGNDRange(gnd, startX, endX, startY, endY)
	}
	const fallbackRadius = 96
	return clampGNDRange(gnd, centerX-fallbackRadius, centerX+fallbackRadius, centerY-fallbackRadius, centerY+fallbackRadius)
}

func gndTileFromWorld(coord float64) int {
	return int(math.Floor(coord * 0.5))
}

func clampGNDRange(gnd *res.GND, startX, endX, startY, endY int) (int, int, int, int, bool) {
	startX = maxInt(0, startX)
	endX = minInt(gnd.Width-1, endX)
	startY = maxInt(0, startY)
	endY = minInt(gnd.Height-1, endY)
	return startX, endX, startY, endY, startX <= endX && startY <= endY
}

func cameraGroundFootprint(projection sceneProjection, screenWidth, screenHeight int) (float64, float64, float64, float64, bool) {
	aspect := 1.0
	if screenHeight > 0 {
		aspect = float64(screenWidth) / float64(screenHeight)
	}

	distance := normalizeSceneCameraZoom(projection.cameraZoom) * 0.5
	pitch := sceneCameraPitch()
	if pitch > 180 {
		pitch -= 180
	}
	pitch = degreesToRadians(pitch)
	yaw := degreesToRadians(projection.cameraYaw)
	horizontal := math.Cos(pitch) * distance
	eye := modelPoint3{
		x: projection.playerX + math.Sin(yaw)*horizontal,
		y: projection.playerZ + math.Sin(pitch)*distance,
		z: projection.playerY - math.Cos(yaw)*horizontal,
	}
	target := modelPoint3{x: projection.playerX, y: projection.playerZ, z: projection.playerY}
	forward := normalize3(sub3(target, eye))
	right := normalize3(cross3(modelPoint3{y: 1}, forward))
	if right == (modelPoint3{}) {
		right = modelPoint3{x: 1}
	}
	up := cross3(forward, right)
	tanHalfFOV := math.Tan(degreesToRadians(sceneCameraFOV()) * 0.5)

	samples := [][2]float64{
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
		{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}
	minX, maxX := 0.0, 0.0
	minY, maxY := 0.0, 0.0
	found := false
	for _, sample := range samples {
		dir := normalize3(add3(add3(forward, mul3(right, sample[0]*tanHalfFOV*aspect)), mul3(up, sample[1]*tanHalfFOV)))
		if math.Abs(dir.y) < 0.000001 {
			continue
		}
		t := (projection.playerZ - eye.y) / dir.y
		if t <= 0 || !isFinite(t) {
			continue
		}
		hit := add3(eye, mul3(dir, t))
		if !isFinite(hit.x) || !isFinite(hit.z) {
			continue
		}
		if !found {
			minX, maxX = hit.x, hit.x
			minY, maxY = hit.z, hit.z
			found = true
			continue
		}
		minX = math.Min(minX, hit.x)
		maxX = math.Max(maxX, hit.x)
		minY = math.Min(minY, hit.z)
		maxY = math.Max(maxY, hit.z)
	}
	return minX, maxX, minY, maxY, found
}

type gndSurfaceDraw struct {
	points      [4]screenPoint
	verts       [4]modelPoint3
	uvs         [4]texturePoint
	vertexOrder [4]int
	indices     []uint16
	surface     res.GNDSurface
	baseTints   [4]color.RGBA
	heights     [4]float32
	vertexNorms [4]modelPoint3
	lighting    sceneLighting
	water       bool
	waterType   int
	waterFrame  int
	tint        color.RGBA
}

func (m *WorldMode) smoothGNDTopNormals(gnd *res.GND) [][4]modelPoint3 {
	if gnd == nil {
		return nil
	}
	if m.gndNormalSource == gnd && len(m.gndTopNormals) == gnd.Width*gnd.Height {
		return m.gndTopNormals
	}
	m.gndNormalSource = gnd
	m.gndTopNormals = buildSmoothGNDTopNormals(gnd)
	return m.gndTopNormals
}

func buildSmoothGNDTopNormals(gnd *res.GND) [][4]modelPoint3 {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 {
		return nil
	}
	count := gnd.Width * gnd.Height
	cellNormals := make([]modelPoint3, count)
	for y := 0; y < gnd.Height; y++ {
		for x := 0; x < gnd.Width; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok || cell.Top < 0 {
				continue
			}
			verts := [4]modelPoint3{
				{x: float64(x) * 2, y: float64(cell.Heights[0]), z: float64(y) * 2},
				{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
				{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
				{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
			}
			cellNormals[x+y*gnd.Width] = quadNormal(verts)
		}
	}

	normals := make([][4]modelPoint3, count)
	for y := 0; y < gnd.Height; y++ {
		for x := 0; x < gnd.Width; x++ {
			cellNormal := gndCellNormalAt(cellNormals, gnd.Width, gnd.Height, x, y)
			if cellNormal == (modelPoint3{}) {
				cellNormal = modelPoint3{y: -1}
			}
			normals[x+y*gnd.Width] = [4]modelPoint3{
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x - 1, y}, {x - 1, y - 1}, {x, y - 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x + 1, y}, {x + 1, y - 1}, {x, y - 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x + 1, y}, {x + 1, y + 1}, {x, y + 1}}),
				smoothGNDNormalAt(cellNormals, gnd.Width, gnd.Height, cellNormal, [][2]int{{x, y}, {x - 1, y}, {x - 1, y + 1}, {x, y + 1}}),
			}
		}
	}
	return normals
}

func gndCellNormalAt(normals []modelPoint3, width, height, x, y int) modelPoint3 {
	if x < 0 || y < 0 || x >= width || y >= height {
		return modelPoint3{}
	}
	return normals[x+y*width]
}

func smoothGNDNormalAt(normals []modelPoint3, width, height int, fallback modelPoint3, samples [][2]int) modelPoint3 {
	var sum modelPoint3
	for _, sample := range samples {
		normal := gndCellNormalAt(normals, width, height, sample[0], sample[1])
		if normal == (modelPoint3{}) {
			continue
		}
		sum = add3(sum, normal)
	}
	if sum == (modelPoint3{}) {
		return fallback
	}
	return normalize3(sum)
}

func gndTopNormalsAt(normals [][4]modelPoint3, gnd *res.GND, x, y int) [4]modelPoint3 {
	if gnd == nil || x < 0 || y < 0 || x >= gnd.Width || y >= gnd.Height {
		return [4]modelPoint3{}
	}
	index := x + y*gnd.Width
	if index < 0 || index >= len(normals) {
		return [4]modelPoint3{}
	}
	return normals[index]
}

func topGNDSurfaceBaseTints(gnd *res.GND, x, y int, fallback color.RGBA) [4]color.RGBA {
	return [4]color.RGBA{
		gndTopTileBaseTintAt(gnd, x, y, fallback),
		gndTopTileBaseTintAt(gnd, x+1, y, fallback),
		gndTopTileBaseTintAt(gnd, x+1, y+1, fallback),
		gndTopTileBaseTintAt(gnd, x, y+1, fallback),
	}
}

func gndTopTileBaseTintAt(gnd *res.GND, x, y int, fallback color.RGBA) color.RGBA {
	if gnd != nil {
		if cell, ok := gnd.Cell(x, y); ok && cell.Top >= 0 {
			if surface, ok := gnd.Surface(cell.Top); ok {
				return gndSurfaceBaseTint(surface.Color)
			}
		}
	}
	return gndSurfaceBaseTint(fallback)
}

func uniformGNDSurfaceBaseTints(surfaceColor color.RGBA) [4]color.RGBA {
	tint := gndSurfaceBaseTint(surfaceColor)
	return [4]color.RGBA{tint, tint, tint, tint}
}

func gndSurfaceBaseTint(surfaceColor color.RGBA) color.RGBA {
	if surfaceColor.A == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.RGBA{R: surfaceColor.R, G: surfaceColor.G, B: surfaceColor.B, A: 255}
}

func uniformGNDNormals(normal modelPoint3) [4]modelPoint3 {
	normal = normalize3(normal)
	return [4]modelPoint3{normal, normal, normal, normal}
}

func newGNDWaterDraw(projection sceneProjection, x, y int, cell res.GNDCell, gnd *res.GND, rsw *res.RSW, now time.Time) (gndSurfaceDraw, bool) {
	water, ok := mapWater(gnd, rsw)
	if !ok {
		return gndSurfaceDraw{}, false
	}
	if !waterVisibleForCell(cell, water) {
		return gndSurfaceDraw{}, false
	}
	heights := waterHeightsForCell(water, x, y, now)
	verts := [4]modelPoint3{
		{x: float64(x) * 2, y: float64(heights[0]), z: float64(y) * 2},
		{x: float64(x+1) * 2, y: float64(heights[1]), z: float64(y) * 2},
		{x: float64(x+1) * 2, y: float64(heights[3]), z: float64(y+1) * 2},
		{x: float64(x) * 2, y: float64(heights[2]), z: float64(y+1) * 2},
	}
	draw := newGNDSurfaceDraw(
		projection,
		verts,
		waterUVs(x, y),
		[4]int{0, 1, 3, 2},
		quadIndices012023,
		res.GNDSurface{},
		heights,
		sceneLighting{},
	)
	draw.water = true
	draw.waterType = int(water.Type)
	draw.waterFrame = waterFrameForTime(water, now)
	draw.tint = waterTint(water, rsw)
	return draw, true
}

func mapWater(gnd *res.GND, rsw *res.RSW) (res.RSWWater, bool) {
	if gnd != nil && gnd.Water.Present {
		return res.RSWWater{
			Level:      gnd.Water.Level,
			Type:       gnd.Water.Type,
			WaveHeight: gnd.Water.WaveHeight,
			WaveSpeed:  gnd.Water.WaveSpeed,
			WavePitch:  gnd.Water.WavePitch,
			AnimSpeed:  gnd.Water.AnimSpeed,
		}, true
	}
	if rsw != nil {
		return rsw.Water, true
	}
	return res.RSWWater{}, false
}

func newGNDSurfaceDraw(projection sceneProjection, verts [4]modelPoint3, uvs [4]texturePoint, vertexOrder [4]int, indices []uint16, surface res.GNDSurface, heights [4]float32, lighting sceneLighting) gndSurfaceDraw {
	return newGNDSurfaceDrawWithNormals(projection, verts, uvs, vertexOrder, indices, surface, uniformGNDSurfaceBaseTints(surface.Color), heights, [4]modelPoint3{}, lighting)
}

func newGNDSurfaceDrawWithNormals(projection sceneProjection, verts [4]modelPoint3, uvs [4]texturePoint, vertexOrder [4]int, indices []uint16, surface res.GNDSurface, baseTints [4]color.RGBA, heights [4]float32, vertexNormals [4]modelPoint3, lighting sceneLighting) gndSurfaceDraw {
	normal := quadNormal(verts)
	for i := range vertexNormals {
		if vertexNormals[i] == (modelPoint3{}) {
			vertexNormals[i] = normal
		} else {
			vertexNormals[i] = normalize3(vertexNormals[i])
		}
	}
	return gndSurfaceDraw{
		points:      projectGNDQuad(projection, verts),
		verts:       verts,
		uvs:         uvs,
		vertexOrder: vertexOrder,
		indices:     indices,
		surface:     surface,
		baseTints:   baseTints,
		heights:     heights,
		vertexNorms: vertexNormals,
		lighting:    lighting,
	}
}

func (m *WorldMode) drawWaterSurface(screen *render.Image, manager *res.Manager, draw gndSurfaceDraw, projection sceneProjection, fog sceneFog) {
	texture := m.waterTexture(manager, draw.waterType, draw.waterFrame)
	tints := fog.mixVertexTints(projection, draw.verts, [4]color.RGBA{draw.tint, draw.tint, draw.tint, draw.tint})
	if texture == nil {
		drawColoredSurfaceTints3DAlpha(screen, m.whitePixel, draw.verts, draw.indices, tints)
		return
	}
	drawTexturedSurface3DAlpha(screen, texture, draw.verts, draw.uvs, draw.indices, tints)
}

func (m *WorldMode) waterTexture(manager *res.Manager, waterType, frame int) *render.Image {
	frame = ((frame % 32) + 32) % 32
	key := fmt.Sprintf("__water_%d_%02d", waterType, frame)
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.WaterTextureCandidates(waterType, frame))
	if err != nil && waterType >= 0 {
		img, _, err = res.LoadImage(manager, res.WaterTextureCandidates(waterType%6, frame))
	}
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) effectTexture(manager *res.Manager, name string) *render.Image {
	if manager == nil || strings.TrimSpace(name) == "" {
		return nil
	}
	key := "__effect_" + strings.TrimSpace(name)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.EffectTextureCandidates(name))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(res.ApplyEffectTransparency(img))
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) groundTexture(manager *res.Manager, name string) *render.Image {
	if name == "" {
		return nil
	}
	if texture, ok := m.textures[name]; ok {
		return texture
	}
	if _, ok := m.textureMiss[name]; ok {
		return nil
	}

	img, _, err := res.LoadImage(manager, res.GroundTextureCandidates(name))
	if err != nil {
		m.textureMiss[name] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[name] = texture
	return texture
}

type screenPoint struct {
	x float32
	y float32
}

type texturePoint struct {
	u float32
	v float32
}

func quadOutside(points [4]screenPoint, width, height float64) bool {
	minX, minY := float64(points[0].x), float64(points[0].y)
	maxX, maxY := minX, minY
	for _, point := range points[1:] {
		minX = math.Min(minX, float64(point.x))
		minY = math.Min(minY, float64(point.y))
		maxX = math.Max(maxX, float64(point.x))
		maxY = math.Max(maxY, float64(point.y))
	}
	return maxX < -32 || maxY < -32 || minX > width+32 || minY > height+32
}

func quadHasInvalidPoint(points [4]screenPoint) bool {
	for _, point := range points {
		if !isFinite(float64(point.x)) || !isFinite(float64(point.y)) {
			return true
		}
		if point.x <= -1<<19 && point.y <= -1<<19 {
			return true
		}
	}
	return false
}

func gndTextureName(gnd *res.GND, textureID int) string {
	if gnd == nil || textureID < 0 || textureID >= len(gnd.Textures) {
		return ""
	}
	return gnd.Textures[textureID]
}

func surfaceUVs(surface res.GNDSurface, order [4]int) [4]texturePoint {
	return [4]texturePoint{
		{u: surface.U[order[0]], v: surface.V[order[0]]},
		{u: surface.U[order[1]], v: surface.V[order[1]]},
		{u: surface.U[order[2]], v: surface.V[order[2]]},
		{u: surface.U[order[3]], v: surface.V[order[3]]},
	}
}

func waterUVs(x, y int) [4]texturePoint {
	const scale = 0.25
	baseU := float32(x&3) * scale
	baseV := float32(y&3) * scale
	return [4]texturePoint{
		{u: baseU, v: baseV},
		{u: baseU + scale, v: baseV},
		{u: baseU + scale, v: baseV + scale},
		{u: baseU, v: baseV + scale},
	}
}

func waterHeightsForCell(water res.RSWWater, x, y int, now time.Time) [4]float32 {
	level := water.Level
	if water.WaveHeight == 0 {
		return [4]float32{level, level, level, level}
	}
	offset := waterOffsetForTime(water, now)
	pitch := float64(water.WavePitch)
	diagonal := float64(x + y)
	h1 := waterSin(offset+pitch*diagonal)*water.WaveHeight + level
	h0 := waterSin(offset+pitch*(diagonal-1))*water.WaveHeight + level
	h3 := waterSin(offset+pitch*(diagonal+1))*water.WaveHeight + level
	return [4]float32{h0, h1, h1, h3}
}

func waterVisibleForCell(cell res.GNDCell, water res.RSWWater) bool {
	threshold := water.Level + water.WaveHeight
	return cell.Heights[0] < threshold ||
		cell.Heights[1] < threshold ||
		cell.Heights[2] < threshold ||
		cell.Heights[3] < threshold
}

func waterFrameForTime(water res.RSWWater, now time.Time) int {
	animSpeed := int(water.AnimSpeed)
	if animSpeed <= 0 {
		animSpeed = 1
	}
	frame := int(now.UnixMilli()*60/1000) / animSpeed
	return frame % 32
}

func waterOffsetForTime(water res.RSWWater, now time.Time) float64 {
	offset := math.Mod(float64(now.UnixMilli()*60/1000)*float64(water.WaveSpeed), 360)
	if offset > 180 {
		offset -= 360
	}
	return offset
}

func waterSin(degrees float64) float32 {
	return float32(math.Sin(degreesToRadians(degrees)))
}

func waterTint(water res.RSWWater, rsw *res.RSW) color.RGBA {
	alpha := uint8(204)
	if water.Type == 4 || water.Type == 6 {
		alpha = 255
	}
	if rsw != nil && water.Type == 4 {
		return color.RGBA{
			R: clampColor(float64(rsw.Light.Ambient[0]) * 255),
			G: clampColor(float64(rsw.Light.Ambient[1]) * 255),
			B: clampColor(float64(rsw.Light.Ambient[2]) * 255),
			A: alpha,
		}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: alpha}
}

func projectGNDQuad(projection sceneProjection, verts [4]modelPoint3) [4]screenPoint {
	return [4]screenPoint{
		projection.Project(verts[0].x, verts[0].z, verts[0].y),
		projection.Project(verts[1].x, verts[1].z, verts[1].y),
		projection.Project(verts[2].x, verts[2].z, verts[2].y),
		projection.Project(verts[3].x, verts[3].z, verts[3].y),
	}
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
		x: -math.Sin(longitude) * math.Sin(latitude),
		y: -math.Cos(latitude),
		z: -math.Cos(longitude) * math.Sin(latitude),
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
	weight := math.Max(dot3(normalize3(normal), l.direction), 0)
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

func surfaceTint(base color.RGBA, height float32, normal modelPoint3, lighting sceneLighting) color.RGBA {
	scale := lighting.groundScale(normal)
	heightShade := groundHeightShadeAt(height)
	return color.RGBA{
		R: clampColor(float64(base.R) * scale.x * heightShade),
		G: clampColor(float64(base.G) * scale.y * heightShade),
		B: clampColor(float64(base.B) * scale.z * heightShade),
		A: 255,
	}
}

func surfaceVertexTints(baseTints [4]color.RGBA, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting) [4]color.RGBA {
	return [4]color.RGBA{
		surfaceTint(baseTints[0], heights[0], normals[0], lighting),
		surfaceTint(baseTints[1], heights[1], normals[1], lighting),
		surfaceTint(baseTints[2], heights[2], normals[2], lighting),
		surfaceTint(baseTints[3], heights[3], normals[3], lighting),
	}
}

func scaleSurfaceVertexTints(baseTints [4]color.RGBA, lightScales [4]modelPoint3) [4]color.RGBA {
	var tints [4]color.RGBA
	for i := range tints {
		lightScale := lightScales[i]
		base := baseTints[i]
		tints[i] = color.RGBA{
			R: clampColor(float64(base.R) * lightScale.x),
			G: clampColor(float64(base.G) * lightScale.y),
			B: clampColor(float64(base.B) * lightScale.z),
			A: base.A,
		}
	}
	return tints
}

func vertexLightScales(lighting sceneLighting, normals [4]modelPoint3) [4]modelPoint3 {
	return [4]modelPoint3{
		lighting.groundScale(normals[0]),
		lighting.groundScale(normals[1]),
		lighting.groundScale(normals[2]),
		lighting.groundScale(normals[3]),
	}
}

func groundSurfaceVertexColors(textureName string, surfaceColor color.RGBA, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting) [4]color.RGBA {
	base := textureColor(textureName)
	if surfaceColor.A != 0 && !(surfaceColor.R == 255 && surfaceColor.G == 255 && surfaceColor.B == 255) {
		base.R = uint8((uint16(base.R)*2 + uint16(surfaceColor.R)) / 3)
		base.G = uint8((uint16(base.G)*2 + uint16(surfaceColor.G)) / 3)
		base.B = uint8((uint16(base.B)*2 + uint16(surfaceColor.B)) / 3)
	}
	var colors [4]color.RGBA
	for i := range colors {
		scale := lighting.groundScale(normals[i])
		heightShade := groundHeightShadeAt(heights[i])
		colors[i] = color.RGBA{
			R: clampColor(float64(base.R) * scale.x * heightShade),
			G: clampColor(float64(base.G) * scale.y * heightShade),
			B: clampColor(float64(base.B) * scale.z * heightShade),
			A: 255,
		}
	}
	return colors
}

func groundHeightShadeAt(height float32) float64 {
	return 0.88 + math.Sin(float64(height)*0.08)*0.06
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

func maxUint8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
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

func posterizeGNDLightmapColor(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: posterizeGNDLightmapChannel(c.R),
		G: posterizeGNDLightmapChannel(c.G),
		B: posterizeGNDLightmapChannel(c.B),
		A: c.A,
	}
}

func posterizeGNDLightmapChannel(v uint8) uint8 {
	return (v / 16) * 16
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
