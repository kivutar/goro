package game

import (
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestFollowCameraInitializesToRenderedPlayerPosition(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{
		X:            20,
		Y:            30,
		Moving:       true,
		FromX:        10,
		FromY:        20,
		ToX:          20,
		ToY:          30,
		MoveStarted:  now.Add(-750 * time.Millisecond),
		MoveDuration: 1500 * time.Millisecond,
	}
	ctx := client.Context{World: world}

	camera := followCamera{}
	camera.Update(ctx, now)

	if camera.x != 15.5 || camera.y != 25.5 {
		t.Fatalf("camera target = %.2f, %.2f, want rendered player center 15.5, 25.5", camera.x, camera.y)
	}
	if world.Camera.X != camera.x || world.Camera.Y != camera.y {
		t.Fatalf("world camera = %.2f, %.2f, want %.2f, %.2f", world.Camera.X, world.Camera.Y, camera.x, camera.y)
	}
}

func TestFollowCameraEasesTowardRenderedPlayerLikeReferenceView(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	camera := followCamera{}
	now := time.Now()
	camera.Update(ctx, now)
	world.Player = worldstate.Actor{X: 14, Y: 20}
	camera.Update(ctx, now.Add(time.Second/60))

	if math.Abs(camera.x-10.9) > 0.001 || camera.y != 20.5 {
		t.Fatalf("camera target = %.3f, %.3f, want 10.900, 20.5", camera.x, camera.y)
	}
}

func TestCameraFollowLerpClampsLikeReferenceView(t *testing.T) {
	if got := cameraFollowLerp(100 * time.Millisecond); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("camera lerp = %.2f, want 0.60", got)
	}
	if got := cameraFollowLerp(time.Second); got != 1 {
		t.Fatalf("camera lerp = %.2f, want clamped 1.00", got)
	}
}

func TestCameraZoomLerpMatchesRobrowserZoomCurve(t *testing.T) {
	if got := cameraZoomLerp(cameraFollowLerp(time.Second / 60)); math.Abs(got-0.2) > 0.001 {
		t.Fatalf("zoom lerp = %.3f, want 0.200", got)
	}
	if got := cameraZoomLerp(cameraFollowLerp(100 * time.Millisecond)); got != 1 {
		t.Fatalf("large zoom lerp = %.2f, want clamped 1.00", got)
	}
}

func TestFollowCameraZoomEasesTowardTargetLikeRobrowser(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	ctx := client.Context{World: world}

	now := time.Now()
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, zoom: 125, zoomTarget: 125, lastUpdate: now}
	camera.ZoomByDelta(-15)

	if got := camera.currentZoom(); got != 125 {
		t.Fatalf("current zoom before update = %.1f, want unchanged 125.0", got)
	}
	if got := camera.targetZoom(); got != 110 {
		t.Fatalf("target zoom = %.1f, want 110.0", got)
	}

	camera.Update(ctx, now.Add(time.Second/60))
	if math.Abs(camera.currentZoom()-122) > 0.01 {
		t.Fatalf("smoothed zoom = %.2f, want 122.00", camera.currentZoom())
	}
}

