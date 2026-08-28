package game

import (
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func actorBillboardScreenScale(projection sceneProjection, x, y, z float64) float64 {
	_, _, worldUnitsPerScreenPixel, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return 1
	}
	scale := actorBillboardWorldUnitsPerSourcePixel / worldUnitsPerScreenPixel
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1
	}
	return scale
}

// robr stores entity positions as cells and adds +0.5 in SpriteRenderer.vs.
// Goro projects world coordinates directly, so actor-like entities use cell centers.
func cellWorldAnchor(x, y float64) (float64, float64) {
	return cellCenter(x), cellCenter(y)
}

func actorWorldAnchor(_ worldstate.Actor, x, y float64) (float64, float64) {
	return cellWorldAnchor(x, y)
}

func pointInActorPickBounds(mouseX, mouseY, centerX, centerY, scale float64) bool {
	scale = normalizePickScale(scale)
	left := centerX - 44*scale
	right := centerX + 44*scale
	top := centerY - float64(humanoidBillboardAnchorY)*scale
	bottom := centerY + 20*scale
	return mouseX >= left && mouseX <= right && mouseY >= top && mouseY <= bottom
}

func actorPickBoundsCenter(centerX, centerY, scale float64) (float64, float64) {
	scale = normalizePickScale(scale)
	top := centerY - float64(humanoidBillboardAnchorY)*scale
	bottom := centerY + 20*scale
	return centerX, (top + bottom) / 2
}

func normalizePickScale(scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1
	}
	return scale
}

func actorCanBeSkillTargeted(ctx client.Context, skill session.Skill, actor worldstate.Actor) bool {
	if actor.ID == 0 || isWarpActor(actor) {
		return false
	}
	targetFlags, ok := skillTargetFlagsForActor(ctx, actor)
	if !ok {
		return false
	}
	if skill.Type == skillTargetPet {
		return targetFlags&skillTargetPet != 0
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

func skillCanTargetDeadActor(skill session.Skill, actor worldstate.Actor) bool {
	return skill.ID == db.SkillALLResurrection && actorRepresentsPlayer(actor)
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
	if ctx.World == nil {
		return false
	}
	if actorRepresentsPlayer(actor) {
		return actorCanBePlayerCombatTarget(ctx, actor)
	}
	return ctx.World.MapProperty.IsGvG() && actorCanBeEnemyHomunculus(ctx, actor)
}

func actorCanOpenPlayerContext(ctx client.Context, actor worldstate.Actor) bool {
	if isLocalActor(ctx, actor.ID) || strings.TrimSpace(actor.Name) == "" {
		return false
	}
	return actorRepresentsPlayer(actor)
}

func actorCanBeAttackClicked(ctx client.Context, actor worldstate.Actor) bool {
	if actor.ID == 0 || isLocalActor(ctx, actor.ID) {
		return false
	}
	if actorRepresentsPlayer(actor) {
		return actorCanBePlayerCombatTarget(ctx, actor)
	}
	if actorCanBeEnemyHomunculus(ctx, actor) {
		return true
	}
	return actorHasMobObjectType(actor)
}

func actorCanBePlayerCombatTarget(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.World == nil || !actorRepresentsPlayer(actor) || isLocalActor(ctx, actor.ID) {
		return false
	}
	if ctx.World.MapProperty.IsPvP() {
		return true
	}
	if !ctx.World.MapProperty.IsGvG() {
		return false
	}
	return actor.GuildID == 0 || localGuildID(ctx) == 0 || actor.GuildID != localGuildID(ctx)
}

func actorCanBeEnemyHomunculus(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.World == nil || !ctx.World.MapProperty.IsGvG() || !actor.HasObjectType || actor.ObjectType != actorObjectTypeHomunculus {
		return false
	}
	if ctx.Session != nil && ctx.Session.Homunculus.ID != 0 && actor.ID == ctx.Session.Homunculus.ID {
		return false
	}
	return actor.GuildID == 0 || localGuildID(ctx) == 0 || actor.GuildID != localGuildID(ctx)
}

func localGuildID(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.Guild.ID != 0 {
		return ctx.Session.Guild.ID
	}
	return ctx.Session.GuildID
}

func actorRepresentsPlayer(actor worldstate.Actor) bool {
	if actor.HasObjectType {
		return actor.ObjectType == actorObjectTypePC
	}
	// The 2008 PC-only 0x02ED/0x02EE entries omit the object-type byte.
	// Require their full appearance payload so a move-only update with the
	// zero-value Novice job cannot be mistaken for a player.
	return actor.Appearance && res.HasPlayerJobToken(int(actor.Job))
}

func upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if isLocalActor(ctx, entry.ID) {
		return
	}
	name := companionActorEntryName(ctx, entry)
	applyCompanionActorEntry(ctx, entry)
	dir := entry.Dir
	if entry.Moving {
		dir = directionFromDelta(entry.FromX, entry.FromY, entry.ToX, entry.ToY, dir)
	}
	attackRange := 0
	if ctx.Session != nil {
		switch entry.ObjectType {
		case actorObjectTypeHomunculus:
			attackRange = ctx.Session.Homunculus.AttackRange
		case actorObjectTypeMercenary:
			attackRange = ctx.Session.Mercenary.AttackRange
		}
	}
	actor := worldstate.Actor{
		ID:               entry.ID,
		Name:             name,
		GuildID:          entry.GuildID,
		EmblemVersion:    entry.EmblemVersion,
		X:                entry.X,
		Y:                entry.Y,
		Dir:              dir,
		Job:              entry.Job,
		Head:             entry.Head,
		Weapon:           entry.Weapon,
		Shield:           entry.Shield,
		HeadTop:          entry.HeadTop,
		HeadMid:          entry.HeadMid,
		HeadLow:          entry.HeadLow,
		HeadPal:          entry.HeadPal,
		BodyPal:          entry.BodyPal,
		Sex:              entry.Sex,
		HeadDir:          entry.HeadDir,
		IsAdmin:          actorIsAdmin(ctx, entry.ID),
		Appearance:       entry.Appearance,
		Moving:           entry.Moving,
		FromX:            entry.FromX,
		FromY:            entry.FromY,
		ToX:              entry.ToX,
		ToY:              entry.ToY,
		MoveStartTick:    entry.MoveStartTick,
		HasMoveStartTick: entry.HasMoveStartTick,
		ObjectType:       entry.ObjectType,
		HasObjectType:    entry.HasObjectType,
		Speed:            entry.Speed,
		AttackRange:      attackRange,
		BodyState:        entry.BodyState,
		HealthState:      entry.HealthState,
		EffectState:      entry.EffectState,
		HasState:         entry.HasState,
		Level:            entry.Level,
		HasLevel:         entry.HasLevel,
	}
	if existing, ok := ctx.World.Actors[entry.ID]; ok {
		actor.Opt3State = existing.Opt3State
		if !actor.HasObjectType {
			actor.ObjectType = existing.ObjectType
			actor.HasObjectType = existing.HasObjectType
		}
		if actor.AttackRange <= 0 {
			actor.AttackRange = existing.AttackRange
		}
	}
	applyActorCartStateFromEffect(&actor)
	upsertActor(ctx, actor)
}

func (m *WorldMode) upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if ctx.World == nil {
		return
	}
	m.clearActorVanish(entry.ID)
	oldState := uint32(0)
	if existing, ok := ctx.World.Actors[entry.ID]; ok && existing.HasState {
		oldState = existing.EffectState
	}
	if entry.Moving {
		m.clearActorAction(ctx, entry.ID)
	}
	upsertNetworkActor(ctx, entry)
	m.syncActorEntryLevel99Aura(ctx, entry.ID)
	m.requestActorGuildEmblem(ctx, entry.GuildID, entry.EmblemVersion)
	if !entry.HasState {
		return
	}
	actor, ok := ctx.World.Actors[entry.ID]
	if !ok || !actor.HasState {
		return
	}
	m.applyActorEffectStateEffects(ctx, actor.ID, oldState, actor.EffectState)
}

