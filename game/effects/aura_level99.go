package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

const (
	level99AuraLevel    = 99
	Level99AuraLifetime = 24 * time.Hour
)

var (
	// Level99AuraNormalEffects are launched with /aura on: the ring
	// (EF_LEVEL99), the ground glow (EF_LEVEL99_2) and the blue bubbles
	// (EF_LEVEL99_4, two primitives).
	Level99AuraNormalEffects = []int{Level99, Level99Ground, Level99Aura1}
	// Level99AuraSimpleEffects are the white particles (EF_LEVEL99_3) of /aura off.
	Level99AuraSimpleEffects = []int{Level99Bubble}
	level99AuraAllEffects    = []int{Level99, Level99Ground, Level99Bubble, Level99Aura1}
)

func (m *System) SyncLevel99AuraEffects(ctx client.Context, now time.Time) {
	if m == nil || ctx.World == nil {
		return
	}
	if ctx.Session != nil {
		m.syncActorLevel99AuraEffects(ctx, scene.LocalSkillTarget(ctx), ctx.Session.Progress.BaseLevel, true, now)
	}
	for _, actor := range ctx.World.Actors {
		m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, now)
	}
}

func (m *System) SyncActorEntryLevel99Aura(ctx client.Context, entryID uint32) {
	if m == nil || ctx.World == nil || entryID == 0 {
		return
	}
	actor, ok := ctx.World.Actors[entryID]
	if !ok {
		m.RemoveLevel99AuraEffects(entryID)
		return
	}
	m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, time.Now())
}

func (m *System) syncActorLevel99AuraEffects(ctx client.Context, actorID uint32, level int, hasLevel bool, now time.Time) {
	if actorID == 0 {
		return
	}
	if !hasLevel {
		return
	}
	if actor, ok, _ := scene.ActorForCombatID(ctx, actorID); ok && scene.ActorHasStealth(actor) {
		m.RemoveLevel99AuraEffects(actorID)
		return
	}
	var wanted []int
	if level >= level99AuraLevel {
		job := 0
		if actor, ok, _ := scene.ActorForCombatID(ctx, actorID); ok {
			job = int(actor.Job)
		}
		wanted = level99AuraEffectIDs(ctx, job)
		for _, effectID := range wanted {
			if !m.HasActive(effectID, actorID, now) {
				m.addLevel99AuraEffect(ctx, effectID, actorID, now)
			}
		}
	}
	for _, effectID := range level99AuraAllEffects {
		if !level99AuraWantsEffect(wanted, effectID) {
			m.Remove(effectID, actorID)
		}
	}
}

func (m *System) addLevel99AuraEffect(ctx client.Context, effectID int, actorID uint32, now time.Time) bool {
	x, y, ok := EffectAnchor(ctx, actorID)
	if !ok {
		return false
	}
	return m.AddCellLifetime(ctx, effectID, actorID, x, y, now, Level99AuraLifetime, true)
}

func (m *System) RemoveLevel99AuraEffects(actorID uint32) {
	for _, effectID := range level99AuraAllEffects {
		m.Remove(effectID, actorID)
	}
}

// Classic job IDs that select the level-99 aura variant (CGameActor AM_99).
const (
	level99JobSuperNovice   = 23
	level99JobTreasureBox1  = 1324
	level99JobTreasureBox40 = 1363
	level99JobShecil        = 1638
	level99JobGRawrel       = 1663
	level99JobGTreasureBox  = 1732
	level99JobTreasureBox41 = 1938
	level99JobTreasureBoxI  = 1955
	level99Job2004JobBegin  = 4000
)

// level99AuraEffectIDs follows CGameActor::SendMsg(AM_99): /aura on gives the
// ring, ground glow and blue bubbles; /aura off gives white particles, or
// the blue bubbles for transcendent and later jobs, Super Novices and the
// jobs from Shecil on. Treasure boxes have no aura.
func level99AuraEffectIDs(ctx client.Context, job int) []int {
	simple := ctx.Config.Gameplay.SimpleAura
	if ctx.Session != nil {
		simple = ctx.Session.SimpleAura
	}
	simple = simple || lessEffectsEnabled(ctx)
	upper := job > level99Job2004JobBegin || job == level99JobSuperNovice
	if !upper && (job >= level99JobTreasureBox1 && job <= level99JobTreasureBox40 || job == level99JobGTreasureBox ||
		job >= level99JobTreasureBox41 && job <= level99JobTreasureBoxI) {
		return nil
	}
	if !simple {
		if !upper && job >= level99JobShecil && job <= level99JobGRawrel {
			return []int{Level99Aura1}
		}
		return Level99AuraNormalEffects
	}
	if upper || job >= level99JobShecil {
		return []int{Level99Aura1}
	}
	return Level99AuraSimpleEffects
}