func TestAppendActorDrawEntryUsesPathRenderDirection(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	actor := worldstate.Actor{
		X:            2,
		Y:            1,
		Dir:          4,
		Moving:       true,
		FromX:        0,
		FromY:        0,
		ToX:          2,
		ToY:          1,
		MoveStarted:  now.Add(-225 * time.Millisecond),
		MoveDuration: 450 * time.Millisecond,
		MovePath: []worldstate.WalkStep{
			{X: 0, Y: 0},
			{X: 0, Y: 1},
			{X: 1, Y: 1},
			{X: 2, Y: 1},
		},
	}
	projection := newSceneProjectionForTarget(800, 600, 0.5, 1.5, 0)

	entries := appendActorDrawEntry(nil, world, projection, actor, false, now, 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got, want := entries[0].actor.Dir, directionFromDelta(0, 1, 1, 1, 4); got != want {
		t.Fatalf("entry direction = %d, want %d", got, want)
	}
}

func TestActorBillboardSortDepthUsesTopInCameraProjection(t *testing.T) {
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	footDepth := projection.Depth(10.5, 20.5, 0)
	topDepth := projection.Depth(10.5, 20.5, actorBillboardWorldHeightUnit)
	got := actorBillboardSortDepth(projection, 10.5, 20.5, 0)
	want := math.Min(footDepth, topDepth)
	if got != want {
		t.Fatalf("billboard depth = %.4f, want closer of foot %.4f and top %.4f", got, footDepth, topDepth)
	}
	if got >= footDepth {
		t.Fatalf("billboard depth = %.4f, want closer than foot depth %.4f", got, footDepth)
	}
}

func TestCameraYawForIndoorMapIsLocked(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
	}
	if got := cameraYawForMap(ctx); got != -45 {
		t.Fatalf("indoor camera yaw = %.1f, want -45.0", got)
	}
	ctx.World.MapName = "prontera"
	if got := cameraYawForMap(ctx); got != defaultSceneCameraYaw {
		t.Fatalf("outdoor camera yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
}

func TestCameraYawForFixedViewPointMapIsLocked(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "viewpointtable.txt"), []byte("fixed_view.rsw#150#50#170#30#30#30#60#30#45#\nfree_view.rsw#150#50#170#-360#360#0#60#30#45#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "fixed_view"},
	}
	if got := cameraYawForMap(ctx); got != -30 {
		t.Fatalf("fixed viewpoint yaw = %.1f, want -30.0", got)
	}
	if !cameraRotationLockedForMap(ctx) {
		t.Fatal("fixed viewpoint should lock camera rotation")
	}
	ctx.World.MapName = "free_view"
	if got := cameraYawForMap(ctx); got != defaultSceneCameraYaw {
		t.Fatalf("free viewpoint yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
	if cameraRotationLockedForMap(ctx) {
		t.Fatal("free viewpoint should not lock camera rotation")
	}
}

func TestIndoorCameraZoomIsLockedWithoutLosingOutdoorZoom(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	world := worldstate.New()
	world.MapName = "geffen_in"
	ctx := client.Context{
		Resources: manager,
		World:     world,
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0, zoom: 150}

	indoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := indoorProjection.cameraZoom; got != sceneCameraZoom() {
		t.Fatalf("indoor projection zoom = %.1f, want %.1f", got, sceneCameraZoom())
	}

	ctx.World.MapName = "prontera"
	outdoorProjection := camera.Projection(ctx, 800, 600, time.Now())
	if got := outdoorProjection.cameraZoom; got != 150 {
		t.Fatalf("restored outdoor projection zoom = %.1f, want 150.0", got)
	}
}

func TestFollowCameraTrackingResetKeepsUserView(t *testing.T) {
	camera := followCamera{
		initialized: true,
		x:           10.5,
		y:           20.5,
		z:           3,
		lastUpdate:  time.Now(),
		yawOffset:   73,
		pitch:       245,
		zoom:        148,
		zoomTarget:  152,
	}

	camera.ResetTracking()

	if camera.initialized || camera.x != 0 || camera.y != 0 || camera.z != 0 || !camera.lastUpdate.IsZero() {
		t.Fatalf("tracking state was not reset: %+v", camera)
	}
	if camera.yawOffset != 73 || camera.pitch != 245 || camera.zoom != 148 || camera.zoomTarget != 152 {
		t.Fatalf("user view = yaw %.1f pitch %.1f zoom %.1f target %.1f", camera.yawOffset, camera.pitch, camera.zoom, camera.zoomTarget)
	}
}

func TestFollowCameraProjectionIncludesRuntimeYawOffset(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.Rotate(90)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != 90 {
		t.Fatalf("projection yaw = %.1f, want 90.0", got)
	}

	camera.ResetRotation()
	projection = camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != defaultSceneCameraYaw {
		t.Fatalf("reset projection yaw = %.1f, want %.1f", got, defaultSceneCameraYaw)
	}
}

func TestIndoorCameraYawIsLockedWithoutLosingOutdoorRotation(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
	}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.Rotate(90)
	camera.Tilt(30)
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != -45 {
		t.Fatalf("indoor projection yaw = %.1f, want -45.0", got)
	}
	if got := projection.cameraPitch; got != sceneCameraPitch() {
		t.Fatalf("indoor projection pitch = %.1f, want fixed %.1f", got, sceneCameraPitch())
	}
	if camera.yawOffset != 90 {
		t.Fatalf("stored outdoor yaw offset = %.1f, want 90.0", camera.yawOffset)
	}
	if got := camera.currentPitch(); got != defaultCameraMaxPitch {
		t.Fatalf("stored outdoor pitch = %.1f, want %.1f", got, defaultCameraMaxPitch)
	}

	ctx.World.MapName = "prontera"
	projection = camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraYaw; got != 90 {
		t.Fatalf("restored outdoor projection yaw = %.1f, want 90.0", got)
	}
	if got := projection.cameraPitch; got != defaultCameraMaxPitch {
		t.Fatalf("restored outdoor projection pitch = %.1f, want %.1f", got, defaultCameraMaxPitch)
	}
}

func TestCameraRotationIsDisabledOnIndoorMap(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.Mkdir(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "indoorrswtable.txt"), []byte("geffen_in.rsw#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	inputState := input.NewState()
	inputState.SetMousePosition(100, 100)
	inputState.SetMouseButton(input.MouseButtonRight, true)
	inputState.SetKey(input.KeyShift, true)
	inputState.SetMousePosition(200, 160)
	mode := &WorldMode{}
	mode.camera.Rotate(90)
	mode.camera.Tilt(15)
	ctx := client.Context{
		Resources: manager,
		World:     &worldstate.World{MapName: "geffen_in"},
		Input:     inputState,
		ScreenW:   800,
		ScreenH:   600,
	}

	mode.updateCameraRotation(ctx)
	if mode.camera.yawOffset != 90 {
		t.Fatalf("stored outdoor camera yaw offset = %.1f, want 90.0", mode.camera.yawOffset)
	}
	if got := mode.camera.currentPitch(); got != 245 {
		t.Fatalf("stored outdoor camera pitch = %.1f, want 245.0", got)
	}
}

func TestCameraDragYawDeltaMatchesRobrowserScale(t *testing.T) {
	if got := cameraDragYawDelta(100, 1000); got != -72 {
		t.Fatalf("drag yaw delta = %.1f, want -72.0", got)
	}
	if got := cameraDragYawDelta(-100, 1000); got != 72 {
		t.Fatalf("reverse drag yaw delta = %.1f, want 72.0", got)
	}
	if got := cameraDragYawDelta(100, 0); got != 0 {
		t.Fatalf("zero width drag yaw delta = %.1f, want 0", got)
	}
}

func TestCameraDragPitchDeltaMatchesReferenceClientScale(t *testing.T) {
	if got := cameraDragPitchDelta(60); got != 18 {
		t.Fatalf("drag pitch delta = %.1f, want 18.0", got)
	}
	if got := cameraDragPitchDelta(-60); got != -18 {
		t.Fatalf("reverse drag pitch delta = %.1f, want -18.0", got)
	}
}

func TestShiftRightDragTiltsCameraWithoutRotating(t *testing.T) {
	inputState := input.NewState()
	inputState.SetMousePosition(100, 100)
	inputState.SetMouseButton(input.MouseButtonRight, true)
	inputState.SetKey(input.KeyShift, true)
	inputState.SetMousePosition(200, 160)
	mode := &WorldMode{}
	mode.camera.Rotate(45)
	ctx := client.Context{
		World:   &worldstate.World{MapName: "prontera"},
		Input:   inputState,
		ScreenW: 800,
		ScreenH: 600,
	}

	mode.updateCameraRotation(ctx)

	if got := mode.camera.currentPitch(); got != defaultCameraMaxPitch {
		t.Fatalf("camera pitch = %.1f, want %.1f", got, defaultCameraMaxPitch)
	}
	if mode.camera.yawOffset != 45 {
		t.Fatalf("camera yaw = %.1f, want unchanged 45.0", mode.camera.yawOffset)
	}
}

func TestCameraPitchIsClampedToReferenceClientOutdoorLimits(t *testing.T) {
	camera := followCamera{}
	camera.Tilt(-1000)
	if got := camera.currentPitch(); got != defaultCameraMinPitch {
		t.Fatalf("minimum camera pitch = %.1f, want %.1f", got, defaultCameraMinPitch)
	}
	camera.Tilt(1000)
	if got := camera.currentPitch(); got != defaultCameraMaxPitch {
		t.Fatalf("maximum camera pitch = %.1f, want %.1f", got, defaultCameraMaxPitch)
	}
}

func TestCameraWheelZoomDeltaMatchesRobrowserStep(t *testing.T) {
	if got := cameraWheelZoomDelta(1); got != -15 {
		t.Fatalf("wheel up delta = %.1f, want -15", got)
	}
	if got := cameraWheelZoomDelta(-2); got != 15 {
		t.Fatalf("wheel down delta = %.1f, want 15", got)
	}
	if got := cameraWheelZoomDelta(0.25); got != -15 {
		t.Fatalf("trackpad wheel delta = %.1f, want one notch", got)
	}
}

func TestCameraZoomRangeMatchesRobrowserOutdoorDefaults(t *testing.T) {
	if got := sceneCameraZoom(); got != 125 {
		t.Fatalf("default zoom = %.1f, want reference client default 125", got)
	}
	if defaultCameraMinZoom != 65 || defaultCameraMaxZoom != 165 {
		t.Fatalf("zoom range = %.1f..%.1f, want goro outdoor 65..165", defaultCameraMinZoom, defaultCameraMaxZoom)
	}
}

func TestSceneFogDepthAtTargetMatchesRobrowserDefaultZoom(t *testing.T) {
	projection := newSceneProjectionForTargetYawZoom(1280, 720, 10.5, 20.5, 0, 0, defaultSceneCameraZoom)
	got := projection.FogDepth(10.5, 20.5, 0)
	const want = 1000 * ((defaultSceneCameraZoom * 0.5) - 1) / 999
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("target fog depth = %.3f, want %.3f", got, want)
	}
}

func TestCameraPinchZoomFactorZoomsInWhenFingersSpread(t *testing.T) {
	if got := cameraPinchZoomFactor(25); got >= 1 {
		t.Fatalf("pinch spread factor = %.3f, want zoom-in factor below 1", got)
	}
	if got := cameraPinchZoomFactor(-25); got <= 1 {
		t.Fatalf("pinch close factor = %.3f, want zoom-out factor above 1", got)
	}
}

func TestFollowCameraZoomIsClampedAndProjected(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{World: world}
	camera := followCamera{initialized: true, x: 10.5, y: 20.5, z: 0}

	camera.ZoomBy(0.1)
	if got := camera.currentZoom(); got != defaultCameraMinZoom {
		t.Fatalf("zoom in clamp = %.1f, want %.1f", got, defaultCameraMinZoom)
	}
	camera.ZoomBy(10)
	if got := camera.currentZoom(); got != defaultCameraMaxZoom {
		t.Fatalf("zoom out clamp = %.1f, want %.1f", got, defaultCameraMaxZoom)
	}
	projection := camera.Projection(ctx, 800, 600, time.Now())
	if got := projection.cameraZoom; got != defaultCameraMaxZoom {
		t.Fatalf("projection zoom = %.1f, want %.1f", got, defaultCameraMaxZoom)
	}
}

func TestCursorRotateInfoMatchesRobrowser(t *testing.T) {
	info := cursorInfo(cursorActionRotate)
	if info.delayMult != 1 {
		t.Fatalf("rotate cursor info = %+v", info)
	}
}

func TestWorldSceneClearColorMatchesReferenceDefaults(t *testing.T) {
	if got := worldSceneClearColor("geffen_in"); got != (color.RGBA{A: 255}) {
		t.Fatalf("default map clear color = %#v, want black", got)
	}
	if got := worldSceneClearColor("data/yuno.gat"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("yuno clear color = %#v", got)
	}
	if got := worldSceneClearColor("airplane_01"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("airplane_01 clear color = %#v", got)
	}
	if got := worldSceneClearColor("sch_gld"); got != (color.RGBA{R: 0x66, G: 0x99, B: 0xcc, A: 255}) {
		t.Fatalf("sch_gld clear color = %#v", got)
	}
	if got := worldSceneClearColor("bat_fild02"); got != (color.RGBA{A: 255}) {
		t.Fatalf("bat_fild02 clear color = %#v, want black", got)
	}
	if got := worldSceneClearColor("5@tower.rsw"); got != (color.RGBA{R: 0x33, G: 0x00, B: 0x33, A: 255}) {
		t.Fatalf("tower clear color = %#v", got)
	}
	if got := worldSceneClearColor("thana_boss.rsw"); got != (color.RGBA{R: 0xe0, G: 0xd4, B: 0xc2, A: 255}) {
		t.Fatalf("thana_boss clear color = %#v", got)
	}
}

func TestSceneLightingFromRSWMatchesReferenceDirection(t *testing.T) {
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 0,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Opacity:   1,
	}})
	want := modelPoint3{x: -math.Sqrt2 / 2, y: -math.Sqrt2 / 2, z: 0}
	if math.Abs(lighting.direction.x-want.x) > 0.0001 ||
		math.Abs(lighting.direction.y-want.y) > 0.0001 ||
		math.Abs(lighting.direction.z-want.z) > 0.0001 {
		t.Fatalf("light direction = %+v, want %+v", lighting.direction, want)
	}
}

