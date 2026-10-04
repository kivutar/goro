package game

import (
	"math"

	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

type sceneProjection struct {
	playerX        float64
	playerY        float64
	playerZ        float64
	screenW        float64
	screenH        float64
	cameraYaw      float64
	cameraPitch    float64
	cameraZoom     float64
	viewProjection mat4
}

const (
	defaultSceneCameraPitch float64 = 230
	defaultSceneCameraYaw   float64 = 0
	defaultSceneCameraZoom  float64 = 125
	defaultSceneCameraFOV   float64 = 15
)

func newSceneProjectionForTargetYawPitchZoom(width, height int, targetX, targetY, targetZ, yaw, pitch, zoom float64) sceneProjection {
	projection := sceneProjection{
		playerX:     targetX,
		playerY:     targetY,
		playerZ:     targetZ,
		screenW:     float64(width),
		screenH:     float64(height),
		cameraYaw:   yaw,
		cameraPitch: clampCameraPitch(pitch),
		cameraZoom:  normalizeSceneCameraZoom(zoom),
	}
	projection.viewProjection = sceneCameraMatrixWithYawPitchZoom(float64(width), float64(height), targetX, targetY, targetZ, yaw, projection.cameraPitch, projection.cameraZoom)
	return projection
}

func (p sceneProjection) Project(x, y, z float64) screenPoint {
	point, ok := p.projectPoint(x, y, z)
	if !ok {
		return screenPoint{x: -1 << 20, y: -1 << 20}
	}
	return point
}

func (p sceneProjection) RenderCamera() render.Camera3D {
	return p.RenderCameraWithFog(sceneFog{})
}

func (p sceneProjection) RenderCameraWithFog(fog sceneFog) render.Camera3D {
	camera := render.Camera3D{Enabled: true}
	for i, value := range p.viewProjection {
		camera.ViewProjection[i] = float32(value)
	}
	if fog.enabled && fog.far > fog.near {
		camera.Fog = render.Fog3D{
			Enabled:  true,
			Near:     float32(fog.near),
			Far:      float32(fog.far),
			Strength: 1,
			ColorR:   float32(fog.color.R) / 255,
			ColorG:   float32(fog.color.G) / 255,
			ColorB:   float32(fog.color.B) / 255,
		}
	}
	return camera
}

func (p sceneProjection) BillboardBasis(x, y, z float64) (modelPoint3, modelPoint3, float64, bool) {
	if p.screenH <= 0 {
		return modelPoint3{}, modelPoint3{}, 0, false
	}
	eye, forward, right, up := p.cameraBasis()
	center := modelPoint3{x: x, y: z, z: y}
	cameraDepth := dot3(sub3(center, eye), forward)
	unitsPerPixel := 2 * cameraDepth * math.Tan(degreesToRadians(sceneCameraFOV())*0.5) / p.screenH
	if unitsPerPixel <= 0 || !isFinite(unitsPerPixel) {
		return modelPoint3{}, modelPoint3{}, 0, false
	}
	return right, up, unitsPerPixel, true
}

func (p sceneProjection) ScreenRay(mouseX, mouseY int) (modelPoint3, modelPoint3, bool) {
	if p.screenW <= 0 || p.screenH <= 0 {
		return modelPoint3{}, modelPoint3{}, false
	}
	eye, forward, right, up := p.cameraBasis()
	aspect := p.screenW / p.screenH
	tanHalfFOV := math.Tan(degreesToRadians(sceneCameraFOV()) * 0.5)
	ndcX := (float64(mouseX)/p.screenW)*2 - 1
	ndcY := 1 - (float64(mouseY)/p.screenH)*2
	dir := normalize3(add3(add3(forward, mul3(right, ndcX*tanHalfFOV*aspect)), mul3(up, ndcY*tanHalfFOV)))
	if dir == (modelPoint3{}) {
		return modelPoint3{}, modelPoint3{}, false
	}
	return eye, dir, true
}

func (p sceneProjection) cameraBasis() (eye, forward, right, up modelPoint3) {
	distance := normalizeSceneCameraZoom(p.cameraZoom) * 0.5
	pitch := clampCameraPitch(p.cameraPitch)
	if pitch > 180 {
		pitch -= 180
	}
	pitch = degreesToRadians(pitch)
	yaw := degreesToRadians(p.cameraYaw)
	horizontal := math.Cos(pitch) * distance
	target := modelPoint3{x: p.playerX, y: p.playerZ, z: p.playerY}
	eye = modelPoint3{
		x: target.x + math.Sin(yaw)*horizontal,
		y: target.y + math.Sin(pitch)*distance,
		z: target.z - math.Cos(yaw)*horizontal,
	}
	forward = normalize3(sub3(target, eye))
	right = normalize3(cross3(modelPoint3{y: 1}, forward))
	if right == (modelPoint3{}) {
		right = modelPoint3{x: 1}
	}
	up = normalize3(cross3(forward, right))
	return eye, forward, right, up
}

func (p sceneProjection) Depth(x, y, z float64) float64 {
	_, _, _, clipW := mat4TransformVec4(p.viewProjection, x, z, y, 1)
	if clipW <= 0 || !isFinite(clipW) {
		return math.Inf(-1)
	}
	return clipW
}

func (p sceneProjection) FogDepth(x, y, z float64) float64 {
	_, _, clipZ, clipW := mat4TransformVec4(p.viewProjection, x, z, y, 1)
	if clipW <= 0 || !finite4(clipZ, clipW, 0, 1) {
		return math.Inf(-1)
	}
	windowZ := (clipZ/clipW + 1) * 0.5
	return windowZ * clipW
}

func (p sceneProjection) VisibleForTriangle(x, y, z float64) bool {
	clipX, clipY, clipZ, clipW := mat4TransformVec4(p.viewProjection, x, z, y, 1)
	if clipW <= 1 || !finite4(clipX, clipY, clipZ, clipW) {
		return false
	}
	return clipZ >= -clipW && clipZ <= clipW
}

func (p sceneProjection) projectPoint(x, y, z float64) (screenPoint, bool) {
	clipX, clipY, _, clipW := mat4TransformVec4(p.viewProjection, x, z, y, 1)
	if clipW <= 0.001 || !finite4(clipX, clipY, clipW, 1) {
		return screenPoint{}, false
	}
	ndcX := clipX / clipW
	ndcY := clipY / clipW
	return screenPoint{
		x: float32((ndcX + 1) * p.screenW * 0.5),
		y: float32((1 - ndcY) * p.screenH * 0.5),
	}, true
}

func sceneCameraMatrixWithYawPitchZoom(width, height, targetX, targetY, targetZ, yawDegrees, pitchDegrees, zoom float64) mat4 {
	distance := normalizeSceneCameraZoom(zoom) * 0.5
	pitch := clampCameraPitch(pitchDegrees)
	if pitch > 180 {
		pitch -= 180
	}
	pitch = degreesToRadians(pitch)
	yaw := degreesToRadians(yawDegrees)
	horizontal := math.Cos(pitch) * distance
	target := modelPoint3{x: targetX, y: targetZ, z: targetY}
	eye := modelPoint3{
		x: target.x + math.Sin(yaw)*horizontal,
		y: target.y + math.Sin(pitch)*distance,
		z: target.z - math.Cos(yaw)*horizontal,
	}
	view := mat4LookAt(eye, target, modelPoint3{y: 1})
	aspect := 1.0
	if height > 0 {
		aspect = width / height
	}
	return mat4Multiply(mat4Perspective(degreesToRadians(sceneCameraFOV()), aspect, 1, 1000), view)
}

func mat4LookAt(eye, target, up modelPoint3) mat4 {
	forward := normalize3(modelPoint3{x: target.x - eye.x, y: target.y - eye.y, z: target.z - eye.z})
	right := normalize3(cross3(up, forward))
	if right == (modelPoint3{}) {
		right = modelPoint3{x: 1}
	}
	cameraUp := cross3(forward, right)
	out := mat4Identity()
	out[0], out[1], out[2] = right.x, cameraUp.x, -forward.x
	out[4], out[5], out[6] = right.y, cameraUp.y, -forward.y
	out[8], out[9], out[10] = right.z, cameraUp.z, -forward.z
	out[12] = -dot3(right, eye)
	out[13] = -dot3(cameraUp, eye)
	out[14] = dot3(forward, eye)
	return out
}

func dot3(a, b modelPoint3) float64 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func mat4Perspective(fovy, aspect, near, far float64) mat4 {
	f := 1 / math.Tan(fovy*0.5)
	out := mat4{}
	out[0] = f / aspect
	out[5] = f
	out[10] = (far + near) / (near - far)
	out[11] = -1
	out[14] = (2 * far * near) / (near - far)
	return out
}

func mat4TransformVec4(matrix mat4, x, y, z, w float64) (float64, float64, float64, float64) {
	return matrix[0]*x + matrix[4]*y + matrix[8]*z + matrix[12]*w,
		matrix[1]*x + matrix[5]*y + matrix[9]*z + matrix[13]*w,
		matrix[2]*x + matrix[6]*y + matrix[10]*z + matrix[14]*w,
		matrix[3]*x + matrix[7]*y + matrix[11]*z + matrix[15]*w
}

func finite4(a, b, c, d float64) bool {
	return isFinite(a) && isFinite(b) && isFinite(c) && isFinite(d)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func sceneCameraPitch() float64 {
	return defaultSceneCameraPitch
}

func sceneCameraYaw() float64 {
	return defaultSceneCameraYaw
}

func sceneCameraZoom() float64 {
	return normalizeSceneCameraZoom(defaultSceneCameraZoom)
}

func sceneCameraFOV() float64 {
	return defaultSceneCameraFOV
}

func normalizeSceneCameraZoom(zoom float64) float64 {
	return clampCameraZoom(zoom)
}

func terrainHeightAt(world *worldstate.World, x, y float64) float64 {
	if world == nil {
		return 0
	}
	if world.GAT != nil {
		cellX := int(math.Floor(x + 0.5))
		cellY := int(math.Floor(y + 0.5))
		cell, ok := world.GAT.Cell(cellX, cellY)
		if ok {
			return bilinearHeight(cell.Heights, x+0.5-float64(cellX), y+0.5-float64(cellY))
		}
	}
	if world.GND != nil {
		gridX := (x + 0.5) * 0.5
		gridY := (y + 0.5) * 0.5
		cellX := int(math.Floor(gridX))
		cellY := int(math.Floor(gridY))
		cell, ok := world.GND.Cell(cellX, cellY)
		if ok {
			return bilinearHeight(cell.Heights, gridX-float64(cellX), gridY-float64(cellY))
		}
	}
	return 0
}

func cellCenter(value float64) float64 {
	return value + 0.5
}

func bilinearHeight(heights [4]float32, x, y float64) float64 {
	if x < 0 {
		x = 0
	} else if x > 1 {
		x = 1
	}
	if y < 0 {
		y = 0
	} else if y > 1 {
		y = 1
	}
	x1 := float64(heights[0]) + (float64(heights[1])-float64(heights[0]))*x
	x2 := float64(heights[2]) + (float64(heights[3])-float64(heights[2]))*x
	return x1 + (x2-x1)*y
}
