package maps

import (
	"image/color"
	"math"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

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

func TestSceneLightingFromRSWMatchesClassicDirection(t *testing.T) {
	// C3dWorldRes::Load rotates (0, 1, 0) about X by the latitude, then about
	// Y by the longitude; Goro mirrors the Y axis.
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{
		Longitude: 90,
		Latitude:  45,
		Diffuse:   [3]float32{1, 1, 1},
		Opacity:   1,
	}})
	want := scene.Point3{X: math.Sqrt2 / 2, Y: -math.Sqrt2 / 2, Z: 0}
	if !testutil.ModelPointNear(lighting.direction, want, 0.0001) {
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
	normal := scene.Point3{X: 0, Y: 1, Z: 0}
	if got, want := half.ModelScaleNormalized(normal), opaque.ModelScaleNormalized(normal); got != want {
		t.Fatalf("model scale changed with opacity: got %+v want %+v", got, want)
	}
}

func TestSceneLightingModelScaleUsesReferenceMinimumLightWeight(t *testing.T) {
	lighting := Lighting{
		direction: scene.Point3{Y: -1},
		Diffuse:   scene.Point3{X: 1, Y: 1, Z: 1},
		ambient:   scene.Point3{},
		Env:       scene.Point3{X: 1, Y: 1, Z: 1},
	}
	got := lighting.ModelScaleNormalized(scene.Point3{Y: 1})
	want := scene.Point3{X: 0.5, Y: 0.5, Z: 0.5}
	if got != want {
		t.Fatalf("model scale = %+v, want %+v", got, want)
	}
}

func TestLightingLitClampsEachChannelAndIgnoresBackFaces(t *testing.T) {
	lighting := Lighting{
		direction: scene.Point3{Y: -1},
		Diffuse:   scene.Point3{X: 0.8, Y: 0.4, Z: 0.1},
		ambient:   scene.Point3{X: 0.8, Y: 0.2, Z: 0.1},
		Env:       scene.Point3{X: 1, Y: 1, Z: 1},
	}
	if got, want := lighting.lit(scene.Point3{Y: -1}), (scene.Point3{X: 1, Y: 0.2 + 0.4, Z: 0.2}); !testutil.ModelPointNear(got, want, 1e-9) {
		t.Fatalf("lit front = %+v, want %+v", got, want)
	}
	if got, want := lighting.lit(scene.Point3{Y: 1}), lighting.ambient; got != want {
		t.Fatalf("lit back = %+v, want ambient %+v", got, want)
	}
}

func TestLightingNightLowersOnlyEnvironment(t *testing.T) {
	m := &System{Night: true}
	rsw := &res.RSW{Light: res.RSWLight{Longitude: 45, Latitude: 45, Diffuse: [3]float32{1, 1, 1}, Ambient: [3]float32{0.2, 0.2, 0.2}}}
	lighting := m.Lighting(rsw)
	if lighting.Diffuse != (scene.Point3{X: 1, Y: 1, Z: 1}) {
		t.Fatalf("night diffuse = %+v, want map diffuse", lighting.Diffuse)
	}
	// CWeather: 1 - (1 - 0.5) * (1 - ambient) for red and green.
	want := scene.Point3{X: 0.6, Y: 0.6, Z: 1}
	if !testutil.ModelPointNear(lighting.Env, want, 1e-6) {
		t.Fatalf("night env = %+v, want %+v", lighting.Env, want)
	}
}

func TestGNDCellLightsKeepFlatTilesUniform(t *testing.T) {
	gnd := testutil.GNDWithTopHeights(3, 3, func(_, _ int) [4]float32 {
		return [4]float32{0, 0, 0, 0}
	})
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{Longitude: 45, Latitude: 30, Diffuse: [3]float32{0.5, 0.5, 0.5}, Ambient: [3]float32{0.3, 0.3, 0.3}}})
	lights := buildGNDCellLights(gnd, lighting)
	center := lights[1+1*gnd.Width].top
	// Flat ground faces (0, 1, 0) in classic space: ambient + cos(latitude) * diffuse.
	value := uint8(255 * (0.3 + math.Cos(math.Pi/6)*0.5))
	want := color.RGBA{R: value, G: value, B: value, A: 255}
	for k, got := range center {
		if got != want {
			t.Fatalf("flat vertex %d = %+v, want %+v", k, got, want)
		}
	}
}

func TestGNDCellLightsVarySlopedTileCorners(t *testing.T) {
	gnd := testutil.GNDWithTopHeights(3, 3, func(x, y int) [4]float32 {
		return [4]float32{
			float32(x*y + y),
			float32(x*3 + y*y + 2),
			float32(x*x + y*4 + 1),
			float32(x*x + y*y + x*y + 7),
		}
	})
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{Longitude: 45, Latitude: 45, Diffuse: [3]float32{1, 1, 1}}})
	tile := buildGNDCellLights(gnd, lighting)[1+1*gnd.Width].top
	if tile[0] == tile[1] && tile[0] == tile[2] && tile[0] == tile[3] {
		t.Fatalf("sloped tile vertex colours are all equal: %+v", tile)
	}
}

