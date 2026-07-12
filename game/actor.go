package game

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func actorBillboardScreenScale(projection sceneProjection, x, y, z float64) float64 {
	base := projection.Project(x, y, z)
	top := projection.Project(x, y, z+actorBillboardWorldHeightUnit)
	projectedHeight := math.Hypot(float64(top.x-base.x), float64(top.y-base.y))
	if projectedHeight <= 0 || math.IsNaN(projectedHeight) || math.IsInf(projectedHeight, 0) {
		return 1
	}
	return projectedHeight / float64(humanoidBillboardAnchorY)
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
	// reference client allows target-type mismatches on PvP/GvG maps. Goro does not yet
	// parse map state packets, so keep the rule isolated until that state exists.
	return false
}

func actorCanOpenPlayerContext(ctx client.Context, actor worldstate.Actor) bool {
	if isLocalActor(ctx, actor.ID) || strings.TrimSpace(actor.Name) == "" {
		return false
	}
	if actor.HasObjectType {
		return actor.ObjectType == actorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
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

func upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if isLocalActor(ctx, entry.ID) {
		return
	}
	dir := entry.Dir
	if entry.Moving {
		dir = directionFromDelta(entry.FromX, entry.FromY, entry.ToX, entry.ToY, dir)
	}
	actor := worldstate.Actor{
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
		HeadPal:       entry.HeadPal,
		BodyPal:       entry.BodyPal,
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
		BodyState:     entry.BodyState,
		HealthState:   entry.HealthState,
		EffectState:   entry.EffectState,
		HasState:      entry.HasState,
	}
	applyActorCartStateFromEffect(&actor)
	ctx.World.UpsertActor(actor)
}

func (m *WorldMode) upsertNetworkActor(ctx client.Context, entry network.ActorEntry) {
	if ctx.World == nil {
		return
	}
	oldState := uint32(0)
	if existing, ok := ctx.World.Actors[entry.ID]; ok && existing.HasState {
		oldState = existing.EffectState
	}
	upsertNetworkActor(ctx, entry)
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
	if m.attackFocusID == vanish.ID {
		m.clearAttackFocus()
	}
	if vanish.Reason == 1 {
		m.startActorDeath(ctx, vanish.ID)
		return
	}
	m.removeActorEffectStateEffects(vanish.ID)
	ctx.World.RemoveActor(vanish.ID)
	delete(m.actorAnims, vanish.ID)
	delete(m.actorDeaths, vanish.ID)
	delete(m.actorSoundFrames, vanish.ID)
	delete(m.actorLife, vanish.ID)
	delete(m.speechBubbles, vanish.ID)
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
		m.removeActorEffectStateEffects(id)
		delete(m.actorDeaths, id)
		delete(m.actorAnims, id)
		delete(m.actorSoundFrames, id)
		delete(m.actorLife, id)
		delete(m.speechBubbles, id)
		if m.pendingAttack.targetID == id {
			m.pendingAttack = attackIntent{}
		}
		if m.lockedAttackID == id {
			m.clearLockedAttack()
		}
		if m.attackFocusID == id {
			m.clearAttackFocus()
		}
		log.Printf("actor death removed id=%d", id)
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
	now := time.Now()
	fastForward := time.Duration(0)
	if ctx.Session != nil {
		if elapsed, ok := ctx.Session.ElapsedSinceServerTick(ack.ServerTick, now); ok {
			fastForward = elapsed
		}
	}
	dir := directionFromDelta(ack.FromX, ack.FromY, ack.ToX, ack.ToY, ctx.World.Dir)
	ctx.World.SetPlayerMovementAt(ack.FromX, ack.FromY, ack.ToX, ack.ToY, dir, now, fastForward)
	ctx.Session.PlayerX = ack.ToX
	ctx.Session.PlayerY = ack.ToY
}

func applyMapAcceptEnter(ctx client.Context, enter network.MapAcceptEnter) {
	if ctx.Session != nil {
		ctx.Session.SyncServerTick(enter.ServerTick, time.Now())
	}
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

func applyActorJumpPosition(ctx client.Context, position network.ActorJumpPosition) {
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
	actorJobHiddenNPC             = 111
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

func (m *WorldMode) drawSceneActorOverlays(screen *render.Image, ctx client.Context, projection sceneProjection, now time.Time, entries []sceneActorDrawEntry) {
	for _, entry := range entries {
		m.drawActorCastBar(screen, entry, now)
		m.drawActorLifeBar(screen, ctx, entry)
	}
	m.drawAttackFocusMarker(screen, ctx, now, entries)
	m.drawVendingBoardLabels(screen, ctx, entries)
	m.drawSpeechBubbles(screen, entries, now)
	m.drawHoveredLocalPlayerNameLabel(screen, ctx, entries)
	m.drawHoveredActorNameLabel(screen, ctx, projection, now)
}

func (m *WorldMode) cursorSpriteView(ctx client.Context) *spriteView {
	if m.cursorView != nil || m.cursorViewMiss {
		return m.cursorView
	}
	if view, status := loadCursorSpriteView(ctx.Resources); view != nil {
		m.cursorView = view
	} else {
		m.cursorViewMiss = true
		log.Printf("cursor resources unavailable: %s", status)
	}
	return m.cursorView
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
	if !player.HasCartState {
		if player.HasState {
			player.EffectState = (player.EffectState &^ actorEffectCartMask) | (character.Option & actorEffectCartMask)
		} else {
			player.EffectState = character.Option
		}
		applyActorCartStateFromEffect(&player)
	}
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
		if !cartDrawAfterActor(entry.actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
		}
		if m.drawPlayerSprite3D(ctx, screen, projection, entry, entry.actor.Dir, cameraYaw, entry.shadow, alpha) {
			if cartDrawAfterActor(entry.actor, cameraYaw) {
				m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
			}
			return
		}
		render.DrawRect(screen, entry.screenX-6, entry.screenY-6, 24, 24, render.ColorPanel)
		render.DrawRect(screen, entry.screenX-6, entry.screenY-6, 24, 2, render.ColorAccent)
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
	if !cartDrawAfterActor(entry.actor, cameraYaw) {
		m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
	}
	if m.drawActorSprite3D(screen, ctx, projection, entry, cameraYaw, entry.shadow) {
		if cartDrawAfterActor(entry.actor, cameraYaw) {
			m.drawActorCart3D(screen, ctx, projection, entry, cameraYaw, entry.shadow, alpha)
		}
		return
	}
	if actorJobHasNoSprite(int(entry.actor.Job)) {
		return
	}
	drawActorMarker(screen, entry.screenX-6, entry.screenY-20, entry.actor, time.Now())
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
	render.DebugPrintAt(screen, fmt.Sprintf("%d", actor.Job), int(x-12), int(y-16))
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
	if size, ok := db.MonsterShadowSize[int(actor.Job)]; ok {
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

func actorDisplayNameWithParty(ctx client.Context, actor worldstate.Actor, name string, isPlayer bool) string {
	name = strings.TrimSpace(name)
	partyName := actorPartyDisplayName(ctx, actor, name, isPlayer)
	if name == "" || partyName == "" {
		return name
	}
	return name + " (" + partyName + ")"
}

func actorPartyDisplayName(ctx client.Context, actor worldstate.Actor, actorName string, isPlayer bool) string {
	if ctx.Session == nil || !ctx.Session.Party.Active() {
		return ""
	}
	name := strings.TrimSpace(ctx.Session.Party.Name)
	if name == "" {
		return ""
	}
	if isPlayer {
		return name
	}
	if !actorCanDisplayPartyName(actor) {
		return ""
	}
	if !actorIsPartyMember(ctx.Session, actor, actorName) {
		return ""
	}
	return name
}

func actorCanDisplayPartyName(actor worldstate.Actor) bool {
	if actor.HasObjectType {
		return actor.ObjectType == actorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
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
		return actorDisplayNameWithParty(ctx, actor, name, false)
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
	render.DrawCenteredUIOutlinedTextAt(screen, label, centerX, labelY, foreground, outline)
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
	return actorSpriteTopY(baseY, scale) - 10
}

func (m *WorldMode) drawActorSprite3D(screen *render.Image, ctx client.Context, projection sceneProjection, entry sceneActorDrawEntry, cameraYaw float64, shadow float64) bool {
	actor := entry.actor
	if !res.HasPlayerJobToken(int(actor.Job)) {
		return m.drawNonPCSprite3D(screen, ctx, projection, entry, cameraYaw, shadow)
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
	if _, ok := m.actorViewMiss[key]; ok {
		return false
	}
	view, ok := m.actorViews[key]
	if !ok {
		loaded, status := loadHumanoidSpriteViewWithAppearance(ctx.Resources, humanoidAppearance(key), "actor")
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
		state.loop = anim.loop
		state.play = anim.play
		state.hasPlay = anim.hasPlay
		state.length = anim.length
		state.hasLength = anim.hasLength
		state.frameOffset = anim.frameOffset
		state.hasFrameOffset = anim.hasFrameOffset
		state.moving = false
		state.fixedMotion = anim.fixedMotion
		state.hasFixedMotion = anim.hasFixedMotion
		state.speed = anim.speed
		state.hasSpeed = anim.hasSpeed
	}
	if !isDeathActionFamily(state.actionFamily) {
		applyActorBodyState(actor, &state)
	}
	billboard, ok := humanoidBillboardForState(view, state, now)
	if !ok {
		return false
	}
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, 1, shadow, actorStateTint(actor))
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
	drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, m.actorDeathAlpha(actor.ID, now), shadow, actorStateTint(actor))
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
		state.loop = anim.loop
		state.play = anim.play
		state.hasPlay = anim.hasPlay
		state.length = anim.length
		state.hasLength = anim.hasLength
		state.frameOffset = anim.frameOffset
		state.hasFrameOffset = anim.hasFrameOffset
		state.moving = false
		state.loopIdle = false
		state.fixedMotion = anim.fixedMotion
		state.hasFixedMotion = anim.hasFixedMotion
		state.speed = anim.speed
		state.hasSpeed = anim.hasSpeed
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
