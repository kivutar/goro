package game

import (
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func movementControlContext(t *testing.T) (client.Context, net.Conn) {
	t.Helper()
	conn, server := testutil.NewConnection(t, 20080910)
	w := world.New()
	w.Player = world.Actor{ID: 2000000, X: 10, Y: 20, Speed: 150}
	w.GAT = testutil.FlatWalkableGAT(64, 64)
	w.Actors[300] = world.Actor{ID: 300, X: 30, Y: 20, Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	return client.Context{Network: conn, World: w, Session: &session.Session{AccountID: 2000000, CharID: 150000, AttackRange: 1}, Input: input.NewState(), ScreenW: 800, ScreenH: 600}, server
}

func readMovementPacket(t *testing.T, server net.Conn, x, y int) {
	t.Helper()
	packet, _ := network.BuildWalkToXYPacketForClientDate(x, y, 20080910)
	testutil.ReadPackets(t, server, packet)
}

func TestMovementReplacesPickupAndPlayerSkillChase(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{pendingPickup: pickupIntent{itemID: 999}}
	m.skillSystem.Pending = skills.PendingTarget{Skill: session.Skill{ID: db.SkillALHeal}, TargetID: 300}
	if !m.requestMove(ctx, 12, 20, "test") {
		t.Fatal("move rejected")
	}
	readMovementPacket(t, server, 12, 20)
	if m.pendingPickup.itemID != 0 || m.skillSystem.Pending.Skill.ID != 0 {
		t.Fatal("new move retained an old pursuit")
	}
}

func TestNewAttackReplacesOldTarget(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{pendingAttack: attackIntent{targetID: 300}, pendingPickup: pickupIntent{itemID: 999}}
	actor := ctx.World.Actors[300]
	actor.ID, actor.X = 301, 11
	ctx.World.Actors[301] = actor
	m.requestAttack(ctx, actor, "test")
	readLegacyBotTestActionPacket(t, server, 301, network.ActionAttack)
	if m.pendingAttack.targetID != 301 || !m.pendingAttack.requested || m.pendingPickup.itemID != 0 {
		t.Fatal("new attack retained a conflicting pursuit")
	}
}

func TestHeldAttackDoesNotBecomeGroundWalking(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{}
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
	m.requestMouseAttack(ctx, ctx.World.Actors[300])
	readMovementPacket(t, server, 28, 20)
	ctx.Input.EndFrame()
	m.walkAwaitingAck, m.walkCooldownUntil = false, time.Time{}
	p := m.sceneProjection(ctx, 800, 600, time.Now()).Project(scene.CellCenter(30), scene.CellCenter(20), 0)
	ctx.Input.SetMousePosition(int(p.X), int(p.Y))
	if m.updateHeldWalk(ctx, false, time.Now()) || m.pendingAttack.targetID != 300 {
		t.Fatal("held attack became a ground walk")
	}
}

func TestMouseAttackReleaseAndLock(t *testing.T) {
	for _, locked := range []bool{false, true} {
		t.Run(map[bool]string{false: "held", true: "ctrl"}[locked], func(t *testing.T) {
			ctx, server := movementControlContext(t)
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			ctx.Input.SetKey(input.KeyCtrl, locked)
			m := &WorldMode{}
			m.requestMouseAttack(ctx, actor)
			readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
			ctx.Input.SetKey(input.KeyCtrl, false)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			m.releaseMouseAttack(ctx)
			if locked {
				if m.pendingAttack.targetID != 300 {
					t.Fatal("Ctrl lock did not survive modifier release")
				}
			} else {
				testutil.ReadPackets(t, server, []byte{0x18, 0x01})
				if m.pendingAttack.targetID != 0 {
					t.Fatal("mouse release retained repeat attack")
				}
			}
		})
	}
}

func TestShortAttackClickStillApproachesAndSwingsOnce(t *testing.T) {
	ctx, server := movementControlContext(t)
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
	m := &WorldMode{}
	m.requestMouseAttack(ctx, ctx.World.Actors[300])
	readMovementPacket(t, server, 28, 20)
	ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
	m.releaseMouseAttack(ctx)
	if m.pendingAttack.mode != attackOnce {
		t.Fatal("released chase lost its single swing")
	}
	ctx.World.SetPlayerPosition(28, 20, 0)
	m.updatePendingAttack(ctx, "test")
	readLegacyBotTestActionPacket(t, server, 300, network.ActionAttackOnce)
}

func TestFreshMouseClickRetriesRejectedAttack(t *testing.T) {
	for _, noCtrl := range []bool{false, true} {
		t.Run(map[bool]string{false: "ctrl", true: "noctrl"}[noCtrl], func(t *testing.T) {
			ctx, server := movementControlContext(t)
			ctx.Session.NoCtrl = noCtrl
			ctx.Input.SetKey(input.KeyCtrl, !noCtrl)
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			m := &WorldMode{}
			m.requestMouseAttack(ctx, actor)
			readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
			// No acknowledgement: the server can silently reject an attack.
			assertNoBotTestPacket(t, server, func() error {
				m.updatePendingAttack(ctx, "test")
				return nil
			})
			m.requestMouseAttack(ctx, actor)
			testutil.ReadPackets(t, server, []byte{0x18, 0x01})
			readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
		})
	}
}

func TestScriptAttackCanRetryWithoutSendingEveryFrame(t *testing.T) {
	ctx, server := movementControlContext(t)
	actor := ctx.World.Actors[300]
	actor.X = 11
	ctx.World.Actors[300] = actor
	m := &WorldMode{}
	m.requestAttack(ctx, actor, "script")
	readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
	assertNoBotTestPacket(t, server, func() error {
		m.requestAttack(ctx, actor, "script")
		return nil
	})
	m.lastAttackAt = time.Now().Add(-scene.AttackRetryInterval)
	m.requestAttack(ctx, actor, "script")
	testutil.ReadPackets(t, server, []byte{0x18, 0x01})
	readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
}

func TestFreeCastAllowsAttacksButNotAnotherSkillDuringCasting(t *testing.T) {
	for _, learned := range []bool{false, true} {
		t.Run(map[bool]string{false: "without Free Cast", true: "with Free Cast"}[learned], func(t *testing.T) {
			ctx, server := movementControlContext(t)
			ctx.World.Player.Job = db.JobSage
			if learned {
				ctx.Session.Skills.List = []session.Skill{{ID: db.SkillSAFreecast, Level: 10}}
			}
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			m := &WorldMode{}
			m.startSkillCastSourceAnimation(ctx, network.SkillCastNotify{
				SourceID: ctx.World.Player.ID, TargetID: actor.ID, SkillID: db.SkillMGFirebolt,
			}, 3*time.Second, time.Now())
			if anim, ok := m.playerAnimation(ctx, time.Now()); !ok || !anim.Casting {
				t.Fatal("cast did not start")
			}
			if m.playerCanUseSkill(ctx, time.Now()) {
				t.Fatal("casting allowed another skill")
			}
			m.requestAttack(ctx, actor, "test")
			if !learned {
				assertNoBotTestPacket(t, server, func() error {
					m.updatePendingAttack(ctx, "test")
					return nil
				})
				m.clearLocalActorAction(ctx)
				m.updatePendingAttack(ctx, "test")
			}
			readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
		})
	}
}

func TestSelfHealWaitsForAttackAnimation(t *testing.T) {
	for _, alias := range []string{"account", "character"} {
		t.Run(alias, func(t *testing.T) {
			ctx, server := movementControlContext(t)
			m := &WorldMode{}
			m.startActorAnimation(ctx, ctx.World.Player.ID, sprites.ActionPCAttack1, time.Now(), time.Second)
			skill := session.Skill{ID: db.SkillALHeal, Type: skills.SkillTargetFriend, Level: 1, Range: 9}
			target := ctx.World.Player
			if alias == "character" {
				target.ID = ctx.Session.CharID
			}
			if err := m.skills().UseTarget(ctx, skill, target, "test"); err != nil {
				t.Fatal(err)
			}
			assertNoBotTestPacket(t, server, func() error {
				m.skills().UpdatePendingTarget(ctx, "test", false)
				m.skills().ProcessPendingTarget(ctx)
				return nil
			})
			if m.skillSystem.Pending.TargetID != target.ID {
				t.Fatal("self-heal was discarded while waiting for the attack to end")
			}
			m.clearLocalActorAction(ctx)
			m.skills().UpdatePendingTarget(ctx, "test", false)
			m.skills().ProcessPendingTarget(ctx)
			testutil.ReadPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, target.ID, 20080910))
		})
	}
}

