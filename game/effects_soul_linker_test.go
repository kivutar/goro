package game

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func soulLinkerEffectContext() client.Context {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20, Job: db.JobLinker}
	w.Actors[300] = worldstate.Actor{ID: 300, X: 18, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 150000}}
}

func soulLinkerTestTextures(m *WorldMode) {
	m.textures = make(map[string]*render.Image)
	for _, id := range []int{effectStin, effectStin3, effectHitLine4, effectStin2, effectHitLine6, effectHitTexture, effectAsuraBodyMonster, effectHated, effectKaizel, effectBlueBody, effectReflectBody, effectSmaReady, effectSoulLink, effectAsuraBody, effectBlueCasting, effectTaeReady, effectStormKick4, effectBabyBodyBack, effectM07} {
		for _, c := range db.EffectSpecs[id].Components {
			for _, file := range append([]string{c.TextureFile}, c.TextureFiles...) {
				if file != "" {
					m.textures["__effectfile_"+file+"|blackkey=true"] = render.NewImage(8, 8)
				}
			}
		}
	}
}

func TestSoulLinkerSkillSequenceMatchesOriginalClient(t *testing.T) {
	for _, tc := range []struct {
		skill              uint16
		projectile, impact int
	}{
		{db.SkillSLStin, effectStin, effectQuakeBody3},
		{db.SkillSLStun, effectStin3, effectHitLine4},
	} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		starts := time.Unix(100, 0)
		hitAt := starts.Add(500 * time.Millisecond)
		action := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, Damage: 10}
		m.addSkillEffect(ctx, action, starts, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.worldEffects) != 3 {
			t.Fatalf("skill %d: effects = %+v", tc.skill, m.worldEffects)
		}
		for i, id := range []int{tc.projectile, tc.impact, effectBlueHit} {
			effect := m.worldEffects[i]
			want := starts
			if i == 2 {
				want = hitAt
			}
			if effect.effectID != id || effect.actorID != 300 || !effect.starts.Equal(want) {
				t.Fatalf("skill %d: wrong effect order, owner or timing: %+v", tc.skill, effect)
			}
		}
		projectile := m.worldEffects[0]
		if projectile.actorOrigin.x != 10.5 || projectile.actorOrigin.z != 20.5 || projectile.projectileDirection != (modelPoint3{x: 1}) {
			t.Fatalf("projectile starts at target instead of caster: %+v", projectile)
		}
		if len(m.scheduledSounds) != 1 {
			t.Fatalf("skill %d: sounds = %+v", tc.skill, m.scheduledSounds)
		}
		m.worldEffects = nil
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.worldEffects) != 0 {
			t.Fatal("a miss must not flash the target blue")
		}
	}
}

func TestEstinNativeMotionAndTrail(t *testing.T) {
	distance, alpha := estinHead(10)
	wantDistance, speed := .8, .8
	for i := 2; i <= 10; i++ {
		speed *= 1.02
		wantDistance += speed
	}
	if math.Abs(distance-wantDistance/5) > 1e-9 || alpha != 225.0/255 {
		t.Fatalf("end of rise = %g/%g", distance, alpha)
	}
	if _, alpha := estinHead(55); alpha != 0 {
		t.Fatal("Estin head should finish fading at tick 55")
	}
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.addWorldEffectBetweenAt(ctx, effectStin, 300, 2000000, starts)
	projection := newSceneProjectionForTarget(800, 600, 14.5, 20.5, 0)
	draw := func() *render.Frame {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(16*db.EffectFrameDuration), false)
		return frame
	}
	frame := draw()
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Estin should draw one batched wind trail")
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() != 16 {
		t.Fatalf("trail vertices=%d, want four squares", vertices.Len())
	}
	for i := 0; i < 16; i++ {
		if math.Abs(vertices.Index(i).FieldByName("Y").Float()-m.worldEffects[0].actorOrigin.y) > 1e-5 {
			t.Fatal("Estin must use world-horizontal quads as in RenderSTIN")
		}
	}
	ctx.World.Player.X, ctx.World.Player.Y = 50, 50
	delete(ctx.World.Actors, 300)
	again := reflect.ValueOf(draw()).Elem().FieldByName("worldCommands").Index(0).FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if vertices.Index(i).FieldByName(axis).Float() != again.Index(i).FieldByName(axis).Float() {
				t.Fatal("movement/removal bent an already launched projectile")
			}
		}
	}
}

