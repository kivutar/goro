package game

import (
	"context"
	"image"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/game/pet"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/bot"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/login"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/game/weather"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type WorldMode struct {
	petSystem         pet.System
	battleSystem      battle.System
	actorSystem       actors.System
	socialSystem      social.System
	npcSystem         npc.System
	skillSystem       skills.System
	mapScene          maps.System
	effectSystem      effects.System
	walkCooldownUntil time.Time
	walkSequence      uint64
	nextHeldWalkAt    time.Time
	camera            followCamera
	cameraShakeStart  time.Time
	cameraShakeEnd    time.Time
	spriteUpscaler    *sprites.Upscaler
	imageCache        map[string]image.Image
	imageMiss         map[string]struct{}

	cursorView     *scene.SpriteView
	cursorViewMiss bool
	cursorFallback *render.Image
	cursorAction   int
	cursorStarted  time.Time

	itemMarker   *render.Image
	itemViews    map[itemSpriteKey]*scene.SpriteView
	itemViewMiss map[itemSpriteKey]struct{}

	pendingWarp      bool
	deferredPackets  []network.Packet
	pendingAttack    attackIntent
	pendingPickup    pickupIntent
	lockedAttackID   uint32
	attackFocusID    uint32
	attackFocusStart time.Time
	scriptHighlight  actorHighlight
	lastAttackAt     time.Time
	lastChaseAt      time.Time

	scheduledSounds  []scheduledSound
	scheduledStops   []scheduledActorStop
	scheduledResumes []scheduledWalkResume
	weather          weather.System

	actorSoundFrames map[uint32]actorSoundFrame

	ui              worldUI
	mapFade         maps.Fade
	hoveredWalk     hoveredWalkCellCache
	bot             *bot.Script
	companionSystem companion.System
}

type worldUI struct {
	preferences          windowPreferences
	pet                  pet.Windows
	battle               battle.Windows
	npc                  npc.Windows
	social               social.Windows
	companions           companion.Windows
	minimap              gameui.Minimap
	pvpCounter           gameui.PvPCounter
	levelUpNotifications gameui.LevelUpNotifications
	announcement         gameui.Announcement
	poptips              gameui.Poptips
	console              gameui.ChatConsole

	escapeMenu       gameui.EscapeMenu
	teleportModal    gameui.TeleportModal
	disconnectDialog gameui.ConfirmModal
	starPlaceConfirm gameui.ConfirmModal
	characterWindow  gameui.CharacterWindow
	basicMenu        gameui.BasicMenu
	inventoryBag     gameui.InventoryBagWindow
	equipmentWindow  gameui.EquipmentWindow
	viewEquipWindow  gameui.ViewEquipmentWindow
	storageWindow    gameui.StorageWindow

	cartWindow       gameui.CartWindow
	changeCartWindow gameui.ChangeCartWindow
	itemPickup       gameui.ItemPickupNotification

	itemWindows    gameui.ItemWindows
	identifyWindow gameui.IdentifyWindow
	cardWindow     gameui.CardCompositionWindow
	makingArrow    gameui.MakingArrowWindow
	makingItem     gameui.MakingItemWindow
	repairItem     gameui.RepairItemWindow
	weaponRefine   gameui.WeaponRefineWindow

	statsWindow     gameui.StatsWindow
	skillWindow     gameui.SkillWindow
	questWindow     gameui.QuestWindow
	worldMap        gameui.WorldMapWindow
	emoteWindow     gameui.EmoteWindow
	chatShortcuts   gameui.ChatShortcutsWindow
	skillTextPrompt gameui.TextPromptWindow
	settingsWindow  gameui.SettingsWindow
	shortcutBar     gameui.ShortcutBar
}

func (u *worldUI) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return scene.PlayerIsDead(ctx) || u.keyboardInputBlocked(ctx)
}

func (u *worldUI) keyboardInputBlocked(ctx client.Context) bool {
	if u == nil {
		return false
	}
	return u.console.Active() || u.nonConsoleKeyboardInputBlocked(ctx)
}

func (u *worldUI) nonConsoleKeyboardInputBlocked(ctx client.Context) bool {
	return u.nonConsoleKeyboardInputBlockedExcept(ctx, nil)
}

// A window may handle its own toggle shortcut, but not bypass other forms.
func (u *worldUI) nonConsoleKeyboardInputBlockedExcept(ctx client.Context, except *gameui.Window) bool {
	if u == nil {
		return false
	}
	open := func(w *gameui.Window) bool { return w != except && w.IsOpen() }
	return u.npc.Dialog.IsOpen() ||
		!u.npcInputAvailable() ||
		open(&u.settingsWindow.Window) ||
		u.chatShortcuts.IsOpen() ||
		u.worldMap.IsOpen() ||
		u.battle.MonsterInfo.IsOpen() ||
		u.identifyWindow.IsOpen() ||
		u.cardWindow.IsOpen() ||
		u.makingArrow.IsOpen() ||
		u.makingItem.IsOpen() ||
		u.repairItem.IsOpen() ||
		u.weaponRefine.IsOpen() ||
		u.pet.Egg.IsOpen() ||
		(except != &u.pet.Info.Window && u.pet.Info.KeyboardShortcutsBlocked()) ||
		(except != &u.companions.HomunculusInfo.Window && u.companions.HomunculusInfo.KeyboardShortcutsBlocked()) ||
		u.changeCartWindow.IsOpen() ||
		u.itemWindows.KeyboardShortcutsBlocked() ||
		u.npc.Shop.KeyboardShortcutsBlocked() ||
		u.social.VendingWindow.KeyboardShortcutsBlocked() ||
		u.social.TradeWindow.IsOpen() ||
		u.social.MailWindow.IsOpen() ||
		u.social.GuildWindow.KeyboardShortcutsBlocked() ||
		u.social.FriendSettings.IsOpen() ||
		u.social.WhisperWindows.IsOpen() ||
		open(&u.social.ChatRoomCreate.Window) ||
		u.social.ChatRoom.IsOpen() ||
		u.social.PartySettings.IsOpen() ||
		u.social.PartyCreate.IsOpen() ||
		u.social.PartyInvite.IsOpen() ||
		u.skillTextPrompt.IsOpen()
}

