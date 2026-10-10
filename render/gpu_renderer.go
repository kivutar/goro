package render

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
)

const (
	screenVertexFloatCount      = 8
	worldVertexFloatCount       = 16
	billboardVertexFloatCount   = 4
	billboardInstanceFloatCount = 24
	screenVertexStride          = screenVertexFloatCount * 4
	worldVertexStride           = worldVertexFloatCount * 4
	billboardVertexStride       = billboardVertexFloatCount * 4
	billboardInstanceStride     = billboardInstanceFloatCount * 4
)

// Depth24Plus produces incorrect model occlusion on affected AMD Vulkan
// setups. Keep the explicit float format consistent across all attachments.
const depthFormat = gputypes.TextureFormatDepth32Float

type gpuRenderer struct {
	dev                    *wgpu.Device
	queue                  *wgpu.Queue
	format                 gputypes.TextureFormat
	bgl                    *wgpu.BindGroupLayout
	worldBGL               *wgpu.BindGroupLayout
	layout                 *wgpu.PipelineLayout
	worldLayout            *wgpu.PipelineLayout
	pipelineAlpha          *wgpu.RenderPipeline
	pipelineAdd            *wgpu.RenderPipeline
	pipelineSrcDst         *wgpu.RenderPipeline
	worldPipelines         map[worldPipelineKey]*wgpu.RenderPipeline
	billboardAlphaRead     *wgpu.RenderPipeline
	billboardAddRead       *wgpu.RenderPipeline
	billboardSrcDstRead    *wgpu.RenderPipeline
	billboardAlphaNoDepth  *wgpu.RenderPipeline
	billboardAddNoDepth    *wgpu.RenderPipeline
	billboardSrcDstNoDepth *wgpu.RenderPipeline
	uniform                *wgpu.Buffer
	worldUniform           *wgpu.Buffer
	samplers               map[samplerKey]*wgpu.Sampler
	lightmapSampler        *wgpu.Sampler
	anisotropy             int
	anisotropySupported    bool
	smoothSprites          bool
	textures               map[*Image]*gpuImageTexture
	imageGroups            map[*ImageGroup]struct{}
	bindGroups             map[bindGroupKey]*wgpu.BindGroup
	worldMeshes            map[*WorldMesh]*gpuWorldMesh
	sampleCount            uint32
	depthTarget            gpuRenderTarget
	msaaTarget             gpuRenderTarget
	bloomEnabled           bool
	bloom                  *gpuBloom
	ssaoEnabled            bool
	ssao                   *gpuSSAO
	worldVertexBuf         dynamicGPUBuffer
	worldIndexBuf          dynamicGPUBuffer
	screenVertexBuf        dynamicGPUBuffer
	screenIndexBuf         dynamicGPUBuffer
	billboardInstanceBuf   dynamicGPUBuffer
	billboardQuadBuf       *wgpu.Buffer
	neutralLightmap        *Image
	worldMeshBatches       []worldMeshBatch
	worldMeshBatchByKey    map[drawBatchKey]int
	worldMeshBatchCache    map[drawBatchKey]*gpuWorldMeshBatch
	worldMeshBatchFloats   []float32
	worldMeshBatchIndices  []uint32
	worldMeshBufferPages   []*worldMeshBufferPage
	worldMeshSubmission    worldMeshSubmissionCache
	worldBillboardBatches  []worldBillboardBatch
	worldBillboardFloats   []float32
	statsEnabled           bool
	statsLast              time.Time
	worldDebug             bool
	worldDebugLast         time.Time
	damageSource           gpucontext.DamageReporter

	worldFrameScratch worldFrameScratch
}

type gpuImageTexture struct {
	tex       *wgpu.Texture
	view      *wgpu.TextureView
	version   uint64
	width     int
	height    int
	mipLevels int
}

type dynamicGPUBuffer struct {
	buf      *wgpu.Buffer
	capacity int
}

type gpuWorldMesh struct {
	vertexBuf  *wgpu.Buffer
	indexBuf   *wgpu.Buffer
	indexCount uint32
	version    uint64
}

type worldMeshBatch struct {
	key      drawBatchKey
	meshes   []*WorldMesh
	revision uint64
}

type worldBillboardBatch struct {
	key           drawBatchKey
	firstInstance uint32
	instanceCount uint32
}

type worldPipelineKey struct {
	blend      Blend
	depthTest  bool
	depthWrite bool
}

type renderPassState struct {
	pipeline  *wgpu.RenderPipeline
	bindGroup *wgpu.BindGroup
	vertexBuf *wgpu.Buffer
	indexBuf  *wgpu.Buffer
}

func (s *renderPassState) setPipeline(pass *wgpu.RenderPassEncoder, pipeline *wgpu.RenderPipeline) {
	if pipeline == nil || s.pipeline == pipeline {
		return
	}
	pass.SetPipeline(pipeline)
	s.pipeline = pipeline
}

func (s *renderPassState) setBindGroup(pass *wgpu.RenderPassEncoder, bg *wgpu.BindGroup) {
	if bg == nil || s.bindGroup == bg {
		return
	}
	pass.SetBindGroup(0, bg, nil)
	s.bindGroup = bg
}

func (s *renderPassState) setVertexBuffer(pass *wgpu.RenderPassEncoder, buf *wgpu.Buffer) {
	if buf == nil || s.vertexBuf == buf {
		return
	}
	pass.SetVertexBuffer(0, buf, 0)
	s.vertexBuf = buf
}

func (s *renderPassState) setIndexBuffer(pass *wgpu.RenderPassEncoder, buf *wgpu.Buffer) {
	if buf == nil || s.indexBuf == buf {
		return
	}
	pass.SetIndexBuffer(buf, gputypes.IndexFormatUint32, 0)
	s.indexBuf = buf
}

type samplerKey struct {
	filter     Filter
	address    Address
	anisotropy uint16
}

type bindGroupKey struct {
	texture      *gpuImageTexture
	lightTexture *gpuImageTexture
	sampler      *wgpu.Sampler
	layout       *wgpu.BindGroupLayout
}

func newGPURenderer(ctx *gogpu.Context, app *gogpu.App, cfg config.RenderConfig) (*gpuRenderer, error) {
	provider := app.DeviceProvider()
	if provider == nil || provider.Device() == nil {
		return nil, fmt.Errorf("gogpu device provider is not ready")
	}
	r := &gpuRenderer{
		dev:          provider.Device(),
		queue:        provider.Queue(),
		format:       provider.SurfaceFormat(),
		samplers:     make(map[samplerKey]*wgpu.Sampler),
		textures:     make(map[*Image]*gpuImageTexture),
		bindGroups:   make(map[bindGroupKey]*wgpu.BindGroup),
		worldMeshes:  make(map[*WorldMesh]*gpuWorldMesh),
		statsEnabled: cfg.Stats,
		worldDebug:   cfg.WorldDebugStats,
		damageSource: ctx.RegisterDamageSource("goro"),
	}
	r.anisotropy = cfg.Anisotropy
	r.smoothSprites = cfg.SmoothSprites
	r.bloomEnabled = cfg.Bloom
	r.ssaoEnabled = cfg.SSAO
	r.sampleCount = 1
	if cfg.MSAA {
		r.sampleCount = 4
	}
	r.anisotropySupported = app.GPUContextProvider().DownlevelCapabilities().Flags.Contains(gputypes.DownlevelFlagsAnisotropicFiltering)
	if r.queue == nil {
		r.queue = r.dev.Queue()
	}
	if err := r.init(ctx); err != nil {
		r.release()
		return nil, err
	}
	return r, nil
}

