package effects

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestGunslingerSpriteDelayAndGroundLoops(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &System{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{TripleAction, LightningS} {
		m.Active = nil
		m.AddAt(ctx, id, 300, start)
		if id == LightningS {
			m.Active[0].Persistent = true
			m.Active[0].Expires = time.Time{}
		}
		for _, tick := range []int{0, 16, 29, 30, 80, 100} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
			if id == TripleAction {
				if (n > 0) != (tick == 30) {
					t.Fatalf("Triple Action tick %d draws=%d", tick, n)
				}
			} else {
				want := 1
				if tick >= 16 && tick%16 < 7 {
					want = 2
				}
				if n != want {
					t.Fatalf("grenade tick %d draws=%d want %d", tick, n, want)
				}
			}
		}
	}
	ctx.Session.Selected.Job = db.JobGunslinger
	ctx.Session.Selected.Weapon = 0
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Gunslinger sprite missing: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			for family, minFrames := range map[int]int{sprites.ActionPCSkill: 6, sprites.ActionPCAttack1: 3} {
				_, action, ok := sprites.ResolveAction(view.Body.Act, family, direction)
				if !ok || len(action.Animations) < minFrames {
					t.Fatalf("missing Gunslinger pose sex=%d direction=%d family=%d", sex, direction, family)
				}
			}
		}
	}
	for _, path := range []string{"effect\\플립.wav", "effect\\크래커.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int{456, 584, 623, 627, 628, 637, 643, 644, 648} {
		spec, _ := Lookup(id)
		for _, path := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestNinjaNativeAssetsAndRendering(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobNinja, db.JobNinja
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("missing Ninja sprite: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			_, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, direction)
			if !ok || len(action.Animations) < 6 {
				t.Fatalf("Ninja sex %d direction %d lacks native skill poses", sex, direction)
			}
		}
	}
	m := &System{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{613, 614, 615, 616, 618, 619, 620, 621, 622, 624, 630, 631, 632, 633, 634, 635, 636} {
		m.Active = nil
		if !m.AddBetween(ctx, id, 300, 2000000, start) {
			t.Fatalf("effect %d missing", id)
		}
		e := m.Active[0]
		e.HitCount = 3
		spec, _ := Lookup(id)
		for _, c := range spec.Components {
			if c.SpriteFile != "" && m.EffectSpriteView(ctx.Resources, c.SpriteFile) == nil {
				t.Fatalf("missing SPR %s", c.SpriteFile)
			}
			if c.TextureFile != "" && m.EffectFileTexture(ctx.Resources, c.TextureFile) == nil {
				t.Fatalf("missing texture %s", c.TextureFile)
			}
		}
		if id == 618 || id == 621 || id == 622 || id == 624 || id == 631 || id == 634 {
			frame := render.NewFrame(800, 600)
			for _, c := range spec.Components {
				m.DrawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(10*db.EffectFrameDuration))
			}
			v := reflect.ValueOf(frame).Elem()
			if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
				t.Fatalf("effect %d produced no draw commands", id)
			}
		}
	}
}

func TestStalkerClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Stalker sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{scene.SpriteActionIdle, sprites.ActionPCReadyFight, sprites.ActionPCSkill} {
				if _, action, ok := sprites.ResolveAction(view.Body.Act, family, dir); !ok || len(action.Animations) == 0 {
					t.Fatal("Stalker casting/release ACT missing")
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ id, tick, billboards, vertices int }{
		{495, 0, 0, 0}, {495, 50, 32, 0}, {495, 109, 0, 0},
		{496, 10, 0, 180}, {496, 44, 0, 0},
		{db.SkillEffectChaseWalkSmoke, 0, 1, 0},
	} {
		m := &System{}
		if !m.AddAt(ctx, tc.id, 300, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		frame := render.NewFrame(800, 600)
		m.Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem()
		commands := draws.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if tc.id == 495 && tc.tick == 50 {
			billboards := draws.FieldByName("worldBillboards")
			for i := 0; i < billboards.Len(); i++ {
				draw := billboards.Index(i)
				c := draw.FieldByName("Center")
				dx, dy, dz := c.Index(0).Float()-20.5, c.Index(1).Float()-.07, c.Index(2).Float()-20.5
				radius := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if radius < 3.19 || radius > 9.01 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
					t.Fatal("Full Strip fragments need native travel distance and additive blending")
				}
			}
		}
		if draws.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.id, tc.tick, draws.FieldByName("worldBillboards").Len(), vertices)
		}
	}
	guard := &System{}
	guard.AddAt(ctx, 496, 300, start)
	before := render.NewFrame(800, 600)
	guard.Draw(before, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	actorBefore := ctx.World.Actors[300]
	actorAfter := actorBefore
	actorAfter.Dir = (actorBefore.Dir + 4) % 8
	actorAfter.X += 1
	ctx.World.Actors[300] = actorAfter
	after := render.NewFrame(800, 600)
	guard.Draw(after, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	for c := 0; c < 3; c++ {
		a := reflect.ValueOf(before).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		b := reflect.ValueOf(after).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		for v := 0; v < a.Len(); v++ {
			if math.Abs(b.Index(v).FieldByName("X").Float()-a.Index(v).FieldByName("X").Float()-1) > .00001 || b.Index(v).FieldByName("Z").Float() != a.Index(v).FieldByName("Z").Float() {
				t.Fatal("Preserve must follow position but keep its initial rotation")
			}
		}
	}
	ctx.World.Actors[300] = actorBefore
	for _, id := range []int{392, 495, 496, 501} {
		spec, _ := Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound %s: %v", id, sound, err)
			}
		}
	}
	m := &System{}
	view := m.EffectSpriteView(ctx.Resources, "smoke")
	if view == nil || len(view.Act.Actions[0].Animations) == 0 {
		t.Fatal("native smoke ACT missing")
	}
	m.AddAt(ctx, db.SkillEffectChaseWalkSmoke, 300, start)
	e := m.Active[0]
	if e.Expires.Sub(start) != time.Duration(len(view.Act.Actions[0].Animations))*(5*db.EffectFrameDuration/6) {
		t.Fatal("smoke must play once at native speed")
	}
	actor := ctx.World.Actors[300]
	actor.X += 10
	ctx.World.Actors[300] = actor
	frame := render.NewFrame(800, 600)
	m.Draw(frame, ctx, projection, start, false)
	draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
	if math.Abs(draw.FieldByName("Center").Index(0).Float()-20.5) > .001 {
		t.Fatal("smoke followed the hidden actor instead of staying at release position")
	}
}

func TestTaekwonEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	for _, job := range []int16{db.JobTaekwon, db.JobStar, db.JobLinker} {
		ctx.World.Player.Job, ctx.Session.Selected.Job = job, job
		ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
		for _, sex := range []uint8{0, 1} {
			view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
			if view == nil {
				t.Fatalf("job %d sprite missing: %s", job, status)
			}
			for _, pose := range []struct{ family, motion int }{{sprites.ActionPCSkill, 5}, {sprites.ActionPCAttack1, 4}, {sprites.ActionPCAttack2, 8}, {sprites.ActionPCAttack3, 7}, {scene.SpriteActionPickup, 1}, {scene.SpriteActionWalk, 5}} {
				if job == db.JobLinker && pose.family == sprites.ActionPCAttack2 {
					continue // Soul Linkers cannot learn Heel Drop.
				}
				for dir := 0; dir < 8; dir++ {
					_, action, ok := sprites.ResolveAction(view.Body.Act, pose.family, dir)
					if !ok || len(action.Animations) <= pose.motion {
						t.Fatalf("job %d sex %d family %d direction %d lacks motion %d", job, sex, pose.family, dir, pose.motion)
					}
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{411, 60, 200}, {430, 40, 90}, {435, 20, 110}, {439, 20, 60}, {444, 10, 30}, {447, 30, 301}, {448, 30, 301}, {461, 20, 110}, {467, 20, 251}, {473, 20, 251}} {
		m := &System{}
		if !m.AddBetween(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("effect %d has no implementation", tc.id)
		}
		spec, _ := Lookup(tc.id)
		for _, component := range spec.Components {
			for _, file := range append([]string{component.TextureFile}, component.TextureFiles...) {
				if file != "" && m.EffectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem()
			count := commands.FieldByName("worldCommands").Len() + commands.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: draws %d", tc.id, tick, count)
			}
		}
	}
	for _, sound := range []string{"T_낙법.wav", "T_피링.wav", "T_바람방출.wav", "T_벽튕김.wav", "T_회피2.wav", "T_돌려차기.wav", "T_내려찍기.wav", "T_날라차기.wav", "T_회오리차기.wav", "T_효과음1.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRobrowserShockwaveSpec(t *testing.T) {
	shockwave, ok := Lookup(Shockwave)
	if !ok || len(shockwave.Components) != 1 {
		t.Fatalf("EF_SHOCKWAVE spec = %+v ok=%t", shockwave, ok)
	}
	if len(shockwave.Sfx) != 0 {
		t.Fatalf("EF_SHOCKWAVE sfx = %v", shockwave.Sfx)
	}
	if component := shockwave.Components[0]; component.Kind != ComponentSTR || component.StrFile != "shockwave" {
		t.Fatalf("EF_SHOCKWAVE component = %+v", component)
	}
}

func TestWaterBallAndSonicBlowSpecs(t *testing.T) {
	water, ok := Lookup(WaterBall)
	if !ok || len(water.Components) != 1 || water.Duration != 500*time.Millisecond {
		t.Fatalf("EF_WATERBALL spec = %+v ok=%t", water, ok)
	}
	component := water.Components[0]
	if component.Kind != Component3D || len(component.TextureFiles) != 3 || component.TextureFiles[0] != "effect/water_out_a.bmp" || component.TextureFiles[2] != "effect/water_out_c.bmp" {
		t.Fatalf("EF_WATERBALL texture files = %+v", component)
	}
	if component.FrameDelay != 10*time.Millisecond || component.Duration != 500*time.Millisecond || !component.FadeOut || component.PosXRand != 1.5 || component.PosZRand != 1.5 || component.PosYEnd != 3 || !component.PosYSmooth || component.SizeStart != testutil.EffectTableSize(30.5) || !component.RotateWithCamera || !component.BlendAdditive || !component.AttachedEntity {
		t.Fatalf("EF_WATERBALL component = %+v", component)
	}

	sonic, ok := Lookup(SonicBlow)
	if !ok || len(sonic.Components) != 1 || sonic.Duration != 40*db.EffectFrameDuration {
		t.Fatalf("EF_SONICBLOW spec = %+v ok=%t", sonic, ok)
	}
	ring := sonic.Components[0]
	if ring.Kind != ComponentFUNC || ring.FuncAdapter != EffectFuncSonicBlow || ring.TextureFile != "effect/alpha_center.tga" || !ring.AttachedEntity {
		t.Fatalf("EF_SONICBLOW ring = %+v", ring)
	}
	spin, ok := Lookup(SonicBlowHit)
	if !ok || len(spin.Components) != 1 || spin.Components[0].Kind != ComponentFUNC || spin.Components[0].FuncAdapter != EffectFuncHitRing || !spin.Components[0].AttachedEntity {
		t.Fatalf("EF_SONICBLOWHIT spec = %+v ok=%t", spin, ok)
	}
}

func TestCrashEarthSpec(t *testing.T) {
	crash, ok := Lookup(CrashEarth)
	if !ok || len(crash.Components) != 1 {
		t.Fatalf("EF_CRASHEARTH spec = %+v ok=%t", crash, ok)
	}
	if crash.CameraShakeDelay != 35*db.EffectFrameDuration || crash.CameraShake != 650*time.Millisecond {
		t.Fatalf("EF_CRASHEARTH camera shake = delay %s duration %s", crash.CameraShakeDelay, crash.CameraShake)
	}
	if component := crash.Components[0]; component.Kind != ComponentSTR || component.StrFile != "crashearth" || component.AttachedEntity {
		t.Fatalf("EF_CRASHEARTH component = %+v", component)
	}

}

func TestRobrowserQuadHornRuntimeDefaults(t *testing.T) {
	if got := QuadHornDefaultOffset(0); got != 0.5 {
		t.Fatalf("quadHornDefaultOffset(0) = %v, want robr default 0.5", got)
	}
	if got := QuadHornDefaultOffset(-0.2); got != -0.2 {
		t.Fatalf("quadHornDefaultOffset(-0.2) = %v, want -0.2", got)
	}
	effect := Instance{EffectID: EarthSpike, ActorID: 1, Starts: time.Unix(10, 20)}
	if got := QuadHornRange(effect, 1, 0, -0.2); got != -0.2 {
		t.Fatalf("quadHornRange max<min = %v, want -0.2", got)
	}
}

func TestQuadHornUVsMatchSourceFaceStrips(t *testing.T) {
	if len(QuadHornUVs) != 12 {
		t.Fatalf("quadHornUVs len = %d, want 12", len(QuadHornUVs))
	}
	for face := 0; face < 4; face++ {
		u0 := float32(face) * 0.2
		uvs := QuadHornUVs[face*3 : face*3+3]
		want := []scene.UV{
			{U: u0, V: 0},
			{U: u0, V: 1},
			{U: u0 + 0.2, V: 1},
		}
		if !reflect.DeepEqual(uvs, want) {
			t.Fatalf("quad horn face %d uvs = %+v, want %+v", face, uvs, want)
		}
	}
}

func TestQuadHornDefaultRotationKeepsApexAboveBase(t *testing.T) {
	effect := Instance{EffectID: IceWall, ActorID: 1, Starts: time.Unix(10, 20)}
	component := Component{}
	salt := 0
	rotateX := QuadHornRange(effect, salt+5, component.QuadHornRotateXMin, component.QuadHornRotateXMax)
	rotateY := QuadHornRange(effect, salt+6, component.QuadHornRotateYMin, component.QuadHornRotateYMax)
	rotateZ := QuadHornRange(effect, salt+7, component.QuadHornRotateZMin, component.QuadHornRotateZMax)
	height := 3.0
	bottomSize := 0.4
	origin := scene.Point3{Y: height * 0.9}
	apex := scene.Add3(origin, RotateEffectCylinderVector(scene.Point3{Y: height}, rotateX, rotateY, rotateZ))
	base := scene.Add3(origin, RotateEffectCylinderVector(scene.Point3{X: -bottomSize, Y: -height, Z: bottomSize}, rotateX, rotateY, rotateZ))

	if apex.Y <= base.Y {
		t.Fatalf("quad horn apex y = %.3f, base y = %.3f, want apex above base", apex.Y, base.Y)
	}
}

func TestRobrowserOldPortalEffectsZeroToFifty(t *testing.T) {
	entry, ok := Lookup(Entry)
	if !ok {
		t.Fatal("EF_ENTRY spec missing")
	}
	if entry.Duration != 500*time.Millisecond || len(entry.Sfx) != 1 || entry.Sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY timing/sfx = duration %s sfx %#v", entry.Duration, entry.Sfx)
	}
	if len(entry.Components) != 3 {
		t.Fatalf("EF_ENTRY components = %d, want 3", len(entry.Components))
	}
	for i, component := range entry.Components {
		if component.Kind != ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 500*time.Millisecond || component.Animation != 1 || !component.Fade || !component.Rotate {
			t.Fatalf("EF_ENTRY component %d = %+v", i, component)
		}
	}
	if entry.Components[0].AttachedEntity || !entry.Components[1].AttachedEntity || !entry.Components[2].AttachedEntity {
		t.Fatalf("EF_ENTRY attachment flags = %t %t %t", entry.Components[0].AttachedEntity, entry.Components[1].AttachedEntity, entry.Components[2].AttachedEntity)
	}
	if entry.Components[0].Height != 7.5 || entry.Components[1].Height != 8 || entry.Components[2].TopSize != 1.5 {
		t.Fatalf("EF_ENTRY dimensions = %+v", entry.Components)
	}

	warp, ok := Lookup(Warp)
	if !ok || len(warp.Components) != 1 {
		t.Fatalf("EF_WARP spec = %+v ok=%t", warp, ok)
	}
	wave := warp.Components[0]
	if wave.Kind != ComponentCylinder || wave.TextureName != "ring_yellow" || wave.Animation != 4 || wave.Duplicate != 4 || wave.DuplicateDelay != 300*time.Millisecond {
		t.Fatalf("EF_WARP wave = %+v", wave)
	}
	if wave.BottomSize != 10 || wave.TopSize != 13 || wave.PosZ != 0.1 || !wave.AttachedEntity {
		t.Fatalf("EF_WARP dimensions = %+v", wave)
	}
	if got := WorldEffectComponentDuration(warp, wave); got != 1900*time.Millisecond {
		t.Fatalf("EF_WARP resolved duration = %s, want 1900ms", got)
	}

	teleport, ok := Lookup(TeleportOld)
	if !ok || len(teleport.Components) != 1 {
		t.Fatalf("EF_TELEPORTATION spec = %+v ok=%t", teleport, ok)
	}
	beam := teleport.Components[0]
	if teleport.Duration != time.Second || len(teleport.Sfx) != 1 || teleport.Sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_TELEPORTATION timing/sfx = %s %#v", teleport.Duration, teleport.Sfx)
	}
	if beam.Kind != ComponentCylinder || beam.TextureName != "ring_blue" || beam.Animation != 5 || beam.Height != 35 || beam.BottomSize != 0.8 || beam.TopSize != 0.7 || !beam.Rotate {
		t.Fatalf("EF_TELEPORTATION beam = %+v", beam)
	}

	ready, ok := Lookup(ReadyPortalOld)
	if !ok || len(ready.Components) != 1 {
		t.Fatalf("EF_READYPORTAL spec = %+v ok=%t", ready, ok)
	}
	portal := ready.Components[0]
	if ready.Duration != 25*time.Second || len(ready.Sfx) != 1 || ready.Sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_READYPORTAL timing/sfx = %s %#v", ready.Duration, ready.Sfx)
	}
	if portal.Kind != ComponentCylinder || portal.TextureName != "alpha_down" || portal.Color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || portal.Height != 15 || portal.AlphaMax != 0.6 {
		t.Fatalf("EF_READYPORTAL cylinder = %+v", portal)
	}
}

