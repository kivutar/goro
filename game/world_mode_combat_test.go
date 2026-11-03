package game

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestHiddenEffectsDoNotStartSoundsOrCameraShake(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	for _, spawn := range []string{"actor", "ground", "ground lifetime", "persistent actor", "persistent ground"} {
		for _, hidden := range []bool{false, true} {
			ctx.Session.NoEffects = hidden
			m := &WorldMode{}
			for _, id := range []int{effects.MagnumBreak, effects.QuakeMagnum} {
				switch spawn {
				case "actor":
					m.effects().AddAt(ctx, id, 300, start)
				case "ground":
					m.effects().AddCell(ctx, id, 300, 10, 20, start, 0, 0)
				case "ground lifetime":
					m.effects().AddCellLifetime(ctx, id, 300, 10, 20, start, time.Second, false)
				case "persistent actor":
					m.effects().AddPersistentBetween(ctx, id, 300, 0, start)
				case "persistent ground":
					m.effects().AddCellLifetime(ctx, id, 300, 10, 20, start, time.Second, true)
				}
			}
			want := 2
			if hidden && !strings.HasPrefix(spawn, "persistent") {
				want = 0
			}
			if len(m.effects().Active) != want {
				t.Fatalf("%s hidden=%t: active=%d, want %d", spawn, hidden, len(m.effects().Active), want)
			}
			if audible := len(m.scheduledSounds) > 0; audible == hidden {
				t.Fatalf("%s hidden=%t: sound scheduled=%t", spawn, hidden, audible)
			}
			if shaking := !m.cameraShakeEnd.IsZero(); shaking == hidden {
				t.Fatalf("%s hidden=%t: camera shake=%t", spawn, hidden, shaking)
			}
		}
	}
}

func TestDisabledTransientEffectsDoNotInitializeOrScheduleSounds(t *testing.T) {
	for _, name := range []string{"Mammonite", "Soul Strike", "Throw Shuriken"} {
		t.Run(name, func(t *testing.T) {
			for _, hidden := range []bool{false, true} {
				ctx := testutil.KnightEffectContext()
				ctx.Session.NoEffects = hidden
				m := &WorldMode{}
				start := time.Unix(100, 0)
				if name == "Mammonite" {
					m.effects().AddAt(ctx, effects.Mammonite, 300, start)
				} else {
					skill := db.SkillMGSoulstrike
					if name == "Throw Shuriken" {
						skill = db.SkillNJSyuriken
					}
					m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, HitCount: 5}, start)
				}
				if hidden {
					if len(m.effects().Active)+len(m.scheduledSounds)+len(m.effects().StrEffectMiss) != 0 {
						t.Fatal("disabled effect initialized resources, created an instance, or scheduled sounds")
					}
				} else if len(m.effects().Active) != 1 || len(m.scheduledSounds) == 0 {
					t.Fatal("enabled effect lost its visuals or sounds")
				}
			}
		})
	}
}

func TestDisabledEffectsKeepServerOwnedState(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Session.NoEffects = true
	m := &WorldMode{}
	m.actors().SetActorStateEffect(ctx, 300, effects.Sight, true)
	m.actors().SetActorStateEffect(ctx, 300, effects.Sight, true)
	m.effects().ApplyActorDevotion(ctx, network.ActorDevotion{SourceID: 2000000, Targets: [5]uint32{300}})
	m.applyClownGypsySkill(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMarionette, SourceID: 2000000, TargetID: 300}, time.Now())
	if len(m.effects().Active) != 5 || len(m.scheduledSounds) != 0 {
		t.Fatalf("expected silent Sight, Devotion, Marionette and two glows: effects=%d sounds=%d", len(m.effects().Active), len(m.scheduledSounds))
	}
	for _, effect := range m.effects().Active {
		if !effect.Persistent || !effect.Expires.IsZero() {
			t.Fatalf("effect %d lost its server-owned lifetime", effect.EffectID)
		}
	}
	m.effects().AddAt(ctx, effects.SightTrasher, 300, time.Now())
	if m.effects().HasActive(effects.Sight, 300, time.Now()) {
		t.Fatal("suppressed Sight Trasher left Sight active")
	}
	m.effects().RemoveActor(300)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.PinkBody {
		t.Fatal("hidden effects or links survived their owner leaving")
	}
}

func TestKaenEffectTogglePreservesUnitAndSuppressesSound(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		ctx := testutil.KnightEffectContext()
		ctx.Session.NoEffects = hidden
		m := &WorldMode{}
		start := time.Unix(100, 0)
		for id := uint32(1); id <= 32; id++ {
			m.effects().AddCellLifetime(ctx, effects.Kaen, id, 10, 20, start, time.Minute, true)
		}
		if len(m.effects().Active) != 32 || (len(m.scheduledSounds) > 0) == hidden {
			t.Fatalf("hidden=%t: active=%d sounds=%d", hidden, len(m.effects().Active), len(m.scheduledSounds))
		}
	}
}

func TestAcolyteHealAmountAndOffensiveTiming(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		amount uint32
		effect int
	}{
		{0, effects.Exit}, {199, effects.Exit}, {200, effects.Heal},
		{1999, effects.Heal}, {2000, effects.Heal2}, {3999, effects.Heal2}, {4000, effects.Heal4},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: db.SkillALHeal, SourceID: 2000000, TargetID: 300, Amount: tc.amount}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed Heal displayed an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != 300 {
			t.Fatalf("Heal amount %d: %+v", tc.amount, m.effects().Active)
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "_heal_effect.wav" {
			t.Fatal("Heal must play its sound once")
		}
	}
	for _, skill := range []uint16{db.SkillALHeal, db.SkillALHolylight} {
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 200, HitCount: 1, Action: 8}
		m.addSkillEffect(ctx, action, start, hit)
		before := 0
		if skill == db.SkillALHeal {
			before = 1
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.HealOffensive || !m.effects().Active[0].Starts.Equal(start) {
				t.Fatal("offensive Heal must start its white effect at release")
			}
		} else if len(m.effects().Active) != 0 {
			t.Fatal("Holy Light impact appeared before the hit")
		}
		m.addSkillHitEffect(ctx, action, hit)
		impact := effects.HolyLight
		if skill == db.SkillALHeal {
			impact = effects.BashHit
		}
		if len(m.effects().Active) != before+2 {
			t.Fatalf("skill %d: missing native impact pair", skill)
		}
		for i, id := range []int{effects.Hit1, impact} {
			effect := m.effects().Active[before+i]
			if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(hit) {
				t.Fatalf("skill %d: misplaced or mistimed impact", skill)
			}
		}
		m = &WorldMode{}
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.effects().Active) != 0 {
			t.Fatal("miss played an impact")
		}
	}
}

func TestAcolyteSupportOwnershipAndActions(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		owner  uint32
	}{
		{db.SkillALIncagi, effects.IncAgility, 300},
		{db.SkillALDecagi, effects.DecAgility, 300},
		{db.SkillALHolywater, effects.Aqua, 2000000},
		{db.SkillALCrucis, effects.Signum, 2000000},
		{db.SkillALAngelus, effects.Angelus, 2000000},
		{db.SkillALBlessing, effects.Blessing, 300},
		{db.SkillALCure, effects.Cure, 300},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d: failure displayed an effect", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != tc.owner {
			t.Fatalf("skill %d: wrong owner or extra release effect: %+v", tc.skill, m.effects().Active)
		}
		pose := sprites.ActionPCSkill
		if tc.skill == db.SkillALIncagi {
			pose = scene.SpriteActionIdle
		}
		if m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
			t.Fatalf("skill %d: wrong release pose", tc.skill)
		}
	}
}

func TestAlchemistBottlePackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemist, db.JobAlchemist
	for level := 1; level <= 5; level++ {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: db.SkillAMPotionpitcher, SourceID: 2000000, TargetID: 300, Amount: uint32(level)}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed Potion Pitcher created a bottle")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 0 {
			t.Fatal("Potion Pitcher must produce one silent bottle; the server owns the heal effect")
		}
		e := m.effects().Active[0]
		if e.EffectID != effects.ThrowItem2 || e.ActorID != n.SourceID || e.TargetID != n.TargetID || e.ItemLevel != level || m.actorSystem.ActorAnims[n.SourceID].ActionFamily != sprites.ActionPCAttack1 {
			t.Fatal("incorrect bottle owner, level or throwing pose")
		}
		p, _, _ := effects.ThrowItemAt(e, 13)
		if math.Abs(p.X-15.7) > .001 || math.Abs(p.Y-7.67) > .001 {
			t.Fatalf("incorrect half-flight arc: %+v", p)
		}
		actor := ctx.World.Actors[300]
		actor.X = 60
		ctx.World.Actors[300] = actor
		end, _, alpha := effects.ThrowItemAt(e, 28)
		if math.Abs(end.X-21.3) > .001 || alpha != 235 {
			t.Fatalf("bottle must retain its destination and fade after tick 27: %+v alpha=%d", end, alpha)
		}
		actor.X = 20
		ctx.World.Actors[300] = actor
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		n := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillAMAcidterror, Damage: damage, Action: network.ActorActionSkill, SourceSpeed: 1000}
		m.applyActorActionNotify(ctx, n)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.ThrowItem || !m.effects().Active[0].Starts.Equal(m.actorSystem.ActorAnims[n.SourceID].Started) {
			t.Fatal("Acid Terror must throw at release, including misses, with no extra hit spark")
		}
		n.Action = 5
		m.applyActorActionNotify(ctx, n)
		if len(m.effects().Active) != 1 {
			t.Fatal("splash notification replayed Acid Terror")
		}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillAMBerserkpitcher, time.Second, start)
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillAMBerserkpitcher, Result: 1})
	if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != effects.ThrowItem5 || m.effects().Active[0].ItemLevel != 4 || m.effects().Active[1].EffectID != effects.ItemFast3 || !m.actorSystem.ActorAnims[2000000].Started.Equal(start) {
		t.Fatal("Berserk Pitcher must keep its casting pose, throw a white bottle and buff its target")
	}
	p, _, _ := effects.ThrowItemAt(m.effects().Active[0], 13)
	if math.IsNaN(p.Y) || math.Abs(p.Y-7.67) > .001 {
		t.Fatal("a same-cell throw must have a finite vertical arc")
	}
}

func TestAlchemistEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, skill := range []uint16{db.SkillAMCpWeapon, db.SkillAMCpShield, db.SkillAMCpArmor, db.SkillAMCpHelm} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: skill, Result: 1})
		if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 300 || len(m.scheduledSounds) != 1 {
			t.Fatal("Chemical Protection must play once on the recipient")
		}
		e := m.effects().Active[0]
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, e.Starts.Add(9*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != 64 {
			t.Fatal("Chemical Protection must draw eight double-layer rays")
		}
	}
	for _, id := range []int{effects.ThrowItem, effects.ThrowItem2, effects.ThrowItem5, effects.CoupleCasting, effects.Demonstration} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		m.effects().Active[0].ItemLevel = 4
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldBillboards").Len()+v.FieldByName("worldCommands").Len()+v.FieldByName("commands").Len() == 0 {
			t.Fatalf("effect %d did not render with client assets", id)
		}
	}
	for level, rgb := range [][3]float64{{255, 135, 135}, {255, 205, 135}, {255, 255, 135}, {255, 255, 255}, {135, 135, 255}} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillAMPotionpitcher, Amount: uint32(level + 1), Result: 1})
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, m.effects().Active[0].Starts.Add(12*db.EffectFrameDuration), false)
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != 4 {
			t.Fatal("potion must have three fading afterimages")
		}
		for j, field := range []string{"ColorR", "ColorG", "ColorB"} {
			if math.Abs(billboards.Index(0).FieldByName(field).Float()-rgb[j]/255) > .001 {
				t.Fatalf("incorrect potion color for level %d", level+1)
			}
		}
	}
	for _, id := range []int{effects.Twilight1, effects.Twilight2, effects.Twilight3} {
		m := &WorldMode{}
		m.effects().AddBetweenDuration(ctx, id, 300, 0, start, 3*time.Second)
		e := m.effects().Active[0]
		one, many := effects.NewTwilightParticles(ctx, e), effects.NewTwilightParticles(ctx, e)
		one.Advance(e, 100)
		for tick := 0; tick <= 100; tick++ {
			many.Advance(e, tick)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("Twilight movement depends on render frequency")
		}
		for _, p := range e.Twilight.Particles {
			if p.Position.X < -5.5 || p.Position.X > 26.3 || p.Position.Z < 4.5 || p.Position.Z > 36.3 {
				t.Fatal("Twilight must surround the viewer, not the distant caster")
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(60*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if vertices != 1280 || commands.Len() > 3 {
			t.Fatal("Twilight must batch its 320 ingredients by texture")
		}
		frame = render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(4*time.Second), false)
		if len(m.effects().Active) != 0 || reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
			t.Fatal("Twilight survived its cast")
		}
	}
}

func TestArcherConcentrationAndUtility(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillACConcentration, SourceID: 2000000, TargetID: 300}
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
		t.Fatal("failed Concentration must not produce effects or sound")
	}
	notify.Result = 1
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ac_concentration.wav" {
		t.Fatal("Concentration needs only its STR and native sound")
	}
	if e := m.effects().Active[0]; e.EffectID != effects.Concentration || e.ActorID != notify.SourceID {
		t.Fatalf("Concentration effect must belong to the caster: %+v", e)
	}
	for _, skill := range []uint16{db.SkillACOwl, db.SkillACVulture, db.SkillACMakingarrow} {
		m := &WorldMode{}
		notify.SkillID = skill
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("utility skill %d must not invent a visual effect", skill)
		}
	}
}

func TestArcherArrowShowerGroundAndDamagePackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	family := sprites.AttackActionFamily(nil, ctx.World.Player)
	m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(family, 4, 2, []string{"atk"})}}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillACShower, X: 20, Y: 20})
	if len(m.effects().Active) != 1 || m.effects().Active[0].ArrowTargets == nil {
		t.Fatal("Arrow Shower must fire on empty ground")
	}
	ground := m.effects().Active[0]
	if ground.Starts.Sub(m.actorSystem.ActorAnims[2000000].Started) != effects.ArrowShowerReleaseDelay {
		t.Fatal("ground arrows must wait 18 native 24 ms ticks")
	}
	if len(m.scheduledSounds) != 1 || !sameStringSet(m.scheduledSounds[0].paths, db.WeaponAttackSounds(db.WeaponBow)) {
		t.Fatal("Arrow Shower must play the bow release even on empty ground")
	}
	if target := ground.ArrowTargets[0]; target.X != 10.5 || target.Z != 20.5 {
		t.Fatalf("the ground packet's first arrow must follow the native caster endpoint: %+v", target)
	}
	for _, target := range ground.ArrowTargets[1:] {
		if math.Abs(target.X-20.5) > 1 || math.Abs(target.Z-20.5) > 1 || target.X == 20.5 && target.Z == 20.5 {
			t.Fatalf("arrow outside the eight neighbouring cells: %+v", target)
		}
	}
	// The original explicitly processes Arrow Shower's ACTION_SPLASH as a
	// full skill, including arrows and impacts for each affected target.
	m.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000, TargetID: 300, SkillID: db.SkillACShower,
		Action: 5, Damage: 100, HitCount: 1, SourceSpeed: 500,
	})
	if len(m.effects().Active) != 5 {
		t.Fatalf("expected ground spread, bow arrow, victim spread, and both impacts; got %d", len(m.effects().Active))
	}
	spread := m.effects().Active[2]
	if spread.ArrowTargets == nil || spread.ArrowTargets[0].X != 20.5 || spread.ArrowTargets[0].Z != 20.5 {
		t.Fatal("damage notification must aim the spread's first arrow at its victim")
	}
	for i, id := range []int{effects.Hit1, effects.BashHit} {
		if e := m.effects().Active[3+i]; e.EffectID != id || e.ActorID != 300 || !e.Starts.Equal(m.effects().Active[1].Expires) {
			t.Fatalf("wrong Arrow Shower impact: %+v", e)
		}
	}
}

func TestArcherArrowShowerReplacesPendingReleaseSound(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	family := sprites.AttackActionFamily(nil, ctx.World.Player)
	m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(family, 4, 2, []string{"atk"})}}
	ground := network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillACShower, X: 20, Y: 20}
	m.applyGroundSkillNotify(ctx, ground)
	// Other casters and independent effects must survive replacement.
	otherRelease := time.Now().Add(time.Second)
	m.scheduleSkillActionSoundAtActor(otherRelease, 400, db.SkillACShower, db.WeaponAttackSounds(db.WeaponBow)...)
	m.scheduleSoundAtActor(otherRelease, ground.SourceID, "effect\\ac_concentration.wav")
	for _, id := range []uint32{300, 301, 302} {
		ctx.World.Actors[id] = worldstate.Actor{ID: id, Job: 1002, X: 20, Y: 20}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: ground.SourceID, TargetID: id, SkillID: ground.SkillID,
			Action: 5, Damage: 100, HitCount: 1, SourceSpeed: 200,
		})
	}
	releases, hits, others := 0, 0, 0
	for _, sound := range m.scheduledSounds {
		switch {
		case sound.actorID == ground.SourceID && sameStringSet(sound.paths, db.WeaponAttackSounds(db.WeaponBow)):
			releases++
			if want := m.actorSystem.ActorAnims[ground.SourceID].Started.Add(100 * time.Millisecond); !sound.at.Equal(want) {
				t.Fatalf("release at %s, want latest animation's marker at %s", sound.at, want)
			}
		case sound.actorID >= 300 && sound.actorID <= 302:
			hits++
		case sound.at.Equal(otherRelease):
			others++
		}
	}
	// Each victim keeps the normal hit sound and EF_HIT2's sound.
	if releases != 1 || hits != 6 || others != 2 {
		t.Fatalf("sounds: releases=%d hits=%d unrelated=%d, want 1, 6, 2", releases, hits, others)
	}
	// Once the release has played, a new cast must still produce its sound.
	m.playDueScheduledSounds(ctx, otherRelease.Add(time.Second))
	m.applyGroundSkillNotify(ctx, ground)
	if len(m.scheduledSounds) != 1 || !sameStringSet(m.scheduledSounds[0].paths, db.WeaponAttackSounds(db.WeaponBow)) {
		t.Fatal("a later Arrow Shower must have its own release sound")
	}
}

func TestArcherArrowFlightAndSplash(t *testing.T) {
	origin, target := scene.Point3{}, scene.Point3{X: 4}
	if p, ok := effects.ArrowShowerPosition(origin, target, 48*time.Millisecond); !ok || math.Abs(p.X-2) > 1e-9 {
		t.Fatalf("nearby arrow must travel at the native minimum speed: %+v, %t", p, ok)
	}
	for _, elapsed := range []time.Duration{-1, 96 * time.Millisecond} {
		if _, ok := effects.ArrowShowerPosition(origin, target, elapsed); ok {
			t.Fatal("arrow visible before release or after arrival")
		}
	}
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillACDouble, db.SkillACChargearrow} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: 2000000, TargetID: 300, SkillID: skill, Action: 5, Damage: 100,
		})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d secondary splash must not replay the skill: %+v", skill, m.effects().Active)
		}
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: 2000000, TargetID: 300, SkillID: db.SkillACChargearrow,
			Action: network.ActorActionSkill, Damage: damage,
		})
		want := 1
		if damage > 0 {
			want++
		}
		if len(m.effects().Active) != want || m.effects().Active[0].EffectID != effects.BashBegin || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill {
			t.Fatal("Arrow Repel must use the native skill pose and flash, with no bow projectile")
		}
	}
}

func TestAssassinCrossPacketSequence(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAssassinH, db.JobAssassinH
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = db.WeaponKatar, db.WeaponKatar
	start := time.Now()
	for _, skill := range []uint16{378, 406, 407} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill emitted effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		want := []int{493}
		if skill == 406 {
			want = []int{409}
		} else if skill == 407 {
			want = nil
		}
		pose := sprites.ActionPCSkill
		if skill == 406 {
			pose = sprites.ActionPCAttack3
		}
		if len(m.effects().Active) != len(want) || m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
			t.Fatalf("skill %d: effects=%+v pose=%+v", skill, m.effects().Active, m.actorSystem.ActorAnims)
		}
		for i, e := range m.effects().Active {
			if e.EffectID != want[i] || e.ActorID != 2000000 {
				t.Fatalf("skill %d: effect %d must belong to caster: %+v", skill, want[i], e)
			}
		}
		if skill == 378 {
			if len(m.scheduledSounds) != 6 {
				t.Fatal("EDP needs exactly its six Enchant Poison sounds")
			}
			var ticks []int
			for _, sound := range m.scheduledSounds {
				if len(sound.paths) == 1 && sound.paths[0] == "effect\\assasin_enchantpoison.wav" {
					ticks = append(ticks, int(sound.at.Sub(m.effects().Active[0].Starts)/db.EffectFrameDuration))
					if !sound.positioned {
						t.Fatal("poison sounds must be positioned")
					}
				}
			}
			if !reflect.DeepEqual(ticks, []int{0, 5, 11, 18, 26, 40}) {
				t.Fatalf("EDP sound ticks = %v", ticks)
			}
		}
		if skill == 407 && len(m.scheduledSounds) != 0 {
			t.Fatal("Create Deadly Poison must not play a release sound")
		}
		if skill == 406 {
			a := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100}
			m.addSkillEffect(ctx, a, start, start)
			m.addSkillHitEffect(ctx, a, start)
			if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != 0 || len(m.scheduledSounds) != 1 {
				t.Fatal("Meteor Assault damage replayed the burst")
			}
		}
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: 379, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: damage}
		m.addSkillBeginEffect(ctx, a, start)
		m.addSkillBeforeHitEffect(ctx, a, start)
		m.addSkillEffect(ctx, a, start, start.Add(time.Second))
		m.addSkillHitEffect(ctx, a, start.Add(time.Second))
		want := 1
		if damage != 0 {
			want = 2
		}
		if len(m.effects().Active) != want || len(m.scheduledSounds) != 1 {
			t.Fatalf("Soul Breaker damage %d: effects=%+v sounds=%+v", damage, m.effects().Active, m.scheduledSounds)
		}
		e := m.effects().Active[0]
		if e.EffectID != 361 || e.ActorID != 2000000 || e.TargetID != 300 || !e.Starts.Equal(start) || e.ActorOrigin.X != 10.5 || e.ProjectileDirection.X != 10 {
			t.Fatalf("Soul Breaker must snapshot caster to victim at release: %+v", e)
		}
		if m.scheduledSounds[0].at.Sub(start) != 20*db.EffectFrameDuration || !m.scheduledSounds[0].positioned || !m.cameraShakeEnd.IsZero() {
			t.Fatal("Soul Breaker must play its delayed sound without an extra Asura sound or camera shake")
		}
		for _, skill := range []uint16{379, 406} {
			a.SkillID, a.Action = skill, 5
			m.addSkillBeginEffect(ctx, a, start)
			m.addSkillBeforeHitEffect(ctx, a, start)
			m.addSkillEffect(ctx, a, start, start)
			m.addSkillHitEffect(ctx, a, start)
		}
		if len(m.effects().Active) != want || len(m.scheduledSounds) != 1 {
			t.Fatal("secondary splash replayed effects or sounds")
		}
	}
	m := &WorldMode{}
	m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: 379, SourceID: 2000000, TargetID: 999}, start, start)
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Breaker requires a visible target")
	}
}

func TestAssassinCrossClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAssassinH, db.JobAssassinH
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = db.WeaponKatar, db.WeaponKatar
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobAssassinH)) {
			t.Fatalf("must load the Assassin Cross body without fallback: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{sprites.ActionPCReadyFight, sprites.ActionPCSkill, sprites.ActionPCAttack2, sprites.ActionPCAttack3} {
				if _, a, ok := sprites.ResolveAction(view.Body.Act, family, dir); !ok || len(a.Animations) == 0 {
					t.Fatal("Assassin Cross casting/release ACT missing")
				}
			}
		}
	}
	for _, skill := range []uint16{378, 379, 406, 407} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, DelayTime: 1000, Property: 3})
		aura := 16
		if skill == 379 || skill == 406 {
			aura = 59
		}
		if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != aura || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCReadyFight {
			t.Fatalf("skill %d: wrong casting aura or pose", skill)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		pose := sprites.ActionPCSkill
		if skill == 379 || skill == 406 {
			pose = sprites.ActionPCAttack3
		}
		if m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
			t.Fatalf("skill %d: wrong 2008 release pose", skill)
		}
	}
	for _, weapon := range []int16{db.WeaponShortsword, db.WeaponKatar} {
		ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = weapon, weapon
		for _, skill := range []uint16{379, 406} {
			m := &WorldMode{}
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, SourceSpeed: 600})
			pose := sprites.ActionPCAttack2
			if weapon == db.WeaponKatar {
				pose = sprites.ActionPCAttack3
			}
			if m.actorSystem.ActorAnims[2000000].ActionFamily != pose {
				t.Fatalf("skill %d must use the weapon's attack pose %d", skill, pose)
			}
		}
	}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, vertices int }{
		{361, 20, 0, 0}, {361, 30, 0, 16}, {361, 61, 0, 0},
		{409, 20, 0, 0}, {409, 40, 0, 128}, {409, 61, 0, 0},
		{493, 0, 1, 0}, {493, 39, 13, 0}, {493, 120, 0, 0},
	} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, tc.effect, 2000000, 300, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		commands := v.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			cmd := commands.Index(i)
			vertices += cmd.FieldByName("Vertices").Len()
			if cmd.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("2008 slashes use normal alpha")
			}
		}
		if v.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.effect, tc.tick, v.FieldByName("worldBillboards").Len(), vertices)
		}
		if vertices > 0 {
			ctx.World.Player.X += 10
			moved := render.NewFrame(800, 600)
			m.effects().Draw(moved, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
			ctx.World.Player.X -= 10
			before := commands.Index(0).FieldByName("Vertices")
			after := reflect.ValueOf(moved).Elem().FieldByName("worldCommands").Index(0).FieldByName("Vertices")
			for i := 0; i < before.Len(); i++ {
				if before.Index(i).FieldByName("X").Float() != after.Index(i).FieldByName("X").Float() {
					t.Fatal("released slashes followed the moving caster")
				}
			}
		}
	}
	for _, sound := range []string{"기공포", "메테오 어썰트", "assasin_enchantpoison"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound + ".wav"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAssassinBuffsAndSplasherSequence(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	for _, skill := range []uint16{db.SkillASEnchantpoison, db.SkillASSplasher, db.SkillASPoisonreact} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d produced effects or sounds", skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		want, sounds := effects.EnchantPoison, 2
		if skill == db.SkillASPoisonreact {
			want, sounds = effects.PoisonReact, 0
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != want || m.effects().Active[0].ActorID != 300 || len(m.scheduledSounds) != sounds {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", skill, m.effects().Active, m.scheduledSounds)
		}
	}
	starts := time.Unix(100, 0)
	for _, level := range []uint16{5, 0xffff} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: db.SkillASSplasher, SkillLevel: level, SourceID: 2000000, TargetID: 300, Damage: 50}
		m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
		if level != 0xffff {
			if len(m.effects().Active) != 0 {
				t.Fatal("non-detonation packet started an explosion")
			}
			continue
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.VenomSplasher || !m.effects().Active[0].Starts.Equal(starts) {
			t.Fatalf("detonation must start on the server notification: %+v", m.effects().Active)
		}
		if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(starts.Add(10*db.EffectFrameDuration)) {
			t.Fatalf("explosion sound must play at tick 10: %+v", m.scheduledSounds)
		}
	}
	m := &WorldMode{}
	counter := network.ActorActionNotify{SkillID: db.SkillASPoisonreact, SourceID: 2000000, TargetID: 300, Damage: 50}
	m.addSkillEffect(ctx, counter, starts, starts)
	m.addSkillHitEffect(ctx, counter, starts)
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Poison React counterattack replayed the buff or invented a second effect")
	}
}

func TestBardMusicalStrikeTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(sprites.ActionPCAttack3, 4, 2, []string{"atk"})}}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillBaMusicalstrike, SourceSpeed: 500, HitCount: 1, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	anim := m.actorSystem.ActorAnims[a.SourceID]
	if anim.ActionFamily != sprites.ActionPCAttack3 {
		t.Fatal("Musical Strike must use ACT 88")
	}
	if len(m.effects().Active) != 2 {
		t.Fatalf("Musical Strike needs one arrow and one impact, got %v", m.effects().Active)
	}
	arrow, hit := m.effects().Active[0], m.effects().Active[1]
	if arrow.EffectID != effects.ArrowShot || hit.EffectID != effects.Hit1 || arrow.Starts.Sub(anim.Started) != 250*time.Millisecond || hit.Starts.Sub(arrow.Starts) != scene.ReferenceBowArrowFlightDuration {
		t.Fatal("arrow must launch at the ACT marker and hit eight ticks later")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 2 {
		t.Fatal("splash notification replayed Musical Strike")
	}
}

func TestBlacksmithSkillPacketOwnership(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.World.Player.Job = db.JobBlacksmith
	ctx.Session.Selected.Job = db.JobBlacksmith
	ctx.Session.Selected.ID = 100
	for _, tc := range []struct {
		skill  uint16
		effect int
		actor  uint32
		sounds int
	}{
		{db.SkillBSAdrenaline, effects.HasteUp, 2000000, 2},
		{db.SkillBSAdrenaline2, effects.HasteUp, 2000000, 2},
		{db.SkillBSWeaponperfect, effects.WeaponPerfect, 300, 1},
		{db.SkillBSRepairweapon, effects.RepairWeapon, 300, 2},
		{db.SkillBSOverthrust, effects.Overthrust, 2000000, 2},
		{db.SkillBSMaximize, effects.MaximizePower, 2000000, 4},
		{db.SkillBSGreed, effects.GreedSound, 2000000, 1},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must be silent")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != tc.actor || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d has incorrect effect, owner, or sounds", tc.skill)
		}
		if tc.skill == db.SkillBSOverthrust {
			m.applySkillNoDamageNotify(ctx, n)
			if len(m.effects().Active) != 1 {
				t.Fatal("Overthrust refresh must replace the previous ring")
			}
		}
	}
	for _, skill := range []uint16{db.SkillBSIron, db.SkillBSSteel, db.SkillBSEnchantedstone, db.SkillBSOrideocon,
		db.SkillBSDagger, db.SkillBSSword, db.SkillBSTwohandsword, db.SkillBSAxe, db.SkillBSMace, db.SkillBSKnuckle,
		db.SkillBSSpear, db.SkillBSHiltbinding, db.SkillBSFindingore, db.SkillBSWeaponresearch, db.SkillBSSkintemper, db.SkillBSUnfairlytrick} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("passive/forging skill %d must not invent an activation effect", skill)
		}
	}
}

func TestGreedSoundIsUnpositionedAndSurvivesEffectToggle(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	for _, hidden := range []bool{false, true} {
		ctx.Session.NoEffects = hidden
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: db.SkillBSGreed, SourceID: 300, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.scheduledSounds) != 0 {
			t.Fatal("failed Greed played a sound")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ef_entry.wav" {
			t.Fatal("successful Greed must play its sound even with effects disabled")
		}
		if sound := m.scheduledSounds[0]; sound.actorID != 0 || sound.hasPosition {
			t.Fatal("Greed must play without positional attenuation")
		}
	}
}

func TestChampionAttacksUseReleaseEffectsAndSingleSecondaryHit(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	start := time.Unix(100, 0)
	impact := start.Add(800 * time.Millisecond)
	for _, tc := range []struct {
		skill     uint16
		owners    map[int]uint32
		wav       string
		soundTick int
	}{
		{370, map[int]uint32{376: 300, effects.Quake: 2000000}, "effect\\맹호경파산.wav", 0},
		{371, map[int]uint32{377: 2000000, 263: 2000000, effects.Quake: 2000000}, "effect\\mon_맹룡과강.wav", 5},
		{372, map[int]uint32{512: 2000000, 263: 2000000}, "effect\\연주붕격.wav", 44},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, HitCount: 5}
		m.addSkillEffect(ctx, a, start, impact)
		if len(m.effects().Active) != len(tc.owners) {
			t.Fatalf("skill %d: unexpected release effects %+v", tc.skill, m.effects().Active)
		}
		for _, e := range m.effects().Active {
			if e.ActorID != tc.owners[e.EffectID] || e.Starts != start {
				t.Fatalf("skill %d: wrong effect ownership/timing %+v", tc.skill, e)
			}
			if e.EffectID == 512 && (e.TargetID != 300 || e.ProjectileDirection.X != 2 || e.ActorOrigin.X != 10.5) {
				t.Fatal("Chain Crush must point from the caster toward the victim")
			}
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != tc.wav || m.scheduledSounds[0].at.Sub(start) != time.Duration(tc.soundTick)*db.EffectFrameDuration {
			t.Fatalf("skill %d: wrong sound/timing %+v", tc.skill, m.scheduledSounds)
		}
		m.effects().Active = nil
		m.addSkillHitEffect(ctx, a, impact)
		counts := map[int]int{}
		for _, e := range m.effects().Active {
			counts[e.EffectID]++
			if e.ActorID != 300 || e.Starts.Before(impact) || e.EffectID == effects.BashHit && e.Starts != impact {
				t.Fatal("hit markers must stay on the victim at impact")
			}
		}
		want := map[int]int{effects.Hit1: 5}
		if tc.skill != 370 {
			want[effects.BashHit] = 1
		}
		if !reflect.DeepEqual(counts, want) {
			t.Fatalf("skill %d: hit counts %v, want %v", tc.skill, counts, want)
		}
		m = &WorldMode{}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		_, sourceAnimated := m.actorSystem.ActorAnims[a.SourceID]
		if len(m.effects().Active) != 0 || sourceAnimated {
			t.Fatalf("skill %d splash: effects=%+v anims=%+v sounds=%+v", tc.skill, m.effects().Active, m.actorSystem.ActorAnims, m.scheduledSounds)
		}
		for _, sound := range m.scheduledSounds {
			if sound.actorID != a.TargetID {
				t.Fatal("splash replayed caster sound")
			}
		}
	}
}

func TestChampionZenCastPoseAndServerOwnedSpheres(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 401, SourceID: 2000000})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 402 {
		t.Fatal("instant Zen casting must still launch its wind")
	}
	m = &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 401, SourceID: 2000000, TargetID: 2000000, DelayTime: 4000, Property: 3})
	anim := m.actorSystem.ActorAnims[2000000]
	if anim.ActionFamily != sprites.ActionPCSkill || !anim.HasFixedMotion || anim.FixedMotion != 0 || anim.Play {
		t.Fatalf("Zen must hold skill frame zero: %+v", anim)
	}
	if !m.effects().HasActive(effects.BeginSpell3, 2000000, time.Now()) {
		t.Fatal("Zen must respect the casting property in the server packet")
	}
	portalCount := 0
	for _, e := range m.effects().Active {
		if e.EffectID == 402 {
			portalCount++
			if e.ActorID != 2000000 || e.Expires.Sub(e.Starts) != 200*db.EffectFrameDuration || !e.HasActorTransform {
				t.Fatal("Zen's wind needs its own native lifetime and a fixed origin")
			}
		}
	}
	if portalCount != 1 {
		t.Fatal("Zen did not launch its wind during casting")
	}
	m.effects().Active, m.scheduledSounds = nil, nil
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 401, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 || ctx.World.Player.SpiritSpheres != 0 || m.actorSystem.ActorAnims[2000000].ActionFamily != scene.SpriteActionIdle {
		t.Fatal("Zen release must stand, without replaying casting effects or inventing spheres")
	}
	for count := uint16(1); count <= 5; count++ {
		m.handleNetworkPacket(ctx, testutil.SpiritSpherePacket(network.PacketZCSpirits, 2000000, count), time.Now())
	}
	if ctx.World.Player.SpiritSpheres != 5 || len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Chookgi {
		t.Fatal("Champion needs five ordinary spheres, not the baby Monk variant")
	}
	for _, skill := range []uint16{db.SkillMOCallspirits, db.SkillMOKitranslation} {
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, DelayTime: 1000})
		anim := m.actorSystem.ActorAnims[2000000]
		if anim.ActionFamily != sprites.ActionPCSkill || !anim.HasFixedMotion || anim.FixedMotion != 0 || anim.Play {
			t.Fatal("the inherited sphere skills must use the same casting pose as Zen")
		}
	}
}

func TestChampionAttackPoses(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	for _, skill := range []uint16{370, 371, 372} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 1, SourceSpeed: 800, Action: network.ActorActionSkill})
		anim := m.actorSystem.ActorAnims[2000000]
		switch skill {
		case 370:
			if anim.ActionFamily != sprites.ActionPCSkill || anim.FrameOffset != 3 || anim.Length != 2 || !anim.Play || anim.Speed != 10*db.EffectFrameDuration || anim.Duration != 20*db.EffectFrameDuration {
				t.Fatalf("Palm Strike must play skill frames 3 and 4 once: %+v", anim)
			}
		case 371:
			if anim.ActionFamily != sprites.ActionPCSkill || anim.FixedMotion != 4 || !anim.HasFixedMotion || anim.Play {
				t.Fatalf("Tiger Fist must hold skill frame 4: %+v", anim)
			}
		case 372:
			if anim.ActionFamily != sprites.AttackActionFamily(nil, ctx.World.Player) {
				t.Fatal("Chain Crush must retain the weapon's normal attack pose")
			}
		}
	}
}

func TestClownGypsyArrowVulcanTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(sprites.ActionPCAttack3, 6, 3, []string{"atk"})}}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCGArrowvulcan, SourceSpeed: 500, HitCount: 9, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	start := m.actorSystem.ActorAnims[a.SourceID].Started
	if anim := m.actorSystem.ActorAnims[a.SourceID]; anim.ActionFamily != sprites.ActionPCAttack3 || anim.Speed != 500*time.Millisecond/6 {
		t.Fatal("Arrow Vulcan must release with ACT 88 at the server's attack speed")
	}
	counts := map[int]int{}
	var hitAt time.Time
	for _, e := range m.effects().Active {
		counts[e.EffectID]++
		switch e.EffectID {
		case effects.TripleAttack3:
			if e.ActorID != a.SourceID || e.TargetID != a.TargetID || !e.Starts.Equal(start) || e.ProjectileDirection.X != 10 {
				t.Fatal("rays must aim from the caster through the victim at action start")
			}
		case effects.ArrowShot:
			if e.Starts.Sub(start) != 250*time.Millisecond || e.Expires.Sub(e.Starts) != scene.ReferenceBowArrowFlightDuration {
				t.Fatal("arrow must leave at the ACT marker and fly for eight native ticks")
			}
			hitAt = e.Expires
		}
	}
	if !reflect.DeepEqual(counts, map[int]int{effects.TripleAttack3: 1, effects.ArrowShot: 1, effects.Hit1: 9, effects.BashHit: 1}) {
		t.Fatalf("unexpected Arrow Vulcan effects: %v", counts)
	}
	for i := 1; i < 9; i++ {
		at := hitAt.Add(time.Duration(i) * scene.MultiHitDelay)
		anim, _ := m.actors().ActorAnimation(a.SourceID, at)
		if anim.ActionFamily != sprites.AttackActionFamily(nil, ctx.World.Player) || !anim.HasFixedMotion || anim.FixedMotion != 4 {
			t.Fatalf("additional hit %d must restart the native attack", i)
		}
	}
	before := len(m.effects().Active)
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != before {
		t.Fatal("secondary splash packet replayed Arrow Vulcan")
	}
	for _, skill := range []uint16{db.SkillCGArrowvulcan, db.SkillCGTarotcard} {
		m = &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, a.SourceID, skill, time.Second, start)
		m.effects().AddSkillCastEffects(ctx, skill, 5, a.SourceID, a.TargetID, 0, 0, time.Second, start, "test")
		if m.effects().Active[2].EffectID != effects.BeginSpell6 {
			t.Fatal("Clown casts must use the white aura")
		}
		if skill == db.SkillCGArrowvulcan && (!m.actorSystem.ActorAnims[a.SourceID].HasFixedMotion || m.actorSystem.ActorAnims[a.SourceID].ActionFamily != sprites.ActionPCAttack3) {
			t.Fatal("Vulcan cast must hold the ready-shot pose")
		}
	}
}

func TestClownGypsySuccessPacketsAndTarotSelection(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillCGLongingfreedom, db.SkillCGTarotcard, db.SkillCGHermode} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must not play effects or sounds")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.scheduledSounds) != 1 {
			t.Fatal("successful skill must play exactly one sound")
		}
		if skill == db.SkillCGHermode {
			if len(m.effects().Active) != 0 {
				t.Fatal("Hermode activation must not create an extra ground cube on the caster")
			}
			continue
		}
		id := n.TargetID
		if skill == db.SkillCGLongingfreedom {
			id = n.SourceID
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.ChemicalBody || m.effects().Active[0].ActorID != id {
			t.Fatal("success packet must flash the correct body without inventing a card")
		}
		if skill == db.SkillCGTarotcard {
			m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: effects.TarotCard6})
			if len(m.scheduledSounds) != 1 || len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != effects.TarotCard6 || !scene.IsLocalActor(ctx, m.effects().Active[1].ActorID) {
				t.Fatal("Lovers card must use the server's card and actor, without repeating the sound")
			}
		}
	}
	for _, tc := range []struct{ frame, alpha int }{{0, 10}, {21, 220}, {198, 220}, {199, 215}, {242, 0}} {
		if alpha := effects.TarotCardAlpha(tc.frame); alpha != tc.alpha {
			t.Fatalf("Tarot frame %d: alpha %d, want %d", tc.frame, alpha, tc.alpha)
		}
	}
}

func TestCreatorGroundAndProtectionPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	start := time.Unix(100, 0)
	for _, tc := range []struct{ level, effect int }{{1, effects.Slim}, {5, effects.Slim}, {6, effects.Slim2}, {9, effects.Slim2}, {10, effects.Slim3}} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillCRSlimpitcher, time.Second, start)
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRSlimpitcher, Level: uint16(tc.level), X: 22, Y: 21})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 22 || m.effects().Active[0].Y != 21 {
			t.Fatal("Aid Condensed Potion must produce one ground burst of the appropriate color")
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "assulter_attack.wav" || m.scheduledSounds[0].volume != 40.0/150 || m.scheduledSounds[0].positioned || !m.actorSystem.ActorAnims[2000000].Started.Equal(start) {
			t.Fatal("ground burst must play its sound and preserve the casting throw animation")
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRSlimpitcher, Level: uint16(tc.level), X: 22, Y: 21})
		if len(m.effects().Active) != 2 {
			t.Fatal("a second cast in the same cell must not be deduplicated")
		}
	}
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRCultivation, X: 22, Y: 21})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.BeginSpell6 || m.effects().Active[0].ActorID != 2000000 || m.actorSystem.ActorAnims[2000000].ActionFamily != sprites.ActionPCSkill {
		t.Fatal("Cultivation must use the caster's white flash and skill pose; the server owns the spawned plant")
	}
	m = &WorldMode{}
	n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRFullprotection}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 0 {
		t.Fatal("failed Full Chemical Protection produced effects")
	}
	n.Result = 1
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 1 {
		t.Fatal("Full Chemical Protection needs rays, body flash, and one sound")
	}
	for _, e := range m.effects().Active {
		if e.ActorID != n.TargetID || e.EffectID != effects.ChemicalProt && e.EffectID != effects.ChemicalBody {
			t.Fatal("protection effects must belong to the recipient")
		}
	}
}

func TestCreatorAcidDemonstrationPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration, DelayTime: 1000})
	if len(m.effects().Active) != 4 || m.effects().Active[0].EffectID != effects.ThrowItem4 {
		t.Fatal("Acid Demonstration must throw its bottles during casting, with circle, aura, and lock-on")
	}
	e := m.effects().Active[0]
	if e.ActorID != 2000000 || e.TargetID != 300 || e.ActorOrigin.X != 10.5 || e.ProjectileDirection.X != 10 {
		t.Fatal("bottle flight must capture the caster and victim at cast start")
	}
	cast := m.actorSystem.ActorAnims[2000000].Started
	m.effects().Active = nil
	m.scheduledSounds = nil
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration, HitCount: 10, Damage: 100, SourceSpeed: 500, Action: network.ActorActionSkill}
	m.applyActorActionNotify(ctx, a)
	if !m.actorSystem.ActorAnims[2000000].Started.Equal(cast) || len(m.effects().Active) != 11 {
		t.Fatal("Acid Demonstration must keep its cast pose, start one impact ring, and flash for each hit")
	}
	e = m.effects().Active[0]
	if e.EffectID != effects.AcidDemon || !e.HasActorTransform || e.ActorOrigin.X != 20.5 || !e.Starts.Before(m.effects().Active[1].Starts) {
		t.Fatal("Acid Demonstration rings must capture the victim's position at release, before hit flashes")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 11 {
		t.Fatal("secondary damage packet replayed Acid Demonstration")
	}
	a.Action, a.Damage = network.ActorActionSkill, 0
	m.effects().Active = nil
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.AcidDemon {
		t.Fatal("a miss must still show the ground impact, without hit flashes")
	}
	m = &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.ThrowItem4 {
		t.Fatal("a zero-duration server cast must still throw its bottles")
	}
}