func (r *gpuRenderer) init(_ *gogpu.Context) error {
	var err error
	// A separate bilinear sampler keeps the lightmap atlas from sampling into
	// neighboring tiles when the surface texture uses anisotropic filtering.
	r.lightmapSampler, err = r.sampler(DrawTrianglesOptions{Filter: FilterLinear, Address: AddressClampToEdge}, nil)
	if err != nil {
		return err
	}
	r.uniform, err = r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-screen-uniform",
		Size:  16,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return err
	}
	r.worldUniform, err = r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-world-uniform",
		Size:  96,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return err
	}
	shader, err := r.dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "goro-screen-shader",
		WGSL:  screenShaderWGSL,
	})
	if err != nil {
		return fmt.Errorf("create screen shader: %w", err)
	}
	defer shader.Release()
	worldShader, err := r.dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "goro-world-shader",
		WGSL:  worldShaderWGSL,
	})
	if err != nil {
		return fmt.Errorf("create world shader: %w", err)
	}
	defer worldShader.Release()
	billboardShader, err := r.dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "goro-world-billboard-shader",
		WGSL:  worldBillboardShaderWGSL,
	})
	if err != nil {
		return fmt.Errorf("create world billboard shader: %w", err)
	}
	defer billboardShader.Release()

	r.bgl, err = r.dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "goro-screen-bind-layout",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: 16}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &gputypes.SamplerBindingLayout{Type: gputypes.SamplerBindingTypeFiltering}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
		},
	})
	if err != nil {
		return err
	}
	r.worldBGL, err = r.dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "goro-world-bind-layout",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: 96}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &gputypes.SamplerBindingLayout{Type: gputypes.SamplerBindingTypeFiltering}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
			{Binding: 3, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
			{Binding: 4, Visibility: wgpu.ShaderStageFragment, Sampler: &gputypes.SamplerBindingLayout{Type: gputypes.SamplerBindingTypeFiltering}},
		},
	})
	if err != nil {
		return err
	}
	r.layout, err = r.dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            "goro-screen-pipeline-layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{r.bgl},
	})
	if err != nil {
		return err
	}
	r.worldLayout, err = r.dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            "goro-world-pipeline-layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{r.worldBGL},
	})
	if err != nil {
		return err
	}
	r.pipelineAlpha, err = r.createPipeline(shader, gputypes.BlendStateAlpha(), "goro-screen-pipeline-alpha")
	if err != nil {
		return err
	}
	add := gputypes.BlendState{
		Color: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorSrcAlpha, DstFactor: gputypes.BlendFactorOne, Operation: gputypes.BlendOperationAdd},
		Alpha: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorOne, DstFactor: gputypes.BlendFactorOne, Operation: gputypes.BlendOperationAdd},
	}
	r.pipelineAdd, err = r.createPipeline(shader, add, "goro-screen-pipeline-add")
	if err != nil {
		return err
	}
	srcDst := gputypes.BlendState{
		Color: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorSrcAlpha, DstFactor: gputypes.BlendFactorDstAlpha, Operation: gputypes.BlendOperationAdd},
		Alpha: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorSrcAlpha, DstFactor: gputypes.BlendFactorDstAlpha, Operation: gputypes.BlendOperationAdd},
	}
	r.pipelineSrcDst, err = r.createPipeline(shader, srcDst, "goro-screen-pipeline-src-alpha-dst-alpha")
	if err != nil {
		return err
	}
	r.worldPipelines = make(map[worldPipelineKey]*wgpu.RenderPipeline)
	for blend, state := range []gputypes.BlendState{gputypes.BlendStateAlpha(), add, srcDst} {
		for _, depthTest := range []bool{false, true} {
			for _, depthWrite := range []bool{false, true} {
				key := worldPipelineKey{Blend(blend), depthTest, depthWrite}
				label := fmt.Sprintf("goro-world-pipeline-blend-%d-test-%t-write-%t", blend, depthTest, depthWrite)
				pipeline, err := r.createWorldPipeline(worldShader, state, depthTest, depthWrite, label)
				if err != nil {
					return err
				}
				r.worldPipelines[key] = pipeline
			}
		}
	}
	r.billboardAlphaRead, err = r.createWorldBillboardPipeline(billboardShader, gputypes.BlendStateAlpha(), true, "goro-world-billboard-alpha-read")
	if err != nil {
		return err
	}
	r.billboardAddRead, err = r.createWorldBillboardPipeline(billboardShader, add, true, "goro-world-billboard-add-read")
	if err != nil {
		return err
	}
	r.billboardSrcDstRead, err = r.createWorldBillboardPipeline(billboardShader, srcDst, true, "goro-world-billboard-src-alpha-dst-alpha-read")
	if err != nil {
		return err
	}
	r.billboardAlphaNoDepth, err = r.createWorldBillboardPipeline(billboardShader, gputypes.BlendStateAlpha(), false, "goro-world-billboard-alpha-no-depth")
	if err != nil {
		return err
	}
	r.billboardAddNoDepth, err = r.createWorldBillboardPipeline(billboardShader, add, false, "goro-world-billboard-add-no-depth")
	if err != nil {
		return err
	}
	r.billboardSrcDstNoDepth, err = r.createWorldBillboardPipeline(billboardShader, srcDst, false, "goro-world-billboard-src-alpha-dst-alpha-no-depth")
	if err != nil {
		return err
	}
	r.billboardQuadBuf, err = r.createBillboardQuadBuffer()
	return err
}

func (r *gpuRenderer) createPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(r.screenPipelineDescriptor(shader, blend, label))
}

func (r *gpuRenderer) screenPipelineDescriptor(shader *wgpu.ShaderModule, blend gputypes.BlendState, label string) *wgpu.RenderPipelineDescriptor {
	return &wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.layout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []gputypes.VertexBufferLayout{{
				ArrayStride: screenVertexStride,
				StepMode:    gputypes.VertexStepModeVertex,
				Attributes: []gputypes.VertexAttribute{
					{Format: gputypes.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
					{Format: gputypes.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1},
					{Format: gputypes.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2},
				},
			}},
		},
		Primitive: gputypes.PrimitiveState{
			Topology:  gputypes.PrimitiveTopologyTriangleList,
			FrontFace: gputypes.FrontFaceCCW,
			CullMode:  gputypes.CullModeNone,
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []gputypes.ColorTargetState{{
				Format:    r.format,
				Blend:     &blend,
				WriteMask: gputypes.ColorWriteMaskAll,
			}},
		},
	}
}

