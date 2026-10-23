package render

import (
	_ "embed"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

//go:embed hallucination.wgsl
var hallucinationShaderWGSL string

type gpuHallucination struct {
	bindings *wgpu.BindGroupLayout
	layout   *wgpu.PipelineLayout
	uniform  *wgpu.Buffer
	pipeline *wgpu.RenderPipeline
	target   gpuRenderTarget
	group    *wgpu.BindGroup
}

func newGPUHallucination(dev *wgpu.Device, format gputypes.TextureFormat) (_ *gpuHallucination, err error) {
	h := &gpuHallucination{}
	defer func() {
		if err != nil {
			h.release()
		}
	}()
	h.bindings, err = dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "goro-hallucination-bindings",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, MinBindingSize: 16}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
		},
	})
	if err != nil {
		return nil, err
	}
	h.layout, err = dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label: "goro-hallucination-layout", BindGroupLayouts: []*wgpu.BindGroupLayout{h.bindings},
	})
	if err != nil {
		return nil, err
	}
	h.uniform, err = dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-hallucination-phase", Size: 16, Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, err
	}
	shader, err := dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "goro-hallucination", WGSL: hallucinationShaderWGSL})
	if err != nil {
		return nil, err
	}
	defer shader.Release()
	h.pipeline, err = dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "goro-hallucination", Layout: h.layout,
		Vertex:    wgpu.VertexState{Module: shader, EntryPoint: "fullscreen"},
		Primitive: gputypes.PrimitiveState{Topology: gputypes.PrimitiveTopologyTriangleList},
		Fragment: &wgpu.FragmentState{
			Module: shader, EntryPoint: "distort",
			Targets: []gputypes.ColorTargetState{{Format: format, WriteMask: gputypes.ColorWriteMaskAll}},
		},
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *gpuHallucination) ensure(dev *wgpu.Device, format gputypes.TextureFormat, width, height int) error {
	if h.group != nil && h.target.width == width && h.target.height == height {
		return nil
	}
	h.releaseTarget()
	if err := h.target.ensure(dev, width, height, format, 1, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageTextureBinding); err != nil {
		return err
	}
	var err error
	h.group, err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label: "goro-hallucination-input", Layout: h.bindings,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: h.uniform, Size: 16},
			{Binding: 1, TextureView: h.target.view},
		},
	})
	return err
}

func (h *gpuHallucination) draw(enc *wgpu.CommandEncoder, surface *wgpu.TextureView) error {
	enc.TransitionTextures([]wgpu.TextureBarrier{{
		Texture: h.target.texture,
		Usage: wgpu.TextureUsageTransition{
			OldUsage: gputypes.TextureUsageRenderAttachment,
			NewUsage: gputypes.TextureUsageTextureBinding,
		},
	}})
	pass, err := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View: surface, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore,
		}},
	})
	if err != nil {
		return err
	}
	pass.SetPipeline(h.pipeline)
	pass.SetBindGroup(0, h.group, nil)
	pass.Draw(gputypes.DrawArgs{VertexCount: 3, InstanceCount: 1})
	return pass.End()
}

func (h *gpuHallucination) releaseTarget() {
	if h.group != nil {
		h.group.Release()
		h.group = nil
	}
	h.target.release()
}

func (h *gpuHallucination) release() {
	h.releaseTarget()
	if h.pipeline != nil {
		h.pipeline.Release()
	}
	if h.uniform != nil {
		h.uniform.Release()
	}
	if h.layout != nil {
		h.layout.Release()
	}
	if h.bindings != nil {
		h.bindings.Release()
	}
}