func TestRobrowserOldRestoreEffectsZeroToFifty(t *testing.T) {
	exit, ok := Lookup(Exit)
	if !ok {
		t.Fatal("EF_EXIT spec missing")
	}
	if exit.Duration != 2*time.Second || len(exit.Sfx) != 1 || exit.Sfx[0] != "_heal_effect.wav" || len(exit.Components) != 3 {
		t.Fatalf("EF_EXIT spec = %+v", exit)
	}
	if cylinder := exit.Components[0]; cylinder.Kind != ComponentCylinder || cylinder.TextureName != "alpha_down" || cylinder.Duration != 2*time.Second || cylinder.Animation != 1 || cylinder.AlphaMax != 0.2 || !cylinder.BlendAdditive {
		t.Fatalf("EF_EXIT cylinder = %+v", cylinder)
	}
	if particle := exit.Components[1]; particle.Kind != Component3D || particle.TextureFile != "effect/pok3.tga" || particle.Delay != 400*time.Millisecond || particle.Duplicate != 6 || particle.DuplicateDelay != 80*time.Millisecond || !particle.Sparkling {
		t.Fatalf("EF_EXIT first particle = %+v", particle)
	}
	if particle := exit.Components[2]; particle.Duration != 900*time.Millisecond || particle.Delay != 200*time.Millisecond || particle.Duplicate != 3 || particle.DuplicateDelay != 200*time.Millisecond || particle.PosZEnd != 6 {
		t.Fatalf("EF_EXIT second particle = %+v", particle)
	}

	enhance, ok := Lookup(Enhance)
	if !ok || len(enhance.Components) != 3 {
		t.Fatalf("EF_ENHANCE spec = %+v ok=%t", enhance, ok)
	}
	if enhance.Components[0].TextureName != "alpha_down" || enhance.Components[0].BlendAdditive != true || enhance.Components[0].Duration != 2*time.Second {
		t.Fatalf("EF_ENHANCE cylinder = %+v", enhance.Components[0])
	}
	for _, tc := range []struct {
		index     int
		delay     time.Duration
		duplicate int
	}{
		{index: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 2, delay: 400 * time.Millisecond, duplicate: 3},
	} {
		component := enhance.Components[tc.index]
		if component.Kind != Component3D || component.TextureFile != "effect/ac_center2.tga" || component.Delay != tc.delay || component.Duplicate != tc.duplicate || component.DuplicateDelay != 200*time.Millisecond {
			t.Fatalf("EF_ENHANCE particle %d = %+v", tc.index, component)
		}
		if component.SizeStartX != 2.5*scene.PixelRatio || component.SizeRandY != 15*scene.PixelRatio || component.SizeRandYMiddle != 45*scene.PixelRatio {
			t.Fatalf("EF_ENHANCE particle %d size = %+v", tc.index, component)
		}
	}

	healSP, ok := Lookup(HealSP)
	if !ok || len(healSP.Components) != 3 {
		t.Fatalf("EF_HEALSP spec = %+v ok=%t", healSP, ok)
	}
	if len(healSP.Sfx) != 1 || healSP.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("EF_HEALSP sfx = %#v", healSP.Sfx)
	}
	blue := color.RGBA{R: 25, G: 128, B: 255, A: 255}
	if cylinder := healSP.Components[0]; cylinder.TextureName != "ring_blue" || cylinder.Color != blue || !cylinder.Rotate || !cylinder.BlendAdditive {
		t.Fatalf("EF_HEALSP cylinder = %+v", cylinder)
	}
	if healSP.Components[1].Color != blue || healSP.Components[2].Color != blue {
		t.Fatalf("EF_HEALSP particle tints = %+v %+v", healSP.Components[1].Color, healSP.Components[2].Color)
	}
}

func TestRobrowserOldBoltSoundAndStatusEffectsZeroToFifty(t *testing.T) {
	glass, ok := Lookup(GlassWall)
	if !ok || len(glass.Components) != 1 {
		t.Fatalf("EF_GLASSWALL spec = %+v ok=%t", glass, ok)
	}
	if component := glass.Components[0]; component.Kind != ComponentSTR || component.StrFile != "effect/safetywall" || component.AttachedEntity {
		t.Fatalf("EF_GLASSWALL component = %+v", component)
	}
	if len(glass.Sfx) != 1 || glass.Sfx[0] != "effect\\ef_glasswall.wav" {
		t.Fatalf("EF_GLASSWALL sfx = %#v", glass.Sfx)
	}

	ice, ok := Lookup(IceArrow)
	if !ok {
		t.Fatal("EF_ICEARROW spec missing")
	}
	if len(ice.Components) != 1 || ice.Components[0].FuncAdapter != EffectFuncMageBolt || len(ice.Sfx) != 1 || ice.Sfx[0] != "effect\\ef_icearrow%d.wav" || ice.SfxRandMin != 1 || ice.SfxRandMax != 3 {
		t.Fatalf("EF_ICEARROW spec = %+v", ice)
	}

	fire, ok := Lookup(FireArrow)
	if !ok {
		t.Fatal("EF_FIREARROW spec missing")
	}
	if len(fire.Components) != 1 || fire.Components[0].FuncAdapter != EffectFuncMageBolt || len(fire.Sfx) != 1 || fire.Sfx[0] != "effect\\ef_firearrow%d.wav" {
		t.Fatalf("EF_FIREARROW spec = %+v", fire)
	}

	incAgiDex, ok := Lookup(IncAgiDex)
	if !ok || len(incAgiDex.Components) != 1 {
		t.Fatalf("EF_INCAGIDEX spec = %+v ok=%t", incAgiDex, ok)
	}
	if len(incAgiDex.Sfx) != 1 || incAgiDex.Sfx[0] != "effect\\ef_incagidex.wav" {
		t.Fatalf("EF_INCAGIDEX sfx = %#v", incAgiDex.Sfx)
	}
	component := incAgiDex.Components[0]
	if component.FuncAdapter != EffectFuncAgility || component.TextureFile != "effect/dex_agi_up.bmp" || incAgiDex.Duration != 100*db.EffectFrameDuration {
		t.Fatalf("EF_INCAGIDEX must reuse the native agility particles: %+v", component)
	}
}

func TestPneumaEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := Lookup(Pneuma)
	if !ok {
		t.Fatal("Pneuma effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("Pneuma components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentSTR || component.StrFile != "pneuma%d" || component.StrRandMin != 1 || component.StrRandMax != 3 || component.AttachedEntity {
		t.Fatalf("Pneuma component = %+v", component)
	}
}

func TestTorchEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := Lookup(Torch)
	if !ok {
		t.Fatal("torch effect spec missing")
	}
	if spec.Duration != 24*time.Hour {
		t.Fatalf("torch duration = %s", spec.Duration)
	}
	if len(spec.Components) != 1 {
		t.Fatalf("torch components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != Component3D || component.SpriteFile != "torch_01" || !component.SpriteRepeat {
		t.Fatalf("torch component = %+v", component)
	}
	if component.Duration != 600*time.Millisecond || component.SpriteDelay != 100*time.Millisecond {
		t.Fatalf("torch timing = duration %s delay %s", component.Duration, component.SpriteDelay)
	}
	if component.PosX != 0.1 || component.PosZ != 0.8 || component.SizeStart != testutil.EffectTableSize(100) || component.AngleStart != 270 || !component.RotateToTarget {
		t.Fatalf("torch placement = %+v", component)
	}
	ctx := client.Context{World: worldstate.New()}
	angle, ok := EffectSpriteRobrowserRotation(ctx, scene.Projection{}, component, Instance{EffectID: Torch}, 0)
	if !ok || math.Abs(angle+2*math.Pi) > 0.001 {
		t.Fatalf("torch effective angle = %.3f, %v, want -2π radians, true", angle, ok)
	}
}

func TestFireflyEffectSpecUsesFaintSpriteParticles(t *testing.T) {
	spec, ok := Lookup(Firefly)
	if !ok {
		t.Fatal("firefly effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("firefly components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != Component3D || component.TextureFile != "" || component.SpriteFile == "" || !component.SpriteRepeat {
		t.Fatalf("firefly component = %+v", component)
	}
	if component.AlphaMax > 0.25 || component.SizeEnd > testutil.EffectTableSize(120) {
		t.Fatalf("firefly should stay faint and moderately sized: %+v", component)
	}
}

func TestBubbleEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := Lookup(Bubble)
	if !ok {
		t.Fatal("bubble effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("bubble components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentSTR || component.StrFile != "bubble%d" || component.StrRandMin != 1 || component.StrRandMax != 4 {
		t.Fatalf("bubble component = %+v", component)
	}
}

func TestWorldEffectSpecLookupReturnsCopy(t *testing.T) {
	spec, ok := Lookup(BashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	spec.Sfx[0] = "mutated.wav"
	spec.Components[0].TextureFiles[0] = "mutated"

	again, ok := Lookup(BashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing after mutation")
	}
	if again.Sfx[0] != "effect\\ef_bash.wav" || again.Components[0].TextureFiles[0] != "effect/alpha_center.tga" {
		t.Fatalf("catalog mutated: %+v", again)
	}
}

func TestResolveEffectSTRFileUsesDeterministicRandRange(t *testing.T) {
	component := Component{
		StrFile:    "firehit%d",
		StrRandMin: 1,
		StrRandMax: 3,
	}
	effect := Instance{EffectID: FireHit, ActorID: 100, Starts: time.Unix(10, 20)}
	got := ResolveEffectSTRFile(component, effect, false)
	if got != "firehit1" && got != "firehit2" && got != "firehit3" {
		t.Fatalf("resolved STR file = %q, want firehit1..3", got)
	}
	if again := ResolveEffectSTRFile(component, effect, false); again != got {
		t.Fatalf("resolved STR file changed from %q to %q", got, again)
	}
}

func TestResolveEffectSTRFileUsesMinFileForLessEffects(t *testing.T) {
	component := Component{
		StrFile:    "angelus",
		StrMinFile: "jong_mini",
	}
	if got := ResolveEffectSTRFile(component, Instance{}, true); got != "jong_mini" {
		t.Fatalf("resolved STR file = %q, want jong_mini", got)
	}
}

func TestSwordmanSkillEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "SM_BASH begin", SkillBeginEffectIDs(5), BashBegin)
	testutil.ExpectEffectIDs(t, "SM_BASH hit", SkillHitEffectIDs(5), BashHit)
	testutil.ExpectEffectIDs(t, "SM_PROVOKE success", SkillSuccessEffectIDs(6), Provoke)
	testutil.ExpectEffectIDs(t, "SM_MAGNUM damage", SkillEffectIDs(7))
	testutil.ExpectEffectIDs(t, "SM_MAGNUM caster damage", SkillEffectOnCasterIDs(7))
	testutil.ExpectEffectIDs(t, "SM_MAGNUM success", SkillSuccessEffectSelfIDs(7), MagnumBreak, QuakeMagnum)
	testutil.ExpectEffectIDs(t, "SM_ENDURE", SkillSuccessEffectSelfIDs(8), Endure)
}

func TestNoviceSkillEffectMappings(t *testing.T) {
	if action := SkillAction(db.SkillNVBasic); !action.Defined || action.Action != SkillActorActionNone {
		t.Fatalf("basic skill action = %+v, want no source action", action)
	}
	testutil.ExpectEffectIDs(t, "NV_FIRSTAID", SkillSuccessEffectSelfIDs(142), FirstAid)
	testutil.ExpectEffectIDs(t, "NV_TRICKDEAD", SkillEffectIDs(143))
}

func TestMageSkillEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "MG_SIGHT success", SkillSuccessEffectIDs(10))
	testutil.ExpectEffectIDs(t, "MG_SIGHT immediate", SkillEffectIDs(10))
	testutil.ExpectEffectIDs(t, "MG_NAPALMBEAT hit", SkillHitEffectIDs(11), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "MG_SAFETYWALL ground", SkillGroundEffectIDs(12))
	testutil.ExpectEffectIDs(t, "MG_SOULSTRIKE before-hit", SkillBeforeHitEffectIDs(13), SoulStrike)
	testutil.ExpectEffectIDs(t, "MG_SOULSTRIKE hit", SkillHitEffectIDs(13), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "MG_COLDBOLT before-hit", SkillBeforeHitEffectIDs(14), ColdBolt)
	testutil.ExpectEffectIDs(t, "MG_COLDBOLT hit", SkillHitEffectIDs(14), ColdHit)
	testutil.ExpectEffectIDs(t, "MG_FROSTDIVER", SkillEffectIDs(15))
	testutil.ExpectEffectIDs(t, "MG_FROSTDIVER before-hit", SkillBeforeHitEffectIDs(15), FrostDiver)
	testutil.ExpectEffectIDs(t, "MG_FROSTDIVER hit", SkillHitEffectIDs(15), ColdHit, FrostDiverHit)
	testutil.ExpectEffectIDs(t, "MG_STONECURSE", SkillSuccessEffectIDs(16), StoneCurse)
	testutil.ExpectEffectIDs(t, "MG_FIREBOLT before-hit", SkillBeforeHitEffectIDs(19), FireBolt)
	testutil.ExpectEffectIDs(t, "MG_FIREBALL before-hit", SkillBeforeHitEffectIDs(17), FireBall)
	testutil.ExpectEffectIDs(t, "MG_FIREWALL ground", SkillGroundEffectIDs(18))
	testutil.ExpectEffectIDs(t, "MG_FIREBALL hit", SkillHitEffectIDs(17), FireHit, BashHit)
	testutil.ExpectEffectIDs(t, "MG_FIREWALL hit", SkillHitEffectIDs(18), Hit1)
	testutil.ExpectEffectIDs(t, "MG_FIREBOLT hit", SkillHitEffectIDs(19), FireHit)
	testutil.ExpectEffectIDs(t, "MG_LIGHTNINGBOLT hit", SkillHitEffectIDs(20), WindHit)
	testutil.ExpectEffectIDs(t, "MG_THUNDERSTORM hit", SkillHitEffectIDs(21), Hit1)
	testutil.ExpectEffectIDs(t, "MG_LIGHTNINGBOLT before-hit", SkillBeforeHitEffectIDs(20))
	testutil.ExpectEffectIDs(t, "MG_LIGHTNINGBOLT", SkillEffectIDs(20), LightningBolt)
	testutil.ExpectEffectIDs(t, "MG_THUNDERSTORM", SkillEffectIDs(21), ThunderStorm)
	testutil.ExpectEffectIDs(t, "MG_THUNDERSTORM ground", SkillGroundEffectIDs(21))
	testutil.ExpectEffectIDs(t, "MG_ENERGYCOAT", SkillSuccessEffectSelfIDs(157), EnergyCoat)
	testutil.ExpectEffectIDs(t, "MG_THUNDERSTORM before-hit", SkillBeforeHitEffectIDs(21))
	for _, skillID := range []uint16{20, 21} {
		testutil.ExpectEffectIDs(t, "wind skill begin", SkillBeginEffectIDs(skillID))
	}
}

func TestFireHitEffectSpec(t *testing.T) {
	hitSpec, ok := Lookup(FireHit)
	if !ok || len(hitSpec.Components) != 1 {
		t.Fatalf("fire hit effect missing or wrong component count: ok=%t components=%d", ok, len(hitSpec.Components))
	}
	hit := hitSpec.Components[0]
	if hit.Kind != ComponentSTR || hit.StrFile != "firehit%d" || hit.StrRandMin != 1 || hit.StrRandMax != 3 || !hit.AttachedEntity {
		t.Fatalf("fire hit STR = %+v", hit)
	}
	if len(hitSpec.Sfx) != 1 || hitSpec.Sfx[0] != "effect\\ef_firehit.wav" {
		t.Fatalf("fire hit sfx = %#v", hitSpec.Sfx)
	}
}

func TestRepeatedWorldEffectDuplicateProgressUsesDuplicateDelay(t *testing.T) {
	component := Component{
		Duration:       4 * time.Second,
		Delay:          200 * time.Millisecond,
		Repeat:         true,
		Duplicate:      4,
		DuplicateDelay: time.Second,
	}
	starts := time.Unix(1000, 0)
	now := starts.Add(2700 * time.Millisecond)
	for _, tc := range []struct {
		duplicate int
		want      float64
	}{
		{0, 0.625},
		{1, 0.375},
		{2, 0.125},
	} {
		got, active := WorldEffectComponentDuplicateProgressForDraw(starts, component, tc.duplicate, component.Duration, now)
		if !active {
			t.Fatalf("duplicate %d inactive, want progress %.3f", tc.duplicate, tc.want)
		}
		if math.Abs(got-tc.want) > 0.000001 {
			t.Fatalf("duplicate %d progress = %.6f, want %.6f", tc.duplicate, got, tc.want)
		}
	}
	if got, active := WorldEffectComponentDuplicateProgressForDraw(starts, component, 3, component.Duration, now); active {
		t.Fatalf("duplicate 3 active with progress %.6f, want delayed until its own start", got)
	}
}

func TestCylinderAnimationThreeScalesRadiusAndPulsesHeightLikeRobrowser(t *testing.T) {
	component := Component{
		Animation:  3,
		BottomSize: 2,
		TopSize:    4,
		Height:     6,
	}
	for _, tc := range []struct {
		progress   float64
		wantBottom float64
		wantTop    float64
		wantHeight float64
	}{
		{0, 2, 4, 0},
		{0.25, 1.5, 3, 3},
		{0.5, 1, 2, 6},
		{0.75, 0.5, 1, 3},
	} {
		gotBottom, gotTop, gotHeight := EffectCylinderAnimatedDimensions(component, tc.progress)
		if math.Abs(gotBottom-tc.wantBottom) > 0.000001 || math.Abs(gotTop-tc.wantTop) > 0.000001 || math.Abs(gotHeight-tc.wantHeight) > 0.000001 {
			t.Fatalf("progress %.2f dimensions = %.6f %.6f %.6f, want %.6f %.6f %.6f", tc.progress, gotBottom, gotTop, gotHeight, tc.wantBottom, tc.wantTop, tc.wantHeight)
		}
	}
}

func TestRegularHitEffectKeepsRobrowserParticleBurst(t *testing.T) {
	spec, ok := Lookup(Hit1)
	if !ok || len(spec.Components) != 2 {
		t.Fatalf("regular hit spec = %+v ok=%t, want sparks and a ring", spec, ok)
	}
	if spec.Duration != 300*time.Millisecond {
		t.Fatalf("duration = %s, want 300ms", spec.Duration)
	}
	component := spec.Components[0]
	if component.Kind != Component3D || component.TextureFile != "effect/pok3.tga" || component.Duration != 300*time.Millisecond {
		t.Fatalf("component resource/timing = %+v", component)
	}
	if component.Duplicate != 4 || component.AlphaMax != 0.8 || !component.FadeIn || !component.FadeOut || !component.Sparkling {
		t.Fatalf("component duplicate/fade = %+v", component)
	}
	if component.PosZ != 1 || component.PosXEndRand != 2 || component.PosYEndRand != 2 || component.PosZEndRand != 2 {
		t.Fatalf("component position = %+v", component)
	}
	if component.SizeStart != testutil.EffectTableSize(10) || component.SizeEnd != testutil.EffectTableSize(10) || component.SizeRand != testutil.EffectTableSize(20) || !component.SizeSmooth {
		t.Fatalf("component size = %+v", component)
	}
}

func TestSkillHitSlashSpecsMatchRobrowser(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		kind     ComponentKind
		width    float64
		height   float64
		sfx      string
		overlay  bool
	}{
		{"hit5", Hit5, Component3D, testutil.EffectTableSize(15), testutil.EffectTableSize(200), "effect\\ef_hit5.wav", false},
		{"hit6", Hit6, Component2D, testutil.EffectTableSize(10), testutil.EffectTableSize(150), "effect\\ef_hit6.wav", true},
	} {
		spec, ok := Lookup(tc.effectID)
		if !ok || len(spec.Components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two slash components", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.sfx {
			t.Fatalf("%s sfx = %v", tc.name, spec.Sfx)
		}
		for i, component := range spec.Components {
			if component.Kind != tc.kind || component.TextureFile != "effect/lens2.tga" || component.Duration != 400*time.Millisecond || component.AlphaMax != 1 || !component.FadeOut || !component.Rotate || component.Overlay != tc.overlay {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.PosZ != 1 || component.SizeStartX != tc.width || component.SizeEndX != tc.width || component.SizeStartY != testutil.EffectTableSize(10) || component.SizeEndY != tc.height {
				t.Fatalf("%s component %d size/position = %+v", tc.name, i, component)
			}
		}
		if spec.Components[0].AngleStart != 90 || spec.Components[0].AngleEnd != 0 || spec.Components[1].AngleStart != 180 || spec.Components[1].AngleEnd != 90 {
			t.Fatalf("%s slash angles = %+v %+v", tc.name, spec.Components[0], spec.Components[1])
		}
	}
}

func TestSpriteRotationUsesRobrowserWorldTrajectory(t *testing.T) {
	component := Component{ToSrc: true, RotateToTarget: true, RotateWithCamera: true}
	tests := []struct {
		name      string
		sourceX   int
		sourceY   int
		targetX   int
		targetY   int
		cameraYaw float64
		want      float64
	}{
		{name: "same row", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, want: -math.Pi / 2},
		{name: "same column", sourceX: 12, sourceY: 18, targetX: 12, targetY: 20, want: 0},
		{name: "diagonal", sourceX: 10, sourceY: 18, targetX: 12, targetY: 20, want: -math.Pi / 4},
		{name: "camera yaw", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, cameraYaw: 45, want: -3 * math.Pi / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: tt.sourceX, Y: tt.sourceY}
			world.UpsertActor(worldstate.Actor{ID: 300, X: tt.targetX, Y: tt.targetY})
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}
			effect := Instance{EffectID: FireBall, ActorID: 300, TargetID: 2000000}
			startX, startY, _, endX, endY, _, ok := EffectTrajectoryEndpoints(ctx, component, effect)
			if !ok {
				t.Fatal("trajectory endpoints missing")
			}
			if math.Hypot(endX-startX, endY-startY) <= 0.001 {
				t.Fatalf("trajectory did not span caster and target: %.2f,%.2f -> %.2f,%.2f", startX, startY, endX, endY)
			}
			projection := testutil.ProjectionYaw(800, 600, float64(tt.targetX), float64(tt.targetY), 0, tt.cameraYaw)
			angle, ok := EffectSpriteRobrowserRotation(ctx, projection, component, effect, 0)
			if !ok {
				t.Fatal("rotation missing")
			}
			wantFromFormula := -(90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi + tt.cameraYaw) * math.Pi / 180
			if math.Abs(angle-wantFromFormula) > 0.001 {
				t.Fatalf("angle = %.3f, want robr formula %.3f", angle, wantFromFormula)
			}
			if math.Abs(angle-tt.want) > 0.001 {
				t.Fatalf("angle = %.3f, want %.3f", angle, tt.want)
			}
		})
	}
}

