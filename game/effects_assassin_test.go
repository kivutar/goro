package game

import (
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
	"github.com/kivutar/goro/world"
)

func assassinEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobAssassin}
	w.Actors[300] = world.Actor{ID: 300, X: 16, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestAssassinBuffsAndSplasherSequence(t *testing.T) {
	ctx := assassinEffectContext()
	for _, skill := range []uint16{db.SkillASEnchantpoison, db.SkillASSplasher, db.SkillASPoisonreact} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d produced effects or sounds", skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		want, sounds := effectEnchantPoison, 2
		if skill == db.SkillASPoisonreact {
			want, sounds = effectPoisonReact, 0
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != want || m.worldEffects[0].actorID != 300 || len(m.scheduledSounds) != sounds {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", skill, m.worldEffects, m.scheduledSounds)
		}
	}
	starts := time.Unix(100, 0)
	for _, level := range []uint16{5, 0xffff} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: db.SkillASSplasher, SkillLevel: level, SourceID: 2000000, TargetID: 300, Damage: 50}
		m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
		if level != 0xffff {
			if len(m.worldEffects) != 0 {
				t.Fatal("non-detonation packet started an explosion")
			}
			continue
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectVenomSplasher || !m.worldEffects[0].starts.Equal(starts) {
			t.Fatalf("detonation must start on the server notification: %+v", m.worldEffects)
		}
		if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(starts.Add(10*db.EffectFrameDuration)) {
			t.Fatalf("explosion sound must play at tick 10: %+v", m.scheduledSounds)
		}
	}
	m := &WorldMode{}
	counter := network.ActorActionNotify{SkillID: db.SkillASPoisonreact, SourceID: 2000000, TargetID: 300, Damage: 50}
	m.addSkillEffect(ctx, counter, starts, starts)
	m.addSkillHitEffect(ctx, counter, starts)
	if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Poison React counterattack replayed the buff or invented a second effect")
	}
}

func TestSonicBlowCasterEffectsAndEightImpacts(t *testing.T) {
	ctx := assassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	hitAt := starts.Add(400 * time.Millisecond)
	action := network.ActorActionNotify{SkillID: db.SkillASSonicblow, SourceID: 2000000, TargetID: 300, Damage: 80, HitCount: 8}
	m.addSkillEffect(ctx, action, starts, hitAt)
	m.addSkillHitEffect(ctx, action, hitAt)
	if len(m.worldEffects) != 10 {
		t.Fatalf("effects = %+v", m.worldEffects)
	}
	for i, effect := range m.worldEffects {
		if i < 2 {
			if effect.effectID != []int{effectSonicBlow, effectSonicBlow2}[i] || effect.actorID != 2000000 || !effect.starts.Equal(starts) {
				t.Fatalf("Sonic Blow cast effect on wrong actor or delayed: %+v", effect)
			}
		} else if effect.effectID != effectSonicBlowHit || effect.actorID != 300 || !effect.starts.Equal(hitAt.Add(time.Duration(i-2)*multiHitDelay)) {
			t.Fatalf("wrong Sonic Blow impact: %+v", effect)
		}
	}
}

func TestSonicBlowRingKeepsItsScreenDimensions(t *testing.T) {
	ctx := assassinEffectContext()
	m := &WorldMode{textures: map[string]*render.Image{"__effectfile_effect/alpha_center.tga|blackkey=true": render.NewImage(8, 8)}}
	starts := time.Unix(100, 0)
	spec, _ := worldEffectSpecForID(effectSonicBlow)
	for _, center := range []modelPoint3{{x: 10.5, z: 20.5}, {x: 10.5, z: 35.5}} {
		projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
		frame := render.NewFrame(800, 600)
		m.drawSonicBlowEffect(frame, ctx, projection, spec.components[0], worldEffect{starts: starts}, center.x, center.z, center.y, starts.Add(9*db.EffectFrameDuration))
		command := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Index(0)
		if command.FieldByName("Options").FieldByName("DepthTest").Bool() {
			t.Fatal("Sonic Blow screen ring must not be hidden by scene geometry")
		}
		vertices := command.FieldByName("Vertices")
		origin := projection.Project(center.x, center.z, center.y)
		for i, radius := range []float64{50.375, 60.375} { // ten native radius updates
			v := vertices.Index(i)
			point := projection.Project(v.FieldByName("X").Float(), v.FieldByName("Z").Float(), v.FieldByName("Y").Float())
			if math.Abs(float64(point.x-origin.x)-radius) > .001 || math.Abs(float64(point.y-origin.y)+20) > .001 {
				t.Fatalf("ring point %+v, caster %+v: want radius %g and height 20 pixels", point, origin, radius)
			}
		}
	}
}