func TestDancerScreamAndSlingingArrow(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillDCScream, SourceID: 2000000, TargetID: 300, Result: 1})
	anim := m.actorSystem.ActorAnims[2000000]
	if !anim.HasFixedMotion || anim.FixedMotion != 0 || anim.ActionFamily != sprites.ActionPCSkill || anim.Duration != 1200*time.Millisecond {
		t.Fatal("Scream must hold skill frame zero for 1200ms")
	}
	if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 2000000 || len(m.scheduledSounds) != 0 {
		t.Fatal("Scream's silent chat effect must belong to the caster")
	}
	if _, ok := m.actors().ActorAnimation(2000000, anim.Started.Add(1200*time.Millisecond)); ok {
		t.Fatal("Scream pose outlived its native duration")
	}
	m = &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(sprites.ActionPCAttack3, 4, 2, []string{"atk"})}}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillDCThrowarrow, SourceSpeed: 500, HitCount: 1, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	anim = m.actorSystem.ActorAnims[a.SourceID]
	if len(m.effects().Active) != 2 || anim.ActionFamily != sprites.ActionPCAttack3 {
		t.Fatal("Slinging Arrow needs ACT 88, one arrow, and one impact")
	}
	arrow, hit := m.effects().Active[0], m.effects().Active[1]
	if arrow.EffectID != effects.ArrowShot || hit.EffectID != effects.Hit1 || arrow.Starts.Sub(anim.Started) != 250*time.Millisecond || hit.Starts.Sub(arrow.Starts) != scene.ReferenceBowArrowFlightDuration {
		t.Fatal("Slinging Arrow must use the ACT marker and native arrow flight time")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 2 {
		t.Fatal("splash notification replayed Slinging Arrow")
	}
}

func TestGunslingerNativePacketOwnership(t *testing.T) {
	for skill, id := range map[uint16]int{501: 548, 502: 648, 503: 649, 507: 644, 512: 647, 514: 623, 515: 643, 518: 628, 519: 584, 520: 645} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, HitCount: 5}, start)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != id {
			t.Fatalf("skill %d must launch once per packet, got %+v", skill, m.effects().Active)
		}
		e := m.effects().Active[0]
		owner := uint32(300)
		if id == 623 || id == 584 || id == 645 {
			owner = 2000000
			if e.ProjectileDirection.X <= 0 || e.ActorOrigin.X != 10.5 {
				t.Fatalf("caster direction/origin lost: %+v", e)
			}
		}
		if e.ActorID != owner {
			t.Fatalf("skill %d owner=%d want %d", skill, e.ActorID, owner)
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d replayed on no-damage", skill)
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
		owner uint32
		sound string
	}{
		{500, 0, 0, "effect\\플립.wav"}, {508, 0, 0, "effect\\크래커.wav"},
		{504, 625, 2000000, ""}, {505, 626, 2000000, ""}, {506, 456, 2000000, "effect\\ef_incagility.wav"},
		{513, 627, 300, "effect\\디스암.wav"}, {516, 637, 2000000, "effect\\데스페라도.wav"}, {517, 626, 2000000, ""},
	} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		if tc.id != 0 {
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != tc.owner {
				t.Fatalf("release %d: %+v", tc.skill, m.effects().Active)
			}
		} else if len(m.effects().Active) != 0 {
			t.Fatal("sound-only skill invented a visual")
		}
		if tc.sound != "" {
			if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{tc.sound}) {
				t.Fatalf("skill %d sound %+v", tc.skill, m.scheduledSounds)
			}
		} else if len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d should be silent", tc.skill)
		}
		if tc.id == 0 && m.scheduledSounds[0].positioned {
			t.Fatal("Flip Coin/Cracker should have centered sound")
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now())
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("release %d replayed on damage", tc.skill)
		}
		n.Result = 0
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("release %d played on failure", tc.skill)
		}
	}
	// Passive accuracy/double-attack skills have no independent effect.
	for _, id := range []uint16{509, 510, 511} {
		testutil.ExpectEffectIDs(t, "passive", effects.SkillEffectIDs(id))
		testutil.ExpectEffectIDs(t, "passive hit", effects.SkillBeforeHitEffectIDs(id))
	}
}

func TestHighWizardSoulDrainPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHWSouldrain, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Drain needs its returning souls and victim tint, without a sound")
	}
	drain, victim := m.effects().Active[0], m.effects().Active[1]
	if drain.EffectID != 378 || drain.ActorID != n.SourceID || drain.TargetID != n.TargetID || victim.EffectID != 379 || victim.ActorID != n.TargetID {
		t.Fatal("Soul Drain effects belong to the wrong actors")
	}
	if drain.ActorOrigin.X != 20.5 || drain.ProjectileDirection.X != -10 {
		t.Fatal("souls must travel from the defeated monster back to the caster")
	}
	anim := m.actorSystem.ActorAnims[n.SourceID]
	if anim.ActionFamily*8 != 88 || anim.Length != 4 || anim.Speed != 10*db.EffectFrameDuration || anim.Duration != time.Second {
		t.Fatal("Soul Drain must force ACT 88 frames 0–3 for one second")
	}
	for _, tick := range []int{1, 30, 60} {
		p := effects.SoulDrainPosition(drain, tick, 0)
		if tick == 30 && (p.X <= 10.5 || p.X >= 25 || p.Y < 7) || tick == 60 && math.Abs(p.X-10.1) > .00001 {
			t.Fatalf("incorrect native spline at tick %d: %+v", tick, p)
		}
	}
	n.Result = 0
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 2 {
		t.Fatal("failed skill must not trigger Soul Drain")
	}
	n.Result = 1
	ctx.World.Player.Moving = true
	ctx.World.Player.MoveStarted, ctx.World.Player.MoveDuration = time.Now(), 5*time.Second
	ctx.World.Player.FromX, ctx.World.Player.ToX = 10, 15
	ctx.World.Player.FromY, ctx.World.Player.ToY = 20, 20
	m.applySkillNoDamageNotify(ctx, n)
	if !ctx.World.Player.Moving || ctx.World.Player.ToX != 15 {
		t.Fatal("Soul Drain's forced animation must not discard an existing walk")
	}
	delete(ctx.World.Actors, n.TargetID)
	m.effects().Active = nil
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 0 {
		t.Fatal("Soul Drain requires both the caster and victim")
	}
}

func TestHunterBlitzBeatFirstHitOnly(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHTBlitzbeat, Damage: 500, HitCount: 5, SourceSpeed: 500}
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.BlitzBeat || m.effects().Active[0].ActorID != 300 {
		t.Fatal("Blitz Beat needs one victim effect for the whole attack")
	}
	e := m.effects().Active[0]
	for i, delay := range []time.Duration{5 * db.EffectFrameDuration, 30 * db.EffectFrameDuration} {
		found := false
		for _, sound := range m.scheduledSounds {
			if sound.at.Equal(e.Starts.Add(delay)) && sound.paths[0] == db.EffectSpecs[effects.BlitzBeat].SFX[i] {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing Blitz Beat sound at %s", delay)
		}
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.effects().Active) != 1 {
		t.Fatal("splash victim replayed Blitz Beat")
	}
}

func TestHunterPhantasmicArrowRelease(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	family := sprites.AttackActionFamily(nil, ctx.World.Player)
	m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(family, 4, 2, []string{"atk"})}}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHTPhantasmic, Damage: 100, HitCount: 1, SourceSpeed: 500})
	if m.actorSystem.ActorAnims[2000000].ActionFamily != family || len(m.effects().Active) != 4 {
		t.Fatal("Phantasmic Arrow must use the normal bow pose, flash, arrow, and two impacts")
	}
	arrow := m.effects().Active[1]
	if arrow.EffectID != effects.ArrowShot || arrow.Starts.Sub(m.actorSystem.ActorAnims[2000000].Started) != 250*time.Millisecond || arrow.Expires.Sub(arrow.Starts) != scene.ReferenceBowArrowFlightDuration {
		t.Fatalf("wrong arrow timing: %+v", arrow)
	}
	for _, e := range m.effects().Active[2:] {
		if !e.Starts.Equal(arrow.Expires) {
			t.Fatal("impact precedes arrow arrival")
		}
	}
}

func TestKnightSplashDoesNotReplayCasterOrEffects(t *testing.T) {
	for _, skill := range []uint16{db.SkillKNBrandishspear, db.SkillKNSpearstab, db.SkillKNBowlingbash} {
		ctx := testutil.KnightEffectContext()
		ctx.World.Player.Dir = 2
		m := &WorldMode{}
		starts := time.Now()
		m.setCombatActorAction(ctx, 2000000, scene.ActorAnimation{ActionFamily: sprites.ActionPCAttack1, Started: starts, Duration: time.Second})
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 50, HitCount: 1, Action: 5,
		})
		if len(m.effects().Active) != 0 {
			t.Fatalf("splash skill %d replayed effects: %+v", skill, m.effects().Active)
		}
		if !m.actorSystem.ActorAnims[2000000].Started.Equal(starts) || ctx.World.Player.Dir != 2 {
			t.Fatalf("splash skill %d restarted or turned the caster", skill)
		}
	}
}

func TestKnightQuickenAndAutoCounter(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, skill := range []uint16{db.SkillKNTwohandquicken, db.SkillKNOnehand} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed Quicken played an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 2000000 || m.effects().Active[0].EffectID != effects.TwoHandQuicken || len(m.scheduledSounds) != 1 {
			t.Fatal("Quicken must play once on the caster")
		}
	}
	ctx.Session.Selected = session.Character{ID: 100, Job: db.JobKnight, Weapon: db.WeaponSword}
	for _, state := range []string{"ready", "expired", "cancelled"} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillKNAutocounter, SourceID: 2000000, TargetID: 2000000, Result: 1})
		anim := m.actorSystem.ActorAnims[2000000]
		if !anim.AutoCounter || anim.ActionFamily != sprites.ActionPCAttack2 || !anim.HasFixedMotion || anim.FixedMotion != 0 || anim.Duration != 1500*time.Millisecond || len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("Auto Counter must silently hold the first attack frame: %+v", anim)
		}
		if state == "expired" {
			anim.Started = time.Now().Add(-2 * time.Second)
			m.setCombatActorAction(ctx, 2000000, anim)
		} else if state == "cancelled" {
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, Action: network.ActionSitDown})
		}
		action := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, Action: 10, Damage: 50, HitCount: 1}
		m.applyActorActionNotify(ctx, action)
		count := 0
		for _, effect := range m.effects().Active {
			if effect.EffectID == effects.AutoCounter {
				count++
			}
		}
		want := 0
		if state == "ready" {
			want = 1
		}
		if count != want {
			t.Fatalf("state=%s counter effects=%d, want %d", state, count, want)
		}
		before := len(m.effects().Active)
		m.applyActorActionNotify(ctx, action)
		for _, effect := range m.effects().Active[before:] {
			if effect.EffectID == effects.AutoCounter {
				t.Fatal("a normal attack replayed an already consumed counter effect")
			}
		}
	}
}

func TestLordKnightAttackEffects(t *testing.T) {
	for _, tc := range []struct {
		skill              uint16
		effect, hit, count int
		owner              uint32
	}{
		{db.SkillLKSpiralpierce, effects.Magnum2, effects.Pierce, 1, 300},
		{db.SkillLKHeadcrush, effects.Bash3D3, effects.Hit1, 5, 2000000},
		{db.SkillLKJointbeat, effects.Bash3D4, effects.Hit1, 5, 2000000},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 500, HitCount: 5, Action: 8}
		m.addSkillBeginEffect(ctx, action, start)
		m.addSkillEffect(ctx, action, start, hit)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != tc.owner || !m.effects().Active[0].Starts.Equal(start) {
			t.Fatalf("skill %d must start once on %d: %+v", tc.skill, tc.owner, m.effects().Active)
		}
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.effects().Active) != 1+tc.count {
			t.Fatalf("skill %d: wrong impact count: %+v", tc.skill, m.effects().Active)
		}
		for i, effect := range m.effects().Active[1:] {
			if effect.EffectID != tc.hit || effect.ActorID != 300 || !effect.Starts.Equal(hit.Add(time.Duration(i)*scene.MultiHitDelay)) {
				t.Fatalf("skill %d: misplaced or mistimed impact: %+v", tc.skill, effect)
			}
		}
		m = &WorldMode{}
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.effects().Active) != 0 {
			t.Fatal("miss displayed an impact")
		}
		action.Damage, action.Action = 500, 5
		m.applyActorActionNotify(ctx, action)
		if len(m.effects().Active) != 0 {
			t.Fatal("splash notification replayed effects")
		}
	}
}

func TestLordKnightBuffsAndCast(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
	}{
		{db.SkillLKAurablade, effects.AuraBlade},
		{db.SkillLKConcentration, effects.LKConcentration},
		{db.SkillLKParrying, effects.Guard},
		{db.SkillLKBerserk, effects.RedBody},
	} {
		ctx := testutil.KnightEffectContext()
		ctx.Session.Selected = session.Character{ID: 100, Job: db.JobKnightH}
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed buff played an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		m.applySkillNoDamageNotify(ctx, notify)
		count := 0
		for _, effect := range m.effects().Active {
			if effect.EffectID == tc.effect {
				count++
				if effect.ActorID != 2000000 {
					t.Fatalf("skill %d buff must belong to caster", tc.skill)
				}
			}
		}
		if tc.skill != db.SkillLKParrying && count != 1 {
			t.Fatalf("skill %d stacked buff effects", tc.skill)
		}
		if tc.skill == db.SkillLKParrying && m.actorSystem.ActorAnims[2000000].ActionFamily != scene.SpriteActionIdle {
			t.Fatal("Parrying must use standing pose")
		}
		if tc.skill == db.SkillLKAurablade {
			anim := m.actorSystem.ActorAnims[2000000]
			if anim.ActionFamily*8 != 32 || anim.Length != 6 || anim.Duration != time.Second || anim.Speed != 10*db.EffectFrameDuration {
				t.Fatalf("Aura Blade must animate native action 32, frames 0..5: %+v", anim)
			}
		}
	}
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.effects().AddSkillCastEffects(ctx, db.SkillLKSpiralpierce, 0, 2000000, 300, 0, 0, 2*time.Second, start, "test")
	if !m.effects().HasActive(effects.PierceBody, 2000000, start) || len(m.scheduledSounds) != 1 {
		t.Fatal("Spiral Pierce must use its yellow casting body and begin-spell sound")
	}
	if m.effects().HasActive(effects.PierceBody, 2000000, start.Add(2*time.Second+time.Nanosecond)) {
		t.Fatal("Spiral Pierce body outlasted its cast")
	}
	// Relax's sitting state is sent by the server; it has no extra effect.
	m = &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillLKTensionrelax, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
		t.Fatal("Relax invented an effect or sound")
	}
}

func TestMageSuccessAndSplashEffects(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill uint16
		hits  []int
	}{
		{db.SkillMGNapalmbeat, []int{effects.Hit1, effects.BashHit}},
		{db.SkillMGSoulstrike, []int{effects.Hit1, effects.BashHit}},
		{db.SkillMGFrostdiver, []int{effects.ColdHit, effects.FrostDiverHit}},
		{db.SkillMGFireball, []int{effects.FireHit, effects.BashHit}},
		{db.SkillMGFirewall, []int{effects.Hit1}},
		{db.SkillMGThunderstorm, []int{effects.Hit1}},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 2, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.effects().Active)
			m.addSkillHitEffect(ctx, action, hit)
			if splash && tc.skill != db.SkillMGNapalmbeat {
				if len(m.effects().Active) != 0 {
					t.Fatalf("skill %d replayed effects for a splash victim", tc.skill)
				}
				continue
			}
			impacts := m.effects().Active[before:]
			if len(impacts) != len(tc.hits)+1 {
				t.Fatalf("skill %d: expected two base impacts and one secondary impact", tc.skill)
			}
			for i, id := range tc.hits {
				if impacts[i].EffectID != id || impacts[i].ActorID != 300 || !impacts[i].Starts.Equal(hit) {
					t.Fatalf("skill %d: wrong first impact", tc.skill)
				}
			}
			last := impacts[len(impacts)-1]
			if last.EffectID != tc.hits[0] || !last.Starts.Equal(hit.Add(scene.MultiHitDelay)) {
				t.Fatalf("skill %d: wrong subsequent impact", tc.skill)
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
		actor uint32
	}{
		{db.SkillMGStonecurse, effects.StoneCurse, 300},
		{db.SkillMGEnergycoat, effects.EnergyCoat, 2000000},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill played an effect or sound")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != tc.actor {
			t.Fatalf("skill %d: incorrect success effect", tc.skill)
		}
		if tc.skill == db.SkillMGStonecurse && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "_stonecurse.wav") {
			t.Fatal("Stone Curse must play its original sound on success")
		}
	}
}

func TestMerchantUtilityEffects(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillMCLoud, SourceID: 2000000, TargetID: 300}
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
		t.Fatal("failed Crazy Uproar must be silent")
	}
	notify.Result = 1
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Loud || m.effects().Active[0].ActorID != 2000000 || len(m.scheduledSounds) != 1 {
		t.Fatalf("Crazy Uproar must show only Loud on its caster: %+v", m.effects().Active)
	}
	for _, skill := range []uint16{db.SkillMCInccarry, db.SkillMCDiscount, db.SkillMCOvercharge, db.SkillMCPushcart, db.SkillMCIdentify, db.SkillMCVending, db.SkillMCChangecart} {
		m := &WorldMode{}
		notify.SkillID = skill
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("utility/passive skill %d must not invent an effect", skill)
		}
	}
}

func TestCallSpiritsFinishesStandingWithoutAnotherSkillAnimation(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOCallspirits, DelayTime: 1000})
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOCallspirits, Result: 1})
	if anim := m.actorSystem.ActorAnims[150000]; anim.ActionFamily != scene.SpriteActionIdle {
		t.Fatalf("Call Spirits result animation = %+v, want standing", anim)
	}
	if ctx.World.Player.SpiritSpheres != 0 {
		t.Fatal("skill result must not invent a sphere count")
	}
}

func TestFuryHasOnlyActivationSound(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOExplosionspirits, DelayTime: 1000})
	if len(m.scheduledSounds) != 0 {
		t.Fatalf("Fury cast scheduled unwanted sound: %+v", m.scheduledSounds)
	}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOExplosionspirits, Result: 1})
	if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{"effect\\mon_폭기.wav"}) {
		t.Fatalf("Fury activation sounds = %+v", m.scheduledSounds)
	}
}

func TestMonkSkillResultsUseDedicatedPoses(t *testing.T) {
	for _, tc := range []struct {
		skill         uint16
		family, frame int
		play          bool
	}{
		{db.SkillMOInvestigate, sprites.ActionPCSkill, 3, true},
		{db.SkillMOFingeroffensive, sprites.ActionPCSkill, 3, true},
		{db.SkillMOChaincombo, sprites.ActionPCSkill, 5, false},
		{db.SkillMOCombofinish, sprites.ActionPCAttack2, 3, true},
		{db.SkillMOTripleattack, sprites.ActionPCAttack3, 0, true},
		{db.SkillMOExplosionspirits, sprites.ActionPCSkill, 1, true},
	} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SkillLevel: 5, SourceSpeed: 800, Damage: 100, HitCount: 1, Action: network.ActorActionSkill})
		anim := m.actorSystem.ActorAnims[150000]
		if anim.ActionFamily != tc.family || anim.FrameOffset != tc.frame || anim.Play != tc.play {
			t.Fatalf("skill %d pose = %+v", tc.skill, anim)
		}
		if _, active := m.actors().ActorAnimation(150000, anim.Started.Add(2*time.Second)); active {
			t.Fatalf("skill %d pose never released", tc.skill)
		}
	}
}

func TestTripleAttackAnimationWithClientAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, job := range []int16{db.JobMonk, db.JobMonkH} {
		for _, sex := range []byte{0, 1} {
			ctx.Session.Selected.Job, ctx.Session.Sex = job, sex
			view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
			if view == nil || view.Body == nil {
				t.Fatalf("job %d sex %d: %s", job, sex, status)
			}
			actor, _, _ := scene.ActorForCombatID(ctx, ctx.Session.AccountID)
			m := &WorldMode{}
			m.startSkillActionAnimation(ctx, actor.ID, actor, effects.SkillAction(db.SkillMOTripleattack), start, 1500*time.Millisecond)
			anim := m.actorSystem.ActorAnims[ctx.Session.CharID]
			state := sprites.State{}
			actors.ApplyActorAnimationToSpriteState(&state, anim, true)
			for direction := 0; direction < 8; direction++ {
				index, action, ok := sprites.ResolveAction(view.Body.Act, anim.ActionFamily, direction)
				if !ok || index != 88+direction || len(action.Animations) < 8 {
					t.Fatalf("job %d sex %d direction %d: action=%d frames=%d, want ACT 88+direction with eight frames", job, sex, direction, index, len(action.Animations))
				}
				for frame := 0; frame < 8; frame++ {
					at := start.Add(time.Duration(frame)*4*db.EffectFrameDuration + time.Millisecond)
					if got := sprites.BodyMotionForState(action, state, start, at); got != frame {
						t.Fatalf("job %d sex %d direction %d: frame=%d, want %d", job, sex, direction, got, frame)
					}
				}
				if got := sprites.BodyMotionForState(action, state, start, start.Add(time.Second)); got != 7 {
					t.Fatalf("job %d sex %d direction %d: held frame=%d, want 7", job, sex, direction, got)
				}
			}
		}
	}
}

func TestComboDelayBeforeSkillResult(t *testing.T) {
	for _, sourceID := range []uint32{2000000, 150000, 300} {
		for _, skillID := range []uint16{db.SkillMOTripleattack, db.SkillMOChaincombo, db.SkillMOCombofinish, db.SkillChTigerfist} {
			t.Run(fmt.Sprintf("%d/%d", sourceID, skillID), func(t *testing.T) {
				ctx := testutil.MonkEffectContext()
				m := &WorldMode{}
				data := make([]byte, 10)
				binary.LittleEndian.PutUint16(data, network.PacketZCComboDelay)
				binary.LittleEndian.PutUint32(data[2:], sourceID)
				binary.LittleEndian.PutUint32(data[6:], 1500)
				m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCComboDelay, Data: data}, time.Now())
				targetID := uint32(300)
				if sourceID == targetID {
					targetID = 2000000
				}
				m.applyActorActionNotify(ctx, network.ActorActionNotify{
					SourceID: sourceID, TargetID: targetID, SkillID: skillID,
					SkillLevel: 5, SourceSpeed: 800, Damage: 100, HitCount: 1, Action: network.ActorActionSkill,
				})
				ids := []uint32{sourceID}
				if scene.IsLocalActor(ctx, sourceID) {
					ids = []uint32{ctx.Session.AccountID, ctx.Session.CharID}
				}
				for _, id := range ids {
					anim := m.actorSystem.ActorAnims[id]
					if !anim.Combo || anim.Duration != 1500*time.Millisecond {
						t.Fatalf("actor %d: combo animation = %+v", id, anim)
					}
					if skillID == db.SkillMOTripleattack && (!anim.HasSpeed || anim.Speed != 4*db.EffectFrameDuration || anim.Length != 8) {
						t.Fatalf("Triple Attack must play at a fixed speed before holding its final pose: %+v", anim)
					}
					if _, active := m.actors().ActorAnimation(id, anim.Started.Add(time.Second)); !active {
						t.Fatal("combo pose ended at the ordinary attack duration")
					}
					if _, active := m.actors().ActorAnimation(id, anim.Started.Add(1500*time.Millisecond)); active {
						t.Fatal("combo pose outlived the server delay")
					}
				}
			})
		}
	}
}

