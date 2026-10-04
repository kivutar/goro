package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const (
	defaultCameraFollowLerpPerMS   = 0.006
	defaultCameraWheelZoomUnits    = 15
	defaultCameraPinchZoomScale    = 240
	defaultCameraMinZoom           = 65.0
	defaultCameraMaxZoom           = 165.0
	defaultCameraMinPitch          = 205.0
	defaultCameraMaxPitch          = 245.0
	defaultCameraPitchDragPerPixel = 0.3
)

type followCamera struct {
	initialized bool
	x           float64
	y           float64
	z           float64
	lastUpdate  time.Time
	yawOffset   float64
	pitch       float64
	zoom        float64
	zoomTarget  float64
}

func (c *followCamera) ResetTracking() {
	yawOffset, pitch, zoom, zoomTarget := c.yawOffset, c.pitch, c.zoom, c.zoomTarget
	*c = followCamera{
		yawOffset:  yawOffset,
		pitch:      pitch,
		zoom:       zoom,
		zoomTarget: zoomTarget,
	}
}

func (c *followCamera) Update(ctx client.Context, now time.Time) {
	targetX, targetY, targetZ := playerCameraTarget(ctx.World, now)
	if !c.initialized {
		c.x = targetX
		c.y = targetY
		c.z = targetZ
		c.zoom = c.targetZoom()
		c.lastUpdate = now
		c.initialized = true
		c.store(ctx)
		return
	}
	delta := now.Sub(c.lastUpdate)
	lerp := cameraFollowLerp(delta)
	c.lastUpdate = now
	c.x += (targetX - c.x) * lerp
	c.y += (targetY - c.y) * lerp
	c.z += (targetZ - c.z) * lerp
	zoomLerp := cameraZoomLerp(lerp)
	c.zoom += (c.targetZoom() - c.currentZoom()) * zoomLerp
	c.store(ctx)
}

func cameraFollowLerp(delta time.Duration) float64 {
	if delta <= 0 {
		return 0
	}
	return math.Min(float64(delta)/float64(time.Millisecond)*defaultCameraFollowLerpPerMS, 1.0)
}

func (c *followCamera) Rotate(delta float64) {
	c.yawOffset = normalizeCameraYaw(c.yawOffset + delta)
}

func (c *followCamera) Tilt(delta float64) {
	if delta == 0 || !isFinite(delta) {
		return
	}
	c.pitch = clampCameraPitch(c.currentPitch() + delta)
}

func (c *followCamera) currentPitch() float64 {
	if c.pitch == 0 || !isFinite(c.pitch) {
		return sceneCameraPitch()
	}
	return clampCameraPitch(c.pitch)
}

func (c *followCamera) ZoomBy(factor float64) {
	if factor <= 0 || !isFinite(factor) {
		return
	}
	c.zoomTarget = clampCameraZoom(c.targetZoom() * factor)
}

func (c *followCamera) ZoomByDelta(delta float64) {
	if delta == 0 || !isFinite(delta) {
		return
	}
	c.zoomTarget = clampCameraZoom(c.targetZoom() + delta)
}

func (c *followCamera) currentZoom() float64 {
	if c.zoom <= 0 || !isFinite(c.zoom) {
		return c.targetZoom()
	}
	return clampCameraZoom(c.zoom)
}

func (c *followCamera) targetZoom() float64 {
	if c.zoomTarget <= 0 || !isFinite(c.zoomTarget) {
		if c.zoom > 0 && isFinite(c.zoom) {
			return clampCameraZoom(c.zoom)
		}
		return sceneCameraZoom()
	}
	return clampCameraZoom(c.zoomTarget)
}

func cameraZoomLerp(followLerp float64) float64 {
	if followLerp <= 0 {
		return 0
	}
	return math.Min(followLerp*2, 1.0)
}

func (c *followCamera) ResetRotation() {
	c.yawOffset = 0
}

func (c *followCamera) Projection(ctx client.Context, width, height int, now time.Time) sceneProjection {
	return c.ProjectionWithOffset(ctx, width, height, now, 0, 0)
}

func (c *followCamera) ProjectionWithOffset(ctx client.Context, width, height int, now time.Time, offsetX, offsetY float64) sceneProjection {
	if !c.initialized {
		c.Update(ctx, now)
	}
	c.store(ctx)
	yawOffset := c.yawOffset
	if cameraRotationLockedForMap(ctx) {
		yawOffset = 0
	}
	outdoorZoom := c.currentZoom()
	projectedZoom := cameraZoomForMap(ctx, outdoorZoom)
	projectedPitch := c.currentPitch()
	if cameraRotationLockedForMap(ctx) {
		projectedPitch = sceneCameraPitch()
	}
	return newSceneProjectionForTargetYawPitchZoom(width, height, c.x+offsetX, c.y+offsetY, c.z, cameraYawForMap(ctx)+yawOffset, projectedPitch, projectedZoom)
}

func (c *followCamera) store(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	ctx.World.Camera.X = c.x
	ctx.World.Camera.Y = c.y
}

func (m *WorldMode) sceneProjection(ctx client.Context, width, height int, now time.Time) sceneProjection {
	offsetX, offsetY := m.cameraShakeOffset(now)
	return m.camera.ProjectionWithOffset(ctx, width, height, now, offsetX, offsetY)
}

func (m *WorldMode) startCameraShake(starts time.Time, duration time.Duration) {
	if duration <= 0 {
		return
	}
	m.cameraShakeStart = starts
	m.cameraShakeEnd = starts.Add(duration)
}