func TestEstunEmitsParticlesAndStops(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.addWorldEffectBetweenAt(ctx, effectStin3, 300, 2000000, starts)
	effect := m.worldEffects[0]
	spec, _ := worldEffectSpecForID(effectStin3)
	projection := newSceneProjectionForTarget(800, 600, 14.5, 20.5, 0)
	for _, tc := range []struct{ frame, count int }{{-1, 0}, {0, 0}, {1, 4}, {10, 40}, {199, 196}, {230, 76}, {250, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawSoulLinkerParticles(frame, ctx, projection, spec.components[0], effect, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("tick %d: %d particles, want %d", tc.frame, draws.Len(), tc.count)
		}
		for i := 0; i < draws.Len(); i++ {
			if draws.Index(i).FieldByName("Center").Index(1).Float() <= effect.actorOrigin.y {
				t.Fatal("particles must rise above the emitter")
			}
		}
	}
}

func TestSoulLinkerTargetFlashAndShake(t *testing.T) {
	ctx := soulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{worldEffects: []worldEffect{{effectID: effectBlueHit, actorID: 300, starts: starts}}}
	billboard := &spriteBillboard{image: render.NewImage(64, 64), anchorX: 32, anchorY: 60}
	projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
	entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 18.5, worldY: 20.5, scale: 1, shadow: 1}
	for _, tc := range []struct{ frame, count int }{{-1, 1}, {0, 1}, {4, 3}, {16, 3}, {17, 1}} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("tick %d: body draws %d, want %d", tc.frame, draws.Len(), tc.count)
		}
		if tc.count > 0 && draws.Index(0).FieldByName("ColorB").Float() != 1 {
			t.Fatal("base body must remain visible under the overlay")
		}
		if tc.count > 0 {
			normal := render.NewFrame(800, 600)
			m.drawActorSpriteBillboardTintAlpha3D(normal, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver)
			base := reflect.ValueOf(normal).Elem().FieldByName("worldBillboards").Index(0)
			for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
				for axis := 0; axis < 3; axis++ {
					if math.Abs(base.FieldByName(name).Index(axis).Float()-draws.Index(0).FieldByName(name).Index(axis).Float()) > 1e-5 {
						t.Fatalf("flash changed normal sprite geometry: %s", name)
					}
				}
			}
			for i := 1; i < tc.count; i++ {
				if draws.Index(i).FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
					t.Fatal("blue body overlays must use additive blending")
				}
			}
		}
	}
	m.worldEffects[0].effectID = effectQuakeBody3
	for _, tick := range []int{30, 31, 50, 51} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		active := math.Abs(draw.FieldByName("AnchorX").Float()-32) > 1e-5 || math.Abs(draw.FieldByName("AnchorY").Float()-60) > 1e-5
		if active != (tick > 30 && tick <= 50) {
			t.Fatalf("shake active at tick %d: %v", tick, active)
		}
	}
	if billboard.anchorX != 32 || billboard.anchorY != 60 || ctx.World.Actors[300].X != 18 {
		t.Fatal("shake mutated sprite or actor")
	}
}

func TestEstunImpactStreaksHaveBoundedLifetime(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	spec, _ := worldEffectSpecForID(effectHitLine4)
	starts := time.Unix(100, 0)
	effect := worldEffect{effectID: effectHitLine4, actorID: 300, starts: starts}
	for _, tick := range []int{5, 30, 80, 200} {
		frame := render.NewFrame(800, 600)
		m.drawSoulLinkerHitLines(frame, ctx, spec.components[0], effect, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 30) {
			t.Fatalf("tick %d: impact draws=%d", tick, commands.Len())
		}
		if tick == 30 {
			vertices := commands.Index(0).FieldByName("Vertices")
			indices := commands.Index(0).FieldByName("Indices")
			for i := 0; i < indices.Len(); i++ {
				if int(indices.Index(i).Uint()) >= vertices.Len() {
					t.Fatal("invalid streak mesh index")
				}
			}
		}
	}
}

func TestSoulLinkerRealEffectAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real effect assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	m := &WorldMode{}
	for _, id := range []int{effectStin, effectStin3, effectHitLine4, effectStin2, effectHitLine6, effectHitTexture, effectAsuraBodyMonster, effectHated, effectKaizel, effectBlueBody, effectReflectBody, effectSmaReady, effectSoulLink, effectAsuraBody, effectBlueCasting, effectTaeReady, effectStormKick4, effectBabyBodyBack, effectM07} {
		spec, _ := worldEffectSpecForID(id)
		for _, c := range spec.components {
			if c.spriteFile != "" && m.effectSpriteView(manager, c.spriteFile) == nil {
				t.Errorf("effect %d missing sprite %s", id, c.spriteFile)
			}
			paths := append([]string{c.textureFile}, c.textureFiles...)
			for _, p := range paths {
				if p != "" && m.effectFileTexture(manager, p) == nil {
					t.Errorf("effect %d missing %s", id, p)
				}
			}
		}
		for _, s := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + s); err != nil {
				t.Error(err)
			}
		}
	}
	for _, skill := range []uint16{db.SkillSLSka, db.SkillSLSwoo, db.SkillMOExplosionspirits} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestSoulLinkerBuffsUseOriginalVisualsAndOneSound(t *testing.T) {
	for _, tc := range []struct {
		skill uint16
		id    int
		sound string
	}{
		{db.SkillSLKaahi, effectHated, "effect\\t_보조마법.wav"},
		{db.SkillSLKaizel, effectKaizel, "effect\\priest_resurrection.wav"},
		{db.SkillSLKaupe, effectBlueBody, "effect\\t_치잉.wav"},
		{db.SkillSLKaite, effectReflectBody, "effect\\t_마법반사.wav"},
		{db.SkillSLSke, effectAsuraBodyMonster, "effect\\t_공격력.wav"},
	} {
		t.Run(db.SkillResourceName[tc.skill], func(t *testing.T) {
			ctx := soulLinkerEffectContext()
			m := &WorldMode{}
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 0})
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("failed skill produced effects")
			}
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1})
			if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != 300 {
				t.Fatalf("effects = %+v", m.worldEffects)
			}
			if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{tc.sound}) {
				t.Fatalf("sounds = %+v", m.scheduledSounds)
			}
		})
	}
}

func TestEsmaSequenceAndHoming(t *testing.T) {
	ctx := soulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillSLSma, Damage: 100}, starts, starts.Add(500*time.Millisecond))
	if len(m.worldEffects) != 4 {
		t.Fatalf("Esma effects = %+v", m.worldEffects)
	}
	for i, id := range []int{effectFourWayBody, effectStin2, effectHitLine6, effectHitTexture} {
		if e := m.worldEffects[i]; e.effectID != id || e.actorID != 300 || !e.starts.Equal(starts) {
			t.Fatalf("wrong Esma component/timing: %+v", e)
		}
	}
	if len(m.scheduledSounds) != 5 {
		t.Fatalf("Esma sounds = %+v", m.scheduledSounds)
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(5+15*i) * db.EffectFrameDuration)) {
			t.Fatalf("wrong Esma sound timing: %+v", sound)
		}
	}
	e := m.worldEffects[1]
	e.esma.advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	left, right := e.esma.heads[0].samples[0], e.esma.heads[1].samples[0]
	if left.position.z <= e.actorOrigin.z || right.position.z >= e.actorOrigin.z || math.Abs(left.position.x-right.position.x) > 1e-8 {
		t.Fatalf("Esma must leave the caster in opposite arcs: %v %v", left, right)
	}
	// A second render pass in the same tick must not advance the simulation.
	before := *e.esma
	e.esma.advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	if *e.esma != before {
		t.Fatal("drawing twice advanced Esma")
	}
	actor := ctx.World.Actors[300]
	actor.X, actor.Y = 24, 30
	ctx.World.Actors[300] = actor
	e.esma.advance(ctx, e, starts.Add(11*db.EffectFrameDuration))
	wantHistory := before.heads[0].samples[1]
	wantHistory.alpha *= .8
	if e.esma.heads[0].samples[2] != wantHistory {
		t.Fatal("moving target rewrote an old trail position")
	}
	for i := range e.esma.heads {
		step := sub3(e.esma.heads[i].samples[0].position, before.heads[i].samples[0].position)
		if math.Abs(math.Hypot(step.x, step.z)-.4) > 1e-8 {
			t.Fatal("moving target teleported an Esma head")
		}
	}
	delete(ctx.World.Actors, 300)
	e.esma.advance(ctx, e, starts.Add(100*db.EffectFrameDuration))
	for _, head := range e.esma.heads {
		for _, sample := range head.samples {
			if sample.alpha > 0 {
				t.Fatal("Esma must fade when its target leaves")
			}
		}
	}
}

