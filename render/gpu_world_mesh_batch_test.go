package render

import (
	"slices"
	"testing"

	"github.com/gogpu/wgpu"
)

func TestWorldMeshBatchPreservesVertexDataAndTriangleOrder(t *testing.T) {
	texture, light := NewImage(32, 16), NewImage(64, 128)
	options := DrawTrianglesOptions{DepthWrite: true, DepthBias: 0.005, DisableFog: true}
	a := NewWorldMeshWithLightmap([]Vertex3D{
		{X: 7, Y: 8, Z: 9, SrcX: 16, SrcY: 8, LightSrcX: 32, LightSrcY: 64,
			ColorR: 0.25, ColorG: 0.5, ColorB: 0.75, ColorA: 1, DepthX: 1, DepthY: 2, DepthZ: 3},
		{X: 10, SrcX: 32, LightSrcY: 128},
		{X: 11, SrcY: 16, LightSrcX: 64},
	}, []uint16{2, 0, 1}, texture, light, &options)
	b := testWorldMesh(texture, &options)
	b.lightTexture = light
	batch := worldMeshBatch{key: drawBatchKey{texture: texture, lightTexture: light, options: options}, meshes: []*WorldMesh{a, b, a}}
	floats, indices := worldMeshBatchGPUData(nil, nil, batch, 32, 16, 64, 128)
	if want := []uint32{2, 0, 1, 3, 4, 5, 8, 6, 7}; !slices.Equal(indices, want) {
		t.Fatalf("merged triangle order = %v, want %v", indices, want)
	}
	first := []float32{7, 8, 9, 0.5, 0.5, 0.25, 0.5, 0.75, 1, 1, 2, 3, 0.005, 0, 0.5, 0.5}
	if !slices.Equal(floats[:worldVertexFloatCount], first) {
		t.Fatalf("first vertex = %v, want %v", floats[:worldVertexFloatCount], first)
	}
	var separate []float32
	for _, mesh := range batch.meshes {
		packed, _ := worldMeshGPUData(mesh, 32, 16)
		separate = append(separate, packed...)
	}
	if !slices.Equal(floats, separate) {
		t.Fatal("merging changed vertex data compared with separate mesh uploads")
	}
}

func TestWorldMeshBatchRebasesIndicesBeyondUint16(t *testing.T) {
	texture := WhiteImage()
	options := DrawTrianglesOptions{DepthWrite: true}
	a := NewWorldMeshWithLightmap(make([]Vertex3D, 65535), []uint16{0, 65533, 65534}, texture, nil, &options)
	b := testWorldMesh(texture, &options)
	_, indices := worldMeshBatchGPUData(nil, nil, worldMeshBatch{meshes: []*WorldMesh{a, b, b}}, 1, 1, 1, 1)
	want := []uint32{0, 65533, 65534, 65535, 65536, 65537, 65538, 65539, 65540}
	if !slices.Equal(indices, want) {
		t.Fatalf("merged indices = %v, want %v", indices, want)
	}
}

func TestWorldMeshBatchCacheTracksVisibilityOrderVersionsAndTextureSizes(t *testing.T) {
	a, b := testWorldMesh(WhiteImage(), nil), testWorldMesh(WhiteImage(), nil)
	cached := &gpuWorldMeshBatch{}
	meshes := []*WorldMesh{a, b}
	cached.remember(worldMeshBatch{meshes: meshes}, 32, 16, 64, 128)
	if !cached.matches(worldMeshBatch{meshes: meshes}, 32, 16, 64, 128) {
		t.Fatal("unchanged visible meshes invalidated the cache")
	}
	for _, changed := range [][]*WorldMesh{{a}, {b, a}, {a, a}, {a, b, a}, nil} {
		if cached.matches(worldMeshBatch{meshes: changed}, 32, 16, 64, 128) {
			t.Fatal("changed mesh visibility, order or multiplicity reused stale geometry")
		}
	}
	for _, size := range [][4]int{{64, 16, 64, 128}, {32, 32, 64, 128}, {32, 16, 128, 128}, {32, 16, 64, 256}} {
		if cached.matches(worldMeshBatch{meshes: meshes}, size[0], size[1], size[2], size[3]) {
			t.Fatal("changed texture dimensions reused stale UVs")
		}
	}
	a.version++
	if cached.matches(worldMeshBatch{meshes: meshes}, 32, 16, 64, 128) {
		t.Fatal("changed mesh version reused stale geometry")
	}
	cached.remember(worldMeshBatch{meshes: meshes[:1]}, 32, 16, 64, 128)
	if !cached.matches(worldMeshBatch{meshes: meshes[:1]}, 32, 16, 64, 128) {
		t.Fatal("updated visible set did not replace cached membership")
	}
	if cached.meshes[:cap(cached.meshes)][1].mesh != nil {
		t.Fatal("cache retained an invisible mesh in the unused slice tail")
	}
}

