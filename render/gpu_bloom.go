package render

import (
	_ "embed"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

//go:embed bloom.wgsl
var bloomShaderWGSL string

// gpuBloom owns the scene and three quarter-resolution images: bright pixels,
// horizontal blur, and vertical blur. The final pass composites onto the
// surface before the UI, so neither text nor windows contribute to the glow.
type gpuBloom struct {
	layout    *wgpu.PipelineLayout
	bindings  *wgpu.BindGroupLayout
	sampler   *wgpu.Sampler
	pipelines [4]*wgpu.RenderPipeline
	targets   [4]gpuRenderTarget
	groups    [4]*wgpu.BindGroup
}

func newGPUBloom(dev *wgpu.Device, format gputypes.TextureFormat) (_ *gpuBloom, err error) {
	b := &gpuBloom{}
	defer func() {
		if err != nil {
			b.release()
		}
	}()
	b.bindings, err = dev.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label: "goro-bloom-bindings",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: wgpu.ShaderStageFragment, Sampler: &gputypes.SamplerBindingLayout{Type: gputypes.SamplerBindingTypeFiltering}},
			{Binding: 1, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
			{Binding: 2, Visibility: wgpu.ShaderStageFragment, Texture: &gputypes.TextureBindingLayout{SampleType: gputypes.TextureSampleTypeFloat, ViewDimension: gputypes.TextureViewDimension2D}},
		},
	})
	if err != nil {
		return nil, err
	}
	b.layout, err = dev.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label: "goro-bloom-layout", BindGroupLayouts: []*wgpu.BindGroupLayout{b.bindings},
	})
	if err != nil {
		return nil, err
	}
	b.sampler, err = dev.CreateSampler(&wgpu.SamplerDescriptor{
		Label:        "goro-bloom-sampler",
		AddressModeU: gputypes.AddressModeClampToEdge,
		AddressModeV: gputypes.AddressModeClampToEdge,
		AddressModeW: gputypes.AddressModeClampToEdge,
		MinFilter:    gputypes.FilterModeLinear, MagFilter: gputypes.FilterModeLinear,
	})
	if err != nil {
		return nil, err
	}
	shader, err := dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{Label: "goro-bloom", WGSL: bloomShaderWGSL})
	if err != nil {
		return nil, err
	}
	defer shader.Release()
	for i, entry := range []string{"extract", "blur_horizontal", "blur_vertical", "composite"} {
		b.pipelines[i], err = dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
			Label: "goro-bloom-" + entry, Layout: b.layout,
			Vertex:    wgpu.VertexState{Module: shader, EntryPoint: "fullscreen"},
			Primitive: gputypes.PrimitiveState{Topology: gputypes.PrimitiveTopologyTriangleList},
			Fragment: &wgpu.FragmentState{
				Module: shader, EntryPoint: entry,
				Targets: []gputypes.ColorTargetState{{Format: format, WriteMask: gputypes.ColorWriteMaskAll}},
			},
		})
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *gpuBloom) ensure(dev *wgpu.Device, format gputypes.TextureFormat, width, height int) error {
	if b.groups[3] != nil && b.targets[0].width == width && b.targets[0].height == height {
		return nil
	}
	b.releaseTargets()
	for i := range b.targets {
		w, h := width, height
		if i > 0 {
			w, h = max(1, (width+3)/4), max(1, (height+3)/4)
		}
		if err := b.targets[i].ensure(dev, w, h, format, 1, wgpu.TextureUsageRenderAttachment|wgpu.TextureUsageTextureBinding); err != nil {
			return err
		}
	}
	for i := range b.groups {
		input, glow := b.targets[i].view, b.targets[i].view
		if i == 3 {
			input = b.targets[0].view
		}
		var err error
		b.groups[i], err = dev.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label: "goro-bloom-input", Layout: b.bindings,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Sampler: b.sampler},
				{Binding: 1, TextureView: input},
				{Binding: 2, TextureView: glow},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *gpuBloom) draw(enc *wgpu.CommandEncoder, surface *wgpu.TextureView) error {
	for i, pipeline := range b.pipelines {
		// Each input has just been written by the preceding render pass.
		enc.TransitionTextures([]wgpu.TextureBarrier{{
			Texture: b.targets[i].texture,
			Usage: wgpu.TextureUsageTransition{
				OldUsage: gputypes.TextureUsageRenderAttachment,
				NewUsage: gputypes.TextureUsageTextureBinding,
			},
		}})
		output := surface
		if i < 3 {
			output = b.targets[i+1].view
		}
		pass, err := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
			ColorAttachments: []wgpu.RenderPassColorAttachment{{
				View: output, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore,
			}},
		})
		if err != nil {
			return err
		}
		pass.SetPipeline(pipeline)
		pass.SetBindGroup(0, b.groups[i], nil)
		pass.Draw(gputypes.DrawArgs{VertexCount: 3, InstanceCount: 1})
		if err := pass.End(); err != nil {
			return err
		}
	}
	return nil
}

func (b *gpuBloom) releaseTargets() {
	for i, group := range b.groups {
		if group != nil {
			group.Release()
			b.groups[i] = nil
		}
	}
	for i := range b.targets {
		b.targets[i].release()
	}
}

func (b *gpuBloom) release() {
	b.releaseTargets()
	for _, pipeline := range b.pipelines {
		if pipeline != nil {
			pipeline.Release()
		}
	}
	if b.sampler != nil {
		b.sampler.Release()
	}
	if b.layout != nil {
		b.layout.Release()
	}
	if b.bindings != nil {
		b.bindings.Release()
	}
}