func TestGroundSampleEffectSpecUsesMagicTargetPlane(t *testing.T) {
	spec, ok := Lookup(GroundSample)
	if !ok {
		t.Fatal("ground sample effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentFUNC || component.FuncAdapter != EffectFuncGroundSample || component.FuncName != "MagicTarget" || component.TextureFile != "effect/magic_target.tga" || component.SizeStart != 1 {
		t.Fatalf("component = %+v", component)
	}
	if component.Duration != 0 {
		t.Fatalf("ground sample component duration = %s, want inherited cast duration", component.Duration)
	}
}

func TestGroundSampleRotationAngleUsesClientSpeedOverride(t *testing.T) {
	start := time.Unix(10, 0)
	now := start.Add(time.Second)
	speed := 0.5 / 7 * 60
	effect := Instance{Starts: start, GroundSampleRotationRadiansPerSecond: speed}
	if got := GroundSampleRotationAngle(effect, now); math.Abs(got-speed) > 0.001 {
		t.Fatalf("override angle = %.4f, want %.4f", got, speed)
	}
	fallback := Instance{Starts: start}
	wantFallback := 40 * math.Pi / 180
	if got := GroundSampleRotationAngle(fallback, now); math.Abs(got-wantFallback) > 0.001 {
		t.Fatalf("fallback angle = %.4f, want %.4f", got, wantFallback)
	}
}

func TestGroundSampleDrawOptionsClampRotatedMagicTargetTexture(t *testing.T) {
	options := GroundSampleDrawOptions()
	if options.Filter != render.FilterLinear {
		t.Fatalf("filter = %v, want linear", options.Filter)
	}
	if options.Address != render.AddressClampToZero {
		t.Fatalf("address = %v, want clamp-to-zero to avoid repeated rotated MagicTarget corners", options.Address)
	}
	if !options.DepthTest {
		t.Fatal("ground sample depth test is disabled")
	}
}

func TestLockOnTargetEffectSpecMatchesRobrowserCastTargetCircle(t *testing.T) {
	spec, ok := Lookup(LockOnTarget)
	if !ok {
		t.Fatal("lock-on target effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentFUNC || component.FuncAdapter != EffectFuncLockOnTarget || component.FuncName != "LockOnTarget" || component.TextureFile != "effect/lockon128.tga" || !component.AttachedEntity {
		t.Fatalf("component = %+v", component)
	}
	start := time.Unix(10, 0)
	if got := LockOnTargetSize(start, start); got != 15 {
		t.Fatalf("initial lock-on size = %.1f, want 15", got)
	}
	if got := LockOnTargetSize(start, start.Add(250*time.Millisecond)); got != 3 {
		t.Fatalf("settled lock-on size = %.1f, want 3", got)
	}
	if got := LockOnTargetTint(start, start); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("initial lock-on tint = %+v", got)
	}
	if got := LockOnTargetTint(start, start.Add(380*time.Millisecond)); got != (color.RGBA{R: 255, G: 12, B: 12, A: 255}) {
		t.Fatalf("low lock-on tint = %+v", got)
	}
}

func TestLightningBoltHasNoExtraImpactFlash(t *testing.T) {
	spec, ok := Lookup(LightningBolt)
	if !ok || len(spec.Components) != 1 || spec.Components[0].StrFile != "lightning" {
		t.Fatal("lightning must play only its STR; impact flashes are scheduled per damage hit")
	}
}

func TestSkillCastAuraEffectMappings(t *testing.T) {
	tests := []struct {
		property uint32
		want     int
	}{
		{property: 0, want: BeginSpell},
		{property: 1, want: BeginSpell2},
		{property: 2, want: BeginSpell4},
		{property: 3, want: BeginSpell3},
		{property: 4, want: BeginSpell5},
		{property: 5, want: BeginSpell7},
		{property: 6, want: BeginSpell6},
		{property: 7, want: BeginSpell},
		{property: 8, want: BeginSpell},
		{property: 9, want: BeginSpell},
	}
	for _, tt := range tests {
		if got := skillCastAuraEffectID(tt.property); got != tt.want {
			t.Fatalf("cast aura property %d = %d, want %d", tt.property, got, tt.want)
		}
	}
}

func TestSkillVisualMetadataMappings(t *testing.T) {
	if SkillAction(5).Action != SkillActorActionAttack || SkillAction(7).Action != SkillActorActionAttack {
		t.Fatalf("swordman weapon-action skills = bash:%d magnum:%d", SkillAction(5).Action, SkillAction(7).Action)
	}
	if SkillAction(8).Action != SkillActorActionReadyFight {
		t.Fatalf("endure action = %d, want ready fight", SkillAction(8).Action)
	}
	if SkillAction(28).Action != SkillActorActionSkill {
		t.Fatalf("heal action = %d, want default skill action", SkillAction(28).Action)
	}
	defaultAction := SkillAction(28)
	if !defaultAction.Play || defaultAction.Repeat || defaultAction.Next == nil || defaultAction.Next.Action != SkillActorActionIdle || !defaultAction.Next.Repeat {
		t.Fatalf("default skill action shape = %+v next=%+v, want robr-style skill action followed by repeating idle", defaultAction, defaultAction.Next)
	}
	if size := skillCastGroundSampleSize(19); size != 1 {
		t.Fatalf("firebolt marker size = %.1f, want default 1", size)
	}
	for skillID, wantSize := range map[uint16]float64{
		db.SkillACShower:        5,
		db.SkillALPneuma:        5,
		db.SkillASGrimtooth:     5,
		db.SkillASVenomdust:     5,
		db.SkillBSHammerfall:    5,
		db.SkillCRSlimpitcher:   9,
		db.SkillHTBlitzbeat:     5,
		db.SkillHTDetecting:     5,
		db.SkillHWGanbantein:    5,
		db.SkillHWGravitation:   7,
		db.SkillMGFireball:      7,
		db.SkillMGFirewall:      2,
		db.SkillMGNapalmbeat:    5,
		db.SkillMGThunderstorm:  7,
		db.SkillPRBenedictio:    5,
		db.SkillPRMagnus:        9,
		db.SkillPRSanctuary:     7,
		db.SkillSNFalconassault: 5,
		db.SkillWZFirepillar:    5,
		db.SkillWZFrostnova:     5,
		db.SkillWZHeavendrive:   7,
		db.SkillWZMeteor:        11,
		db.SkillWZQuagmire:      7,
		db.SkillWZStormgust:     11,
		db.SkillWZVermilion:     13,
		db.SkillWZWaterball:     5,
	} {
		if size := skillCastGroundSampleSize(skillID); size != wantSize {
			t.Fatalf("skill %d marker size = %.1f, want classic client rendered scope %.1f", skillID, size, wantSize)
		}
		gotSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		wantSpeed := 0.5 / wantSize * 60
		if math.Abs(gotSpeed-wantSpeed) > 0.001 {
			t.Fatalf("skill %d marker speed = %.4f rad/s, want %.4f", skillID, gotSpeed, wantSpeed)
		}
	}
	if speed := skillCastGroundSampleRotationRadiansPerSecond(19); speed != 0 {
		t.Fatalf("default marker speed override = %.4f, want fallback path", speed)
	}
}

func TestSelfTargetSkillCastDoesNotAddLockOnTarget(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &System{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.AddSkillCastEffects(ctx, db.SkillMGFireball, 3, 2000000, 2000000, 0, 0, 900*time.Millisecond, time.Now(), "self")

	if len(mode.Active) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and aura", mode.Active)
	}
	if mode.Active[0].EffectID != CastRing || mode.Active[1].EffectID != BeginSpell3 {
		t.Fatalf("effects = %+v", mode.Active)
	}
}

func TestSkillCastEffectsDedupeServerAndLocalFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &System{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.AddSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start, "local")
	mode.AddSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start.Add(20*time.Millisecond), "server")

	if len(mode.Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.Active))
	}
	if mode.Active[0].EffectID != CastRing || mode.Active[1].EffectID != BeginSpell3 {
		t.Fatalf("effects = %+v", mode.Active)
	}
}

func TestGroundSkillCastEffectsAddGroundSampleMarker(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &System{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.AddSkillCastEffects(ctx, 21, 4, 2000000, 0, 123, 456, 1800*time.Millisecond, start, "local-ground")

	if len(mode.Active) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.Active))
	}
	marker := mode.Active[0]
	wantSpeed := 0.5 / 7 * 60
	if marker.EffectID != GroundSample || marker.ActorID != 0 || marker.X != 123 || marker.Y != 456 || marker.Duration != 1800*time.Millisecond || marker.Size != 7 || math.Abs(marker.GroundSampleRotationRadiansPerSecond-wantSpeed) > 0.001 {
		t.Fatalf("ground marker = %+v", marker)
	}
	if mode.Active[1].EffectID != CastRing || mode.Active[2].EffectID != BeginSpell5 {
		t.Fatalf("cast effects = %+v", mode.Active)
	}
}

func TestAcolyteSkillEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "AL_DP passive", SkillEffectIDs(22))
	testutil.ExpectEffectIDs(t, "AL_DEMONBANE passive", SkillEffectIDs(23))
	testutil.ExpectEffectIDs(t, "AL_RUWACH hit", SkillHitEffectIDs(24), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "AL_PNEUMA uses skill unit", SkillGroundEffectIDs(25))
	testutil.ExpectEffectIDs(t, "AL_TELEPORT", SkillEffectIDs(26))
	testutil.ExpectEffectIDs(t, "AL_WARP", SkillEffectIDs(27))
	testutil.ExpectEffectIDs(t, "AL_HEAL offensive", SkillEffectIDs(28), HealOffensive)
	testutil.ExpectEffectIDs(t, "AL_HEAL hit", SkillHitEffectIDs(28), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "AL_INCAGI", SkillSuccessEffectIDs(29), IncAgility)
	testutil.ExpectEffectIDs(t, "AL_DECAGI", SkillSuccessEffectIDs(30), DecAgility)
	testutil.ExpectEffectIDs(t, "AL_HOLYWATER", SkillSuccessEffectSelfIDs(31), Aqua)
	testutil.ExpectEffectIDs(t, "AL_CRUCIS", SkillSuccessEffectSelfIDs(32), Signum)
	testutil.ExpectEffectIDs(t, "AL_ANGELUS", SkillSuccessEffectSelfIDs(33), Angelus)
	testutil.ExpectEffectIDs(t, "AL_BLESSING", SkillSuccessEffectIDs(34), Blessing)
	testutil.ExpectEffectIDs(t, "AL_CURE", SkillSuccessEffectIDs(35), Cure)
}

func TestMercenarySupportSkillEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "MER_MAGNIFICAT", SkillEffectIDs(db.SkillMerMagnificat), Magnificat)
	testutil.ExpectEffectIDs(t, "MER_QUICKEN", SkillEffectIDs(db.SkillMerQuicken), TwoHandQuicken)
	testutil.ExpectEffectIDs(t, "MER_PROVOKE success", SkillSuccessEffectIDs(db.SkillMerProvoke), Provoke)
	testutil.ExpectEffectIDs(t, "MER_DECAGI", SkillEffectIDs(db.SkillMerDecagi), DecAgility)
	testutil.ExpectEffectIDs(t, "MER_LEXDIVINA", SkillEffectIDs(db.SkillMerLexdivina), LexDivina)
	testutil.ExpectEffectIDs(t, "MER_ESTIMATION", SkillEffectIDs(db.SkillMerEstimation))
	testutil.ExpectEffectIDs(t, "MER_KYRIE", SkillEffectIDs(db.SkillMerKyrie), Kyrie)
	testutil.ExpectEffectIDs(t, "MER_BLESSING", SkillEffectIDs(db.SkillMerBlessing), Blessing)
	testutil.ExpectEffectIDs(t, "MER_INCAGI", SkillEffectIDs(db.SkillMerIncagi), IncAgility)

	for _, tc := range []struct {
		name    string
		skillID uint16
	}{
		{"MER_SIGHT", db.SkillMerSight},
		{"MER_CRASH", db.SkillMerCrash},
		{"MER_REGAIN", db.SkillMerRegain},
		{"MER_TENDER", db.SkillMerTender},
		{"MER_BENEDICTION", db.SkillMerBenediction},
		{"MER_RECUPERATE", db.SkillMerRecuperate},
		{"MER_MENTALCURE", db.SkillMerMentalcure},
		{"MER_COMPRESS", db.SkillMerCompress},
		{"MER_AUTOBERSERK", db.SkillMerAutoberserk},
		{"MER_SCAPEGOAT", db.SkillMerScapegoat},
	} {
		testutil.ExpectEffectIDs(t, tc.name, SkillEffectIDs(tc.skillID))
		testutil.ExpectEffectIDs(t, tc.name+" success", SkillSuccessEffectIDs(tc.skillID))
	}
}