func TestSceneLightingModelScaleIgnoresOpacityLikeRobrowser(t *testing.T) {
	opaque := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{0.3, 0.3, 0.3},
		Ambient:   [3]float32{0.4, 0.4, 0.4},
		Opacity:   1,
	}})
	half := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{0.3, 0.3, 0.3},
		Ambient:   [3]float32{0.4, 0.4, 0.4},
		Opacity:   0.5,
	}})
	normal := modelPoint3{x: 0, y: 1, z: 0}
	if got, want := half.modelScale(normal), opaque.modelScale(normal); got != want {
		t.Fatalf("model scale changed with opacity: got %+v want %+v", got, want)
	}
}

func TestSceneLightingModelScaleUsesReferenceMinimumLightWeight(t *testing.T) {
	lighting := sceneLighting{
		direction: modelPoint3{y: -1},
		diffuse:   modelPoint3{x: 1, y: 1, z: 1},
		ambient:   modelPoint3{},
		env:       modelPoint3{x: 1, y: 1, z: 1},
	}
	got := lighting.modelScale(modelPoint3{y: 1})
	want := modelPoint3{x: 0.5, y: 0.5, z: 0.5}
	if got != want {
		t.Fatalf("model scale = %+v, want %+v", got, want)
	}
}

func TestSceneLightingScaleClampsLightBeforeEnvLikeRobrowser(t *testing.T) {
	lighting := sceneLighting{
		direction: modelPoint3{y: -1},
		diffuse:   modelPoint3{x: 0.8, y: 0.8, z: 0.8},
		ambient:   modelPoint3{x: 0.8, y: 0.8, z: 0.8},
		env:       modelPoint3{x: 0.5, y: 0.5, z: 0.5},
	}
	got := lighting.groundScale(modelPoint3{y: -1})
	want := modelPoint3{x: 0.5, y: 0.5, z: 0.5}
	if got != want {
		t.Fatalf("ground scale = %+v, want %+v", got, want)
	}
}