func (m *WorldMode) applyWarpPortalEntry(ctx client.Context, entry network.ActorEntry) {
	effectID, ok := warpPortalActorEffectID(entry.Job)
	if !ok {
		return
	}
	m.removeOtherWarpPortalActorEffects(entry.ID, effectID)
	m.addWorldEffectBetweenAtDurationIfMissing(ctx, effectID, entry.ID, 0, time.Now(), warpPortalActorEffectLifetime)
}

func warpPortalActorEffectID(job int16) (int, bool) {
	switch int(job) {
	case actorJobWarpPortal, actorJobWarpPortalActive:
		return effectWarpZone2, true
	case actorJobWarpPortalWaiting:
		return effectReadyPortal, true
	default:
		return 0, false
	}
}

func (m *WorldMode) removeOtherWarpPortalActorEffects(actorID uint32, keepEffectID int) {
	for _, effectID := range []int{effectWarpZone2, effectReadyPortal, effectPortal} {
		if effectID != keepEffectID {
			m.removeWorldEffect(effectID, actorID)
		}
	}
}

const (
	actorVanishOutOfSight = 0
	actorVanishDeath      = 1
	actorVanishLogout     = 2
	actorVanishTeleport   = 3
)

const (
	actorVanishOutOfSightFadeDuration = time.Second
	warpPortalActorEffectLifetime     = 24 * time.Hour
)

type actorVanishFade struct {
	started  time.Time
	removeAt time.Time
}

func (m *WorldMode) applyActorVanish(ctx client.Context, vanish network.ActorVanish) {
	glog.Debugf("actor vanish id=%d reason=%d", vanish.ID, vanish.Reason)
	pendingHomunculusDelete := m.homDeleteID != 0 && vanish.ID == m.homDeleteID
	pendingMercenaryDelete := m.mercDeleteID != 0 && vanish.ID == m.mercDeleteID
	if m.attackFocusID == vanish.ID {
		m.clearAttackFocus()
	}
	if vanish.Reason == actorVanishDeath {
		m.clearActorVanish(vanish.ID)
		m.startActorDeath(ctx, vanish.ID)
		return
	}
	if vanish.Reason == actorVanishOutOfSight && !pendingHomunculusDelete && !pendingMercenaryDelete {
		if m.startActorOutOfSightVanish(ctx, vanish.ID) {
			return
		}
	}
	m.addActorVanishTeleportEffect(ctx, vanish)
	m.removeActorEffectStateEffects(vanish.ID)
	m.removeLevel99AuraEffects(vanish.ID)
	m.removeActorNow(ctx, vanish.ID)
	if pendingHomunculusDelete {
		m.clearDeletedHomunculus(ctx)
	}
	if pendingMercenaryDelete {
		m.clearDeletedMercenary(ctx)
	}
}

