package render

import (
	"encoding/binary"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/naga"
	"github.com/gogpu/naga/spirv"
	"github.com/gogpu/wgpu"
)

func TestRendererShadersParse(t *testing.T) {
	for name, source := range map[string]string{
		"screen":          screenShaderWGSL,
		"world":           worldShaderWGSL,
		"world-billboard": worldBillboardShaderWGSL,
		"bloom":           bloomShaderWGSL,
		"ssao":            ssaoShaderWGSL,
		"ssao-depth":      worldShaderWGSL + ssaoDepthShaderWGSL,
	} {
		ast, err := naga.Parse(source)
		if err != nil {
			t.Fatalf("%s shader parse: %v", name, err)
		}
		if _, err := naga.Lower(ast); err != nil {
			t.Fatalf("%s shader lower: %v", name, err)
		}
	}
}

func TestRendererShadersGenerateSPIRV(t *testing.T) {
	for name, source := range map[string]string{
		"screen":          screenShaderWGSL,
		"world":           worldShaderWGSL,
		"world-billboard": worldBillboardShaderWGSL,
		"bloom":           bloomShaderWGSL,
		"ssao":            ssaoShaderWGSL,
		"ssao-depth":      worldShaderWGSL + ssaoDepthShaderWGSL,
	} {
		ast, err := naga.Parse(source)
		if err != nil {
			t.Fatalf("%s shader parse: %v", name, err)
		}
		module, err := naga.Lower(ast)
		if err != nil {
			t.Fatalf("%s shader lower: %v", name, err)
		}
		data, err := naga.GenerateSPIRV(module, spirv.Options{Version: spirv.Version1_3})
		if err != nil {
			t.Fatalf("%s shader SPIR-V: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("%s shader SPIR-V is empty", name)
		}
	}
}

func TestWorldShadersUseProjectedFogDepth(t *testing.T) {
	for name, source := range map[string]string{
		"world":           worldShaderWGSL,
		"world-billboard": worldBillboardShaderWGSL,
	} {
		if !strings.Contains(source, "fog_depth") || !strings.Contains(source, "smoothstep(uniforms.fog[0], uniforms.fog[1], input.fog_depth)") {
			t.Fatalf("%s shader does not use interpolated fog depth", name)
		}
		if !strings.Contains(source, "* uniforms.fog[3]") {
			t.Fatalf("%s shader does not use fog strength", name)
		}
		if strings.Contains(source, "input.clip[2] /") {
			t.Fatalf("%s shader uses fragment clip depth for fog", name)
		}
	}
	if !strings.Contains(worldBillboardShaderWGSL, "* input.fog_enabled") {
		t.Fatal("world billboard shader does not honor per-billboard fog toggle")
	}
}

func TestWorldMeshSubmissionDoesNotCreateDynamicWorldCommand(t *testing.T) {
	screen := NewFrame(320, 240)
	texture := WhiteImage()
	mesh := NewWorldMesh([]Vertex3D{
		{X: 0, Y: 0, Z: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{X: 1, Y: 0, Z: 0, SrcX: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{X: 1, Y: 1, Z: 0, SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}, []uint16{0, 1, 2}, texture, &DrawTrianglesOptions{DepthWrite: true})

	screen.BeginFrame()
	screen.DrawWorldMesh(mesh)

	if got := len(screen.worldMeshes); got != 1 {
		t.Fatalf("world mesh commands = %d, want 1", got)
	}
	if got := len(screen.worldCommands); got != 0 {
		t.Fatalf("dynamic world commands = %d, want 0", got)
	}
}

func TestDepthWriteWorldMeshBatchesGroupByRenderKey(t *testing.T) {
	screen := NewFrame(320, 240)
	texture := WhiteImage()
	otherTexture := NewImage(1, 1)
	otherTexture.Fill(color.White)
	options := DrawTrianglesOptions{DepthWrite: true}
	meshA := testWorldMesh(texture, &options)
	meshB := testWorldMesh(texture, &options)
	meshC := testWorldMesh(otherTexture, &options)

	screen.BeginFrame()
	screen.DrawWorldMesh(meshA)
	screen.DrawWorldMesh(meshB)
	screen.DrawWorldMesh(meshC)

	batches := (&gpuRenderer{}).depthWriteWorldMeshBatches(screen)
	if got := len(batches); got != 2 {
		t.Fatalf("world mesh batches = %d, want 2", got)
	}
	if got := len(batches[0].meshes); got != 2 {
		t.Fatalf("first batch meshes = %d, want 2", got)
	}
	if got := len(batches[1].meshes); got != 1 {
		t.Fatalf("second batch meshes = %d, want 1", got)
	}
}

func TestDepthWriteWorldMeshBatchesSkipNonDepthMesh(t *testing.T) {
	screen := NewFrame(320, 240)
	texture := WhiteImage()

	screen.BeginFrame()
	screen.DrawWorldMesh(testWorldMesh(texture, &DrawTrianglesOptions{}))

	if got := len((&gpuRenderer{}).depthWriteWorldMeshBatches(screen)); got != 0 {
		t.Fatalf("world mesh batches = %d, want 0", got)
	}
}

func TestWorldBillboardSubmissionDoesNotCreateDynamicWorldCommand(t *testing.T) {
	screen := NewFrame(320, 240)
	screen.BeginFrame()
	screen.DrawWorldBillboard(WorldBillboardCommand{
		Texture: WhiteImage(),
		Width:   16,
		Height:  16,
		ColorR:  1,
		ColorG:  1,
		ColorB:  1,
		ColorA:  1,
	})

	if got := len(screen.worldBillboards); got != 1 {
		t.Fatalf("world billboard commands = %d, want 1", got)
	}
	if got := len(screen.worldCommands); got != 0 {
		t.Fatalf("dynamic world commands = %d, want 0", got)
	}
}

func TestWorldBillboardCommandsKeepSeparateInstanceData(t *testing.T) {
	screen := NewFrame(320, 240)
	texture := WhiteImage()
	screen.BeginFrame()
	screen.DrawWorldBillboard(WorldBillboardCommand{
		Texture: texture,
		Center:  [3]float32{1, 2, 3},
		Width:   16,
		Height:  16,
		ColorA:  1,
	})
	screen.DrawWorldBillboard(WorldBillboardCommand{
		Texture: texture,
		Center:  [3]float32{7, 8, 9},
		Width:   16,
		Height:  16,
		ColorA:  0.5,
	})

	if got := len(screen.worldBillboards); got != 2 {
		t.Fatalf("world billboard commands = %d, want 2", got)
	}
	if screen.worldBillboards[0].Center == screen.worldBillboards[1].Center {
		t.Fatalf("billboard centers unexpectedly alias: %v", screen.worldBillboards[0].Center)
	}
	if screen.worldBillboards[0].ColorA == screen.worldBillboards[1].ColorA {
		t.Fatalf("billboard alpha unexpectedly identical: %v", screen.worldBillboards[0].ColorA)
	}
}

func TestWorldBillboardBatchesKeepContiguousOrder(t *testing.T) {
	texture := WhiteImage()
	otherTexture := NewImage(1, 1)
	otherTexture.Fill(color.White)
	commands := []WorldBillboardCommand{
		{Texture: texture, Center: [3]float32{1, 2, 3}, Width: 16, Height: 16, ColorA: 1},
		{Texture: texture, Center: [3]float32{4, 5, 6}, Width: 16, Height: 16, ColorA: 1},
		{Texture: otherTexture, Center: [3]float32{7, 8, 9}, Width: 16, Height: 16, ColorA: 1},
		{Texture: texture, Center: [3]float32{10, 11, 12}, Width: 16, Height: 16, ColorA: 1},
	}

	instances, batches := (&gpuRenderer{}).buildWorldBillboardBatches(commands)

	if got := len(instances); got != len(commands)*billboardInstanceFloatCount {
		t.Fatalf("billboard instance floats = %d, want %d", got, len(commands)*billboardInstanceFloatCount)
	}
	if got := len(batches); got != 3 {
		t.Fatalf("billboard batches = %d, want 3", got)
	}
	wantFirst := []uint32{0, 2, 3}
	wantCount := []uint32{2, 1, 1}
	for i := range batches {
		if batches[i].firstInstance != wantFirst[i] || batches[i].instanceCount != wantCount[i] {
			t.Fatalf("batch %d = first %d count %d, want first %d count %d", i, batches[i].firstInstance, batches[i].instanceCount, wantFirst[i], wantCount[i])
		}
	}
	for i, cmd := range commands {
		base := i * billboardInstanceFloatCount
		got := [3]float32{instances[base], instances[base+1], instances[base+2]}
		if got != cmd.Center {
			t.Fatalf("instance %d center = %v, want %v", i, got, cmd.Center)
		}
	}
}

func TestWorldBillboardInstanceDataCarriesFogToggle(t *testing.T) {
	texture := WhiteImage()
	fogged := billboardInstanceData(WorldBillboardCommand{Texture: texture})
	unfogged := billboardInstanceData(WorldBillboardCommand{Texture: texture, Options: DrawTrianglesOptions{DisableFog: true}})
	if got := fogged[21]; got != 1 {
		t.Fatalf("fogged billboard flag = %.1f, want 1", got)
	}
	if got := unfogged[21]; got != 0 {
		t.Fatalf("unfogged billboard flag = %.1f, want 0", got)
	}
}

func TestWorldDepthCompareHonorsDepthTestOption(t *testing.T) {
	if got := worldDepthCompare(true); got != gputypes.CompareFunctionLessEqual {
		t.Fatalf("depth-tested billboard compare = %v, want less-equal", got)
	}
	if got := worldDepthCompare(false); got != gputypes.CompareFunctionAlways {
		t.Fatalf("overlay billboard compare = %v, want always", got)
	}
}

func TestWorldPipelineHonorsIndependentDepthOptions(t *testing.T) {
	r := &gpuRenderer{worldPipelines: make(map[worldPipelineKey]*wgpu.RenderPipeline)}
	for _, blend := range []Blend{BlendSourceOver, BlendLighter, BlendSrcAlphaDstAlpha} {
		for _, depthTest := range []bool{false, true} {
			for _, depthWrite := range []bool{false, true} {
				key := worldPipelineKey{blend, depthTest, depthWrite}
				r.worldPipelines[key] = &wgpu.RenderPipeline{}
			}
		}
	}
	for key, pipeline := range r.worldPipelines {
		options := DrawTrianglesOptions{Blend: key.blend, DepthTest: key.depthTest, DepthWrite: key.depthWrite}
		if got := r.worldPipelineFor(options); got != pipeline {
			t.Fatalf("pipeline for %+v = %p, want %p", options, got, pipeline)
		}
		desc := r.worldPipelineDescriptor(nil, gputypes.BlendStateAlpha(), key.depthTest, key.depthWrite, "test")
		if desc.DepthStencil.Format != gputypes.TextureFormatDepth32Float {
			t.Fatalf("GPU depth format = %v, want Depth32Float for AMD Vulkan occlusion", desc.DepthStencil.Format)
		}
		wantCompare := gputypes.CompareFunctionAlways
		if key.depthTest {
			wantCompare = gputypes.CompareFunctionLessEqual
		}
		if desc.DepthStencil.DepthCompare != wantCompare || desc.DepthStencil.DepthWriteEnabled != key.depthWrite {
			t.Fatalf("GPU depth state for %+v = %+v", options, desc.DepthStencil)
		}
	}
}

func TestScreenPipelineIsIndependentOfWorldMSAA(t *testing.T) {
	for _, samples := range []uint32{1, 4} {
		r := &gpuRenderer{format: gputypes.TextureFormatBGRA8Unorm, sampleCount: samples}
		world := r.worldPipelineDescriptor(nil, gputypes.BlendStateAlpha(), true, true, "world")
		screen := r.screenPipelineDescriptor(nil, gputypes.BlendStateAlpha(), "screen")
		if world.Multisample.Count != samples || world.Multisample.Mask != 0xFFFFFFFF {
			t.Fatalf("world multisampling = %+v, want all %d samples", world.Multisample, samples)
		}
		if screen.DepthStencil != nil || screen.Multisample.Count > 1 {
			t.Fatalf("UI must use a single-sample pass without depth: %+v", screen)
		}
	}
}

func TestWorldBillboardPipelineSelectionHonorsBlendAndDepthTest(t *testing.T) {
	alphaRead := &wgpu.RenderPipeline{}
	addRead := &wgpu.RenderPipeline{}
	srcDstRead := &wgpu.RenderPipeline{}
	alphaNoDepth := &wgpu.RenderPipeline{}
	addNoDepth := &wgpu.RenderPipeline{}
	srcDstNoDepth := &wgpu.RenderPipeline{}
	renderer := &gpuRenderer{
		billboardAlphaRead:     alphaRead,
		billboardAddRead:       addRead,
		billboardSrcDstRead:    srcDstRead,
		billboardAlphaNoDepth:  alphaNoDepth,
		billboardAddNoDepth:    addNoDepth,
		billboardSrcDstNoDepth: srcDstNoDepth,
	}
	cases := []struct {
		name      string
		blend     Blend
		depthTest bool
		want      *wgpu.RenderPipeline
	}{
		{name: "alpha read", depthTest: true, want: alphaRead},
		{name: "add read", blend: BlendLighter, depthTest: true, want: addRead},
		{name: "src-dst read", blend: BlendSrcAlphaDstAlpha, depthTest: true, want: srcDstRead},
		{name: "alpha no depth", want: alphaNoDepth},
		{name: "add no depth", blend: BlendLighter, want: addNoDepth},
		{name: "src-dst no depth", blend: BlendSrcAlphaDstAlpha, want: srcDstNoDepth},
	}
	for _, tc := range cases {
		if got := renderer.worldBillboardPipelineFor(tc.blend, tc.depthTest); got != tc.want {
			t.Fatalf("%s pipeline = %p, want %p", tc.name, got, tc.want)
		}
	}
}

func TestBuildFrameAppliesScreenScaleTo2DVertices(t *testing.T) {
	screen := NewFrame(320, 240)
	screen.BeginFrame()
	screen.SetScreenScale(2, 3)
	var opts DrawImageOptions
	opts.GeoM.Scale(10, 5)
	screen.DrawImage(WhiteImage(), &opts)

	frame := (&gpuRenderer{}).buildFrame(screen)
	if len(frame.floats) < screenVertexFloatCount*4 {
		t.Fatalf("frame floats = %d, want at least one quad", len(frame.floats))
	}
	got := [][2]float32{
		{frame.floats[0], frame.floats[1]},
		{frame.floats[screenVertexFloatCount], frame.floats[screenVertexFloatCount+1]},
		{frame.floats[screenVertexFloatCount*2], frame.floats[screenVertexFloatCount*2+1]},
		{frame.floats[screenVertexFloatCount*3], frame.floats[screenVertexFloatCount*3+1]},
	}
	want := [][2]float32{{0, 0}, {20, 0}, {0, 15}, {20, 15}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("vertex %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestFramebufferScaleFallsBackToOneForInvalidFramebuffer(t *testing.T) {
	scaleX, scaleY := framebufferScale(800, 600, 0, 0)
	if scaleX != 1 || scaleY != 1 {
		t.Fatalf("invalid framebuffer scale = %v,%v, want 1,1", scaleX, scaleY)
	}
	scaleX, scaleY = framebufferScale(800, 600, 1200, 900)
	if scaleX != 1.5 || scaleY != 1.5 {
		t.Fatalf("framebuffer scale = %v,%v, want 1.5,1.5", scaleX, scaleY)
	}
}

func TestBlendLighterWeightsSourceByAlpha(t *testing.T) {
	dst := NewImage(1, 1)
	dst.Fill(color.RGBA{R: 10, G: 20, B: 30, A: 40})
	dst.blendPixel(0, 0, color.RGBA{R: 100, G: 80, B: 60, A: 128}, BlendLighter)

	got := dst.RGBA().RGBAAt(0, 0)
	want := color.RGBA{R: 60, G: 60, B: 60, A: 168}
	if got != want {
		t.Fatalf("additive blend = %+v, want %+v", got, want)
	}
}

func TestBlendSrcAlphaDstAlphaUsesBothAlphaFactors(t *testing.T) {
	dst := NewImage(1, 1)
	dst.Fill(color.RGBA{R: 10, G: 20, B: 30, A: 40})
	dst.blendPixel(0, 0, color.RGBA{R: 100, G: 80, B: 60, A: 128}, BlendSrcAlphaDstAlpha)

	got := dst.RGBA().RGBAAt(0, 0)
	want := color.RGBA{R: 52, G: 43, B: 35, A: 71}
	if got != want {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %+v, want %+v", got, want)
	}
}

func TestWorldUniformBytesPacksMatrixAndFog(t *testing.T) {
	camera := Camera3D{
		Enabled: true,
		ViewProjection: [16]float32{
			0: 1, 5: 2, 10: 3, 15: 4,
		},
		Fog: Fog3D{
			Enabled:  true,
			Near:     10,
			Far:      20,
			Strength: 0.5,
			ColorR:   0.25,
			ColorG:   0.5,
			ColorB:   0.75,
		},
	}
	data := worldUniformBytes(camera)
	if len(data) != 96 {
		t.Fatalf("world uniform len = %d, want 96", len(data))
	}
	if got := f32At(data, 0); got != 1 {
		t.Fatalf("matrix[0] = %v, want 1", got)
	}
	if got := f32At(data, 64); got != 10 {
		t.Fatalf("fog near = %v, want 10", got)
	}
	if got := f32At(data, 68); got != 20 {
		t.Fatalf("fog far = %v, want 20", got)
	}
	if got := f32At(data, 72); got != 1 {
		t.Fatalf("fog enabled = %v, want 1", got)
	}
	if got := f32At(data, 76); got != 0.5 {
		t.Fatalf("fog strength = %v, want 0.5", got)
	}
	if got := f32At(data, 80); got != 0.25 {
		t.Fatalf("fog red = %v, want 0.25", got)
	}
}

func TestWorldVertexPackingCarriesFogToggle(t *testing.T) {
	texture := WhiteImage()
	vertices := []Vertex3D{
		{X: 0, Y: 0, Z: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	indices := []uint16{0}
	fogged := NewWorldMesh(vertices, indices, texture, &DrawTrianglesOptions{})
	unfogged := NewWorldMesh(vertices, indices, texture, &DrawTrianglesOptions{DisableFog: true})

	foggedData, _ := worldMeshGPUData(fogged, 1, 1)
	unfoggedData, _ := worldMeshGPUData(unfogged, 1, 1)

	if got := foggedData[13]; got != 1 {
		t.Fatalf("fogged vertex flag = %.1f, want 1", got)
	}
	if got := unfoggedData[13]; got != 0 {
		t.Fatalf("unfogged vertex flag = %.1f, want 0", got)
	}
}

func testWorldMesh(texture *Image, options *DrawTrianglesOptions) *WorldMesh {
	return NewWorldMesh([]Vertex3D{
		{X: 0, Y: 0, Z: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{X: 1, Y: 0, Z: 0, SrcX: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{X: 1, Y: 1, Z: 0, SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}, []uint16{0, 1, 2}, texture, options)
}

func f32At(data []byte, offset int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
}