func TestSmoothGNDTopNormalsKeepsFlatTilesUniform(t *testing.T) {
	gnd := testGNDWithTopHeights(2, 2, func(_, _ int) [4]float32 {
		return [4]float32{0, 0, 0, 0}
	})
	normals := buildSmoothGNDTopNormals(gnd)
	if len(normals) != 4 {
		t.Fatalf("normal count = %d, want 4", len(normals))
	}
	center := normals[0]
	for i := 1; i < 4; i++ {
		if !modelPointNear(center[0], center[i], 0.0001) {
			t.Fatalf("flat tile normals differ: %v vs %v", center[0], center[i])
		}
	}
}

func TestSmoothGNDTopNormalsVariesSlopedTileCorners(t *testing.T) {
	gnd := testGNDWithTopHeights(3, 3, func(x, y int) [4]float32 {
		return [4]float32{
			float32(x*y + y),
			float32(x*3 + y*y + 2),
			float32(x*x + y*4 + 1),
			float32(x*x + y*y + x*y + 7),
		}
	})
	normals := buildSmoothGNDTopNormals(gnd)
	tile := normals[1+1*gnd.Width]
	same := true
	for i := 1; i < 4; i++ {
		if !modelPointNear(tile[0], tile[i], 0.0001) {
			same = false
		}
	}
	if same {
		t.Fatalf("sloped tile normals are all equal: %+v", tile)
	}
}

