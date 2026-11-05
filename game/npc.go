package game

import (
	"time"

	"github.com/kivutar/goro/game/pet"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func clickedTalkTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || !npc.CanTalk(actor) {
		return worldstate.Actor{}, false
	}
	return actor, true
}

func (m *WorldMode) startServerProgress(ctx client.Context, progress network.ProgressBar, now time.Time) {
	m.npc().StartProgress(progress, now)
	// A progress bar is a server-owned WIP_DISABLE_ALL interval. Drop queued
	// character work so it cannot fire when that interval ends.
	m.cancelPlayerAction(ctx)
	m.pendingPickup = pickupIntent{}
	m.skillSystem.Pending = skills.PendingTarget{}
	m.skillSystem.PendingText = skills.PendingTextTarget{}
	m.petSystem.PendingCapture = pet.CaptureState{}
	m.clearLocalActorAction(ctx)
}

func (m *WorldMode) requestNPCTalk(ctx client.Context, actor worldstate.Actor, source string) {
	if scene.PlayerIsDead(ctx) {
		return
	}
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	m.cancelPlayerAction(ctx)
	if err := ctx.Network.SendNPCContact(actor.ID); err == nil {
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
		glog.Warnf("%s npc talk failed target=%d player=%d,%d target=%d,%d: %v", source, actor.ID, playerX, playerY, actor.X, actor.Y, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

// Higher-priority modals block NPC input in both normal UI dispatch and Lua.
// This does not require an open NPC dialog: Update also unpublishes closed ones.
func (u *worldUI) npcInputAvailable() bool {
	return !u.disconnectDialog.IsOpen() &&
		!u.escapeMenu.IsOpen() &&
		!u.interactionModalOpen() &&
		!u.social.PartyInfo.IsOpen() &&
		!u.pet.Confirm.IsOpen() &&
		!u.companions.HomunculusConfirm.IsOpen() &&
		!u.companions.MercenaryConfirm.IsOpen() &&
		!u.inventoryBag.KeyboardShortcutsBlocked()
}

func (m *WorldMode) npc() *npc.System {
	m.npcSystem.Bind(npc.Bindings{
		Windows:        &m.ui.npc,
		Storage:        &m.ui.storageWindow,
		Map:            &m.mapScene,
		PointerBlocked: m.mapPointerBlocked,
	})
	return &m.npcSystem
}