func TestVenomKnifeFlightAndAfterimages(t *testing.T) {
	ctx := assassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillASVenomknife, SourceID: 2000000, TargetID: 300}, starts)
	if len(m.worldEffects) != 1 {
		t.Fatal("Venom Knife must launch exactly one projectile")
	}
	effect := m.worldEffects[0]
	first, angle, alpha := venomKnifeAt(effect, 1)
	if math.Abs(first.x-11.1) > 1e-9 || angle != degreesToRadians(15) || alpha != 255 {
		t.Fatalf("knife should immediately advance three native units: %+v, %g, %d", first, angle, alpha)
	}
	near := effect
	near.projectileDirection.x = 1.2
	_, _, nearAlpha := venomKnifeAt(near, 10)
	_, _, farAlpha := venomKnifeAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("flight time must depend on distance")
	}
	if _, _, alpha := venomKnifeAt(effect, 30); alpha != 0 {
		t.Fatal("knife must fade after arriving")
	}
	m.textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/베넘나이프.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	m.drawWorldEffects(frame, ctx, projection, starts.Add(6*db.EffectFrameDuration), false)
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if commands.Len() != 4 {
		t.Fatalf("knife and three afterimages: got %d commands", commands.Len())
	}
	for i, process := range []int{7, 6, 4, 2} {
		want, _, _ := venomKnifeAt(effect, process)
		billboard := commands.Index(i)
		if math.Abs(billboard.FieldByName("Center").Index(0).Float()-want.x) > 1e-5 {
			t.Fatalf("afterimage %d did not use the previous two-tick sample", i)
		}
	}
}

func TestGrimtoothCastPackets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage int32
	}{
		{"normal", -30000},
		{"hidden WoE damage", 1},
		{"hallucination", 1234},
		{"hallucination zero", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := assassinEffectContext()
			ctx.World.Actors[301] = world.Actor{ID: 301, X: 16, Y: 21, Job: 1002}
			ctx.World.Actors[2000001] = world.Actor{ID: 2000001, X: 10, Y: 21, Job: db.JobAssassin}
			m := &WorldMode{}
			starts := time.Unix(100, 0)
			action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SkillLevel: 5,
				SourceID: 2000000, TargetID: 300, ServerTick: 1000, SourceSpeed: 500,
				Damage: tc.damage, HitCount: 1, Action: network.ActorActionSkill}
			m.addSkillEffect(ctx, action, starts, starts.Add(500*time.Millisecond))
			if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatalf("cast must start a trail and sound regardless of damage: %+v", m.worldEffects)
			}
			// The preamble may refer to a splash victim, not the primary target.
			action.TargetID, action.Damage, action.TargetSpeed = 301, 30, 200
			m.addSkillEffect(ctx, action, starts.Add(time.Millisecond), starts.Add(500*time.Millisecond))
			action.Action = 5 // splash damage must never launch a trail
			m.addSkillEffect(ctx, action, starts.Add(2*time.Millisecond), starts.Add(500*time.Millisecond))
			if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatal("damage packets duplicated the trail or sound")
			}
			action.Action = network.ActorActionSkill
			action.SourceID = 2000001
			m.addSkillEffect(ctx, action, starts, starts)
			action.SourceID, action.ServerTick = 2000000, 1500
			m.addSkillEffect(ctx, action, starts.Add(500*time.Millisecond), starts.Add(time.Second))
			if len(m.worldEffects) != 3 || len(m.scheduledSounds) != 3 {
				t.Fatal("another caster or subsequent cast was suppressed while the first trail was active")
			}
			m = &WorldMode{}
			action.Action = 5
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a splash-only notification started a trail")
			}
			action.Action, action.SourceID = network.ActorActionSkill, 999
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a missing caster started a trail at the victim")
			}
		})
	}
}

func TestGrimtoothPreambleAndFixedTrail(t *testing.T) {
	ctx := assassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SourceID: 2000000, TargetID: 300, Damage: -30000, Action: network.ActorActionSkill}
	m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
	action.Damage = 30
	m.addSkillEffect(ctx, action, starts, starts)
	m.addSkillHitEffect(ctx, action, starts)
	action.Action = 5 // secondary victim's damage-only packet
	m.addSkillEffect(ctx, action, starts, starts)
	if len(m.worldEffects) != 2 || m.worldEffects[0].effectID != effectGrimtooth || m.worldEffects[1].effectID != effectGrimtoothAtk {
		t.Fatalf("one cast must produce one trail and one impact per victim: %+v", m.worldEffects)
	}
	trail := m.worldEffects[0]
	if trail.actorID != 2000000 || m.scheduledSounds[0].actorID != 2000000 || trail.actorOrigin.x != 10.5 || trail.actorOrigin.z != 20.5 || math.Abs(trail.projectileDirection.x-.4) > 1e-9 || !trail.starts.Equal(starts) {
		t.Fatalf("trail does not start at caster toward target: %+v", trail)
	}
	m.textures = map[string]*render.Image{"__effectfile_effect/stone.bmp|blackkey=true": render.NewImage(8, 8)}
	m.worldEffects = m.worldEffects[:1]
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	draw := func() reflect.Value {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(30*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("expected one batched trail, got %d commands", commands.Len())
		}
		return commands.Index(0).FieldByName("Vertices")
	}
	before := draw()
	if before.Len() != 11*12 {
		t.Fatalf("tick 30 should have eleven horns: %d vertices", before.Len())
	}
	ctx.World.Player.X = 80
	delete(ctx.World.Actors, 300)
	after := draw()
	for i := 0; i < before.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if before.Index(i).FieldByName(axis).Float() != after.Index(i).FieldByName(axis).Float() {
				t.Fatal("moving actors pulled an already emitted Grimtooth trail")
			}
		}
	}
	spike := grimtoothSpike(trail, 0, 6)
	if spike.origin != grimtoothSpike(trail, 0, 20).origin {
		t.Fatal("trail spike kept moving after the native seven updates")
	}
	if grimtoothSpike(trail, 0, 39).alpha != 0 {
		t.Fatal("trail spike did not finish fading")
	}
	impact := worldEffect{effectID: effectGrimtoothAtk}
	if grimtoothSpike(impact, 0, 10).origin != grimtoothSpike(impact, 0, 989).origin || grimtoothSpike(impact, 0, 999).alpha != 0 {
		t.Fatal("impact spikes should stop after eleven ticks and fade at the end")
	}
}