func TestAttackDistanceFailureResumesAndIgnoresOldTarget(t *testing.T) {
	ctx, server := movementControlContext(t)
	actor := ctx.World.Actors[300]
	actor.X = 11
	ctx.World.Actors[300] = actor
	m := &WorldMode{}
	m.requestAttack(ctx, actor, "test")
	readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
	m.applyAttackFailureForDistance(ctx, network.AttackFailureForDistance{TargetID: 300, SourceX: 10, SourceY: 20, TargetX: 14, TargetY: 20, AttackRange: 1})
	readMovementPacket(t, server, 12, 20)
	if m.pendingAttack.targetID != 300 || m.pendingAttack.requested {
		t.Fatal("distance failure lost the chase")
	}
	m.cancelPlayerAction(ctx)
	ctx.World.SetPlayerPosition(15, 20, 0)
	m.applyAttackFailureForDistance(ctx, network.AttackFailureForDistance{TargetID: 300, SourceX: 10, SourceY: 20, TargetX: 14, TargetY: 20, AttackRange: 1})
	if ctx.World.Player.X != 15 || m.pendingAttack.targetID != 0 {
		t.Fatal("stale failure moved the player or resumed a canceled attack")
	}
}

func TestAttackDoesNotWaitForWalkEnd(t *testing.T) {
	ctx, server := movementControlContext(t)
	actor := ctx.World.Actors[300]
	actor.X = 11
	ctx.World.Actors[300] = actor
	setPlayerMovementAt(ctx, 10, 20, 14, 20, 0, time.Now(), 0)
	m := &WorldMode{pendingAttack: attackIntent{targetID: 300}}
	m.updatePendingAttack(ctx, "test")
	readLegacyBotTestActionPacket(t, server, 300, network.ActionAttack)
	if !scene.ActorIsMovingAt(ctx.World.Player, time.Now()) {
		t.Fatal("setup did not test an active walk")
	}
}