func TestComboDelayUpdatesExistingPose(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	start := time.Unix(100, 0)
	for _, sourceID := range []uint32{2000000, 300} {
		for _, delay := range []uint32{0, 1200} {
			m := &WorldMode{}
			actor, _, _ := scene.ActorForCombatID(ctx, sourceID)
			m.startSkillActionAnimation(ctx, sourceID, actor, effects.SkillAction(db.SkillMOTripleattack), start, 800*time.Millisecond)
			before := m.actorSystem.ActorAnims[sourceID]
			received := start.Add(200 * time.Millisecond)
			m.actors().ApplyComboDelay(ctx, network.ActorComboDelay{ID: sourceID, DelayTime: delay}, received)
			ids := []uint32{sourceID}
			if scene.IsLocalActor(ctx, sourceID) {
				ids = append(ids, ctx.Session.CharID)
			}
			for _, id := range ids {
				after := m.actorSystem.ActorAnims[id]
				if after.Started != before.Started || after.Speed != before.Speed || after.FrameOffset != before.FrameOffset {
					t.Fatalf("delay restarted or retimed the animation: before=%+v after=%+v", before, after)
				}
				end := received.Add(time.Duration(delay) * time.Millisecond)
				if after.Started.Add(after.Duration) != end {
					t.Fatalf("combo end = %v, want %v", after.Started.Add(after.Duration), end)
				}
				if _, active := m.actors().ActorAnimation(id, end); active {
					t.Fatal("combo pose did not expire")
				}
			}
			// Zero is an explicit duration, not a request to use the default.
			m.startSkillActionAnimation(ctx, sourceID, actor, effects.SkillAction(db.SkillMOChaincombo), received, 800*time.Millisecond)
			if _, active := m.actors().ActorAnimation(sourceID, received); active != (delay > 0) {
				t.Fatalf("cached delay %d: active=%v", delay, active)
			}
		}
	}
}

func TestComboDelayLeavesOtherActionsUnchanged(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	actor, _, _ := scene.ActorForCombatID(ctx, 300)
	m.startSkillActionAnimation(ctx, 300, actor, effects.SkillAction(db.SkillMOInvestigate), start, 800*time.Millisecond)
	before := m.actorSystem.ActorAnims[300]
	m.actors().ApplyComboDelay(ctx, network.ActorComboDelay{ID: 300, DelayTime: 2000}, start)
	if after := m.actorSystem.ActorAnims[300]; !reflect.DeepEqual(after, before) {
		t.Fatalf("combo packet changed an unrelated skill: %+v", after)
	}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 300, TargetID: 2000000, SourceSpeed: 800, Damage: 1, HitCount: 1})
	if anim := m.actorSystem.ActorAnims[300]; anim.Combo || anim.Duration != 800*time.Millisecond {
		t.Fatalf("cached combo delay affected a normal attack: %+v", anim)
	}
	m.removeActorNow(ctx, 300)
	if _, ok := m.actorSystem.ComboDelays[300]; ok {
		t.Fatal("removed actor retained its combo delay")
	}
	m.actors().ApplyComboDelay(ctx, network.ActorComboDelay{ID: 300, DelayTime: 2000}, start)
	if len(m.actorSystem.ComboDelays) != 0 {
		t.Fatal("unknown actor's combo packet recreated its state")
	}
}

func TestBladeStopWaitingPoseExpires(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOBladestop, Result: 1})
	anim := m.actorSystem.ActorAnims[150000]
	if anim.ActionFamily != sprites.ActionPCSkill || anim.FixedMotion != 3 || anim.Play {
		t.Fatalf("Blade Stop waiting pose = %+v", anim)
	}
	if _, active := m.actors().ActorAnimation(150000, anim.Started.Add(2*time.Second)); !active {
		t.Fatal("Blade Stop waiting pose ended too soon")
	}
	if _, active := m.actors().ActorAnimation(150000, anim.Started.Add(3*time.Second)); active {
		t.Fatal("Blade Stop waiting pose did not expire")
	}
}

func TestMonkAttackActivationSounds(t *testing.T) {
	for _, tc := range []struct {
		skill uint16
		sound string
	}{
		{db.SkillMOExtremityfist, "effect\\mon_아수라 패황권.wav"},
		{db.SkillMOCombofinish, "effect\\mon_맹룡과강.wav"},
	} {
		for _, damage := range []int32{100, 0, -1} {
			ctx := testutil.MonkEffectContext()
			m := &WorldMode{}
			action := network.ActorActionNotify{
				SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SkillLevel: 5,
				SourceSpeed: 800, Damage: damage, HitCount: 3, Action: network.ActorActionSkill,
			}
			m.applyActorActionNotify(ctx, action)
			start := m.actorSystem.ActorAnims[150000].Started
			count := 0
			for _, sound := range m.scheduledSounds {
				if reflect.DeepEqual(sound.paths, []string{tc.sound}) {
					count++
					if sound.actorID != action.SourceID || !sound.at.Equal(start) {
						t.Fatalf("skill %d activation sound must play at the caster on release: %+v", tc.skill, sound)
					}
				}
			}
			if count != 1 {
				t.Fatalf("skill %d damage %d: got %d activation sounds, want one", tc.skill, damage, count)
			}
			if tc.skill == db.SkillMOExtremityfist {
				m = &WorldMode{}
				action.Action = 5
				m.applyActorActionNotify(ctx, action)
				for _, sound := range m.scheduledSounds {
					if reflect.DeepEqual(sound.paths, []string{tc.sound}) {
						t.Fatal("splash packet replayed Asura's activation sound")
					}
				}
			}
		}
	}
}

func TestAsuraGlyphsBelongToCasterDuringCastOnly(t *testing.T) {
	for _, job := range []int16{db.JobMonk, db.JobMonkH} {
		ctx := testutil.MonkEffectContext()
		ctx.World.Player.Job = job
		ctx.Session.Selected.Job = job
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOExtremityfist, DelayTime: 3000})
		want := effects.BeginAsura
		if job == db.JobMonkH {
			want = effects.BeginAsura11
		}
		if !m.effects().HasActive(want, 2000000, time.Now()) || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\EF_BeginSpell.wav" {
			t.Fatalf("Asura cast = %+v sounds=%+v", m.effects().Active, m.scheduledSounds)
		}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOExtremityfist, Action: network.ActorActionSkill, Damage: 100, HitCount: 1})
		glyphs := 0
		for _, effect := range m.effects().Active {
			if effect.EffectID == effects.BeginAsura || effect.EffectID == effects.BeginAsura11 {
				glyphs++
				if effect.ActorID != 2000000 {
					t.Fatal("Asura glyphs appeared on victim")
				}
			}
		}
		if glyphs != 1 {
			t.Fatalf("Asura replayed glyphs on impact: %d sequences", glyphs)
		}
	}
}

func TestNinjaBeforeHitRouting(t *testing.T) {
	for skill, id := range map[uint16]int{523: 613, 524: 614, 525: 615, 526: 616, 528: 632, 530: 630, 534: 618, 537: 619, 540: 621, 542: 624, 544: 633} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: skill, SkillLevel: 10, HitCount: 3, SourceID: 2000000, TargetID: 300}, start)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != id {
			t.Fatalf("skill %d needs one launch, got %+v", skill, m.effects().Active)
		}
		e := m.effects().Active[0]
		if id >= 613 && id <= 616 {
			if e.ActorID != 2000000 || e.TargetID != 300 || !e.HasActorTransform || e.ProjectileDirection.X <= 0 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned {
				t.Fatalf("throw must fly from caster to target with centered sound: %+v", e)
			}
		}
		if id == 621 && (e.HitCount != 3 || e.ActorID != 2000000 || e.TargetID != 300 || e.ActorOrigin.X != 10.5 || e.NinjaWind == nil) {
			t.Fatalf("wind must launch from caster with packet hit count: %+v", e)
		}
		if id == 618 && (e.NinjaPetals == nil || len(m.scheduledSounds) != 4 || m.scheduledSounds[0].at.Sub(start) != 12*db.EffectFrameDuration) {
			t.Fatal("Fire Petal needs drifting petals and four delayed sounds")
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d duplicated its damage effect on no-damage packet", skill)
		}
	}
	testutil.ExpectEffectIDs(t, "Fire Petal impact", effects.SkillHitEffectIDs(534), effects.FireHit)
	testutil.ExpectEffectIDs(t, "Ice Spear impact", effects.SkillHitEffectIDs(537), effects.ColdHit)
}

func TestPaladinAttackEffects(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		caster bool
	}{
		{db.SkillCRShieldcharge, effects.ShieldCharge, false},
		{db.SkillCRShieldboomerang, effects.ShieldBoomer, true},
		{db.SkillCRHolycross, effects.HolyCross, false},
		{db.SkillCRGrandcross, effects.GrandCross, true},
		{db.SkillPaPressure, effects.Pressure, false},
		{db.SkillPaSacrifice, effects.Bash3D, true},
		{db.SkillPaShieldchain, effects.ShieldBoomerang3, false},
	} {
		ctx := testutil.KnightEffectContext()
		ctx.World.Player.Job = db.JobCrusaderH
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 500, HitCount: 5, Action: 8}
		m.addSkillEffect(ctx, action, start, hit)
		m.addSkillBeforeHitEffect(ctx, action, start)
		owner := uint32(300)
		if tc.caster {
			owner = 2000000
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != owner || !m.effects().Active[0].Starts.Equal(start) {
			t.Fatalf("skill %d must launch once at action start on %d: %+v", tc.skill, owner, m.effects().Active)
		}
		if tc.skill == db.SkillCRHolycross && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned) {
			t.Fatal("Holy Cross must play one unpositioned sound")
		}
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.effects().Active) != 6 {
			t.Fatalf("skill %d must retain each ordinary impact, got %d", tc.skill, len(m.effects().Active))
		}
		for i, effect := range m.effects().Active[1:] {
			if effect.EffectID != effects.Hit1 || effect.ActorID != 300 || !effect.Starts.Equal(hit.Add(time.Duration(i)*scene.MultiHitDelay)) {
				t.Fatalf("skill %d impact %d: %+v", tc.skill, i, effect)
			}
		}
		action.Action = 5
		m = &WorldMode{}
		m.applyActorActionNotify(ctx, action)
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d replayed its effect on a splash victim", tc.skill)
		}
	}
}

func TestPaladinBuffOwnershipAndSacrificeActivation(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		target uint32
	}{
		{db.SkillCRAutoguard, effects.Guard, 2000000},
		{db.SkillCRReflectshield, effects.ReflectShield, 2000000},
		{db.SkillCRDefender, effects.CrusaderDef, 2000000},
		{db.SkillCRDevotion, effects.Devotion, 2000000},
		{db.SkillCRSpearquicken, effects.SpearQuicken, 2000000},
		{db.SkillCRProvidence, effects.Providence, 300},
		{db.SkillCRShrink, effects.Shrink, 2000000},
		{db.SkillPaSacrifice, -1, 0},
	} {
		m := &WorldMode{}
		ctx := testutil.KnightEffectContext()
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed cast played effects")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if tc.effect == -1 {
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("Sacrifice activation must not play its attack effect")
			}
		} else if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != tc.target {
			t.Fatalf("skill %d affected the wrong actor: %+v", tc.skill, m.effects().Active)
		}
	}
}

func TestPaladinCastAurasAndQuickenRefresh(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillPaPressure, effects.BeginSpell6},
		{db.SkillPaShieldchain, effects.BeginSpell6},
		{db.SkillCRGrandcross, effects.BeginSpell7},
		{db.SkillCRProvidence, effects.BeginSpell7},
		{db.SkillCRDevotion, effects.BeginSpell7},
	} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: tc.skill, Property: 5, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.aura {
			t.Fatalf("skill %d needs casting aura %d", tc.skill, tc.aura)
		}
	}
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillCRSpearquicken, SourceID: 2000000, TargetID: 2000000, Result: 1}
	m.applySkillNoDamageNotify(ctx, notify)
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active) != 1 {
		t.Fatal("Spear Quicken must replace its previous STR")
	}
}

func TestPriestBuffAndOffensivePaths(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill uint16
		hits  []int
	}{
		{db.SkillALLResurrection, []int{effects.HolyLight}},
		{db.SkillPRAspersio, []int{effects.Hit1, effects.HolyLight}},
		{db.SkillPRBenedictio, []int{effects.Hit1}},
		{db.SkillPRSanctuary, []int{effects.Hit1}},
		{db.SkillPRTurnundead, []int{effects.Hit1, effects.HolyLight}},
		{db.SkillPRMagnus, []int{effects.Hit1, effects.HolyLight}},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 1, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.effects().Active)
			if tc.skill == db.SkillPRTurnundead && !splash {
				if before != 1 || m.effects().Active[0].EffectID != effects.TurnUndead || m.effects().Active[0].ActorID != 2000000 || !m.effects().Active[0].Starts.Equal(start) {
					t.Fatal("Turn Undead must start its burst on the caster")
				}
				if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(37*db.EffectFrameDuration)) {
					t.Fatal("Turn Undead sound must wait for tick 37")
				}
			} else if before != 0 {
				t.Fatalf("skill %d: damage replayed a buff or ground effect", tc.skill)
			}
			m.addSkillHitEffect(ctx, action, hit)
			want := tc.hits
			if splash {
				want = nil
			}
			if len(m.effects().Active)-before != len(want) {
				t.Fatalf("skill %d splash=%t: wrong impact count", tc.skill, splash)
			}
			for i, id := range want {
				effect := m.effects().Active[before+i]
				if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(hit) {
					t.Fatalf("skill %d: misplaced impact", tc.skill)
				}
			}
		}
	}
	for _, job := range []int16{db.JobPriest, 1002} {
		actor := ctx.World.Actors[300]
		actor.Job = job
		ctx.World.Actors[300] = actor
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: db.SkillALLResurrection, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed Resurrection played effects")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		want := 1
		if job == db.JobPriest {
			want = 2
		}
		if len(m.effects().Active) != want || len(m.scheduledSounds) != 1 {
			t.Fatalf("Resurrection on job %d: effects=%d sounds=%d", job, len(m.effects().Active), len(m.scheduledSounds))
		}
	}
}

func TestTurnUndeadDelaysImpactAndDamageTogether(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SkillID: db.SkillPRTurnundead, SourceID: 2000000, TargetID: 300,
			SourceSpeed: 200, Damage: damage, HitCount: 1, Action: network.ActorActionSkill,
		})
		start := m.effects().Active[0].Starts
		if len(m.battleSystem.DamageFloaters) != 1 || m.battleSystem.DamageFloaters[0].Starts.Sub(start) != 700*time.Millisecond {
			t.Fatal("Turn Undead damage/miss must wait for the attack delay plus 500 ms")
		}
		for _, effect := range m.effects().Active[1:] {
			if !effect.Starts.Equal(m.battleSystem.DamageFloaters[0].Starts) {
				t.Fatal("Turn Undead damage and impact disagree")
			}
		}
	}
}

func TestPriestSupportEffectOwners(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill  uint16
		effect int
		owner  uint32
	}{
		{db.SkillPRImpositio, effects.Impositio, 300},
		{db.SkillPRSuffragium, effects.Suffragium, 300},
		{db.SkillPRAspersio, effects.Aspersio, 300},
		{db.SkillPRSlowpoison, effects.SlowPoison, 300},
		{db.SkillPRStrecovery, effects.Recovery, 300},
		{db.SkillPRKyrie, effects.Kyrie, 2000000},
		{db.SkillPRMagnificat, effects.Magnificat, 2000000},
		{db.SkillPRGloria, effects.Gloria, 2000000},
		{db.SkillPRLexdivina, effects.LexDivina, 300},
		{db.SkillPRLexaeterna, effects.LexAeterna, 300},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d: failed cast played effects", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != tc.owner {
			t.Fatalf("skill %d: effect on wrong actor", tc.skill)
		}
	}
}

func TestProfessorSkillPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, tc := range []struct {
		skill  uint16
		owners map[int][]uint32
		sounds int
	}{
		{373, map[int][]uint32{383: {2000000}}, 0},
		{374, map[int][]uint32{384: {2000000}, 385: {300, 2000000}, 407: {300, 2000000}}, 1},
		{375, map[int][]uint32{406: {300}, 380: {300}}, 1},
		{402, map[int][]uint32{403: {300}}, 1},
		{403, map[int][]uint32{505: {2000000}}, 1},
		{482, map[int][]uint32{521: {300}}, 1},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		owners := map[int][]uint32{}
		for _, e := range m.effects().Active {
			owners[e.EffectID] = append(owners[e.EffectID], e.ActorID)
		}
		if !reflect.DeepEqual(owners, tc.owners) || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d: effects=%v sounds=%+v", tc.skill, owners, m.scheduledSounds)
		}
		if m.actorSystem.ActorAnims[n.SourceID].ActionFamily != sprites.ActionPCSkill {
			t.Fatalf("skill %d must use the skill release pose", tc.skill)
		}
		if tc.skill == 374 && (m.effects().Active[0].TargetID != 300 || !m.scheduledSounds[0].positioned) {
			t.Fatal("Soul Change must connect caster to recipient and position its sound at the caster")
		}
		if tc.skill == 402 && m.scheduledSounds[0].at.Sub(m.effects().Active[0].Starts) != 25*db.EffectFrameDuration {
			t.Fatal("the 2008 Mind Breaker sound plays at tick 25")
		}
		if tc.skill == 482 && m.scheduledSounds[0].positioned {
			t.Fatal("Double Casting's activation sound is centered")
		}
		n.Result = 0
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.scheduledSounds) != tc.sounds {
			t.Fatal("failed cast emitted sounds")
		}
	}
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 374, SourceID: 2000000, TargetID: 999, Result: 1})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Change requires both actors")
	}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 374, SourceID: 2000000, TargetID: 300, Result: 1})
	m.effects().RemoveActor(300)
	if len(m.effects().Active) != 2 || m.effects().Active[0].ActorID != 2000000 || m.effects().Active[1].ActorID != 2000000 {
		t.Fatal("recipient removal must remove the connecting beam and recipient effects, retaining the caster flashes")
	}
	// Soul Burn level 5 sends both no-damage and damage packets; failure can hit its caster.
	for _, target := range []uint32{300, 2000000} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 375, SourceID: 2000000, TargetID: target, Result: 1})
		at := time.Now()
		a := network.ActorActionNotify{SkillID: 375, SourceID: 2000000, TargetID: target, Action: network.ActorActionSkill, Damage: 100, HitCount: 1}
		m.addSkillEffect(ctx, a, at, at)
		m.addSkillHitEffect(ctx, a, at)
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != 0 || m.effects().Active[2].ActorID != target || len(m.scheduledSounds) != 1 {
			t.Fatal("Soul Burn damage must add the hit flash without replaying its activation")
		}
		a.Action = 5
		m.addSkillEffect(ctx, a, at, at)
		m.addSkillHitEffect(ctx, a, at)
		if len(m.effects().Active) != 3 {
			t.Fatal("splash packet replayed Soul Burn effects")
		}
	}
}

func TestSightlessMindOneBurstForMultipleVictims(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillRGRaid, SourceID: 2000000, TargetID: 2000000, Result: 1})
	for _, target := range []uint32{300, 301, 302} {
		action := network.ActorActionNotify{SkillID: db.SkillRGRaid, SourceID: 2000000, TargetID: target, Damage: 50}
		m.addSkillEffect(ctx, action, time.Now(), time.Now())
		m.addSkillHitEffect(ctx, action, time.Now())
	}
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.TeiHit3 || m.effects().Active[0].ActorID != 2000000 {
		t.Fatalf("Raid must emit once per cast, not per victim: %+v", m.effects().Active)
	}
}

func TestRogueSuccessEffectsAndDivestSound(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	for _, skill := range []uint16{db.SkillRGStealcoin, db.SkillRGStripweapon, db.SkillRGStripshield, db.SkillRGStriparmor, db.SkillRGStriphelm, db.SkillRGCloseconfine} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d emitted effects", skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		count := 1
		if skill == db.SkillRGStealcoin {
			count = 2
		}
		if len(m.effects().Active) != count || len(m.scheduledSounds) != 1 {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", skill, m.effects().Active, m.scheduledSounds)
		}
		for _, effect := range m.effects().Active {
			if effect.ActorID != 300 {
				t.Fatalf("skill %d effect on wrong actor: %+v", skill, effect)
			}
		}
		if skill >= db.SkillRGStripweapon && skill <= db.SkillRGStriphelm {
			sound := m.scheduledSounds[0]
			if !reflect.DeepEqual(sound.paths, []string{"effect\\복호격.wav"}) || !sound.at.Equal(m.effects().Active[0].Starts.Add(6*db.EffectFrameDuration)) {
				t.Fatalf("divest sound = %+v", sound)
			}
		}
	}
}