func (r *gpuRenderer) createWorldPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, depthTest, depthWrite bool, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(r.worldPipelineDescriptor(shader, blend, depthTest, depthWrite, label))
}

func (r *gpuRenderer) worldPipelineDescriptor(shader *wgpu.ShaderModule, blend gputypes.BlendState, depthTest, depthWrite bool, label string) *wgpu.RenderPipelineDescriptor {
	return &wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.worldLayout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []gputypes.VertexBufferLayout{{
				ArrayStride: worldVertexStride,
				StepMode:    gputypes.VertexStepModeVertex,
				Attributes: []gputypes.VertexAttribute{
					{Format: gputypes.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
					{Format: gputypes.VertexFormatFloat32x2, Offset: 12, ShaderLocation: 1},
					{Format: gputypes.VertexFormatFloat32x4, Offset: 20, ShaderLocation: 2},
					{Format: gputypes.VertexFormatFloat32x3, Offset: 36, ShaderLocation: 3},
					{Format: gputypes.VertexFormatFloat32, Offset: 48, ShaderLocation: 4},
					{Format: gputypes.VertexFormatFloat32, Offset: 52, ShaderLocation: 5},
					{Format: gputypes.VertexFormatFloat32x2, Offset: 56, ShaderLocation: 6},
				},
			}},
		},
		Primitive: gputypes.PrimitiveState{
			Topology:  gputypes.PrimitiveTopologyTriangleList,
			FrontFace: gputypes.FrontFaceCCW,
			CullMode:  gputypes.CullModeNone,
		},
		Multisample: gputypes.MultisampleState{Count: r.sampleCount, Mask: 0xFFFFFFFF},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            depthFormat,
			DepthWriteEnabled: depthWrite,
			DepthCompare:      worldDepthCompare(depthTest),
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []gputypes.ColorTargetState{{
				Format:    r.format,
				Blend:     &blend,
				WriteMask: gputypes.ColorWriteMaskAll,
			}},
		},
	}
}

func (r *gpuRenderer) createWorldBillboardPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, depthTest bool, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.worldLayout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []gputypes.VertexBufferLayout{
				{
					ArrayStride: billboardVertexStride,
					StepMode:    gputypes.VertexStepModeVertex,
					Attributes: []gputypes.VertexAttribute{
						{Format: gputypes.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
						{Format: gputypes.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1},
					},
				},
				{
					ArrayStride: billboardInstanceStride,
					StepMode:    gputypes.VertexStepModeInstance,
					Attributes: []gputypes.VertexAttribute{
						{Format: gputypes.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 2},
						{Format: gputypes.VertexFormatFloat32x3, Offset: 12, ShaderLocation: 3},
						{Format: gputypes.VertexFormatFloat32x3, Offset: 24, ShaderLocation: 4},
						{Format: gputypes.VertexFormatFloat32x3, Offset: 36, ShaderLocation: 5},
						{Format: gputypes.VertexFormatFloat32x4, Offset: 48, ShaderLocation: 6},
						{Format: gputypes.VertexFormatFloat32x4, Offset: 64, ShaderLocation: 7},
						{Format: gputypes.VertexFormatFloat32, Offset: 80, ShaderLocation: 8},
						{Format: gputypes.VertexFormatFloat32, Offset: 84, ShaderLocation: 9},
					},
				},
			},
		},
		Primitive: gputypes.PrimitiveState{
			Topology:  gputypes.PrimitiveTopologyTriangleList,
			FrontFace: gputypes.FrontFaceCCW,
			CullMode:  gputypes.CullModeNone,
		},
		Multisample: gputypes.MultisampleState{Count: r.sampleCount, Mask: 0xFFFFFFFF},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            depthFormat,
			DepthWriteEnabled: false,
			DepthCompare:      worldDepthCompare(depthTest),
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []gputypes.ColorTargetState{{
				Format:    r.format,
				Blend:     &blend,
				WriteMask: gputypes.ColorWriteMaskAll,
			}},
		},
	})
}

func worldDepthCompare(depthTest bool) gputypes.CompareFunction {
	if depthTest {
		return gputypes.CompareFunctionLessEqual
	}
	return gputypes.CompareFunctionAlways
}

