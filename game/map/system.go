package maps

import (
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

type System struct {
	WhitePixel       *render.Image
	tileCursor       *render.Image
	Textures         map[string]*render.Image
	Images           Textures
	UpscaleTextures  bool // Captured when loading the map; cached meshes keep matching UVs.
	TextureMiss      map[string]struct{}
	rsmMeshCache     map[int][]retainedWorldMesh
	rsmNodeMatrices  map[*res.RSM]map[string]scene.Mat4
	rsmAnimNodes     map[*res.RSM]animatedRSMNodeCache
	rsmBoundsCache   map[rsmBoundsCacheKey]rsmBounds
	rsmFaceMetaCache map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta
	rsmAnimScratch   animatedRSMScratch
	rsmPlacementGrid *rsmPlacementGrid
	runtimeRSMModels map[string]*res.RSM
	gndMeshCache     *gndRetainedMeshCache
	gndNormalSource  *res.GND
	gndTopNormals    [][4]scene.Point3
	MapSoundNext     map[int]time.Time
	rswEffectSource  *res.RSW
	rswEffects       []rswEffectEmitter
	Night            bool
}

func (m *System) ResetGeometry() {
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
	m.rsmNodeMatrices = make(map[*res.RSM]map[string]scene.Mat4)
	m.rsmAnimNodes = make(map[*res.RSM]animatedRSMNodeCache)
	m.rsmBoundsCache = make(map[rsmBoundsCacheKey]rsmBounds)
	m.rsmFaceMetaCache = make(map[*res.RSM]map[*res.RSMNode][]rsmFaceMeta)
	m.rsmAnimScratch.reset()
	m.rsmPlacementGrid = nil
	m.gndMeshCache = nil
}

func (m *System) InvalidateLighting() {
	m.gndMeshCache = nil
	m.rsmMeshCache = make(map[int][]retainedWorldMesh)
}