func TestSageSupportEffects(t *testing.T) {
	for _, tc := range []struct {
		skill uint16
		id    int
		self  bool
	}{
		{db.SkillSAMagicrod, effects.MagicRod, true},
		{db.SkillSASpellbreaker, effects.SpellBreaker, false},
		{db.SkillSADispell, effects.Dispell, false},
		{db.SkillSAFlamelauncher, effects.FlameLauncher, false},
		{db.SkillSAFrostweapon, effects.FrostWeapon, false},
		{db.SkillSALightningloader, effects.LightningLoad, false},
		{db.SkillSASeismicweapon, effects.SeismicWeapon, false},
		{db.SkillSAElementfire, effects.FlameLauncher, false},
		{db.SkillSAElementwater, effects.FrostWeapon, false},
		{db.SkillSAElementwind, effects.LightningLoad, false},
		{db.SkillSAElementground, effects.SeismicWeapon, false},
	} {
		for _, source := range []uint32{2000000, 300} {
			ctx := testutil.KnightEffectContext()
			m := &WorldMode{}
			target := uint32(300)
			if source == target {
				target = 2000000
			}
			notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: source, TargetID: target}
			m.applySkillNoDamageNotify(ctx, notify)
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatalf("skill %d played an effect on failure", tc.skill)
			}
			notify.Result = 1
			m.applySkillNoDamageNotify(ctx, notify)
			wantActor := target
			if tc.self {
				wantActor = source
			}
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != wantActor {
				t.Fatalf("skill %d: wrong effect recipient: %+v", tc.skill, m.effects().Active)
			}
			if tc.id >= effects.FlameLauncher && tc.id <= effects.SeismicWeapon {
				if len(m.scheduledSounds) != 1 {
					t.Fatalf("skill %d: expected one endowment sound", tc.skill)
				}
				sound := m.scheduledSounds[0]
				if !sound.at.Equal(m.effects().Active[0].Starts.Add(10*db.EffectFrameDuration)) ||
					len(sound.paths) != 1 || sound.paths[0] != "_enemy_hit_wind1.wav" {
					t.Fatalf("skill %d: wrong endowment sound or timing: %+v", tc.skill, sound)
				}
			}
		}
	}
	// These skills use server-driven outcomes (skills, transformations, emotions,
	// healing, crafting), without a separate activation effect in Sakexe.
	for _, skill := range []uint16{
		db.SkillSACastcancel, db.SkillSAAutospell, db.SkillSACreatecon,
		db.SkillSAAbracadabra, db.SkillSAMonocell, db.SkillSAClasschange, db.SkillSASummonmonster,
		db.SkillSAReverseorcish, db.SkillSADeath, db.SkillSAFortune, db.SkillSATamingmonster,
		db.SkillSAQuestion, db.SkillSAGravity, db.SkillSALevelup, db.SkillSAInstantdeath, db.SkillSAFullrecovery, db.SkillSAComa,
	} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(testutil.KnightEffectContext(), network.SkillNoDamageNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1,
		})
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d added a non-native activation effect", skill)
		}
	}
}

func TestSniperCastAndBuffOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Now()
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{{db.SkillSNSharpshooting, effects.BeginSpell6}, {db.SkillSNWindwalk, effects.BeginSpell6}, {db.SkillSNFalconassault, effects.BashBegin}} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, time.Second, start)
		m.effects().AddSkillCastEffects(ctx, tc.skill, 5, 2000000, 300, 0, 0, time.Second, start, "test")
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.aura {
			t.Fatalf("skill %d needs cast circle, lock-on, and native aura", tc.skill)
		}
		if tc.skill == db.SkillSNSharpshooting {
			anim := m.actorSystem.ActorAnims[2000000]
			if anim.ActionFamily*8 != 88 || !anim.HasFixedMotion || anim.FixedMotion != 0 || anim.Play {
				t.Fatal("Sharp Shooting must hold ready-shot frame zero while casting")
			}
		}
	}
	for _, tc := range []struct {
		skill  uint16
		effect int
	}{{db.SkillSNSight, effects.TrueSight}, {db.SkillSNWindwalk, effects.Portal4}} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed buff produced effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != n.SourceID {
			t.Fatalf("skill %d must launch once on its packet source", tc.skill)
		}
		if tc.skill == db.SkillSNSight && m.actorSystem.ActorAnims[n.SourceID].ActionFamily != scene.SpriteActionIdle {
			t.Fatal("True Sight should leave the caster standing")
		}
		if tc.skill == db.SkillSNWindwalk && (!m.effects().Active[0].HasActorTransform || m.effects().Active[0].ActorOrigin.X != 10.5) {
			t.Fatal("Wind Walk must capture the caster's position on activation")
		}
	}
}

func TestSniperDamageTimingAndSplash(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	family := sprites.AttackActionFamily(nil, ctx.World.Player)
	for _, skill := range []uint16{db.SkillSNSharpshooting, db.SkillSNFalconassault} {
		m := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(family, 4, 2, []string{"atk"})}}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: skill, SourceSpeed: 500, HitCount: 5, Damage: 100}
		m.applyActorActionNotify(ctx, a)
		start := m.actorSystem.ActorAnims[a.SourceID].Started
		counts := map[int]int{}
		var arrow effects.Instance
		for _, e := range m.effects().Active {
			counts[e.EffectID]++
			switch e.EffectID {
			case effects.TripleAttack2:
				if e.ActorID != a.SourceID || e.TargetID != a.TargetID || !e.Starts.Equal(start) || e.ProjectileDirection.X != 10 {
					t.Fatal("Sharp Shooting rays must start at the caster and aim through the victim")
				}
			case effects.FalconAssault:
				if e.ActorID != a.TargetID || !e.Starts.Equal(start) || e.Expires.Sub(start) != 200*db.EffectFrameDuration {
					t.Fatal("Falcon Assault's rotation and quake must start on the victim at release")
				}
			case effects.ArrowShot:
				arrow = e
				if e.Starts.Sub(start) != 250*time.Millisecond || e.Expires.Sub(e.Starts) != scene.ReferenceBowArrowFlightDuration {
					t.Fatal("Sharp Shooting's arrow must use the bow ACT marker and eight-tick flight")
				}
			}
		}
		if skill == db.SkillSNSharpshooting {
			if !reflect.DeepEqual(counts, map[int]int{effects.TripleAttack2: 1, effects.ArrowShot: 1, effects.Hit1: 5, effects.BashHit: 1}) {
				t.Fatalf("incorrect Sharp Shooting effects: %v", counts)
			}
			for _, e := range m.effects().Active {
				if e.EffectID == effects.BashHit && !e.Starts.Equal(arrow.Expires) {
					t.Fatal("secondary impact must coincide with arrow arrival")
				}
			}
		} else if !reflect.DeepEqual(counts, map[int]int{effects.FalconAssault: 1, effects.BlitzBeat: 1, effects.Hit1: 5}) {
			t.Fatalf("incorrect Falcon Assault effects: %v", counts)
		}
		before := len(m.effects().Active)
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != before {
			t.Fatal("secondary splash packet replayed the skill")
		}
	}
}

func TestSoulLinkerBuffsUseOriginalVisualsAndOneSound(t *testing.T) {
	for _, tc := range []struct {
		skill uint16
		id    int
		sound string
	}{
		{db.SkillSLKaahi, effects.Hated, "effect\\t_보조마법.wav"},
		{db.SkillSLKaizel, effects.Kaizel, "effect\\priest_resurrection.wav"},
		{db.SkillSLKaupe, effects.BlueBody, "effect\\t_치잉.wav"},
		{db.SkillSLKaite, effects.ReflectBody, "effect\\t_마법반사.wav"},
		{db.SkillSLSke, effects.AsuraBodyMonster, "effect\\t_공격력.wav"},
	} {
		t.Run(db.SkillResourceName[tc.skill], func(t *testing.T) {
			ctx := testutil.SoulLinkerEffectContext()
			m := &WorldMode{}
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 0})
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("failed skill produced effects")
			}
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1})
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != 300 {
				t.Fatalf("effects = %+v", m.effects().Active)
			}
			if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{tc.sound}) {
				t.Fatalf("sounds = %+v", m.scheduledSounds)
			}
		})
	}
}

func TestAllSoulLinkerSkillsHaveCastAndActivationEffects(t *testing.T) {
	active, links := 0, 0
	for skill := uint16(1); skill <= db.SkillSLHigh; skill++ {
		if !db.IsSoulLinkerActiveSkill(skill) {
			continue
		}
		active++
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1500})
		if len(m.scheduledSounds) != 1 || !m.effects().HasActive(effects.BlueCasting, 2000000, time.Now()) {
			t.Fatalf("skill %d lacks blue casting or plays duplicate sounds: %+v", skill, m.scheduledSounds)
		}
		for _, id := range append(effects.SkillEffectIDs(skill), effects.SkillSuccessEffectIDs(skill)...) {
			if _, ok := effects.Lookup(id); !ok {
				t.Fatalf("skill %d has missing effect %d", skill, id)
			}
		}
		if !db.IsSoulLinkSkill(skill) {
			continue
		}
		links++
		m = &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.effects().Active) != 3 || len(m.scheduledSounds) != 2 {
			t.Fatalf("link %d: %+v %+v", skill, m.effects().Active, m.scheduledSounds)
		}
		if m.scheduledSounds[1].at.Sub(m.scheduledSounds[0].at) != 145*db.EffectFrameDuration {
			t.Fatal("Soul Link sound timing")
		}
		anim := m.actorSystem.ActorAnims[150000]
		if anim.ActionFamily != sprites.ActionPCSkill || anim.FixedMotion != 3 || !anim.HasFixedMotion {
			t.Fatalf("link %d caster pose: %+v", skill, anim)
		}
	}
	if active != 26 || links != 16 || db.IsSoulLinkerActiveSkill(db.SkillSLKaina) {
		t.Fatalf("active=%d links=%d; Kaina is passive", active, links)
	}
}

func TestEskaDoesNotPlayFurySound(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSLSka, SourceID: 2000000, TargetID: 300, Result: 1})
	if len(m.scheduledSounds) != 2 {
		t.Fatalf("Eska sounds: %+v", m.scheduledSounds)
	}
	for i, want := range []string{"effect\\t_방어형.wav", "effect\\mon_금강불괴.wav"} {
		if !reflect.DeepEqual(m.scheduledSounds[i].paths, []string{want}) {
			t.Fatalf("Eska sounds: %+v", m.scheduledSounds)
		}
	}
}

func TestStalkerPacketEffects(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, tc := range []struct {
		skill uint16
		ids   []int
		owner uint32
	}{
		{389, []int{db.SkillEffectChaseWalkSmoke}, 2000000},
		{390, []int{392}, 2000000},
		{475, []int{496}, 2000000},
		{476, []int{495}, 300},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Amount: 65535}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must not play effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != len(tc.ids) || m.actorSystem.ActorAnims[n.SourceID].ActionFamily != sprites.ActionPCSkill {
			t.Fatalf("skill %d: wrong effects or release pose", tc.skill)
		}
		for i, e := range m.effects().Active {
			if e.EffectID != tc.ids[i] || e.ActorID != tc.owner {
				t.Fatalf("skill %d: wrong packet owner: %+v", tc.skill, e)
			}
		}
		if tc.skill == 476 {
			if len(m.scheduledSounds) != 4 {
				t.Fatal("Full Strip needs four fragment sounds")
			}
			for i, sound := range m.scheduledSounds {
				if sound.at.Sub(m.effects().Active[0].Starts) != time.Duration(1+10*i)*db.EffectFrameDuration || !sound.positioned || sound.paths[0] != "effect\\복호격.wav" {
					t.Fatal("wrong Full Strip sound timing")
				}
			}
		}
		m = &WorldMode{}
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: n.SourceID, TargetID: n.TargetID, Damage: 100, Action: network.ActorActionSkill}, time.Now(), time.Now())
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d: damage packet replayed a no-damage effect", tc.skill)
		}
	}
}

func TestStalkerCastingAndSpin(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, tc := range []struct {
		skill        uint16
		aura, action int
	}{{389, 501, scene.SpriteActionIdle}, {390, 16, sprites.ActionPCReadyFight}, {475, 59, scene.SpriteActionIdle}, {476, 16, sprites.ActionPCReadyFight}} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: tc.skill, SourceID: 2000000, DelayTime: 5000, Property: 3})
		if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != tc.aura || m.actorSystem.ActorAnims[2000000].ActionFamily != tc.action {
			t.Fatalf("skill %d: wrong casting aura/pose", tc.skill)
		}
		if tc.skill != 389 {
			continue
		}
		start := m.effects().Active[1].Starts
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\EF_BeginSpell.wav" {
			t.Fatal("Chase Walk needs its native casting sound")
		}
		for _, test := range []struct{ tick, direction int }{{0, 0}, {15, 4}, {16, 5}, {30, 2}, {40, 0}, {41, 1}, {100, 0}} {
			now := start.Add(time.Duration(test.tick) * db.EffectFrameDuration)
			if got := m.actors().SkillEffectDirection(ctx, 100, 0, now); got != test.direction {
				t.Fatalf("Chase Walk tick %d: direction %d, want %d", test.tick, got, test.direction)
			}
		}
		if ctx.World.Player.Dir != 0 {
			t.Fatal("casting spin changed authoritative facing")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 389, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if m.actors().SkillEffectDirection(ctx, 100, 0, start.Add(20*db.EffectFrameDuration)) != 0 {
			t.Fatal("casting spin survived skill release")
		}
		delete(m.actorSystem.ActorAnims, 2000000)
		if m.actors().SkillEffectDirection(ctx, 100, 0, start.Add(20*db.EffectFrameDuration)) != 0 {
			t.Fatal("casting spin survived interrupted animation")
		}
	}
}

func TestStarGladiatorNoDamageOwnershipAndSounds(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		ids    []int
		actor  uint32
		sounds int
	}{
		{427, nil, 0, 0},
		{431, []int{488, 543}, 2000000, 1},
		{432, []int{488, 543}, 2000000, 1},
		{433, []int{488, 543}, 2000000, 1},
		{434, []int{543}, 300, 1},
		{444, []int{effects.Quake}, 2000000, 1},
	} {
		ctx := testutil.ArcherEffectContext()
		ctx.World.Player.Job = db.JobStar
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != len(tc.ids) || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", tc.skill, m.effects().Active, m.scheduledSounds)
		}
		for i, e := range m.effects().Active {
			if e.EffectID != tc.ids[i] || e.ActorID != tc.actor {
				t.Fatalf("skill %d effect ownership: %+v", tc.skill, e)
			}
		}
		if tc.skill >= 431 && tc.skill <= 433 {
			anim := m.actorSystem.ActorAnims[100]
			if anim.FixedMotion != 2 || !anim.HasFixedMotion || anim.Duration != time.Second {
				t.Fatal("Comfort must hold skill motion two for one second")
			}
			if m.scheduledSounds[0].at.Sub(m.effects().Active[0].Starts) != 5*db.EffectFrameDuration {
				t.Fatal("Comfort's sound belongs to effect tick five")
			}
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("damage notifications must not replay Star Gladiator's no-damage effects")
		}
	}
}

func TestStarGladiatorCastsAndRejectedSkills(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job = db.JobStar
	for _, skill := range []uint16{427, 434, 444} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
		found := false
		for _, e := range m.effects().Active {
			found = found || e.EffectID == effects.BlueCasting
		}
		if !found {
			t.Fatalf("skill %d needs BlueCasting", skill)
		}
	}
	for skill := uint16(427); skill <= 444; skill++ {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300})
		if len(m.effects().Active)+len(m.actorSystem.ActorAnims)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill created release effects")
		}
	}
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 428, SourceID: 2000000, TargetID: 999, Result: 1})
	if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
		t.Fatal("Warmth requires both actors, like the native packet handler")
	}
	m.effects().Add(ctx, effects.Hated, 300)
	if len(m.scheduledSounds) != 0 {
		t.Fatal("the shared Hated effect must not play Kaahi's sound")
	}
}

func TestMagnumBreakOncePerCast(t *testing.T) {
	for _, targets := range []int{0, 1, 5} {
		ctx := testutil.SwordmanContext()
		m := &WorldMode{}
		now := time.Now()
		for i := 0; i < targets; i++ {
			action := network.ActorActionNotify{SkillID: db.SkillSMMagnum, SourceID: 2000000, TargetID: uint32(300 + i), Damage: int32(i * 50)}
			m.addSkillBeginEffect(ctx, action, now)
			m.addSkillEffect(ctx, action, now, now)
			m.addSkillHitEffect(ctx, action, now)
		}
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatal("Magnum Break victim notifications replayed the cast effect")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSMMagnum, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 1 {
			t.Fatalf("%d victims: effects=%+v sounds=%+v", targets, m.effects().Active, m.scheduledSounds)
		}
		burst := m.effects().Active[0]
		if burst.EffectID != effects.MagnumBreak || burst.ActorID != 2000000 || burst.ActorOrigin.X != 10.5 || !burst.HasActorTransform {
			t.Fatalf("burst must capture the caster's position: %+v", burst)
		}
		if m.effects().Active[1].EffectID != effects.QuakeMagnum || m.cameraShakeEnd.Sub(m.cameraShakeStart) != 50*time.Millisecond {
			t.Fatal("missing native Magnum Break camera shake")
		}
	}
}

func TestSwordmanSuccessEffects(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		actor  uint32
		delays []time.Duration
	}{
		{db.SkillSMProvoke, effects.Provoke, 300, []time.Duration{5 * db.EffectFrameDuration, 30 * db.EffectFrameDuration}},
		{db.SkillSMEndure, effects.Endure, 2000000, []time.Duration{0}},
	} {
		ctx := testutil.SwordmanContext()
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d emitted effects", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != tc.actor || m.effects().Active[0].EffectID != tc.effect {
			t.Fatalf("skill %d has wrong effect or owner: %+v", tc.skill, m.effects().Active)
		}
		if len(m.scheduledSounds) != len(tc.delays) {
			t.Fatalf("skill %d sounds: %+v", tc.skill, m.scheduledSounds)
		}
		for i, delay := range tc.delays {
			if !m.scheduledSounds[i].at.Equal(m.effects().Active[0].Starts.Add(delay)) {
				t.Fatalf("skill %d sound %d has wrong delay", tc.skill, i)
			}
		}
	}
	// Auto Berserk only changes server status; it must not play Provoke on cast.
	m := &WorldMode{}
	m.applySkillNoDamageNotify(testutil.SwordmanContext(), network.SkillNoDamageNotify{SkillID: db.SkillSMAutoberserk, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Auto Berserk invented a cast effect")
	}
}

func TestTaekwonHighJumpCastAndLanding(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobTaekwon, db.JobTaekwon
	for _, delay := range []uint32{0, 1000} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 426, SourceID: 2000000, DelayTime: delay})
		count := 2
		if delay > 0 {
			count++ // The casting circle still accompanies non-instant casts.
		}
		if len(m.effects().Active) != count || m.effects().Active[0].EffectID != effects.JumpBody || m.effects().Active[1].EffectID != effects.Peong {
			t.Fatalf("High Jump cast must launch takeoff and dust, even without a delay: %+v", m.effects().Active)
		}
		start := m.actorSystem.ActorAnims[2000000].Started
		for _, tc := range []struct{ tick, motion int }{{0, 0}, {29, 0}, {30, 1}, {40, 1}} {
			anim, ok := m.actors().ActorAnimation(2000000, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
			if !ok || anim.ActionFamily != sprites.ActionPCAttack1 || anim.FixedMotion != tc.motion {
				t.Fatalf("High Jump casting pose tick %d: %+v", tc.tick, anim)
			}
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 426, SourceID: 100, TargetID: 100, Result: 1})
		if len(m.scheduledSounds) != 0 {
			t.Fatal("landing before takeoff must cancel the pending takeoff sound")
		}
		for _, e := range m.effects().Active {
			if e.EffectID == effects.JumpBody {
				t.Fatal("landing did not remove takeoff through the local actor alias")
			}
		}
		start = m.actorSystem.ActorAnims[100].Started
		anim, ok := m.actors().ActorAnimation(100, start.Add(40*db.EffectFrameDuration))
		if !ok || anim.ActionFamily != scene.SpriteActionWalk || anim.FixedMotion != 5 {
			t.Fatalf("High Jump landing must finish on walk motion five: %+v", anim)
		}
		if ctx.World.Player.X != 10 || ctx.World.Player.Y != 20 {
			t.Fatal("High Jump visuals moved the authoritative actor")
		}
	}
}

func TestTaekwonSecondaryKickDoesNotRepeatRelease(t *testing.T) {
	for _, skill := range []uint16{415, 417, 419, 421} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		before := time.Now()
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: 5, SourceSpeed: 400, Damage: 10, HitCount: 1})
		if len(m.effects().Active) != 0 {
			t.Fatal("secondary kick replayed release effects")
		}
		if _, ok := m.actorSystem.ActorAnims[100]; ok {
			t.Fatal("secondary kick restarted the caster pose")
		}
		for _, sound := range m.scheduledSounds {
			if sound.actorID != 300 || sound.at.Sub(before) > 450*time.Millisecond {
				t.Fatalf("secondary kick acquired the primary hit's extra delay: %+v", sound)
			}
		}
	}
}

func TestTaekwonKicksUseNativeMotionSequences(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, tc := range []struct {
		skill  uint16
		job    int16
		family int
		frames [5]int
		ticks  int
	}{
		{415, db.JobTaekwon, sprites.ActionPCAttack2, [5]int{3, 5, 7, 8, 2}, 5},
		{417, db.JobTaekwon, sprites.ActionPCAttack3, [5]int{4, 5, 6, 7, 2}, 5},
		{419, db.JobTaekwon, sprites.ActionPCAttack1, [5]int{2, 3, 4, 4, 4}, 4},
		{419, db.JobLinker, sprites.ActionPCAttack3, [5]int{4, 5, 6, 7, 2}, 4},
	} {
		ctx.World.Player.Job, ctx.Session.Selected.Job = tc.job, int16(tc.job)
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, SourceSpeed: 400, Damage: 10, HitCount: 1})
		start := m.actorSystem.ActorAnims[100].Started
		for i, frame := range tc.frames {
			anim, ok := m.actors().ActorAnimation(100, start.Add(time.Duration(i*tc.ticks)*db.EffectFrameDuration))
			if !ok || anim.ActionFamily != tc.family || !anim.HasFixedMotion || anim.FixedMotion != frame || anim.Play {
				t.Fatalf("skill %d job %d motion %d = %+v", tc.skill, tc.job, i, anim)
			}
		}
		if _, ok := m.actors().ActorAnimation(100, start.Add(500*time.Millisecond)); ok {
			t.Fatal("kick animation did not expire")
		}
	}
}

func TestMildWindReplacesCasterEffectsAndUsesLevel(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for level := uint16(0); level <= 8; level++ {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: 425, SourceID: 2000000, TargetID: 300, Amount: uint32(level), Result: 1}
		m.applySkillNoDamageNotify(ctx, notify)
		if level == 0 || level == 8 {
			if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
				t.Fatal("invalid Mild Wind level produced an effect")
			}
			continue
		}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != effects.StormKick3 || m.effects().Active[1].EffectID != effects.BeginAsura1+int(level)-1 {
			t.Fatalf("level %d effects: %+v", level, m.effects().Active)
		}
		for _, e := range m.effects().Active {
			if !scene.IsLocalActor(ctx, e.ActorID) {
				t.Fatal("Mild Wind belongs to the caster")
			}
		}
		if len(m.scheduledSounds) != 2 || m.scheduledSounds[0].positioned || m.scheduledSounds[0].paths[0] != "effect\\T_바람방출.wav" {
			t.Fatal("Mild Wind must play its centered sound once per use")
		}
	}
}