func TestImportedSkillEffectFallback(t *testing.T) {
	testutil.ExpectEffectIDs(t, "PR_IMPOSITIO imported", SkillEffectIDs(db.SkillPRImpositio), Impositio)
	testutil.ExpectEffectIDs(t, "ALL_RESURRECTION success", SkillSuccessEffectIDs(db.SkillALLResurrection), Resurrection, Revive)
	testutil.ExpectEffectIDs(t, "PR_SUFFRAGIUM imported", SkillEffectIDs(db.SkillPRSuffragium), Suffragium)
	testutil.ExpectEffectIDs(t, "PR_ASPERSIO success", SkillSuccessEffectIDs(db.SkillPRAspersio), Aspersio)
	testutil.ExpectEffectIDs(t, "PR_BENEDICTIO ground", SkillGroundEffectIDs(db.SkillPRBenedictio), Benedictio)
	testutil.ExpectEffectIDs(t, "PR_SANCTUARY healing has no STR", SkillEffectIDs(db.SkillPRSanctuary))
	testutil.ExpectEffectIDs(t, "PR_KYRIE success", SkillSuccessEffectSelfIDs(db.SkillPRKyrie), Kyrie)
	testutil.ExpectEffectIDs(t, "PR_MAGNIFICAT success", SkillSuccessEffectSelfIDs(db.SkillPRMagnificat), Magnificat)
	testutil.ExpectEffectIDs(t, "PR_GLORIA success", SkillSuccessEffectSelfIDs(db.SkillPRGloria), Gloria)
	testutil.ExpectEffectIDs(t, "PR_LEXDIVINA imported", SkillEffectIDs(db.SkillPRLexdivina), LexDivina)
	testutil.ExpectEffectIDs(t, "PR_LEXAETERNA imported", SkillEffectIDs(db.SkillPRLexaeterna), LexAeterna)
	testutil.ExpectEffectIDs(t, "PR_TURNUNDEAD hit", SkillHitEffectIDs(db.SkillPRTurnundead), Hit1, HolyLight)
	testutil.ExpectEffectIDs(t, "PR_MAGNUS damage has no STR", SkillEffectIDs(db.SkillPRMagnus))
	testutil.ExpectEffectIDs(t, "PR_MAGNUS ground", SkillGroundEffectIDs(db.SkillPRMagnus), Magnus)
	testutil.ExpectEffectIDs(t, "PR_SANCTUARY imported ground", SkillGroundEffectIDs(db.SkillPRSanctuary), BottomSanc)
	testutil.ExpectEffectIDs(t, "PR_SLOWPOISON imported", SkillEffectIDs(db.SkillPRSlowpoison), SlowPoison)
	testutil.ExpectEffectIDs(t, "PR_STRECOVERY imported", SkillEffectIDs(db.SkillPRStrecovery), Recovery)
	testutil.ExpectEffectIDs(t, "PR_REDEMPTIO imported empty", SkillEffectIDs(db.SkillPRRedemptio))
	testutil.ExpectEffectIDs(t, "MC_IDENTIFY imported empty", SkillEffectIDs(db.SkillMCIdentify))
	testutil.ExpectEffectIDs(t, "MC_VENDING imported empty", SkillEffectIDs(db.SkillMCVending))
	testutil.ExpectEffectIDs(t, "MC_CHANGECART imported empty", SkillEffectIDs(db.SkillMCChangecart))
	testutil.ExpectEffectIDs(t, "MC_CARTDECORATE imported empty", SkillEffectIDs(db.SkillMCCartdecorate))
	testutil.ExpectEffectIDs(t, "KN_PIERCE begin", SkillBeginEffectIDs(db.SkillKNPierce), PierceSelf)
	testutil.ExpectEffectIDs(t, "KN_PIERCE imported hit", SkillHitEffectIDs(db.SkillKNPierce), Pierce)
	testutil.ExpectEffectIDs(t, "KN_BRANDISHSPEAR imported", SkillEffectIDs(db.SkillKNBrandishspear), BrandishSpear)
	testutil.ExpectEffectIDs(t, "KN_BRANDISHSPEAR imported caster", SkillBeginEffectIDs(db.SkillKNBrandishspear), BrandishSpear2)
	testutil.ExpectEffectIDs(t, "KN_SPEARSTAB imported caster", SkillBeginEffectIDs(db.SkillKNSpearstab), SpearStabSelf)
	testutil.ExpectEffectIDs(t, "KN_SPEARBOOMERANG imported caster", SkillBeginEffectIDs(db.SkillKNSpearboomerang), SpearBmrSelf)
	testutil.ExpectEffectIDs(t, "KN_SPEARBOOMERANG imported before hit", SkillBeforeHitEffectIDs(db.SkillKNSpearboomerang), SpearBoomerang)
	testutil.ExpectEffectIDs(t, "KN_SPEARBOOMERANG imported hit", SkillHitEffectIDs(db.SkillKNSpearboomerang), Hit4)
	testutil.ExpectEffectIDs(t, "KN_TWOHANDQUICKEN imported", SkillSuccessEffectSelfIDs(db.SkillKNTwohandquicken), TwoHandQuicken)
	testutil.ExpectEffectIDs(t, "KN_ONEHAND imported", SkillSuccessEffectSelfIDs(db.SkillKNOnehand), TwoHandQuicken)
	testutil.ExpectEffectIDs(t, "KN_CHARGEATK imported begin", SkillBeginEffectIDs(db.SkillKNChargeatk))
	testutil.ExpectEffectIDs(t, "KN_CHARGEATK imported hit", SkillHitEffectIDs(db.SkillKNChargeatk), BashHit)
	testutil.ExpectEffectIDs(t, "KN_BOWLINGBASH imported caster", SkillBeginEffectIDs(db.SkillKNBowlingbash), BowlingSelf)
	testutil.ExpectEffectIDs(t, "HT_SKIDTRAP imported", SkillGroundEffectIDs(db.SkillHTSkidtrap), SkidTrap)
	testutil.ExpectEffectIDs(t, "HT_LANDMINE imported empty", SkillEffectIDs(db.SkillHTLandmine))
	testutil.ExpectEffectIDs(t, "HT_ANKLESNARE imported ground", SkillGroundEffectIDs(db.SkillHTAnklesnare))
	testutil.ExpectEffectIDs(t, "HT_SHOCKWAVE imported", SkillEffectIDs(db.SkillHTShockwave))
	testutil.ExpectEffectIDs(t, "HT_SHOCKWAVE imported hit", SkillHitEffectIDs(db.SkillHTShockwave))
	testutil.ExpectEffectIDs(t, "HT_SANDMAN imported hit", SkillHitEffectIDs(db.SkillHTSandman))
	testutil.ExpectEffectIDs(t, "HT_FLASHER imported hit", SkillHitEffectIDs(db.SkillHTFlasher))
	testutil.ExpectEffectIDs(t, "HT_FREEZINGTRAP imported hit", SkillHitEffectIDs(db.SkillHTFreezingtrap), Hit1)
	testutil.ExpectEffectIDs(t, "HT_BLASTMINE imported hit", SkillHitEffectIDs(db.SkillHTBlastmine), Hit1)
	testutil.ExpectEffectIDs(t, "HT_CLAYMORE imported hit", SkillHitEffectIDs(db.SkillHTClaymoretrap), Hit1)
	testutil.ExpectEffectIDs(t, "HT_REMOVETRAP imported", SkillEffectIDs(db.SkillHTRemovetrap), RemoveTrap)
	testutil.ExpectEffectIDs(t, "HT_BLITZBEAT imported", SkillHitEffectIDs(db.SkillHTBlitzbeat), BlitzBeat)
	testutil.ExpectEffectIDs(t, "HT_DETECTING imported", SkillGroundEffectIDs(db.SkillHTDetecting), Detecting)
	testutil.ExpectEffectIDs(t, "HT_SPRINGTRAP imported", SkillEffectIDs(db.SkillHTSpringtrap), SpringTrap)
	testutil.ExpectEffectIDs(t, "HT_TALKIEBOX imported empty", SkillEffectIDs(db.SkillHTTalkiebox))
	testutil.ExpectEffectIDs(t, "HT_POWER imported empty", SkillEffectIDs(db.SkillHTPower))
	testutil.ExpectEffectIDs(t, "AS_CLOAKING imported", SkillEffectIDs(db.SkillASCloaking), Cloaking)
	testutil.ExpectEffectIDs(t, "AS_SONICBLOW imported", SkillEffectIDs(db.SkillASSonicblow))
	testutil.ExpectEffectIDs(t, "AS_SONICBLOW imported caster", SkillEffectOnCasterIDs(db.SkillASSonicblow), SonicBlow, SonicBlow2)
	testutil.ExpectEffectIDs(t, "AS_SONICBLOW imported hit", SkillHitEffectIDs(db.SkillASSonicblow), SonicBlowHit)
	testutil.ExpectEffectIDs(t, "AS_GRIMTOOTH imported", SkillEffectIDs(db.SkillASGrimtooth), Grimtooth)
	testutil.ExpectEffectIDs(t, "AS_GRIMTOOTH imported hit", SkillHitEffectIDs(db.SkillASGrimtooth), GrimtoothAtk)
	testutil.ExpectEffectIDs(t, "AS_POISONREACT imported", SkillSuccessEffectIDs(db.SkillASPoisonreact), PoisonReact)
	testutil.ExpectEffectIDs(t, "AS_POISONREACT imported hit", SkillHitEffectIDs(db.SkillASPoisonreact))
	testutil.ExpectEffectIDs(t, "AS_VENOMDUST imported", SkillEffectIDs(db.SkillASVenomdust), VenomDust)
	testutil.ExpectEffectIDs(t, "AS_VENOMDUST imported ground", SkillGroundEffectIDs(db.SkillASVenomdust), VenomDust2)
	testutil.ExpectEffectIDs(t, "AS_SPLASHER imported", SkillSuccessEffectIDs(db.SkillASSplasher), EnchantPoison)
	testutil.ExpectEffectIDs(t, "NPC_DARKCROSS imported", SkillEffectIDs(db.SkillNPCDarkcross), DarkGrandCross)
	testutil.ExpectEffectIDs(t, "NPC_DARKSTRIKE imported", SkillEffectIDs(db.SkillNPCDarkstrike), DarkSoulStrike)
	testutil.ExpectEffectIDs(t, "NPC_STOP imported", SkillEffectIDs(db.SkillNPCStop), NPCStop)
	testutil.ExpectEffectIDs(t, "NPC_POWERUP imported", SkillEffectIDs(db.SkillNPCPowerup))
	testutil.ExpectEffectIDs(t, "NPC_DARKBREATH imported", SkillEffectIDs(db.SkillNPCDarkbreath), DarkBreath)
	testutil.ExpectEffectIDs(t, "NPC_DEFENDER imported", SkillEffectIDs(db.SkillNPCDefender), Defender)
	testutil.ExpectEffectIDs(t, "NPC_KEEPING imported", SkillEffectIDs(db.SkillNPCKeeping), Keeping)
	testutil.ExpectEffectIDs(t, "NPC_BLOODDRAIN imported caster", SkillEffectOnCasterIDs(db.SkillNPCBlooddrain), BloodDrain)
	testutil.ExpectEffectIDs(t, "NPC_ENERGYDRAIN imported caster", SkillEffectOnCasterIDs(db.SkillNPCEnergydrain), EnergyDrain)
	testutil.ExpectEffectIDs(t, "NPC_EARTHQUAKE imported caster", SkillEffectOnCasterIDs(db.SkillNPCEarthquake), NPCEarthquake)
	testutil.ExpectEffectIDs(t, "NPC_DRAGONFEAR imported", SkillEffectIDs(db.SkillNPCDragonfear), DragonFear)
	testutil.ExpectEffectIDs(t, "NPC_WIDEBLEEDING imported caster", SkillEffectOnCasterIDs(db.SkillNPCWidebleeding), WideBleeding)
	testutil.ExpectEffectIDs(t, "NPC_EVILLAND imported ground", SkillGroundEffectIDs(db.SkillNPCEvilland), BottomEvilLand)
	testutil.ExpectEffectIDs(t, "NPC_CRITICALWOUND imported hit", SkillHitEffectIDs(db.SkillNPCCriticalwound), CriticalWound)
	testutil.ExpectEffectIDs(t, "RG_STEALCOIN imported success", SkillSuccessEffectIDs(db.SkillRGStealcoin), StealCoin, RogueCoin)
	testutil.ExpectEffectIDs(t, "RG_BACKSTAP action", SkillEffectIDs(db.SkillRGBackstap), BackStab)
	testutil.ExpectEffectIDs(t, "RG_RAID success", SkillSuccessEffectSelfIDs(db.SkillRGRaid), TeiHit3)
	testutil.ExpectEffectIDs(t, "RG_STRIPWEAPON imported success", SkillSuccessEffectIDs(db.SkillRGStripweapon), StripWeapon)
	testutil.ExpectEffectIDs(t, "RG_STRIPSHIELD imported success", SkillSuccessEffectIDs(db.SkillRGStripshield), StripShield)
	testutil.ExpectEffectIDs(t, "RG_STRIPARMOR imported success", SkillSuccessEffectIDs(db.SkillRGStriparmor), StripArmor)
	testutil.ExpectEffectIDs(t, "RG_STRIPHELM imported success", SkillSuccessEffectIDs(db.SkillRGStriphelm), StripHelm)
	testutil.ExpectEffectIDs(t, "RG_INTIMIDATE caster", SkillEffectOnCasterIDs(db.SkillRGIntimidate), Intimidate)
	testutil.ExpectEffectIDs(t, "RG_GRAFFITI imported empty", SkillEffectIDs(db.SkillRGGraffiti))
	testutil.ExpectEffectIDs(t, "RG_FLAGGRAFFITI imported empty", SkillEffectIDs(db.SkillRGFlaggraffiti))
	testutil.ExpectEffectIDs(t, "RG_CLEANER imported empty", SkillEffectIDs(db.SkillRGCleaner))
	testutil.ExpectEffectIDs(t, "RG_CLOSECONFINE success", SkillSuccessEffectIDs(db.SkillRGCloseconfine), QuakeBody4)
	testutil.ExpectEffectIDs(t, "RG_CLOSECONFINE imported ground", SkillGroundEffectIDs(db.SkillRGCloseconfine))
	testutil.ExpectEffectIDs(t, "AM_PHARMACY imported empty", SkillEffectIDs(db.SkillAMPharmacy))
	testutil.ExpectEffectIDs(t, "AM_DEMONSTRATION imported ground", SkillGroundEffectIDs(db.SkillAMDemonstration))
	testutil.ExpectEffectIDs(t, "AM_ACIDTERROR at release", SkillEffectIDs(db.SkillAMAcidterror), ThrowItem)
	testutil.ExpectEffectIDs(t, "AM_POTIONPITCHER imported", SkillEffectIDs(db.SkillAMPotionpitcher))
	testutil.ExpectEffectIDs(t, "AM_CANNIBALIZE imported empty", SkillEffectIDs(db.SkillAMCannibalize))
	testutil.ExpectEffectIDs(t, "AM_SPHEREMINE imported empty", SkillEffectIDs(db.SkillAMSpheremine))
	testutil.ExpectEffectIDs(t, "AM_CP_WEAPON imported", SkillSuccessEffectIDs(db.SkillAMCpWeapon), ChemicalProt)
	testutil.ExpectEffectIDs(t, "AM_CP_SHIELD imported", SkillSuccessEffectIDs(db.SkillAMCpShield), ChemicalProt)
	testutil.ExpectEffectIDs(t, "AM_CP_ARMOR imported", SkillSuccessEffectIDs(db.SkillAMCpArmor), ChemicalProt)
	testutil.ExpectEffectIDs(t, "AM_CP_HELM imported", SkillSuccessEffectIDs(db.SkillAMCpHelm), ChemicalProt)
	testutil.ExpectEffectIDs(t, "AM_CALLHOMUN imported empty", SkillEffectIDs(db.SkillAMCallhomun))
	testutil.ExpectEffectIDs(t, "AM_REST imported empty", SkillEffectIDs(db.SkillAMRest))
	testutil.ExpectEffectIDs(t, "AM_RESURRECTHOMUN imported empty", SkillEffectIDs(db.SkillAMResurrecthomun))
	testutil.ExpectEffectIDs(t, "CR_AUTOGUARD imported", SkillSuccessEffectSelfIDs(db.SkillCRAutoguard), Guard)
	testutil.ExpectEffectIDs(t, "CR_SHIELDCHARGE imported", SkillEffectIDs(db.SkillCRShieldcharge), ShieldCharge)
	testutil.ExpectEffectIDs(t, "CR_SHIELDBOOMERANG imported", SkillEffectIDs(db.SkillCRShieldboomerang), ShieldBoomer)
	testutil.ExpectEffectIDs(t, "CR_SHIELDBOOMERANG imported before hit", SkillBeforeHitEffectIDs(db.SkillCRShieldboomerang))
	testutil.ExpectEffectIDs(t, "CR_REFLECTSHIELD imported", SkillSuccessEffectSelfIDs(db.SkillCRReflectshield), ReflectShield)
	testutil.ExpectEffectIDs(t, "CR_HOLYCROSS imported", SkillEffectIDs(db.SkillCRHolycross), HolyCross)
	testutil.ExpectEffectIDs(t, "CR_GRANDCROSS imported", SkillEffectOnCasterIDs(db.SkillCRGrandcross), GrandCross)
	testutil.ExpectEffectIDs(t, "CR_DEVOTION imported", SkillSuccessEffectSelfIDs(db.SkillCRDevotion), Devotion)
	testutil.ExpectEffectIDs(t, "CR_PROVIDENCE imported", SkillEffectIDs(db.SkillCRProvidence), Providence)
	testutil.ExpectEffectIDs(t, "CR_DEFENDER imported", SkillSuccessEffectSelfIDs(db.SkillCRDefender), CrusaderDef)
	testutil.ExpectEffectIDs(t, "CR_SPEARQUICKEN imported", SkillSuccessEffectSelfIDs(db.SkillCRSpearquicken), SpearQuicken)
	testutil.ExpectEffectIDs(t, "MO_CALLSPIRITS imported empty", SkillEffectIDs(db.SkillMOCallspirits))
	testutil.ExpectEffectIDs(t, "MO_ABSORBSPIRITS imported self success", SkillSuccessEffectSelfIDs(db.SkillMOAbsorbspirits), AbsorbSpirits)
	testutil.ExpectEffectIDs(t, "MO_BODYRELOCATION imported empty", SkillEffectIDs(db.SkillMOBodyrelocation))
	testutil.ExpectEffectIDs(t, "MO_INVESTIGATE imported", SkillEffectIDs(db.SkillMOInvestigate), TeiHit2, Chimto)
	testutil.ExpectEffectIDs(t, "MO_FINGEROFFENSIVE imported", SkillEffectIDs(db.SkillMOFingeroffensive), Tanji)
	testutil.ExpectEffectIDs(t, "MO_STEELBODY imported", SkillEffectIDs(db.SkillMOSteelbody), SteelBody, db.SkillEffectSteelBodyRing, Quake)
	testutil.ExpectEffectIDs(t, "MO_BLADESTOP imported empty", SkillEffectIDs(db.SkillMOBladestop))
	testutil.ExpectEffectIDs(t, "MO_EXPLOSIONSPIRITS imported caster", SkillEffectOnCasterIDs(db.SkillMOExplosionspirits), Gumgang2, Quake)
	testutil.ExpectEffectIDs(t, "MO_EXTREMITYFIST imported", SkillEffectIDs(db.SkillMOExtremityfist), TeiHit1X)
	testutil.ExpectEffectIDs(t, "MO_EXTREMITYFIST imported hit", SkillHitEffectIDs(db.SkillMOExtremityfist), Hit1)
	testutil.ExpectEffectIDs(t, "MO_TRIPLEATTACK imported", SkillEffectIDs(db.SkillMOTripleattack), TripleAttack)
	testutil.ExpectEffectIDs(t, "MO_CHAINCOMBO imported", SkillEffectIDs(db.SkillMOChaincombo), TeiHit1, ChainCombo)
	testutil.ExpectEffectIDs(t, "MO_CHAINCOMBO imported caster", SkillEffectOnCasterIDs(db.SkillMOChaincombo), Gumgang3)
	testutil.ExpectEffectIDs(t, "MO_COMBOFINISH imported", SkillEffectOnCasterIDs(db.SkillMOCombofinish), Gumgang3, HitLine, Quake)
	testutil.ExpectEffectIDs(t, "SA_CASTCANCEL imported empty", SkillEffectIDs(db.SkillSACastcancel))
	testutil.ExpectEffectIDs(t, "SA_MAGICROD success on caster", SkillSuccessEffectSelfIDs(db.SkillSAMagicrod), MagicRod)
	testutil.ExpectEffectIDs(t, "SA_SPELLBREAKER imported success", SkillSuccessEffectIDs(db.SkillSASpellbreaker), SpellBreaker)
	testutil.ExpectEffectIDs(t, "SA_AUTOSPELL imported empty", SkillEffectIDs(db.SkillSAAutospell))
	testutil.ExpectEffectIDs(t, "SA_FLAMELAUNCHER imported success", SkillSuccessEffectIDs(db.SkillSAFlamelauncher), FlameLauncher)
	testutil.ExpectEffectIDs(t, "SA_FROSTWEAPON imported success", SkillSuccessEffectIDs(db.SkillSAFrostweapon), FrostWeapon)
	testutil.ExpectEffectIDs(t, "SA_LIGHTNINGLOADER imported success", SkillSuccessEffectIDs(db.SkillSALightningloader), LightningLoad)
	testutil.ExpectEffectIDs(t, "SA_SEISMICWEAPON imported success", SkillSuccessEffectIDs(db.SkillSASeismicweapon), SeismicWeapon)
	testutil.ExpectEffectIDs(t, "SA_DISPELL imported success", SkillSuccessEffectIDs(db.SkillSADispell), Dispell)
	testutil.ExpectEffectIDs(t, "SA_VOLCANO ground activation", SkillGroundEffectIDs(db.SkillSAVolcano), Volcano)
	testutil.ExpectEffectIDs(t, "SA_DELUGE ground activation", SkillGroundEffectIDs(db.SkillSADeluge), Deluge)
	testutil.ExpectEffectIDs(t, "SA_VIOLENTGALE ground activation", SkillGroundEffectIDs(db.SkillSAViolentgale), ViolentGale)
	testutil.ExpectEffectIDs(t, "SA_LANDPROTECTOR ground activation", SkillGroundEffectIDs(db.SkillSALandprotector), LandProtector)
	testutil.ExpectEffectIDs(t, "SA_ABRACADABRA imported empty", SkillEffectIDs(db.SkillSAAbracadabra))
	for _, skillID := range []uint16{
		db.SkillSAMonocell,
		db.SkillSAClasschange,
		db.SkillSASummonmonster,
		db.SkillSAReverseorcish,
		db.SkillSADeath,
		db.SkillSAFortune,
		db.SkillSATamingmonster,
		db.SkillSAQuestion,
		db.SkillSAGravity,
		db.SkillSALevelup,
		db.SkillSAInstantdeath,
		db.SkillSAFullrecovery,
		db.SkillSAComa,
	} {
		testutil.ExpectEffectIDs(t, "SA_ABRACADABRA result imported empty", SkillEffectIDs(skillID))
	}
	testutil.ExpectEffectIDs(t, "BD_ADAPTATION imported empty", SkillEffectIDs(db.SkillBDAdaptation))
	testutil.ExpectEffectIDs(t, "BD_ENCORE imported empty", SkillEffectIDs(db.SkillBDEncore))
	testutil.ExpectEffectIDs(t, "BD_LULLABY imported", SkillEffectIDs(db.SkillBDLullaby))
	testutil.ExpectEffectIDs(t, "BD_LULLABY imported ground", SkillGroundEffectIDs(db.SkillBDLullaby), BottomLullabyGround)
	testutil.ExpectEffectIDs(t, "BD_RICHMANKIM imported", SkillEffectIDs(db.SkillBDRichmankim))
	testutil.ExpectEffectIDs(t, "BD_RICHMANKIM imported ground", SkillGroundEffectIDs(db.SkillBDRichmankim), BottomRichKimGround)
	testutil.ExpectEffectIDs(t, "BD_ETERNALCHAOS imported", SkillEffectIDs(db.SkillBDEternalchaos))
	testutil.ExpectEffectIDs(t, "BD_ETERNALCHAOS imported ground", SkillGroundEffectIDs(db.SkillBDEternalchaos), BottomChaosGround)
	testutil.ExpectEffectIDs(t, "BD_DRUMBATTLEFIELD imported", SkillEffectIDs(db.SkillBDDrumbattlefield))
	testutil.ExpectEffectIDs(t, "BD_DRUMBATTLEFIELD imported ground", SkillGroundEffectIDs(db.SkillBDDrumbattlefield), BottomDrumGround)
	testutil.ExpectEffectIDs(t, "BD_RINGNIBELUNGEN imported", SkillEffectIDs(db.SkillBDRingnibelungen))
	testutil.ExpectEffectIDs(t, "BD_RINGNIBELUNGEN imported ground", SkillGroundEffectIDs(db.SkillBDRingnibelungen), BottomNibelungGround)
	testutil.ExpectEffectIDs(t, "BD_ROKISWEIL imported", SkillEffectIDs(db.SkillBDRokisweil))
	testutil.ExpectEffectIDs(t, "BD_ROKISWEIL imported ground", SkillGroundEffectIDs(db.SkillBDRokisweil), BottomRokiGround)
	testutil.ExpectEffectIDs(t, "BD_INTOABYSS imported", SkillEffectIDs(db.SkillBDIntoabyss))
	testutil.ExpectEffectIDs(t, "BD_INTOABYSS imported ground", SkillGroundEffectIDs(db.SkillBDIntoabyss), BottomAbyssGround)
	testutil.ExpectEffectIDs(t, "BD_SIEGFRIED imported", SkillEffectIDs(db.SkillBDSiegfried))
	testutil.ExpectEffectIDs(t, "BD_SIEGFRIED imported ground", SkillGroundEffectIDs(db.SkillBDSiegfried), BottomSiegGround)
	testutil.ExpectEffectIDs(t, "BA_MUSICALLESSON imported empty", SkillEffectIDs(db.SkillBaMusicallesson))
	testutil.ExpectEffectIDs(t, "BA_MUSICALSTRIKE imported before hit", SkillBeforeHitEffectIDs(db.SkillBaMusicalstrike), ArrowShot)
	if skillHidesCastAura(db.SkillBaMusicalstrike) {
		t.Fatal("BA_MUSICALSTRIKE must show the native Bash casting effect")
	}
	testutil.ExpectEffectIDs(t, "BA_DISSONANCE imported ground", SkillGroundEffectIDs(db.SkillBaDissonance), BottomDissonanceGround)
	testutil.ExpectEffectIDs(t, "BA_FROSTJOKE success", SkillSuccessEffectSelfIDs(db.SkillBaFrostjoke), TalkFrostJoke)
	testutil.ExpectEffectIDs(t, "BA_WHISTLE imported", SkillEffectIDs(db.SkillBaWhistle))
	testutil.ExpectEffectIDs(t, "BA_WHISTLE imported ground", SkillGroundEffectIDs(db.SkillBaWhistle), BottomWhistleGround)
	testutil.ExpectEffectIDs(t, "BA_ASSASSINCROSS imported", SkillEffectIDs(db.SkillBaAssassincross))
	testutil.ExpectEffectIDs(t, "BA_ASSASSINCROSS imported ground", SkillGroundEffectIDs(db.SkillBaAssassincross), BottomSinXGround)
	testutil.ExpectEffectIDs(t, "BA_POEMBRAGI imported", SkillEffectIDs(db.SkillBaPoembragi))
	testutil.ExpectEffectIDs(t, "BA_POEMBRAGI imported ground", SkillGroundEffectIDs(db.SkillBaPoembragi), BottomBragiGround)
	testutil.ExpectEffectIDs(t, "BA_APPLEIDUN imported", SkillEffectIDs(db.SkillBaAppleidun))
	testutil.ExpectEffectIDs(t, "BA_APPLEIDUN imported ground", SkillGroundEffectIDs(db.SkillBaAppleidun), BottomAppleGround)
	testutil.ExpectEffectIDs(t, "BA_PANGVOICE imported success", SkillSuccessEffectIDs(db.SkillBaPangvoice), FVoice)
	testutil.ExpectEffectIDs(t, "DC_DANCINGLESSON imported empty", SkillEffectIDs(db.SkillDCDancinglesson))
	testutil.ExpectEffectIDs(t, "DC_THROWARROW imported before hit", SkillBeforeHitEffectIDs(db.SkillDCThrowarrow), ArrowShot)
	if skillHidesCastAura(db.SkillDCThrowarrow) {
		t.Fatal("DC_THROWARROW uses the original casting flare")
	}
	testutil.ExpectEffectIDs(t, "DC_UGLYDANCE imported ground", SkillGroundEffectIDs(db.SkillDCUglydance), BottomUglyDanceGround)
	testutil.ExpectEffectIDs(t, "DC_SCREAM original caster chat", SkillSuccessEffectSelfIDs(db.SkillDCScream), TalkScream)
	testutil.ExpectEffectIDs(t, "DC_HUMMING imported", SkillEffectIDs(db.SkillDCHumming))
	testutil.ExpectEffectIDs(t, "DC_HUMMING imported ground", SkillGroundEffectIDs(db.SkillDCHumming), BottomHummingGround)
	testutil.ExpectEffectIDs(t, "DC_DONTFORGETME imported", SkillEffectIDs(db.SkillDCDontforgetme))
	testutil.ExpectEffectIDs(t, "DC_DONTFORGETME imported ground", SkillGroundEffectIDs(db.SkillDCDontforgetme), BottomForgetGround)
	testutil.ExpectEffectIDs(t, "DC_FORTUNEKISS imported", SkillEffectIDs(db.SkillDCFortunekiss))
	testutil.ExpectEffectIDs(t, "DC_FORTUNEKISS imported ground", SkillGroundEffectIDs(db.SkillDCFortunekiss), BottomFortuneGround)
	testutil.ExpectEffectIDs(t, "DC_SERVICEFORYOU imported", SkillEffectIDs(db.SkillDCServiceforyou))
	testutil.ExpectEffectIDs(t, "DC_SERVICEFORYOU imported ground", SkillGroundEffectIDs(db.SkillDCServiceforyou), BottomServiceGround)
	testutil.ExpectEffectIDs(t, "DC_WINKCHARM imported success", SkillSuccessEffectIDs(db.SkillDCWinkcharm), Wink)
	testutil.ExpectEffectIDs(t, "SL_KAIZEL imported", SkillEffectIDs(db.SkillSLKaizel), Kaizel)
	testutil.ExpectEffectIDs(t, "SL_SMA original", SkillEffectIDs(db.SkillSLSma), FourWayBody, Stin2, HitLine6, HitTexture)
	testutil.ExpectEffectIDs(t, "SL_SWOO imported", SkillEffectIDs(db.SkillSLSwoo), M07, BabyBody)
	testutil.ExpectEffectIDs(t, "SL_SKA imported", SkillEffectIDs(db.SkillSLSka), SteelBody, Gumgang2, Quake)
	testutil.ExpectEffectIDs(t, "AM_BERSERKPITCHER imported", SkillSuccessEffectIDs(db.SkillAMBerserkpitcher), ItemFast3)
	testutil.ExpectEffectIDs(t, "AM_BERSERKPITCHER imported before hit", SkillBeforeHitEffectIDs(db.SkillAMBerserkpitcher))
	testutil.ExpectEffectIDs(t, "AM_TWILIGHT1 imported", SkillEffectIDs(db.SkillAMTwilight1))
	testutil.ExpectEffectIDs(t, "AM_TWILIGHT2 imported", SkillEffectIDs(db.SkillAMTwilight2))
	testutil.ExpectEffectIDs(t, "AM_TWILIGHT3 imported", SkillEffectIDs(db.SkillAMTwilight3))
	testutil.ExpectEffectIDs(t, "CR_ALCHEMY imported empty", SkillEffectIDs(db.SkillCRAlchemy))
	testutil.ExpectEffectIDs(t, "CR_SYNTHESISPOTION imported empty", SkillEffectIDs(db.SkillCRSynthesispotion))
	testutil.ExpectEffectIDs(t, "CR_SLIMPITCHER imported empty", SkillEffectIDs(db.SkillCRSlimpitcher))
	testutil.ExpectEffectIDs(t, "CR_FULLPROTECTION success", SkillSuccessEffectIDs(db.SkillCRFullprotection), ChemicalProt, ChemicalBody)
	testutil.ExpectEffectIDs(t, "CR_CULTIVATION imported empty", SkillEffectIDs(db.SkillCRCultivation))
	testutil.ExpectEffectIDs(t, "SA_CREATECON imported empty", SkillEffectIDs(db.SkillSACreatecon))
	testutil.ExpectEffectIDs(t, "SA_ELEMENTWATER success", SkillSuccessEffectIDs(db.SkillSAElementwater), FrostWeapon)
	testutil.ExpectEffectIDs(t, "SA_ELEMENTGROUND success", SkillSuccessEffectIDs(db.SkillSAElementground), SeismicWeapon)
	testutil.ExpectEffectIDs(t, "SA_ELEMENTFIRE success", SkillSuccessEffectIDs(db.SkillSAElementfire), FlameLauncher)
	testutil.ExpectEffectIDs(t, "SA_ELEMENTWIND success", SkillSuccessEffectIDs(db.SkillSAElementwind), LightningLoad)
	testutil.ExpectEffectIDs(t, "MO_KITRANSLATION imported empty", SkillEffectIDs(db.SkillMOKitranslation))
	testutil.ExpectEffectIDs(t, "MO_BALKYOUNG imported", SkillHitEffectIDs(db.SkillMOBalkyoung), Hit3)
	testutil.ExpectEffectIDs(t, "HP_ASSUMPTIO", SkillEffectIDs(db.SkillHPAssumptio), Assumptio, Assumptio2)
	testutil.ExpectEffectIDs(t, "HP_BASILICA imported ground", SkillGroundEffectIDs(db.SkillHPBasilica), BottomBasilica)
	testutil.ExpectEffectIDs(t, "HP_MEDITATIO has no robr effect row", SkillEffectIDs(db.SkillHPMeditatio))
	testutil.ExpectEffectIDs(t, "HP_MANARECHARGE has no robr effect row", SkillEffectIDs(db.SkillHPManarecharge))
	testutil.ExpectEffectIDs(t, "PA_PRESSURE imported before hit", SkillEffectIDs(db.SkillPaPressure), Pressure)
	testutil.ExpectEffectIDs(t, "PA_SACRIFICE imported", SkillEffectOnCasterIDs(db.SkillPaSacrifice), Bash3D)
	testutil.ExpectEffectIDs(t, "PA_GOSPEL imported", SkillEffectIDs(db.SkillPaGospel), BottomGospel)
	testutil.ExpectEffectIDs(t, "PA_GOSPEL imported ground", SkillGroundEffectIDs(db.SkillPaGospel), GospelGround)
	testutil.ExpectEffectIDs(t, "PA_SHIELDCHAIN imported before hit", SkillEffectIDs(db.SkillPaShieldchain), ShieldBoomerang3)
	testutil.ExpectEffectIDs(t, "CH_PALMSTRIKE imported hit", SkillHitEffectIDs(db.SkillChPalmstrike), Hit1)
	testutil.ExpectEffectIDs(t, "CH_TIGERFIST imported", SkillEffectOnCasterIDs(db.SkillChTigerfist), Bash3D2, Quake, Gumgang3)
	testutil.ExpectEffectIDs(t, "CH_CHAINCRUSH imported", SkillEffectIDs(db.SkillChChaincrush), Chemical2Dash)
	testutil.ExpectEffectIDs(t, "CH_SOULCOLLECT imported begin", SkillBeginEffectIDs(db.SkillChSoulcollect))
	testutil.ExpectEffectIDs(t, "SN_SIGHT imported", SkillSuccessEffectSelfIDs(db.SkillSNSight), TrueSight)
	testutil.ExpectEffectIDs(t, "SN_FALCONASSAULT imported", SkillEffectIDs(db.SkillSNFalconassault), FalconAssault)
	testutil.ExpectEffectIDs(t, "HT_PHANTASMIC imported before hit", SkillBeforeHitEffectIDs(db.SkillHTPhantasmic), ArrowShot)
	testutil.ExpectEffectIDs(t, "HT_PHANTASMIC imported hit", SkillHitEffectIDs(db.SkillHTPhantasmic), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "SN_SHARPSHOOTING imported begin", SkillBeginEffectIDs(db.SkillSNSharpshooting))
	testutil.ExpectEffectIDs(t, "SN_SHARPSHOOTING imported before hit", SkillBeforeHitEffectIDs(db.SkillSNSharpshooting), ArrowShot)
	testutil.ExpectEffectIDs(t, "SN_SHARPSHOOTING imported hit", SkillHitEffectIDs(db.SkillSNSharpshooting), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "SN_WINDWALK imported", SkillSuccessEffectSelfIDs(db.SkillSNWindwalk), Portal4)
	testutil.ExpectEffectIDs(t, "WS_CREATECOIN imported empty", SkillEffectIDs(db.SkillWSCreatecoin))
	testutil.ExpectEffectIDs(t, "WS_CREATENUGGET imported empty", SkillEffectIDs(db.SkillWSCreatenugget))
	testutil.ExpectEffectIDs(t, "WS_SYSTEMCREATE imported empty", SkillEffectIDs(db.SkillWSSystemcreate))
	testutil.ExpectEffectIDs(t, "WS_WEAPONREFINE imported empty", SkillEffectIDs(db.SkillWSWeaponrefine))
	testutil.ExpectEffectIDs(t, "CG_ARROWVULCAN imported", SkillEffectIDs(db.SkillCGArrowvulcan), TripleAttack3)
	testutil.ExpectEffectIDs(t, "CG_ARROWVULCAN imported before hit", SkillBeforeHitEffectIDs(db.SkillCGArrowvulcan), ArrowShot)
	testutil.ExpectEffectIDs(t, "CG_MOONLIT imported", SkillEffectIDs(db.SkillCGMoonlit))
	testutil.ExpectEffectIDs(t, "CG_MOONLIT imported ground", SkillGroundEffectIDs(db.SkillCGMoonlit))
	testutil.ExpectEffectIDs(t, "CG_MARIONETTE imported", SkillEffectIDs(db.SkillCGMarionette))
	testutil.ExpectEffectIDs(t, "CG_MARIONETTE imported hit", SkillHitEffectIDs(db.SkillCGMarionette))
	testutil.ExpectEffectIDs(t, "CG_LONGINGFREEDOM imported", SkillSuccessEffectSelfIDs(db.SkillCGLongingfreedom), ChemicalBody)
	testutil.ExpectEffectIDs(t, "CG_HERMODE imported music", SkillEffectIDs(db.SkillCGHermode))
	testutil.ExpectEffectIDs(t, "CG_HERMODE imported ground", SkillGroundEffectIDs(db.SkillCGHermode))
	testutil.ExpectEffectIDs(t, "CG_TAROTCARD imported success", SkillSuccessEffectIDs(db.SkillCGTarotcard), ChemicalBody)
	testutil.ExpectEffectIDs(t, "CR_ACIDDEMONSTRATION imported", SkillEffectIDs(db.SkillCRAciddemonstration), AcidDemon)
	testutil.ExpectEffectIDs(t, "SL_KAAHI imported", SkillEffectIDs(db.SkillSLKaahi), Hated)
	testutil.ExpectEffectIDs(t, "WE_MALE imported empty", SkillEffectIDs(db.SkillWEMale))
	testutil.ExpectEffectIDs(t, "WE_FEMALE imported empty", SkillEffectIDs(db.SkillWEFemale))
	testutil.ExpectEffectIDs(t, "WE_CALLPARTNER imported empty", SkillEffectIDs(db.SkillWECallpartner))
	testutil.ExpectEffectIDs(t, "WE_BABY imported", SkillEffectIDs(db.SkillWEBaby), 408)
	testutil.ExpectEffectIDs(t, "WE_CALLPARENT imported empty", SkillEffectIDs(db.SkillWECallparent))
	testutil.ExpectEffectIDs(t, "WE_CALLBABY imported empty", SkillEffectIDs(db.SkillWECallbaby))
	testutil.ExpectEffectIDs(t, "TK_JUMPKICK imported hit", SkillHitEffectIDs(db.SkillTKJumpkick))
	testutil.ExpectEffectIDs(t, "AS_VENOMKNIFE imported before hit", SkillBeforeHitEffectIDs(db.SkillASVenomknife), ThrowItem6)
	testutil.ExpectEffectIDs(t, "ALL_WEWISH imported", SkillEffectIDs(db.SkillALLWewish), ChristmasCarol)
	testutil.ExpectEffectIDs(t, "NPC_VENOMFOG imported", SkillEffectIDs(db.SkillNPCVenomfog), VenomFog)
	testutil.ExpectEffectIDs(t, "RK_IGNITIONBREAK imported caster", SkillEffectOnCasterIDs(db.SkillRKIgnitionbreak), IgnitionBreak)
	testutil.ExpectEffectIDs(t, "RK_DRAGONBREATH imported hit", SkillHitEffectIDs(db.SkillRKDragonbreath), M05)
	testutil.ExpectEffectIDs(t, "RK_DRAGONHOWLING imported", SkillEffectIDs(db.SkillRKDragonhowling), DragonHowling)
	testutil.ExpectEffectIDs(t, "RK_MILLENNIUMSHIELD imported", SkillEffectIDs(db.SkillRKMillenniumshield), MillenniumShield)
	testutil.ExpectEffectIDs(t, "RK_ENCHANTBLADE imported", SkillEffectIDs(db.SkillRKEnchantblade), BerserkPotion2)
	testutil.ExpectEffectIDs(t, "RK_SONICWAVE imported", SkillEffectIDs(db.SkillRKSonicwave), HealN)
	testutil.ExpectEffectIDs(t, "WL_WHITEIMPRISON imported", SkillEffectIDs(db.SkillWLWhiteimprison), BottomBasilica2)
	testutil.ExpectEffectIDs(t, "WL_FROSTMISTY imported", SkillEffectIDs(db.SkillWLFrostmisty), FrostMisty)
	testutil.ExpectEffectIDs(t, "WL_MARSHOFABYSS imported", SkillEffectIDs(db.SkillWLMarshofabyss), MarshOfAbyss)
	testutil.ExpectEffectIDs(t, "WL_RECOGNIZEDSPELL imported", SkillEffectIDs(db.SkillWLRecognizedspell), Recognized)
	testutil.ExpectEffectIDs(t, "WL_STASIS imported", SkillEffectIDs(db.SkillWLStasis), Stasis)
	testutil.ExpectEffectIDs(t, "WL_CRIMSONROCK imported", SkillEffectIDs(db.SkillWLCrimsonrock), CrimsonRock)
	testutil.ExpectEffectIDs(t, "WL_HELLINFERNO imported ground", SkillGroundEffectIDs(db.SkillWLHellinferno), HellInferno)
	testutil.ExpectEffectIDs(t, "WL_CHAINLIGHTNING_ATK imported", SkillEffectIDs(db.SkillWLChainlightningAtk), ChainLightning)
	testutil.ExpectEffectIDs(t, "WL_EARTHSTRAIN imported ground", SkillGroundEffectIDs(db.SkillWLEarthstrain), EarthWall)
	testutil.ExpectEffectIDs(t, "WL_TETRAVORTEX imported", SkillEffectIDs(db.SkillWLTetravortex), Tetra)
	testutil.ExpectEffectIDs(t, "WL_TETRAVORTEX imported begin", SkillBeginEffectIDs(db.SkillWLTetravortex), TetraCasting)
	testutil.ExpectEffectIDs(t, "GC_ROLLINGCUTTER imported", SkillEffectIDs(db.SkillGCRollingcutter), CastSpin2)
	testutil.ExpectEffectIDs(t, "AB_JUDEX imported", SkillEffectIDs(db.SkillABJudex), FirePillarOn2)
	testutil.ExpectEffectIDs(t, "AB_JUDEX imported hit", SkillHitEffectIDs(db.SkillABJudex), HolyLight)
	testutil.ExpectEffectIDs(t, "AB_ADORAMUS imported", SkillEffectIDs(db.SkillABAdoramus), Adoramus)
	testutil.ExpectEffectIDs(t, "AB_EPICLESIS imported", SkillEffectIDs(db.SkillABEpiclesis), GlassWall4)
	testutil.ExpectEffectIDs(t, "AB_EPICLESIS imported ground", SkillGroundEffectIDs(db.SkillABEpiclesis), GlassWall3)
	testutil.ExpectEffectIDs(t, "RA_ARROWSTORM imported", SkillEffectIDs(db.SkillRAArrowstorm), ArrowStorm)
	testutil.ExpectEffectIDs(t, "RA_AIMEDBOLT imported", SkillEffectIDs(db.SkillRAAimedbolt), AimedBolt)
	testutil.ExpectEffectIDs(t, "RA_AIMEDBOLT imported before hit", SkillBeforeHitEffectIDs(db.SkillRAAimedbolt), ArrowShot)
	testutil.ExpectEffectIDs(t, "RA_DETONATOR imported", SkillEffectIDs(db.SkillRADetonator), Concentration2)
	testutil.ExpectEffectIDs(t, "NC_POWERSWING imported", SkillEffectIDs(db.SkillNCPowerswing), CrashAxe)
	testutil.ExpectEffectIDs(t, "SR_EARTHSHAKER imported", SkillEffectIDs(db.SkillSREarthshaker), Electric4)
	testutil.ExpectEffectIDs(t, "GC_DARKCROW imported", SkillEffectIDs(db.SkillGCDarkcrow), GCDarkCrow)
	testutil.ExpectEffectIDs(t, "GN_ILLUSIONDOPING imported", SkillEffectIDs(db.SkillGNIllusiondoping), GNIllusionDoping)
	testutil.ExpectEffectIDs(t, "RK_LUXANIMA imported", SkillEffectIDs(db.SkillRKLuxanima), RKLuxAnima)
	testutil.ExpectEffectIDs(t, "NC_MAGMA_ERUPTION imported", SkillEffectIDs(db.SkillNCMagmaEruption), NCMagmaEruption)
	testutil.ExpectEffectIDs(t, "SO_ELEMENTAL_SHIELD imported", SkillEffectIDs(db.SkillSOElementalShield), SOElemShield)
	testutil.ExpectEffectIDs(t, "SR_FLASHCOMBO imported", SkillEffectIDs(db.SkillSRFlashcombo), SRFlashCombo)
	testutil.ExpectEffectIDs(t, "AB_OFFERTORIUM imported", SkillEffectIDs(db.SkillABOffertorium), ABOffertorium)
	testutil.ExpectEffectIDs(t, "WL_TELEKINESIS_INTENSE imported", SkillEffectIDs(db.SkillWLTelekinesisIntense), WLTelekinesis)
	testutil.ExpectEffectIDs(t, "ALL_FULL_THROTTLE imported", SkillEffectIDs(db.SkillALLFullThrottle), AllFullThrottle)
	testutil.ExpectEffectIDs(t, "SC_BODYPAINT imported", SkillEffectIDs(db.SkillSCBodypaint), Stretch)
	testutil.ExpectEffectIDs(t, "SC_ENERVATION imported", SkillEffectIDs(db.SkillSCEnervation), Enervation)
	testutil.ExpectEffectIDs(t, "SC_GROOMY imported", SkillEffectIDs(db.SkillSCGroomy), Enervation2)
	testutil.ExpectEffectIDs(t, "SC_IGNORANCE imported", SkillEffectIDs(db.SkillSCIgnorance), Enervation3)
	testutil.ExpectEffectIDs(t, "SC_LAZINESS imported", SkillEffectIDs(db.SkillSCLaziness), Enervation4)
	testutil.ExpectEffectIDs(t, "SC_UNLUCKY imported", SkillEffectIDs(db.SkillSCUnlucky), Enervation5)
	testutil.ExpectEffectIDs(t, "SC_WEAKNESS imported", SkillEffectIDs(db.SkillSCWeakness), Enervation6)
	testutil.ExpectEffectIDs(t, "SC_MANHOLE imported ground", SkillGroundEffectIDs(db.SkillSCManhole), BottomManhole)
	testutil.ExpectEffectIDs(t, "SC_MANHOLE imported success", SkillSuccessEffectIDs(db.SkillSCManhole), Manhole)
	testutil.ExpectEffectIDs(t, "SC_DIMENSIONDOOR imported ground", SkillGroundEffectIDs(db.SkillSCDimensiondoor), ForestLight6)
	testutil.ExpectEffectIDs(t, "SC_CHAOSPANIC imported ground", SkillGroundEffectIDs(db.SkillSCChaospanic), BottomAni)
	testutil.ExpectEffectIDs(t, "SC_MAELSTROM imported ground", SkillGroundEffectIDs(db.SkillSCMaelstrom), BottomMaelstrom)
	testutil.ExpectEffectIDs(t, "SC_BLOODYLUST imported ground", SkillGroundEffectIDs(db.SkillSCBloodylust), BottomBloodyLust)
	testutil.ExpectEffectIDs(t, "LG_SHIELDPRESS imported before hit", SkillBeforeHitEffectIDs(db.SkillLGShieldpress), Pressure2)
	testutil.ExpectEffectIDs(t, "LG_PRESTIGE imported", SkillEffectIDs(db.SkillLGPrestige), PrimeCharge2)
	testutil.ExpectEffectIDs(t, "LG_BANDING imported", SkillEffectIDs(db.SkillLGBanding), PrimeCharge3)
	testutil.ExpectEffectIDs(t, "LG_INSPIRATION imported", SkillEffectIDs(db.SkillLGInspiration), PrimeCharge4)
	testutil.ExpectEffectIDs(t, "SO_FIREWALK imported ground", SkillGroundEffectIDs(db.SkillSOFirewalk), FireWall2)
	testutil.ExpectEffectIDs(t, "SO_ELECTRICWALK imported ground", SkillGroundEffectIDs(db.SkillSOElectricwalk), Shockwave2)
	testutil.ExpectEffectIDs(t, "SO_DIAMONDDUST imported", SkillEffectIDs(db.SkillSODiamonddust), ColdThrow2)
	testutil.ExpectEffectIDs(t, "SO_PSYCHIC_WAVE imported", SkillEffectIDs(db.SkillSOPsychicWave), SprPlant10)
	testutil.ExpectEffectIDs(t, "SO_WARMER imported", SkillEffectIDs(db.SkillSOWarmer), DemonicFire4)
	testutil.ExpectEffectIDs(t, "SO_VARETYR_SPEAR imported before hit", SkillBeforeHitEffectIDs(db.SkillSOVaretyrSpear), Pressure3)
	testutil.ExpectEffectIDs(t, "WM_REVERBERATION imported ground", SkillGroundEffectIDs(db.SkillWmReverberation), BotReverb)
	testutil.ExpectEffectIDs(t, "WM_REVERBERATION_MELEE imported", SkillEffectIDs(db.SkillWmReverberationMelee), BotReverb2)
	testutil.ExpectEffectIDs(t, "WM_SEVERE_RAINSTORM imported", SkillEffectIDs(db.SkillWmSevereRainstorm), RainParticle)
	testutil.ExpectEffectIDs(t, "WM_POEMOFNETHERWORLD imported ground", SkillGroundEffectIDs(db.SkillWmPoemofnetherworld), BotReverb2)
	testutil.ExpectEffectIDs(t, "WM_VOICEOFSIREN imported ground", SkillGroundEffectIDs(db.SkillWmVoiceofsiren), HeartAsura)
	testutil.ExpectEffectIDs(t, "WM_LULLABY_DEEPSLEEP imported", SkillEffectIDs(db.SkillWmLullabyDeepsleep), ChemicalV2)
	testutil.ExpectEffectIDs(t, "WM_SIRCLEOFNATURE imported", SkillEffectIDs(db.SkillWmSircleofnature), CirclePower2)
	testutil.ExpectEffectIDs(t, "WM_RANDOMIZESPELL imported", SkillEffectIDs(db.SkillWmRandomizespell), Secra2)
	testutil.ExpectEffectIDs(t, "WM_GLOOMYDAY imported", SkillEffectIDs(db.SkillWmGloomyday), Dance1)
	testutil.ExpectEffectIDs(t, "WM_SONG_OF_MANA imported ground", SkillGroundEffectIDs(db.SkillWmSongOfMana), SprPlant3)
	testutil.ExpectEffectIDs(t, "WM_DANCE_WITH_WUG imported ground", SkillGroundEffectIDs(db.SkillWmDanceWithWug), SprPlant2)
	testutil.ExpectEffectIDs(t, "WM_SATURDAY_NIGHT_FEVER imported ground", SkillGroundEffectIDs(db.SkillWmSaturdayNightFever), SprPlant4)
	testutil.ExpectEffectIDs(t, "WM_LERADS_DEW imported ground", SkillGroundEffectIDs(db.SkillWmLeradsDew), SprPlant5)
	testutil.ExpectEffectIDs(t, "WM_MELODYOFSINK imported ground", SkillGroundEffectIDs(db.SkillWmMelodyofsink), SprPlant6)
	testutil.ExpectEffectIDs(t, "WM_BEYOND_OF_WARCRY imported ground", SkillGroundEffectIDs(db.SkillWmBeyondOfWarcry), SprPlant7)
	testutil.ExpectEffectIDs(t, "WM_UNLIMITED_HUMMING_VOICE imported ground", SkillGroundEffectIDs(db.SkillWmUnlimitedHummingVoice), SprPlant8)
	testutil.ExpectEffectIDs(t, "HAMI_CASTLE imported", SkillEffectIDs(db.SkillHamiCastle), HamiCastle)
	testutil.ExpectEffectIDs(t, "HAMI_DEFENCE imported", SkillEffectIDs(db.SkillHamiDefence), HamiDefence)
	testutil.ExpectEffectIDs(t, "HAMI_BLOODLUST imported", SkillEffectIDs(db.SkillHamiBloodlust), HamiBlood)
	testutil.ExpectEffectIDs(t, "MH_POISON_MIST imported", SkillEffectIDs(db.SkillMhPoisonMist), PoisonMist)
	testutil.ExpectEffectIDs(t, "MH_ERASER_CUTTER imported", SkillEffectIDs(db.SkillMhEraserCutter), EraserCutter)
	testutil.ExpectEffectIDs(t, "MH_SONIC_CRAW imported", SkillEffectIDs(db.SkillMhSonicCraw), SonicClaw)
	testutil.ExpectEffectIDs(t, "MH_MIDNIGHT_FRENZY imported", SkillEffectIDs(db.SkillMhMidnightFrenzy), MidnightFrenzy)
	testutil.ExpectEffectIDs(t, "MH_TINDER_BREAKER imported", SkillEffectIDs(db.SkillMhTinderBreaker), TinderBreaker)
	testutil.ExpectEffectIDs(t, "MH_LAVA_SLIDE imported", SkillEffectIDs(db.SkillMhLavaSlide), LavaSlide)
	testutil.ExpectEffectIDs(t, "MH_VOLCANIC_ASH imported", SkillEffectIDs(db.SkillMhVolcanicAsh), VolcanicAsh)
	testutil.ExpectEffectIDs(t, "AB_CHEAL imported", SkillEffectIDs(db.SkillABCheal), Heal2)
	testutil.ExpectEffectIDs(t, "AB_HIGHNESSHEAL imported", SkillEffectIDs(db.SkillABHighnessheal), Heal4)
	testutil.ExpectEffectIDs(t, "AB_HIGHNESSHEAL imported hit", SkillHitEffectIDs(db.SkillABHighnessheal), HealOffensive)
}