func TestWorldMeshBatchCacheReleasesInvisibleBatches(t *testing.T) {
	a, b := testWorldMesh(WhiteImage(), &DrawTrianglesOptions{DepthWrite: true}), testWorldMesh(NewImage(1, 1), &DrawTrianglesOptions{DepthWrite: true})
	screen := NewFrame(320, 240)
	screen.DrawWorldMesh(a)
	screen.DrawWorldMesh(b)
	r := &gpuRenderer{worldMeshBatchCache: make(map[drawBatchKey]*gpuWorldMeshBatch)}
	batches := r.depthWriteWorldMeshBatches(screen)
	page := &worldMeshBufferPage{buf: &wgpu.Buffer{}, free: []worldMeshBufferRange{{size: worldMeshBufferPageSize}}}
	r.worldMeshBufferPages = append(r.worldMeshBufferPages, page)
	for _, batch := range batches {
		allocation, _ := page.allocate(4096)
		cached := &gpuWorldMeshBatch{allocation: allocation}
		cached.remember(batch, 1, 1, 1, 1)
		r.worldMeshBatchCache[batch.key] = cached
	}
	keyA := drawBatchKey{texture: a.texture, options: a.options}
	keyB := drawBatchKey{texture: b.texture, options: b.options}
	cachedA, cachedB := r.worldMeshBatchCache[keyA], r.worldMeshBatchCache[keyB]
	screen.BeginFrame()
	screen.DrawWorldMesh(a)
	r.depthWriteWorldMeshBatches(screen)
	if len(r.worldMeshBatchCache) != 1 || r.worldMeshBatchCache[keyA] != cachedA {
		t.Fatal("visible batch was evicted or invisible batch retained")
	}
	if cachedB.allocation.page != nil || len(cachedB.meshes) != 0 {
		t.Fatal("evicted batch kept its buffers or mesh references")
	}
	if page.buf == nil || len(r.worldMeshBufferPages) != 1 || page.allocations != 1 {
		t.Fatal("evicting one batch released a page still used by another batch")
	}
	screen.BeginFrame()
	r.depthWriteWorldMeshBatches(screen)
	if len(r.worldMeshBatchCache) != 0 || cachedA.allocation.page != nil || len(r.worldMeshBufferPages) != 0 || page.buf != nil {
		t.Fatal("empty frame did not release the previous map's batch buffers")
	}
}

func TestWorldMeshBatchesKeepDistinctLightmapsAndRenderOptions(t *testing.T) {
	texture, light := WhiteImage(), NewImage(8, 8)
	screen := NewFrame(320, 240)
	for _, options := range []DrawTrianglesOptions{
		{DepthWrite: true}, {DepthWrite: true, DepthBias: 0.001},
		{DepthWrite: true, DisableFog: true}, {DepthWrite: true, Blend: BlendLighter},
		{DepthWrite: true, Filter: FilterLinear}, {DepthWrite: true, Address: AddressRepeat},
	} {
		for _, lightTexture := range []*Image{nil, light} {
			mesh := testWorldMesh(texture, &options)
			mesh.lightTexture = lightTexture
			screen.DrawWorldMesh(mesh)
		}
	}
	if got := len((&gpuRenderer{}).depthWriteWorldMeshBatches(screen)); got != 12 {
		t.Fatalf("distinct render states produced %d batches, want 12", got)
	}
}

