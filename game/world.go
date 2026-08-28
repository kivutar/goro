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
	falconViews       map[int]*spriteView
	falconViewMiss    map[int]struct{}
	falcons           map[uint32]*falconRenderState
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
	cursorLevelNums   map[string]*spriteBillboard
	damageMsgView     *spriteView
	damageMsgMiss     bool
	itemMarker        *render.Image
	itemViews         map[itemSpriteKey]*spriteView
	itemViewMiss      map[itemSpriteKey]struct{}
	effectViews       map[string]*spriteView
	effectViewMiss    map[string]struct{}
	actorViews        map[actorSpriteKey]*humanoidSpriteView
	actorViewMiss     map[actorSpriteKey]struct{}
	mercenaryViews    map[actorSpriteKey]*humanoidSpriteView
	mercenaryViewMiss map[actorSpriteKey]struct{}
	nonPCViews        map[int]*spriteView
	nonPCViewMiss     map[int]struct{}
	gr2Models         map[int]*gr2ModelView
	gr2ModelMiss      map[int]struct{}
	petAccessoryIDs   map[uint32]uint32
	petAccessoryViews map[petAccessorySpriteKey]*spriteView
	petAccessoryMiss  map[petAccessorySpriteKey]struct{}
	rsmMeshCache      map[int][]retainedWorldMesh
	rsmNodeMatrices   map[*res.RSM]map[string]mat4
	rsmAnimNodes      map[animatedRSMNodeKey]map[string]mat4
	rsmBoundsCache    map[rsmBoundsCacheKey]rsmBounds
	rsmFaceMetaCache  map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta
	rsmAnimScratch    animatedRSMScratch
	rsmPlacementGrid  *rsmPlacementGrid
	runtimeRSMModels  map[string]*res.RSM
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
	homDeleteID       uint32
	mercDeleteID      uint32
	petSlotMachine    petSlotMachineState
	lockedAttackID    uint32
	attackFocusID     uint32
	attackFocusStart  time.Time
	scriptHighlight   actorHighlight
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
	mapWeatherCloud   mapWeatherCloudState
	mapWeatherPokJuk  mapWeatherFireworkState
	actorDeaths       map[uint32]time.Time
	actorVanishes     map[uint32]actorVanishFade
	actorSoundFrames  map[uint32]actorSoundFrame
	actorLife         map[uint32]actorLife
	skillUnitModels   map[uint32]skillUnitModel
	hiddenSkillUnits  map[uint32]skillUnitModel
	actorNameReqAt    map[uint32]time.Time
	guildEmblems      map[uint32]guildEmblem
	speechBubbles     map[uint32]speechBubble
	gndNormalSource   *res.GND
	gndTopNormals     [][4]modelPoint3
	taekwonNight      bool
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
	pvpCounter        gameui.PvPCounter
	console           gameui.ChatConsole
	npcDialog         gameui.NPCDialog
	escapeMenu        gameui.EscapeMenu
	teleportModal     gameui.TeleportModal
	disconnectDialog  gameui.ConfirmModal
	friendRequest     gameui.ConfirmModal
	friendConfirm     gameui.ConfirmModal
	partyRequest      gameui.ConfirmModal
	guildRequest      gameui.ConfirmModal
	tradeRequest      gameui.ConfirmModal
	adoptionRequest   gameui.ConfirmModal
	characterWindow   gameui.CharacterWindow
	basicMenu         gameui.BasicMenu
	inventoryBag      gameui.InventoryBagWindow
	equipmentWindow   gameui.EquipmentWindow
	viewEquipWindow   gameui.ViewEquipmentWindow
	storageWindow     gameui.StorageWindow
	cartWindow        gameui.CartWindow
	changeCartWindow  gameui.ChangeCartWindow
	itemPickup        gameui.ItemPickupNotification
	shopWindow        gameui.ShopWindow
	vendingWindow     gameui.VendingWindow
	itemInfoWindow    gameui.ItemInfoWindow
	identifyWindow    gameui.IdentifyWindow
	cardWindow        gameui.CardCompositionWindow
	makingArrow       gameui.MakingArrowWindow
	makingItem        gameui.MakingItemWindow
	repairItem        gameui.RepairItemWindow
	weaponRefine      gameui.WeaponRefineWindow
	petEggWindow      gameui.PetEggWindow
	petInfoWindow     gameui.PetInfoWindow
	petContext        gameui.PetContextMenu
	petConfirm        gameui.ConfirmModal
	homunculusInfo    gameui.HomunculusInfoWindow
	homunculusSkill   gameui.HomunculusSkillWindow
	homunculusContext gameui.HomunculusContextMenu
	homunculusConfirm gameui.ConfirmModal
	mercenaryInfo     gameui.MercenaryInfoWindow
	mercenarySkill    gameui.MercenarySkillWindow
	mercenaryContext  gameui.MercenaryContextMenu
	mercenaryConfirm  gameui.ConfirmModal
	statsWindow       gameui.StatsWindow
	skillWindow       gameui.SkillWindow
	emoteWindow       gameui.EmoteWindow
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