func TestThiefSkillPacketEffects(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	for _, tc := range []struct {
		skill  uint16
		effect int
		sound  string
	}{
		{db.SkillTFSteal, effects.Steal, "effect\\ef_steal.wav"},
		{db.SkillTFDetoxify, effects.Detoxication, "effect\\ef_detoxication.wav"},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d produced an effect", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != 300 {
			t.Fatalf("skill %d must affect its target once: %+v", tc.skill, m.effects().Active)
		}
		spec, _ := effects.Lookup(tc.effect)
		if len(m.scheduledSounds) != 1 || len(spec.Sfx) != 1 || spec.Sfx[0] != tc.sound {
			t.Fatalf("skill %d has wrong audio: %+v", tc.skill, m.scheduledSounds)
		}
	}
	starts := time.Unix(100, 0)
	hitAt := starts.Add(500 * time.Millisecond)
	for _, skill := range []uint16{db.SkillTFPoison, db.SkillTFSprinklesand, db.SkillTFThrowstone} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 50}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillBeforeHitEffect(ctx, action, starts)
		m.addSkillEffect(ctx, action, starts, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		want := []int{effects.BashBegin, effects.BashBegin, effects.Hit1, effects.BashHit}
		if skill == db.SkillTFSprinklesand {
			want = []int{effects.SprinkleSand, effects.Hit1}
		} else if skill == db.SkillTFThrowstone {
			want = []int{effects.ThrowItem3, effects.Hit1}
		}
		if len(m.effects().Active) != len(want) {
			t.Fatalf("skill %d effects: %+v, want %v", skill, m.effects().Active, want)
		}
		for i, effect := range m.effects().Active {
			at, actor := hitAt, uint32(300)
			if i == 0 || skill == db.SkillTFPoison && i == 1 {
				at = starts
				if skill != db.SkillTFSprinklesand {
					actor = 2000000
				}
			}
			if effect.EffectID != want[i] || effect.ActorID != actor || !effect.Starts.Equal(at) {
				t.Fatalf("skill %d effect %d has wrong ownership or timing: %+v", skill, i, effect)
			}
		}
		m.effects().Active = nil
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 0 {
			t.Fatalf("missed skill %d must not show an impact", skill)
		}
		action.Damage, action.Action = 50, 5
		m.applyActorActionNotify(ctx, action)
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d replayed effects for a splash notification", skill)
		}
	}
	m := &WorldMode{}
	m.addSkillHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillTFPoison, TargetID: 300, Damage: 50, HitCount: 2}, hitAt)
	if len(m.effects().Active) != 3 || m.effects().Active[0].EffectID != effects.Hit1 ||
		m.effects().Active[1].EffectID != effects.BashHit || m.effects().Active[2].EffectID != effects.Hit1 ||
		!m.effects().Active[2].Starts.Equal(hitAt.Add(scene.MultiHitDelay)) {
		t.Fatal("Envenom's secondary impact must play only on the first hit")
	}
}

func TestThiefEnvenomReducedEffects(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	ctx.Session.LessEffects = true
	m := &WorldMode{}
	action := network.ActorActionNotify{SkillID: db.SkillTFPoison, SourceID: 2000000, TargetID: 300, Damage: 50}
	start := time.Unix(100, 0)
	m.addSkillBeginEffect(ctx, action, start)
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("/mineffect must suppress Envenom's startup flares and their sounds")
	}
	m.addSkillHitEffect(ctx, action, start.Add(500*time.Millisecond))
	if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != effects.Hit1 || m.effects().Active[1].EffectID != effects.BashHit {
		t.Fatal("/mineffect must retain Envenom's impact effects")
	}
}

func TestThiefEnvenomAnimation(t *testing.T) {
	for _, damage := range []int32{0, 50} {
		ctx := testutil.ThiefEffectContext()
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SkillID: db.SkillTFPoison, SourceID: 2000000, TargetID: 300,
			Action: network.ActorActionSkill, Damage: damage, SourceSpeed: 500,
		})
		for _, id := range []uint32{2000000, 100} {
			if anim := m.actorSystem.ActorAnims[id]; anim.ActionFamily != sprites.ActionPCSkill || !anim.Play || anim.Duration != 500*time.Millisecond {
				t.Fatalf("Envenom must use the skill pose even on a miss: %+v", anim)
			}
		}
	}
}

func TestThiefFindStoneAnimation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		frames   int
		duration time.Duration
	}{
		{"unloaded sprite", 0, 300 * time.Millisecond},
		{"three frames", 3, 300 * time.Millisecond},
		{"four frames", 4, 400 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.ThiefEffectContext()
			m := &WorldMode{}
			if tc.frames > 0 {
				m.actorSystem.PlayerView = humanoidTimingView(scene.SpriteActionPickup, tc.frames)
				for i := range m.actorSystem.PlayerView.Body.Act.Actions {
					m.actorSystem.PlayerView.Body.Act.Actions[i].DelayMS = 100
				}
			}
			notify := network.SkillCastNotify{SkillID: db.SkillTFPickstone, SourceID: 2000000, TargetID: 2000000, DelayTime: 500}
			m.applySkillCastNotify(ctx, notify)
			for _, id := range []uint32{2000000, 100} {
				anim := m.actorSystem.ActorAnims[id]
				if anim.ActionFamily != scene.SpriteActionPickup || anim.Duration != tc.duration || anim.HasSpeed {
					t.Fatalf("Find Stone must play pickup once at the sprite's own speed: %+v", anim)
				}
				end := anim.Started.Add(tc.duration)
				if _, active := m.actors().ActorAnimation(id, end.Add(-time.Millisecond)); !active {
					t.Fatal("pickup ended before its last frame finished")
				}
				if _, active := m.actors().ActorAnimation(id, end); active {
					t.Fatal("pickup must end before the cast finishes")
				}
				bar := m.battleSystem.ActorCastBars[id]
				if _, active := battle.ActorCastBarProgress(bar, end); !active || bar.Duration != 500*time.Millisecond {
					t.Fatalf("cast bar must retain the server duration: %+v", bar)
				}
			}
			// A fast cast can finish before the pickup animation does.
			notify.DelayTime = 100
			m.applySkillCastNotify(ctx, notify)
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillTFPickstone, SourceID: 2000000, TargetID: 2000000, Result: 1})
			for _, id := range []uint32{2000000, 100} {
				if anim := m.actorSystem.ActorAnims[id]; anim.ActionFamily != scene.SpriteActionIdle {
					t.Fatalf("Find Stone must return to standing on success: %+v", anim)
				}
			}
		})
	}
}

func TestWhitesmithBuffsAndCasting(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobBlacksmithH, db.JobBlacksmithH
	for _, tc := range []struct {
		skill                  uint16
		effect, action, sounds int
	}{
		{db.SkillWSMeltdown, effects.Meltdown, scene.SpriteActionIdle, 1},
		{db.SkillWSCartboost, effects.CartBoost, scene.SpriteActionIdle, 1},
		{db.SkillWSOverthrustmax, effects.Overthrust, sprites.ActionPCSkill, 2},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed buff must not play effects or sounds")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != n.SourceID || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d must play once on its caster", tc.skill)
		}
		if m.actorSystem.ActorAnims[n.SourceID].ActionFamily != tc.action {
			t.Fatalf("skill %d has the wrong release pose", tc.skill)
		}
		if tc.skill == db.SkillWSOverthrustmax {
			m.applySkillNoDamageNotify(ctx, n)
			if len(m.effects().Active) != 1 {
				t.Fatal("Maximum Power Thrust must replace its previous ring")
			}
		}
	}
	for _, skill := range []uint16{db.SkillWSMeltdown, db.SkillWSCartboost, db.SkillWSOverthrustmax, db.SkillWSCarttermination, db.SkillWSWeaponrefine} {
		m := &WorldMode{}
		m.effects().AddSkillCastEffects(ctx, skill, 3, 2000000, 300, 0, 0, time.Second, time.Now(), "test")
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != effects.BashBegin {
			t.Fatalf("skill %d must use the physical casting flare", skill)
		}
	}
}

func TestCartTerminationDamageTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobBlacksmithH, db.JobBlacksmithH
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillWSCarttermination, Action: network.ActorActionSkill, Damage: damage, SourceSpeed: 1000}
		m.applyActorActionNotify(ctx, a)
		anim := m.actorSystem.ActorAnims[a.SourceID]
		start := anim.Started
		if anim.ActionFamily != sprites.ActionPCSkill || start.IsZero() {
			t.Fatal("Cart Termination must use the skill release pose")
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.CartTermination || m.effects().Active[0].ActorID != 300 || !m.effects().Active[0].Starts.Equal(start) {
			t.Fatal("Cart Termination must start on its victim at release, including misses, without an extra hit spark")
		}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 1 {
			t.Fatal("secondary splash notification replayed Cart Termination")
		}
		bursts := 0
		for _, sound := range m.scheduledSounds {
			if sound.paths[0] == "effect\\wizard_fire_pillar_b.wav" {
				bursts++
				if !sound.at.Equal(start.Add(30 * db.EffectFrameDuration)) {
					t.Fatal("Cart Termination impact audio must coincide with its burst at tick 30")
				}
			}
		}
		if bursts != 1 {
			t.Fatal("primary and splash packets must produce just one burst sound")
		}
	}
}

func TestWizardPacketEffects(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill              uint16
		release, secondary int
	}{
		{db.SkillWZJupitel, effects.JupitelThunder, effects.JupitelHit},
		{db.SkillWZWaterball, effects.WaterBall2, -1},
		{db.SkillWZFrostnova, -1, effects.FrostDiverHit},
		{db.SkillWZMeteor, -1, effects.FireHit},
		{db.SkillWZVermilion, -1, effects.WindHit},
		{db.SkillWZStormgust, -1, effects.StormGust},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 3, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.effects().Active)
			m.addSkillHitEffect(ctx, action, hit)
			if splash {
				if len(m.effects().Active) != 0 {
					t.Fatalf("skill %d replayed the spell on a splash victim", tc.skill)
				}
				continue
			}
			if tc.release < 0 {
				if before != 0 {
					t.Fatalf("skill %d launched an unexpected projectile", tc.skill)
				}
			} else if before != 1 || m.effects().Active[0].EffectID != tc.release || !m.effects().Active[0].Starts.Equal(start) ||
				m.effects().Active[0].ActorID != action.SourceID || m.effects().Active[0].TargetID != action.TargetID {
				t.Fatalf("skill %d must launch one projectile from caster to target at release", tc.skill)
			}
			want := 3
			if tc.secondary >= 0 {
				want++
			}
			if len(m.effects().Active)-before != want {
				t.Fatalf("skill %d repeated or lost an impact", tc.skill)
			}
			secondary := 0
			for _, effect := range m.effects().Active[before:] {
				if effect.ActorID != 300 || effect.Starts.Before(hit) {
					t.Fatalf("skill %d impact has wrong target or timing", tc.skill)
				}
				if effect.EffectID == tc.secondary {
					secondary++
				}
			}
			if tc.secondary >= 0 && secondary != 1 {
				t.Fatalf("skill %d must play its secondary impact once", tc.skill)
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
	}{
		{db.SkillWZSightrasher, effects.SightTrasher}, {db.SkillWZFrostnova, effects.FrostDiverHit},
	} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != 2000000 {
			t.Fatalf("skill %d must work without enemies", tc.skill)
		}
	}
	for skill := db.SkillWZFirepillar; skill <= db.SkillWZQuagmire; skill++ {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: 3})
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != effects.BeginSpell3 {
			t.Fatalf("skill %d must use the fire-property cast aura", skill)
		}
	}
}

func TestWarpPortalActorEntryAddsWarpZoneEffect(t *testing.T) {
	for _, job := range []int16{scene.ActorJobWarpPortal, scene.ActorJobWarpPortalActive} {
		t.Run(fmt.Sprintf("job_%d", job), func(t *testing.T) {
			world := worldstate.New()
			sessionState := &session.Session{AccountID: 2000000}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world}
			entry := network.ActorEntry{ID: 900, Job: job, X: 30, Y: 40}

			actors.UpsertNetworkActor(ctx, entry)
			mode.actors().ApplyWarpPortalEntry(ctx, entry)
			mode.actors().ApplyWarpPortalEntry(ctx, entry)

			if len(mode.effects().Active) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
			}
			if effect := mode.effects().Active[0]; effect.ActorID != 900 || effect.EffectID != effects.WarpZone2 || effect.X != 30 || effect.Y != 40 || effect.Duration != actors.WarpPortalActorEffectLifetime {
				t.Fatalf("effect = %+v", effect)
			}
		})
	}
}

func TestWaitingWarpPortalActorEntryAddsReadyPortalEffect(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: scene.ActorJobWarpPortalWaiting, X: 30, Y: 40}

	actors.UpsertNetworkActor(ctx, entry)
	mode.actors().ApplyWarpPortalEntry(ctx, entry)

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 900 || effect.EffectID != effects.ReadyPortal || effect.X != 30 || effect.Y != 40 || effect.Duration != actors.WarpPortalActorEffectLifetime {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestWarpPortalActorEntryReplacesWaitingEffect(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: scene.ActorJobWarpPortalWaiting, X: 30, Y: 40}

	actors.UpsertNetworkActor(ctx, entry)
	mode.actors().ApplyWarpPortalEntry(ctx, entry)
	entry.Job = scene.ActorJobWarpPortalActive
	actors.UpsertNetworkActor(ctx, entry)
	mode.actors().ApplyWarpPortalEntry(ctx, entry)

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 900 || effect.EffectID != effects.WarpZone2 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestRemoveActorNowRemovesActorWorldEffects(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}
	entry := network.ActorEntry{ID: 900, Job: scene.ActorJobWarpPortal, X: 30, Y: 40}

	actors.UpsertNetworkActor(ctx, entry)
	mode.actors().ApplyWarpPortalEntry(ctx, entry)
	mode.removeActorNow(ctx, entry.ID)

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after remove = %+v, want none", mode.effects().Active)
	}
}

func TestApplyActorActionNotifyUpdatesLocalSitState(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Moving: true}
	mode := &WorldMode{pendingAttack: attackIntent{targetID: 300}, lockedAttackID: 300, attackFocusID: 300, actorSystem: actors.System{ActorAnims: map[uint32]scene.ActorAnimation{
		2000000: {ActionFamily: sprites.ActionPCReadyFight, Loop: true},
		150000:  {ActionFamily: sprites.ActionPCReadyFight, Loop: true},
	}}}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		Action:   network.ActionSitDown,
	})
	if !world.Player.Sitting {
		t.Fatal("local player did not sit")
	}
	if world.Player.Moving {
		t.Fatal("local player kept moving while sitting")
	}
	if _, ok := mode.actorSystem.ActorAnims[2000000]; ok {
		t.Fatal("local account animation survived sitting")
	}
	if _, ok := mode.actorSystem.ActorAnims[150000]; ok {
		t.Fatal("local character animation survived sitting")
	}
	if mode.pendingAttack.targetID != 0 || mode.lockedAttackID != 0 || mode.attackFocusID != 0 {
		t.Fatalf("local attack intent survived sitting: pending=%d locked=%d focus=%d", mode.pendingAttack.targetID, mode.lockedAttackID, mode.attackFocusID)
	}

	mode.actorSystem.ActorAnims[2000000] = scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}
	mode.actorSystem.ActorAnims[150000] = scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		Action:   network.ActionStandUp,
	})
	if world.Player.Sitting {
		t.Fatal("local player did not stand")
	}
	if _, ok := mode.actorSystem.ActorAnims[2000000]; ok {
		t.Fatal("local account animation survived standing")
	}
	if _, ok := mode.actorSystem.ActorAnims[150000]; ok {
		t.Fatal("local character animation survived standing")
	}
}

func TestApplyActorActionNotifyUpdatesRemoteSitState(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Moving: true})
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: map[uint32]scene.ActorAnimation{
		300: {ActionFamily: sprites.ActionPCReadyFight, Loop: true},
	}}}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300,
		Action:   network.ActionSitDown,
	})
	if actor := world.Actors[300]; !actor.Sitting || actor.Moving {
		t.Fatalf("remote actor sit state = sitting %t moving %t", actor.Sitting, actor.Moving)
	}
	if _, ok := mode.actorSystem.ActorAnims[300]; ok {
		t.Fatal("remote actor animation survived sitting")
	}

	mode.actorSystem.ActorAnims[300] = scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 300,
		Action:   network.ActionStandUp,
	})
	if actor := world.Actors[300]; actor.Sitting {
		t.Fatalf("remote actor stayed sitting: %+v", actor)
	}
	if _, ok := mode.actorSystem.ActorAnims[300]; ok {
		t.Fatal("remote actor animation survived standing")
	}
}

func TestPickupWaitsForServerAnimationAndDisappear(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1}
	world.UpsertItem(item)
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: make(map[uint32]scene.ActorAnimation)}}
	mode.actorSystem.ActorAnims[150000] = scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
		},
		Network: networkClient,
		World:   world,
	}

	if !mode.requestPickup(ctx, item, "test") {
		t.Fatal("pickup request was rejected")
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(item.ID, 20080910))

	if _, ok := world.Items[item.ID]; !ok {
		t.Fatal("pickup request removed item before server disappearance")
	}
	if anim := mode.actorSystem.ActorAnims[150000]; anim.ActionFamily != sprites.ActionPCReadyFight || len(mode.actorSystem.ActorAnims) != 1 {
		t.Fatalf("pickup request changed the player's animation before server confirmation: %+v", mode.actorSystem.ActorAnims)
	}
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: world.Player.ID,
		TargetID: item.ID,
		Action:   network.ActorActionPickupItem,
	})
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local pickup animation missing")
	}
	if anim.ActionFamily != scene.SpriteActionPickup {
		t.Fatalf("pickup action = %d, want %d", anim.ActionFamily, scene.SpriteActionPickup)
	}
	if anim.Next != nil {
		t.Fatalf("pickup ack animation next = %+v, want nil so it returns to idle", anim.Next)
	}
	if expired, ok := mode.actors().ActorAnimation(150000, anim.Started.Add(anim.Duration)); ok {
		t.Fatalf("pickup request expired animation = %+v, want idle fallback", expired)
	}
	if world.Dir != actors.DirectionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}

	mode.applyItemPickupAck(ctx, network.ItemPickupAck{ItemID: 909, Amount: 1})
	if _, ok := world.Items[item.ID]; !ok {
		t.Fatal("pickup acknowledgment removed item before server disappearance")
	}

	mode.applyFloorItemDisappear(ctx, network.FloorItemDisappear{ID: item.ID})
	if _, ok := world.Items[item.ID]; ok {
		t.Fatal("server disappearance did not remove picked item")
	}
}

func TestRejectedPickupDoesNotAnimate(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1}
	world.UpsertItem(item)
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: world.Player.ID},
		Network: networkClient,
		World:   world,
	}
	if !mode.requestPickup(ctx, item, "test") {
		t.Fatal("pickup request was not sent")
	}
	readBotTestPackets(t, serverConn, network.BuildItemPickupPacketForClientDate(item.ID, 20080910))
	mode.applyItemPickupAck(ctx, network.ItemPickupAck{ItemID: item.ItemID, Result: 1})
	if len(mode.actorSystem.ActorAnims) != 0 || len(mode.effects().Active) != 0 || len(mode.scheduledSounds) != 0 {
		t.Fatalf("rejected pickup played an animation, effect or sound: animations=%+v effects=%+v sounds=%+v", mode.actorSystem.ActorAnims, mode.effects().Active, mode.scheduledSounds)
	}
	if len(ctx.Session.Inventory.Items) != 0 || len(world.Items) != 1 {
		t.Fatal("rejected pickup changed the inventory or floor items")
	}
}

func TestApplyItemPickupAckReceiveItemReplacesStackAndReportsGain(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 512, Amount: 3, Identified: true}},
		},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState}

	item, gained, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{
		Index:      7,
		ItemID:     512,
		Amount:     5,
		Type:       0,
		Identified: true,
		Result:     itemPickupResultReceive,
	})

	if !ok {
		t.Fatal("receive item ack was not treated as an inventory add")
	}
	if gained != 2 {
		t.Fatalf("gained = %d, want 2", gained)
	}
	if item.Amount != 5 {
		t.Fatalf("display item amount = %d, want server total 5", item.Amount)
	}
	if got := sessionState.Inventory.Items[0].Amount; got != 5 {
		t.Fatalf("inventory amount = %d, want replaced server total 5", got)
	}
}

func TestApplyItemPickupAckFailureDoesNotAddItemOrClearPendingPickup(t *testing.T) {
	sessionState := &session.Session{}
	mode := &WorldMode{
		pendingPickup: pickupIntent{itemID: 9001},
	}
	ctx := client.Context{Session: sessionState}

	if _, _, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{Index: 7, ItemID: 512, Amount: 1, Result: 1}); ok {
		t.Fatal("failed pickup ack was treated as success")
	}
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory items = %+v, want none", sessionState.Inventory.Items)
	}
	if mode.pendingPickup.itemID != 9001 {
		t.Fatalf("pending pickup = %d, want 9001", mode.pendingPickup.itemID)
	}
}

func TestApplyItemPickupAckSuccessDoesNotClearPendingPickup(t *testing.T) {
	mode := &WorldMode{
		pendingPickup: pickupIntent{itemID: 9002},
	}
	ctx := client.Context{Session: &session.Session{}}

	if _, _, ok := mode.applyItemPickupAck(ctx, network.ItemPickupAck{
		Index:  7,
		ItemID: 512,
		Amount: 1,
		Result: itemPickupResultSuccess,
	}); !ok {
		t.Fatal("successful pickup acknowledgment was rejected")
	}
	if mode.pendingPickup.itemID != 9002 {
		t.Fatalf("pending pickup = %d, want 9002", mode.pendingPickup.itemID)
	}
}

func TestApplyActorPickupActionNotifyStartsPickupInsteadOfAttack(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertItem(worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20, Amount: 1})
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: make(map[uint32]scene.ActorAnimation)}}
	mode.actorSystem.ActorAnims[150000] = scene.ActorAnimation{ActionFamily: sprites.ActionPCReadyFight, Loop: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
		},
		World: world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000,
		TargetID: 9001,
		Action:   network.ActorActionPickupItem,
	})

	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local pickup animation missing")
	}
	if anim.ActionFamily != scene.SpriteActionPickup {
		t.Fatalf("pickup action = %d, want %d", anim.ActionFamily, scene.SpriteActionPickup)
	}
	if anim.Next != nil {
		t.Fatalf("pickup animation next = %+v, want nil so it returns to idle", anim.Next)
	}
	if expired, ok := mode.actors().ActorAnimation(150000, anim.Started.Add(anim.Duration)); ok {
		t.Fatalf("pickup expired animation = %+v, want idle fallback", expired)
	}
	if len(mode.battleSystem.DamageFloaters) != 0 {
		t.Fatalf("pickup notify should not create damage floaters: %+v", mode.battleSystem.DamageFloaters)
	}
	if world.Dir != actors.DirectionFromDelta(10, 20, 11, 20, 4) {
		t.Fatalf("player dir = %d", world.Dir)
	}
}