func TestWorldMeshSubmissionInvalidatesGroupingAndGPUValidation(t *testing.T) {
	texture := WhiteImage()
	options := &DrawTrianglesOptions{DepthWrite: true}
	a, b := testWorldMesh(texture, options), testWorldMesh(texture, options)
	screen := NewFrame(320, 240)
	screen.DrawWorldMesh(a)
	screen.DrawWorldMesh(b)
	r := &gpuRenderer{}
	batch := r.depthWriteWorldMeshBatches(screen)[0]
	cached := &gpuWorldMeshBatch{}
	cached.remember(batch, 1, 1, 1, 1)
	if next := r.depthWriteWorldMeshBatches(screen)[0]; next.revision != batch.revision || !cached.matches(next, 1, 1, 1, 1) {
		t.Fatal("unchanged submission did not reuse its grouping and validation")
	}
	emptyPage := &worldMeshBufferPage{buf: &wgpu.Buffer{}}
	r.worldMeshBufferPages = append(r.worldMeshBufferPages, emptyPage)
	r.depthWriteWorldMeshBatches(screen)
	if emptyPage.buf != nil || len(r.worldMeshBufferPages) != 0 {
		t.Fatal("unchanged submission retained an empty page left by an upload")
	}
	// Frame reuses its command backing array, so the cache must own its snapshot.
	screen.worldMeshes[0], screen.worldMeshes[1] = screen.worldMeshes[1], screen.worldMeshes[0]
	reordered := r.depthWriteWorldMeshBatches(screen)[0]
	if reordered.revision == batch.revision || !slices.Equal(reordered.meshes, []*WorldMesh{b, a}) || cached.matches(reordered, 1, 1, 1, 1) {
		t.Fatal("reordered commands reused stale grouping or GPU data")
	}
	// Grouping may run multiple times without any intervening upload (for example
	// with the camera disabled). An unchanged second frame must not validate an
	// older GPU batch just because the current CPU grouping is unchanged.
	if unchanged := r.depthWriteWorldMeshBatches(screen)[0]; cached.matches(unchanged, 1, 1, 1, 1) {
		t.Fatal("unchanged grouping bypassed validation of a stale GPU batch")
	}
	cached.remember(reordered, 1, 1, 1, 1)
	b.version++
	updated := r.depthWriteWorldMeshBatches(screen)[0]
	if updated.revision == reordered.revision || cached.matches(updated, 1, 1, 1, 1) {
		t.Fatal("a changed mesh version did not invalidate the GPU data")
	}
	cached.remember(updated, 1, 1, 1, 1)
	for _, size := range [][4]int{{2, 1, 1, 1}, {1, 2, 1, 1}, {1, 1, 2, 1}, {1, 1, 1, 2}} {
		if cached.matches(updated, size[0], size[1], size[2], size[3]) {
			t.Fatal("unchanged revision bypassed texture dimension validation")
		}
	}
	screen.DrawWorldMesh(b)
	duplicated := r.depthWriteWorldMeshBatches(screen)[0]
	if duplicated.revision == updated.revision || !slices.Equal(duplicated.meshes, []*WorldMesh{b, a, b}) || cached.matches(duplicated, 1, 1, 1, 1) {
		t.Fatal("a duplicate submission was lost or reused stale geometry")
	}
	screen.BeginFrame()
	if len(r.depthWriteWorldMeshBatches(screen)) != 0 {
		t.Fatal("empty frame retained the previous scene's batches")
	}
	for _, entry := range r.worldMeshSubmission.meshes[:cap(r.worldMeshSubmission.meshes)] {
		if entry.mesh != nil {
			t.Fatal("empty frame retained an old mesh in the submission snapshot")
		}
	}
}

func TestWorldMeshSubmissionHandlesNilAndTransparentCommands(t *testing.T) {
	r := &gpuRenderer{}
	screen := NewFrame(320, 240)
	opaque := testWorldMesh(WhiteImage(), &DrawTrianglesOptions{DepthWrite: true})
	transparent := testWorldMesh(WhiteImage(), nil)
	screen.worldMeshes = []WorldMeshCommand{{}, {Mesh: opaque}, {Mesh: transparent}}
	for range 2 {
		batches := r.depthWriteWorldMeshBatches(screen)
		if len(batches) != 1 || !slices.Equal(batches[0].meshes, []*WorldMesh{opaque}) {
			t.Fatal("cached grouping changed nil/transparent command filtering")
		}
	}
	if len(r.depthWriteWorldMeshBatches(nil)) != 0 {
		t.Fatal("nil frame did not clear the grouping")
	}
}

func BenchmarkDepthWriteWorldMeshBatches(b *testing.B) {
	screen := NewFrame(1280, 720)
	for range 32 {
		texture := NewImage(16, 16)
		for range 32 {
			screen.DrawWorldMesh(testWorldMesh(texture, &DrawTrianglesOptions{DepthWrite: true}))
		}
	}
	r := &gpuRenderer{}
	r.depthWriteWorldMeshBatches(screen)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		r.depthWriteWorldMeshBatches(screen)
	}
}