func TestSurfaceVertexTintsUsePerVertexNormals(t *testing.T) {
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 45,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Ambient:   [3]float32{0, 0, 0},
		Opacity:   1,
	}})
	tints := surfaceVertexTints(uniformGNDSurfaceBaseTints(color.RGBA{}), [4]float32{}, [4]modelPoint3{
		{x: -0.5, y: -math.Sqrt2 / 2, z: -0.5},
		{x: 0.5, y: -math.Sqrt2 / 2, z: -0.5},
		{x: 0.5, y: -math.Sqrt2 / 2, z: 0.5},
		{x: -0.5, y: -math.Sqrt2 / 2, z: 0.5},
	}, lighting)
	if tints[0] == tints[1] && tints[0] == tints[2] && tints[0] == tints[3] {
		t.Fatalf("vertex tints are uniform: %+v", tints)
	}
}

func TestPosterizeGNDLightmapColorUsesReferenceClientBuckets(t *testing.T) {
	got := posterizeGNDLightmapColor(color.RGBA{R: 15, G: 31, B: 255, A: 77})
	want := color.RGBA{R: 0, G: 16, B: 240, A: 77}
	if got != want {
		t.Fatalf("posterized lightmap color = %+v, want %+v", got, want)
	}
}

func TestTopGNDSurfaceBaseTintsUseNeighborTileColors(t *testing.T) {
	gnd := &res.GND{
		Width:  2,
		Height: 2,
		Surfaces: []res.GNDSurface{
			{Color: color.RGBA{R: 10, G: 20, B: 30, A: 255}},
			{Color: color.RGBA{R: 40, G: 50, B: 60, A: 255}},
			{Color: color.RGBA{R: 70, G: 80, B: 90, A: 255}},
			{Color: color.RGBA{R: 100, G: 110, B: 120, A: 255}},
		},
		Cells: []res.GNDCell{
			{Top: 0, Front: -1, Right: -1},
			{Top: 1, Front: -1, Right: -1},
			{Top: 2, Front: -1, Right: -1},
			{Top: 3, Front: -1, Right: -1},
		},
	}
	tints := topGNDSurfaceBaseTints(gnd, 0, 0, color.RGBA{})
	want := [4]color.RGBA{
		{R: 10, G: 20, B: 30, A: 255},
		{R: 40, G: 50, B: 60, A: 255},
		{R: 100, G: 110, B: 120, A: 255},
		{R: 70, G: 80, B: 90, A: 255},
	}
	if tints != want {
		t.Fatalf("top GND tints = %+v, want %+v", tints, want)
	}
}