func TestCombatDamageDoesNotInventMonsterLife(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		Job:           1008,
		X:             11,
		Y:             20,
		HasObjectType: true,
		ObjectType:    scene.ActorObjectTypeMob,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if _, ok := mode.battleSystem.ActorLife[300]; ok {
		t.Fatal("life should not be created from combat damage")
	}
}

func TestCombatDamageDoesNotMutateExactMonsterLife(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		Job:           1002,
		X:             11,
		Y:             20,
		HasObjectType: true,
		ObjectType:    scene.ActorObjectTypeMob,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.battle().ApplyActorHPUpdate(network.ActorHPUpdate{ID: 300, HP: 50, MaxHP: 100})
	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      12,
		Action:      0,
	})

	life, ok := mode.battleSystem.ActorLife[300]
	if !ok {
		t.Fatal("life missing")
	}
	if life.HP != 50 || life.MaxHP != 100 {
		t.Fatalf("exact life = %+v, want unchanged 50/100", life)
	}
}

func TestCombatHitDelayUsesActionSoundMotion(t *testing.T) {
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: -1},
		{Sound: 0},
		{Sound: -1},
	}}
	if got := combatMotionDelayFromAction(action, 800*time.Millisecond, firstActionSoundMotion(action)); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want 400ms", got)
	}
}

func TestCombatHitDelayFallsBackToMidpoint(t *testing.T) {
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: -1},
		{Sound: -1},
		{Sound: -1},
	}}
	if got := combatMotionDelayFromAction(action, 800*time.Millisecond, firstActionSoundMotion(action)); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want midpoint", got)
	}
}

func TestActorActionFrameDelayUsesPlayerWeaponActionFrames(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1, Weapon: 3, Dir: 0}
	actionFamily := sprites.AttackActionFamily(nil, world.Player)
	mode := &WorldMode{actorSystem: actors.System{PlayerView: humanoidTimingView(actionFamily, 4)}}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	if got := mode.actors().ActorActionFrameDelay(ctx, world.Player, actionFamily, 800*time.Millisecond); got != 200*time.Millisecond {
		t.Fatalf("frame delay = %s, want attack duration divided by weapon action frames", got)
	}
}

func TestActorActionNotifySetsPlayerWeaponAttackFrameDelay(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1, Weapon: 3, Dir: 0}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	actionFamily := sprites.AttackActionFamily(nil, world.Player)
	mode := &WorldMode{actorSystem: actors.System{PlayerView: humanoidTimingView(actionFamily, 4)}}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Weapon: 3}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("attack animation missing")
	}
	if !anim.HasSpeed || anim.Speed != 200*time.Millisecond {
		t.Fatalf("attack animation = %+v, want frame delay from packet speed/action frame count", anim)
	}
}

func humanoidTimingView(actionFamily int, frames int) *sprites.Humanoid {
	actions := make([]res.ACTAction, actionFamily*8+8)
	for dir := 0; dir < 8; dir++ {
		actions[actionFamily*8+dir] = res.ACTAction{DelayMS: 150, Animations: make([]res.ACTAnimation, frames)}
	}
	return &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions}}}
}

func humanoidSoundView(actionFamily int, frames int, soundMotion int, sounds []string) *sprites.Humanoid {
	actions := make([]res.ACTAction, actionFamily*8+8)
	for dir := 0; dir < 8; dir++ {
		animations := make([]res.ACTAnimation, frames)
		for i := range animations {
			animations[i].Sound = -1
		}
		if soundMotion >= 0 && soundMotion < len(animations) {
			animations[soundMotion].Sound = 0
		}
		actions[actionFamily*8+dir] = res.ACTAction{DelayMS: 150, Animations: animations}
	}
	return &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions, Sounds: sounds}}}
}

func TestActionSoundNameResolvesACTSound(t *testing.T) {
	act := &res.ACT{Sounds: []string{"attack.wav"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{{Sound: -1}, {Sound: 0}}}
	if got := actionSoundName(act, action, 1); got != "attack.wav" {
		t.Fatalf("sound = %q, want attack.wav", got)
	}
}

func TestActionSoundNameIgnoresAttackMarker(t *testing.T) {
	act := &res.ACT{Sounds: []string{"atk"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{{Sound: 0}}}
	if got := actionSoundName(act, action, 0); got != "" {
		t.Fatalf("sound = %q, want empty marker", got)
	}
}

func TestActionAttackMarkerMotionPrefersATKOverEarlierSound(t *testing.T) {
	act := &res.ACT{Sounds: []string{"clothes.wav", "atk"}}
	action := res.ACTAction{Animations: []res.ACTAnimation{
		{Sound: -1},
		{Sound: 0},
		{Sound: -1},
		{Sound: 1},
	}}
	if got := actionAttackMarkerMotion(act, action); got != 3 {
		t.Fatalf("attack marker motion = %d, want ATK motion 3", got)
	}
}

func TestSkillHitSoundUsesReferenceEnemyNormalSounds(t *testing.T) {
	source := worldstate.Actor{Job: db.JobWizard, Weapon: 1601}
	target := worldstate.Actor{Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}

	got := combatHitSFXCandidates(network.ActorActionNotify{SkillID: db.SkillWZEarthspike}, source, true, target, true)
	requireSameStringSet(t, got, db.EnemyHitNormalSounds())
	if got[0] == "_hit_rod.wav" || got[0] == "_hit_arrow.wav" {
		t.Fatalf("skill hit sound = %q, want enemy hit normal sound", got[0])
	}
}

func TestEarthSpikeSchedulesEnemyHitAndSpikeSounds(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: db.JobWizard, Weapon: 1601, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    2000000,
		TargetID:    300,
		SkillID:     db.SkillWZEarthspike,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      84,
		Action:      network.ActorActionSkill,
	})

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want enemy hit and earth spike sounds", mode.scheduledSounds)
	}
	if got := mode.scheduledSounds[0].paths; !sameStringSet(got, db.EnemyHitNormalSounds()) {
		t.Fatalf("enemy hit sound candidates = %v, want rotated %v", got, db.EnemyHitNormalSounds())
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("enemy hit sound source = %+v, want target actor", sound)
	}
	if got := mode.scheduledSounds[1].paths; len(got) != 1 || got[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("earth spike sound = %+v", mode.scheduledSounds[1])
	}
}

func requireSameStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if !sameStringSet(got, want) {
		t.Fatalf("strings = %v, want same set as %v", got, want)
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(want))
	for _, value := range want {
		seen[value]++
	}
	for _, value := range got {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	return true
}

func TestMercenaryAttackSchedulesWeaponSwingAndHitSounds(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 8, Y: 20, Dir: 4}
	mercenary := worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		Job:           6037,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	}
	world.UpsertActor(mercenary)
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	actionFamily := sprites.AttackActionFamily(nil, mercenary)
	mode := &WorldMode{actorSystem: actors.System{MercenaryViews: map[sprites.Appearance]*sprites.Humanoid{
		sprites.MercenaryAppearanceForActor(mercenary): humanoidSoundView(actionFamily, 2, 1, []string{"atk"}),
	}}}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      42,
		Action:      0,
	})

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want swing and hit sounds", mode.scheduledSounds)
	}
	if got := mode.scheduledSounds[0].paths; len(got) != 1 || got[0] != "attack_sword.wav" {
		t.Fatalf("swing sound = %+v", mode.scheduledSounds[0])
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 400 {
		t.Fatalf("swing sound source = %+v, want source actor", sound)
	}
	if got := mode.scheduledSounds[1].paths; len(got) != 1 || got[0] != "_hit_sword.wav" {
		t.Fatalf("hit sound = %+v", mode.scheduledSounds[1])
	}
	if sound := mode.scheduledSounds[1]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("hit sound source = %+v, want target actor", sound)
	}
}

func TestPlayerBowNormalAttackUsesReferenceReleaseAndImpactTimeline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		job    int16
		damage int32
	}{
		{name: "archer_hit", job: db.JobArcher, damage: 42},
		{name: "archer_miss", job: db.JobArcher},
		{name: "hunter_hit", job: db.JobHunter, damage: 42},
		{name: "hunter_miss", job: db.JobHunter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				playerID = uint32(200)
				targetID = uint32(300)
			)
			player := worldstate.Actor{ID: playerID, X: 10, Y: 20, Job: tc.job, Weapon: db.WeaponBow, Dir: 4}
			world := worldstate.New()
			world.Player = player
			world.UpsertActor(worldstate.Actor{
				ID:            targetID,
				X:             15,
				Y:             20,
				Job:           1002,
				ObjectType:    scene.ActorObjectTypeMob,
				HasObjectType: true,
			})
			actionFamily := sprites.AttackActionFamily(nil, player)
			mode := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(actionFamily, 4, 2, []string{"atk"})}}
			ctx := client.Context{
				Session: &session.Session{
					AccountID: playerID,
					CharID:    playerID,
					Selected:  session.Character{ID: playerID, Job: tc.job, Weapon: db.WeaponBow},
				},
				World: world,
			}

			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID:    playerID,
				TargetID:    targetID,
				SourceSpeed: 800,
				TargetSpeed: 480,
				Damage:      tc.damage,
				Action:      0,
			})

			sourceAnim, ok := mode.actorSystem.ActorAnims[playerID]
			if !ok {
				t.Fatal("bow attack animation missing")
			}
			if len(mode.effects().Active) == 0 {
				t.Fatal("arrow projectile missing")
			}
			projectile := mode.effects().Active[0]
			if projectile.EffectID != effects.ArrowShot || projectile.ActorID != targetID || projectile.TargetID != playerID {
				t.Fatalf("arrow projectile = %+v", projectile)
			}
			if delay := projectile.Starts.Sub(sourceAnim.Started); delay != 400*time.Millisecond {
				t.Fatalf("arrow release delay = %s, want ACT marker at 400ms", delay)
			}
			if projectile.Duration != scene.ReferenceBowArrowFlightDuration || projectile.Expires.Sub(projectile.Starts) != scene.ReferenceBowArrowFlightDuration {
				t.Fatalf("arrow flight = duration %s lifetime %s, want %s", projectile.Duration, projectile.Expires.Sub(projectile.Starts), scene.ReferenceBowArrowFlightDuration)
			}

			if len(mode.scheduledSounds) == 0 {
				t.Fatal("bow release sound missing")
			}
			releaseSound := mode.scheduledSounds[0]
			if !releaseSound.at.Equal(projectile.Starts) || releaseSound.actorID != playerID || !sameStringSet(releaseSound.paths, db.WeaponAttackSounds(db.WeaponBow)) {
				t.Fatalf("bow release sound = %+v projectile=%+v", releaseSound, projectile)
			}

			if tc.damage > 0 {
				if len(mode.effects().Active) != 2 {
					t.Fatalf("world effects = %+v, want arrow and impact", mode.effects().Active)
				}
				impact := mode.effects().Active[1]
				if impact.EffectID != effects.Hit1 || impact.ActorID != targetID || !impact.Starts.Equal(projectile.Expires) {
					t.Fatalf("impact = %+v projectile=%+v", impact, projectile)
				}
				targetAnim, ok := mode.actorSystem.ActorAnims[targetID]
				if !ok || !targetAnim.Started.Equal(impact.Starts) {
					t.Fatalf("target hurt animation = %+v ok=%t impact=%+v", targetAnim, ok, impact)
				}
				if len(mode.scheduledSounds) != 2 {
					t.Fatalf("scheduled sounds = %+v, want release and impact", mode.scheduledSounds)
				}
				impactSound := mode.scheduledSounds[1]
				if !impactSound.at.Equal(impact.Starts) || impactSound.actorID != targetID || len(impactSound.paths) != 1 || impactSound.paths[0] != "_hit_arrow.wav" {
					t.Fatalf("arrow impact sound = %+v impact=%+v", impactSound, impact)
				}
				if len(mode.battleSystem.DamageFloaters) != 1 || !mode.battleSystem.DamageFloaters[0].Starts.Equal(impact.Starts) {
					t.Fatalf("damage floater = %+v impact=%+v", mode.battleSystem.DamageFloaters, impact)
				}
				return
			}

			if len(mode.effects().Active) != 1 {
				t.Fatalf("miss effects = %+v, want arrow only", mode.effects().Active)
			}
			if len(mode.scheduledSounds) != 1 {
				t.Fatalf("miss sounds = %+v, want bow release only", mode.scheduledSounds)
			}
			if _, ok := mode.actorSystem.ActorAnims[targetID]; ok {
				t.Fatalf("miss started target hurt animation: %+v", mode.actorSystem.ActorAnims[targetID])
			}
			if len(mode.battleSystem.DamageFloaters) != 1 || mode.battleSystem.DamageFloaters[0].Kind != battle.FloaterMiss || !mode.battleSystem.DamageFloaters[0].Starts.Equal(projectile.Expires) {
				t.Fatalf("miss floater = %+v projectile=%+v", mode.battleSystem.DamageFloaters, projectile)
			}
		})
	}
}

func TestDoubleStrafeUsesReferenceBowReleaseAndMultiHitTimeline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage int32
	}{
		{name: "hit", damage: 100},
		{name: "miss"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				playerID = uint32(200)
				targetID = uint32(300)
			)
			player := worldstate.Actor{ID: playerID, X: 10, Y: 20, Job: db.JobHunter, Weapon: db.WeaponBow, Dir: 4}
			world := worldstate.New()
			world.Player = player
			world.UpsertActor(worldstate.Actor{
				ID:            targetID,
				X:             15,
				Y:             20,
				Job:           1002,
				ObjectType:    scene.ActorObjectTypeMob,
				HasObjectType: true,
			})
			actionFamily := effects.SkillAction(db.SkillACDouble).ActionFamilyForActorWithResources(nil, player)
			mode := &WorldMode{actorSystem: actors.System{PlayerView: humanoidSoundView(actionFamily, 4, 2, []string{"atk"})}}
			ctx := client.Context{
				Session: &session.Session{
					AccountID: playerID,
					CharID:    playerID,
					Selected:  session.Character{ID: playerID, Job: db.JobHunter, Weapon: db.WeaponBow},
				},
				World: world,
			}

			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID:    playerID,
				TargetID:    targetID,
				SourceSpeed: 800,
				TargetSpeed: 480,
				Damage:      tc.damage,
				HitCount:    2,
				Action:      8,
				SkillID:     db.SkillACDouble,
				SkillLevel:  10,
			})

			sourceAnim, ok := mode.actorSystem.ActorAnims[playerID]
			if !ok {
				t.Fatal("Double Strafe animation missing")
			}
			if len(mode.effects().Active) < 2 || mode.effects().Active[0].EffectID != effects.BashBegin {
				t.Fatalf("Double Strafe effects = %+v, want begin effect first", mode.effects().Active)
			}
			projectile := mode.effects().Active[1]
			if projectile.EffectID != effects.ArrowShot || projectile.ActorID != targetID || projectile.TargetID != playerID {
				t.Fatalf("Double Strafe projectile = %+v", projectile)
			}
			if delay := projectile.Starts.Sub(sourceAnim.Started); delay != 400*time.Millisecond {
				t.Fatalf("Double Strafe release delay = %s, want ACT marker at 400ms", delay)
			}
			if projectile.Duration != scene.ReferenceBowArrowFlightDuration || projectile.Expires.Sub(projectile.Starts) != scene.ReferenceBowArrowFlightDuration {
				t.Fatalf("Double Strafe flight = duration %s lifetime %s, want %s", projectile.Duration, projectile.Expires.Sub(projectile.Starts), scene.ReferenceBowArrowFlightDuration)
			}

			var releaseSounds, impactSounds []scheduledSound
			for _, sound := range mode.scheduledSounds {
				switch {
				case sameStringSet(sound.paths, db.WeaponAttackSounds(db.WeaponBow)):
					releaseSounds = append(releaseSounds, sound)
				case sameStringSet(sound.paths, db.EnemyHitNormalSounds()):
					impactSounds = append(impactSounds, sound)
				}
			}
			if len(releaseSounds) != 1 || !releaseSounds[0].at.Equal(projectile.Starts) || releaseSounds[0].actorID != playerID {
				t.Fatalf("Double Strafe release sounds = %+v projectile=%+v", releaseSounds, projectile)
			}

			if tc.damage > 0 {
				if len(mode.effects().Active) != 5 {
					t.Fatalf("Double Strafe effects = %+v, want begin, one arrow, and three impact effects", mode.effects().Active)
				}
				for i, impact := range mode.effects().Active[2:] {
					wantStarts, wantID := projectile.Expires, effects.Hit1
					if i == 1 {
						wantID = effects.BashHit
					} else if i == 2 {
						wantStarts = wantStarts.Add(scene.MultiHitDelay)
					}
					if impact.EffectID != wantID || impact.ActorID != targetID || !impact.Starts.Equal(wantStarts) {
						t.Fatalf("Double Strafe impact %d = %+v, want effect %d at %s", i, impact, wantID, wantStarts)
					}
				}
				targetAnim, ok := mode.actorSystem.ActorAnims[targetID]
				if !ok || !targetAnim.Started.Equal(projectile.Expires) {
					t.Fatalf("Double Strafe target animation = %+v ok=%t, want first impact at %s", targetAnim, ok, projectile.Expires)
				}
				if len(impactSounds) != 2 {
					t.Fatalf("Double Strafe impact sounds = %+v, want two", impactSounds)
				}
				for i, sound := range impactSounds {
					wantStarts := projectile.Expires.Add(scene.MultiHitDelay * time.Duration(i))
					if !sound.at.Equal(wantStarts) || sound.actorID != targetID {
						t.Fatalf("Double Strafe impact sound %d = %+v, want target=%d at=%s", i, sound, targetID, wantStarts)
					}
				}
				normalFloaters := make([]battle.Floater, 0, 2)
				for _, floater := range mode.battleSystem.DamageFloaters {
					if floater.Kind == battle.FloaterNormal {
						normalFloaters = append(normalFloaters, floater)
					}
				}
				if len(normalFloaters) != 2 {
					t.Fatalf("Double Strafe damage floaters = %+v, want two normal hits", mode.battleSystem.DamageFloaters)
				}
				for i, floater := range normalFloaters {
					wantStarts := projectile.Expires.Add(scene.MultiHitDelay * time.Duration(i))
					if !floater.Starts.Equal(wantStarts) {
						t.Fatalf("Double Strafe damage floater %d = %+v, want start %s", i, floater, wantStarts)
					}
				}
				return
			}

			if len(mode.effects().Active) != 2 {
				t.Fatalf("Double Strafe miss effects = %+v, want begin and arrow only", mode.effects().Active)
			}
			if len(impactSounds) != 0 {
				t.Fatalf("Double Strafe miss impact sounds = %+v, want none", impactSounds)
			}
			if _, ok := mode.actorSystem.ActorAnims[targetID]; ok {
				t.Fatalf("Double Strafe miss started target hurt animation: %+v", mode.actorSystem.ActorAnims[targetID])
			}
			if len(mode.battleSystem.DamageFloaters) != 1 || mode.battleSystem.DamageFloaters[0].Kind != battle.FloaterMiss || !mode.battleSystem.DamageFloaters[0].Starts.Equal(projectile.Expires) {
				t.Fatalf("Double Strafe miss floater = %+v projectile=%+v", mode.battleSystem.DamageFloaters, projectile)
			}
		})
	}
}

func TestApplyActorActionNotifyUsesMobACTHitPhase(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 11, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             10,
		Y:             20,
		Dir:           4,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{actorSystem: actors.System{NonPCViews: map[int]*scene.SpriteView{
		1002: {
			Act: &res.ACT{
				Actions: []res.ACTAction{
					{},
					{},
					{Animations: []res.ACTAnimation{
						{Sound: -1},
						{Sound: -1},
						{Sound: 0},
						{Sound: -1},
					}},
				},
				Sounds: []string{"poring_attack.wav"},
			},
		},
	}}}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Sex:       0,
			Selected:  session.Character{ID: 150000, Job: 0, Hair: 1, Weapon: 1201},
		},
		World: world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    300,
		TargetID:    2000000,
		SourceSpeed: 800,
		TargetSpeed: 480,
		Damage:      1,
		Action:      0,
	})

	sourceAnim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("source animation missing")
	}
	targetAnim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("local target animation missing")
	}
	if got := targetAnim.Started.Sub(sourceAnim.Started); got != 400*time.Millisecond {
		t.Fatalf("hit delay = %s, want ACT sound phase", got)
	}
	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want attack and hit sounds", mode.scheduledSounds)
	}
	if !mode.scheduledSounds[0].at.Equal(targetAnim.Started) || mode.scheduledSounds[0].paths[0] != "poring_attack.wav" {
		t.Fatalf("attack sound = %+v targetStarted=%s", mode.scheduledSounds[0], targetAnim.Started)
	}
}

func TestApplyActorVanishDeathKeepsMobForDeathAnimation(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{actorSystem: actors.System{NonPCViews: map[int]*scene.SpriteView{
		1002: {
			Act: &res.ACT{
				Actions: []res.ACTAction{
					{},
					{},
					{},
					{},
					{Animations: []res.ACTAnimation{{Sound: 0}, {Sound: -1}}, DelayMS: 100},
				},
				Sounds: []string{"poring_die.wav"},
			},
		},
	}}}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000}},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})

	if _, ok := world.Actors[300]; !ok {
		t.Fatal("dead actor was removed immediately")
	}
	anim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("death animation missing")
	}
	if anim.ActionFamily != sprites.ActionNonPCDeath {
		t.Fatalf("death action = %d, want %d", anim.ActionFamily, sprites.ActionNonPCDeath)
	}
	if anim.HoldFinal {
		t.Fatal("non-player death animation should not hold forever")
	}
	if removeAt, ok := mode.actorSystem.ActorDeaths[300]; !ok || !removeAt.After(anim.Started) {
		t.Fatalf("death removal time = %s ok=%t", removeAt, ok)
	}
	if got := mode.actorSystem.ActorDeaths[300].Sub(anim.Started); got != scene.NonPCDeathFadeDuration {
		t.Fatalf("death visible duration = %s, want %s", got, scene.NonPCDeathFadeDuration)
	}
	mode.processNonPCMotionSound(ctx, world.Actors[300], anim.Started)
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != "poring_die.wav" {
		t.Fatalf("death sounds = %+v", mode.scheduledSounds)
	}

	mode.cleanupDeadActors(ctx, mode.actorSystem.ActorDeaths[300].Add(time.Millisecond))
	if _, ok := world.Actors[300]; ok {
		t.Fatal("dead actor was not removed after death hold")
	}
}

