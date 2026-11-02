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

type superAngel struct {
	origins [7]scene.Point3
	count   int
}

func superAngelAlpha(layer, birth, process int) int {
	if process <= 0 {
		return 0
	}
	if birth > 0 {
		return max(0, min(process, 40-birth)-3*max(0, process-75))
	}
	switch layer {
	case 0:
		return max(0, min(5*process, 245)-4*max(0, process-100))
	case 1:
		return max(0, min(5*process, 200)-6*max(0, process-75))
	case 2:
		return max(0, 5*min(process, 29)-3*max(0, process-30))
	default:
		return max(0, 5*min(process, 29)-3*max(0, process-60))
	}
}

func (m *System) drawSuperAngelEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if now.Before(effect.Starts) || effect.superAngel == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	state := effect.superAngel
	// Each wing echo keeps its spawn position when the player walks away.
	for state.count < len(state.origins) && frame >= state.count*3 {
		state.origins[state.count] = EffectWorldAnchor(ctx, effect, effect.Starts.Add(time.Duration(state.count*3)*db.EffectFrameDuration))
		state.count++
	}
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return
	}
	action := view.Act.Actions[0]
	count := 1
	if component.spriteFrame == 1 {
		count = state.count
	}
	for i := 0; i < count; i++ {
		process := frame - i*3 + 1
		alpha := superAngelAlpha(component.spriteFrame, i*3, process)
		if alpha == 0 {
			continue
		}
		motion := min(max(0, (process-30)/3), 29)
		if motion >= len(action.Animations) {
			continue
		}
		key := scene.SpriteBillboardKey{Motion: motion}
		billboard, ok := view.Billboards[key]
		if !ok {
			billboard, ok = scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
			if !ok {
				continue
			}
			view.Billboards[key] = billboard
		}
		p := state.origins[i]
		p.Y += (40 + .1*float64(process)) / 5
		options := scene.SpriteBillboardTriangleDrawOptions()
		options.Blend = render.BlendLighter
		scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, p.X, p.Z, p.Y,
			scene.PixelRatio, 0, float64(alpha)/255, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}

func (m *System) drawLevelUpBurst(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for group := 0; group < 2; group++ {
		birth := 5
		if effect.EffectID == Angel2 {
			birth = 65 + group
		}
		m.drawRadialBlades(screen, ctx, component.TextureFile, scene.Point3{X: x, Y: z, Z: y}, frame-birth+1, 2, group*45, 4)
	}
}

// PrimSlash1 is shared by the level-up bursts and Kaizel.
func (m *System) drawRadialBlades(screen *render.Frame, ctx client.Context, file string, origin scene.Point3, age int, initialHeight float64, rotation, blades int) {
	alpha := 5*min(age, 7) - 2*max(age-9, 0)
	if age <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, file)
	if texture == nil {
		return
	}
	inner, outer := float64(1+age)/5, float64(3+3*age)/5
	height := (initialHeight + .4*float64(min(age, int(math.Ceil((4-initialHeight)/.4))))) / 5
	var vertices []render.Vertex3D
	var indices []uint16
	for blade := 0; blade < blades; blade++ {
		for slice := 0; slice < 3; slice++ {
			sin, cos := math.Sincos(scene.DegreesToRadians(float64(rotation + blade*360/blades - 1 + slice)))
			low, high, a := height*.1, height*.4, alpha/3
			if slice == 1 {
				low, high, a = height*.2, height, alpha
			}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				{X: origin.X + cos*inner, Y: origin.Y + low, Z: origin.Z + sin*inner},
				{X: origin.X + cos*outer, Y: origin.Y + high, Z: origin.Z + sin*outer},
				{X: origin.X + cos*outer, Y: origin.Y, Z: origin.Z + sin*outer},
				{X: origin.X + cos*inner, Y: origin.Y, Z: origin.Z + sin*inner},
			}, color.RGBA{255, 255, 255, uint8(a)})
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