func (r *gpuRenderer) Draw(ctx *gogpu.Context, screen *Frame) (bool, error) {
	r.releaseImageGroups()
	if screen == nil {
		return false, nil
	}
	surface := ctx.SurfaceView()
	if surface == nil {
		return false, nil
	}
	width, height := ctx.FramebufferSize()
	if width <= 0 || height <= 0 {
		b := screen.Bounds()
		width, height = b.Dx(), b.Dy()
	}
	if width <= 0 || height <= 0 {
		return false, nil
	}
	for _, upload := range screen.imageUploads {
		if _, err := r.ensureTexture(upload); err != nil {
			return false, err
		}
	}
	if err := r.queue.WriteBuffer(r.uniform, 0, uniformBytes(float32(width), float32(height))); err != nil {
		return false, fmt.Errorf("upload screen uniform: %w", err)
	}
	if screen.camera.Enabled {
		if err := r.queue.WriteBuffer(r.worldUniform, 0, worldUniformBytes(screen.camera)); err != nil {
			return false, fmt.Errorf("upload world uniform: %w", err)
		}
	}
	if screen.camera.Enabled {
		if err := r.depthTarget.ensure(r.dev, width, height, depthFormat, r.sampleCount, wgpu.TextureUsageRenderAttachment); err != nil {
			return false, fmt.Errorf("create world depth: %w", err)
		}
		if r.sampleCount > 1 {
			if err := r.msaaTarget.ensure(r.dev, width, height, r.format, r.sampleCount, wgpu.TextureUsageRenderAttachment); err != nil {
				return false, fmt.Errorf("create MSAA color: %w", err)
			}
		}
	}
	worldTarget := surface
	bloomActive := r.bloomEnabled && screen.camera.Enabled
	if bloomActive {
		if r.bloom == nil {
			var err error
			r.bloom, err = newGPUBloom(r.dev, r.format)
			if err != nil {
				return false, fmt.Errorf("create bloom: %w", err)
			}
		}
		if err := r.bloom.ensure(r.dev, r.format, width, height); err != nil {
			return false, fmt.Errorf("prepare bloom: %w", err)
		}
		worldTarget = r.bloom.targets[0].view
	} else if r.bloom != nil && !r.bloomEnabled {
		r.bloom.releaseTargets()
	}
	world := r.buildWorldFrame(screen)
	ssaoActive := r.ssaoEnabled && screen.camera.Enabled
	if ssaoActive {
		if r.ssao == nil {
			var err error
			r.ssao, err = r.newSSAO()
			if err != nil {
				return false, fmt.Errorf("create SSAO: %w", err)
			}
		}
		if err := r.ssao.ensure(r.dev, width, height); err != nil {
			return false, fmt.Errorf("prepare SSAO: %w", err)
		}
		if err := r.ssao.uploadCamera(r.queue, screen.camera); err != nil {
			return false, err
		}
	} else if r.ssao != nil && !r.ssaoEnabled {
		r.ssao.releaseTargets()
	}
	frame := r.buildFrame(screen)
	meshBatches := r.depthWriteWorldMeshBatches(screen)
	if r.statsEnabled && time.Since(r.statsLast) >= time.Second {
		glog.Debugf("render stats world_commands=%d world_mesh_commands=%d world_mesh_batches=%d world_mesh_buffer_pages=%d world_billboards=%d retained_world_meshes=%d world_batches=%d world_vertices=%d world_indices=%d commands=%d batches=%d vertices=%d indices=%d textures=%d bindgroups=%d", len(screen.worldCommands), len(screen.worldMeshes), len(meshBatches), len(r.worldMeshBufferPages), len(screen.worldBillboards), len(r.worldMeshes), len(world.batches), len(world.floats)/worldVertexFloatCount, len(world.indices), len(screen.commands), len(frame.batches), len(frame.floats)/screenVertexFloatCount, len(frame.indices), len(r.textures), len(r.bindGroups))
		r.statsLast = time.Now()
	}
	if r.worldDebug && time.Since(r.worldDebugLast) >= time.Second {
		r.logWorldDebug(screen)
		r.worldDebugLast = time.Now()
	}
	var worldVertexBuf, worldIndexBuf, vertexBuf, indexBuf *wgpu.Buffer
	var err error
	if len(world.floats) > 0 && len(world.indices) > 0 {
		worldVertexBuf, err = r.dynamicBuffer(&r.worldVertexBuf, "goro-world-vertices", len(world.floats)*4, wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst, floatBytes(world.floats))
		if err != nil {
			return false, err
		}
		worldIndexBuf, err = r.dynamicBuffer(&r.worldIndexBuf, "goro-world-indices", len(world.indices)*4, wgpu.BufferUsageIndex|wgpu.BufferUsageCopyDst, u32Bytes(world.indices))
		if err != nil {
			return false, err
		}
	}
	if len(frame.floats) > 0 && len(frame.indices) > 0 {
		vertexBuf, err = r.dynamicBuffer(&r.screenVertexBuf, "goro-screen-vertices", len(frame.floats)*4, wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst, floatBytes(frame.floats))
		if err != nil {
			return false, err
		}
		indexBuf, err = r.dynamicBuffer(&r.screenIndexBuf, "goro-screen-indices", len(frame.indices)*4, wgpu.BufferUsageIndex|wgpu.BufferUsageCopyDst, u32Bytes(frame.indices))
		if err != nil {
			return false, err
		}
	}

	enc, err := r.dev.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "goro-screen-encoder"})
	if err != nil {
		return false, err
	}
	defer enc.DiscardEncoding()
	if ssaoActive {
		if err := r.drawSSAO(enc, meshBatches, world, worldVertexBuf, worldIndexBuf); err != nil {
			return false, fmt.Errorf("draw SSAO: %w", err)
		}
	}
	clear := clearValue(screen.clear)
	if screen.camera.Enabled {
		pass, err := enc.BeginRenderPass(r.worldPassDescriptor(worldTarget, clear))
		if err != nil {
			return false, err
		}
		if err := r.drawWorldOpaque(pass, meshBatches, world, worldVertexBuf, worldIndexBuf, nil); err != nil {
			_ = pass.End()
			return false, err
		}
		if ssaoActive {
			pass.SetPipeline(r.ssao.composite)
			pass.SetBindGroup(0, r.ssao.groups[1], nil)
			pass.Draw(gputypes.DrawArgs{VertexCount: 3, InstanceCount: 1})
		}
		worldState := renderPassState{}
		if worldVertexBuf != nil && worldIndexBuf != nil {
			worldState.setVertexBuffer(pass, worldVertexBuf)
			worldState.setIndexBuffer(pass, worldIndexBuf)
			for _, batch := range world.batches {
				if batch.indexCount == 0 || batch.key.options.DepthWrite {
					continue
				}
				if err := r.drawWorldBatch(pass, batch, &worldState, nil); err != nil {
					_ = pass.End()
					return false, err
				}
			}
		}
		for _, meshCommand := range screen.worldMeshes {
			mesh := meshCommand.Mesh
			if mesh == nil || mesh.options.DepthWrite {
				continue
			}
			if err := r.drawWorldMesh(pass, mesh, &worldState); err != nil {
				_ = pass.End()
				return false, err
			}
		}
		if err := r.drawWorldBillboards(pass, screen.worldBillboards); err != nil {
			_ = pass.End()
			return false, err
		}
		if err := pass.End(); err != nil {
			return false, err
		}
	}
	if bloomActive {
		if err := r.bloom.draw(enc, surface); err != nil {
			return false, fmt.Errorf("draw bloom: %w", err)
		}
	}
	// Resolve the world before drawing the UI at native resolution. The UI
	// pass has no depth attachment and always uses single-sample pipelines.
	load := gputypes.LoadOpClear
	if screen.camera.Enabled {
		load = gputypes.LoadOpLoad
	}
	pass, err := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View: surface, LoadOp: load, StoreOp: gputypes.StoreOpStore, ClearValue: clear,
		}},
	})
	if err != nil {
		return false, err
	}
	if vertexBuf != nil && indexBuf != nil {
		pass.SetVertexBuffer(0, vertexBuf, 0)
		pass.SetIndexBuffer(indexBuf, gputypes.IndexFormatUint32, 0)
	}
	for _, batch := range frame.batches {
		if batch.indexCount == 0 {
			continue
		}
		tex, err := r.ensureTexture(batch.key.texture)
		if err != nil {
			_ = pass.End()
			return false, err
		}
		sampler, err := r.sampler(batch.key.options, batch.key.texture)
		if err != nil {
			_ = pass.End()
			return false, err
		}
		bg, err := r.bindGroup(r.bgl, r.uniform, 16, tex, sampler)
		if err != nil {
			_ = pass.End()
			return false, err
		}
		pass.SetPipeline(r.pipeline(batch.key.options.Blend))
		pass.SetBindGroup(0, bg, nil)
		pass.DrawIndexed(gputypes.DrawIndexedArgs{
			IndexCount:    batch.indexCount,
			InstanceCount: 1,
			FirstIndex:    batch.firstIndex,
		})
	}
	if err := pass.End(); err != nil {
		return false, err
	}
	cmd, err := enc.Finish()
	if err != nil {
		return false, err
	}
	_, err = r.queue.Submit(cmd)
	if err != nil {
		return false, err
	}
	if r.damageSource != nil {
		r.damageSource.ReportDamage()
	}
	return true, nil
}

