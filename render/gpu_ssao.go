package render

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

//go:embed ssao.wgsl
var ssaoShaderWGSL string

//go:embed ssao_depth.wgsl
var ssaoDepthShaderWGSL string

// gpuSSAO keeps its geometry depth separate from the world's MSAA attachment.
// Only opaque geometry participates; composition precedes water and sprites.
type gpuSSAO struct {
	bindings  *wgpu.BindGroupLayout
	layout    *wgpu.PipelineLayout
	uniform   *wgpu.Buffer
	geometry  *wgpu.RenderPipeline
	estimate  *wgpu.RenderPipeline
	composite *wgpu.RenderPipeline
	depth     gpuRenderTarget
	zbuffer   gpuRenderTarget
	occlusion gpuRenderTarget
	groups    [2]*wgpu.BindGroup
}

func (r *gpuRenderer) newSSAO() (_ *gpuSSAO, err error) {
	a := &gpuSSAO{}
	defer func() {
		if err != nil {
			a.release()
		}
	}()
	shader, err := r.dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "goro-ssao-depth", WGSL: worldShaderWGSL + ssaoDepthShaderWGSL})
	if err != nil {
		return nil, err
	}
	defer shader.Release()
	geometry := r.worldPipelineDescriptor(shader, gputypes.BlendState{}, true, true, "goro-ssao-depth")
	geometry.Multisample.Count = 1
	geometry.Fragment.EntryPoint = "ssao_depth"
	geometry.Fragment.Targets[0].Format = gputypes.TextureFormatRGBA8Unorm
	geometry.Fragment.Targets[0].Blend = nil
	a.geometry, err = r.dev.CreateRenderPipeline(geometry)
	if err != nil {
		return nil, err
	}
	a.bindings, err = r.dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "goro-ssao-bindings",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: 80}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
		},
	})
	if err != nil {
		return nil, err
	}
	a.layout, err = r.dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{Label: "goro-ssao-layout", BindGroupLayouts: []*wgpu.BindGroupLayout{a.bindings}})
	if err != nil {
		return nil, err
	}
	a.uniform, err = r.dev.CreateBuffer(&wgpu.BufferDescriptor{Label: "goro-ssao-uniform", Size: 80, Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst})
	if err != nil {
		return nil, err
	}
	post, err := r.dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "goro-ssao", WGSL: ssaoShaderWGSL})
	if err != nil {
		return nil, err
	}
	defer post.Release()
	desc := &wgpu.RenderPipelineDescriptor{
		Label: "goro-ssao-estimate", Layout: a.layout,
		Vertex:    wgpu.VertexState{Module: post, EntryPoint: "fullscreen"},
		Primitive: gputypes.PrimitiveState{Topology: gputypes.PrimitiveTopologyTriangleList},
		Fragment: &wgpu.FragmentState{Module: post, EntryPoint: "estimate", Targets: []gputypes.ColorTargetState{{
			Format: gputypes.TextureFormatRGBA8Unorm, WriteMask: gputypes.ColorWriteMaskAll,
		}}},
	}
	a.estimate, err = r.dev.CreateRenderPipeline(desc)
	if err != nil {
		return nil, err
	}
	desc.Label = "goro-ssao-composite"
	desc.Fragment.EntryPoint = "composite"
	desc.Fragment.Targets[0].Format = r.format
	desc.Fragment.Targets[0].Blend = &gputypes.BlendState{
		Color: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorZero, DstFactor: gputypes.BlendFactorSrc, Operation: gputypes.BlendOperationAdd},
		Alpha: gputypes.BlendComponent{SrcFactor: gputypes.BlendFactorZero, DstFactor: gputypes.BlendFactorOne, Operation: gputypes.BlendOperationAdd},
	}
	desc.Multisample = gputypes.MultisampleState{Count: r.sampleCount, Mask: 0xFFFFFFFF}
	desc.DepthStencil = &wgpu.DepthStencilState{Format: depthFormat, DepthCompare: gputypes.CompareFunctionAlways}
	a.composite, err = r.dev.CreateRenderPipeline(desc)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *gpuSSAO) ensure(dev *wgpu.Device, width, height int) error {
	if a.groups[1] != nil && a.depth.width == width && a.depth.height == height {
		return nil
	}
	a.releaseTargets()
	// Full-resolution geometry preserves cutouts and supplies depth for bilateral
	// upsampling. The expensive occlusion calculation runs at half resolution.
	for _, target := range []struct {
		target *gpuRenderTarget
		w, h   int
		format gputypes.TextureFormat
		usage  gputypes.TextureUsage
	}{
		{&a.depth, width, height, gputypes.TextureFormatRGBA8Unorm, wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding},
		{&a.zbuffer, width, height, depthFormat, wgpu.TextureUsageRenderAttachment},
		{&a.occlusion, (width + 1) / 2, (height + 1) / 2, gputypes.TextureFormatRGBA8Unorm, wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding},
	} {
		if err := target.target.ensure(dev, target.w, target.h, target.format, 1, target.usage); err != nil {
			return err
		}
	}
	for i, source := range []*wgpu.TextureView{a.depth.view, a.occlusion.view} {
		var err error
		a.groups[i], err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label: "goro-ssao-input", Layout: a.bindings,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Buffer: a.uniform, Size: 80},
				{Binding: 1, TextureView: a.depth.view},
				{Binding: 2, TextureView: source},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *gpuSSAO) uploadCamera(queue *wgpu.Queue, camera Camera3D) error {
	inverse, ok := inverseViewProjection(camera.ViewProjection)
	if !ok {
		return fmt.Errorf("SSAO camera matrix is not invertible")
	}
	values := [20]float32{}
	copy(values[:16], inverse[:])
	values[16], values[17] = 2, 0.55
	return queue.WriteBuffer(a.uniform, 0, floatBytes(values[:]))
}

