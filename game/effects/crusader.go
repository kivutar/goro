package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

// Defender draws eight 40-degree faces, sampling the first nine native heights.
func (m *System) drawCrusaderDefenseEffect(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := min(n, 80) - 5*max(0, n-179)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rotationSpeed, waveSpeed := 1+i%2, 1+i/2
		height := float64(40-i) / 5 * (1 + .3*math.Sin(scene.DegreesToRadians(float64(n*waveSpeed+i*90))))
		appendEffectBand(&vertices, &indices, texture, origin, (8-.1*float64(i))/5,
			scene.DegreesToRadians(float64(180+90*i+n*rotationSpeed)), scene.DegreesToRadians(320), 0, 8, color.RGBA{255, 255, 255, uint8(alpha)},
			func(j int) float64 { return height * math.Sin(scene.DegreesToRadians(float64(j*9))) })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}

func (m *System) drawProvidenceEffect(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 40 - max(0, n-50)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rise := scene.DegreesToRadians(float64(83 - i))
		height := 24 * math.Sin(scene.DegreesToRadians(float64(min(n, 90)))) * math.Sin(rise)
		appendEffectBand(&vertices, &indices, texture, origin, (6+.5*float64(i))/5,
			scene.DegreesToRadians(float64(90*i+n)), 2*math.Pi, 1/math.Tan(rise), 20, color.RGBA{255, 255, 255, uint8(alpha)},
			func(int) float64 { return height })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

func (m *System) drawGrandCrossEffect(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	origin := effect.ActorOrigin
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		alpha := min(n, 9)*10 - max(0, n-9)
		if alpha > 0 {
			vertices, indices := []render.Vertex3D{}, []uint16{}
			for i, offset := range []scene.Point3{{X: 4.6, Z: 4.6}, {X: 4.6, Z: -4.6}, {X: -4.6, Z: -4.6}, {X: -4.6, Z: 4.6}} {
				height := 24 * math.Sin(scene.DegreesToRadians(float64(n)))
				appendEffectBand(&vertices, &indices, texture, scene.Add3(origin, offset), (19.9-.01*float64(n))/5,
					scene.DegreesToRadians(float64(180-90*i)), scene.DegreesToRadians(80), 0, 8, color.RGBA{255, 175, 175, uint8(alpha)}, func(int) float64 { return height })
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
		}
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	alpha := min(n, 9)*9 - max(0, n-9)
	if texture == nil || alpha <= 0 {
		return
	}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	floorVertices, floorIndices := []render.Vertex3D{}, []uint16{}
	length, width := (24+.005*float64(n))/5, (3.2+.005*float64(n))/5
	height := 12 * math.Sin(scene.DegreesToRadians(float64(n)))
	for arm := 0; arm < 2; arm++ {
		corners := [4]scene.Point3{{X: -length, Z: -width}, {X: length, Z: -width}, {X: length, Z: width}, {X: -length, Z: width}}
		for i := range corners {
			if arm == 1 {
				corners[i].X, corners[i].Z = -corners[i].Z, corners[i].X
			}
			corners[i] = scene.Add3(origin, corners[i])
			corners[i].Y += .2
		}
		for repeat := 0; repeat < 2; repeat++ {
			appendEffectQuad(&floorVertices, &floorIndices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha / 2)})
		}
		for i, bottom := range corners {
			next := corners[(i+1)%4]
			appendEffectQuad(&vertices, &indices, texture,
				[4]scene.Point3{scene.Add3(bottom, scene.Point3{Y: height}), scene.Add3(next, scene.Point3{Y: height}), next, bottom},
				color.RGBA{255, 255, 255, uint8(alpha)})
		}
	}
	screen.DrawTriangles3DOwned(floorVertices, floorIndices, texture, TexturedEffectBillboardDrawOptions(true, false))
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}

func (m *System) ApplyActorDevotion(ctx client.Context, notify network.ActorDevotion) {
	active := m.Active[:0]
	for _, effect := range m.Active {
		if effect.EffectID != LineLink || effect.ActorID != notify.SourceID {
			active = append(active, effect)
		}
	}
	m.Active = active
	now := time.Now()
	for i, targetID := range notify.Targets {
		if targetID == 0 || targetID == notify.SourceID {
			continue
		}
		duplicate := false
		for _, previous := range notify.Targets[:i] {
			duplicate = duplicate || previous == targetID
		}
		if duplicate {
			continue
		}
		m.AddPersistentBetween(ctx, LineLink, notify.SourceID, targetID, now)
	}
}

func (m *System) drawDevotionLinkEffect(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); !ok {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	source := EffectWorldAnchor(ctx, effect, now)
	target := EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, now)
	source.Y += 1.6
	target.Y += 1.6
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	target = scene.Add3(source, scene.Mul3(scene.Sub3(target, source), math.Sin(scene.DegreesToRadians(float64(min(n, 15)*6)))))
	angle := math.Atan2(target.Z-source.Z, target.X-source.X) + math.Pi/2
	pulse := n % 10
	alpha := 45 + 2*min(pulse, 4) - 2*max(0, pulse-4)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for layer := 0; layer < 2; layer++ {
		radius := .08 * float64(layer+1)
		tint := color.RGBA{255, 255, 0, uint8(alpha)}
		if layer == 1 {
			tint = color.RGBA{0, 0, 255, uint8(alpha * 2)}
		}
		offset := func(j int) scene.Point3 {
			a := float64(j) * math.Pi / 5
			r := radius * math.Cos(a)
			return scene.Point3{X: r * math.Cos(angle), Y: radius * math.Sin(a), Z: r * math.Sin(angle)}
		}
		for j := 0; j < 10; j++ {
			a, b := offset(j), offset(j+1)
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				scene.Add3(target, a), scene.Add3(target, b), scene.Add3(source, b), scene.Add3(source, a),
			}, tint)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