func (m *WorldMode) cameraShakeOffset(now time.Time) (float64, float64) {
	if m.cameraShakeStart.IsZero() || now.Before(m.cameraShakeStart) || !now.Before(m.cameraShakeEnd) {
		return 0, 0
	}
	duration := m.cameraShakeEnd.Sub(m.cameraShakeStart)
	if duration <= 0 {
		return 0, 0
	}
	elapsed := now.Sub(m.cameraShakeStart)
	progress := clampFloat(float64(elapsed)/float64(duration), 0, 1)
	amplitude := 0.18 * (1 - progress)
	seconds := float64(elapsed) / float64(time.Second)
	return math.Sin(seconds*120*2*math.Pi) * amplitude, math.Cos(seconds*150*2*math.Pi) * amplitude
}

func (m *WorldMode) updateCameraRotation(ctx client.Context) {
	if cameraRotationLockedForMap(ctx) {
		return
	}
	delta := 0.0
	if ctx.Input.MousePressed(input.MouseButtonRight) {
		screenW, _ := ctx.ScreenSize()
		if ctx.Input.Pressed(input.KeyShift) {
			m.camera.Tilt(cameraDragPitchDelta(ctx.Input.MouseDY))
			return
		}
		delta = cameraDragYawDelta(ctx.Input.MouseDX, screenW)
	}
	if delta != 0 {
		m.camera.Rotate(delta)
	}
}

func (m *WorldMode) updateCameraZoom(ctx client.Context) {
	if cameraZoomLockedForMap(ctx) {
		return
	}
	factor := 1.0
	if ctx.Input.WheelY != 0 {
		m.camera.ZoomByDelta(cameraWheelZoomDelta(ctx.Input.WheelY))
	}
	if ctx.Input.PinchDelta != 0 {
		factor *= cameraPinchZoomFactor(ctx.Input.PinchDelta)
	}
	if factor != 1 {
		m.camera.ZoomBy(factor)
	}
}

func playerCameraTarget(world *worldstate.World, now time.Time) (float64, float64, float64) {
	if world == nil {
		return 0.5, 0.5, 0
	}
	playerX, playerY := actorRenderPosition(world.Player, now)
	return cellCenter(playerX), cellCenter(playerY), cameraTargetHeightAt(world, playerX, playerY)
}

func cameraTargetHeightAt(world *worldstate.World, x, y float64) float64 {
	return terrainHeightAt(world, x, y)
}

func cameraYawForMap(ctx client.Context) float64 {
	if viewPoint, ok := lockedCameraViewPointForMap(ctx); ok {
		return -float64(viewPoint.InitialLongitude)
	}
	if cameraRotationLockedForMap(ctx) {
		return -45
	}
	return sceneCameraYaw()
}

func cameraRotationLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if _, ok := lockedCameraViewPointForMap(ctx); ok {
		return true
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func lockedCameraViewPointForMap(ctx client.Context) (res.CameraViewPoint, bool) {
	if ctx.Resources == nil || ctx.World == nil {
		return res.CameraViewPoint{}, false
	}
	viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName)
	if !ok || !viewPoint.LocksLongitude() {
		return res.CameraViewPoint{}, false
	}
	return viewPoint, true
}

func cameraZoomForMap(ctx client.Context, outdoorZoom float64) float64 {
	if ctx.Resources == nil || ctx.World == nil {
		return clampCameraZoom(outdoorZoom)
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok && viewPoint.DistanceScope <= 0 {
		if viewPoint.InitialDistance > 0 {
			return clampCameraZoom(float64(viewPoint.InitialDistance))
		}
		return sceneCameraZoom()
	}
	if ctx.Resources.IsIndoorMap(ctx.World.MapName) {
		return sceneCameraZoom()
	}
	return clampCameraZoom(outdoorZoom)
}

func cameraZoomLockedForMap(ctx client.Context) bool {
	if ctx.Resources == nil || ctx.World == nil {
		return false
	}
	if viewPoint, ok := ctx.Resources.CameraViewPoint(ctx.World.MapName); ok {
		return viewPoint.DistanceScope <= 0
	}
	return ctx.Resources.IsIndoorMap(ctx.World.MapName)
}

func cameraDragYawDelta(mouseDX, screenWidth int) float64 {
	if mouseDX == 0 || screenWidth <= 0 {
		return 0
	}
	return -(float64(mouseDX) / float64(screenWidth)) * 720
}

func cameraDragPitchDelta(mouseDY int) float64 {
	return float64(mouseDY) * defaultCameraPitchDragPerPixel
}

func cameraWheelZoomDelta(wheelY float64) float64 {
	if wheelY == 0 || !isFinite(wheelY) {
		return 0
	}
	return -math.Copysign(cameraZoomWheelUnits(), wheelY)
}

func cameraPinchZoomFactor(delta float64) float64 {
	if delta == 0 || !isFinite(delta) {
		return 1
	}
	return math.Exp(-delta / cameraPinchZoomScale())
}

func cameraZoomWheelUnits() float64 {
	return defaultCameraWheelZoomUnits
}

func cameraPinchZoomScale() float64 {
	return defaultCameraPinchZoomScale
}

func clampCameraZoom(zoom float64) float64 {
	if !isFinite(zoom) || zoom <= 0 {
		zoom = defaultSceneCameraZoom
	}
	return math.Max(defaultCameraMinZoom, math.Min(defaultCameraMaxZoom, zoom))
}

func clampCameraPitch(pitch float64) float64 {
	if !isFinite(pitch) || pitch == 0 {
		pitch = defaultSceneCameraPitch
	}
	return math.Max(defaultCameraMinPitch, math.Min(defaultCameraMaxPitch, pitch))
}

func normalizeCameraYaw(yaw float64) float64 {
	yaw = math.Mod(yaw, 360)
	if yaw <= -180 {
		yaw += 360
	}
	if yaw > 180 {
		yaw -= 360
	}
	return yaw
}