func testGNDWithTopHeights(width, height int, fn func(x, y int) [4]float32) *res.GND {
	gnd := &res.GND{
		Width:    width,
		Height:   height,
		Cells:    make([]res.GNDCell, width*height),
		Surfaces: []res.GNDSurface{{TextureID: 0, LightmapID: -1}},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gnd.Cells[x+y*width] = res.GNDCell{Top: 0, Front: -1, Right: -1, Heights: fn(x, y)}
		}
	}
	return gnd
}

func modelPointNear(a, b modelPoint3, epsilon float64) bool {
	return math.Abs(a.x-b.x) <= epsilon && math.Abs(a.y-b.y) <= epsilon && math.Abs(a.z-b.z) <= epsilon
}

func TestGNDDrawBoundsUseCameraFootprint(t *testing.T) {
	gnd := &res.GND{Width: 200, Height: 200}
	projection := newSceneProjectionForTargetYaw(1024, 768, 200.5, 260.5, 8, 0)
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, 1024, 768)
	if !ok {
		t.Fatal("missing GND bounds")
	}
	centerX := gndTileFromWorld(projection.playerX)
	centerY := gndTileFromWorld(projection.playerY)
	if startX > centerX || endX < centerX || startY > centerY || endY < centerY {
		t.Fatalf("bounds %d..%d,%d..%d do not include center %d,%d", startX, endX, startY, endY, centerX, centerY)
	}
	if endX-startX >= gnd.Width-1 || endY-startY >= gnd.Height-1 {
		t.Fatalf("camera bounds %d..%d,%d..%d should not cover the full map", startX, endX, startY, endY)
	}
}