func level99AuraWantsEffect(wanted []int, effectID int) bool {
	for _, wantedID := range wanted {
		if wantedID == effectID {
			return true
		}
	}
	return false
}

// drawLevel99AuraEffect ports the ring of CRagEffect::Level99 (a
// PP_3DCASTING primitive, CEffectPrim::Prim3DCasting and Render3DCasting):
// three bands that rise into a cone over 90 frames and rotate.
func (m *System) drawLevel99AuraEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if screen == nil || texture == nil {
		return
	}
	frame := max(0, int(now.Sub(effect.Starts)/db.EffectFrameDuration))
	// process counts frames from one; heights stop growing after 90.
	build := math.Sin(scene.DegreesToRadians(float64(min(frame+1, 90))))
	// alphaB starts at 120 and gains 10 per frame up to 180.
	alpha := uint8(min(180, 120+10*(frame+1)))
	tint := color.RGBA{R: 100, G: 100, B: 255, A: alpha}
	origin := scene.Point3{X: x, Y: z, Z: y}
	var vertices []render.Vertex3D
	var indices []uint16
	for band := 0; band < 3; band++ {
		appendLevel99AuraBand(&vertices, &indices, texture, origin, band, frame, build, tint)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

const level99AuraDivisions = 21 // E_DIVISION

func appendLevel99AuraBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, origin scene.Point3, band, frame int, build float64, tint color.RGBA) {
	// Level99: distance 3.9, height 15 and rise angle 55, each band 0.2
	// further out, 2 lower and 5 degrees flatter; all in client units.
	distance := 3.9 + 0.2*float64(band)
	maxHeight := 15 - 2*float64(band)
	rise := scene.DegreesToRadians(float64(55 - 5*band))
	rotation := (band*90 + (band+3)*(frame+1)) % 360
	const fullAngle = 315
	step := fullAngle / (level99AuraDivisions - 1)
	middle := (level99AuraDivisions - 1) / 2
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	var prevBase, prevTop scene.Point3
	order := 0
	for count := 0; count <= fullAngle; count += step {
		angle := scene.DegreesToRadians(float64((count + rotation) % 360))
		cosA, sinA := math.Cos(angle), math.Sin(angle)
		height := maxHeight * math.Sin(scene.DegreesToRadians(float64(90+(order-middle)*90/middle))) * build
		radial := math.Cos(rise) * height
		base := scene.Point3{X: origin.X + cosA*distance/5, Y: origin.Y, Z: origin.Z + sinA*distance/5}
		top := scene.Point3{X: origin.X + cosA*(distance+radial)/5, Y: origin.Y + math.Sin(rise)*height/5, Z: origin.Z + sinA*(distance+radial)/5}
		if order > 0 {
			part := order - 1
			u0 := float32(part) / level99AuraDivisions
			u1 := float32(part+1) / level99AuraDivisions
			first := uint16(len(*vertices))
			*vertices = append(*vertices,
				scene.TexturedSurfaceVertex3D(prevBase, scene.UV{U: u0, V: 1}, tint, w, h),
				scene.TexturedSurfaceVertex3D(base, scene.UV{U: u1, V: 1}, tint, w, h),
				scene.TexturedSurfaceVertex3D(top, scene.UV{U: u1, V: 0}, tint, w, h),
				scene.TexturedSurfaceVertex3D(prevTop, scene.UV{U: u0, V: 0}, tint, w, h),
			)
			*indices = append(*indices, first, first+1, first+2, first, first+2, first+3)
		}
		order = min(order+1, level99AuraDivisions-1)
		prevBase, prevTop = base, top
	}
}

// level99BubbleState ports CRagEffect::Level99_3 and CEffectPrim::Prim3DAura_2.
// Each primitive has four groups of four particles that rise from the feet
// 0.15 client units per frame, drift on randomly wandering angles once
// visible, and restart after rising 30 units. The aura launches two
// primitives, kept here as eight groups.
type level99BubbleState struct {
	frame, randomIndex int
	groups             []level99BubbleGroup
}

type level99BubbleGroup struct {
	// vecB_pre, vecB_now, vecT_pre, vecT_now in client units; negative Y is up.
	particles [4]scene.Point3
	// Angle and target pairs; particle p drifts by angles 4p and 4p+2.
	angles [16]int
}

const (
	level99BubblePrimitives = 2
	level99BubbleRise       = 0.15
	level99BubbleDrift      = 0.15
	level99BubbleReset      = -30
	// Particle quad radius in client units (m_GI.distance).
	level99BubbleRadiusBlue  = 2.4
	level99BubbleRadiusWhite = 0.8
)

