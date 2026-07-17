package render

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/gogpu/gogpu"
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

type gpuRenderer struct {
	dev              *wgpu.Device
	queue            *wgpu.Queue
	format           gputypes.TextureFormat
	bgl              *wgpu.BindGroupLayout
	worldBGL         *wgpu.BindGroupLayout
	layout           *wgpu.PipelineLayout
	worldLayout      *wgpu.PipelineLayout
	pipelineAlpha    *wgpu.RenderPipeline
	pipelineAdd      *wgpu.RenderPipeline
	worldAlphaWrite  *wgpu.RenderPipeline
	worldAddWrite    *wgpu.RenderPipeline
	worldAlphaRead   *wgpu.RenderPipeline
	worldAddRead     *wgpu.RenderPipeline
	billboardAlpha   *wgpu.RenderPipeline
	billboardAdd     *wgpu.RenderPipeline
	uniform          *wgpu.Buffer
	worldUniform     *wgpu.Buffer
	samplers         map[samplerKey]*wgpu.Sampler
	textures         map[*Image]*gpuImageTexture
	bindGroups       map[bindGroupKey]*wgpu.BindGroup
	worldMeshes      map[*WorldMesh]*gpuWorldMesh
	depthTex         *wgpu.Texture
	depthView        *wgpu.TextureView
	depthWidth       int
	depthHeight      int
	worldVertexBuf   dynamicGPUBuffer
	worldIndexBuf    dynamicGPUBuffer
	screenVertexBuf  dynamicGPUBuffer
	screenIndexBuf   dynamicGPUBuffer
	billboardQuadBuf *wgpu.Buffer
	frameBuffers     []*wgpu.Buffer
	frameBindGroups  []*wgpu.BindGroup
	neutralLightmap  *Image
	statsEnabled     bool
	statsLast        time.Time
	worldDebug       bool
	worldDebugLast   time.Time

	worldFrameScratch worldFrameScratch
}

type gpuImageTexture struct {
	tex     *gogpu.Texture
	version uint64
	width   int
	height  int
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
	filter  Filter
	address Address
}

type bindGroupKey struct {
	texture      *gogpu.Texture
	lightTexture *gogpu.Texture
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
	}
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
		Entries: []wgpu.BindGroupLayoutEntry{
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
		Entries: []wgpu.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageVertex | wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: 96}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Sampler: &gputypes.SamplerBindingLayout{Type: gputypes.SamplerBindingTypeFiltering}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
			{Binding: 3, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
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
	r.worldAlphaWrite, err = r.createWorldPipeline(worldShader, gputypes.BlendStateAlpha(), true, "goro-world-pipeline-alpha-write")
	if err != nil {
		return err
	}
	r.worldAddWrite, err = r.createWorldPipeline(worldShader, add, true, "goro-world-pipeline-add-write")
	if err != nil {
		return err
	}
	r.worldAlphaRead, err = r.createWorldPipeline(worldShader, gputypes.BlendStateAlpha(), false, "goro-world-pipeline-alpha-read")
	if err != nil {
		return err
	}
	r.worldAddRead, err = r.createWorldPipeline(worldShader, add, false, "goro-world-pipeline-add-read")
	if err != nil {
		return err
	}
	r.billboardAlpha, err = r.createWorldBillboardPipeline(billboardShader, gputypes.BlendStateAlpha(), "goro-world-billboard-alpha")
	if err != nil {
		return err
	}
	r.billboardAdd, err = r.createWorldBillboardPipeline(billboardShader, add, "goro-world-billboard-add")
	if err != nil {
		return err
	}
	r.billboardQuadBuf, err = r.createBillboardQuadBuffer()
	return err
}

func (r *gpuRenderer) createPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.layout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{{
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
	})
}