func TestGNDShadowMapPointMatchesReferenceClientCellCenterMapping(t *testing.T) {
	x, y := gndShadowMapPoint(10, 20)
	if x != 42 || y != 82 {
		t.Fatalf("shadow map point for even cell = %d,%d, want 42,82", x, y)
	}
	x, y = gndShadowMapPoint(11, 21)
	if x != 46 || y != 86 {
		t.Fatalf("shadow map point for odd cell = %d,%d, want 46,86", x, y)
	}
}

func TestActorShadowFactorAveragesGNDLightmapAlpha(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 128
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	got := actorShadowFactor(&worldstate.World{GND: gnd}, 3, 3)
	want := float64(128) / 255
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("shadow factor = %.4f, want %.4f", got, want)
	}
}

func TestActorShadowFactorIgnoresGroundShadowBelowElevatedGAT(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 32
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}
	gat := &res.GAT{
		Width:  8,
		Height: 8,
		Cells:  make([]res.GATCell, 64),
	}
	for i := range gat.Cells {
		gat.Cells[i].Heights = [4]float32{2, 2, 2, 2}
	}

	got := actorShadowFactor(&worldstate.World{GAT: gat, GND: gnd}, 3, 3)
	if got != 1 {
		t.Fatalf("elevated GAT shadow factor = %.4f, want 1", got)
	}
}

func TestActorShadowFactorDefaultsToLitWithoutGroundLightmap(t *testing.T) {
	if got := actorShadowFactor(nil, 3, 3); got != 1 {
		t.Fatalf("nil world shadow = %.2f, want 1", got)
	}
	gnd := &res.GND{
		Width:    1,
		Height:   1,
		Surfaces: []res.GNDSurface{{LightmapID: 0}},
		Cells:    []res.GNDCell{{Top: -1}},
	}
	if got := actorShadowFactor(&worldstate.World{GND: gnd}, 0, 0); got != 1 {
		t.Fatalf("missing top surface shadow = %.2f, want 1", got)
	}
}

func TestQuadHasInvalidPointDetectsCameraSentinel(t *testing.T) {
	points := [4]screenPoint{{x: -1 << 20, y: -1 << 20}, {x: 1, y: 1}, {x: 2, y: 1}, {x: 1, y: 2}}
	if !quadHasInvalidPoint(points) {
		t.Fatal("expected camera sentinel point to invalidate GND quad")
	}
}

func TestMapWaterPrefersGNDOverride(t *testing.T) {
	gnd := &res.GND{Water: res.GNDWater{Present: true, Level: -4, Type: 3, WaveHeight: 2, WaveSpeed: 5, WavePitch: 20, AnimSpeed: 6}}
	rsw := &res.RSW{Water: res.RSWWater{Level: -1, Type: 1}}
	water, ok := mapWater(gnd, rsw)
	if !ok {
		t.Fatal("missing water")
	}
	if water.Level != -4 || water.Type != 3 || water.WaveHeight != 2 || water.AnimSpeed != 6 {
		t.Fatalf("water = %+v, want GND override", water)
	}
}

func TestWaterUVsUseFourTileRepeat(t *testing.T) {
	uvs := waterUVs(3, 4)
	if uvs[0] != (texturePoint{u: 0.75, v: 0}) || uvs[2] != (texturePoint{u: 1.0, v: 0.25}) {
		t.Fatalf("water uvs = %+v", uvs)
	}
}