func TestSoulLinkerPlayerGlowsAndKaiteStatus(t *testing.T) {
	ctx := soulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	billboard := &spriteBillboard{image: render.NewImage(64, 64), anchorX: 32, anchorY: 60}
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	draw := func(now time.Time) reflect.Value {
		frame := render.NewFrame(800, 600)
		entry := sceneActorDrawEntry{actor: ctx.World.Player, worldX: 10.5, worldY: 20.5, scale: 1, shadow: 1}
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, .5, .6, .8, color.RGBA{100, 120, 200, 200}, render.BlendSourceOver, now)
		return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	}
	base := draw(starts).Index(0)
	for _, id := range []int{effectHated, effectKaizel, effectBlueBody, effectReflectBody} {
		// The player's sprite uses CharID, while the effect arrives with AccountID.
		m.worldEffects = []worldEffect{{effectID: id, actorID: ctx.Session.AccountID, starts: starts}}
		commands := draw(starts.Add(40 * db.EffectFrameDuration))
		if commands.Len() != 3 {
			t.Fatalf("effect %d: want body and two blue passes, got %d", id, commands.Len())
		}
		for _, field := range []string{"ColorR", "ColorG", "ColorB", "ColorA", "AnchorX", "AnchorY"} {
			if base.FieldByName(field).Float() != commands.Index(0).FieldByName(field).Float() {
				t.Fatalf("effect %d changed player's %s", id, field)
			}
		}
		if draw(starts.Add(200*db.EffectFrameDuration)).Len() != 1 {
			t.Fatalf("effect %d glow did not end", id)
		}
	}
	m.worldEffects = nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusKaite, ActorID: ctx.Session.AccountID, Active: true})
	if draw(starts).Len() != 5 {
		t.Fatal("Kaite needs four silhouettes and the player body")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusKaite, ActorID: ctx.Session.AccountID, Active: false})
	if draw(starts).Len() != 1 {
		t.Fatal("Kaite silhouettes survived the server clearing the status")
	}
}