func (m *WorldMode) startActorOutOfSightVanish(ctx client.Context, id uint32) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return false
	}
	now := time.Now()
	actor = freezeActorAtRenderedPosition(actor, now)
	ctx.World.Actors[id] = actor
	if m.actorVanishes == nil {
		m.actorVanishes = make(map[uint32]actorVanishFade)
	}
	m.actorVanishes[id] = actorVanishFade{
		started:  now,
		removeAt: now.Add(actorVanishOutOfSightFadeDuration),
	}
	m.removeActorEffectStateEffects(id)
	m.removeLevel99AuraEffects(id)
	delete(m.actorLife, id)
	delete(m.actorCastBars, id)
	delete(m.speechBubbles, id)
	glog.Debugf("actor out-of-sight fade id=%d duration_ms=%d", id, actorVanishOutOfSightFadeDuration.Milliseconds())
	return true
}

func freezeActorAtRenderedPosition(actor worldstate.Actor, now time.Time) worldstate.Actor {
	x, y := actorRenderPosition(actor, now)
	actor.X = int(math.Round(x))
	actor.Y = int(math.Round(y))
	actor.Moving = false
	actor.FromX = actor.X
	actor.FromY = actor.Y
	actor.ToX = actor.X
	actor.ToY = actor.Y
	actor.MovePath = nil
	actor.HasMoveStart = false
	actor.MoveStartX = 0
	actor.MoveStartY = 0
	actor.WalkDistance = 0
	return actor
}

func (m *WorldMode) removeActorNow(ctx client.Context, id uint32) {
	if ctx.World != nil {
		ctx.World.RemoveActor(id)
	}
	m.clearRemovedActorState(id)
}

func (m *WorldMode) clearRemovedActorState(id uint32) {
	m.removeWorldEffectsForActor(id)
	if m.scriptHighlight.id == id {
		m.clearScriptHighlight()
	}
	delete(m.actorAnims, id)
	delete(m.actorDeaths, id)
	delete(m.actorVanishes, id)
	delete(m.actorSoundFrames, id)
	delete(m.actorLife, id)
	delete(m.actorCastBars, id)
	delete(m.speechBubbles, id)
	delete(m.petAccessoryIDs, id)
}

func (m *WorldMode) clearActorVanish(id uint32) {
	delete(m.actorVanishes, id)
}

func (m *WorldMode) addActorVanishTeleportEffect(ctx client.Context, vanish network.ActorVanish) {
	if vanish.Reason != actorVanishLogout && vanish.Reason != actorVanishTeleport {
		return
	}
	if ctx.World == nil {
		return
	}
	now := time.Now()
	if isLocalActor(ctx, vanish.ID) {
		x, y := actorRenderPosition(ctx.World.Player, now)
		m.addWorldEffectAtCellDurationSize(ctx, effectTeleportation, 0, int(math.Round(x)), int(math.Round(y)), now, 0, 0)
		return
	}
	actor, ok := ctx.World.Actors[vanish.ID]
	if !ok {
		return
	}
	x, y := actorRenderPosition(actor, now)
	m.addWorldEffectAtCellDurationSize(ctx, effectTeleportation, 0, int(math.Round(x)), int(math.Round(y)), now, 0, 0)
}

func (m *WorldMode) cleanupDeadActors(ctx client.Context, now time.Time) {
	if len(m.actorDeaths) == 0 || ctx.World == nil {
		return
	}
	for id, removeAt := range m.actorDeaths {
		// Dead player characters remain in the world so resurrection skills can
		// target them. Their zero deadline is cleared by ZC_RESURRECTION, an
		// actor refresh, or a later non-death vanish packet.
		if removeAt.IsZero() {
			continue
		}
		if now.Before(removeAt) {
			continue
		}
		ctx.World.RemoveActor(id)
		m.removeActorEffectStateEffects(id)
		m.removeLevel99AuraEffects(id)
		delete(m.actorDeaths, id)
		delete(m.actorAnims, id)
		delete(m.actorSoundFrames, id)
		delete(m.actorLife, id)
		delete(m.speechBubbles, id)
		delete(m.petAccessoryIDs, id)
		if m.pendingAttack.targetID == id {
			m.pendingAttack = attackIntent{}
		}
		if m.lockedAttackID == id {
			m.clearLockedAttack()
		}
		if m.attackFocusID == id {
			m.clearAttackFocus()
		}
		if m.scriptHighlight.id == id {
			m.clearScriptHighlight()
		}
		glog.Debugf("actor death removed id=%d", id)
	}
}