func TestRobrowserMiniSTREffectSpecs(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		min  string
	}{
		{"Mammonite", Mammonite, "maemor", "memor_min"},
		{"Angelus", Angelus, "angelus", "jong_mini"},
		{"Cure", Cure, "cure", "cure_min"},
		{"Gloria", Gloria, "gloria", "gloria_min"},
		{"Magnificat", Magnificat, "magnificat", "magnificat_min"},
		{"Resurrection", Resurrection, "resurrection", "resurrection_min"},
		{"Lex Aeterna", LexAeterna, "lexaeterna", "lexaeterna_min"},
		{"Suffragium", Suffragium, "suffragium", "suffragium_min"},
		{"Storm Gust", StormGust, "stormgust", "storm_min"},
		{"Weapon Perfection", WeaponPerfect, "weaponperfection", "weaponperfection_min"},
		{"Maximize Power", MaximizePower, "maximizepower", "maximize_min"},
		{"Kyrie Eleison", Kyrie, "kyrie", "kyrie_min"},
		{"Christmas Carol", ChristmasCarol, "angelus", "jong_mini"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) == 0 {
			t.Fatalf("%s spec = %+v ok=%t, want a STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.StrMinFile != tc.min || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want STR %q min %q attached", tc.name, component, tc.file, tc.min)
		}
	}
}

func TestMVPEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := Lookup(Mvp)
	if !ok {
		t.Fatal("MVP effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("MVP components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentSTR || component.StrFile != "mvp" || !component.AttachedEntity {
		t.Fatalf("MVP component = %+v, want attached STR mvp", component)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\st_mvp.wav" {
		t.Fatalf("MVP sfx = %#v", spec.Sfx)
	}
}

func TestHunterStringKeyEffectsMatchRobrowser(t *testing.T) {
	cast, ok := Lookup(SharpShootingCast)
	if !ok || cast.Duration != 10*time.Second || len(cast.Components) != 1 {
		t.Fatalf("496_beforecast spec = %+v ok=%t, want one 10s CastRing component", cast, ok)
	}
	component := cast.Components[0]
	if component.Kind != ComponentFUNC || component.FuncName != "CastRing" || component.FuncAdapter != EffectFuncCastRing || component.TextureName != "ring_jadu" {
		t.Fatalf("496_beforecast component identity = %+v", component)
	}
	if component.BottomSize != 0.8 || component.TopSize != 2.45 || component.Height != 2.8 || component.PosZ != 0.08 {
		t.Fatalf("496_beforecast component shape = %+v", component)
	}
	if component.TotalCircleSides != 20 || component.CircleSides != 20 || component.AlphaMax != 0.9 || !component.Fade || !component.Rotate || !component.AttachedEntity {
		t.Fatalf("496_beforecast component flags = %+v", component)
	}
}

func TestBlacksmithStringKeyEffectsMatchRobrowser(t *testing.T) {
	adrenaline, ok := Lookup(AdrenalineCast)
	if !ok || adrenaline.Duration != 500*time.Millisecond || len(adrenaline.Components) != 0 || len(adrenaline.Sfx) != 1 || adrenaline.Sfx[0] != "effect\\black_adrenalinerush_a.wav" {
		t.Fatalf("98_beforecast spec = %+v ok=%t, want robr adrenaline pre-cast sound", adrenaline, ok)
	}

	maximize, ok := Lookup(MaximizeSounds)
	if !ok || maximize.Duration != 950*time.Millisecond || len(maximize.Components) != 0 {
		t.Fatalf("maximize_power_sounds spec = %+v ok=%t, want robr delayed sound row", maximize, ok)
	}
	wantSFX := []string{
		"effect\\black_maximize_power_circle.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword_bic.wav",
	}
	wantDelays := []time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond}
	if !reflect.DeepEqual(maximize.Sfx, wantSFX) || !reflect.DeepEqual(maximize.SfxDelays, wantDelays) {
		t.Fatalf("maximize_power_sounds sfx = %#v delays %#v", maximize.Sfx, maximize.SfxDelays)
	}

	greed, ok := Lookup(GreedSound)
	if !ok || greed.Duration != 500*time.Millisecond || len(greed.Components) != 0 || len(greed.Sfx) != 1 || greed.Sfx[0] != "effect\\ef_entry.wav" {
		t.Fatalf("ef_greed_sound spec = %+v ok=%t, want robr greed sound", greed, ok)
	}
}

func TestKnightStringKeyEffectsMatchRobrowser(t *testing.T) {
	white, ok := Lookup(WhitePulse)
	if !ok || white.Duration != 500*time.Millisecond || len(white.Components) != 0 || len(white.Sfx) != 0 {
		t.Fatalf("white_pulse spec = %+v ok=%t, want robr no-draw 500ms row", white, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"spear_hit_sound", SpearHitSound, "_hit_spear.wav"},
		{"enemy_hit_normal1", EnemyHitNormal1, "_enemy_hit_normal1.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 500*time.Millisecond || len(spec.Components) != 0 || len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s spec = %+v ok=%t, want sound %q", tc.name, spec, ok, tc.wav)
		}
	}

	quake, ok := Lookup(Quake)
	if !ok || quake.Duration != 650*time.Millisecond || quake.CameraShake != 650*time.Millisecond || len(quake.Components) != 1 {
		t.Fatalf("quake spec = %+v ok=%t, want 650ms CameraQuake", quake, ok)
	}
	quakeComponent := quake.Components[0]
	if quakeComponent.Kind != ComponentFUNC || quakeComponent.FuncName != "CameraQuake" || quakeComponent.Duration != 650*time.Millisecond || !quakeComponent.AttachedEntity {
		t.Fatalf("quake component = %+v", quakeComponent)
	}
}

func TestGospelGroundHasTileAndFloatingCross(t *testing.T) {
	gospel, ok := Lookup(GospelGround)
	if !ok || gospel.Duration != 1500*time.Millisecond || len(gospel.Components) != 2 {
		t.Fatalf("370_ground spec = %+v ok=%t, want two song ground components", gospel, ok)
	}
	tile, cross := gospel.Components[0], gospel.Components[1]
	if tile.Kind != ComponentFUNC || tile.FuncName != "FlatColorTile" || tile.FuncAdapter != EffectFuncFlatColorTile || tile.Color != (color.RGBA{R: 255, G: 255, B: 255, A: 13}) || tile.SizeStart != 1 || !tile.RenderBefore || tile.AttachedEntity {
		t.Fatalf("370_ground tile = %+v", tile)
	}
	if cross.Kind != ComponentFUNC || cross.FuncAdapter != EffectFuncGospelCross || cross.TextureFile != "effect/cross_old.bmp" || !cross.Repeat || cross.RenderBefore {
		t.Fatalf("370_ground cross = %+v", cross)
	}
}

func TestArcherThiefMerchantSkillEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "AC_CONCENTRATION", SkillSuccessEffectSelfIDs(45), Concentration)
	testutil.ExpectEffectIDs(t, "AC_DOUBLE begin", SkillBeginEffectIDs(46), BashBegin)
	testutil.ExpectEffectIDs(t, "AC_DOUBLE before-hit", SkillBeforeHitEffectIDs(46), ArrowShot)
	testutil.ExpectEffectIDs(t, "AC_DOUBLE hit", SkillHitEffectIDs(46), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "AC_SHOWER", SkillEffectIDs(47), ArrowShower)
	testutil.ExpectEffectIDs(t, "AC_SHOWER hit", SkillHitEffectIDs(47), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "AC_CHARGEARROW hit", SkillHitEffectIDs(148), Hit1)
	if skillHidesCastAura(db.SkillACChargearrow) {
		t.Fatal("AC_CHARGEARROW must show the native casting flash")
	}
	testutil.ExpectEffectIDs(t, "TF_DOUBLE passive", SkillEffectIDs(48))
	testutil.ExpectEffectIDs(t, "TF_MISS passive", SkillEffectIDs(49))
	testutil.ExpectEffectIDs(t, "TF_STEAL success", SkillSuccessEffectIDs(50), Steal)
	testutil.ExpectEffectIDs(t, "TF_HIDING", SkillEffectIDs(51))
	testutil.ExpectEffectIDs(t, "TF_POISON begin", SkillBeginEffectIDs(52), BashBegin, BashBegin)
	testutil.ExpectEffectIDs(t, "TF_POISON hit", SkillHitEffectIDs(52), Hit1, BashHit)
	testutil.ExpectEffectIDs(t, "TF_DETOXIFY", SkillEffectIDs(53), Detoxication)
	testutil.ExpectEffectIDs(t, "TF_SPRINKLESAND before-hit", SkillBeforeHitEffectIDs(149), SprinkleSand)
	testutil.ExpectEffectIDs(t, "TF_SPRINKLESAND hit", SkillHitEffectIDs(149), Hit1)
	testutil.ExpectEffectIDs(t, "TF_BACKSLIDING", SkillEffectIDs(150))
	testutil.ExpectEffectIDs(t, "TF_PICKSTONE", SkillEffectIDs(151))
	if !skillHidesCastAura(db.SkillTFPickstone) {
		t.Fatal("TF_PICKSTONE should hide cast aura like roBrowser")
	}
	testutil.ExpectEffectIDs(t, "TF_THROWSTONE before-hit", SkillBeforeHitEffectIDs(152), ThrowItem3)
	testutil.ExpectEffectIDs(t, "TF_THROWSTONE hit", SkillHitEffectIDs(152), Hit1)
	testutil.ExpectEffectIDs(t, "MC_MAMMONITE hit", SkillHitEffectIDs(42), Hit1, Mammonite)
	testutil.ExpectEffectIDs(t, "MC_CARTREVOLUTION before hit", SkillBeforeHitEffectSelfIDs(153), CartRevolution)
	testutil.ExpectEffectIDs(t, "MC_CARTREVOLUTION hit", SkillHitEffectIDs(153), Hit1, CartRevolution)
	testutil.ExpectEffectIDs(t, "MC_LOUD", SkillSuccessEffectSelfIDs(155), Loud)
	testutil.ExpectEffectIDs(t, "AL_HOLYLIGHT", SkillHitEffectIDs(156), Hit1, HolyLight)
}

func TestArcherProjectileEffects(t *testing.T) {
	shot, ok := Lookup(ArrowShot)
	if !ok || len(shot.Components) != 1 {
		t.Fatalf("arrow shot spec = %#v ok=%t, want one component", shot, ok)
	}
	component := shot.Components[0]
	if component.Kind != Component3D || component.SpriteFile != "data/sprite/npc/skel_archer_arrow" {
		t.Fatalf("arrow shot component = %#v, want skel_archer_arrow 3D sprite", component)
	}
	if !component.ToSrc || !component.RotateToTarget || !component.RotateWithCamera || component.Duration != 140*time.Millisecond {
		t.Fatalf("arrow shot robr flags = %#v", component)
	}

	shower, ok := Lookup(ArrowShower)
	if !ok || len(shower.Components) != 1 {
		t.Fatalf("arrow shower spec = %#v ok=%t, want one component", shower, ok)
	}
	component = shower.Components[0]
	if component.FuncAdapter != EffectFuncArrowShower || shower.Duration != scene.ReferenceBowArrowFlightDuration {
		t.Fatalf("arrow shower must use native flight and spread: %#v", component)
	}
}

func TestWarpEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "AL_TELEPORT begin", SkillBeginEffectIDs(26))
	testutil.ExpectEffectIDs(t, "Butterfly Wing item", itemUseEffectSpec(602).effectIDs, Teleportation)
	testutil.ExpectEffectIDs(t, "Fly Wing item", itemUseEffectSpec(601).effectIDs)
}

func TestSpeedPotionItemEffectMappingsMatchRobrowser(t *testing.T) {
	testutil.ExpectEffectIDs(t, "Concentration Potion item", itemUseEffectSpec(645).effectIDs, ItemFast)
	testutil.ExpectEffectIDs(t, "Awakening Potion item", itemUseEffectSpec(656).effectIDs, ItemFast2)
	testutil.ExpectEffectIDs(t, "Berserk Potion item", itemUseEffectSpec(657).effectIDs, ItemFast3)
}

func TestSkillUnitEffectMappings(t *testing.T) {
	testutil.ExpectEffectIDs(t, "UNT_SAFETYWALL", SkillUnitEffectIDs(126), SafetyWall)
	testutil.ExpectEffectIDs(t, "UNT_FIREWALL", SkillUnitEffectIDs(127), FireWall)
	testutil.ExpectEffectIDs(t, "UNT_WARPPORTAL", SkillUnitEffectIDs(128), Portal)
	testutil.ExpectEffectIDs(t, "UNT_PRE_WARPPORTAL", SkillUnitEffectIDs(129), ReadyPortal)
	testutil.ExpectEffectIDs(t, "UNT_SANCTUARY", SkillUnitEffectIDs(131), BottomSanc)
	testutil.ExpectEffectIDs(t, "UNT_MAGNUS", SkillUnitEffectIDs(132), BottomMagnus)
	testutil.ExpectEffectIDs(t, "UNT_PNEUMA", SkillUnitEffectIDs(133), Pneuma)
	testutil.ExpectEffectIDs(t, "UNT_FIREPILLAR_WAITING", SkillUnitEffectIDs(135), FirePillarOn)
	testutil.ExpectEffectIDs(t, "UNT_ICEWALL", SkillUnitEffectIDs(141), IceWall)
	testutil.ExpectEffectIDs(t, "UNT_QUAGMIRE", SkillUnitEffectIDs(142), Quagmire)
	testutil.ExpectEffectIDs(t, "UNT_VENOMDUST", SkillUnitEffectIDs(146), VenomDust2)
	testutil.ExpectEffectIDs(t, "UNT_VOLCANO", SkillUnitEffectIDs(154), BottomVolcano)
	testutil.ExpectEffectIDs(t, "UNT_DELUGE", SkillUnitEffectIDs(155), BottomDeluge)
	testutil.ExpectEffectIDs(t, "UNT_VIOLENTGALE", SkillUnitEffectIDs(156), BottomViolent)
	testutil.ExpectEffectIDs(t, "UNT_LANDPROTECTOR", SkillUnitEffectIDs(157), BottomLand)
	testutil.ExpectEffectIDs(t, "UNT_LULLABY", SkillUnitEffectIDs(158), BottomLullabyGround)
	testutil.ExpectEffectIDs(t, "UNT_RICHMANKIM", SkillUnitEffectIDs(159), BottomRichKimGround)
	testutil.ExpectEffectIDs(t, "UNT_ETERNALCHAOS", SkillUnitEffectIDs(160), BottomChaosGround)
	testutil.ExpectEffectIDs(t, "UNT_DRUMBATTLEFIELD", SkillUnitEffectIDs(161), BottomDrumGround)
	testutil.ExpectEffectIDs(t, "UNT_RINGNIBELUNGEN", SkillUnitEffectIDs(162), BottomNibelungGround)
	testutil.ExpectEffectIDs(t, "UNT_ROKISWEIL", SkillUnitEffectIDs(163), BottomRokiGround)
	testutil.ExpectEffectIDs(t, "UNT_INTOABYSS", SkillUnitEffectIDs(164), BottomAbyssGround)
	testutil.ExpectEffectIDs(t, "UNT_SIEGFRIED", SkillUnitEffectIDs(165), BottomSiegGround)
	testutil.ExpectEffectIDs(t, "UNT_DISSONANCE", SkillUnitEffectIDs(166), BottomDissonanceGround)
	testutil.ExpectEffectIDs(t, "UNT_WHISTLE", SkillUnitEffectIDs(167), BottomWhistleGround)
	testutil.ExpectEffectIDs(t, "UNT_ASSASSINCROSS", SkillUnitEffectIDs(168), BottomSinXGround)
	testutil.ExpectEffectIDs(t, "UNT_POEMBRAGI", SkillUnitEffectIDs(169), BottomBragiGround)
	testutil.ExpectEffectIDs(t, "UNT_APPLEIDUN", SkillUnitEffectIDs(170), BottomAppleGround)
	testutil.ExpectEffectIDs(t, "UNT_UGLYDANCE", SkillUnitEffectIDs(171), BottomUglyDanceGround)
	testutil.ExpectEffectIDs(t, "UNT_HUMMING", SkillUnitEffectIDs(172), BottomHummingGround)
	testutil.ExpectEffectIDs(t, "UNT_DONTFORGETME", SkillUnitEffectIDs(173), BottomForgetGround)
	testutil.ExpectEffectIDs(t, "UNT_FORTUNEKISS", SkillUnitEffectIDs(174), BottomFortuneGround)
	testutil.ExpectEffectIDs(t, "UNT_SERVICEFORYOU", SkillUnitEffectIDs(175), BottomServiceGround)
	testutil.ExpectEffectIDs(t, "UNT_DEMONSTRATION", SkillUnitEffectIDs(177), Demonstration)
	testutil.ExpectEffectIDs(t, "UNT_GOSPEL", SkillUnitEffectIDs(179), GospelGround)
	testutil.ExpectEffectIDs(t, "UNT_BASILICA", SkillUnitEffectIDs(180), BottomBasilica)
	testutil.ExpectEffectIDs(t, "UNT_MOONLIT", SkillUnitEffectIDs(181))
	testutil.ExpectEffectIDs(t, "UNT_FOGWALL", SkillUnitEffectIDs(182), FogWallGround)
	testutil.ExpectEffectIDs(t, "UNT_SPIDERWEB", SkillUnitEffectIDs(183), BottomSpider)
	testutil.ExpectEffectIDs(t, "UNT_GRAVITATION", SkillUnitEffectIDs(184), Gravitation)
	testutil.ExpectEffectIDs(t, "UNT_HERMODE", SkillUnitEffectIDs(185), BottomHermode)
	testutil.ExpectEffectIDs(t, "UNT_TATAMIGAESHI", SkillUnitEffectIDs(188), Tatami)
	testutil.ExpectEffectIDs(t, "UNT_KAEN", SkillUnitEffectIDs(189), Kaen)
	testutil.ExpectEffectIDs(t, "UNT_EVILLAND", SkillUnitEffectIDs(199), BottomEvilLand)
	testutil.ExpectEffectIDs(t, "UNT_EPICLESIS", SkillUnitEffectIDs(202), GlassWall3)
	testutil.ExpectEffectIDs(t, "UNT_EARTHSTRAIN", SkillUnitEffectIDs(203), EarthWall)
	testutil.ExpectEffectIDs(t, "UNT_MANHOLE", SkillUnitEffectIDs(204), BottomManhole)
	testutil.ExpectEffectIDs(t, "UNT_DIMENSIONDOOR", SkillUnitEffectIDs(205), ForestLight6)
	testutil.ExpectEffectIDs(t, "UNT_CHAOSPANIC", SkillUnitEffectIDs(206), BottomAni)
	testutil.ExpectEffectIDs(t, "UNT_MAELSTROM", SkillUnitEffectIDs(207), BottomMaelstrom)
	testutil.ExpectEffectIDs(t, "UNT_BLOODYLUST", SkillUnitEffectIDs(208), BottomBloodyLust)
	testutil.ExpectEffectIDs(t, "UNT_REVERBERATION", SkillUnitEffectIDs(218), BotReverb)
	testutil.ExpectEffectIDs(t, "UNT_FIREWALK", SkillUnitEffectIDs(220), FireWall2)
	testutil.ExpectEffectIDs(t, "UNT_ELECTRICWALK", SkillUnitEffectIDs(221), Shockwave2)
	testutil.ExpectEffectIDs(t, "UNT_NETHERWORLD", SkillUnitEffectIDs(222), BotReverb2)
}

func TestSpeedPotionEffectSpecsMatchRobrowserSTRRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		file     string
	}{
		{"Concentration Potion", ItemFast, "집중"},
		{"Awakening Potion", ItemFast2, "각성"},
	} {
		spec, ok := Lookup(tc.effectID)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ac_concentration.wav" {
			t.Fatalf("%s sfx = %v, want ac_concentration", tc.name, spec.Sfx)
		}
	}
}

func TestTeleportationEffectSpecUsesRobrowserCylinderStack(t *testing.T) {
	spec, ok := Lookup(Teleportation)
	if !ok {
		t.Fatal("teleportation effect spec missing")
	}
	if spec.Duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.Duration)
	}
	if !spec.DetachLocalActor {
		t.Fatal("teleportation should detach from the local actor")
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	expected := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.6, 0.8, 25},
		{0.8, 1.0, 13},
		{1.0, 1.3, 5},
	}
	for i, want := range expected {
		component := spec.Components[i]
		if component.Kind != ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 1500*time.Millisecond || component.BlendMode != 2 || !component.BlendAdditive {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.BottomSize != want.bottom || component.TopSize != want.top || component.Height != want.height {
			t.Fatalf("component %d size = %.1f %.1f %.1f, want %.1f %.1f %.1f", i, component.BottomSize, component.TopSize, component.Height, want.bottom, want.top, want.height)
		}
		if component.FixedPerspective {
			t.Fatalf("component %d uses fixed perspective, want world-space cylinder", i)
		}
		if !component.AttachedEntity {
			t.Fatalf("component %d is not attached to the entity", i)
		}
	}
}

func TestWarpPortalEffectSpecUsesPortal2Cylinders(t *testing.T) {
	spec, ok := Lookup(Portal)
	if !ok {
		t.Fatal("portal effect spec missing")
	}
	if len(spec.Sfx) != 2 || spec.Sfx[0] != "effect\\ef_readyportal.wav" || spec.Sfx[1] != "effect\\ef_portal.wav" {
		t.Fatalf("portal sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	first := spec.Components[0]
	if first.Kind != ComponentCylinder || first.TextureName != "ring_blue" || first.Duration != 500*time.Millisecond || first.Animation != 4 {
		t.Fatalf("first portal component = %+v", first)
	}
	if !first.Repeat || first.RepeatDelay != -300*time.Millisecond {
		t.Fatalf("first portal repeat = %t delay=%s, want reference client repeat -300ms", first.Repeat, first.RepeatDelay)
	}
	if spec.Components[3].TextureName != "alpha1" || spec.Components[3].PosZ != 2 || spec.Components[3].Height != 1 {
		t.Fatalf("portal cap component = %+v", spec.Components[3])
	}
}

func TestHealEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := Lookup(Heal)
	if !ok {
		t.Fatal("heal effect spec missing")
	}
	if spec.Duration != 1840*time.Millisecond {
		t.Fatalf("duration = %s, want 1840ms", spec.Duration)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	for i, component := range spec.Components[:2] {
		if component.Kind != ComponentCylinder || component.TextureName != "ring_white" || component.Animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.Duration != 1500*time.Millisecond || component.Height != 8 || component.AlphaMax != 0.2 {
			t.Fatalf("component %d timing/shape = %+v", i, component)
		}
		if component.Color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) || !component.BlendAdditive {
			t.Fatalf("component %d tint/blend = %+v", i, component)
		}
	}
	firstParticle := spec.Components[2]
	if firstParticle.Kind != Component3D || firstParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("first heal particle = %+v", firstParticle)
	}
	if firstParticle.Duration != 1300*time.Millisecond || firstParticle.Delay != 400*time.Millisecond || firstParticle.DuplicateDelay != 10*time.Millisecond || firstParticle.Duplicate != 15 {
		t.Fatalf("first heal particle timing = %+v", firstParticle)
	}
	if firstParticle.AlphaMax != 0.6 || !firstParticle.FadeIn || !firstParticle.FadeOut || firstParticle.Sparkling || firstParticle.SparkNumber != 0 {
		t.Fatalf("first heal particle fade = %+v", firstParticle)
	}
	if firstParticle.PosXRand != 1.5 || firstParticle.PosYRand != 1.5 || firstParticle.PosZEndRand != 2 || firstParticle.PosZEndMiddle != 6 {
		t.Fatalf("first heal particle position = %+v", firstParticle)
	}
	if firstParticle.SizeStart != 9*scene.PixelRatio || firstParticle.SizeEnd != 9*scene.PixelRatio || firstParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("first heal particle size = %+v", firstParticle)
	}
	secondParticle := spec.Components[3]
	if secondParticle.Kind != Component3D || secondParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("second heal particle = %+v", secondParticle)
	}
	if secondParticle.Duration != 1100*time.Millisecond || secondParticle.Delay != 200*time.Millisecond || secondParticle.DuplicateDelay != 50*time.Millisecond || secondParticle.Duplicate != 7 {
		t.Fatalf("second heal particle timing = %+v", secondParticle)
	}
	if secondParticle.AlphaMax != 0.6 || !secondParticle.FadeIn || !secondParticle.FadeOut || secondParticle.Sparkling || secondParticle.SparkNumber != 0 {
		t.Fatalf("second heal particle fade = %+v", secondParticle)
	}
	if secondParticle.PosXRand != 1 || secondParticle.PosYRand != 1 || secondParticle.PosZEnd != 5 || secondParticle.PosZStartRand != 1 {
		t.Fatalf("second heal particle position = %+v", secondParticle)
	}
	if secondParticle.SizeStart != 9*scene.PixelRatio || secondParticle.SizeEnd != 9*scene.PixelRatio || secondParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("second heal particle size = %+v", secondParticle)
	}
}

func TestHealOffensiveEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := Lookup(HealOffensive)
	if !ok {
		t.Fatal("offensive heal effect spec missing")
	}
	if spec.Duration != 1490*time.Millisecond {
		t.Fatalf("duration = %s, want 1490ms", spec.Duration)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	for i, component := range spec.Components[:2] {
		if component.Kind != ComponentCylinder || component.TextureName != "ring_white" || component.Animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.Duration != time.Second || !component.BlendAdditive || component.Color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
			t.Fatalf("component %d timing/tint = %+v", i, component)
		}
	}
	if spec.Components[0].Height != 10 || spec.Components[1].Height != 9 {
		t.Fatalf("cylinder heights = %.1f %.1f", spec.Components[0].Height, spec.Components[1].Height)
	}
	firstParticle := spec.Components[2]
	if firstParticle.Kind != Component3D || firstParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("first offensive heal particle = %+v", firstParticle)
	}
	if firstParticle.Duration != time.Second || firstParticle.Delay != 400*time.Millisecond || firstParticle.DuplicateDelay != 10*time.Millisecond || firstParticle.Duplicate != 10 {
		t.Fatalf("first offensive heal particle timing = %+v", firstParticle)
	}
	if firstParticle.AlphaMax != 0.8 || !firstParticle.FadeIn || !firstParticle.FadeOut || !firstParticle.BlendAdditive || !firstParticle.Sparkling || firstParticle.SparkNumber != 2 {
		t.Fatalf("first offensive heal particle fade/blend = %+v", firstParticle)
	}
	if firstParticle.PosXRand != 1.5 || firstParticle.PosYRand != 1.5 || firstParticle.PosZEndRand != 3 || firstParticle.PosZEndMiddle != 6 {
		t.Fatalf("first offensive heal particle position = %+v", firstParticle)
	}
	secondParticle := spec.Components[3]
	if secondParticle.Kind != Component3D || secondParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("second offensive heal particle = %+v", secondParticle)
	}
	if secondParticle.Duration != 900*time.Millisecond || secondParticle.Delay != 200*time.Millisecond || secondParticle.DuplicateDelay != 50*time.Millisecond || secondParticle.Duplicate != 5 {
		t.Fatalf("second offensive heal particle timing = %+v", secondParticle)
	}
	if secondParticle.AlphaMax != 0.8 || !secondParticle.FadeIn || !secondParticle.FadeOut || !secondParticle.BlendAdditive || !secondParticle.Sparkling || secondParticle.SparkNumber != 2 {
		t.Fatalf("second offensive heal particle fade/blend = %+v", secondParticle)
	}
	if secondParticle.PosXRand != 1 || secondParticle.PosYRand != 1 || secondParticle.PosZEnd != 6 || secondParticle.PosZStartRand != 1 {
		t.Fatalf("second offensive heal particle position = %+v", secondParticle)
	}
	if secondParticle.SizeStart != 9*scene.PixelRatio || secondParticle.SizeEnd != 9*scene.PixelRatio || secondParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("second offensive heal particle size = %+v", secondParticle)
	}
}

func TestAngelusEffectSpecUsesRobrowserSTR(t *testing.T) {
	spec, ok := Lookup(Angelus)
	if !ok {
		t.Fatal("angelus effect spec missing")
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ef_angelus.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentSTR || component.StrFile != "angelus" || component.StrMinFile != "jong_mini" {
		t.Fatalf("STR resource = %+v", component)
	}
	if !component.AttachedEntity || !component.SpriteHead {
		t.Fatalf("STR attachment flags = %+v", component)
	}
}

func TestEmotionEffectSpecUsesEntityAttachmentOffset(t *testing.T) {
	spec, ok := Lookup(Emotion)
	if !ok {
		t.Fatal("emotion effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentSPR || component.SpriteFile != "emotion" {
		t.Fatalf("emotion component = %+v", component)
	}
	if !component.AttachedEntity || component.SpriteYOffset != -100 || component.SpriteHead {
		t.Fatalf("emotion placement = %+v", component)
	}
}

func TestWorldEffectDuplicateDeltasMatchRobrowserSemantics(t *testing.T) {
	component := Component{
		AlphaMax:      0.2,
		AlphaMaxDelta: 0.2,
		SizeStart:     100 * scene.PixelRatio,
		SizeEnd:       100 * scene.PixelRatio,
		SizeDelta:     -10,
	}
	if got := EffectBillboardAlphaForDuplicate(0.5, component, 2); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("duplicate alpha = %.3f, want 0.6", got)
	}
	sizeX, sizeY := Effect3DSize(component, Instance{}, 0, 0.5, 2)
	want := 80 * scene.PixelRatio
	if math.Abs(sizeX-want) > 0.001 || math.Abs(sizeY-want) > 0.001 {
		t.Fatalf("duplicate size = %.3f x %.3f, want %.3f", sizeX, sizeY, want)
	}
}

func TestEffect3DSpriteScaleUsesRobrowserSpriteUnits(t *testing.T) {
	size := testutil.EffectTableSize(200)
	got := Effect3DSpriteScale(size)
	want := size / 100
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("sprite pixel scale = %.5f, want %.5f", got, want)
	}
	fireballWorldWidth := 64 * got
	wantWidth := 128 * scene.PixelRatio
	if math.Abs(fireballWorldWidth-wantWidth) > 0.001 {
		t.Fatalf("64px fireball width = %.3f, want robr 128/35 %.3f", fireballWorldWidth, wantWidth)
	}
}

func TestEffect3DSpriteDrawOptionsHonorAdditiveBlend(t *testing.T) {
	defaultOptions := Effect3DSpriteDrawOptions(Component{})
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default sprite effect blend = %v, want source-over", got)
	}
	if got := defaultOptions.DepthBias; got != 0 {
		t.Fatalf("default sprite effect depth bias = %.3f, want 0", got)
	}
	if got := Effect3DSpriteDrawOptions(Component{BlendAdditive: true}).Blend; got != render.BlendLighter {
		t.Fatalf("additive sprite effect blend = %v, want lighter", got)
	}
	if got := Effect3DSpriteDrawOptions(Component{WorldSizedSprite: true}).DepthBias; got != StrEffectDepthBias {
		t.Fatalf("world-sized sprite effect depth bias = %.3f, want %.3f", got, StrEffectDepthBias)
	}
	if Effect3DSpriteDrawOptions(Component{Overlay: true}).DepthTest {
		t.Fatal("overlay sprite effect depth test enabled, want disabled")
	}
}

func TestTexturedEffectBillboardDrawOptionsHonorOverlay(t *testing.T) {
	defaultOptions := TexturedEffectBillboardDrawOptions(false, false)
	if !defaultOptions.DepthTest {
		t.Fatal("default textured effect depth test disabled, want enabled")
	}
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default textured effect blend = %v, want source-over", got)
	}
	additiveOptions := TexturedEffectBillboardDrawOptions(true, false)
	if got := additiveOptions.Blend; got != render.BlendLighter {
		t.Fatalf("additive textured effect blend = %v, want lighter", got)
	}
	overlayOptions := TexturedEffectBillboardDrawOptions(false, true)
	if overlayOptions.DepthTest {
		t.Fatal("overlay textured effect depth test enabled, want disabled")
	}
}

func TestWorldEffectBillboardSparklingAlphaMatchesRobrowser(t *testing.T) {
	component := Component{
		AlphaMax:    1,
		Sparkling:   true,
		SparkNumber: 2,
	}
	if got := EffectBillboardAlphaForDuplicate(0, component, 0); math.Abs(got-1) > 0.001 {
		t.Fatalf("spark alpha at start = %.3f, want 1", got)
	}
	if got := EffectBillboardAlphaForDuplicate(0.25, component, 0); math.Abs(got-0.008) > 0.001 {
		t.Fatalf("spark alpha at quarter = %.3f, want about 0.008", got)
	}
	if got := EffectBillboardAlphaForDuplicate(0.75, component, 0); math.Abs(got-0.067) > 0.001 {
		t.Fatalf("spark alpha at three quarters = %.3f, want about 0.067", got)
	}
}

func TestWorldEffectBillboardAngleCanRotateWithCamera(t *testing.T) {
	projection := testutil.ProjectionYaw(800, 600, 0, 0, 0, 45)
	component := Component{AngleStart: 90, AngleEnd: 180, RotateWithCamera: true}
	got := WorldEffectBillboardAngleForEffect(component, projection, Instance{}, 0, 0.5)
	want := scene.DegreesToRadians(180)
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("angle = %.3f, want %.3f", got, want)
	}
}

func TestEffectCylinderAngleXRotatesHeightAxisLikeRobrowser(t *testing.T) {
	got := RotateEffectCylinderVector(scene.Point3{Y: 1}, -90, 0, 0)
	want := scene.Point3{Z: -1}
	if !testutil.ModelPointNear(got, want, 0.001) {
		t.Fatalf("rotated height axis = %+v, want %+v", got, want)
	}
}

func TestWorldCylinderBandAllowsRobrowserNegativeHeight(t *testing.T) {
	screen := render.NewFrame(320, 240)
	DrawWorldCylinderBandWithBasis(screen, render.WhiteImage(), render.WhiteImage(), 0, 0, 0, 1, 2, -4, color.RGBA{A: 255}, 8, scene.Point3{X: 1}, scene.Point3{Z: 1}, scene.Point3{Y: 1})
	commands := reflect.ValueOf(screen).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatalf("world commands = %d, want one negative-height cylinder", commands.Len())
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() < 2 {
		t.Fatalf("vertices = %d, want cylinder vertices", vertices.Len())
	}
	bottomY := vertices.Index(0).FieldByName("Y").Float()
	topY := vertices.Index(1).FieldByName("Y").Float()
	if topY >= bottomY {
		t.Fatalf("top Y = %.1f bottom Y = %.1f, want top below bottom", topY, bottomY)
	}
}

func TestSTRAnimationAttachedEntityUsesActorAnchor(t *testing.T) {
	anim := res.STRAnimation{Pos: [2]float32{320, 320}}

	_, groundY := StrAnimationOffset(anim, false)
	_, attachedY := StrAnimationOffset(anim, true)

	if groundY != -0.5 {
		t.Fatalf("ground STR y offset = %.3f, want -0.5", groundY)
	}
	if attachedY != 0 {
		t.Fatalf("attached STR y offset = %.3f, want 0", attachedY)
	}
}

func TestSTRAnimationLocalOffsetFlipsYBeforeRotationLikeRobrowser(t *testing.T) {
	anim := res.STRAnimation{
		Pos:   [2]float32{330, 340},
		Angle: 90,
	}
	gotX, gotY := StrAnimationLocalOffset(anim, 10, 20, true)
	wantX, wantY := -10.0/35.0, -30.0/35.0
	if math.Abs(gotX-wantX) > 0.0001 || math.Abs(gotY-wantY) > 0.0001 {
		t.Fatalf("STR local offset = %.4f %.4f, want %.4f %.4f", gotX, gotY, wantX, wantY)
	}
}

func TestSTRAnimationBlendMatchesRobrowserD3DBlend(t *testing.T) {
	if got := StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 7}); got != render.BlendSrcAlphaDstAlpha {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %v, want BlendSrcAlphaDstAlpha", got)
	}
	if got := StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 2}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/ONE blend = %v, want BlendLighter", got)
	}
	if got := StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6}); got != render.BlendSourceOver {
		t.Fatalf("regular STR blend = %v, want BlendSourceOver", got)
	}
}

func TestSTRAnimationDrawOptionsDisableFogToMatchRobrowser(t *testing.T) {
	options := StrAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if !options.DisableFog {
		t.Fatal("STR draw options enabled map fog, want disabled")
	}
}

func TestSTRAnimationDrawOptionsUseDepthBiasToMatchRobrowser(t *testing.T) {
	options := StrAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if options.DepthBias <= 0 {
		t.Fatalf("STR depth bias = %.3f, want positive robr-style camera bias", options.DepthBias)
	}
}

func TestSTRAnimationVertexUsesCenterDepthToMatchRobrowser(t *testing.T) {
	point := scene.Point3{X: 1, Y: 2, Z: 3}
	depthPoint := scene.Point3{X: 4, Y: 5, Z: 6}
	vertex := StrAnimationVertex3D(point, scene.UV{U: 0.25, V: 0.5}, color.RGBA{R: 51, G: 102, B: 153, A: 204}, 80, 40, depthPoint)

	if vertex.X != 1 || vertex.Y != 2 || vertex.Z != 3 {
		t.Fatalf("vertex position = %.1f %.1f %.1f, want rendered point", vertex.X, vertex.Y, vertex.Z)
	}
	if vertex.DepthX != 4 || vertex.DepthY != 5 || vertex.DepthZ != 6 {
		t.Fatalf("vertex depth = %.1f %.1f %.1f, want STR center depth", vertex.DepthX, vertex.DepthY, vertex.DepthZ)
	}
	if vertex.SrcX != 20 || vertex.SrcY != 20 {
		t.Fatalf("vertex uv = %.1f %.1f, want texture pixel coords", vertex.SrcX, vertex.SrcY)
	}
}

func TestLevelUpEffectSpecsUseSTRResources(t *testing.T) {
	base, ok := Lookup(BaseLevelUp)
	if !ok {
		t.Fatal("base level-up effect spec missing")
	}
	if len(base.Components) != 1 || base.Components[0].Kind != ComponentSTR || base.Components[0].StrFile != "angel" || !base.Components[0].AttachedEntity {
		t.Fatalf("base level-up spec = %+v", base)
	}
	if len(base.Sfx) != 1 || base.Sfx[0] != "levelup.wav" {
		t.Fatalf("base level-up sfx = %#v", base.Sfx)
	}
	job, ok := Lookup(JobLevelUp)
	if !ok {
		t.Fatal("job level-up effect spec missing")
	}
	if len(job.Components) != 1 || job.Components[0].Kind != ComponentSTR || job.Components[0].StrFile != "joblvup" {
		t.Fatalf("job level-up spec = %+v", job)
	}
	if len(job.Sfx) != 0 {
		t.Fatalf("job level-up sfx = %#v", job.Sfx)
	}
}

func TestRepeatedSTRKeyIndexLoops(t *testing.T) {
	starts := time.Unix(100, 0)
	now := starts.Add(250 * time.Millisecond)
	if got := StrEffectKeyIndex(starts, now, 60, 12, true); got != 3 {
		t.Fatalf("repeated key index = %.2f, want 3", got)
	}
	if got := StrEffectKeyIndex(starts, now, 60, 12, false); got != 15 {
		t.Fatalf("one-shot key index = %.2f, want 15", got)
	}
}

func TestTaekwonFamilySkillEffectsMatchRobrowser(t *testing.T) {
	spiritLinks := []uint16{
		db.SkillSLAlchemist, db.SkillSLMonk, db.SkillSLStar, db.SkillSLSage,
		db.SkillSLCrusader, db.SkillSLSupernovice, db.SkillSLKnight, db.SkillSLWizard,
		db.SkillSLPriest, db.SkillSLBarddancer, db.SkillSLRogue, db.SkillSLAssasin,
		db.SkillSLBlacksmith, db.SkillSLHunter, db.SkillSLSoullinker, db.SkillSLHigh,
	}
	for _, skillID := range spiritLinks {
		testutil.ExpectEffectIDs(t, "spirit link", SkillEffectIDs(skillID), 424, 503)
	}
	testutil.ExpectEffectIDs(t, "SG_FUSION", SkillEffectIDs(db.SkillSGFusion))
	testutil.ExpectEffectIDs(t, "SG_FUSION success", SkillSuccessEffectSelfIDs(db.SkillSGFusion), Quake)
	testutil.ExpectEffectIDs(t, "SL_SKA", SkillEffectIDs(db.SkillSLSka), SteelBody, Gumgang2, Quake)
}

func TestUseItemAckAddsItemUseEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &System{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 501, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.Active))
	}
	if effect := mode.Active[0]; effect.ActorID != 2000000 || effect.EffectID != PotionRed || effect.X != 10 || effect.Y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestMercenaryPotionItemEffectsTargetVisibleMercenary(t *testing.T) {
	testCases := []struct {
		name     string
		itemID   uint16
		effectID int
	}{
		{"red", 12184, Food},
		{"blue", 12185, Food},
		{"concentration", 12241, ItemFast},
		{"awakening", 12242, ItemFast2},
		{"berserk", 12243, ItemFast3},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.UpsertActor(worldstate.Actor{ID: 400, X: 13, Y: 24, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary})
			sessionState := &session.Session{
				AccountID: 2000000,
				Mercenary: session.Companion{
					ID:     400,
					Active: true,
				},
			}
			mode := &System{}
			ctx := client.Context{Session: sessionState, World: world}

			mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: tc.itemID, AID: 2000000, Amount: 2, Result: 1})

			if len(mode.Active) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.Active))
			}
			if effect := mode.Active[0]; effect.ActorID != 400 || effect.EffectID != tc.effectID || effect.X != 13 || effect.Y != 24 {
				t.Fatalf("effect = %+v", effect)
			}
		})
	}
}

func TestMercenaryPotionItemEffectsFallbackToAckActorWithoutMercenary(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &System{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12184, AID: 2000000, Amount: 2, Result: 1})

	if len(mode.Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.Active))
	}
	if effect := mode.Active[0]; effect.ActorID != 2000000 || effect.EffectID != Food || effect.X != 10 || effect.Y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestRemoteMercenaryPotionItemEffectKeepsAckActor(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{ID: 400, X: 13, Y: 24, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary})
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 30, Y: 40})
	sessionState := &session.Session{
		AccountID: 2000000,
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
		},
	}
	mode := &System{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12184, AID: 1100, Amount: 2, Result: 1})

	if len(mode.Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.Active))
	}
	if effect := mode.Active[0]; effect.ActorID != 1100 || effect.EffectID != Food || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestUseItemAckDispatchesAllMappedItemEffectArrays(t *testing.T) {
	const itemID uint16 = 65000
	itemEffectSpecs[itemID] = itemEffectSpec{
		effectIDs:         []int{PotionRed, Blessing},
		effectIDsOnCaster: []int{Endure},
	}
	defer delete(itemEffectSpecs, itemID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &System{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: itemID, AID: 1100, Amount: 2, Result: 1})

	want := []struct {
		effectID int
		actorID  uint32
		x        int
		y        int
	}{
		{PotionRed, 1100, 12, 22},
		{Blessing, 1100, 12, 22},
		{Endure, 1100, 12, 22},
	}
	if len(mode.Active) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.Active), len(want), mode.Active)
	}
	for i, wantEffect := range want {
		got := mode.Active[i]
		if got.EffectID != wantEffect.effectID || got.ActorID != wantEffect.actorID || got.X != wantEffect.x || got.Y != wantEffect.y {
			t.Fatalf("effect %d = %+v, want %+v", i, got, wantEffect)
		}
	}
}

func TestButterflyWingEffectIsPinnedAtUsePosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &System{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 602, AID: 2000000, Amount: 1, Result: 1})
	world.Player.X = 30
	world.Player.Y = 40

	if len(mode.Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.Active))
	}
	if effect := mode.Active[0]; effect.ActorID != 0 || effect.EffectID != Teleportation || effect.X != 10 || effect.Y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestMappedSkillUnitEntriesUsePersistentEffects(t *testing.T) {
	for unitID, spec := range skillUnitEffectSpecs {
		if len(spec.effectIDs) == 0 {
			continue
		}
		world := worldstate.New()
		world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
		mode := &System{}
		ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
		entryID := uint32(unitID) + 100000

		mode.AddSkillUnitEffects(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: unitID, X: 12, Y: 34, Visible: true})
		if len(mode.Active) != len(spec.effectIDs) {
			t.Fatalf("unit %d world effects = %d, want %d", unitID, len(mode.Active), len(spec.effectIDs))
		}
		for _, effect := range mode.Active {
			if !effect.Persistent {
				t.Fatalf("unit %d effect %d is not persistent", unitID, effect.EffectID)
			}
		}
	}
}
func TestSkillUnitEntryDispatchesAllMappedUnitEffectArrays(t *testing.T) {
	const unitID uint16 = 65000
	skillUnitEffectSpecs[unitID] = skillUnitEffectSpec{effectIDs: []int{Pneuma, SafetyWall}}
	defer delete(skillUnitEffectSpecs, unitID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &System{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.AddSkillUnitEffects(ctx, network.SkillUnitEntry{ID: 9004, CreatorID: 2000000, UnitID: unitID, X: 123, Y: 456, Visible: true})

	want := []int{Pneuma, SafetyWall}
	if len(mode.Active) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.Active), len(want), mode.Active)
	}
	for i, wantEffectID := range want {
		effect := mode.Active[i]
		if effect.ActorID != 9004 || effect.EffectID != wantEffectID || effect.X != 123 || effect.Y != 456 {
			t.Fatalf("effect %d = %+v, want effect %d on unit", i, effect, wantEffectID)
		}
	}

	mode.RemoveUnit(9004)
	if len(mode.Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.Active))
	}
}