func newLevel99BubbleState(effect Instance) *level99BubbleState {
	s := &level99BubbleState{frame: -1, groups: make([]level99BubbleGroup, 4*level99BubblePrimitives)}
	for i := range s.groups {
		g := &s.groups[i]
		for p := range g.particles {
			g.particles[p] = scene.Point3{Y: float64(s.random(effect, 100))}
			for k := 0; k < 4; k++ {
				g.angles[p*4+k] = s.random(effect, 360)
			}
		}
	}
	return s
}

func (s *level99BubbleState) random(effect Instance, n int) int {
	s.randomIndex++
	return HitEffectRandom(effect, 800+s.randomIndex, max(1, n))
}

func (s *level99BubbleState) advance(effect Instance, frame int) {
	// Discard suspended time instead of simulating hours of an aura.
	if frame-s.frame > 600 {
		s.frame = frame - 600
	}
	for s.frame < frame {
		s.frame++
		for i := range s.groups {
			g := &s.groups[i]
			for axis := 0; axis < len(g.angles); axis += 2 {
				angle, target := g.angles[axis], g.angles[axis+1]
				if target > angle {
					angle = min(359, angle+3)
					if angle > target {
						target = s.random(effect, angle)
					}
				} else {
					angle = max(0, angle-2)
					if angle <= target {
						target = 359 - s.random(effect, angle)
					}
				}
				g.angles[axis], g.angles[axis+1] = angle, target
			}
			for p := range g.particles {
				particle := &g.particles[p]
				if particle.Y < 0 {
					signX, signZ := level99BubbleDriftSigns(p)
					particle.X += signX * level99BubbleDrift * math.Sin(scene.DegreesToRadians(float64(g.angles[p*4])))
					particle.Z += signZ * level99BubbleDrift * math.Sin(scene.DegreesToRadians(float64(g.angles[p*4+2])))
					if particle.Y < level99BubbleReset {
						*particle = scene.Point3{Y: float64(s.random(effect, 100))}
					}
				}
				particle.Y -= level99BubbleRise
			}
		}
	}
}

// drawLevel99BubbleEffect ports CEffectPrim::Render3DAura_2: each visible
// particle is a screen-facing quad drawn twice, additively.
func (m *System) drawLevel99BubbleEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	s := effect.level99Bubbles
	if s == nil {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	s.advance(effect, int(now.Sub(effect.Starts)/db.EffectFrameDuration))
	radius, rgb := level99BubbleRadiusWhite, color.RGBA{R: 255, G: 255, B: 255}
	if effect.EffectID == Level99Aura1 {
		radius, rgb = level99BubbleRadiusBlue, color.RGBA{R: 80, G: 80, B: 255}
	}
	_, up, _, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return
	}
	size := radius * math.Sqrt2 / 5
	options := TexturedEffectBillboardDrawOptions(true, false)
	for _, g := range s.groups {
		for _, p := range g.particles {
			// Visibility and fading use the quad's first corner.
			cornerY := p.Y - up.Y*radius/math.Sqrt2
			if cornerY > 0 {
				continue
			}
			tint := rgb
			tint.A = uint8(scene.ClampFloat(250+math.Min(0, cornerY+20)*25, 0, 250))
			if tint.A == 0 {
				continue
			}
			for range 2 {
				DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, x+p.X/5, y+p.Z/5, z+component.PosZ-p.Y/5,
					size, size, 0, tint, options)
			}
		}
	}
}

func level99BubbleDriftSigns(anchor int) (float64, float64) {
	switch anchor % 4 {
	case 1:
		return -1, -1
	case 2:
		return 1, -1
	case 3:
		return -1, 1
	default:
		return 1, 1
	}
}

// drawLevel99GroundEffect ports CRagEffect::Level99_2 (a PP_3DAURA primitive,
// CEffectPrim::Prim3DAura and Render3DAura): two additive squares, turned
// 0 and 23 degrees, whose corners breathe between 12 and 15 client units
// every 120 frames, half a period apart.
func (m *System) drawLevel99GroundEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if screen == nil || texture == nil {
		return
	}
	frame := max(0, int(now.Sub(effect.Starts)/db.EffectFrameDuration))
	const distance = 15.0
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 200}
	var vertices []render.Vertex3D
	var indices []uint16
	for square := 0; square < 2; square++ {
		rise := scene.DegreesToRadians(float64((square*180 + 3*(frame+1)) % 360))
		radius := (distance*0.8 + distance*0.1*(math.Sin(rise)+1)) / 5
		var corners [4]scene.Point3
		for k := range corners {
			angle := scene.DegreesToRadians(float64(square*23 + k*90))
			// Render3DAura's corners run B_pre, B_now, T_now, T_pre with V
			// from 1 to 0; appendEffectQuad expects them from V = 0.
			corners[3-k] = scene.Point3{X: x + math.Cos(angle)*radius, Y: z + 1.0/5, Z: y + math.Sin(angle)*radius}
		}
		appendEffectQuad(&vertices, &indices, texture, corners, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