func (m *WorldMode) cleanupVanishedActors(ctx client.Context, now time.Time) {
	if len(m.actorVanishes) == 0 {
		return
	}
	for id, fade := range m.actorVanishes {
		if now.Before(fade.removeAt) {
			continue
		}
		m.removeActorNow(ctx, id)
		glog.Debugf("actor out-of-sight removed id=%d", id)
	}
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
	upsertActor(ctx, actor)
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
			glog.Debugf("ignored mob look-base player job id=%d old_job=%d value=%d", actor.ID, actor.Job, look.Value)
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

func isLocalActor(ctx client.Context, id uint32) bool {
	return ctx.Session != nil && id != 0 && (id == ctx.Session.AccountID || id == ctx.Session.CharID)
}

func actorIsAdmin(ctx client.Context, id uint32) bool {
	return ctx.Session != nil && ctx.Session.IsAdminID(id)
}

func localPlayerIsAdmin(ctx client.Context) bool {
	if ctx.Session == nil {
		return false
	}
	return ctx.Session.IsAdminID(ctx.Session.AccountID) || ctx.Session.IsAdminID(ctx.Session.CharID)
}

func applyMapAcceptEnter(ctx client.Context, enter network.MapAcceptEnter) {
	if ctx.Session == nil || ctx.World == nil {
		return
	}
	resetWorldForSelectedCharacterIfNeeded(ctx)
	ctx.Session.SyncServerTick(enter.ServerTick, time.Now())
	ctx.Session.PlayerX = enter.X
	ctx.Session.PlayerY = enter.Y
	ctx.Session.PlayerDir = enter.Dir
	ctx.Session.Playing = true
	ctx.World.SetPlayerPosition(enter.X, enter.Y, enter.Dir)
	ctx.World.Player.IsAdmin = localPlayerIsAdmin(ctx)
}

func resetWorldForSelectedCharacterIfNeeded(ctx client.Context) {
	if ctx.Session == nil || ctx.World == nil || ctx.Session.CharID == 0 {
		return
	}
	if ctx.Session.Playing && ctx.World.Player.ID == ctx.Session.CharID {
		return
	}
	ctx.World.Player = worldActorForSelectedCharacter(ctx)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	ctx.World.Dir = 0
}

func worldActorForSelectedCharacter(ctx client.Context) worldstate.Actor {
	character := ctx.Session.SelectedCharacter()
	actor := worldstate.Actor{
		ID:            ctx.Session.CharID,
		Name:          character.Name,
		Job:           character.Job,
		Head:          character.Hair,
		Weapon:        character.Weapon,
		Shield:        character.Shield,
		HeadTop:       character.HeadTop,
		HeadMid:       character.HeadMid,
		HeadLow:       character.HeadLow,
		HeadPal:       character.HeadPal,
		BodyPal:       character.BodyPal,
		Sex:           ctx.Session.Sex,
		IsAdmin:       localPlayerIsAdmin(ctx),
		Appearance:    true,
		Speed:         defaultPlayerMoveSpeedMS,
		EffectState:   character.Option,
		HasState:      character.Option != 0,
		Level:         ctx.Session.Progress.BaseLevel,
		HasLevel:      ctx.Session.Progress.BaseLevel > 0,
		AttackRange:   ctx.Session.AttackRange,
		PartyName:     ctx.Session.Party.Name,
		GuildID:       ctx.Session.GuildID,
		EmblemVersion: ctx.Session.EmblemVersion,
		GuildName:     ctx.Session.GuildName,
		HasObjectType: true,
		ObjectType:    actorObjectTypePC,
	}
	if character.Option&actorEffectCartMask != 0 {
		applyActorCartStateFromEffect(&actor)
	}
	return actor
}

func applyActorNameAck(ctx client.Context, ack network.ActorNameAck) {
	name := sanitizeActorName(ack.Name)
	if name == "" || ctx.World == nil {
		return
	}
	guildName := strings.TrimSpace(ack.GuildName)
	partyName := strings.TrimSpace(ack.PartyName)
	if isLocalActor(ctx, ack.ID) {
		ctx.World.Player.Name = name
		ctx.World.Player.PartyName = partyName
		if guildName != "" {
			ctx.World.Player.GuildName = guildName
		}
		if ctx.Session != nil && guildName != "" {
			ctx.Session.GuildName = guildName
			ctx.Session.PendingGuildName = ""
		}
		return
	}
	actor, ok := ctx.World.Actors[ack.ID]
	if !ok {
		return
	}
	actor.Name = name
	actor.PartyName = partyName
	actor.GuildName = guildName
	ctx.World.Actors[ack.ID] = actor
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
	shadowScale float64
	shadowDepth float64
	depth       float64
	isPlayer    bool
	hidden      bool
}

const (
	// RO clients map 175 source pixels to the five world units of one map cell.
	actorBillboardSourcePixelsPerCell      = 175.0
	actorBillboardWorldUnitsPerSourcePixel = actorBillboardCellWorldUnits / actorBillboardSourcePixelsPerCell
)

const (
	actorBillboardCellWorldUnits  = 5.0
	actorBillboardWorldHeightUnit = 1.0 * actorBillboardCellWorldUnits
	babyPlayerBodyScale           = 0.8
	actorJobWarpPortal            = 45
	actorJobWarpPortalActive      = 128
	actorJobWarpPortalWaiting     = 129
	actorJobHiddenNPC             = 111
	actorJobClearNPC              = 844
	actorObjectTypePC             = 0
	actorObjectTypeDisguised      = 1
	actorObjectTypeMob            = 5
	actorObjectTypeNPC            = 6
	actorObjectTypePet            = 7
	actorObjectTypeHomunculus     = 8
	actorObjectTypeMercenary      = 9
	actorObjectTypeElemental      = 10
	actorObjectTypeUnit           = 11
	actorObjectTypeNPC2           = 12
	actorObjectTypeNPCABR         = 13
	actorObjectTypeNPCBionic      = 14
)

func (m *WorldMode) drawSceneActors(screen *render.Frame, ctx client.Context, projection sceneProjection) []sceneActorDrawEntry {
	entries := m.collectSceneActorEntries(screen, ctx, projection)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].depth > entries[j].depth
	})
	for _, entry := range entries {
		m.drawActorShadowEntry(screen, ctx, projection, entry)
	}
	for _, entry := range entries {
		m.drawSceneActorEntry(screen, ctx, projection, entry)
	}
	m.drawSceneActorFalcons(screen, ctx, projection, entries)
	return entries
}