func (r *gpuRenderer) createWorldPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, depthWrite bool, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.worldLayout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{{
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
		DepthStencil: &wgpu.DepthStencilState{
			Format:            gputypes.TextureFormatDepth24Plus,
			DepthWriteEnabled: depthWrite,
			DepthCompare:      gputypes.CompareFunctionLessEqual,
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

func (r *gpuRenderer) createWorldBillboardPipeline(shader *wgpu.ShaderModule, blend gputypes.BlendState, label string) (*wgpu.RenderPipeline, error) {
	return r.dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  label,
		Layout: r.worldLayout,
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{
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
					},
				},
			},
		},
		Primitive: gputypes.PrimitiveState{
			Topology:  gputypes.PrimitiveTopologyTriangleList,
			FrontFace: gputypes.FrontFaceCCW,
			CullMode:  gputypes.CullModeNone,
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            gputypes.TextureFormatDepth24Plus,
			DepthWriteEnabled: false,
			DepthCompare:      gputypes.CompareFunctionLessEqual,
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

func (r *gpuRenderer) Draw(ctx *gogpu.Context, screen *Frame) error {
	if screen == nil {
		return nil
	}
	surface := ctx.SurfaceView()
	if surface == nil {
		return nil
	}
	width, height := ctx.FramebufferSize()
	if width <= 0 || height <= 0 {
		b := screen.Bounds()
		width, height = b.Dx(), b.Dy()
	}
	if width <= 0 || height <= 0 {
		return nil
	}
	if err := r.queue.WriteBuffer(r.uniform, 0, uniformBytes(float32(width), float32(height))); err != nil {
		return fmt.Errorf("upload screen uniform: %w", err)
	}
	if screen.camera.Enabled {
		if err := r.queue.WriteBuffer(r.worldUniform, 0, worldUniformBytes(screen.camera)); err != nil {
			return fmt.Errorf("upload world uniform: %w", err)
		}
	}
	if err := r.ensureDepth(width, height); err != nil {
		return err
	}
	r.releaseFrameResources()
	world := r.buildWorldFrame(screen)
	frame := r.buildFrame(screen)
	if r.statsEnabled && time.Since(r.statsLast) >= time.Second {
		glog.Debugf("render stats world_commands=%d world_mesh_commands=%d world_billboards=%d retained_world_meshes=%d world_batches=%d world_vertices=%d world_indices=%d commands=%d batches=%d vertices=%d indices=%d textures=%d bindgroups=%d", len(screen.worldCommands), len(screen.worldMeshes), len(screen.worldBillboards), len(r.worldMeshes), len(world.batches), len(world.floats)/worldVertexFloatCount, len(world.indices), len(screen.commands), len(frame.batches), len(frame.floats)/screenVertexFloatCount, len(frame.indices), len(r.textures), len(r.bindGroups))
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
			return err
		}
		worldIndexBuf, err = r.dynamicBuffer(&r.worldIndexBuf, "goro-world-indices", len(world.indices)*4, wgpu.BufferUsageIndex|wgpu.BufferUsageCopyDst, u32Bytes(world.indices))
		if err != nil {
			return err
		}
	}
	if len(frame.floats) > 0 && len(frame.indices) > 0 {
		vertexBuf, err = r.dynamicBuffer(&r.screenVertexBuf, "goro-screen-vertices", len(frame.floats)*4, wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst, floatBytes(frame.floats))
		if err != nil {
			return err
		}
		indexBuf, err = r.dynamicBuffer(&r.screenIndexBuf, "goro-screen-indices", len(frame.indices)*4, wgpu.BufferUsageIndex|wgpu.BufferUsageCopyDst, u32Bytes(frame.indices))
		if err != nil {
			return err
		}
	}

	enc, err := r.dev.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "goro-screen-encoder"})
	if err != nil {
		return err
	}
	clear := clearValue(screen.clear)
	pass, err := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View:       surface,
			LoadOp:     gputypes.LoadOpClear,
			StoreOp:    gputypes.StoreOpStore,
			ClearValue: clear,
		}},
		DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
			View:            r.depthView,
			DepthLoadOp:     gputypes.LoadOpClear,
			DepthStoreOp:    gputypes.StoreOpStore,
			DepthClearValue: 1,
		},
	})
	if err != nil {
		return err
	}
	worldState := renderPassState{}
	if screen.camera.Enabled {
		for _, meshCommand := range screen.worldMeshes {
			mesh := meshCommand.Mesh
			if mesh == nil || !mesh.options.DepthWrite {
				continue
			}
			if err := r.drawWorldMesh(ctx, pass, mesh, &worldState); err != nil {
				_ = pass.End()
				return err
			}
		}
	}
	if worldVertexBuf != nil && worldIndexBuf != nil && screen.camera.Enabled {
		worldState.setVertexBuffer(pass, worldVertexBuf)
		worldState.setIndexBuffer(pass, worldIndexBuf)
		for _, batch := range world.batches {
			if batch.indexCount == 0 {
				continue
			}
			tex, err := r.ensureTexture(ctx, batch.key.texture, batch.key.options)
			if err != nil {
				_ = pass.End()
				return err
			}
			sampler, err := r.sampler(batch.key.options)
			if err != nil {
				_ = pass.End()
				return err
			}
			lightTex, err := r.ensureBatchLightTexture(ctx, batch.key)
			if err != nil {
				_ = pass.End()
				return err
			}
			bg, err := r.bindWorldGroup(r.worldUniform, 96, tex.tex, lightTex.tex, sampler)
			if err != nil {
				_ = pass.End()
				return err
			}
			worldState.setPipeline(pass, r.worldPipelineFor(batch.key.options.Blend, batch.key.options.DepthWrite))
			worldState.setBindGroup(pass, bg)
			pass.DrawIndexed(batch.indexCount, 1, batch.firstIndex, 0, 0)
		}
	}
	if screen.camera.Enabled {
		for _, meshCommand := range screen.worldMeshes {
			mesh := meshCommand.Mesh
			if mesh == nil || mesh.options.DepthWrite {
				continue
			}
			if err := r.drawWorldMesh(ctx, pass, mesh, &worldState); err != nil {
				_ = pass.End()
				return err
			}
		}
		for _, billboard := range screen.worldBillboards {
			if err := r.drawWorldBillboard(ctx, pass, billboard); err != nil {
				_ = pass.End()
				return err
			}
		}
	}
	if vertexBuf != nil && indexBuf != nil {
		pass.SetVertexBuffer(0, vertexBuf, 0)
		pass.SetIndexBuffer(indexBuf, gputypes.IndexFormatUint32, 0)
	}
	for _, batch := range frame.batches {
		if batch.indexCount == 0 {
			continue
		}
		tex, err := r.ensureTexture(ctx, batch.key.texture, batch.key.options)
		if err != nil {
			_ = pass.End()
			return err
		}
		sampler, err := r.sampler(batch.key.options)
		if err != nil {
			_ = pass.End()
			return err
		}
		bg, err := r.bindGroup(r.bgl, r.uniform, 16, tex.tex, sampler)
		if err != nil {
			_ = pass.End()
			return err
		}
		pass.SetPipeline(r.pipeline(batch.key.options.Blend))
		pass.SetBindGroup(0, bg, nil)
		pass.DrawIndexed(batch.indexCount, 1, batch.firstIndex, 0, 0)
	}
	if err := pass.End(); err != nil {
		return err
	}
	cmd, err := enc.Finish()
	if err != nil {
		return err
	}
	_, err = r.queue.Submit(cmd)
	return err
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