func (u *worldUI) interactionModalOpen() bool {
	if u == nil {
		return false
	}
	return u.teleportModal.IsOpen() ||
		u.social.MailWindow.ModalOpen() ||
		u.battle.AutoSpell.IsOpen() ||
		u.npc.StoragePassword.IsOpen() ||
		u.social.FriendRequest.IsOpen() ||
		u.social.FriendConfirm.IsOpen() ||
		u.social.PartyRequest.IsOpen() ||
		u.social.GuildRequest.IsOpen() ||
		u.social.GuildAllianceRequest.IsOpen() ||
		u.social.GuildRelationConfirm.IsOpen() ||
		u.social.GuildMemberPrompt.IsOpen() ||
		u.social.ChatRoomPassword.IsOpen() ||
		u.social.TradeRequest.IsOpen() ||
		u.social.AdoptionRequest.IsOpen() ||
		u.starPlaceConfirm.IsOpen()
}

func (m *WorldMode) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return m.ui.KeyboardShortcutsBlocked(ctx)
}

type pickupIntent struct {
	itemID  uint32
	expires time.Time
	readyAt time.Time
}

type actorHighlight struct {
	id      uint32
	started time.Time
}

func NewWorldMode() *WorldMode {
	m := &WorldMode{}
	m.ui.npc.BindDialogLifecycle()
	return m
}

func (m *WorldMode) Name() string {
	return "world"
}