func TestSoulLinkerParticleAndKaizelTiming(t *testing.T) {
	ctx := soulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
	for _, tc := range []struct{ id, tick, count int }{
		{effectHated, 0, 0}, {effectHated, 1, 4}, {effectHated, 5, 8}, {effectHated, 135, 0},
		{effectSmaReady, 39, 0}, {effectSmaReady, 41, 4}, {effectSmaReady, 175, 0},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := worldEffectSpecForID(tc.id)
		m.addWorldEffectAt(ctx, tc.id, 300, starts)
		effect := m.worldEffects[len(m.worldEffects)-1]
		m.drawSoulLinkerParticles(frame, ctx, projection, spec.components[0], effect, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != tc.count {
			t.Fatalf("effect %d tick %d: got %d particles, want %d", tc.id, tc.tick, n, tc.count)
		}
	}
	spec, _ := worldEffectSpecForID(effectKaizel)
	for _, tick := range []int{14, 15, 25, 45} {
		frame := render.NewFrame(800, 600)
		m.drawKaizelEffect(frame, ctx, spec.components[1], worldEffect{starts: starts}, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 15 || tick == 25) {
			t.Fatalf("Kaizel burst at wrong tick %d", tick)
		}
		if commands.Len() > 0 && commands.Index(0).FieldByName("Vertices").Len() != 8*3*4 {
			t.Fatal("Kaizel needs eight blades of three quads each")
		}
	}
}

func TestSoulLinkerBuffParticlesKeepEmissionPositions(t *testing.T) {
	for _, id := range []int{effectHated, effectKaizel, effectSmaReady} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		starts := time.Unix(100, 0)
		m.addWorldEffectAt(ctx, id, 300, starts)
		projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
		first := 0
		if id == effectSmaReady {
			first = 40
		}
		draw := func(tick int) reflect.Value {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		}
		before := draw(first + 1)
		actor := ctx.World.Actors[300]
		actor.X += 10
		ctx.World.Actors[300] = actor
		after := draw(first + 5)
		if before.Len() != 4 || after.Len() != 8 {
			t.Fatalf("effect %d: particles before/after movement = %d/%d", id, before.Len(), after.Len())
		}
		for i := 0; i < 4; i++ {
			for _, axis := range []int{0, 2} {
				if before.Index(i).FieldByName("Center").Index(axis).Float() != after.Index(i).FieldByName("Center").Index(axis).Float() {
					t.Fatalf("effect %d: movement relocated an emitted particle", id)
				}
			}
			if after.Index(i+4).FieldByName("Center").Index(0).Float() < 27 {
				t.Fatalf("effect %d: new particles did not follow the actor", id)
			}
		}
	}
}

func TestEskeFollowsMonsterOptionAndEndsOnClear(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	m.handleNetworkPacket(ctx, soulLinkerSkillPacket(db.SkillSLSke, 1), time.Now())
	actor := ctx.World.Actors[300]
	if actor.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatal("Eske did not set the monster's native option bit")
	}
	if tint := actorStateTint(actor); tint != (color.RGBA{255, 255, 255, 255}) {
		t.Fatalf("Eske should not use the player's Energy Coat tint: %v", tint)
	}
	starts := m.worldEffects[0].starts
	projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
	billboard := &spriteBillboard{image: render.NewImage(64, 64), anchorX: 32, anchorY: 60}
	draw := func(at time.Time) int {
		frame := render.NewFrame(800, 600)
		entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 18.5, worldY: 20.5}
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, at)
		return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
	}
	if draw(starts.Add(time.Second)) != 11 {
		t.Fatal("Eske burst needs ten expanding outlines")
	}
	if draw(starts.Add(time.Minute)) != 5 {
		t.Fatal("Eske must retain four faint outlines after the burst")
	}
	m.handleNetworkPacket(ctx, npcEffectUpdatePacket(300, 0), time.Now())
	if len(m.worldEffects) != 0 || draw(starts.Add(time.Second)) != 1 {
		t.Fatal("Eske persisted after its server status ended")
	}
	// The same option bit on a player still means Energy Coat, not Eske.
	ctx.World.Player.Opt3State = db.Opt3Energycoat
	if actorStateTint(ctx.World.Player) == (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("monster Eske changed player Energy Coat tint")
	}
}

func TestEsmaImpactLayersAndReadyNotification(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
	m.addWorldEffectAt(ctx, effectHitLine6, 300, starts)
	m.addWorldEffectAt(ctx, effectHitTexture, 300, starts)
	for _, tick := range []int{25, 50, 120} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		mesh := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		quads := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if (mesh.Len() > 0) != (tick == 50) || (quads.Len() > 0) != (tick == 50) {
			t.Fatalf("wrong impact timing at %d: streaks=%d, flash=%d", tick, mesh.Len(), quads.Len())
		}
	}
	m.worldEffects = nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSmaReady, ActorID: ctx.Session.AccountID, Active: true})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectSmaReady || len(m.scheduledSounds) != 0 {
		t.Fatal("Esma readiness must launch its silent particle effect")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSmaReady, ActorID: ctx.Session.AccountID, Active: false})
	if len(m.worldEffects) != 1 {
		t.Fatal("consuming readiness must neither replay nor cut off the native one-shot sparkle")
	}
}