func (r *gpuRenderer) ensureTexture(ctx *gogpu.Context, img *Image, opts DrawTrianglesOptions) (*gpuImageTexture, error) {
	if img == nil || img.pix == nil {
		return nil, fmt.Errorf("nil render texture")
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	existing := r.textures[img]
	if existing != nil && existing.version == img.version && existing.width == w && existing.height == h {
		return existing, nil
	}
	if existing != nil {
		if existing.width == w && existing.height == h {
			if err := existing.tex.UpdateData(img.RGBA().Pix); err != nil {
				return nil, fmt.Errorf("update render texture: %w", err)
			}
			existing.version = img.version
			return existing, nil
		}
		r.releaseTexture(existing.tex)
		delete(r.textures, img)
	}
	tex, err := ctx.Renderer().NewTextureFromRGBAWithOptions(w, h, img.RGBA().Pix, gogpu.TextureOptions{
		Label:        "goro-image-texture",
		MagFilter:    gpuFilter(opts.Filter),
		MinFilter:    gpuFilter(opts.Filter),
		AddressModeU: gpuAddress(opts.Address),
		AddressModeV: gpuAddress(opts.Address),
	})
	if err != nil {
		return nil, fmt.Errorf("create render texture: %w", err)
	}
	out := &gpuImageTexture{tex: tex, version: img.version, width: w, height: h}
	r.textures[img] = out
	return out, nil
}

func (r *gpuRenderer) ensureBatchLightTexture(ctx *gogpu.Context, key drawBatchKey) (*gpuImageTexture, error) {
	if key.lightTexture != nil && key.lightTexture.pix != nil {
		return r.ensureTexture(ctx, key.lightTexture, DrawTrianglesOptions{Filter: FilterLinear, Address: AddressClampToZero})
	}
	return r.ensureTexture(ctx, r.neutralLightmapImage(), DrawTrianglesOptions{Filter: FilterLinear, Address: AddressClampToZero})
}

func (r *gpuRenderer) ensureMeshLightTexture(ctx *gogpu.Context, mesh *WorldMesh) (*gpuImageTexture, error) {
	if mesh != nil && mesh.lightTexture != nil && mesh.lightTexture.pix != nil {
		return r.ensureTexture(ctx, mesh.lightTexture, DrawTrianglesOptions{Filter: FilterLinear, Address: AddressClampToZero})
	}
	if mesh == nil {
		return nil, fmt.Errorf("nil world mesh")
	}
	return r.ensureTexture(ctx, r.neutralLightmapImage(), DrawTrianglesOptions{Filter: FilterLinear, Address: AddressClampToZero})
}

func (r *gpuRenderer) neutralLightmapImage() *Image {
	if r.neutralLightmap == nil {
		r.neutralLightmap = NewImage(1, 1)
		r.neutralLightmap.Fill(color.RGBA{A: 255})
	}
	return r.neutralLightmap
}

func (r *gpuRenderer) releaseTexture(tex *gogpu.Texture) {
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
	tex.Destroy()
}

func (r *gpuRenderer) sampler(opts DrawTrianglesOptions) (*wgpu.Sampler, error) {
	key := samplerKey{filter: opts.Filter, address: opts.Address}
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
		MipmapFilter: gputypes.FilterModeNearest,
		LodMaxClamp:  32,
	})
	if err != nil {
		return nil, err
	}
	r.samplers[key] = sampler
	return sampler, nil
}

