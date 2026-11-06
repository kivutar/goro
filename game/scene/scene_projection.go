package scene

import (
	"math"

	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

type Projection struct {
	PlayerX        float64
	PlayerY        float64
	PlayerZ        float64
	ScreenW        float64
	ScreenH        float64
	CameraYaw      float64
	CameraPitch    float64
	CameraZoom     float64
	ViewProjection Mat4
}

const (
	DefaultSceneCameraPitch float64 = 230
	DefaultSceneCameraYaw   float64 = 0
	DefaultSceneCameraZoom  float64 = 125
	defaultSceneCameraFOV   float64 = 15
)

func (p Projection) Project(x, y, z float64) Point2 {
	point, ok := p.ProjectPoint(x, y, z)
	if !ok {
		return Point2{X: -1 << 20, Y: -1 << 20}
	}
	return point
}

func (p Projection) RenderCamera() render.Camera3D {
	return p.RenderCameraWithFog(SceneFog{})
}

func (p Projection) RenderCameraWithFog(fog SceneFog) render.Camera3D {
	camera := render.Camera3D{Enabled: true}
	for i, value := range p.ViewProjection {
		camera.ViewProjection[i] = float32(value)
	}
	if fog.Enabled && fog.Far > fog.Near {
		camera.Fog = render.Fog3D{
			Enabled:  true,
			Near:     float32(fog.Near),
			Far:      float32(fog.Far),
			Strength: 1,
			ColorR:   float32(fog.Color.R) / 255,
			ColorG:   float32(fog.Color.G) / 255,
			ColorB:   float32(fog.Color.B) / 255,
		}
	}
	return camera
}

func (p Projection) BillboardBasis(x, y, z float64) (Point3, Point3, float64, bool) {
	if p.ScreenH <= 0 {
		return Point3{}, Point3{}, 0, false
	}
	eye, forward, right, up := p.CameraBasis()
	center := Point3{X: x, Y: z, Z: y}
	cameraDepth := Dot3(Sub3(center, eye), forward)
	unitsPerPixel := 2 * cameraDepth * math.Tan(DegreesToRadians(SceneCameraFOV())*0.5) / p.ScreenH
	if unitsPerPixel <= 0 || !IsFinite(unitsPerPixel) {
		return Point3{}, Point3{}, 0, false
	}
	return right, up, unitsPerPixel, true
}

func (p Projection) ScreenRay(mouseX, mouseY int) (Point3, Point3, bool) {
	if p.ScreenW <= 0 || p.ScreenH <= 0 {
		return Point3{}, Point3{}, false
	}
	eye, forward, right, up := p.CameraBasis()
	aspect := p.ScreenW / p.ScreenH
	tanHalfFOV := math.Tan(DegreesToRadians(SceneCameraFOV()) * 0.5)
	ndcX := (float64(mouseX)/p.ScreenW)*2 - 1
	ndcY := 1 - (float64(mouseY)/p.ScreenH)*2
	dir := Normalize3(Add3(Add3(forward, Mul3(right, ndcX*tanHalfFOV*aspect)), Mul3(up, ndcY*tanHalfFOV)))
	if dir == (Point3{}) {
		return Point3{}, Point3{}, false
	}
	return eye, dir, true
}

func (p Projection) CameraBasis() (eye, forward, right, up Point3) {
	distance := NormalizeSceneCameraZoom(p.CameraZoom) * 0.5
	pitch := ClampCameraPitch(p.CameraPitch)
	if pitch > 180 {
		pitch -= 180
	}
	pitch = DegreesToRadians(pitch)
	yaw := DegreesToRadians(p.CameraYaw)
	horizontal := math.Cos(pitch) * distance
	target := Point3{X: p.PlayerX, Y: p.PlayerZ, Z: p.PlayerY}
	eye = Point3{
		X: target.X + math.Sin(yaw)*horizontal,
		Y: target.Y + math.Sin(pitch)*distance,
		Z: target.Z - math.Cos(yaw)*horizontal,
	}
	forward = Normalize3(Sub3(target, eye))
	right = Normalize3(Cross3(Point3{Y: 1}, forward))
	if right == (Point3{}) {
		right = Point3{X: 1}
	}
	up = Normalize3(Cross3(forward, right))
	return eye, forward, right, up
}

func (p Projection) Depth(x, y, z float64) float64 {
	_, _, _, clipW := mat4TransformVec4(p.ViewProjection, x, z, y, 1)
	if clipW <= 0 || !IsFinite(clipW) {
		return math.Inf(-1)
	}
	return clipW
}

func (p Projection) VisibleForTriangle(x, y, z float64) bool {
	clipX, clipY, clipZ, clipW := mat4TransformVec4(p.ViewProjection, x, z, y, 1)
	if clipW <= 1 || !finite4(clipX, clipY, clipZ, clipW) {
		return false
	}
	return clipZ >= -clipW && clipZ <= clipW
}

func (p Projection) ProjectPoint(x, y, z float64) (Point2, bool) {
	clipX, clipY, _, clipW := mat4TransformVec4(p.ViewProjection, x, z, y, 1)
	if clipW <= 0.001 || !finite4(clipX, clipY, clipW, 1) {
		return Point2{}, false
	}
	ndcX := clipX / clipW
	ndcY := clipY / clipW
	return Point2{
		X: float32((ndcX + 1) * p.ScreenW * 0.5),
		Y: float32((1 - ndcY) * p.ScreenH * 0.5),
	}, true
}

func Dot3(a, b Point3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func mat4TransformVec4(matrix Mat4, x, y, z, w float64) (float64, float64, float64, float64) {
	return matrix[0]*x + matrix[4]*y + matrix[8]*z + matrix[12]*w,
		matrix[1]*x + matrix[5]*y + matrix[9]*z + matrix[13]*w,
		matrix[2]*x + matrix[6]*y + matrix[10]*z + matrix[14]*w,
		matrix[3]*x + matrix[7]*y + matrix[11]*z + matrix[15]*w
}

func finite4(a, b, c, d float64) bool {
	return IsFinite(a) && IsFinite(b) && IsFinite(c) && IsFinite(d)
}

func IsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func SceneCameraYaw() float64 {
	return DefaultSceneCameraYaw
}

func SceneCameraFOV() float64 {
	return defaultSceneCameraFOV
}

func NormalizeSceneCameraZoom(zoom float64) float64 {
	return ClampCameraZoom(zoom)
}

func TerrainHeightAt(world *worldstate.World, x, y float64) float64 {
	if world == nil {
		return 0
	}
	if world.GAT != nil {
		cellX := int(math.Floor(x + 0.5))
		cellY := int(math.Floor(y + 0.5))
		cell, ok := world.GAT.Cell(cellX, cellY)
		if ok {
			return BilinearHeight(cell.Heights, x+0.5-float64(cellX), y+0.5-float64(cellY))
		}
	}
	if world.GND != nil {
		gridX := (x + 0.5) * 0.5
		gridY := (y + 0.5) * 0.5
		cellX := int(math.Floor(gridX))
		cellY := int(math.Floor(gridY))
		cell, ok := world.GND.Cell(cellX, cellY)
		if ok {
			return BilinearHeight(cell.Heights, gridX-float64(cellX), gridY-float64(cellY))
		}
	}
	return 0
}

func CellCenter(value float64) float64 {
	return value + 0.5
}

func BilinearHeight(heights [4]float32, x, y float64) float64 {
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
