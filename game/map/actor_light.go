package maps

import (
	"math"
	"sync"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

// actorLightTable holds one light tint per GAT cell, as computed by
// C3dGround15::CalcLmIntensityFloatRGBAll when a map is entered.
type actorLightTable struct {
	gnd    *res.GND
	gat    *res.GAT
	width  int // GAT-resolution width: twice the GND width.
	height int
	tints  []scene.LightTint
}

var actorLightCache struct {
	sync.Mutex
	table *actorLightTable
}

// lightmapSampleOffsets are the five GAT cells averaged per cell: itself and
// its four orthogonal neighbours.
var lightmapSampleOffsets = [5][2]int{{0, 0}, {0, -1}, {0, 1}, {1, 0}, {-1, 0}}

// ActorLightTint returns the lightmap tint applied to sprites standing at the
// given world position (CRenderObject::SetLmIntensity).
func ActorLightTint(world *worldstate.World, x, y float64) scene.LightTint {
	if world == nil || world.GND == nil || len(world.GND.Lightmaps) == 0 {
		return scene.FullLight
	}
	table := actorLightTableFor(world.GND, world.GAT)
	cellX := int(math.Floor(x + 0.5))
	cellY := int(math.Floor(y + 0.5))
	return table.at(cellX, cellY)
}

func actorLightTableFor(gnd *res.GND, gat *res.GAT) *actorLightTable {
	actorLightCache.Lock()
	defer actorLightCache.Unlock()
	if table := actorLightCache.table; table != nil && table.gnd == gnd && table.gat == gat {
		return table
	}
	table := buildActorLightTable(gnd, gat)
	actorLightCache.table = table
	return table
}

func (t *actorLightTable) at(cellX, cellY int) scene.LightTint {
	if t == nil || cellX < 0 || cellY < 0 || cellX >= t.width || cellY >= t.height {
		return scene.FullLight
	}
	return t.tints[cellX+cellY*t.width]
}

func buildActorLightTable(gnd *res.GND, gat *res.GAT) *actorLightTable {
	table := &actorLightTable{gnd: gnd, gat: gat, width: gnd.Width * 2, height: gnd.Height * 2}
	table.tints = make([]scene.LightTint, table.width*table.height)
	for sy := 0; sy < table.height; sy++ {
		for sx := 0; sx < table.width; sx++ {
			table.tints[sx+sy*table.width] = actorLightTintAtGATCell(gnd, gat, sx, sy)
		}
	}
	return table
}

func actorLightTintAtGATCell(gnd *res.GND, gat *res.GAT, sx, sy int) scene.LightTint {
	// The classic client works in raw file heights, where smaller is higher;
	// Goro stores heights negated and divided by five.
	gatHeight := 0.0
	if gat != nil {
		if cell, ok := gat.Cell(sx, sy); ok {
			gatHeight = -float64(cell.Heights[0]+cell.Heights[1]+cell.Heights[2]+cell.Heights[3]) * 5 / 4
		}
	}
	topHeight, ok := gndTopRawHighestHeight(gnd, sx/2, sy/2)
	// Cells standing on a model (bridges, floors) rather than the ground stay fully lit.
	if !ok || gatHeight < topHeight-2.5 {
		return scene.FullLight
	}
	var total, totalR, totalG, totalB int
	for _, offset := range lightmapSampleOffsets {
		in, r, g, b := gndLightmapIntensityAtGATCell(gnd, sx+offset[0], sy+offset[1])
		total += int(in)
		totalR += int(r)
		totalG += int(g)
		totalB += int(b)
	}
	intensity := float64(total) / 5 / 255
	tint := scene.LightTint{
		R: float64(totalR)/5/128 + intensity,
		G: float64(totalG)/5/128 + intensity,
		B: float64(totalB)/5/128 + intensity,
	}
	if maxChannel := math.Max(tint.R, math.Max(tint.G, tint.B)); maxChannel > 1 {
		tint.R /= maxChannel
		tint.G /= maxChannel
		tint.B /= maxChannel
	}
	return tint
}

// gndTopRawHighestHeight returns the smallest raw height (the highest point)
// of a GND cell's top surface (C3dGround15::GetHeightMinMax).
func gndTopRawHighestHeight(gnd *res.GND, x, y int) (float64, bool) {
	cell, ok := gnd.Cell(x, y)
	if !ok {
		return 0, true
	}
	if cell.Top < 0 {
		return 0, false
	}
	top := math.Inf(1)
	for _, h := range cell.Heights {
		top = math.Min(top, -float64(h)*5)
	}
	return top, true
}

// gndLightmapIntensityAtGATCell samples the lightmap texel the classic client
// keeps for each quarter of a GND cell (CLightmapMgr::Create,
// C3dGround15::GetLmIntensityRGBByGatCoor).
func gndLightmapIntensityAtGATCell(gnd *res.GND, cx, cy int) (in, r, g, b uint8) {
	if cx < 0 || cy < 0 || cx/2 >= gnd.Width || cy/2 >= gnd.Height {
		return 255, 255, 255, 255
	}
	cell, ok := gnd.Cell(cx/2, cy/2)
	if !ok || cell.Top < 0 {
		return 255, 255, 255, 255
	}
	surface, ok := gnd.Surface(cell.Top)
	if !ok {
		return 255, 255, 255, 255
	}
	lightmap, ok := gnd.Lightmap(surface.LightmapID)
	if !ok {
		return 255, 255, 255, 255
	}
	texelX, texelY := 1, 1
	if cx%2 == 1 {
		texelX = 5
	}
	if cy%2 == 1 {
		texelY = 5
	}
	c := lightmap.Color[texelY][texelX]
	return lightmap.Alpha[texelY][texelX], c.R, c.G, c.B
}