func (r *gpuRenderer) bindGroup(layout *wgpu.BindGroupLayout, uniform *wgpu.Buffer, uniformSize uint64, tex *gogpu.Texture, sampler *wgpu.Sampler) (*wgpu.BindGroup, error) {
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
			{Binding: 2, TextureView: tex.View()},
		},
	})
	if err != nil {
		return nil, err
	}
	r.bindGroups[key] = bg
	return bg, nil
}

func (r *gpuRenderer) bindWorldGroup(uniform *wgpu.Buffer, uniformSize uint64, tex, lightTex *gogpu.Texture, sampler *wgpu.Sampler) (*wgpu.BindGroup, error) {
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
			{Binding: 2, TextureView: tex.View()},
			{Binding: 3, TextureView: lightTex.View()},
		},
	})
	if err != nil {
		return nil, err
	}
	r.bindGroups[key] = bg
	return bg, nil
}

func (r *gpuRenderer) drawWorldMesh(ctx *gogpu.Context, pass *wgpu.RenderPassEncoder, mesh *WorldMesh, state *renderPassState) error {
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
	tex, err := r.ensureTexture(ctx, mesh.texture, mesh.options)
	if err != nil {
		return err
	}
	lightTex, err := r.ensureMeshLightTexture(ctx, mesh)
	if err != nil {
		return err
	}
	sampler, err := r.sampler(mesh.options)
	if err != nil {
		return err
	}
	bg, err := r.bindWorldGroup(r.worldUniform, 96, tex.tex, lightTex.tex, sampler)
	if err != nil {
		return err
	}
	state.setPipeline(pass, r.worldPipelineFor(mesh.options.Blend, mesh.options.DepthWrite))
	state.setBindGroup(pass, bg)
	state.setVertexBuffer(pass, gpuMesh.vertexBuf)
	state.setIndexBuffer(pass, gpuMesh.indexBuf)
	pass.DrawIndexed(gpuMesh.indexCount, 1, 0, 0, 0)
	return nil
}