func (m *WorldMode) drawSceneActorOverlays(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time, entries []sceneActorDrawEntry) {
	m.drawSiegeGuildEmblems(screen, ctx, entries)
	for _, entry := range entries {
		m.drawActorCastBar(screen, entry, now)
		m.drawActorLifeBar(screen, ctx, entry)
	}
	m.drawAttackFocusMarker(screen, ctx, now, entries)
	m.drawScriptHighlightMarker(screen, ctx, now, entries)
	m.drawVendingBoardLabels(screen, ctx, entries)
	m.drawChatRoomBoardLabels(screen, ctx, entries)
	m.drawSpeechBubbles(screen, entries, now)
	m.drawHoveredLocalPlayerNameLabel(screen, ctx, entries)
	m.drawHoveredActorNameLabel(screen, ctx, projection, now)
}

func (m *WorldMode) drawHoveredLocalPlayerNameLabel(screen *render.Frame, ctx client.Context, entries []sceneActorDrawEntry) {
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
		drawActorNameLabelsAtY(screen, actorDisplayLabels(ctx, entry.actor, true), m.actorGuildEmblem(ctx, entry.actor, true), entry.screenX, labelY, actorNameLabelColor(entry.actor, true))
		return
	}
}

func (m *WorldMode) collectSceneActorEntries(screen *render.Frame, ctx client.Context, projection sceneProjection) []sceneActorDrawEntry {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	entries := make([]sceneActorDrawEntry, 0, len(ctx.World.Actors)+1)
	player := ctx.World.Player
	player.ID = ctx.Session.CharID
	character := ctx.Session.SelectedCharacter()
	player.Job = character.Job
	player.Head = character.Hair
	player.Sex = ctx.Session.Sex
	player.IsAdmin = localPlayerIsAdmin(ctx)
	ctx.World.Player.IsAdmin = player.IsAdmin
	if !player.HasCartState {
		if player.HasState {
			player.EffectState = (player.EffectState &^ actorPersistentEffectOptionMask) | (character.Option & actorPersistentEffectOptionMask)
		} else {
			player.EffectState = character.Option
		}
		applyActorCartStateFromEffect(&player)
	}
	if character.Name != "" {
		player.Name = character.Name
	}
	if ctx.Session != nil {
		if player.GuildID == 0 {
			player.GuildID = localGuildID(ctx)
		}
		if player.EmblemVersion == 0 {
			player.EmblemVersion = ctx.Session.EmblemVersion
		}
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

func (m *WorldMode) drawSceneActorEntry(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry) {
	cameraYaw := projection.cameraYaw
	alpha := 1.0
	if entry.hidden {
		alpha = 0.35
	}
	alpha *= m.actorVisualAlpha(entry.actor.ID, time.Now())
	if entry.isPlayer {
		if !cartDrawAfterActor(entry.actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
		}
		if m.drawPlayerSprite3D(ctx, screen, projection, entry, entry.actor.Dir, cameraYaw, entry.shadow, alpha) {
			if cartDrawAfterActor(entry.actor, cameraYaw) {
				m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
			}
			return
		}
		return
	}
	if visual := specialNPCVisualForActor(ctx, entry.actor); visual != specialNPCVisualNone {
		if m.drawSpecialNPCVisual(screen, ctx, projection, entry, visual, time.Now()) {
			return
		}
	}
	if isWarpActor(entry.actor) {
		return
	}
	if !cartDrawAfterActor(entry.actor, cameraYaw) {
		m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
	}
	if m.drawActorSprite3D(screen, ctx, projection, entry, cameraYaw, entry.shadow) {
		if cartDrawAfterActor(entry.actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
		}
		return
	}
}

func (m *WorldMode) drawActorShadowEntry(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry) {
	if !entry.castShadow || m.shadowView == nil || m.shadowViewMiss {
		return
	}
	if entry.hidden {
		return
	}
	if !entry.isPlayer && m.nonPCActorHasGR2Model(ctx, entry.actor) {
		return
	}
	now := time.Now()
	scale := m.actorShadowRenderScale(ctx, entry, now)
	if scale == 0 {
		return
	}
	if !m.actorShadowSuppressed(entry.actor, now) {
		drawFixedSpriteShadowBillboard3D(screen, projection, m.shadowView, entry.worldX, entry.worldY, entry.worldZ+actorShadowTerrainLift, scale, m.actorVisualAlpha(entry.actor.ID, now), entry.shadow)
	}
	m.drawActorCartShadow3D(screen, ctx, projection, entry, projection.cameraYaw, now)
}

func (m *WorldMode) actorShadowRenderScale(ctx client.Context, entry sceneActorDrawEntry, now time.Time) float64 {
	baseScale := m.actorRenderScale(entry.actor.ID, entry.scale, now)
	if entry.isPlayer {
		baseScale = m.playerRenderScale(ctx, entry.actor, entry.scale, now)
	} else if actorRepresentsPlayer(entry.actor) {
		baseScale = m.playerBodyRenderScale(entry.actor.ID, entry.actor.Job, entry.scale, now)
	}
	scale := baseScale * entry.shadowScale
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 0
	}
	return scale
}

func appendActorDrawEntry(entries []sceneActorDrawEntry, world *worldstate.World, projection sceneProjection, actor worldstate.Actor, isPlayer bool, now time.Time, screenWidth, screenHeight int) []sceneActorDrawEntry {
	actorX, actorY := actorRenderPosition(actor, now)
	actor.Dir = actorRenderDirection(actor, now)
	actor = actorWithVisualJob(actor)
	terrainZ := terrainHeightAt(world, actorX, actorY)
	worldX, worldY := actorWorldAnchor(actor, actorX, actorY)
	point := projection.Project(worldX, worldY, terrainZ)
	scale := actorBillboardScreenScale(projection, worldX, worldY, terrainZ)
	if isPlayer || actorRepresentsPlayer(actor) {
		scale *= playerBodyScaleForJob(actor.Job)
	}
	if actorAnchorOutsideViewport(float64(point.x), float64(point.y), screenWidth, screenHeight, scale) {
		return entries
	}
	depth := actorBillboardSortDepth(projection, worldX, worldY, terrainZ)
	shadowDepth := projection.Depth(worldX, worldY, terrainZ+actorShadowTerrainLift)
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
		shadowScale: actorShadowSize(actor),
		shadowDepth: shadowDepth,
		depth:       depth,
		isPlayer:    isPlayer,
	})
}