func (r *gpuRenderer) logWorldDebug(screen *Frame) {
	if screen == nil {
		glog.Debugf("world debug empty camera=false commands=0")
		return
	}
	if !screen.camera.Enabled || len(screen.worldCommands) == 0 {
		glog.Debugf("world debug empty camera=%t commands=%d", screen.camera.Enabled, len(screen.worldCommands))
		return
	}
	m := screen.camera.ViewProjection
	count := 0
	inside := 0
	front := 0
	minX, minY, minZ := float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY, maxZ := float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))
	sumR, sumG, sumB, sumA := 0.0, 0.0, 0.0, 0.0
	for _, cmd := range screen.worldCommands {
		for _, v := range cmd.Vertices {
			clipX := m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]
			clipY := m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]
			clipZ := m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]
			clipW := m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]
			if clipW > 0 {
				front++
				ndcX := clipX / clipW
				ndcY := clipY / clipW
				ndcZ := (clipZ/clipW)*0.5 + 0.5
				minX, minY, minZ = minFloat32(minX, ndcX), minFloat32(minY, ndcY), minFloat32(minZ, ndcZ)
				maxX, maxY, maxZ = maxFloat32(maxX, ndcX), maxFloat32(maxY, ndcY), maxFloat32(maxZ, ndcZ)
				if ndcX >= -1 && ndcX <= 1 && ndcY >= -1 && ndcY <= 1 && ndcZ >= 0 && ndcZ <= 1 {
					inside++
				}
			}
			sumR += float64(v.ColorR)
			sumG += float64(v.ColorG)
			sumB += float64(v.ColorB)
			sumA += float64(v.ColorA)
			count++
		}
	}
	if count == 0 {
		glog.Debugf("world debug no vertices commands=%d", len(screen.worldCommands))
		return
	}
	inv := 1 / float64(count)
	glog.Debugf("world debug vertices=%d front=%d inside=%d ndc=(%.2f..%.2f, %.2f..%.2f, %.2f..%.2f) avg_rgba=(%.3f,%.3f,%.3f,%.3f)",
		count, front, inside, minX, maxX, minY, maxY, minZ, maxZ, sumR*inv, sumG*inv, sumB*inv, sumA*inv)
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (r *gpuRenderer) ensureTexture(img *Image) (*gpuImageTexture, error) {
	if img == nil || img.pix == nil {
		return nil, fmt.Errorf("nil render texture")
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	levels := 1 + len(img.mipmaps)
	existing := r.textures[img]
	if existing != nil {
		if existing.width == w && existing.height == h && existing.mipLevels == levels {
			if existing.version != img.version {
				if err := r.uploadImage(existing, img); err != nil {
					return nil, err
				}
			}
			return existing, nil
		}
		r.releaseTexture(existing)
		delete(r.textures, img)
	}
	tex, err := r.dev.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "goro-image-texture",
		Size:          wgpu.Extent3D{Width: uint32(w), Height: uint32(h), DepthOrArrayLayers: 1},
		MipLevelCount: uint32(levels),
		SampleCount:   1,
		Dimension:     gputypes.TextureDimension2D,
		Format:        gputypes.TextureFormatRGBA8Unorm,
		Usage:         wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create render texture: %w", err)
	}
	out := &gpuImageTexture{tex: tex, width: w, height: h, mipLevels: levels}
	if err := r.uploadImage(out, img); err != nil {
		out.release()
		return nil, err
	}
	out.view, err = r.dev.CreateTextureView(tex, nil)
	if err != nil {
		out.release()
		return nil, fmt.Errorf("create render texture view: %w", err)
	}
	r.textures[img] = out
	if img.group != nil {
		if r.imageGroups == nil {
			r.imageGroups = make(map[*ImageGroup]struct{})
		}
		r.imageGroups[img.group] = struct{}{}
	}
	return out, nil
}

func (r *gpuRenderer) uploadImage(tex *gpuImageTexture, img *Image) error {
	for level := 0; level < tex.mipLevels; level++ {
		pix := img.pix
		if level > 0 {
			pix = img.mipmaps[level-1]
		}
		err := r.queue.WriteTexture(
			&wgpu.ImageCopyTexture{Texture: tex.tex, MipLevel: uint32(level), Aspect: gputypes.TextureAspectAll},
			pix.Pix,
			&wgpu.ImageDataLayout{BytesPerRow: uint32(pix.Stride), RowsPerImage: uint32(pix.Rect.Dy())},
			&wgpu.Extent3D{Width: uint32(pix.Rect.Dx()), Height: uint32(pix.Rect.Dy()), DepthOrArrayLayers: 1},
		)
		if err != nil {
			return fmt.Errorf("upload render texture mip %d: %w", level, err)
		}
	}
	tex.version = img.version
	return nil
}

func (r *gpuRenderer) ensureBatchLightTexture(key drawBatchKey) (*gpuImageTexture, error) {
	if key.lightTexture != nil && key.lightTexture.pix != nil {
		return r.ensureTexture(key.lightTexture)
	}
	return r.ensureTexture(r.neutralLightmapImage())
}

func (r *gpuRenderer) ensureMeshLightTexture(mesh *WorldMesh) (*gpuImageTexture, error) {
	if mesh != nil && mesh.lightTexture != nil && mesh.lightTexture.pix != nil {
		return r.ensureTexture(mesh.lightTexture)
	}
	if mesh == nil {
		return nil, fmt.Errorf("nil world mesh")
	}
	return r.ensureTexture(r.neutralLightmapImage())
}

func (r *gpuRenderer) neutralLightmapImage() *Image {
	if r.neutralLightmap == nil {
		r.neutralLightmap = NewImage(1, 1)
		r.neutralLightmap.Fill(color.RGBA{A: 255})
	}
	return r.neutralLightmap
}

func (r *gpuRenderer) releaseTexture(tex *gpuImageTexture) {
	if tex == nil {
		return
	}
	for key, bg := range r.bindGroups {
		if key.texture == tex || key.lightTexture == tex {
			if bg != nil {
				bg.Release()
			}
			delete(r.bindGroups, key)
		}
	}
	tex.release()
}

func (tex *gpuImageTexture) release() {
	if tex.view != nil {
		tex.view.Release()
		tex.view = nil
	}
	if tex.tex != nil {
		tex.tex.Release()
		tex.tex = nil
	}
}

func (r *gpuRenderer) releaseImageTexture(img *Image) {
	if r == nil || img == nil {
		return
	}
	existing := r.textures[img]
	if existing == nil {
		return
	}
	r.releaseTexture(existing)
	delete(r.textures, img)
}

func imageSamplerKey(opts DrawTrianglesOptions, texture *Image, anisotropy int) samplerKey {
	key := samplerKey{filter: opts.Filter, address: opts.Address, anisotropy: 1}
	if opts.Filter == FilterLinear && texture != nil && len(texture.mipmaps) > 0 {
		key.anisotropy = uint16(max(1, min(16, anisotropy)))
	}
	return key
}