func TestMovementKeepsLatestClickDuringAnimationAndAck(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{}
	start := time.Now()
	m.startActorAnimation(ctx, ctx.World.Player.ID, sprites.ActionPCAttack1, start, time.Second)
	m.requestMove(ctx, 12, 20, "test")
	m.requestMove(ctx, 14, 20, "test")
	if m.walkSequence != 2 || m.queuedWalk == nil || m.queuedWalk.x != 14 {
		t.Fatal("movement was sent during attack or lost the latest click")
	}
	m.clearLocalActorAction(ctx)
	m.processQueuedWalk(ctx, time.Now())
	readMovementPacket(t, server, 14, 20)
	m.requestMove(ctx, 16, 20, "test")
	m.walkCooldownUntil = time.Time{}
	m.processQueuedWalk(ctx, time.Now())
	if m.walkSequence != 3 || m.queuedWalk == nil {
		t.Fatal("movement bypassed the outstanding acknowledgement")
	}
	m.walkAwaitingAck = false
	m.processQueuedWalk(ctx, time.Now())
	readMovementPacket(t, server, 16, 20)
	if m.queuedWalk != nil {
		t.Fatal("sent walk remains queued")
	}
}

func TestPlayerCommandsPreserveCompanionSkillChase(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{}
	m.skillSystem.Pending = skills.PendingTarget{Skill: session.Skill{ID: db.SkillHlifHeal}, TargetID: 300}
	m.requestMove(ctx, 12, 20, "test")
	readMovementPacket(t, server, 12, 20)
	if m.skillSystem.Pending.Skill.ID != db.SkillHlifHeal {
		t.Fatal("player movement canceled a companion order")
	}
}