func TestWaterVisibleForCellUsesInvertedHeightConvention(t *testing.T) {
	water := res.RSWWater{Level: -2, WaveHeight: 0.5}
	if !waterVisibleForCell(res.GNDCell{Heights: [4]float32{-3, -2, -2, -2}}, water) {
		t.Fatal("expected water where terrain is below the water threshold")
	}
	if waterVisibleForCell(res.GNDCell{Heights: [4]float32{-1, -1.2, -0.5, 0}}, water) {
		t.Fatal("unexpected water where all terrain vertices are above the water threshold")
	}
}

func TestRayWalkCellHitsProjectedGATCellCenter(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestRayWalkCellSkipsBlockedGATCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	world.GAT.Cells[87*world.GAT.Width+107] = res.GATCell{Type: res.GATTypeNone}
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	_, _, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if ok {
		t.Fatal("blocked ray walk cell should not be picked")
	}
}

func TestRayWalkCellWorksAtCloseZoom(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, defaultCameraMinZoom)
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("close zoom ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestClickedWalkTargetUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = flatWalkableGAT(180, 160)
	projection := newSceneProjectionForTargetYawZoom(1280, 720, cellCenter(107), cellCenter(90), 0, -45, sceneCameraZoom())
	point := projection.Project(cellCenter(107), cellCenter(87), 0)

	x, y, ok := clickedWalkTarget(client.Context{World: world}, projection, int(math.Round(float64(point.x))), int(math.Round(float64(point.y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("clicked walk target = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestHoveredWalkCellUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.Player.X = 5
	world.Player.Y = 5
	world.GAT = flatWalkableGAT(12, 12)
	projection := newSceneProjectionForTarget(800, 600, 5.5, 5.5, 0)
	point := projection.Project(6.5, 5.5, 0)

	x, y, ok := hoveredWalkCell(client.Context{World: world}, projection, int(point.x), int(point.y))
	if !ok || x != 6 || y != 5 {
		t.Fatalf("hovered cell = %d,%d ok=%t, want 6,5 true", x, y, ok)
	}
	if _, _, ok := hoveredWalkCell(client.Context{World: world}, projection, -10000, -10000); ok {
		t.Fatal("hover should not fall back to nearest cell outside the projected map")
	}
}

func flatWalkableGAT(width, height int) *res.GAT {
	gat := &res.GAT{
		Width:  width,
		Height: height,
		Cells:  make([]res.GATCell, width*height),
	}
	for i := range gat.Cells {
		gat.Cells[i] = res.GATCell{Type: res.GATTypeWalkable}
	}
	return gat
}

func TestTileCursorCellVertsUseWalkSurfaceHeights(t *testing.T) {
	gat := &res.GAT{
		Width:  4,
		Height: 4,
		Cells:  make([]res.GATCell, 16),
	}
	gat.Cells[2*gat.Width+1] = res.GATCell{Heights: [4]float32{2, 3, 4, 5}, Type: res.GATTypeWalkable}
	verts, ok := tileCursorCellVerts(gat, 1, 2)
	if !ok {
		t.Fatal("missing cursor cell")
	}
	want := [4]float64{2, 3, 4, 5}
	for i := range verts {
		if math.Abs(verts[i].y-want[i]) > 0.001 {
			t.Fatalf("cursor vertex %d y = %.4f, want %.4f", i, verts[i].y, want[i])
		}
	}
}

func TestTileCursorDepthBiasClearsRSMWalkSurfacesWithoutOverdrawingFurniture(t *testing.T) {
	maxRSMBias := rsmModelDepthBias + (rsmHorizontalDepthBiasLayers-1)*rsmHorizontalDepthBiasStep
	if tileCursorDepthBias <= maxRSMBias {
		t.Fatalf("tile cursor depth bias = %.10f, want above maximum RSM floor bias %.10f", tileCursorDepthBias, maxRSMBias)
	}
	if tileCursorDepthBias >= 0.001 {
		t.Fatalf("tile cursor depth bias = %.10f, want below old overdraw-prone bias", tileCursorDepthBias)
	}
}