func (r *gpuRenderer) sampler(opts DrawTrianglesOptions, texture *Image) (*wgpu.Sampler, error) {
	opts.Filter = opts.Filter.resolveSprite(r.smoothSprites)
	anisotropy := 1
	if r.anisotropySupported {
		anisotropy = r.anisotropy
	}
	key := imageSamplerKey(opts, texture, anisotropy)
	if sampler := r.samplers[key]; sampler != nil {
		return sampler, nil
	}
	sampler, err := r.dev.CreateSampler(&wgpu.SamplerDescriptor{
		Label:        "goro-screen-sampler",
		AddressModeU: gpuAddress(opts.Address),
		AddressModeV: gpuAddress(opts.Address),
		AddressModeW: gputypes.AddressModeClampToEdge,
		MagFilter:    gpuFilter(opts.Filter),
		MinFilter:    gpuFilter(opts.Filter),
		MipmapFilter: gpuFilter(opts.Filter),
		LodMaxClamp:  32,
		Anisotropy:   key.anisotropy,
	})
	if err != nil {
		return nil, err
	}
	r.samplers[key] = sampler
	return sampler, nil
}

func (r *gpuRenderer) bindGroup(layout *wgpu.BindGroupLayout, uniform *wgpu.Buffer, uniformSize uint64, tex *gpuImageTexture, sampler *wgpu.Sampler) (*wgpu.BindGroup, error) {
	key := bindGroupKey{texture: tex, sampler: sampler, layout: layout}
	if bg := r.bindGroups[key]; bg != nil {
		return bg, nil
	}
	bg, err := r.dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "goro-screen-bind-group",
		Layout: layout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: uniform, Size: uniformSize},
			{Binding: 1, Sampler: sampler},
			{Binding: 2, TextureView: tex.view},
		},
	})
	if err != nil {
		return nil, err
	}
	r.bindGroups[key] = bg
	return bg, nil
}

func (r *gpuRenderer) bindWorldGroup(uniform *wgpu.Buffer, uniformSize uint64, tex, lightTex *gpuImageTexture, sampler *wgpu.Sampler) (*wgpu.BindGroup, error) {
	key := bindGroupKey{texture: tex, lightTexture: lightTex, sampler: sampler, layout: r.worldBGL}
	if bg := r.bindGroups[key]; bg != nil {
		return bg, nil
	}
	bg, err := r.dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "goro-world-bind-group",
		Layout: r.worldBGL,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: uniform, Size: uniformSize},
			{Binding: 1, Sampler: sampler},
			{Binding: 2, TextureView: tex.view},
			{Binding: 3, TextureView: lightTex.view},
			{Binding: 4, Sampler: r.lightmapSampler},
		},
	})
	if err != nil {
		return nil, err
	}
	r.bindGroups[key] = bg
	return bg, nil
}

func (r *gpuRenderer) drawWorldMesh(pass *wgpu.RenderPassEncoder, mesh *WorldMesh, state *renderPassState) error {
	if mesh == nil || mesh.texture == nil || mesh.texture.pix == nil || len(mesh.vertices) == 0 || len(mesh.indices) == 0 {
		return nil
	}
	if state == nil {
		state = &renderPassState{}
	}
	gpuMesh, err := r.ensureWorldMesh(mesh)
	if err != nil {
		return err
	}
	tex, err := r.ensureTexture(mesh.texture)
	if err != nil {
		return err
	}
	lightTex, err := r.ensureMeshLightTexture(mesh)
	if err != nil {
		return err
	}
	sampler, err := r.sampler(mesh.options, mesh.texture)
	if err != nil {
		return err
	}
	bg, err := r.bindWorldGroup(r.worldUniform, 96, tex, lightTex, sampler)
	if err != nil {
		return err
	}
	state.setPipeline(pass, r.worldPipelineFor(mesh.options))
	state.setBindGroup(pass, bg)
	state.setVertexBuffer(pass, gpuMesh.vertexBuf)
	state.setIndexBuffer(pass, gpuMesh.indexBuf)
	pass.DrawIndexed(gputypes.DrawIndexedArgs{
		IndexCount:    gpuMesh.indexCount,
		InstanceCount: 1,
	})
	return nil
}

func (r *gpuRenderer) depthWriteWorldMeshBatches(screen *Frame) []worldMeshBatch {
	var commands []WorldMeshCommand
	if screen != nil {
		commands = screen.worldMeshes
	}
	if r.worldMeshSubmission.matches(commands) {
		// A batch growing during the previous draw may have left an empty page.
		r.pruneWorldMeshBufferPages()
		return r.worldMeshBatches
	}
	r.worldMeshSubmission.remember(commands)
	for i := range r.worldMeshBatches {
		r.worldMeshBatches[i].key = drawBatchKey{}
		clear(r.worldMeshBatches[i].meshes)
		r.worldMeshBatches[i].meshes = r.worldMeshBatches[i].meshes[:0]
	}
	r.worldMeshBatches = r.worldMeshBatches[:0]
	commandCount := len(commands)
	if r.worldMeshBatchByKey == nil {
		r.worldMeshBatchByKey = make(map[drawBatchKey]int, commandCount)
	} else {
		clear(r.worldMeshBatchByKey)
	}
	if commandCount == 0 {
		r.pruneWorldMeshBatchCache()
		return nil
	}
	for _, meshCommand := range commands {
		mesh := meshCommand.Mesh
		if mesh == nil || !mesh.options.DepthWrite || mesh.texture == nil || mesh.texture.pix == nil || len(mesh.vertices) == 0 || len(mesh.indices) == 0 {
			continue
		}
		key := drawBatchKey{texture: mesh.texture, lightTexture: mesh.lightTexture, options: mesh.options}
		batchIndex, ok := r.worldMeshBatchByKey[key]
		if !ok {
			batchIndex = len(r.worldMeshBatches)
			if batchIndex < cap(r.worldMeshBatches) {
				r.worldMeshBatches = r.worldMeshBatches[:batchIndex+1]
				r.worldMeshBatches[batchIndex].key = key
			} else {
				r.worldMeshBatches = append(r.worldMeshBatches, worldMeshBatch{key: key})
			}
			r.worldMeshBatches[batchIndex].revision = r.worldMeshSubmission.revision
			r.worldMeshBatchByKey[key] = batchIndex
		}
		r.worldMeshBatches[batchIndex].meshes = append(r.worldMeshBatches[batchIndex].meshes, mesh)
	}
	r.pruneWorldMeshBatchCache()
	return r.worldMeshBatches
}