func TestAssassinPoisonParticleMotionAndFieldLifetime(t *testing.T) {
	for _, id := range []int{effectEnchantPoison, effectVenomDust2} {
		effect := worldEffect{effectID: id, starts: time.Unix(100, 0)}
		life := 40
		if id == effectVenomDust2 {
			life = 10
		}
		first := poisonParticleAt(effect, 0, 0)
		later := poisonParticleAt(effect, 0, life-3)
		if first.alpha <= 0 || later.position.y <= first.position.y || later.angle <= first.angle || poisonParticleAt(effect, 0, life-1).alpha != 0 {
			t.Fatalf("effect %d has wrong rise, rotation or fade: %+v / %+v", id, first, later)
		}
		if id == effectVenomDust2 && (first.position.x != 0 || first.position.z != 0) {
			t.Fatal("Venom Dust particles must stay inside their field cell")
		}
	}
	ctx := assassinEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, CreatorID: 2000000, UnitID: 146, X: 12, Y: 20, Visible: true})
	if len(m.worldEffects) != 1 || !m.worldEffects[0].persistent || m.worldEffects[0].effectID != effectVenomDust2 {
		t.Fatal("Venom Dust must be owned by the server's ground unit")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
	if len(m.worldEffects) != 0 {
		t.Fatal("Venom Dust survived its field's removal")
	}
}

func TestAssassinPoisonRotationMatchesSakexe(t *testing.T) {
	ctx := assassinEffectContext()
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	for _, id := range []int{effectEnchantPoison, effectVenomDust2} {
		spec, _ := worldEffectSpecForID(id)
		m := &WorldMode{effectViews: map[string]*spriteView{spec.components[0].spriteFile: effectRenderTestSprite()}}
		m.addWorldEffectAt(ctx, id, 300, starts)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts, false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 1 {
			t.Fatalf("effect %d: expected one initial particle, got %d", id, commands.Len())
		}
		command := commands.Index(0)
		center, axis := command.FieldByName("Center"), command.FieldByName("RightAxis")
		_, up, _, _ := projection.BillboardBasis(center.Index(0).Float(), center.Index(2).Float(), center.Index(1).Float())
		right := modelPoint3{x: axis.Index(0).Float(), y: axis.Index(1).Float(), z: axis.Index(2).Float()}
		// Sakexe's positive roll turns the sprite's right axis down the screen.
		if vertical := dot3(right, up); vertical >= 0 {
			t.Errorf("effect %d: particle rotates upward instead of downward: %g", id, vertical)
		}
	}
}

func TestAssassinRealEffectAssets(t *testing.T) {
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
	for _, id := range []int{effectCloaking, effectSonicBlow, effectSonicBlow2, effectSonicBlowHit, effectGrimtooth, effectGrimtoothAtk, effectEnchantPoison, effectPoisonReact, effectVenomDust, effectVenomDust2, effectVenomSplasher, effectThrowItem6} {
		spec, _ := worldEffectSpecForID(id)
		for _, c := range spec.components {
			if c.textureFile != "" && m.effectFileTexture(manager, c.textureFile) == nil {
				t.Errorf("effect %d missing texture %s", id, c.textureFile)
			}
			if c.spriteFile != "" && m.effectSpriteView(manager, c.spriteFile) == nil {
				t.Errorf("effect %d missing sprite %s", id, c.spriteFile)
			}
			if c.strFile != "" && m.loadWorldEffectSTR(manager, c.strFile, c.texturePath) == nil {
				t.Errorf("effect %d missing STR %s", id, c.strFile)
			}
		}
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
	ctx := assassinEffectContext()
	ctx.Resources = manager
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	for _, id := range []int{effectSonicBlow, effectGrimtooth, effectGrimtoothAtk, effectEnchantPoison, effectVenomDust2, effectThrowItem6} {
		m.worldEffects = nil
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, starts)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(6*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
			t.Errorf("effect %d loaded its assets but rendered nothing", id)
		}
	}
}
