package game

import (
	"encoding/binary"
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

// Automatic cloud assignments in the 2008-09-10 client, cross-checked against
// classic-ro-client. Keep this explicit so missing or overbroad routing fails.
var referenceCloudMaps = map[int][]string{
	effectCloud:  {"gef_fild07", "mjolnir_01"},
	effectCloud2: {"yuno", "gonryun", "gon_dun02", "ra_temsky", "que_temsky", "sch_gld", "bat_fild02", "bat_b01", "bat_b02"},
	effectCloud3: {"valkyrie", "rwc01", "himinn", "que_qsch01", "que_qsch02", "que_qsch03", "que_qsch04", "que_qsch05", "que_qaru01", "que_qaru02", "que_qaru03", "que_qaru04", "que_qaru05"},
	effectCloud4: {"einbroch"},
	effectCloud5: {"airplane", "airplane_01"},
	effectCloud6: {"thana_boss", "moc_fild22", "moc_fild22b"},
	effectCloud7: {"6@tower"},
	effectCloud8: {"5@tower"},
}

func TestAllOriginalMapCloudAssignments(t *testing.T) {
	for want, names := range referenceCloudMaps {
		for _, name := range names {
			for _, input := range []string{name, name + ".gat", "DATA\\" + strings.ToUpper(name) + ".RSW"} {
				if got := mapWeatherEffectIDForMap(input); got != want {
					t.Errorf("%s: effect %d, want %d", input, got, want)
				}
			}
		}
	}
	for name, want := range map[string]int{
		"0005@tower.gat": effectCloud8, "DATA/0016@TOWER.RSW": effectCloud7,
		"1235@tower": effectCloud8, "abc6@tower": effectCloud7,
		"4@tower": 0, "0014@tower": 0, "15@tower": 0, "00005@tower": 0,
		"0005@tower_extra": 0, "que_qsch06": 0, "que_qaru00": 0,
		"gef_fild06": 0, "mjolnir_02": 0, "moc_fild21": 0,
	} {
		if got := mapWeatherEffectIDForMap(name); got != want {
			t.Errorf("%s: effect %d, want %d", name, got, want)
		}
	}
}

func TestAdditionalCloudVariantsMatchOriginalPlacementAndColor(t *testing.T) {
	tests := []struct {
		effectID                                       int
		count                                          int
		zMin, zMax, offsetMin, offsetMax, alpha, drift float64
		tint                                           color.RGBA
	}{
		{effectCloud, 160, 23, 25, 0, 30, 160.0 / 255, 0.6, color.RGBA{255, 255, 255, 255}},
		{effectCloud3, 160, -2, 0, 0, 30, 160.0 / 255, 0.6, color.RGBA{255, 255, 255, 255}},
		{effectCloud6, 320, -6, -4, 0, 30, 160.0 / 255, 0.42, color.RGBA{94, 0, 0, 255}},
		{effectCloud7, 320, -10, -8, 5, 45, 240.0 / 255, 0.6, color.RGBA{0, 0, 0, 255}},
		{effectCloud8, 320, -10, -8, 5, 45, 240.0 / 255, 0.6, color.RGBA{255, 180, 180, 255}},
	}
	world := &worldstate.World{GND: testGNDWithTopHeights(64, 64, func(_, _ int) [4]float32 {
		return [4]float32{100, 100, 100, 100}
	})}
	for _, tt := range tests {
		params, ok := weatherCloudParamsForEffect(tt.effectID)
		if !ok {
			t.Fatalf("missing profile %d", tt.effectID)
		}
		if params.tint != tt.tint || params.alphaMax != tt.alpha || math.Abs(params.driftSpeed-tt.drift) > 1e-9 || params.overlay || params.useGround || params.additive || params.blackKey || !params.disableFog || params.screenHaze.A != 0 {
			t.Fatalf("wrong cloud profile %d: %+v", tt.effectID, params)
		}
		state := mapWeatherCloudState{}
		state.ensure("map.rsw", params, world, 64, 64, time.Unix(100, 0))
		if len(state.clouds) != tt.count {
			t.Fatalf("effect %d: %d clouds, want %d", tt.effectID, len(state.clouds), tt.count)
		}
		for _, cloud := range state.clouds {
			dx, dy := math.Abs(cloud.x-64), math.Abs(cloud.y-64)
			if dx < tt.offsetMin || dx > tt.offsetMax || dy < tt.offsetMin || dy > tt.offsetMax || cloud.z < tt.zMin || cloud.z > tt.zMax {
				t.Fatalf("effect %d: incorrect absolute placement %+v", tt.effectID, cloud)
			}
		}
	}
}

func cloudTestWorldAtOriginalHeight(t *testing.T, height float32) *worldstate.World {
	t.Helper()
	data := make([]byte, 34)
	copy(data, "GRAT")
	data[4], data[5] = 1, 2
	binary.LittleEndian.PutUint32(data[6:], 1)
	binary.LittleEndian.PutUint32(data[10:], 1)
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(data[14+4*i:], math.Float32bits(height))
	}
	gat, err := res.ParseGAT(data)
	if err != nil {
		t.Fatal(err)
	}
	return &worldstate.World{GAT: gat}
}