func TestCanceledChaseDiscardsQueuedWalk(t *testing.T) {
	for _, action := range []string{"attack", "pickup", "skill"} {
		t.Run(action, func(t *testing.T) {
			ctx, server := movementControlContext(t)
			m := &WorldMode{walkAwaitingAck: true, walkRequestAt: time.Now()}
			switch action {
			case "attack":
				m.requestAttack(ctx, ctx.World.Actors[300], "test")
			case "pickup":
				item := world.FloorItem{ID: 400, X: 30, Y: 20}
				ctx.World.Items[400] = item
				m.requestPickup(ctx, item, "test")
			case "skill":
				skill := session.Skill{ID: db.SkillMGFirebolt, Type: skills.SkillTargetEnemy, Level: 1, Range: 9}
				if err := m.skills().UseTarget(ctx, skill, ctx.World.Actors[300], "test"); err != nil {
					t.Fatal(err)
				}
			}
			if m.queuedWalk == nil {
				t.Fatal("chase was not queued")
			}
			delete(ctx.World.Actors, 300)
			delete(ctx.World.Items, 400)
			assertNoBotTestPacket(t, server, func() error {
				m.updatePendingAttack(ctx, "test")
				m.updatePendingPickup(ctx, "test", false)
				m.skills().UpdatePendingTarget(ctx, "test", false)
				m.walkAwaitingAck = false
				m.processQueuedWalk(ctx, time.Now())
				return nil
			})
			if m.queuedWalk != nil || m.playerChasing() {
				t.Fatal("vanished target retained its chase")
			}
		})
	}
}

func TestSkillChaseCanBeCanceledBeforeWalkAck(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := &WorldMode{walkAwaitingAck: true, walkRequestAt: time.Now()}
	skill := session.Skill{ID: db.SkillMGFirebolt, Type: skills.SkillTargetEnemy, Level: 1, Range: 9}
	if err := m.skills().UseTarget(ctx, skill, ctx.World.Actors[300], "test"); err != nil {
		t.Fatal(err)
	}
	m.skills().Cancel("test")
	m.walkAwaitingAck = false
	assertNoBotTestPacket(t, server, func() error { m.processQueuedWalk(ctx, time.Now()); return nil })
	if m.queuedWalk != nil {
		t.Fatal("canceled skill kept its queued walk")
	}
}

func TestTargetedSkillsRespectAnimationLocksAndComboWindows(t *testing.T) {
	for _, state := range []string{"walking", "attacking", "combo"} {
		t.Run(state, func(t *testing.T) {
			ctx, server := movementControlContext(t)
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			m := &WorldMode{}
			skill := session.Skill{ID: db.SkillSMBash, Type: skills.SkillTargetEnemy, Level: 1, Range: 1}
			if state == "walking" {
				setPlayerMovementAt(ctx, 10, 20, 14, 20, 0, time.Now(), 0)
			} else {
				m.actorSystem.ActorAnims = map[uint32]scene.ActorAnimation{
					ctx.World.Player.ID: {ActionFamily: sprites.ActionPCAttack1, Started: time.Now(), Duration: time.Second, Combo: state == "combo"},
				}
			}
			if err := m.skills().UseTarget(ctx, skill, actor, "test"); err != nil {
				t.Fatal(err)
			}
			if state == "attacking" {
				if m.skillSystem.Pending.TargetID != 300 {
					t.Fatal("skill input during an attack was lost")
				}
				assertNoBotTestPacket(t, server, func() error {
					m.skills().UpdatePendingTarget(ctx, "test", false)
					m.skills().ProcessPendingTarget(ctx)
					return nil
				})
				m.clearLocalActorAction(ctx)
				m.skills().ProcessPendingTarget(ctx)
			}
			testutil.ReadPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, 300, 20080910))
		})
	}
}