func (r *gpuRenderer) drawWorldBillboard(ctx *gogpu.Context, pass *wgpu.RenderPassEncoder, cmd WorldBillboardCommand) error {
	if cmd.Texture == nil || cmd.Texture.pix == nil || r.billboardQuadBuf == nil {
		return nil
	}
	instance := billboardInstanceData(cmd)
	instanceBuf, err := r.frameBuffer("goro-world-billboard-instances", len(instance)*4, wgpu.BufferUsageVertex|wgpu.BufferUsageCopyDst, floatBytes(instance))
	if err != nil {
		return err
	}
	tex, err := r.ensureTexture(ctx, cmd.Texture, cmd.Options)
	if err != nil {
		return err
	}
	sampler, err := r.sampler(cmd.Options)
	if err != nil {
		return err
	}
	bg, err := r.bindWorldGroup(r.worldUniform, 96, tex.tex, tex.tex, sampler)
	if err != nil {
		return err
	}
	if cmd.Options.Blend == BlendLighter {
		pass.SetPipeline(r.billboardAdd)
	} else {
		pass.SetPipeline(r.billboardAlpha)
	}
	pass.SetBindGroup(0, bg, nil)
	pass.SetVertexBuffer(0, r.billboardQuadBuf, 0)
	pass.SetVertexBuffer(1, instanceBuf, 0)
	pass.Draw(6, 1, 0, 0)
	return nil
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
	return []float32{
		cmd.Center[0], cmd.Center[1], cmd.Center[2],
		cmd.RightAxis[0], cmd.RightAxis[1], cmd.RightAxis[2],
		cmd.UpAxis[0], cmd.UpAxis[1], cmd.UpAxis[2],
		cmd.DepthUpAxis[0], cmd.DepthUpAxis[1], cmd.DepthUpAxis[2],
		cmd.Width, cmd.Height, cmd.AnchorX, cmd.AnchorY,
		saneColor(cmd.ColorR), saneColor(cmd.ColorG), saneColor(cmd.ColorB), saneColor(cmd.ColorA),
		saneDepthBias(cmd.DepthBias), 0, 0, 0,
	}
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

func (r *gpuRenderer) dynamicBuffer(slot *dynamicGPUBuffer, label string, size int, usage wgpu.BufferUsage, data []byte) (*wgpu.Buffer, error) {
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

func (r *gpuRenderer) frameBuffer(label string, size int, usage wgpu.BufferUsage, data []byte) (*wgpu.Buffer, error) {
	if size <= 0 {
		size = 4
	}
	buf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: label,
		Size:  uint64(size),
		Usage: usage,
	})
	if err != nil {
		return nil, err
	}
	r.frameBuffers = append(r.frameBuffers, buf)
	if len(data) > 0 {
		if err := r.queue.WriteBuffer(buf, 0, data); err != nil {
			return nil, err
		}
	}
	return buf, nil
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
	return r.pipelineAlpha
}

func (r *gpuRenderer) worldPipelineFor(blend Blend, depthWrite bool) *wgpu.RenderPipeline {
	if blend == BlendLighter {
		if depthWrite {
			return r.worldAddWrite
		}
		return r.worldAddRead
	}
	if depthWrite {
		return r.worldAlphaWrite
	}
	return r.worldAlphaRead
}

func (r *gpuRenderer) ensureDepth(width, height int) error {
	if r.depthView != nil && r.depthWidth == width && r.depthHeight == height {
		return nil
	}
	if r.depthView != nil {
		r.depthView.Release()
		r.depthView = nil
	}
	if r.depthTex != nil {
		r.depthTex.Release()
		r.depthTex = nil
	}
	tex, err := r.dev.CreateTexture(&wgpu.TextureDescriptor{
		Label: "goro-depth",
		Size: wgpu.Extent3D{
			Width:              uint32(width),
			Height:             uint32(height),
			DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     gputypes.TextureDimension2D,
		Format:        gputypes.TextureFormatDepth24Plus,
		Usage:         wgpu.TextureUsageRenderAttachment,
	})
	if err != nil {
		return err
	}
	view, err := r.dev.CreateTextureView(tex, nil)
	if err != nil {
		tex.Release()
		return err
	}
	r.depthTex = tex
	r.depthView = view
	r.depthWidth = width
	r.depthHeight = height
	return nil
}

func (r *gpuRenderer) releaseFrameResources() {
	for _, buf := range r.frameBuffers {
		if buf != nil {
			buf.Release()
		}
	}
	r.frameBuffers = r.frameBuffers[:0]
	for _, bg := range r.frameBindGroups {
		if bg != nil {
			bg.Release()
		}
	}
	r.frameBindGroups = r.frameBindGroups[:0]
}

func (r *gpuRenderer) release() {
	r.releaseFrameResources()
	for _, bg := range r.bindGroups {
		if bg != nil {
			bg.Release()
		}
	}
	for _, tex := range r.textures {
		if tex != nil && tex.tex != nil {
			tex.tex.Destroy()
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
	if r.worldAlphaWrite != nil {
		r.worldAlphaWrite.Release()
	}
	if r.worldAddWrite != nil {
		r.worldAddWrite.Release()
	}
	if r.worldAlphaRead != nil {
		r.worldAlphaRead.Release()
	}
	if r.worldAddRead != nil {
		r.worldAddRead.Release()
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
	if r.depthView != nil {
		r.depthView.Release()
	}
	if r.depthTex != nil {
		r.depthTex.Release()
	}
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
	fogValues := [8]float32{
		camera.Fog.Near,
		camera.Fog.Far,
		fogEnabled,
		0,
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