func (r *gpuRenderer) drawWorldMeshBatch(pass *wgpu.RenderPassEncoder, batch worldMeshBatch, state *renderPassState, pipeline *wgpu.RenderPipeline) error {
	if len(batch.meshes) == 0 || batch.key.texture == nil || batch.key.texture.pix == nil {
		return nil
	}
	if state == nil {
		state = &renderPassState{}
	}
	tex, err := r.ensureTexture(batch.key.texture)
	if err != nil {
		return err
	}
	lightTex, err := r.ensureBatchLightTexture(batch.key)
	if err != nil {
		return err
	}
	sampler, err := r.sampler(batch.key.options, batch.key.texture)
	if err != nil {
		return err
	}
	bg, err := r.bindWorldGroup(r.worldUniform, 96, tex, lightTex, sampler)
	if err != nil {
		return err
	}
	if pipeline == nil {
		pipeline = r.worldPipelineFor(batch.key.options)
	}
	state.setPipeline(pass, pipeline)
	state.setBindGroup(pass, bg)
	gpuBatch, err := r.ensureWorldMeshBatch(batch)
	if err != nil {
		return err
	}
	state.setVertexBuffer(pass, gpuBatch.allocation.page.buf)
	state.setIndexBuffer(pass, gpuBatch.allocation.page.buf)
	pass.DrawIndexed(gputypes.DrawIndexedArgs{
		IndexCount:    gpuBatch.indexCount,
		InstanceCount: 1,
		FirstIndex:    gpuBatch.firstIndex,
		BaseVertex:    int32(gpuBatch.allocation.offset / worldVertexStride),
	})
	return nil
}

func (r *gpuRenderer) drawWorldBillboards(pass *wgpu.RenderPassEncoder, commands []WorldBillboardCommand) error {
	if len(commands) == 0 || r.billboardQuadBuf == nil {
		return nil
	}
	instances, batches := r.buildWorldBillboardBatches(commands)
	if len(instances) == 0 || len(batches) == 0 {
		return nil
	}
	instanceBuf, err := r.dynamicBuffer(
		&r.billboardInstanceBuf,
		"goro-world-billboard-instances",
		len(instances)*4,
		wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst,
		floatBytes(instances),
	)
	if err != nil {
		return err
	}
	pass.SetVertexBuffer(0, r.billboardQuadBuf, 0)
	for _, batch := range batches {
		tex, err := r.ensureTexture(batch.key.texture)
		if err != nil {
			return err
		}
		sampler, err := r.sampler(batch.key.options, batch.key.texture)
		if err != nil {
			return err
		}
		bg, err := r.bindWorldGroup(r.worldUniform, 96, tex, tex, sampler)
		if err != nil {
			return err
		}
		pass.SetPipeline(r.worldBillboardPipelineFor(batch.key.options.Blend, batch.key.options.DepthTest))
		pass.SetBindGroup(0, bg, nil)
		pass.SetVertexBuffer(1, instanceBuf, uint64(batch.firstInstance)*billboardInstanceStride)
		pass.Draw(gputypes.DrawArgs{
			VertexCount:   6,
			InstanceCount: batch.instanceCount,
		})
	}
	return nil
}

func (r *gpuRenderer) buildWorldBillboardBatches(commands []WorldBillboardCommand) ([]float32, []worldBillboardBatch) {
	r.worldBillboardFloats = r.worldBillboardFloats[:0]
	r.worldBillboardBatches = r.worldBillboardBatches[:0]
	for _, cmd := range commands {
		if cmd.Texture == nil || cmd.Texture.pix == nil || cmd.Width <= 0 || cmd.Height <= 0 {
			continue
		}
		key := drawBatchKey{texture: cmd.Texture, options: cmd.Options}
		if len(r.worldBillboardBatches) == 0 || r.worldBillboardBatches[len(r.worldBillboardBatches)-1].key != key {
			r.worldBillboardBatches = append(r.worldBillboardBatches, worldBillboardBatch{
				key:           key,
				firstInstance: uint32(len(r.worldBillboardFloats) / billboardInstanceFloatCount),
			})
		}
		r.worldBillboardFloats = appendBillboardInstanceData(r.worldBillboardFloats, cmd)
		r.worldBillboardBatches[len(r.worldBillboardBatches)-1].instanceCount++
	}
	return r.worldBillboardFloats, r.worldBillboardBatches
}

func (r *gpuRenderer) worldBillboardPipelineFor(blend Blend, depthTest bool) *wgpu.RenderPipeline {
	if depthTest {
		if blend == BlendLighter {
			return r.billboardAddRead
		}
		if blend == BlendSrcAlphaDstAlpha {
			return r.billboardSrcDstRead
		}
		return r.billboardAlphaRead
	}
	if blend == BlendLighter {
		return r.billboardAddNoDepth
	}
	if blend == BlendSrcAlphaDstAlpha {
		return r.billboardSrcDstNoDepth
	}
	return r.billboardAlphaNoDepth
}

func (r *gpuRenderer) createBillboardQuadBuffer() (*wgpu.Buffer, error) {
	data := []float32{
		0, 0, 0, 0,
		1, 0, 1, 0,
		1, 1, 1, 1,
		0, 0, 0, 0,
		1, 1, 1, 1,
		0, 1, 0, 1,
	}
	buf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-world-billboard-quad",
		Size:  uint64(len(data) * 4),
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, err
	}
	if err := r.queue.WriteBuffer(buf, 0, floatBytes(data)); err != nil {
		buf.Release()
		return nil, err
	}
	return buf, nil
}

func billboardInstanceData(cmd WorldBillboardCommand) []float32 {
	return appendBillboardInstanceData(nil, cmd)
}

func appendBillboardInstanceData(dst []float32, cmd WorldBillboardCommand) []float32 {
	fogEnabled := float32(1)
	if cmd.Options.DisableFog {
		fogEnabled = 0
	}
	return append(dst,
		cmd.Center[0], cmd.Center[1], cmd.Center[2],
		cmd.RightAxis[0], cmd.RightAxis[1], cmd.RightAxis[2],
		cmd.UpAxis[0], cmd.UpAxis[1], cmd.UpAxis[2],
		cmd.DepthUpAxis[0], cmd.DepthUpAxis[1], cmd.DepthUpAxis[2],
		cmd.Width, cmd.Height, cmd.AnchorX, cmd.AnchorY,
		saneColor(cmd.ColorR), saneColor(cmd.ColorG), saneColor(cmd.ColorB), saneColor(cmd.ColorA),
		saneDepthBias(cmd.DepthBias), fogEnabled, 0, 0,
	)
}

func (r *gpuRenderer) ensureWorldMesh(mesh *WorldMesh) (*gpuWorldMesh, error) {
	if cached := r.worldMeshes[mesh]; cached != nil && cached.version == mesh.version {
		return cached, nil
	}
	if old := r.worldMeshes[mesh]; old != nil {
		if old.vertexBuf != nil {
			old.vertexBuf.Release()
		}
		if old.indexBuf != nil {
			old.indexBuf.Release()
		}
		delete(r.worldMeshes, mesh)
	}
	width, height := mesh.texture.Bounds().Dx(), mesh.texture.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("world mesh texture has invalid size")
	}
	floats, indices := worldMeshGPUData(mesh, width, height)
	vertexBuf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-retained-world-vertices",
		Size:  uint64(len(floats) * 4),
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, err
	}
	if err := r.queue.WriteBuffer(vertexBuf, 0, floatBytes(floats)); err != nil {
		vertexBuf.Release()
		return nil, err
	}
	indexBuf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-retained-world-indices",
		Size:  uint64(len(indices) * 4),
		Usage: wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		vertexBuf.Release()
		return nil, err
	}
	if err := r.queue.WriteBuffer(indexBuf, 0, u32Bytes(indices)); err != nil {
		vertexBuf.Release()
		indexBuf.Release()
		return nil, err
	}
	gpuMesh := &gpuWorldMesh{
		vertexBuf:  vertexBuf,
		indexBuf:   indexBuf,
		indexCount: uint32(len(indices)),
		version:    mesh.version,
	}
	r.worldMeshes[mesh] = gpuMesh
	return gpuMesh, nil
}