func (r *gpuRenderer) drawSSAO(enc *wgpu.CommandEncoder, meshes []worldMeshBatch, world worldFrame, vertices, indices *wgpu.Buffer) error {
	a := r.ssao
	pass, err := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{View: a.depth.view, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore}},
		DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
			View: a.zbuffer.view, DepthLoadOp: gputypes.LoadOpClear, DepthStoreOp: gputypes.StoreOpDiscard, DepthClearValue: 1,
		},
	})
	if err != nil {
		return err
	}
	if err := r.drawWorldOpaque(pass, meshes, world, vertices, indices, a.geometry); err != nil {
		_ = pass.End()
		return err
	}
	if err := pass.End(); err != nil {
		return err
	}
	enc.TransitionTextures([]wgpu.TextureBarrier{{Texture: a.depth.texture, Usage: wgpu.TextureUsageTransition{OldUsage: wgpu.TextureUsageRenderAttachment, NewUsage: wgpu.TextureUsageTextureBinding}}})
	pass, err = enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{View: a.occlusion.view, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore}},
	})
	if err != nil {
		return err
	}
	pass.SetPipeline(a.estimate)
	pass.SetBindGroup(0, a.groups[0], nil)
	pass.Draw(gputypes.DrawArgs{VertexCount: 3, InstanceCount: 1})
	if err := pass.End(); err != nil {
		return err
	}
	enc.TransitionTextures([]wgpu.TextureBarrier{{Texture: a.occlusion.texture, Usage: wgpu.TextureUsageTransition{OldUsage: wgpu.TextureUsageRenderAttachment, NewUsage: wgpu.TextureUsageTextureBinding}}})
	return nil
}

func (a *gpuSSAO) releaseTargets() {
	for i, group := range a.groups {
		if group != nil {
			group.Release()
			a.groups[i] = nil
		}
	}
	a.depth.release()
	a.zbuffer.release()
	a.occlusion.release()
}

func (a *gpuSSAO) release() {
	a.releaseTargets()
	for _, pipeline := range []*wgpu.RenderPipeline{a.geometry, a.estimate, a.composite} {
		if pipeline != nil {
			pipeline.Release()
		}
	}
	if a.uniform != nil {
		a.uniform.Release()
	}
	if a.layout != nil {
		a.layout.Release()
	}
	if a.bindings != nil {
		a.bindings.Release()
	}
}

// inverseViewProjection inverts column-major camera matrices with pivoting.
func inverseViewProjection(matrix [16]float32) ([16]float32, bool) {
	var rows [4][8]float64
	for row := range 4 {
		for col := range 4 {
			rows[row][col] = float64(matrix[col*4+row])
		}
		rows[row][row+4] = 1
	}
	for col := range 4 {
		pivot := col
		for row := col + 1; row < 4; row++ {
			if math.Abs(rows[row][col]) > math.Abs(rows[pivot][col]) {
				pivot = row
			}
		}
		divisor := rows[pivot][col]
		if divisor == 0 || math.IsNaN(divisor) || math.IsInf(divisor, 0) {
			return [16]float32{}, false
		}
		rows[col], rows[pivot] = rows[pivot], rows[col]
		for i := range 8 {
			rows[col][i] /= divisor
		}
		for row := range 4 {
			if row == col {
				continue
			}
			scale := rows[row][col]
			for i := range 8 {
				rows[row][i] -= rows[col][i] * scale
			}
		}
	}
	var inverse [16]float32
	for row := range 4 {
		for col := range 4 {
			value := rows[row][col+4]
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > math.MaxFloat32 {
				return [16]float32{}, false
			}
			inverse[col*4+row] = float32(value)
		}
	}
	return inverse, true
}