func TestGNDCellLightsUseNeighbourSurfaceColoursAndBlackenMissingOnes(t *testing.T) {
	gnd := &res.GND{
		Width:  2,
		Height: 2,
		Zoom:   10,
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
	// Full ambient light leaves the surface colours unchanged.
	lighting := sceneLightingFromRSW(&res.RSW{Light: res.RSWLight{Ambient: [3]float32{1, 1, 1}}})
	lights := buildGNDCellLights(gnd, lighting)
	want := [4]color.RGBA{
		{R: 10, G: 20, B: 30, A: 255},
		{R: 40, G: 50, B: 60, A: 255},
		{R: 70, G: 80, B: 90, A: 255},
		{R: 100, G: 110, B: 120, A: 255},
	}
	if got := lights[0].top; got != want {
		t.Fatalf("top cell colours = %+v, want %+v", got, want)
	}
	black := color.RGBA{A: 255}
	if got := lights[3].top; got[0] != want[3] || got[1] != black || got[2] != black || got[3] != black {
		t.Fatalf("edge cell colours = %+v, want own colour then black", got)
	}
}

func TestPosterizeGNDLightmapColorUsesReferenceClientBuckets(t *testing.T) {
	got := posterizeGNDLightmapColor(color.RGBA{R: 15, G: 31, B: 255, A: 77})
	want := color.RGBA{R: 0, G: 16, B: 240, A: 77}
	if got != want {
		t.Fatalf("posterized lightmap color = %+v, want %+v", got, want)
	}
}

func TestGNDDrawBoundsUseCameraFootprint(t *testing.T) {
	gnd := &res.GND{Width: 200, Height: 200}
	projection := testutil.ProjectionYaw(1024, 768, 200.5, 260.5, 8, 0)
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, 1024, 768)
	if !ok {
		t.Fatal("missing GND bounds")
	}
	centerX := gndTileFromWorld(projection.PlayerX)
	centerY := gndTileFromWorld(projection.PlayerY)
	if startX > centerX || endX < centerX || startY > centerY || endY < centerY {
		t.Fatalf("bounds %d..%d,%d..%d do not include center %d,%d", startX, endX, startY, endY, centerX, centerY)
	}
	if endX-startX >= gnd.Width-1 || endY-startY >= gnd.Height-1 {
		t.Fatalf("camera bounds %d..%d,%d..%d should not cover the full map", startX, endX, startY, endY)
	}
}

func TestQuadHasInvalidPointDetectsCameraSentinel(t *testing.T) {
	points := [4]scene.Point2{{X: -1 << 20, Y: -1 << 20}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 1, Y: 2}}
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
	if uvs[0] != (scene.UV{U: 0.75, V: 0}) || uvs[2] != (scene.UV{U: 1.0, V: 0.25}) {
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
	world.GAT = testutil.FlatWalkableGAT(180, 160)
	projection := testutil.ProjectionYawZoom(1280, 720, scene.CellCenter(107), scene.CellCenter(90), 0, -45, scene.CameraZoom())
	point := projection.Project(scene.CellCenter(107), scene.CellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestRayWalkCellSkipsBlockedGATCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(180, 160)
	world.GAT.Cells[87*world.GAT.Width+107] = res.GATCell{Type: res.GATTypeNone}
	projection := testutil.ProjectionYawZoom(1280, 720, scene.CellCenter(107), scene.CellCenter(90), 0, -45, scene.CameraZoom())
	point := projection.Project(scene.CellCenter(107), scene.CellCenter(87), 0)

	_, _, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	if ok {
		t.Fatal("blocked ray walk cell should not be picked")
	}
}

func TestRayWalkCellWorksAtCloseZoom(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(180, 160)
	projection := testutil.ProjectionYawZoom(1280, 720, scene.CellCenter(107), scene.CellCenter(90), 0, -45, scene.DefaultCameraMinZoom)
	point := projection.Project(scene.CellCenter(107), scene.CellCenter(87), 0)

	x, y, ok := rayWalkCell(client.Context{World: world}, projection, int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("close zoom ray walk cell = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
}

func TestClickedWalkTargetUsesRayWalkCell(t *testing.T) {
	world := worldstate.New()
	world.GAT = testutil.FlatWalkableGAT(180, 160)
	projection := testutil.ProjectionYawZoom(1280, 720, scene.CellCenter(107), scene.CellCenter(90), 0, -45, scene.CameraZoom())
	point := projection.Project(scene.CellCenter(107), scene.CellCenter(87), 0)

	x, y, ok := ClickedWalkTarget(client.Context{World: world}, projection, int(math.Round(float64(point.X))), int(math.Round(float64(point.Y))))
	if !ok || x != 107 || y != 87 {
		t.Fatalf("clicked walk target = %d,%d ok=%t, want 107,87 true", x, y, ok)
	}
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
		if math.Abs(verts[i].Y-want[i]) > 0.001 {
			t.Fatalf("cursor vertex %d y = %.4f, want %.4f", i, verts[i].Y, want[i])
		}
	}
}

func TestTileCursorDepthBiasClearsRSMWalkSurfacesWithoutOverdrawingFurniture(t *testing.T) {
	maxRSMBias := rsmModelDepthBias + (rsmHorizontalDepthBiasLayers-1)*rsmHorizontalDepthBiasStep
	if TileCursorDepthBias <= maxRSMBias {
		t.Fatalf("tile cursor depth bias = %.10f, want above maximum RSM floor bias %.10f", TileCursorDepthBias, maxRSMBias)
	}
	if TileCursorDepthBias >= 0.001 {
		t.Fatalf("tile cursor depth bias = %.10f, want below old overdraw-prone bias", TileCursorDepthBias)
	}
}

func TestLightingInvalidatesRetainedMapMeshes(t *testing.T) {
	m := &System{gndMeshCache: &gndRetainedMeshCache{}, rsmMeshCache: map[int][]retainedWorldMesh{1: nil}}
	m.InvalidateLighting()
	if m.gndMeshCache != nil || len(m.rsmMeshCache) != 0 {
		t.Fatalf("gnd=%p rsm=%v", m.gndMeshCache, m.rsmMeshCache)
	}
}