func playerBodyScaleForJob(job int16) float64 {
	if db.IsBabyJob(int(job)) {
		return babyPlayerBodyScale
	}
	return 1
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
	if size, ok := db.MonsterShadowSize[int(actor.Job)]; ok {
		return size
	}
	return 1
}

func actorShadowFactor(world *worldstate.World, x, y float64) float64 {
	if world == nil || world.GND == nil {
		return 1
	}
	if gndHeight, ok := worldGNDHeightAt(world, x, y); ok && terrainHeightAt(world, x, y)-gndHeight > 0.5 {
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

func actorBillboardSortDepth(projection sceneProjection, x, y, z float64) float64 {
	footDepth := projection.Depth(x, y, z)
	topDepth := projection.Depth(x, y, z+actorBillboardWorldHeightUnit)
	if topDepth <= 0 || !isFinite(topDepth) {
		return footDepth
	}
	return math.Min(footDepth, topDepth)
}

func actorDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	labels := actorDisplayLabels(ctx, actor, isPlayer)
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

func actorDisplayLabels(ctx client.Context, actor worldstate.Actor, isPlayer bool) []string {
	name := actorDisplayPrimaryName(ctx, actor, isPlayer)
	if name == "" {
		return nil
	}
	if guildName := actorGuildDisplayName(ctx, actor, isPlayer); guildName != "" {
		return []string{name, guildName}
	}
	return []string{name}
}

func actorDisplayPrimaryName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if isPlayer {
		if name := sanitizeActorName(selectedCharacterName(ctx.Session)); name != "" {
			return actorDisplayNameWithParty(ctx, actor, name, true)
		}
		return actorDisplayNameWithParty(ctx, actor, sanitizeActorName(actor.Name), true)
	}
	if isWarpActor(actor) {
		return ""
	}
	if name := sanitizeActorName(actor.Name); name != "" {
		return actorDisplayNameWithParty(ctx, actor, name, false)
	}
	if res.HasPlayerJobToken(int(actor.Job)) || ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job)); ok {
		return displayNameFromResource(resourceName)
	}
	if name, ok := db.MonsterDisplayName[int(actor.Job)]; ok {
		return name
	}
	return ""
}

func actorGuildDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if !actorCanDisplayGuildName(actor, isPlayer) {
		return ""
	}
	if guildName := strings.TrimSpace(actor.GuildName); guildName != "" {
		return guildName
	}
	if isPlayer && ctx.Session != nil {
		return strings.TrimSpace(ctx.Session.GuildName)
	}
	return ""
}

func actorCanDisplayGuildName(actor worldstate.Actor, isPlayer bool) bool {
	if strings.TrimSpace(actor.GuildName) == "" && !isPlayer {
		return false
	}
	if isPlayer {
		return true
	}
	if actor.HasObjectType {
		return actor.ObjectType == actorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
}

func actorDisplayNameWithParty(ctx client.Context, actor worldstate.Actor, name string, isPlayer bool) string {
	name = strings.TrimSpace(name)
	partyName := actorPartyDisplayName(ctx, actor, isPlayer)
	if name == "" || partyName == "" {
		return name
	}
	return name + " (" + partyName + ")"
}

func actorPartyDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	if name := strings.TrimSpace(actor.PartyName); name != "" {
		return name
	}
	if isPlayer && ctx.Session != nil {
		return strings.TrimSpace(ctx.Session.Party.Name)
	}
	return ""
}

func actorIsPartyMember(s *session.Session, actor worldstate.Actor, actorName string) bool {
	if s == nil || actor.ID == 0 {
		return false
	}
	for _, member := range s.Party.Members {
		if member.AccountID == actor.ID {
			return true
		}
		if actorName != "" && strings.EqualFold(sanitizeActorName(member.Name), actorName) {
			return true
		}
	}
	return false
}

func (m *WorldMode) drawHoveredActorNameLabel(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
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
	labels := m.hoveredActorDisplayLabels(ctx, actor, now)
	if len(labels) == 0 {
		return
	}
	actorX, actorY := actorRenderPosition(actor, now)
	terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
	isPlayer := isLocalActor(ctx, actor.ID)
	labelY := actorNameLabelY(float64(point.y), scale)
	if life, ok := m.actorLifeForDisplay(ctx, actor); ok {
		labelY = actorNameBelowLifeBarY(float64(point.y), scale, life)
	}
	drawActorNameLabelsAtY(screen, labels, m.actorGuildEmblem(ctx, actor, isPlayer), float64(point.x), labelY, actorNameLabelColor(actor, isPlayer))
}