func TestApplyActorVanishDeathFreezesMovingMobAtRenderedPosition(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		FromX:         10,
		FromY:         20,
		ToX:           20,
		ToY:           20,
		Moving:        true,
		MoveStarted:   now.Add(-50 * time.Second),
		MoveDuration:  100 * time.Second,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000}},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})

	actor, ok := world.Actors[300]
	if !ok {
		t.Fatal("dead actor was removed immediately")
	}
	if actor.Moving {
		t.Fatal("dead actor should stop moving")
	}
	if actor.X != 15 || actor.Y != 20 {
		t.Fatalf("dead actor position = %d,%d, want rendered position 15,20 instead of destination 20,20", actor.X, actor.Y)
	}
	if actor.FromX != actor.X || actor.ToX != actor.X || actor.FromY != actor.Y || actor.ToY != actor.Y {
		t.Fatalf("dead actor movement endpoints = from %d,%d to %d,%d, want frozen at %d,%d", actor.FromX, actor.FromY, actor.ToX, actor.ToY, actor.X, actor.Y)
	}
}

func TestRemotePlayerDeathRemainsUntilResurrection(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1,
		Appearance:    true,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actors.ActorVanishDeath})

	anim, ok := mode.actorSystem.ActorAnims[300]
	if !ok || anim.ActionFamily != sprites.ActionPCDeath || !anim.HoldFinal {
		t.Fatalf("remote player death animation = %+v ok=%t", anim, ok)
	}
	if removeAt, ok := mode.actorSystem.ActorDeaths[300]; !ok || !removeAt.IsZero() {
		t.Fatalf("remote player death deadline = %s ok=%t, want persistent zero deadline", removeAt, ok)
	}
	if alpha := mode.actors().ActorDeathAlpha(300, anim.Started.Add(time.Hour)); alpha != 1 {
		t.Fatalf("remote player death alpha = %f, want 1", alpha)
	}
	mode.cleanupDeadActors(ctx, anim.Started.Add(time.Hour))
	if _, ok := world.Actors[300]; !ok {
		t.Fatal("remote player body was removed by death cleanup")
	}

	mode.applyActorResurrection(ctx, network.ActorResurrection{ID: 300})
	if _, ok := mode.actorSystem.ActorDeaths[300]; ok {
		t.Fatal("remote player remained marked dead after resurrection")
	}
	if _, ok := mode.actorSystem.ActorAnims[300]; ok {
		t.Fatal("remote player death animation remained after resurrection")
	}
	if actor := world.Actors[300]; !actor.HasAIMotion || actor.AIMotion != companion.MotionStand {
		t.Fatalf("resurrected player motion = %+v, want standing", actor)
	}
}

func TestApplyActorVanishLogoutAndTeleportAddTeleportEffect(t *testing.T) {
	for _, reason := range []uint8{actors.ActorVanishLogout, actors.ActorVanishTeleport} {
		t.Run(fmt.Sprintf("reason_%d", reason), func(t *testing.T) {
			now := time.Now()
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.Actors[300] = worldstate.Actor{
				ID:           300,
				X:            20,
				Y:            20,
				FromX:        10,
				FromY:        20,
				ToX:          20,
				ToY:          20,
				Moving:       true,
				MoveStarted:  now.Add(-50 * time.Second),
				MoveDuration: 100 * time.Second,
			}
			mode := &WorldMode{}
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}

			mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: reason})

			if _, ok := world.Actors[300]; ok {
				t.Fatal("vanished actor was not removed")
			}
			if len(mode.effects().Active) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
			}
			if effect := mode.effects().Active[0]; effect.ActorID != 0 || effect.EffectID != effects.Teleportation || effect.X != 15 || effect.Y != 20 {
				t.Fatalf("effect = %+v, want pinned teleportation at rendered position 15,20", effect)
			}
		})
	}
}

func TestApplyActorVanishOutOfSightDoesNotAddTeleportEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 11, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actors.ActorVanishOutOfSight})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %+v, want none", mode.effects().Active)
	}
}

func TestApplyActorVanishOutOfSightFadesBeforeRemovingNPC(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{
		ID:            300,
		X:             20,
		Y:             20,
		Job:           100,
		ObjectType:    scene.ActorObjectTypeNPC,
		HasObjectType: true,
	}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actors.ActorVanishOutOfSight})

	if _, ok := world.Actors[300]; !ok {
		t.Fatal("out-of-sight NPC was removed immediately")
	}
	fade, ok := mode.actorSystem.ActorVanishes[300]
	if !ok {
		t.Fatal("out-of-sight fade missing")
	}
	if got := fade.RemoveAt.Sub(fade.Started); got != actors.ActorVanishOutOfSightFadeDuration {
		t.Fatalf("fade duration = %s, want %s", got, actors.ActorVanishOutOfSightFadeDuration)
	}
	if got := mode.actors().ActorVanishAlpha(300, fade.Started.Add(actors.ActorVanishOutOfSightFadeDuration/2)); math.Abs(got-0.5) > 0.001 {
		t.Fatalf("fade alpha halfway = %.3f, want 0.5", got)
	}

	mode.cleanupVanishedActors(ctx, fade.Started.Add(actors.ActorVanishOutOfSightFadeDuration/2))
	if _, ok := world.Actors[300]; !ok {
		t.Fatal("out-of-sight NPC was removed before fade completed")
	}
	mode.cleanupVanishedActors(ctx, fade.RemoveAt.Add(time.Millisecond))
	if _, ok := world.Actors[300]; ok {
		t.Fatal("out-of-sight NPC remained after fade completed")
	}
	if _, ok := mode.actorSystem.ActorVanishes[300]; ok {
		t.Fatal("out-of-sight fade state remained after cleanup")
	}
}

func TestMobLookChangeToPlayerJobDoesNotChangeDeathSpriteFamily(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1161,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
		Appearance:    true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	actors.ApplyActorLookChange(ctx, network.ActorLookChange{ID: 300, Type: 0, Value: 0})
	if got := world.Actors[300].Job; got != 1161 {
		t.Fatalf("mob job after look change = %d, want 1161", got)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: 1})
	anim, ok := mode.actorSystem.ActorAnims[300]
	if !ok {
		t.Fatal("death animation missing")
	}
	if anim.ActionFamily != sprites.ActionNonPCDeath {
		t.Fatalf("death action = %d, want %d", anim.ActionFamily, sprites.ActionNonPCDeath)
	}
}

func TestPendingSkillTargetCancelWithEscape(t *testing.T) {
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: session.Skill{ID: 6, Level: 2, Range: 9}}}}
	inputState := input.NewState()
	inputState.SetKey(input.KeyEscape, true)

	if !mode.skills().CancelFromInput(client.Context{Input: inputState}) {
		t.Fatal("pending skill target was not canceled")
	}
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want 0", mode.skillSystem.Pending.Skill.ID)
	}
}

func TestBasicMenuOptionTogglesEscapeMenu(t *testing.T) {
	mode := &WorldMode{}

	mode.basicMenuCallbacks(client.Context{}).OnOption()

	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu did not open")
	}
	if mode.ui.escapeMenu.Action() != gameui.EscapeMenuActionNone {
		t.Fatalf("escape menu action = %d, want none", mode.ui.escapeMenu.Action())
	}
	if mode.ui.escapeMenu.Pending() {
		t.Fatal("escape menu kept stale pending state")
	}

	mode.basicMenuCallbacks(client.Context{}).OnOption()

	if mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu stayed open after second option click")
	}
}

func TestEscapeKeyOpensEscapeMenuGlobally(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	inputState.SetKey(input.KeyEscape, true)
	manager := &worldModeTestUIManager{}

	if !mode.openEscapeMenuFromInput(client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}) {
		t.Fatal("escape key did not open escape menu")
	}
	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("escape menu is not open")
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("overlays = %d, want 1", len(manager.overlays))
	}
}

func TestDeathEscapeBypassesInactiveChatConsole(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	manager := &worldModeTestUIManager{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	mode.ui.console.UpdatePresentation(ctx)
	mode.ui.escapeMenu.OpenDeath(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !mode.updateDeathUIInput(ctx) {
		t.Fatal("death UI did not consume Escape")
	}
	if mode.ui.escapeMenu.IsOpen() {
		t.Fatal("inactive chat console consumed Escape before the death menu")
	}
	if !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("hiding the death menu ended death mode")
	}
}

func TestActiveChatGetsFirstDeathEscape(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	manager := &worldModeTestUIManager{}
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	mode.ui.console.UpdatePresentation(ctx)
	mode.ui.escapeMenu.OpenDeath(ctx)

	inputState.SetKey(input.KeyEnter, true)
	if !mode.updateDeathUIInput(ctx) || !mode.ui.console.Active() {
		t.Fatal("Enter did not activate chat while dead")
	}
	inputState.EndFrame()
	inputState.SetKey(input.KeyEnter, false)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, true)

	if !mode.updateDeathUIInput(ctx) {
		t.Fatal("active chat did not consume Escape")
	}
	if mode.ui.console.Active() {
		t.Fatal("Escape did not leave chat input")
	}
	if !mode.ui.escapeMenu.IsOpen() {
		t.Fatal("chat Escape also hid the death menu")
	}
}

func TestPendingSkillTargetCancelWithRightClick(t *testing.T) {
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: session.Skill{ID: 6, Level: 2, Range: 9}}}}
	inputState := input.NewState()
	inputState.SetMouseButton(input.MouseButtonRight, true)

	if !mode.skills().CancelFromInput(client.Context{Input: inputState}) {
		t.Fatal("pending skill target was not canceled")
	}
	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want 0", mode.skillSystem.Pending.Skill.ID)
	}
}

func TestPendingSkillTargetClickIgnoresWalkCooldown(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(64, 64)
	lunatic := worldstate.Actor{
		ID:            300,
		X:             30,
		Y:             20,
		Job:           1063,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(lunatic)

	inputState := input.NewState()
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: db.SkillMGSoulstrike, Level: 1, Type: skills.SkillTargetEnemy, Range: 9},
		MaxLevel: 10,
	}},

		walkCooldownUntil: time.Now().Add(time.Hour),
	}
	ctx := client.Context{
		Input:   inputState,
		Network: network.NewClient(20080910, false),
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	projection := mode.sceneProjection(ctx, ctx.ScreenW, ctx.ScreenH, time.Now())
	point := projection.Project(scene.CellCenter(float64(lunatic.X)), scene.CellCenter(float64(lunatic.Y)), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.skillSystem.Pending.TargetID != lunatic.ID {
		t.Fatalf("pending skill target = %d, want Lunatic %d despite walk cooldown", mode.skillSystem.Pending.TargetID, lunatic.ID)
	}
}

func TestPendingSkillWheelAdjustsLevelAndConsumesWheel(t *testing.T) {
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: 19, Level: 10, Range: 9},
		MaxLevel: 10,
	}},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -2)

	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level was not adjusted")
	}
	if mode.skillSystem.Pending.Skill.Level != 8 {
		t.Fatalf("pending skill level = %d, want 8", mode.skillSystem.Pending.Skill.Level)
	}
	if inputState.WheelY != 0 {
		t.Fatalf("wheel was not consumed: %f", inputState.WheelY)
	}

	inputState.AddWheel(0, 20)
	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level cap was not handled")
	}
	if mode.skillSystem.Pending.Skill.Level != 10 {
		t.Fatalf("pending skill level = %d, want capped to 10", mode.skillSystem.Pending.Skill.Level)
	}
	if inputState.WheelY != 0 {
		t.Fatalf("wheel was not consumed at cap: %f", inputState.WheelY)
	}
}

func TestPendingSkillWheelDoesNotGoBelowLevelOne(t *testing.T) {
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: 19, Level: 2, Range: 9},
		MaxLevel: 10,
	}},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -10)

	if !mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("pending skill level was not adjusted")
	}
	if mode.skillSystem.Pending.Skill.Level != 1 {
		t.Fatalf("pending skill level = %d, want capped to 1", mode.skillSystem.Pending.Skill.Level)
	}
}

func TestPendingSkillWheelIgnoresKnownFixedLevelSkill(t *testing.T) {
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{
		Skill:    session.Skill{ID: db.SkillCRShieldcharge, Level: 5, Range: 3},
		MaxLevel: 5,
	}},
	}
	inputState := input.NewState()
	inputState.AddWheel(0, -1)

	if mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("fixed-level Shield Charge consumed the skill-level wheel")
	}
	if mode.skillSystem.Pending.Skill.Level != 5 || inputState.WheelY != -1 {
		t.Fatalf("fixed-level wheel changed state: skill=%+v wheel=%f", mode.skillSystem.Pending.Skill, inputState.WheelY)
	}
}

func TestPendingSkillWheelIgnoredWithoutPendingSkill(t *testing.T) {
	mode := &WorldMode{}
	inputState := input.NewState()
	inputState.AddWheel(0, -1)

	if mode.skills().AdjustPendingLevelFromWheel(client.Context{Input: inputState}) {
		t.Fatal("wheel was consumed without a pending skill")
	}
	if inputState.WheelY != -1 {
		t.Fatalf("wheel = %f, want unchanged", inputState.WheelY)
	}
}

func TestPendingTargetSkillCancelsWhenClickingGround(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := testutil.Projection(1280, 720, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: session.Skill{ID: 6, Level: 2, Type: skills.SkillTargetEnemy, Range: 9}}}}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.skillSystem.Pending.Skill.ID != 0 {
		t.Fatalf("pending skill id = %d, want canceled", mode.skillSystem.Pending.Skill.ID)
	}
}

func TestPendingGroundSkillDoesNotCancelWhenClickingGround(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GAT = testutil.FlatWalkableGAT(32, 32)
	inputState := input.NewState()
	projection := testutil.Projection(1280, 720, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)
	inputState.SetMousePosition(int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	mode := &WorldMode{skillSystem: skills.System{Pending: skills.PendingTarget{Skill: session.Skill{ID: 18, Level: 1, Type: skills.SkillTargetPlace, Range: 9}}}}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.skills().HandleClick(ctx, projection, time.Now())

	if mode.skillSystem.Pending.Skill.ID != 18 {
		t.Fatalf("pending ground skill id = %d, want still pending after send failure", mode.skillSystem.Pending.Skill.ID)
	}
}

func TestLocalDeathAnimationHoldsUntilPlayerAlive(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Selected:  session.Character{ID: 150000, Job: 0, HP: 0},
			Vitals:    session.Vitals{HP: 0},
		},
		World: world,
	}

	mode.startActorDeath(ctx, 150000)

	if !mode.ui.escapeMenu.IsOpen() || !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death escape menu should open for local death")
	}
	if !ctx.Session.Dead {
		t.Fatal("session should be marked dead")
	}
	anim, ok := mode.actorSystem.ActorAnims[150000]
	if !ok {
		t.Fatal("character death animation missing")
	}
	if anim.ActionFamily != sprites.ActionPCDeath {
		t.Fatalf("death action = %d, want %d", anim.ActionFamily, sprites.ActionPCDeath)
	}
	if !anim.HoldFinal {
		t.Fatal("local death animation should hold final frame")
	}
	accountAnim, ok := mode.actorSystem.ActorAnims[2000000]
	if !ok || !accountAnim.HoldFinal || accountAnim.ActionFamily != sprites.ActionPCDeath {
		t.Fatalf("account death animation = %+v ok=%t", accountAnim, ok)
	}
	if held, ok := mode.actors().ActorAnimation(150000, anim.Started.Add(anim.Duration+time.Second)); !ok || held.ActionFamily != sprites.ActionPCDeath {
		t.Fatalf("expired local death animation = %+v ok=%t", held, ok)
	}

	ctx.Session.Vitals.HP = 1
	mode.clearLocalDeathStateIfAlive(ctx)

	if mode.ui.escapeMenu.IsOpen() || mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death escape menu should clear when player is alive")
	}
	if ctx.Session.Dead {
		t.Fatal("session should be marked alive")
	}
	if _, ok := mode.actorSystem.ActorAnims[150000]; ok {
		t.Fatal("character death animation should clear when player is alive")
	}
	if _, ok := mode.actorSystem.ActorAnims[2000000]; ok {
		t.Fatal("account death animation should clear when player is alive")
	}
}

func TestSavePointRespawnHoldsDeathAnimationUntilMapChange(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := ln.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), addr.IP.String(), addr.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150000,
			Selected:  session.Character{ID: 150000, Job: 0, HP: 0},
			Vitals:    session.Vitals{HP: 0},
		},
		World: world,
	}

	mode.startActorDeath(ctx, 150000)
	mode.ui.escapeMenu.ReturnToSavePoint(ctx)
	if got := mode.ui.escapeMenu.PendingAction(); got != gameui.EscapeMenuActionSavePoint {
		t.Fatalf("pending death action = %d, want save point", got)
	}

	ctx.Session.Vitals.HP = 1
	ctx.Session.Selected.HP = 1
	mode.clearLocalDeathStateIfAlive(ctx)
	if _, ok := mode.actorSystem.ActorAnims[150000]; !ok {
		t.Fatal("positive HP cleared death animation before respawn map change")
	}
	if !mode.ui.escapeMenu.IsOpen() || !mode.ui.escapeMenu.DeathMode() {
		t.Fatal("positive HP closed death menu before respawn map change")
	}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "geffen", X: 120, Y: 80})
	if next == nil {
		t.Fatal("respawn map change did not create the destination world mode")
	}
	if _, ok := mode.actorSystem.ActorAnims[150000]; ok {
		t.Fatal("death animation remained after respawn map change")
	}
	if mode.ui.escapeMenu.IsOpen() || mode.ui.escapeMenu.DeathMode() {
		t.Fatal("death menu remained open after respawn map change")
	}
	if ctx.Session.Dead {
		t.Fatal("session remained dead after respawn map change")
	}
}

func TestActorDeathAlphaFadesOverVisibleDuration(t *testing.T) {
	started := time.Unix(10, 0)
	mode := &WorldMode{actorSystem: actors.System{ActorAnims: map[uint32]scene.ActorAnimation{
		300: {ActionFamily: sprites.ActionNonPCDeath, Started: started, Duration: scene.NonPCDeathFadeDuration},
	}, ActorDeaths: map[uint32]time.Time{
		300: started.Add(scene.NonPCDeathFadeDuration),
	}}}
	if got := mode.actors().ActorDeathAlpha(300, started); got != 1 {
		t.Fatalf("alpha at start = %.2f, want 1", got)
	}
	if got := mode.actors().ActorDeathAlpha(300, started.Add(scene.NonPCDeathFadeDuration/2)); math.Abs(got-0.5) > 0.001 {
		t.Fatalf("alpha halfway = %.2f, want 0.5", got)
	}
	if got := mode.actors().ActorDeathAlpha(300, started.Add(scene.NonPCDeathFadeDuration)); got != 0 {
		t.Fatalf("alpha at end = %.2f, want 0", got)
	}
}

func TestProcessNonPCMotionSoundSchedulesIdleACTSound(t *testing.T) {
	now := time.Unix(10, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	actor := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	}
	world.UpsertActor(actor)
	mode := &WorldMode{actorSystem: actors.System{NonPCViews: map[int]*scene.SpriteView{
		1002: {
			Started: now,
			Act: &res.ACT{
				Actions: []res.ACTAction{
					{Animations: []res.ACTAnimation{{Sound: 0}, {Sound: -1}}, DelayMS: 100},
				},
				Sounds: []string{"poring_idle.wav"},
			},
		},
	}}}
	ctx := client.Context{World: world}

	mode.processNonPCMotionSound(ctx, actor, now)
	mode.processNonPCMotionSound(ctx, actor, now)

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one idle sound", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "poring_idle.wav" {
		t.Fatalf("idle sound = %+v", mode.scheduledSounds[0])
	}
	if sound := mode.scheduledSounds[0]; !sound.positioned || sound.actorID != 300 {
		t.Fatalf("idle sound source = %+v, want actor position", sound)
	}
}

func TestSpatialSoundGainUsesClampedInverseDistance(t *testing.T) {
	for _, tc := range []struct {
		name string
		dist float64
		want float64
	}{
		{name: "same cell", dist: 0, want: 1},
		{name: "inside min", dist: 3, want: 1},
		{name: "at min", dist: 4, want: 1},
		{name: "inverse distance", dist: 8, want: 0.5},
		{name: "at max", dist: 25, want: 4.0 / 25.0},
		{name: "beyond max clamps", dist: 50, want: 4.0 / 25.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := referenceSpatialSoundGain(0, 0, tc.dist, 0)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Fatalf("gain = %.4f, want %.4f", got, tc.want)
			}
		})
	}
}

func TestProcessMapSoundsSchedulesNearbyRSWSound(t *testing.T) {
	now := time.Unix(20, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 110, Y: 220}
	world.GND = &res.GND{Width: 100, Height: 200}
	world.RSW = &res.RSW{Sounds: []res.RSWSound{
		{
			File:     "water.wav",
			Position: res.RSWVector3{X: 10, Z: 20},
			Volume:   0.7,
			Range:    5,
			Cycle:    2,
		},
		{
			File:     "far.wav",
			Position: res.RSWVector3{X: 40, Z: 20},
			Volume:   1,
			Range:    5,
			Cycle:    2,
		},
	}}
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.mapScene.ProcessMapSounds(ctx, now, mode.scheduleSoundVolumeAtPosition)
	mode.mapScene.ProcessMapSounds(ctx, now.Add(time.Second), mode.scheduleSoundVolumeAtPosition)

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one nearby map sound", mode.scheduledSounds)
	}
	sound := mode.scheduledSounds[0]
	if sound.paths[0] != "water.wav" || math.Abs(sound.volume-0.7) > 0.0001 {
		t.Fatalf("scheduled map sound = %+v", sound)
	}
	if !sound.positioned || sound.actorID != 0 || sound.x != 110 || sound.y != 220 {
		t.Fatalf("scheduled map sound source = %+v, want fixed world position", sound)
	}
	if got := mode.mapScene.MapSoundNext[0]; !got.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("next map sound time = %s, want %s", got, now.Add(2*time.Second))
	}
	if _, ok := mode.mapScene.MapSoundNext[1]; ok {
		t.Fatalf("far sound should not have a timer: %+v", mode.mapScene.MapSoundNext)
	}
}

func TestProcessMapSoundsUsesMinimumReplayDelayForZeroCycle(t *testing.T) {
	now := time.Unix(20, 0)
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.GND = &res.GND{Width: 10, Height: 20}
	world.RSW = &res.RSW{Sounds: []res.RSWSound{
		{
			File:   "loop.wav",
			Volume: 1,
			Range:  5,
		},
	}}
	mode := &WorldMode{}
	ctx := client.Context{World: world}

	mode.mapScene.ProcessMapSounds(ctx, now, mode.scheduleSoundVolumeAtPosition)
	mode.mapScene.ProcessMapSounds(ctx, now.Add(50*time.Millisecond), mode.scheduleSoundVolumeAtPosition)
	mode.mapScene.ProcessMapSounds(ctx, now.Add(100*time.Millisecond), mode.scheduleSoundVolumeAtPosition)

	if len(mode.scheduledSounds) != 2 {
		t.Fatalf("scheduled sounds = %+v, want initial sound and replay after 100ms", mode.scheduledSounds)
	}
}