func (r *gpuRenderer) dynamicBuffer(slot *dynamicGPUBuffer, label string, size int, usage gputypes.BufferUsage, data []byte) (*wgpu.Buffer, error) {
	if size <= 0 {
		size = 4
	}
	if slot.buf == nil || slot.capacity < size {
		if slot.buf != nil {
			slot.buf.Release()
			slot.buf = nil
		}
		capacity := nextBufferCapacity(size)
		buf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
			Label: label,
			Size:  uint64(capacity),
			Usage: usage,
		})
		if err != nil {
			return nil, err
		}
		slot.buf = buf
		slot.capacity = capacity
	}
	if len(data) > 0 {
		if err := r.queue.WriteBuffer(slot.buf, 0, data); err != nil {
			return nil, err
		}
	}
	return slot.buf, nil
}

func nextBufferCapacity(size int) int {
	capacity := 4096
	for capacity < size {
		capacity *= 2
	}
	return capacity
}

func (r *gpuRenderer) pipeline(blend Blend) *wgpu.RenderPipeline {
	if blend == BlendLighter {
		return r.pipelineAdd
	}
	if blend == BlendSrcAlphaDstAlpha {
		return r.pipelineSrcDst
	}
	return r.pipelineAlpha
}

func (r *gpuRenderer) worldPipelineFor(options DrawTrianglesOptions) *wgpu.RenderPipeline {
	blend := options.Blend
	if blend != BlendLighter && blend != BlendSrcAlphaDstAlpha {
		blend = BlendSourceOver
	}
	return r.worldPipelines[worldPipelineKey{blend, options.DepthTest, options.DepthWrite}]
}

func (r *gpuRenderer) release() {
	for key, batch := range r.worldMeshBatchCache {
		batch.release()
		delete(r.worldMeshBatchCache, key)
	}
	r.pruneWorldMeshBufferPages()
	for _, bg := range r.bindGroups {
		if bg != nil {
			bg.Release()
		}
	}
	for _, tex := range r.textures {
		if tex != nil {
			tex.release()
		}
	}
	for _, mesh := range r.worldMeshes {
		if mesh != nil {
			if mesh.vertexBuf != nil {
				mesh.vertexBuf.Release()
			}
			if mesh.indexBuf != nil {
				mesh.indexBuf.Release()
			}
		}
	}
	for _, sampler := range r.samplers {
		if sampler != nil {
			sampler.Release()
		}
	}
	for _, buf := range []*wgpu.Buffer{
		r.worldVertexBuf.buf,
		r.worldIndexBuf.buf,
		r.screenVertexBuf.buf,
		r.screenIndexBuf.buf,
		r.billboardQuadBuf,
	} {
		if buf != nil {
			buf.Release()
		}
	}
	if r.pipelineAlpha != nil {
		r.pipelineAlpha.Release()
	}
	if r.pipelineAdd != nil {
		r.pipelineAdd.Release()
	}
	if r.pipelineSrcDst != nil {
		r.pipelineSrcDst.Release()
	}
	for _, pipeline := range r.worldPipelines {
		pipeline.Release()
	}
	for _, pipeline := range []*wgpu.RenderPipeline{
		r.billboardAlphaRead,
		r.billboardAddRead,
		r.billboardSrcDstRead,
		r.billboardAlphaNoDepth,
		r.billboardAddNoDepth,
		r.billboardSrcDstNoDepth,
	} {
		if pipeline != nil {
			pipeline.Release()
		}
	}
	if r.layout != nil {
		r.layout.Release()
	}
	if r.worldLayout != nil {
		r.worldLayout.Release()
	}
	if r.bgl != nil {
		r.bgl.Release()
	}
	if r.worldBGL != nil {
		r.worldBGL.Release()
	}
	if r.uniform != nil {
		r.uniform.Release()
	}
	if r.worldUniform != nil {
		r.worldUniform.Release()
	}
	if r.bloom != nil {
		r.bloom.release()
	}
	if r.ssao != nil {
		r.ssao.release()
	}
	r.depthTarget.release()
	r.msaaTarget.release()
}

func gpuFilter(filter Filter) gputypes.FilterMode {
	if filter == FilterNearest {
		return gputypes.FilterModeNearest
	}
	return gputypes.FilterModeLinear
}

func gpuAddress(address Address) gputypes.AddressMode {
	if address == AddressRepeat {
		return gputypes.AddressModeRepeat
	}
	return gputypes.AddressModeClampToEdge
}

func clearValue(c color.RGBA) gputypes.Color {
	return gputypes.Color{
		R: float64(c.R) / 255,
		G: float64(c.G) / 255,
		B: float64(c.B) / 255,
		A: float64(c.A) / 255,
	}
}

func saneColor(v float32) float32 {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		return 1
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func saneDepthBias(v float32) float32 {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || v < 0 {
		return 0
	}
	if v > 0.02 {
		return 0.02
	}
	return v
}

func uniformBytes(width, height float32) []byte {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:4], math.Float32bits(width))
	binary.LittleEndian.PutUint32(data[4:8], math.Float32bits(height))
	return data
}

func worldUniformBytes(camera Camera3D) []byte {
	data := make([]byte, 96)
	for i, v := range camera.ViewProjection {
		binary.LittleEndian.PutUint32(data[i*4:i*4+4], math.Float32bits(v))
	}
	fogEnabled := float32(0)
	if camera.Enabled && camera.Fog.Enabled && camera.Fog.Far > camera.Fog.Near {
		fogEnabled = 1
	}
	fogStrength := camera.Fog.Strength
	if fogStrength <= 0 || math.IsNaN(float64(fogStrength)) || math.IsInf(float64(fogStrength), 0) {
		fogStrength = 1
	}
	if fogStrength > 1 {
		fogStrength = 1
	}
	fogValues := [8]float32{
		camera.Fog.Near,
		camera.Fog.Far,
		fogEnabled,
		fogStrength,
		camera.Fog.ColorR,
		camera.Fog.ColorG,
		camera.Fog.ColorB,
		1,
	}
	for i, v := range fogValues {
		offset := 64 + i*4
		binary.LittleEndian.PutUint32(data[offset:offset+4], math.Float32bits(v))
	}
	return data
}

func floatBytes(values []float32) []byte {
	data := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(data[i*4:i*4+4], math.Float32bits(v))
	}
	return data
}

func u32Bytes(values []uint32) []byte {
	data := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(data[i*4:i*4+4], v)
	}
	return data
}