func (m *WorldMode) hoveredActorDisplayName(ctx client.Context, actor worldstate.Actor, now time.Time) string {
	labels := m.hoveredActorDisplayLabels(ctx, actor, now)
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

func (m *WorldMode) hoveredActorDisplayLabels(ctx client.Context, actor worldstate.Actor, now time.Time) []string {
	if isLocalActor(ctx, actor.ID) {
		return actorDisplayLabels(ctx, actor, true)
	}
	if name := sanitizeActorName(actor.Name); name != "" {
		return actorDisplayLabels(ctx, actor, false)
	}
	if shouldUseServerNameForHoverActor(actor) {
		m.requestActorName(ctx, actor.ID, now)
		if name := actorResourceDisplayName(ctx, actor); name != "" {
			return []string{name}
		}
		if res.HasPlayerJobToken(int(actor.Job)) {
			return []string{"Player"}
		}
		if isMonsterLikeHoverActor(actor) {
			return []string{"Monster"}
		}
		return []string{"NPC"}
	}
	if name := actorResourceDisplayName(ctx, actor); name != "" {
		return []string{name}
	}
	return []string{"Entity"}
}

func actorResourceDisplayName(ctx client.Context, actor worldstate.Actor) string {
	if ctx.Resources == nil {
		return ""
	}
	if resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job)); ok {
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
		glog.Warnf("send name request failed id=%d: %v", id, err)
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
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case actorObjectTypeNPC, actorObjectTypeHomunculus, actorObjectTypeMercenary:
		return true
	default:
		return false
	}
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
	return s.SelectedCharacter().Name
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
	name = strings.TrimSuffix(name, ".gr2")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ToLower(strings.TrimSpace(name))
	fields := strings.Fields(name)
	for i, field := range fields {
		fields[i] = titleASCIIWord(field)
	}
	return strings.Join(fields, " ")
}

func isWarpActor(actor worldstate.Actor) bool {
	return isWarpActorJob(int(actor.Job))
}

func isWarpActorJob(job int) bool {
	return job == actorJobWarpPortal || job == actorJobWarpPortalActive || job == actorJobWarpPortalWaiting
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
	if actor.IsAdmin {
		return color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
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

func drawActorNameLabelsAtY(screen *render.Frame, labels []string, emblem *render.Image, centerX, labelY float64, foreground color.RGBA) {
	if len(labels) == 0 {
		return
	}
	labels = actorVisibleNameLabels(labels)
	if len(labels) == 0 {
		return
	}
	render.DrawActorUILabels(screen, labels, emblem, centerX, labelY, foreground, color.RGBA{A: 196})
}

func actorVisibleNameLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == 0 {
			label = sanitizeActorName(label)
		} else {
			label = strings.TrimSpace(label)
		}
		if label != "" {
			out = append(out, label)
		}
	}
	return out
}

func actorSpriteTopY(baseY, scale float64) float64 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	return baseY - float64(humanoidBillboardAnchorY)*scale
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

func actorCastBarY(baseY, scale float64) float64 {
	return actorSpriteTopY(baseY, scale) + 4
}

func (m *WorldMode) drawActorSprite3D(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.actor
	if actorIsMercenary(actor) {
		return m.drawMercenarySprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
	}
	if !res.HasPlayerJobToken(int(actor.Job)) {
		if m.nonPCActorHasGR2Model(ctx, actor) {
			return m.drawNonPCGR2Model3D(screen, ctx, entry, shadow)
		}
		return m.drawNonPCSprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
	}
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := actorSpriteKey{
		job:         int(actor.Job),
		head:        int(actor.Head),
		sex:         actor.Sex,
		admin:       actor.IsAdmin,
		bodyPalette: int(actor.BodyPal),
		headPalette: int(actor.HeadPal),
		weapon:      weapon,
		shield:      shield,
		headTop:     int(actor.HeadTop),
		headMid:     int(actor.HeadMid),
		headLow:     int(actor.HeadLow),
	}
	if _, ok := m.actorViewMiss[key]; ok {
		return false
	}
	view, ok := m.actorViews[key]
	if !ok {
		loaded, status := loadHumanoidSpriteViewWithAppearance(ctx.Resources, humanoidAppearance(key), "actor")
		if loaded == nil {
			m.actorViewMiss[key] = struct{}{}
			glog.Warnf("actor sprite unavailable id=%d job=%d head=%d sex=%d: %s", actor.ID, key.job, key.head, key.sex, status)
			return false
		}
		m.actorViews[key] = loaded
		view = loaded
		glog.Debugf("actor sprite resources id=%d job=%d head=%d sex=%d %s", actor.ID, key.job, key.head, key.sex, status)
	}
	now := time.Now()
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    actor.Dir,
		headDir:      actor.HeadDir,
		headTurn:     true,
		cameraYaw:    cameraYaw,
		moving:       actorIsMovingAt(actor, now),
		moveSpeedMS:  actor.Speed,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.actionFamily = spriteActionSit
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		applyActorAnimationToSpriteState(&state, anim, true)
	}
	if !isDeathActionFamily(state.actionFamily) {
		applyActorBodyState(actor, &state)
	}
	billboard, ok := humanoidBillboardForState(view, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, m.playerBodyRenderScale(actor.ID, actor.Job, entry.scale, now), m.actorVisualAlpha(actor.ID, now), shadow, m.actorRenderTint(actor, now))
	return true
}