func (u *worldUI) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return playerIsDead(ctx) || u.keyboardInputBlocked(ctx)
}

func (u *worldUI) keyboardInputBlocked(ctx client.Context) bool {
	if u == nil {
		return false
	}
	return u.console.Active() ||
		u.npcDialog.IsOpen() ||
		u.escapeMenu.IsOpen() ||
		u.teleportModal.IsOpen() ||
		u.disconnectDialog.IsOpen() ||
		u.friendRequest.IsOpen() ||
		u.friendConfirm.IsOpen() ||
		u.partyRequest.IsOpen() ||
		u.guildRequest.IsOpen() ||
		u.tradeRequest.IsOpen() ||
		u.adoptionRequest.IsOpen() ||
		u.partyInfo.IsOpen() ||
		u.petConfirm.IsOpen() ||
		u.homunculusConfirm.IsOpen() ||
		u.mercenaryConfirm.IsOpen() ||
		u.settingsWindow.IsOpen() ||
		u.identifyWindow.IsOpen() ||
		u.cardWindow.IsOpen() ||
		u.makingArrow.IsOpen() ||
		u.makingItem.IsOpen() ||
		u.repairItem.IsOpen() ||
		u.weaponRefine.IsOpen() ||
		u.petEggWindow.IsOpen() ||
		u.petInfoWindow.IsOpen() ||
		u.homunculusInfo.IsOpen() ||
		u.homunculusSkill.IsOpen() ||
		u.mercenaryInfo.IsOpen() ||
		u.mercenarySkill.IsOpen() ||
		u.changeCartWindow.IsOpen() ||
		u.shopWindow.KeyboardShortcutsBlocked() ||
		u.vendingWindow.KeyboardShortcutsBlocked() ||
		u.tradeWindow.IsOpen() ||
		u.friendSettings.IsOpen() ||
		u.whisperWindow.IsOpen() ||
		u.chatRoomCreate.IsOpen() ||
		u.chatRoom.IsOpen() ||
		u.partySettings.IsOpen() ||
		u.partyCreate.IsOpen() ||
		u.partyInvite.IsOpen() ||
		u.skillTextPrompt.IsOpen()
}

func (m *WorldMode) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return m.ui.KeyboardShortcutsBlocked(ctx)
}

type actorSpriteKey struct {
	job         int
	head        int
	sex         byte
	admin       bool
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
	skill       session.Skill
	maxLevel    int
	targetID    uint32
	expires     time.Time
	readyAt     time.Time
	source      string
	started     time.Time
	lastChaseAt time.Time
}

type actorHighlight struct {
	id      uint32
	started time.Time
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
	mapFadePrewarm
	mapFadeIn
)

type mapFadeState struct {
	phase           mapFadePhase
	started         time.Time
	coveredFrames   int
	change          network.MapChange
	hasChange       bool
	characterSelect bool
}