func (m *WorldMode) Enter(ctx client.Context) Mode {
	if ctx.Session != nil {
		ctx.Session.SuperNovicePrayer = session.SuperNovicePrayer{}
	}
	m.releaseMapTextures()
	now := time.Now()
	m.ui.npc.BindDialogLifecycle()
	m.mapFade.StartPrewarm()
	m.camera.ResetTracking()
	ctx.World.GAT = nil
	ctx.World.GND = nil
	ctx.World.RSW = nil
	ctx.World.RSM = nil
	ctx.World.RSMFail = 0
	if ctx.World.MapName != "" {
		gat, _, err := maps.LoadGAT(ctx.Resources, ctx.World.MapName)
		if err != nil {
			return newMapErrorMode(ctx, err, m.ui.console)
		}
		ctx.World.GAT = gat
	}
	m.mapScene.Textures = make(map[string]*render.Image)
	m.mapScene.TextureMiss = make(map[string]struct{})
	m.actorSystem.PlayerView = nil
	m.actorSystem.ShadowView = nil
	m.actorSystem.ShadowViewMiss = false
	m.cursorView = nil
	m.cursorViewMiss = false
	m.cursorFallback = nil
	m.cursorAction = cursorActionDefault
	m.cursorStarted = now
	m.battleSystem.DamageNumberView = nil
	m.battleSystem.DamageNumberMiss = false
	m.battleSystem.DamageNumbers = make(map[string]*scene.Billboard)
	m.skillSystem.CursorLevelNums = make(map[string]*scene.Billboard)
	m.battleSystem.DamageMsgView = nil
	m.battleSystem.DamageMsgMiss = false
	m.npcSystem.TimeFontView = nil
	m.npcSystem.TimeFontMiss = false
	m.itemMarker = nil
	m.itemViews = make(map[itemSpriteKey]*scene.SpriteView)
	m.itemViewMiss = make(map[itemSpriteKey]struct{})
	m.effects().EffectViews = make(map[string]*scene.SpriteView)
	m.effects().EffectViewMiss = make(map[string]struct{})
	m.actorSystem.ActorViews = make(map[sprites.Appearance]*sprites.Humanoid)
	m.actorSystem.ActorViewMiss = make(map[sprites.Appearance]struct{})
	m.actorSystem.MercenaryViews = make(map[sprites.Appearance]*sprites.Humanoid)
	m.actorSystem.MercenaryViewMiss = make(map[sprites.Appearance]struct{})
	m.actorSystem.NonPCViews = make(map[int]*scene.SpriteView)
	m.actorSystem.NonPCViewMiss = make(map[int]struct{})
	m.actorSystem.GR2Models = make(map[int]*actors.GR2ModelView)
	m.actorSystem.GR2ModelMiss = make(map[int]struct{})
	m.actorSystem.PetAccessoryIDs = make(map[uint32]uint32)
	m.actorSystem.PetAccessoryViews = make(map[actors.PetAccessorySpriteKey]*scene.SpriteView)
	m.actorSystem.PetAccessoryMiss = make(map[actors.PetAccessorySpriteKey]struct{})
	m.mapScene.ResetGeometry()
	m.pendingWarp = false
	m.pendingAttack = attackIntent{}
	m.pendingPickup = pickupIntent{}
	m.skillSystem.Pending = skills.PendingTarget{}
	m.skillSystem.PendingText = skills.PendingTextTarget{}
	m.socialSystem.GuildAction = gameui.GuildMemberAction{}
	m.socialSystem.GuildOpenPending = false
	m.ui.social.GuildMemberPrompt.Close()
	m.lockedAttackID = 0
	m.clearAttackFocus()
	m.clearScriptHighlight()
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
	m.actorSystem.ActorAnims = make(map[uint32]scene.ActorAnimation)
	m.actorSystem.ComboDelays = nil
	m.effects().ActorBlurs = nil
	m.battleSystem.DamageFloaters = nil
	m.npcSystem.Progress = npc.ProgressState{}
	m.npcSystem.ShowDigit = npc.ShowDigitState{}
	m.ui.minimap.ClearBossMarker()
	m.ui.poptips.Clear()
	m.scheduledSounds = nil
	m.scheduledStops = nil
	m.mapScene.MapSoundNext = make(map[int]time.Time)
	m.weather = weather.System{}
	m.effects().StarFlights = nil
	m.actorSystem.ActorDeaths = make(map[uint32]time.Time)
	m.actorSystem.ActorVanishes = make(map[uint32]actors.VanishFade)
	m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	m.battleSystem.ActorLife = make(map[uint32]battle.Life)
	m.actorSystem.SpeechBubbles = make(map[uint32]actors.SpeechBubble)
	m.syncCurrentActorEffectStateEffects(ctx)
	m.ui.npc.Dialog.ResetPublished(ctx)
	m.ui.npc.Cutin.Clear()
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	// Bots need the collision grid and map acknowledgement, but no scene assets.
	if ctx.Config.Headless {
		if ctx.World.MapName != "" {
			_ = ctx.Network.SendLoadEndAck()
		}
		return nil
	}
	playerStatus := ""
	character := ctx.Session.SelectedCharacter()
	visualCharacter := actors.LocalPlayerVisualCharacter(ctx)
	if view, status := sprites.LoadPlayer(ctx.Resources, visualCharacter, ctx.Session.Sex, actors.LocalPlayerIsAdmin(ctx)); view != nil {
		m.actorSystem.PlayerView = view
		playerStatus = status
	} else {
		playerStatus = status
	}
	if view, status := sprites.LoadShadow(ctx.Resources); view != nil {
		m.actorSystem.ShadowView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.actorSystem.ShadowViewMiss = true
		glog.Warnf("actor shadow resources unavailable: %s", status)
	}
	m.actorSystem.CartViews = make(map[int]*scene.SpriteView)
	m.actorSystem.CartViewMiss = make(map[int]struct{})
	m.actorSystem.FalconViews = make(map[int]*scene.SpriteView)
	m.actorSystem.FalconViewMiss = make(map[int]struct{})
	m.actorSystem.Falcons = make(map[uint32]*actors.FalconRenderState)
	if view, status := sprites.LoadCursor(ctx.Resources); view != nil {
		m.cursorView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.cursorViewMiss = true
		glog.Warnf("cursor resources unavailable: %s", status)
	}
	render.SetCursorMode(render.CursorModeHidden)
	glog.Debugf("player sprite resources char_id=%d name=%s admin=%t job=%d visual_job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, actors.LocalPlayerIsAdmin(ctx), character.Job, visualCharacter.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
	m.rebindPersistentUI(ctx)
	if ctx.World.MapName == "" {
		return nil
	}

	if gnd, _, err := maps.LoadGND(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.GND = gnd
	} else {
		ctx.World.GND = nil
	}
	if rsw, rswSource, err := maps.LoadRSW(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.RSW = rsw
		ctx.World.RSM, ctx.World.RSMFail = maps.LoadRSMModels(ctx.Resources, rsw)
		m.playMapBGM(ctx, rswSource)
		m.mapScene.PreloadMapSounds(ctx)
	} else {
		ctx.World.RSW = nil
		ctx.World.RSM = nil
		ctx.World.RSMFail = 0
		m.playMapBGM(ctx, ctx.World.MapName)
	}
	m.mapScene.PreloadTextures(ctx)
	_ = ctx.Network.SendLoadEndAck()
	return nil
}

func (m *WorldMode) rebindPersistentUI(ctx client.Context) {
	m.ui.console.Rebind(ctx)
	m.ui.console.OnGuildWindow = func() { m.social().ToggleGuildWindow(ctx) }
	m.ui.social.GuildWindow.EmblemImage = m.social().SessionGuildEmblemImage
	m.social().SetGuildEmblemOptions(ctx)
	m.ui.characterWindow.Rebind(ctx)
	m.ui.basicMenu.Rebind(ctx, m.basicMenuCallbacks(ctx))
	m.ui.inventoryBag.Rebind(ctx, &m.ui.itemWindows)
	m.ui.equipmentWindow.Rebind(ctx, &m.ui.itemWindows, &m.ui.cartWindow, m)
	m.ui.cartWindow.Rebind(ctx, &m.ui.itemWindows)
	m.ui.itemWindows.Rebind(ctx, m)
	m.ui.statsWindow.Rebind(ctx)
	m.ui.skillWindow.Rebind(ctx, m)
	m.ui.questWindow.Rebind(ctx, func(id uint32, active bool) { m.setQuestActive(ctx, id, active) })
	m.ui.worldMap.Rebind(ctx)
	m.ui.levelUpNotifications.Rebind(ctx)
	m.ui.emoteWindow.OnSelect = m.ui.chatShortcuts.SelectEmotion
	m.ui.emoteWindow.Rebind(ctx, &m.ui.console)
	m.ui.chatShortcuts.Rebind(ctx, &m.ui.console, func() {
		m.ui.emoteWindow.OpenWindow(ctx, &m.ui.console)
	})
	m.ui.companions.HomunculusSkill.Rebind(ctx, m)
	m.ui.companions.MercenarySkill.Rebind(ctx, m)
	m.ui.social.FriendsWindow.Rebind(ctx)
	m.ui.social.GuildWindow.Rebind(ctx)
	m.ui.social.GuildMemberPrompt.Rebind(ctx)
	m.ui.social.FriendSettings.Rebind(ctx)
	m.ui.social.PartySettings.Rebind(ctx)
	m.ui.social.PartyCreate.Rebind(ctx)
	m.ui.social.PartyInvite.Rebind(ctx)
	m.ui.social.ChatRoomCreate.Rebind(ctx)
	m.ui.social.ChatRoomPassword.Rebind(ctx)
	m.ui.social.ChatRoom.Rebind(ctx)
	m.ui.settingsWindow.Rebind(ctx)
	m.ui.companions.HomunculusInfo.Rebind(ctx)
	m.ui.companions.MercenaryInfo.Rebind(ctx)
	m.ui.social.WhisperWindows.Rebind(ctx)
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
	if !ctx.Config.Headless {
		defer m.SaveUISettings(ctx)
	}
	now := time.Now()
	if m.mapFade.Phase == maps.FadeOut {
		if !m.mapFade.Elapsed(now) {
			return nil, nil
		}
		m.mapFade.Phase = maps.FadeHold
		m.mapFade.CoveredFrames = 0
		return nil, nil
	}
	if m.mapFade.Phase == maps.FadeHold && m.mapFade.CoveredFrames >= maps.FadeHandoffFrames {
		if m.mapFade.CharacterSelect {
			return m.nextCharacterSelectMode(ctx), nil
		}
		if m.mapFade.HasChange {
			change := m.mapFade.Change
			m.mapFade = maps.Fade{Phase: maps.FadeHold, Started: now}
			next := m.handleMapChange(ctx, change)
			if next != nil {
				return next, nil
			}
			if !m.pendingWarp {
				m.mapFade.StartIn(now)
			}
			return nil, nil
		}
	}
	m.mapFade.AdvancePrewarm(now, len(m.mapScene.Images.Uploads) != 0)
	if m.mapFade.Phase == maps.FadeIn && m.mapFade.Elapsed(now) {
		m.mapFade = maps.Fade{}
	}

	if m.updateDisconnectDialog(ctx) {
		return nil, nil
	}

	if next, stop := m.handleNetworkPackets(ctx, now); stop {
		return next, nil
	}
	if !ctx.Config.Headless {
		m.restoreUISettings(ctx)
	}
	// Status presentation must follow server updates even when a window or
	// modal consumes input for the rest of the frame.
	m.battle().RemoveExpiredStatusEffects(ctx.Session, now)
	m.effects().AdvanceTaekwonFootprints(ctx, now)
	m.social().UpdateMail(ctx, now)
	progressBlocksActions := m.npc().UpdateProgress(ctx, now)
	if !ctx.Config.Headless {
		m.ui.inventoryBag.UpdatePresentation(ctx, &m.ui.itemWindows)
		m.ui.statsWindow.UpdatePresentation(ctx)
		m.ui.battle.StatusIcons.Update(ctx, now)
		m.ui.pvpCounter.Update(ctx)
		if !progressBlocksActions && m.handleLevelUpNotificationAction(ctx, m.ui.levelUpNotifications.Update(ctx)) {
			// The notification click belongs exclusively to the UI. Returning here
			// prevents the same press from reaching the map after the icon closes.
			return nil, nil
		}
	}
	if !progressBlocksActions {
		m.updatePendingAttack(ctx, "update", false)
		m.processPendingAttack(ctx)
		m.updatePendingPickup(ctx, "update", false)
		m.processPendingPickup(ctx)
		m.skills().UpdatePendingTarget(ctx, "update", false)
		m.skills().ProcessPendingTarget(ctx)
		m.processLockedAttack(ctx)
	}
	now = time.Now()
	m.cleanupDeadActors(ctx, now)
	m.cleanupVanishedActors(ctx, now)
	m.processScheduledActorStops(ctx, now)
	m.processScheduledWalkResumes(ctx, now)
	if ctx.Config.Headless {
		m.playDueScheduledSounds(ctx, now)
		if !progressBlocksActions && m.mapFade.Phase != maps.FadeHold && m.mapFade.Phase != maps.FadePrewarm {
			dead := scene.PlayerIsDead(ctx)
			m.updateBotInput(ctx, !dead)
			if !dead {
				m.companions().Update(ctx, m.actorSystem.ActorDeaths, now)
			}
			m.updateBot(ctx, now)
		}
		return nil, nil
	}
	m.processActorMotionSounds(ctx, now)
	m.mapScene.ProcessMapSounds(ctx, now, m.scheduleSoundVolumeAtPosition)
	m.playDueScheduledSounds(ctx, now)

	m.camera.Update(ctx, now)
	if m.mapFade.Phase == maps.FadeHold || m.mapFade.Phase == maps.FadePrewarm {
		return nil, nil
	}
	m.ui.console.UpdatePresentation(ctx)
	m.ui.questWindow.UpdatePresentation(ctx, now)
	m.ui.worldMap.UpdatePresentation(ctx)
	if progressBlocksActions {
		return nil, nil
	}
	if m.npc().UpdateStoragePassword(ctx) {
		return nil, nil
	}
	// The drop amount prompt is modal and may overlap the always-present chat
	// field. Give it first refusal on Escape and pointer input so a focused chat
	// field cannot consume the cancellation before the prompt sees it.
	if m.ui.inventoryBag.UpdateDropPrompt(ctx) {
		return nil, nil
	}
	if m.ui.social.MailWindow.UpdateModal(ctx) || m.ui.social.MailWindow.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	if m.ui.social.GuildWindow.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	if m.ui.chatShortcuts.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	dead := scene.PlayerIsDead(ctx)
	keyboardBlocked := m.ui.keyboardInputBlocked(ctx)
	m.updateBotInput(ctx, !dead && !keyboardBlocked)
	if m.pets().UpdateSlotMachine(ctx) {
		return nil, nil
	}
	// Window.Update consumes pointer hover so that map input does not pass
	// through the UI. Handle keyboard-only window shortcuts before pointer
	// dispatch, otherwise their JustPressed event can be lost.
	if m.chatShortcutFromInput(ctx) || m.toggleEmoteWindowFromInput(ctx) || m.toggleGuildWindowFromInput(ctx) || m.toggleQuestWindowFromInput(ctx) || m.toggleWorldMapFromInput(ctx) || m.toggleWindowFromInput(ctx) {
		return nil, nil
	}
	if !dead && !m.ui.nonConsoleKeyboardInputBlocked(ctx) && m.ui.shortcutBar.UpdateKeyboardInput(ctx, m, m.ui.console.Active()) {
		return nil, nil
	}
	if dead {
		m.updateBot(ctx, now)
		if m.updateDeathUIInput(ctx) {
			return nil, nil
		}
	} else if !keyboardBlocked {
		if m.skills().CancelFromInput(ctx) {
			return nil, nil
		}
		if m.pets().CancelCaptureFromInput(ctx) {
			return nil, nil
		}
		if m.openEscapeMenuFromInput(ctx) {
			return nil, nil
		}
		m.skills().AdjustPendingLevelFromWheel(ctx)
	}
	petContextConsumed := m.ui.pet.Context.Update(ctx)
	if action := m.ui.pet.Context.PopAction(); action.Kind != gameui.PetContextActionNone {
		if !dead {
			m.pets().HandleContextAction(ctx, action)
		}
		return nil, nil
	}
	if petContextConsumed {
		return nil, nil
	}
	if !dead && m.openPetContextFromInput(ctx, now) {
		return nil, nil
	}
	homunculusContextConsumed := m.ui.companions.HomunculusContext.Update(ctx)
	if action := m.ui.companions.HomunculusContext.PopAction(); action.Kind != gameui.HomunculusContextActionNone {
		if !dead {
			m.companions().HandleHomunculusContextAction(ctx, action)
		}
		return nil, nil
	}
	if homunculusContextConsumed {
		return nil, nil
	}
	if !dead && m.openHomunculusContextFromInput(ctx, now) {
		return nil, nil
	}
	mercenaryContextConsumed := m.ui.companions.MercenaryContext.Update(ctx)
	if action := m.ui.companions.MercenaryContext.PopAction(); action.Kind != gameui.MercenaryContextActionNone {
		if !dead {
			m.companions().HandleMercenaryContextAction(ctx, action)
		}
		return nil, nil
	}
	if mercenaryContextConsumed {
		return nil, nil
	}
	if !dead && m.openMercenaryContextFromInput(ctx, now) {
		return nil, nil
	}
	playerContextConsumed := m.ui.social.PlayerContext.Update(ctx)
	playerAction := m.ui.social.PlayerContext.PopAction()
	if dead && playerAction.Kind != gameui.PlayerContextActionNone {
		return nil, nil
	}
	switch action := playerAction; action.Kind {
	case gameui.PlayerContextActionGMCommand:
		m.ui.console.SendText(ctx, action.Command)
		return nil, nil
	case gameui.PlayerContextActionAddFriend:
		m.social().SendAddFriend(ctx, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteParty:
		m.social().SendPartyInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteGuild:
		m.social().SendGuildInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionGuildAlliance:
		m.social().SendGuildAllianceRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionGuildHostility:
		m.social().SendGuildHostilityRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionTrade:
		m.social().SendTradeRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionSeeEquipment:
		m.sendViewEquipmentRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionAdopt:
		m.social().SendAdoptionRequest(ctx, action.ActorID, action.Name)
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
	if !m.mapPointerBlocked(ctx) && !m.ui.escapeMenu.IsOpen() && !m.ui.interactionModalOpen() && !m.ui.settingsWindow.IsOpen() && !m.ui.identifyWindow.IsOpen() && !m.ui.pet.Egg.IsOpen() && !m.ui.pet.Confirm.IsOpen() && !m.ui.companions.HomunculusConfirm.IsOpen() && !m.ui.companions.MercenaryConfirm.IsOpen() {
		m.updateCameraRotation(ctx)
	}
	if !dead && m.ui.escapeMenu.IsOpen() {
		if m.ui.escapeMenu.Update(ctx) {
			m.handleEscapeMenuAction(ctx)
			return nil, nil
		}
	}
	if m.ui.social.FriendRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.FriendConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.PartyRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.GuildRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.GuildAllianceRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.GuildRelationConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.PartyInfo.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.TradeRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.AdoptionRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.starPlaceConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.pet.Confirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.pet.Info.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusInfo.Update(ctx) {
		if action := m.ui.companions.HomunculusInfo.PopAction(); action.Kind != gameui.HomunculusInfoActionNone {
			m.companions().HandleHomunculusInfoAction(ctx, action)
		}
		return nil, nil
	}
	if m.ui.companions.MercenaryConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.MercenaryInfo.Update(ctx) {
		if action := m.ui.companions.MercenaryInfo.PopAction(); action.Kind != gameui.MercenaryInfoActionNone {
			m.companions().HandleMercenaryInfoAction(ctx, action)
		}
		return nil, nil
	}
	if m.ui.teleportModal.Update(ctx) {
		return nil, nil
	}
	if m.battle().UpdateAutoSpellWindow(ctx) {
		return nil, nil
	}
	if m.ui.npcInputAvailable() && m.ui.npc.Dialog.Update(ctx) {
		return nil, nil
	}
	if m.ui.npc.Cutin.Update(ctx) {
		return nil, nil
	}
	if m.social().UpdateWhisperWindow(ctx) {
		return nil, nil
	}
	if m.social().UpdateChatRoomWindows(ctx) {
		return nil, nil
	}
	if m.social().UpdatePartyHelperWindows(ctx) {
		return nil, nil
	}
	if m.skills().UpdateSkillTextPrompt(ctx) {
		return nil, nil
	}
	if m.social().UpdateGuildMemberPrompt(ctx) {
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
	// Let the atlas close on Escape before the console handles that key.
	if m.ui.worldMap.Update(ctx) {
		return nil, nil
	}
	if !dead && !m.ui.chatShortcuts.KeyboardShortcutsBlocked() && m.ui.console.UpdateInput(ctx) {
		return nil, nil
	}
	if m.ui.chatShortcuts.Update(ctx) {
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
	if consumed, err := m.ui.itemWindows.Update(ctx, m); consumed {
		if err != nil {
			m.ui.console.AddErrorMessage("Unable to display this item.")
			glog.Warnf("item window: %v", err)
		}
		return nil, nil
	}
	if m.ui.battle.MonsterInfo.Update(ctx) {
		return nil, nil
	}
	if m.ui.identifyWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.cardWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.pet.Egg.Update(ctx) {
		return nil, nil
	}
	if m.ui.inventoryBag.UpdateDrag(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.social.TradeWindow, &m.ui.equipmentWindow, &m.ui.social.MailWindow) {
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
	if m.ui.companions.HomunculusSkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.companions.MercenarySkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.social.GuildWindow.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.shortcutBar.Update(ctx, m) {
		return nil, nil
	}
	if m.ui.social.MailWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.inventoryBag.Update(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.social.TradeWindow, &m.ui.equipmentWindow, &m.ui.itemWindows, &m.ui.social.MailWindow) {
		return nil, nil
	}
	if m.ui.social.TradeWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.equipmentWindow.Update(ctx, &m.ui.itemWindows, &m.ui.cartWindow, m) {
		return nil, nil
	}
	if m.ui.viewEquipWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.storageWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.cartWindow, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.cartWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.storageWindow, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.changeCartWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.npc.Shop.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.social.VendingWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.skillWindow.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.questWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusSkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.companions.MercenarySkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.emoteWindow.Update(ctx, &m.ui.console) {
		return nil, nil
	}
	if m.social().UpdateFriendsWindow(ctx) {
		return nil, nil
	}
	if m.social().UpdateGuildWindow(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.social.FriendSettings.Update(ctx) {
		return nil, nil
	}
	if m.ui.social.PartySettings.Update(ctx) {
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
	m.effects().SyncLevel99AuraEffects(ctx, now)
	pointerBlocked := minimapDragging || m.mapPointerBlocked(ctx)
	if !pointerBlocked {
		m.updateCameraZoom(ctx)
	}
	if dead {
		return nil, nil
	}
	m.companions().Update(ctx, m.actorSystem.ActorDeaths, now)
	m.updateBot(ctx, now)

	leftClick := !pointerBlocked && ctx.Input.MouseJustPressed(input.MouseButtonLeft)
	if leftClick {
		// One-shot interactions must not be delayed by the short throttle used
		// for repeated walk requests. Scheduling the held-click repeat here also
		// prevents a throttled ground click from turning into a walk on the next
		// frame merely because the button is still down.
		m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)
	}
	if leftClick && m.skillSystem.Pending.Skill.ID != 0 {
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
		playerX, playerY := scene.CurrentPlayerCell(ctx, now)
		if actor, ok := m.actors().HoveredVendingBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click vending target mouse=%d,%d id=%d name=%q shop=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.VendingName, playerX, playerY, actor.X, actor.Y)
			m.social().RequestVendingList(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := m.actors().HoveredChatRoomBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click chat room target mouse=%d,%d id=%d name=%q room=%d title=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.ChatRoomID, actor.ChatRoomTitle, playerX, playerY, actor.X, actor.Y)
			m.social().RequestChatRoomEnter(ctx, actor)
			return nil, nil
		}
		if item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click pickup target mouse=%d,%d id=%d item_id=%d amount=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, item.ID, item.ItemID, item.Amount, playerX, playerY, item.X, item.Y)
			m.clearLockedAttack()
			m.clearAttackFocus()
			m.requestPickup(ctx, item, "click")
			return nil, nil
		}
		if actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths); ok && npc.IsWarpActor(actor) {
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
		if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths); ok {
			glog.Debugf("click attack target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.requestAttack(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths); ok {
			glog.Debugf("click npc talk target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.clearAttackFocus()
			m.requestNPCTalk(ctx, actor, "click")
			return nil, nil
		}
		if targetX, targetY, ok := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok && m.walkReady(now) {
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
	case gameui.EscapeMenuActionAutoRevive:
		m.ui.escapeMenu.RequestAutoRevive(ctx)
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
	message := login.MessageText(ctx.Resources, 502, "You cannot exit the game right now.")
	m.ui.console.AddErrorMessage("%s", message)
}

func (m *WorldMode) openEscapeMenuFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.escapeMenu.IsOpen() || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	if m.ui.interactionModalOpen() {
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
		OnComm:   func() { m.ui.social.ChatRoomCreate.Open(ctx) },
		OnFriend: func() { m.ui.social.FriendsWindow.Toggle(ctx) },
	}
}

func (m *WorldMode) toggleEmoteWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	if !plainAltDown(ctx.Input) || !ctx.Input.JustPressed(input.KeyL) {
		return false
	}
	m.ui.emoteWindow.Toggle(ctx, &m.ui.console)
	return true
}

func (m *WorldMode) toggleGuildWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	if !plainAltDown(ctx.Input) || !ctx.Input.JustPressed(input.KeyG) {
		return false
	}
	m.social().ToggleGuildWindow(ctx)
	return true
}

func (m *WorldMode) handleMapChange(ctx client.Context, change network.MapChange) Mode {
	if m.ui.social.MailWindow.IsOpen() || m.socialSystem.Mail.Pending.Kind != gameui.MailActionNone {
		m.social().CloseMail(ctx)
	}
	m.npc().ClearProgress()
	m.npcSystem.ShowDigit = npc.ShowDigitState{}
	m.ui.minimap.ClearBossMarker()
	m.ui.poptips.Clear()
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.clearLocalActorAction(ctx)
	m.scheduledStops = nil
	m.ui.npc.Dialog.ResetPublished(ctx)
	m.ui.npc.Cutin.Clear()
	m.ui.teleportModal = gameui.TeleportModal{}
	m.ui.battle.AutoSpell.Reset(ctx)
	if m.ui.starPlaceConfirm.IsOpen() {
		m.ui.starPlaceConfirm.Close(ctx)
	}
	m.clearLocalDeathState(ctx)
	currentMap := ctx.World.MapName
	reuseLoadedMap := !change.ServerMove && sameLoadedMap(ctx, change.MapName)
	glog.Debugf("map change current=%s target=%s x=%d y=%d server_move=%t addr=%s port=%d reuse_loaded=%t", currentMap, change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port, reuseLoadedMap)
	ctx.World.ResetMapProperty()
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	actors.ApplyWarpPosition(ctx, change.X, change.Y)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	if reuseLoadedMap {
		m.camera.ResetTracking()
		m.camera.Update(ctx, time.Now())
		if ctx.Network != nil {
			if err := ctx.Network.SendLoadEndAck(); err != nil {
				glog.Warnf("same-map warp load ack failed map=%s x=%d y=%d: %v", change.MapName, change.X, change.Y, err)
			}
		}
		return nil
	}
	if change.ServerMove {
		m.socialSystem.Mail = social.MailState{}
		m.deferredPackets = nil
		ctx.Session.Zone.Address = change.Address
		ctx.Session.Zone.Port = change.Port
		dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := ctx.Network.Connect(dialCtx, change.Address, int(change.Port))
		cancel()
		if err != nil {
			glog.Warnf("map reconnect failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			login.OpenConnectionFailedDialog(ctx, &m.ui.disconnectDialog)
			return nil
		}
		if err := ctx.Network.SendMapServerEnter(ctx.Session.AccountID, ctx.Session.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex); err != nil {
			glog.Warnf("map re-enter failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			message := login.MessageText(ctx.Resources, 2, "Disconnected from Server.")
			login.OpenDisconnectDialog(ctx, &m.ui.disconnectDialog, message)
			return nil
		}
		m.pendingWarp = true
		return nil
	}
	return m.nextWorldMode()
}

func (m *WorldMode) handleLevelUpNotificationAction(ctx client.Context, action gameui.LevelUpNotificationAction) bool {
	if action&gameui.LevelUpNotificationBase != 0 {
		m.ui.statsWindow.OpenWindow(ctx)
	}
	if action&gameui.LevelUpNotificationJob != 0 {
		m.ui.skillWindow.OpenWindow(ctx)
	}
	return action != gameui.LevelUpNotificationNone
}

func (m *WorldMode) nextWorldMode() *WorldMode {
	next := NewWorldMode()
	next.socialSystem.Mail = m.socialSystem.Mail
	next.deferredPackets = m.deferredPackets
	next.camera.yawOffset = m.camera.yawOffset
	next.camera.pitch = m.camera.pitch
	next.camera.zoom = m.camera.zoom
	next.camera.zoomTarget = m.camera.zoomTarget
	next.ui.console = m.ui.console
	next.ui.preferences = m.ui.preferences
	next.ui.characterWindow = m.ui.characterWindow
	next.ui.basicMenu = m.ui.basicMenu
	next.ui.inventoryBag = m.ui.inventoryBag
	next.ui.equipmentWindow = m.ui.equipmentWindow
	next.ui.cartWindow = m.ui.cartWindow
	next.ui.itemWindows = m.ui.itemWindows
	next.ui.cardWindow = m.ui.cardWindow
	next.ui.pet.Egg = m.ui.pet.Egg
	next.ui.pet.Info = m.ui.pet.Info
	next.ui.companions.HomunculusInfo = m.ui.companions.HomunculusInfo
	next.ui.companions.HomunculusSkill = m.ui.companions.HomunculusSkill
	next.ui.companions.MercenaryInfo = m.ui.companions.MercenaryInfo
	next.ui.companions.MercenarySkill = m.ui.companions.MercenarySkill
	next.petSystem.Property = m.petSystem.Property
	next.petSystem.HasProperty = m.petSystem.HasProperty
	next.petSystem.OldFullness = m.petSystem.OldFullness
	next.petSystem.LastTalk = m.petSystem.LastTalk
	next.ui.statsWindow = m.ui.statsWindow
	next.ui.skillWindow = m.ui.skillWindow
	next.ui.questWindow = m.ui.questWindow
	next.ui.worldMap = m.ui.worldMap
	next.ui.emoteWindow = m.ui.emoteWindow
	next.ui.chatShortcuts = m.ui.chatShortcuts
	next.ui.social.FriendsWindow = m.ui.social.FriendsWindow
	next.ui.social.GuildWindow = m.ui.social.GuildWindow
	next.ui.social.FriendSettings = m.ui.social.FriendSettings
	next.ui.social.WhisperWindows = m.ui.social.WhisperWindows
	next.ui.social.ChatRoomCreate = m.ui.social.ChatRoomCreate
	next.ui.social.ChatRoom = m.ui.social.ChatRoom
	next.socialSystem.PendingChatRoom = m.socialSystem.PendingChatRoom
	next.ui.social.PartySettings = m.ui.social.PartySettings
	next.ui.settingsWindow = m.ui.settingsWindow
	next.ui.social.PartyCreate = m.ui.social.PartyCreate
	next.ui.social.PartyInvite = m.ui.social.PartyInvite
	next.ui.shortcutBar = m.ui.shortcutBar
	next.ui.minimap = m.ui.minimap
	next.ui.battle.StatusIcons = m.ui.battle.StatusIcons
	next.ui.pvpCounter = m.ui.pvpCounter
	next.ui.levelUpNotifications = m.ui.levelUpNotifications
	m.companionSystem.Close()
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

func (m *WorldMode) Draw(ctx client.Context, screen *render.Frame) {
	if m.mapScene.Images.PrepareUploads(screen) {
		maps.ClearWorldScene(screen, ctx.World.MapName)
		return
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	battle.ApplyHallucination(screen, ctx.Session, now)
	projection := m.sceneProjection(ctx, width, height, now)
	fogConfig := ctx.Config.Fog
	if ctx.Session != nil {
		fogConfig.Enabled = ctx.Session.FogEnabled
	}
	fog := weather.FogFromMap(ctx.Resources, ctx.World.MapName, fogConfig)
	maps.ClearWorldScene(screen, ctx.World.MapName)
	var actorOverlays []scene.ActorDrawEntry
	screen.SetCamera3D(projection.RenderCameraWithFog(fog))
	vertexFog := scene.SceneFog{}

	if ctx.World.GND != nil {
		m.mapScene.DrawGNDMeshes(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection)
		m.mapScene.DrawGNDWater(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection, now, vertexFog)
		m.skills().DrawGraffiti(screen, ctx, projection)
		if !ctx.Config.Render.NoUI {
			m.drawTileCursor(screen, ctx, projection)
		}
		if ctx.World.RSW != nil && len(ctx.World.RSM) > 0 {
			actorOverlays = m.drawSceneModelsAndActors(screen, ctx, projection, now)
		} else {
			m.skills().DrawSkillUnitRSMModels(screen, ctx, projection, now)
			m.effects().Draw(screen, ctx, projection, now, true)
			m.drawGroundItems(screen, ctx, projection, now)
			actorOverlays = m.actors().DrawSceneActors(screen, ctx, projection)
		}
	} else if ctx.World.GAT != nil {
		m.effects().Draw(screen, ctx, projection, now, true)
		m.drawGroundItems(screen, ctx, projection, now)
		actorOverlays = m.actors().DrawSceneActors(screen, ctx, projection)
	} else {
		m.effects().Draw(screen, ctx, projection, now, true)
	}

	if !ctx.Config.Render.NoUI {
		m.drawSceneActorOverlays(screen, ctx, projection, now, actorOverlays)
	}
	m.mapScene.DrawRSWEffects(screen, ctx, projection, now, m.effects)
	m.weather.Draw(screen, ctx, projection, now, m.effects(), m.scheduleSound)
	m.effects().Draw(screen, ctx, projection, now, false)
	m.battle().DrawDamageFloaters(screen, ctx, projection, now)

	if !ctx.Config.Render.NoUI {
		m.ui.npc.Cutin.Draw(screen)
		m.ui.inventoryBag.Publish(ctx)
		m.ui.storageWindow.Publish(ctx)
		m.ui.cartWindow.Publish(ctx)
		m.ui.npc.Shop.Publish(ctx)
		m.ui.social.VendingWindow.Publish(ctx)
		m.ui.skillWindow.Publish(ctx)
		m.ui.companions.HomunculusSkill.Publish(ctx)
		m.ui.companions.MercenarySkill.Publish(ctx)
		m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
	}
}

func (m *WorldMode) DrawOverlay(ctx client.Context, screen *render.Frame) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	m.mapFade.Draw(screen, now)
	if !ctx.Config.Render.NoUI {
		m.drawUIDragGhosts(screen, ctx)
		m.pets().DrawSlotMachine(screen, ctx, now)
		m.drawROCursor(screen, ctx, projection, now)
	}
}

func (m *WorldMode) FrameSubmitted() {
	if m.mapScene.Images.FrameSubmitted() {
		return
	}
	m.mapFade.RecordCoveredFrame()
}

func (m *WorldMode) DrawUIOverlay(ctx client.Context, screen *render.Frame) {
	if ctx.Config.Render.NoUI {
		return
	}
	now := time.Now()
	m.npc().DrawShowDigit(screen, ctx, now)
	m.ui.announcement.Draw(screen, now)
	m.drawPendingTargetBanner(screen, ctx)
	m.ui.poptips.Draw(screen, now)
	m.ui.inventoryBag.DrawTooltip(ctx, screen)
	m.ui.equipmentWindow.DrawTooltip(ctx, screen)
	m.ui.cartWindow.DrawTooltip(ctx, screen)
	m.ui.itemWindows.DrawTooltip(ctx, screen)
	m.ui.skillWindow.DrawTooltip(ctx, screen)
	m.ui.companions.HomunculusSkill.DrawTooltip(ctx, screen)
	m.ui.companions.MercenarySkill.DrawTooltip(ctx, screen)
	m.ui.social.GuildWindow.DrawTooltip(ctx, screen)
	m.ui.shortcutBar.DrawTooltip(ctx, screen)
	m.ui.itemPickup.Draw(screen, ctx, m, now)
}

func (m *WorldMode) drawUIDragGhosts(screen *render.Frame, ctx client.Context) {
	m.ui.inventoryBag.DrawDragGhost(screen, ctx, m)
	m.ui.storageWindow.DrawDragGhost(screen, ctx, m)
	m.ui.cartWindow.DrawDragGhost(screen, ctx, m)
	m.ui.npc.Shop.DrawDragGhost(screen, ctx, m)
	m.ui.social.VendingWindow.DrawDragGhost(screen, ctx, m)
	m.ui.skillWindow.DrawDragGhost(screen, ctx, m)
	m.ui.companions.HomunculusSkill.DrawDragGhost(screen, ctx, m)
	m.ui.companions.MercenarySkill.DrawDragGhost(screen, ctx, m)
	m.ui.social.GuildWindow.DrawDragGhost(screen, ctx, m)
}

func clickedAttackTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actors.ActorCanBeAttackClicked(ctx, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !actors.PointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedSkillTarget(ctx client.Context, projection scene.Projection, skill session.Skill, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range skillTargetCandidates(ctx, skill) {
		if _, dead := deadActors[actor.ID]; dead && !actors.SkillCanTargetDeadActor(skill, actor) {
			continue
		}
		if !actors.ActorCanBeSkillTargeted(ctx, skill, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !actors.PointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedPlayerTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actors.ActorCanOpenPlayerContext(ctx, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !actors.PointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
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
		if actor, ok, _ := scene.ActorForCombatID(ctx, scene.LocalSkillTarget(ctx)); ok {
			candidates = append(candidates, actor)
		}
	}
	for _, actor := range ctx.World.Actors {
		candidates = append(candidates, actor)
	}
	return candidates
}

func skillTargetFlagsIncludeSelfPick(flags uint32) bool {
	return flags&(skills.SkillTargetFriend|skills.SkillTargetSelf) != 0
}

type sceneDrawEntry struct {
	depth           float64
	entityID        uint32
	actorIndex      int
	shadowIndex     int
	itemIndex       int
	itemShadowIndex int
}

func (m *WorldMode) drawSceneModelsAndActors(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) []scene.ActorDrawEntry {
	m.mapScene.DrawRSMModels(screen, ctx.Resources, ctx.World.RSW, ctx.World.RSM, ctx.World.GND, projection, now)
	m.skills().DrawSkillUnitRSMModels(screen, ctx, projection, now)
	m.effects().Draw(screen, ctx, projection, now, true)
	actors := m.actors().CollectSceneActorEntries(screen, ctx, projection)
	items := m.collectSceneItemEntries(screen, ctx, projection, now)
	for _, entry := range sortedSceneDrawEntries(actors, items) {
		if entry.itemShadowIndex >= 0 {
			m.drawGroundItemShadowEntry3D(screen, projection, items[entry.itemShadowIndex])
			continue
		}
		if entry.shadowIndex >= 0 {
			m.actors().DrawActorShadowEntry(screen, ctx, projection, actors[entry.shadowIndex])
			continue
		}
		if entry.itemIndex >= 0 {
			m.drawGroundItemEntry3D(screen, projection, items[entry.itemIndex])
			continue
		}
		m.actors().DrawSceneActorEntry(screen, ctx, projection, actors[entry.actorIndex])
	}
	m.actors().DrawSceneActorFalcons(screen, ctx, projection, actors)
	return actors
}

func sortedSceneDrawEntries(actors []scene.ActorDrawEntry, items []sceneItemDrawEntry) []sceneDrawEntry {
	entries := make([]sceneDrawEntry, 0, len(actors)*2+len(items)*2)
	for i, item := range items {
		entries = append(entries,
			sceneDrawEntry{depth: item.shadowDepth, entityID: item.item.ID, actorIndex: -1, shadowIndex: -1, itemIndex: -1, itemShadowIndex: i},
			sceneDrawEntry{depth: item.depth, entityID: item.item.ID, actorIndex: -1, shadowIndex: -1, itemIndex: i, itemShadowIndex: -1},
		)
	}
	for i, actor := range actors {
		if actor.CastShadow {
			entries = append(entries, sceneDrawEntry{depth: actor.ShadowDepth, entityID: actor.Actor.ID, actorIndex: -1, shadowIndex: i, itemIndex: -1, itemShadowIndex: -1})
		}
		entries = append(entries, sceneDrawEntry{depth: actor.Depth, entityID: actor.Actor.ID, actorIndex: i, shadowIndex: -1, itemIndex: -1, itemShadowIndex: -1})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		// A stable sort alone preserves the world's random map iteration order
		// at equal depths, making overlapping loot flicker from frame to frame.
		if entries[i].depth == entries[j].depth {
			return entries[i].entityID < entries[j].entityID
		}
		return entries[i].depth > entries[j].depth
	})
	return entries
}

func (m *WorldMode) releaseMapTextures() {
	for id := range m.skillSystem.Graffiti {
		m.skills().RemoveGraffiti(id)
	}
	m.skillSystem.Graffiti = nil
	if m.spriteUpscaler != nil {
		m.spriteUpscaler.Close()
		m.spriteUpscaler = nil
	}
	m.mapScene.Images.Release()
}

// Leave also covers disconnects, character selection, and failed map loads.
func (m *WorldMode) Leave() {
	m.social().CancelChatRoomRequests()
	m.releaseMapTextures()
}

func (m *WorldMode) startMapFadeOut(change network.MapChange, now time.Time) {
	m.ui.npc.Cutin.Clear()
	m.social().CancelChatRoomRequests()
	// A queued cell (or message) belongs to the map where it was selected,
	// even when the server warps us within the same loaded map.
	m.skillSystem.Pending = skills.PendingTarget{}
	m.skillSystem.PendingText = skills.PendingTextTarget{}
	if m.ui.skillTextPrompt.IsOpen() {
		m.ui.skillTextPrompt.Close()
	}
	m.mapFade.StartOut(change, now)
}

func (m *WorldMode) startCharacterSelectFadeOut(now time.Time) {
	m.ui.npc.Cutin.Clear()
	m.mapFade.StartCharacterSelectOut(now)
}