func TestSelfTargetedCombosPreserveAttackUntilMouseRelease(t *testing.T) {
	for _, skillID := range []uint16{
		db.SkillMOChaincombo, db.SkillMOCombofinish, db.SkillChTigerfist, db.SkillChChaincrush,
		db.SkillMOExtremityfist, db.SkillTKTurnkick, db.SkillTKStormkick, db.SkillTKDownkick,
		db.SkillTKCounter, db.SkillTKJumpkick, db.SkillHTPower,
	} {
		t.Run(skills.SkillLabel(session.Skill{ID: skillID}), func(t *testing.T) {
			ctx, server := movementControlContext(t)
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			m := &WorldMode{}
			m.requestMouseAttack(ctx, actor)
			readLegacyBotTestActionPacket(t, server, actor.ID, network.ActionAttack)
			m.actorSystem.ActorAnims = map[uint32]scene.ActorAnimation{
				ctx.World.Player.ID: {ActionFamily: sprites.ActionPCAttack3, Started: time.Now(), Duration: time.Second, Combo: true},
			}
			// The server resolves these self-targeted requests against the
			// current attack, including skills switched to self during a combo.
			skill := session.Skill{ID: skillID, Type: skills.SkillTargetSelf, Level: 1, Range: 1}
			if err := m.skills().Use(ctx, skill, "test"); err != nil {
				t.Fatal(err)
			}
			testutil.ReadPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, ctx.Session.AccountID, 20080910))
			if m.pendingAttack.targetID != actor.ID || !m.pendingAttack.requested {
				t.Fatal("combo forgot the active attack")
			}
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			m.releaseMouseAttack(ctx)
			testutil.ReadPackets(t, server, []byte{0x18, 0x01})
			if m.pendingAttack.targetID != 0 {
				t.Fatal("attack survived releasing the mouse after a combo")
			}
		})
	}
}

func TestOtherSkillsStillCancelAttacksDuringComboWindow(t *testing.T) {
	for _, skill := range []session.Skill{
		{ID: db.SkillALAngelus, Type: skills.SkillTargetSelf, Level: 1},
		{ID: db.SkillMOExtremityfist, Type: skills.SkillTargetEnemy, Level: 1, Range: 1},
	} {
		t.Run(skills.SkillLabel(skill), func(t *testing.T) {
			ctx, server := movementControlContext(t)
			actor := ctx.World.Actors[300]
			actor.X = 11
			ctx.World.Actors[300] = actor
			ctx.Session.NoCtrl = true
			m := &WorldMode{}
			m.requestMouseAttack(ctx, actor)
			readLegacyBotTestActionPacket(t, server, actor.ID, network.ActionAttack)
			m.actorSystem.ActorAnims = map[uint32]scene.ActorAnimation{
				ctx.World.Player.ID: {ActionFamily: sprites.ActionPCAttack3, Started: time.Now(), Duration: time.Second, Combo: true},
			}
			targetID := ctx.Session.AccountID
			if skills.IsSelfTargetSkill(skill) {
				if err := m.skills().Use(ctx, skill, "test"); err != nil {
					t.Fatal(err)
				}
			} else {
				targetID = actor.ID
				if err := m.skills().UseTarget(ctx, skill, actor, "test"); err != nil {
					t.Fatal(err)
				}
			}
			testutil.ReadPackets(t, server, []byte{0x18, 0x01})
			testutil.ReadPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(skill.ID, 1, targetID, 20080910))
			if m.pendingAttack.targetID != 0 {
				t.Fatal("ordinary skill retained an attack")
			}
		})
	}
}

func TestMapTransitionDiscardsQueuedMovement(t *testing.T) {
	ctx, server := movementControlContext(t)
	m := NewWorldMode()
	m.walkAwaitingAck, m.walkRequestAt = true, time.Now()
	m.requestMove(ctx, 12, 20, "test")
	m.startMapFadeOut(network.MapChange{MapName: "geffen"}, time.Now())
	m.walkAwaitingAck = false
	assertNoBotTestPacket(t, server, func() error { m.processQueuedWalk(ctx, time.Now()); return nil })
	if m.queuedWalk != nil {
		t.Fatal("map transition kept a walk for the old map")
	}
}