const (
	mapFadeOutDuration = 220 * time.Millisecond
	mapFadeInDuration  = 340 * time.Millisecond
	// Present the transition cover before blocking on synchronous map loading.
	mapFadeHandoffFrames = 1
	// Warm the base map, then allow a frame for the initial post-ack actors.
	mapFadePrewarmFrames     = 2
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
	m.startMapPrewarm()
	m.camera.ResetTracking()
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
	m.cursorLevelNums = make(map[string]*spriteBillboard)
	m.damageMsgView = nil
	m.damageMsgMiss = false
	m.itemMarker = nil
	m.itemViews = make(map[itemSpriteKey]*spriteView)
	m.itemViewMiss = make(map[itemSpriteKey]struct{})
	m.effectViews = make(map[string]*spriteView)
	m.effectViewMiss = make(map[string]struct{})
	m.actorViews = make(map[actorSpriteKey]*humanoidSpriteView)
	m.actorViewMiss = make(map[actorSpriteKey]struct{})
	m.mercenaryViews = make(map[actorSpriteKey]*humanoidSpriteView)
	m.mercenaryViewMiss = make(map[actorSpriteKey]struct{})
	m.nonPCViews = make(map[int]*spriteView)
	m.nonPCViewMiss = make(map[int]struct{})
	m.gr2Models = make(map[int]*gr2ModelView)
	m.gr2ModelMiss = make(map[int]struct{})
	m.petAccessoryIDs = make(map[uint32]uint32)
	m.petAccessoryViews = make(map[petAccessorySpriteKey]*spriteView)
	m.petAccessoryMiss = make(map[petAccessorySpriteKey]struct{})
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	m.rsmNodeMatrices = make(map[*res.RSM]map[string]mat4)
	m.rsmAnimNodes = make(map[animatedRSMNodeKey]map[string]mat4)
	m.rsmBoundsCache = make(map[rsmBoundsCacheKey]rsmBounds)
	m.rsmFaceMetaCache = make(map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta)
	m.rsmAnimScratch.reset()
	m.rsmPlacementGrid = nil
	m.gndMeshCache = nil
	m.pendingWarp = false
	m.pendingAttack = attackIntent{}
	m.pendingPickup = pickupIntent{}
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	m.lockedAttackID = 0
	m.clearAttackFocus()
	m.clearScriptHighlight()
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
	m.actorAnims = make(map[uint32]actorAnimation)
	m.damageFloaters = nil
	m.scheduledSounds = nil
	m.scheduledStops = nil
	m.mapSoundNext = make(map[int]time.Time)
	m.mapWeatherSounds = make(map[int]time.Time)
	m.mapWeatherCloud = mapWeatherCloudState{}
	m.mapWeatherPokJuk = mapWeatherFireworkState{}
	m.actorDeaths = make(map[uint32]time.Time)
	m.actorVanishes = make(map[uint32]actorVanishFade)
	m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	m.actorLife = make(map[uint32]actorLife)
	m.speechBubbles = make(map[uint32]speechBubble)
	m.syncCurrentActorEffectStateEffects(ctx)
	m.ui.npcDialog.ResetPublished(ctx)
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	playerStatus := ""
	character := ctx.Session.SelectedCharacter()
	visualCharacter := localPlayerVisualCharacter(ctx)
	if view, status := loadPlayerHumanoidSpriteView(ctx.Resources, visualCharacter, ctx.Session.Sex, localPlayerIsAdmin(ctx)); view != nil {
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
	m.falconViews = make(map[int]*spriteView)
	m.falconViewMiss = make(map[int]struct{})
	m.falcons = make(map[uint32]*falconRenderState)
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
	glog.Debugf("player sprite resources char_id=%d name=%s admin=%t job=%d visual_job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, localPlayerIsAdmin(ctx), character.Job, visualCharacter.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
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
	m.ui.inventoryBag.Rebind(ctx, &m.ui.itemInfoWindow)
	m.ui.equipmentWindow.Rebind(ctx, &m.ui.itemInfoWindow, &m.ui.cartWindow, m)
	m.ui.cartWindow.Rebind(ctx, &m.ui.itemInfoWindow)
	m.ui.itemInfoWindow.Rebind(ctx, m)
	m.ui.statsWindow.Rebind(ctx)
	m.ui.skillWindow.Rebind(ctx, m)
	m.ui.emoteWindow.Rebind(ctx, &m.ui.console)
	m.ui.homunculusSkill.Rebind(ctx, m)
	m.ui.mercenarySkill.Rebind(ctx, m)
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
	m.ui.mercenaryInfo.Rebind(ctx)
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
		m.mapFade.phase = mapFadeHold
		m.mapFade.coveredFrames = 0
		return nil, nil
	}
	if m.mapFade.phase == mapFadeHold && m.mapFade.coveredFrames >= mapFadeHandoffFrames {
		if m.mapFade.characterSelect {
			return m.nextCharacterSelectMode(ctx), nil
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
	}
	m.advanceMapPrewarm(now)
	if m.mapFade.phase == mapFadeIn && m.mapFadeElapsed(now) {
		m.mapFade = mapFadeState{}
	}

	if m.updateDisconnectDialog(ctx) {
		return nil, nil
	}

	if next, stop := m.handleNetworkPackets(ctx, now); stop {
		return next, nil
	}
	m.ui.pvpCounter.Update(ctx)

	m.updatePendingAttack(ctx, "update", false)
	m.processPendingAttack(ctx)
	m.updatePendingPickup(ctx, "update", false)
	m.processPendingPickup(ctx)
	m.skills().UpdatePendingTarget(ctx, "update", false)
	m.skills().ProcessPendingTarget(ctx)
	m.processLockedAttack(ctx)
	now = time.Now()
	m.cleanupDeadActors(ctx, now)
	m.cleanupVanishedActors(ctx, now)
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
	if m.mapFade.phase == mapFadeHold || m.mapFade.phase == mapFadePrewarm {
		return nil, nil
	}
	m.ui.console.UpdatePresentation(ctx)
	dead := playerIsDead(ctx)
	keyboardBlocked := m.ui.keyboardInputBlocked(ctx)
	m.updateBotInput(ctx, !dead && !keyboardBlocked)
	if m.updatePetSlotMachine(ctx) {
		return nil, nil
	}
	if dead {
		if m.updateDeathUIInput(ctx) {
			return nil, nil
		}
	} else if !keyboardBlocked {
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
	}
	petContextConsumed := m.ui.petContext.Update(ctx)
	if action := m.ui.petContext.PopAction(); action.Kind != gameui.PetContextActionNone {
		if !dead {
			m.handlePetContextAction(ctx, action)
		}
		return nil, nil
	}
	if petContextConsumed {
		return nil, nil
	}
	if !dead && m.openPetContextFromInput(ctx, now) {
		return nil, nil
	}
	homunculusContextConsumed := m.ui.homunculusContext.Update(ctx)
	if action := m.ui.homunculusContext.PopAction(); action.Kind != gameui.HomunculusContextActionNone {
		if !dead {
			m.handleHomunculusContextAction(ctx, action)
		}
		return nil, nil
	}
	if homunculusContextConsumed {
		return nil, nil
	}
	if !dead && m.openHomunculusContextFromInput(ctx, now) {
		return nil, nil
	}
	mercenaryContextConsumed := m.ui.mercenaryContext.Update(ctx)
	if action := m.ui.mercenaryContext.PopAction(); action.Kind != gameui.MercenaryContextActionNone {
		if !dead {
			m.handleMercenaryContextAction(ctx, action)
		}
		return nil, nil
	}
	if mercenaryContextConsumed {
		return nil, nil
	}
	if !dead && m.openMercenaryContextFromInput(ctx, now) {
		return nil, nil
	}
	playerContextConsumed := m.ui.playerContext.Update(ctx)
	playerAction := m.ui.playerContext.PopAction()
	if dead && playerAction.Kind != gameui.PlayerContextActionNone {
		return nil, nil
	}
	switch action := playerAction; action.Kind {
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
	case gameui.PlayerContextActionAdopt:
		m.sendAdoptionRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	}
	if playerContextConsumed {
		return nil, nil
	}
	if !dead && m.handleCompanionAICommandClick(ctx, now) {
		return nil, nil
	}
	if !dead && m.openPlayerContextFromInput(ctx, now) {
		return nil, nil
	}
	if !m.petSlotMachine.active && !m.ui.escapeMenu.IsOpen() && !m.ui.teleportModal.IsOpen() && !m.ui.friendRequest.IsOpen() && !m.ui.friendConfirm.IsOpen() && !m.ui.partyRequest.IsOpen() && !m.ui.guildRequest.IsOpen() && !m.ui.tradeRequest.IsOpen() && !m.ui.adoptionRequest.IsOpen() && !m.ui.settingsWindow.IsOpen() && !m.ui.identifyWindow.IsOpen() && !m.ui.petEggWindow.IsOpen() && !m.ui.petInfoWindow.IsOpen() && !m.ui.petConfirm.IsOpen() && !m.ui.homunculusInfo.IsOpen() && !m.ui.homunculusSkill.IsOpen() && !m.ui.homunculusConfirm.IsOpen() && !m.ui.mercenaryInfo.IsOpen() && !m.ui.mercenarySkill.IsOpen() && !m.ui.mercenaryConfirm.IsOpen() {
		m.updateCameraRotation(ctx)
	}
	if !dead && m.ui.escapeMenu.IsOpen() {
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
	if m.ui.adoptionRequest.Update(ctx) {
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
	if m.ui.mercenaryConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.mercenaryInfo.Update(ctx) {
		if action := m.ui.mercenaryInfo.PopAction(); action.Kind != gameui.MercenaryInfoActionNone {
			m.handleMercenaryInfoAction(ctx, action)
		}
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
	if m.ui.makingArrow.Update(ctx) {
		return nil, nil
	}
	if m.ui.makingItem.Update(ctx) {
		return nil, nil
	}
	if m.ui.repairItem.Update(ctx) {
		return nil, nil
	}
	if m.ui.weaponRefine.Update(ctx) {
		return nil, nil
	}
	if !dead && m.ui.console.UpdateInput(ctx) {
		return nil, nil
	}
	if m.ui.settingsWindow.Update(ctx) {
		return nil, nil
	}
	if !dead && m.ui.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return nil, nil
	}
	characterWindowConsumed := m.ui.characterWindow.Update(ctx)
	m.ui.basicMenu.FollowCharacterWindow(ctx, &m.ui.characterWindow)
	if characterWindowConsumed {
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
	if m.ui.homunculusSkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.mercenarySkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.guildWindow.UpdateDrag(ctx, &m.ui.shortcutBar) {
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
	if m.ui.homunculusSkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.mercenarySkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.toggleEmoteWindowFromInput(ctx) {
		return nil, nil
	}
	if m.ui.emoteWindow.Update(ctx, &m.ui.console) {
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
	if m.ui.guildWindow.Update(ctx, &m.ui.shortcutBar, m) {
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
	if !m.ui.keyboardInputBlocked(ctx) {
		if m.ui.basicMenu.Update(ctx, m.basicMenuCallbacks(ctx)) {
			return nil, nil
		}
	}
	minimapDragging := m.ui.minimap.Update(ctx)
	removeExpiredStatusEffects(ctx.Session, now)
	m.ui.statusIcons.Update(ctx, now)
	m.syncLevel99AuraEffects(ctx, now)
	pointerBlocked := minimapDragging || m.mapPointerBlocked(ctx)
	if !pointerBlocked {
		m.updateCameraZoom(ctx)
	}
	if dead {
		return nil, nil
	}
	m.updateCompanionAI(ctx, now)
	m.updateBot(ctx, now)

	leftClick := !pointerBlocked && ctx.Input.MouseJustPressed(input.MouseButtonLeft)
	if leftClick {
		// One-shot interactions must not be delayed by the short throttle used
		// for repeated walk requests. Scheduling the held-click repeat here also
		// prevents a throttled ground click from turning into a walk on the next
		// frame merely because the button is still down.
		m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)
	}
	if leftClick && m.pendingSkill.skill.ID != 0 {
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		m.skills().HandleClick(ctx, projection, now)
		return nil, nil
	}

	if leftClick {
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		if m.handlePetCaptureClick(ctx, projection, now) {
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
		if actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok && isWarpActor(actor) {
			m.clearLockedAttack()
			m.clearAttackFocus()
			if targetX, targetY, targetOK := warpWalkTarget(ctx, actor, now); targetOK {
				glog.Debugf("click warp target mouse=%d,%d id=%d job=%d player=%d,%d warp=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Job, playerX, playerY, actor.X, actor.Y, targetX, targetY)
				m.requestWalk(ctx, targetX, targetY, "warp click")
			} else {
				glog.Debugf("click warp target unreachable mouse=%d,%d id=%d job=%d player=%d,%d warp=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Job, playerX, playerY, actor.X, actor.Y)
			}
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
		if targetX, targetY, ok := clickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok && m.walkReady(now) {
			glog.Debugf("click walk target mouse=%d,%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, playerX, playerY, targetX, targetY)
			m.cancelAttackIntent()
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
	case gameui.EscapeMenuActionSavePoint:
		m.ui.escapeMenu.ReturnToSavePoint(ctx)
	case gameui.EscapeMenuActionCharacterSelect:
		m.ui.escapeMenu.RequestCharacterSelect(ctx)
	case gameui.EscapeMenuActionSettings:
		m.ui.settingsWindow.OpenWindow(ctx)
	}
}

func (m *WorldMode) updateDeathUIInput(ctx client.Context) bool {
	// Active chat owns Escape so the first press leaves text entry. Enter also
	// gets priority so chat can be activated even while the pointer is over the
	// death menu. Otherwise the menu owns Escape and may be freely toggled.
	chatFirst := m.ui.console.Active() || (ctx.Input != nil && ctx.Input.JustPressed(input.KeyEnter))
	if chatFirst && m.ui.console.UpdateInput(ctx) {
		return true
	}
	if m.ui.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return true
	}
	return !chatFirst && m.ui.console.UpdateInput(ctx)
}

func (m *WorldMode) applyQuitGameAck(ctx client.Context, ack network.QuitGameAck) {
	handled := m.ui.escapeMenu.ApplyQuitGameAck(ack)
	if ack.Allowed {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
		return
	}
	if handled {
		m.addLeaveWorldRefusalMessage(ctx)
	}
}

func (m *WorldMode) addLeaveWorldRefusalMessage(ctx client.Context) {
	message := disconnectMessageText(ctx.Resources, disconnectMessage{
		msgID:    502,
		fallback: "You cannot exit the game right now.",
	})
	m.ui.console.AddErrorMessage("%s", message)
}

func (m *WorldMode) openEscapeMenuFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.escapeMenu.IsOpen() || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	if m.ui.teleportModal.IsOpen() || m.ui.friendRequest.IsOpen() || m.ui.friendConfirm.IsOpen() || m.ui.partyRequest.IsOpen() || m.ui.guildRequest.IsOpen() || m.ui.tradeRequest.IsOpen() || m.ui.adoptionRequest.IsOpen() {
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

func (m *WorldMode) toggleEmoteWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.keyboardInputBlocked(ctx) {
		return false
	}
	if !ctx.Input.Pressed(input.KeyAlt) || !ctx.Input.JustPressed(input.KeyL) {
		return false
	}
	m.ui.emoteWindow.Toggle(ctx, &m.ui.console)
	return true
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
	ctx.World.ResetMapProperty()
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	applyWarpPosition(ctx, change.X, change.Y)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	if reuseLoadedMap {
		m.camera.ResetTracking()
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
	next.camera.yawOffset = m.camera.yawOffset
	next.camera.pitch = m.camera.pitch
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
	next.ui.homunculusSkill = m.ui.homunculusSkill
	next.ui.mercenaryInfo = m.ui.mercenaryInfo
	next.ui.mercenarySkill = m.ui.mercenarySkill
	next.petProperty = m.petProperty
	next.hasPetProperty = m.hasPetProperty
	next.petOldFullness = m.petOldFullness
	next.petLastTalk = m.petLastTalk
	next.ui.statsWindow = m.ui.statsWindow
	next.ui.skillWindow = m.ui.skillWindow
	next.ui.emoteWindow = m.ui.emoteWindow
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
	next.ui.pvpCounter = m.ui.pvpCounter
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
	if playerIsDead(ctx) {
		return
	}
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
			m.drawSkillUnitRSMModels(screen, ctx, projection, now)
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
		m.ui.homunculusSkill.Draw(screen, ctx, m)
		m.ui.mercenarySkill.Draw(screen, ctx, m)
		m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
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

func (m *WorldMode) FrameSubmitted() {
	m.recordCoveredMapFrame()
}

func (m *WorldMode) DrawUIOverlay(ctx client.Context, screen *render.Frame) {
	if ctx.Config.Render.NoUI {
		return
	}
	m.ui.inventoryBag.DrawTooltip(ctx, screen)
	m.ui.equipmentWindow.DrawTooltip(ctx, screen)
	m.ui.cartWindow.DrawTooltip(ctx, screen)
	m.ui.itemInfoWindow.DrawTooltip(ctx, screen)
	m.ui.skillWindow.DrawTooltip(ctx, screen)
	m.ui.homunculusSkill.DrawTooltip(ctx, screen)
	m.ui.mercenarySkill.DrawTooltip(ctx, screen)
	m.ui.guildWindow.DrawTooltip(ctx, screen)
	m.ui.shortcutBar.DrawTooltip(ctx, screen)
	m.ui.itemPickup.Draw(screen, ctx, m, time.Now())
}

func (m *WorldMode) drawUIDragGhosts(screen *render.Frame, ctx client.Context) {
	m.ui.inventoryBag.DrawDragGhost(screen, ctx, m)
	m.ui.storageWindow.DrawDragGhost(screen, ctx, m)
	m.ui.cartWindow.DrawDragGhost(screen, ctx, m)
	m.ui.shopWindow.DrawDragGhost(screen, ctx, m)
	m.ui.vendingWindow.DrawDragGhost(screen, ctx, m)
	m.ui.skillWindow.DrawDragGhost(screen, ctx, m)
	m.ui.homunculusSkill.DrawDragGhost(screen, ctx, m)
	m.ui.mercenarySkill.DrawDragGhost(screen, ctx, m)
	m.ui.guildWindow.DrawDragGhost(screen, ctx, m)
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
		if _, dead := deadActors[actor.ID]; dead && !skillCanTargetDeadActor(skill, actor) {
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
	m.drawSkillUnitRSMModels(screen, ctx, projection, now)
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
	m.drawSceneActorFalcons(screen, ctx, projection, actors)
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

func (m *WorldMode) sceneLighting(rsw *res.RSW) sceneLighting {
	lighting := sceneLightingFromRSW(rsw)
	if !m.taekwonNight {
		return lighting
	}
	lighting.diffuse.x = math.Min(lighting.diffuse.x, 0.5)
	lighting.diffuse.y = math.Min(lighting.diffuse.y, 0.5)
	lighting.env = modelPoint3{
		x: 1 - (1-lighting.ambient.x)*(1-lighting.diffuse.x),
		y: 1 - (1-lighting.ambient.y)*(1-lighting.diffuse.y),
		z: 1 - (1-lighting.ambient.z)*(1-lighting.diffuse.z),
	}
	return lighting
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
