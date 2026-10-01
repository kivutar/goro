package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawHitRingEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	distance, alpha := hitRingAnimation(component, now.Sub(effect.starts)-component.delay)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	// Hit1 captures 180 - master.roty; DirToDeg(dir) is 180 + dir*45.
	// Keep this orientation even if the target turns during the impact.
	yaw := -45 * float64(normalizeDirectionIndex(effect.actorDirection))
	right := rotateEffectCylinderVector(modelPoint3{x: 1}, component.angleX, yaw, 0)
	depth := rotateEffectCylinderVector(modelPoint3{z: 1}, component.angleX, yaw, 0)
	axis := rotateEffectCylinderVector(modelPoint3{y: 1}, component.angleX, yaw, 0)
	x += axis.x * distance
	y += axis.z * distance
	z += component.posZ + axis.y*distance
	// Prim3DCylinder advances U by 1/4 per face and uses RF_ALPHA.
	drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, x, y, z,
		component.bottomSize, component.topSize, component.height,
		effectComponentTint(component, alpha), component.circleSides, right, depth, axis,
		float64(component.circleSides)/4, render.BlendSourceOver)
}

func hitRingAnimation(component worldEffectComponent, elapsed time.Duration) (distance, alpha float64) {
	if elapsed < 0 || elapsed >= component.duration || component.frameDelay <= 0 {
		return 0, 0
	}
	frames := float64(component.duration / component.frameDelay)
	if frames < 2 {
		return 0, 0
	}
	// Sakexe Hit1/Prim3DCylinder: speed .7, acceleration -speed/(2*duration).
	// Acceleration and movement run before the first draw. One native unit is .2 cells.
	n := float64(elapsed/component.frameDelay + 1)
	const speed = 0.7 / 5
	acceleration := -speed / (2 * frames)
	distance = n*speed + acceleration*n*(n+1)/2
	// ProcessAlpha starts fading at stateCnt=5, reaching zero at stateCnt=9.
	alpha = component.alphaMax * clampFloat((frames-n)/(frames/2), 0, 1)
	return distance, alpha
}