func TestMountainCloudAltitudeUsesGATUnitsAndCompletesFade(t *testing.T) {
	params, _ := weatherCloudParamsForEffect(effectCloud)
	for _, height := range []float32{-151.99, -152, -152.01} {
		world := cloudTestWorldAtOriginalHeight(t, height)
		now := time.Unix(100, 0)
		state := mapWeatherCloudState{}
		state.ensure("mjolnir_01.rsw", params, world, 0.5, 0.5, now)
		state.update(params, world, 0.5, 0.5, now.Add(100*time.Millisecond))
		if visible := mapWeatherCloudAlpha(state.clouds[0], params) > 0; visible != (height < -152) {
			t.Fatalf("original GAT height %g: cloud visible=%t", height, visible)
		}
	}

	high, low := cloudTestWorldAtOriginalHeight(t, -160), cloudTestWorldAtOriginalHeight(t, -100)
	now := time.Unix(100, 0)
	state := mapWeatherCloudState{}
	state.ensure("mjolnir_01.rsw", params, high, 0.5, 0.5, now)
	advance := func(world *worldstate.World, steps int) {
		for i := 0; i < steps; i++ {
			now = now.Add(100 * time.Millisecond)
			state.update(params, world, 0.5, 0.5, now)
		}
	}
	advance(high, 10)
	alpha := mapWeatherCloudAlpha(state.clouds[0], params)
	advance(low, 1)
	if got := mapWeatherCloudAlpha(state.clouds[0], params); got != alpha || got == 0 {
		t.Fatalf("descending popped or brightened the clouds: %f -> %f", alpha, got)
	}
	advance(low, 200)
	for _, cloud := range state.clouds {
		if mapWeatherCloudAlpha(cloud, params) != 0 {
			t.Fatal("clouds kept forming below the altitude threshold")
		}
	}
	visible := false
	for i := 0; i < 200; i++ {
		advance(high, 1)
		for _, cloud := range state.clouds {
			visible = visible || mapWeatherCloudAlpha(cloud, params) > 0
		}
	}
	if !visible {
		t.Fatal("clouds failed to return after climbing above the threshold")
	}
}

func TestEinbrochCloudsSpawnAboveTerrain(t *testing.T) {
	params, _ := weatherCloudParamsForEffect(effectCloud4)
	world := &worldstate.World{GND: testGNDWithTopHeights(64, 64, func(_, _ int) [4]float32 { return [4]float32{10, 10, 10, 10} })}
	state := mapWeatherCloudState{}
	state.ensure("einbroch.rsw", params, world, 64, 64, time.Unix(100, 0))
	for _, cloud := range state.clouds {
		if cloud.z < 14 || cloud.z > 15 {
			t.Fatalf("cloud height %f, want terrain + 4..5", cloud.z)
		}
	}
}

func TestAllCloudMapsRenderRealData(t *testing.T) {
	manager := realDataManager(t)
	mode := &WorldMode{}
	for effectID, names := range referenceCloudMaps {
		for _, mapName := range names {
			t.Run(mapName, func(t *testing.T) {
				gnd, _, err := loadGND(manager, mapName)
				if err != nil {
					t.Skipf("map geometry unavailable: %v", err)
				}
				gat, _, err := loadGAT(manager, mapName)
				if err != nil {
					t.Fatal(err)
				}
				world := worldstate.New()
				world.MapName, world.GND, world.GAT = mapName, gnd, gat
				// Use the highest walkable cell so mountain clouds pass the altitude gate.
				best := math.Inf(-1)
				for i, cell := range gat.Cells {
					if cell.Type&res.GATTypeWalkable == 0 {
						continue
					}
					x, y := i%gat.Width, i/gat.Width
					if z := terrainHeightAt(world, float64(x), float64(y)); z > best {
						best = z
						world.Player.X = x
						world.Player.Y = y
					}
				}
				if math.IsInf(best, -1) {
					t.Fatal("no walkable cells")
				}
				ctx := client.Context{World: world, Resources: manager}
				projection := newSceneProjectionForTarget(800, 600, float64(world.Player.X)+0.5, float64(world.Player.Y)+0.5, best)
				frame := render.NewFrame(800, 600)
				now := time.Unix(100, 0)
				mode.drawMapWeatherEffects(frame, ctx, projection, now)
				mode.drawMapWeatherEffects(frame, ctx, projection, now.Add(100*time.Millisecond))
				commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				params, _ := weatherCloudParamsForEffect(effectID)
				if mode.mapWeatherCloud.effectID != effectID || commands.Len() != params.count {
					t.Fatalf("effect %d: got %d billboards, want %d", mode.mapWeatherCloud.effectID, commands.Len(), params.count)
				}
				t.Logf("%s: %d cloud billboards", mapName, commands.Len())
				world.MapName = "prontera"
				frame.BeginFrame()
				mode.drawMapWeatherEffects(frame, ctx, projection, now.Add(200*time.Millisecond))
				if len(mode.mapWeatherCloud.clouds) != 0 || commands.Len() != 0 {
					t.Fatal("clouds survived leaving the map")
				}
			})
		}
	}
}