func (m *WorldMode) drawMercenarySprite3D(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.actor
	view := m.mercenaryHumanoidSpriteView(ctx, actor)
	if view == nil {
		return false
	}
	now := time.Now()
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    actor.Dir,
		headDir:      actor.HeadDir,
		headTurn:     true,
		cameraYaw:    cameraYaw,
		moving:       actorIsMovingAt(actor, now),
		moveSpeedMS:  actor.Speed,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actorRenderWalkDistance(actor, now)
	} else if actor.Sitting {
		state.actionFamily = spriteActionSit
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		applyActorAnimationToSpriteState(&state, anim, true)
	}
	if !isDeathActionFamily(state.actionFamily) {
		applyActorBodyState(actor, &state)
	}
	billboard, ok := humanoidBillboardForState(view, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, m.actorRenderScale(actor.ID, entry.scale, now), m.actorVisualAlpha(actor.ID, now), shadow, m.actorRenderTint(actor, now))
	return true
}

func (m *WorldMode) mercenaryHumanoidSpriteView(ctx client.Context, actor worldstate.Actor) *humanoidSpriteView {
	key := mercenarySpriteKeyForActor(actor)
	if _, ok := m.mercenaryViewMiss[key]; ok {
		return nil
	}
	if m.mercenaryViews == nil {
		m.mercenaryViews = make(map[actorSpriteKey]*humanoidSpriteView)
	}
	if view, ok := m.mercenaryViews[key]; ok {
		return view
	}
	if ctx.Resources == nil {
		return nil
	}
	loaded, status := loadMercenaryHumanoidSpriteView(ctx.Resources, humanoidAppearance(key), "mercenary")
	if loaded == nil {
		if m.mercenaryViewMiss == nil {
			m.mercenaryViewMiss = make(map[actorSpriteKey]struct{})
		}
		m.mercenaryViewMiss[key] = struct{}{}
		glog.Warnf("mercenary sprite unavailable id=%d job=%d head=%d sex=%d: %s", actor.ID, key.job, key.head, key.sex, status)
		return nil
	}
	m.mercenaryViews[key] = loaded
	glog.Debugf("mercenary sprite resources id=%d job=%d head=%d sex=%d %s", actor.ID, key.job, key.head, key.sex, status)
	return loaded
}

func (m *WorldMode) drawNonPCSprite3D(screen *render.Frame, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
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
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, m.actorRenderScale(actor.ID, entry.scale, now), m.actorVisualAlpha(actor.ID, now), shadow, m.actorRenderTint(actor, now))
	return true
}

func (m *WorldMode) nonPCSpriteState(actor worldstate.Actor, now time.Time) spriteState {
	state := spriteState{
		actionFamily: spriteActionIdle,
		direction:    actor.Dir,
		moving:       actorIsMovingAt(actor, now),
		loopIdle:     true,
		moveSpeedMS:  actor.Speed,
	}
	if state.moving {
		state.actionFamily = spriteActionWalk
		state.loop = true
		state.walkDistance = actorRenderWalkDistance(actor, now)
	}
	if anim, ok := m.actorAnimation(actor.ID, now); ok {
		if applyActorAnimationToSpriteState(&state, anim, false) {
			state.loopIdle = false
		}
	}
	if !isDeathActionFamily(state.actionFamily) {
		applyActorBodyState(actor, &state)
	}
	return state
}

func isDeathActionFamily(actionFamily int) bool {
	return actionFamily == spriteActionPCDeath || actionFamily == spriteActionNonPCDeath
}

func (m *WorldMode) nonPCSpriteView(ctx client.Context, actor worldstate.Actor) *spriteView {
	job := int(actor.Job)
	if _, ok := m.nonPCViewMiss[job]; ok {
		return nil
	}
	if m.nonPCViews == nil {
		m.nonPCViews = make(map[int]*spriteView)
	}
	view, ok := m.nonPCViews[job]
	if ok {
		return m.petAccessorySpriteView(ctx, actor, view)
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
		glog.Warnf("nonpc sprite unavailable id=%d job=%d: %s", actor.ID, job, status)
		return nil
	}
	m.nonPCViews[job] = loaded
	glog.Debugf("nonpc sprite resources id=%d job=%d %s", actor.ID, job, status)
	return m.petAccessorySpriteView(ctx, actor, loaded)
}

func (m *WorldMode) petAccessorySpriteView(ctx client.Context, actor worldstate.Actor, base *spriteView) *spriteView {
	accessoryID := m.petAccessoryIDs[actor.ID]
	if accessoryID == 0 {
		return base
	}
	key := petAccessorySpriteKey{job: int(actor.Job), accessoryID: accessoryID}
	if view, ok := m.petAccessoryViews[key]; ok {
		return view
	}
	if _, miss := m.petAccessoryMiss[key]; miss {
		return base
	}
	view, status := loadPetAccessorySpriteView(ctx.Resources, base, accessoryID, "pet accessory")
	if view == nil {
		if m.petAccessoryMiss == nil {
			m.petAccessoryMiss = make(map[petAccessorySpriteKey]struct{})
		}
		m.petAccessoryMiss[key] = struct{}{}
		glog.Warnf("pet accessory sprite unavailable id=%d job=%d accessory=%d: %s", actor.ID, actor.Job, accessoryID, status)
		return base
	}
	if m.petAccessoryViews == nil {
		m.petAccessoryViews = make(map[petAccessorySpriteKey]*spriteView)
	}
	m.petAccessoryViews[key] = view
	glog.Debugf("pet accessory sprite resources id=%d job=%d accessory=%d %s", actor.ID, actor.Job, accessoryID, status)
	return view
}
